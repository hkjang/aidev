import * as http from "node:http";
import * as https from "node:https";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { config, stringSetting } from "../settings";
import { writeAudit } from "./workspace";
import { writeAutonomyPreset } from "./defaults";

export type UsageWindow = "weekly" | "monthly";

/** Shape of vibe-coders `GET /me/report` (internal/proxy/me_actions.go). */
export interface UsageReport {
	user_id: string;
	window: UsageWindow;
	since: string;
	requests: number;
	tokens: number;
	cost_krw: number;
	errors: number;
	success_rate: number;
	prior_cost_krw: number;
	prior_requests: number;
	cost_delta_ratio: number;
	avg_latency_ms: number;
	cache_rate: number;
	risk_score: number;
	top_models: Array<{ model?: string; name?: string; cost_krw?: number; requests?: number; tokens?: number; [k: string]: unknown }>;
	top_task_types: unknown;
	potential_savings_krw: number;
	potential_savings_model: string;
	recommendation_count: number;
}

export interface UsageResult {
	ok: boolean;
	report?: UsageReport;
	status?: number;
	error?: string;
}

function proxyOrigin(): string {
	const base = stringSetting("vibeCodersBaseUrl", "http://localhost:8080/v1");
	try {
		const url = new URL(base);
		return url.origin;
	} catch {
		return "http://localhost:8080";
	}
}

/** GET /me/report on the vibe-coders proxy with the profile's proxy key as bearer token. */
export function fetchUsageReport(apiKey: string, window: UsageWindow, timeoutMs = 3000): Promise<UsageResult> {
	return new Promise((resolve) => {
		try {
			const url = new URL(`/me/report?window=${window}`, proxyOrigin());
			const client = url.protocol === "https:" ? https : http;
			const req = client.request(
				{ method: "GET", hostname: url.hostname, port: url.port, path: url.pathname + url.search, timeout: timeoutMs, headers: { Authorization: `Bearer ${apiKey}`, Accept: "application/json" } },
				(res) => {
					let body = "";
					res.setEncoding("utf8");
					res.on("data", (chunk: string) => (body += chunk));
					res.on("end", () => {
						const status = res.statusCode ?? 0;
						if (status < 200 || status >= 300) return resolve({ ok: false, status, error: body.slice(0, 200) });
						try {
							resolve({ ok: true, status, report: JSON.parse(body) as UsageReport });
						} catch (error) {
							resolve({ ok: false, status, error: `invalid JSON: ${error}` });
						}
					});
				},
			);
			req.on("error", (error: NodeJS.ErrnoException) => resolve({ ok: false, error: error.code || error.message }));
			req.on("timeout", () => {
				req.destroy();
				resolve({ ok: false, error: "timeout" });
			});
			req.end();
		} catch (error) {
			resolve({ ok: false, error: String(error) });
		}
	});
}

export const formatKrw = (value: number | undefined): string => `₩${Math.round(value || 0).toLocaleString("ko-KR")}`;
export const formatTokens = (value: number | undefined): string => {
	const n = value || 0;
	return n >= 1_000_000 ? `${(n / 1_000_000).toFixed(2)}M` : n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);
};
export const formatPercent = (ratio: number | undefined): string => `${((ratio || 0) * 100).toFixed(1)}%`;
export const modelName = (row: UsageReport["top_models"][number]): string => String(row.model || row.name || "(unknown)");

/** One-line summary for the status bar, e.g. `₩12,340 · 45.2k tok (주간)`. */
export function usageSummaryLine(report: UsageReport): string {
	return `${formatKrw(report.cost_krw)} · ${formatTokens(report.tokens)} tok (${report.window === "monthly" ? "월간" : "주간"})`;
}

export interface UsageCache {
	fetchedAt: number;
	result: UsageResult;
}

let cache: UsageCache | undefined;
const USAGE_TTL_MS = 5 * 60 * 1000;

/** Weekly report, cached for five minutes; null when the proxy route is off or offline mode is set. */
export async function cachedWeeklyUsage(host: CoreHost, force = false): Promise<UsageResult | null> {
	if ((config().get<string>("offlineMode") || "auto") === "offline") return null;
	const cp = await host.contextProxy.getInstance(host.context);
	const vals = cp.getValues();
	const expected = stringSetting("vibeCodersBaseUrl", "http://localhost:8080/v1").replace(/\/+$/g, "");
	if (vals.apiProvider !== "openai" || vals.openAiBaseUrl !== expected) return null;
	const apiKey = String(vals.openAiApiKey || "");
	if (!apiKey) return null;
	if (!force && cache && Date.now() - cache.fetchedAt < USAGE_TTL_MS) return cache.result;
	const result = await fetchUsageReport(apiKey, "weekly");
	cache = { fetchedAt: Date.now(), result };
	if (result.ok && result.report) await enforceBudget(host, result.report);
	return result;
}

