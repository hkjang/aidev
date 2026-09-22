/** randomUUID is secure-context-only; getRandomValues is also available on intranet HTTP. */
export function ensureSecureRandomUUID() {
  if (
    globalThis.crypto &&
    typeof globalThis.crypto.randomUUID !== "function" &&
    typeof globalThis.crypto.getRandomValues === "function"
  ) {
    Object.defineProperty(globalThis.crypto, "randomUUID", {
      configurable: true,
      value: () => {
        const value = crypto.getRandomValues(new Uint8Array(16));
        value[6] = (value[6] & 15) | 64;
        value[8] = (value[8] & 63) | 128;
        const hex = Array.from(value, (byte) =>
          byte.toString(16).padStart(2, "0"),
        ).join("");
        return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
      },
    });
  }
}
