import * as crypto from "node:crypto";
import * as vscode from "vscode";
import type { CoreHost } from "../core/host";
import { log } from "../log";
import { fetchUsageReport, formatKrw, formatPercent, formatTokens, modelName, UsageReport, UsageResult } from "./usage";
import { writeAudit } from "./workspace";

const escapeHtml = (value: unknown): string =>
	String(value ?? "").replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch] as string);

function statTile(label: string, value: string, note = ""): string {
	return `<div class="tile"><div class="label">${escapeHtml(label)}</div><div class="value">${escapeHtml(value)}</div>${note ? `<div class="note">${escapeHtml(note)}</div>` : ""}</div>`;
}

/**
 * One window (weekly/monthly) as a KPI row plus a single-hue bar list of top models by cost.
 * Magnitude is one hue (VS Code's chart blue); every bar carries its value as a direct label,
 * and the same numbers are available as a table for screen readers and copy/paste.
 */
function renderWindow(result: UsageResult): string {
	if (!result.ok || !result.report) {
		return `<p class="error">조회 실패: ${escapeHtml(result.status ? `HTTP ${result.status} ` : "")}${escapeHtml(result.error || "")}</p>`;
	}
	const r: UsageReport = result.report;
	const delta = r.cost_delta_ratio;
	const deltaText = `${delta >= 0 ? "+" : ""}${(delta * 100).toFixed(1)}% vs 이전 구간 (${formatKrw(r.prior_cost_krw)})`;
	const models = Array.isArray(r.top_models) ? r.top_models : [];
	const maxCost = Math.max(1, ...models.map((m) => Number(m.cost_krw) || 0));
	const bars = models
		.map((m) => {
			const cost = Number(m.cost_krw) || 0;
			const pct = Math.max(2, Math.round((cost / maxCost) * 100));
			return `<li title="${escapeHtml(modelName(m))}: ${escapeHtml(formatKrw(cost))}${m.requests ? `, ${escapeHtml(m.requests)}회` : ""}">
				<span class="name">${escapeHtml(modelName(m))}</span>
				<span class="track"><span class="bar" style="width:${pct}%"></span></span>
				<span class="num">${escapeHtml(formatKrw(cost))}</span>
			</li>`;
		})
		.join("");
	const rows = models
		.map((m) => `<tr><td>${escapeHtml(modelName(m))}</td><td class="num">${escapeHtml(formatKrw(Number(m.cost_krw)))}</td><td class="num">${escapeHtml(m.requests ?? "")}</td><td class="num">${escapeHtml(m.tokens ? formatTokens(Number(m.tokens)) : "")}</td></tr>`)
		.join("");
	return `
		<div class="tiles">
			${statTile("비용", formatKrw(r.cost_krw), deltaText)}
			${statTile("요청", r.requests.toLocaleString("ko-KR") + "회", `이전 구간 ${r.prior_requests.toLocaleString("ko-KR")}회`)}
			${statTile("토큰", formatTokens(r.tokens))}
			${statTile("성공률", formatPercent(r.success_rate), `오류 ${r.errors}건`)}
			${statTile("평균 지연", `${Math.round(r.avg_latency_ms)} ms`, `캐시 적중 ${formatPercent(r.cache_rate)}`)}
			${statTile("절감 가능", formatKrw(r.potential_savings_krw), r.potential_savings_model ? `${r.potential_savings_model} · 추천 ${r.recommendation_count}건` : "")}
		</div>
		<h3>상위 모델 (비용)</h3>
		${models.length ? `<ul class="bars">${bars}</ul><details><summary>표로 보기</summary><table><thead><tr><th>모델</th><th class="num">비용</th><th class="num">요청</th><th class="num">토큰</th></tr></thead><tbody>${rows}</tbody></table></details>` : `<p class="muted">집계된 모델 호출이 없습니다.</p>`}
		<p class="muted">since ${escapeHtml(r.since)} · user ${escapeHtml(r.user_id)}</p>`;
}