/** Weekly budget guard: over `vibe-code.weeklyBudgetKrw` (>0) drops auto/yolo autonomy to assist once per day and notifies. */
export async function enforceBudget(host: CoreHost, report: UsageReport): Promise<boolean> {
	const budget = Number(config().get<number>("weeklyBudgetKrw") || 0);
	if (budget <= 0 || report.cost_krw <= budget) return false;
	const today = new Date().toISOString().slice(0, 10);
	const state = host.context.globalState;
	if (state.get<string>("vibeCode.budgetNotifiedDate") === today) return true;
	await state.update("vibeCode.budgetNotifiedDate", today);
	const applied = state.get<string>("vibeCode.autonomyApplied") || "";
	const lowered = applied === "auto" || applied === "yolo";
	if (lowered) await writeAutonomyPreset(host, "assist", true);
	writeAudit(host, "proxy", "budgetExceeded", { budgetKrw: budget, costKrw: report.cost_krw, autonomyBefore: applied, lowered });
	log(host, `weekly budget exceeded: ${formatKrw(report.cost_krw)} > ${formatKrw(budget)}${lowered ? " (autonomy -> assist)" : ""}`);
	void vscode.window.showWarningMessage(`vibe-coders 주간 비용 ${formatKrw(report.cost_krw)}이(가) 예산 ${formatKrw(budget)}을(를) 넘었습니다.${lowered ? " 자율성을 assist로 낮췄습니다." : ""}`, "사용량 대시보드");
	return true;
}

export function isOverBudget(report: UsageReport | undefined): boolean {
	const budget = Number(config().get<number>("weeklyBudgetKrw") || 0);
	return !!report && budget > 0 && report.cost_krw > budget;
}

export function invalidateUsageCache(): void {
	cache = undefined;
}

function printReport(output: vscode.OutputChannel, report: UsageReport): void {
	const delta = report.cost_delta_ratio;
	output.appendLine(`--- ${report.window === "monthly" ? "월간 (30일)" : "주간 (7일)"} since ${report.since} ---`);
	output.appendLine(`  요청 ${report.requests.toLocaleString("ko-KR")}회 (이전 구간 ${report.prior_requests.toLocaleString("ko-KR")}회) · 오류 ${report.errors} · 성공률 ${formatPercent(report.success_rate)}`);
	output.appendLine(`  토큰 ${formatTokens(report.tokens)} · 비용 ${formatKrw(report.cost_krw)} (이전 ${formatKrw(report.prior_cost_krw)}, ${delta >= 0 ? "+" : ""}${(delta * 100).toFixed(1)}%)`);
	output.appendLine(`  평균 지연 ${Math.round(report.avg_latency_ms)}ms · 캐시 적중 ${formatPercent(report.cache_rate)} · 위험 점수 ${report.risk_score}`);
	if (report.potential_savings_krw > 0) output.appendLine(`  절감 가능 ${formatKrw(report.potential_savings_krw)} (${report.potential_savings_model}) · 추천 ${report.recommendation_count}건`);
	if (Array.isArray(report.top_models) && report.top_models.length) {
		output.appendLine("  상위 모델:");
		for (const row of report.top_models) output.appendLine(`    - ${modelName(row)} :: ${formatKrw(Number(row.cost_krw))}${row.requests ? ` :: ${row.requests}회` : ""}${row.tokens ? ` :: ${formatTokens(Number(row.tokens))} tok` : ""}`);
	}
}

/** `Vibe Code: 사용량 리포트 보기` — weekly + monthly `/me/report` in the Output channel. */
export function registerUsageReport(host: CoreHost): void {
	const { context, output } = host;
	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.showUsageReport", async () => {
			const cp = await host.contextProxy.getInstance(context);
			const apiKey = String(cp.getValues().openAiApiKey || "");
			if (!apiKey) {
				void vscode.window.showWarningMessage("현재 provider profile에 API key가 없습니다. 먼저 vibe-coders 프록시 적용 명령을 실행하세요.");
				return;
			}
			const [weekly, monthly] = await Promise.all([fetchUsageReport(apiKey, "weekly"), fetchUsageReport(apiKey, "monthly")]);
			output.show(true);
			output.appendLine(`\n=== vibe-coders 사용량 리포트 (${proxyOrigin()}) ===`);
			for (const result of [weekly, monthly]) {
				if (result.ok && result.report) printReport(output, result.report);
				else output.appendLine(`  조회 실패: ${result.status ? `HTTP ${result.status} ` : ""}${result.error || ""}`);
			}
			output.appendLine("================================\n");
			writeAudit(host, "proxy", "showUsageReport", { weekly: weekly.ok, monthly: monthly.ok, error: weekly.error || monthly.error || "" });
			if (!weekly.ok) void vscode.window.showWarningMessage(`사용량 리포트 조회 실패: ${weekly.status ? `HTTP ${weekly.status}` : weekly.error}`);
		}),
	);
	log(host, "usage report command registered");
}
