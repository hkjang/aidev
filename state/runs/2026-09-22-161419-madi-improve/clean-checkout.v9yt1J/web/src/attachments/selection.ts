/** textarea normalizes CRLF/CR to LF; citations keep canonical UTF-8 bytes. */
export function attachmentSelectionRange(raw: string, start: number, end: number): readonly [number, number] | null {
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(end) || start < 0 || end <= start) return null;
  let normalized = 0, first = -1, last = -1;
  for (let i = 0; i <= raw.length; i++) {
    if (normalized === start && first < 0) first = i;
    if (normalized === end) { last = i; break; }
    if (raw[i] === "\r" && raw[i + 1] === "\n") i++;
    normalized++;
  }
  const splitSurrogate = (i: number) => i > 0 && i < raw.length && /[\uD800-\uDBFF]/.test(raw[i - 1]) && /[\uDC00-\uDFFF]/.test(raw[i]);
  return first >= 0 && last > first && !splitSurrogate(first) && !splitSurrogate(last) ? [first, last] : null;
}