function renderPage(nonce: string, origin: string, weekly: UsageResult, monthly: UsageResult, fetchedAt: string): string {
	return `<!DOCTYPE html>
<html lang="ko">
<head>
<meta charset="UTF-8">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'nonce-${nonce}'; script-src 'nonce-${nonce}';">
<title>vibe-coders 사용량</title>
<style nonce="${nonce}">
	body { font-family: var(--vscode-font-family); color: var(--vscode-foreground); background: var(--vscode-editor-background); padding: 16px 20px; line-height: 1.4; }
	h1 { font-size: 1.3em; margin: 0 0 4px; } h2 { font-size: 1.05em; margin: 20px 0 8px; border-bottom: 1px solid var(--vscode-panel-border); padding-bottom: 4px; } h3 { font-size: 0.95em; margin: 16px 0 6px; }
	.toolbar { display: flex; gap: 8px; align-items: center; margin: 6px 0 12px; } .muted { color: var(--vscode-descriptionForeground); font-size: 0.85em; } .error { color: var(--vscode-errorForeground); }
	button { background: var(--vscode-button-background); color: var(--vscode-button-foreground); border: none; padding: 4px 12px; border-radius: 2px; cursor: pointer; } button:hover { background: var(--vscode-button-hoverBackground); }
	.tiles { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 8px; }
	.tile { background: var(--vscode-editorWidget-background); border: 1px solid var(--vscode-panel-border); border-radius: 4px; padding: 10px 12px; }
	.tile .label { color: var(--vscode-descriptionForeground); font-size: 0.8em; } .tile .value { font-size: 1.4em; font-weight: 600; margin: 2px 0; } .tile .note { color: var(--vscode-descriptionForeground); font-size: 0.78em; }
	ul.bars { list-style: none; padding: 0; margin: 0; } ul.bars li { display: grid; grid-template-columns: minmax(120px, 30%) 1fr 90px; align-items: center; gap: 8px; padding: 3px 0; }
	ul.bars li:hover { background: var(--vscode-list-hoverBackground); } .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.track { height: 8px; background: var(--vscode-editorWidget-background); border-radius: 0 4px 4px 0; overflow: hidden; } .bar { display: block; height: 100%; background: var(--vscode-charts-blue); border-radius: 0 4px 4px 0; }
	.num { text-align: right; font-variant-numeric: tabular-nums; }
	table { border-collapse: collapse; margin-top: 6px; } th, td { padding: 3px 10px 3px 0; text-align: left; border-bottom: 1px solid var(--vscode-panel-border); } th.num, td.num { text-align: right; }
	details summary { cursor: pointer; color: var(--vscode-textLink-foreground); font-size: 0.85em; margin-top: 6px; }
</style>
</head>
<body>
	<h1>vibe-coders 사용량</h1>
	<div class="toolbar"><span class="muted">${escapeHtml(origin)} · 조회 ${escapeHtml(fetchedAt)}</span><button id="refresh">새로고침</button><button id="report">Output에 리포트</button></div>
	<h2>주간 (7일)</h2>${renderWindow(weekly)}
	<h2>월간 (30일)</h2>${renderWindow(monthly)}
	<script nonce="${nonce}">
		const vscode = acquireVsCodeApi();
		document.getElementById("refresh").addEventListener("click", () => vscode.postMessage({ type: "refresh" }));
		document.getElementById("report").addEventListener("click", () => vscode.postMessage({ type: "report" }));
	</script>
</body>
</html>`;
}

/** `Vibe Code: 사용량 대시보드` — a webview panel over vibe-coders `/me/report` (weekly + monthly). */
export function registerUsageDashboard(host: CoreHost): void {
	const { context } = host;
	let panel: vscode.WebviewPanel | undefined;

	const load = async () => {
		if (!panel) return;
		const cp = await host.contextProxy.getInstance(context);
		const vals = cp.getValues();
		const apiKey = String(vals.openAiApiKey || "");
		const origin = (() => {
			try {
				return new URL(String(vals.openAiBaseUrl || "http://localhost:8080/v1")).origin;
			} catch {
				return "http://localhost:8080";
			}
		})();
		const nonce = crypto.randomBytes(16).toString("base64");
		if (!apiKey) {
			const missing: UsageResult = { ok: false, error: "현재 provider profile에 API key가 없습니다. 먼저 vibe-coders 프록시 적용 명령을 실행하세요." };
			panel.webview.html = renderPage(nonce, origin, missing, missing, new Date().toLocaleTimeString("ko-KR"));
			return;
		}
		const [weekly, monthly] = await Promise.all([fetchUsageReport(apiKey, "weekly"), fetchUsageReport(apiKey, "monthly")]);
		panel.webview.html = renderPage(nonce, origin, weekly, monthly, new Date().toLocaleTimeString("ko-KR", { hour12: false }));
		writeAudit(host, "proxy", "openUsageDashboard", { weekly: weekly.ok, monthly: monthly.ok, error: weekly.error || monthly.error || "" });
	};

	context.subscriptions.push(
		vscode.commands.registerCommand("vibe-code.openUsageDashboard", async () => {
			if (panel) {
				panel.reveal();
				await load();
				return;
			}
			panel = vscode.window.createWebviewPanel("vibe-code.usageDashboard", "vibe-coders 사용량", vscode.ViewColumn.Beside, { enableScripts: true, localResourceRoots: [] });
			panel.iconPath = new vscode.ThemeIcon("graph");
			panel.onDidDispose(() => (panel = undefined));
			panel.webview.onDidReceiveMessage(async (message: { type?: string }) => {
				if (message?.type === "refresh") await load();
				else if (message?.type === "report") await vscode.commands.executeCommand("vibe-code.showUsageReport");
			});
			await load();
		}),
	);
	log(host, "usage dashboard command registered");
}
