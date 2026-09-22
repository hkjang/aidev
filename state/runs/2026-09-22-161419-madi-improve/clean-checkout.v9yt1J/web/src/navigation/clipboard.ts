export async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return;
    } catch {}
  }
  const area = document.createElement("textarea"),
    previous = document.activeElement as HTMLElement | null;
  area.value = value;
  area.readOnly = true;
  area.style.cssText = "position:fixed;left:-10000px;top:0";
  document.body.appendChild(area);
  try {
    area.select();
    if (!document.execCommand("copy"))
      throw new Error(
        "브라우저가 복사를 허용하지 않습니다. 주소를 직접 선택해 복사하세요.",
      );
  } finally {
    area.remove();
    previous?.focus();
  }
}
