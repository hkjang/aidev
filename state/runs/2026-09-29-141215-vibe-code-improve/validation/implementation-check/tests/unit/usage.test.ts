import { describe, expect, it } from "vitest";
import { formatKrw, formatPercent, formatTokens, usageSummaryLine, UsageReport } from "../../src/features/usage";

const report = { window: "weekly", cost_krw: 12345.6, tokens: 45200, requests: 10 } as UsageReport;

describe("usage formatting", () => {
	it("formats KRW, tokens and percentages", () => {
		expect(formatKrw(12345.6)).toBe("₩12,346");
		expect(formatKrw(undefined)).toBe("₩0");
		expect(formatTokens(999)).toBe("999");
		expect(formatTokens(45200)).toBe("45.2k");
		expect(formatTokens(2_500_000)).toBe("2.50M");
		expect(formatPercent(0.9876)).toBe("98.8%");
	});
	it("builds the status bar line", () => {
		expect(usageSummaryLine(report)).toBe("₩12,346 · 45.2k tok (주간)");
		expect(usageSummaryLine({ ...report, window: "monthly" })).toBe("₩12,346 · 45.2k tok (월간)");
	});
});
