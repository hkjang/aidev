import { describe, expect, it } from "vitest";
import { compareVersions } from "../../src/util/semver";
import { proxyRootUrl } from "../../src/features/vibe-coders-proxy";

describe("compareVersions", () => {
	it("compares numerically per segment", () => {
		expect(compareVersions("1.1.0", "1.0.0")).toBeGreaterThan(0);
		expect(compareVersions("1.0.0", "1.10.0")).toBeLessThan(0);
		expect(compareVersions("1.0", "1.0.0")).toBe(0);
		expect(["1.10.0", "1.2.0", "1.9.9"].sort(compareVersions)).toEqual(["1.2.0", "1.9.9", "1.10.0"]);
	});
});

describe("proxyRootUrl", () => {
	it("strips /v1 and query parts", () => {
		expect(proxyRootUrl("http://localhost:8080/v1")).toBe("http://localhost:8080");
		expect(proxyRootUrl("https://proxy.example.com/v1/?x=1")).toBe("https://proxy.example.com");
		expect(proxyRootUrl("not a url")).toBe("http://localhost:8080");
	});
});
