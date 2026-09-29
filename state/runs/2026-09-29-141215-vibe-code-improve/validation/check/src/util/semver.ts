/** Numeric dotted-version compare: negative if a < b, positive if a > b, 0 if equal. */
export function compareVersions(a: string, b: string): number {
	const ap = a.split(".").map(Number);
	const bp = b.split(".").map(Number);
	for (let i = 0; i < Math.max(ap.length, bp.length); i++) {
		const x = ap[i] || 0;
		const y = bp[i] || 0;
		if (x !== y) return x - y;
	}
	return 0;
}
