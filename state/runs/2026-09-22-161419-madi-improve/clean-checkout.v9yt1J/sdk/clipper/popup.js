const api = globalThis.chrome || globalThis.browser;
const $ = (id) => document.getElementById(id);
async function send(message) {
  const value = await api.runtime.sendMessage(message);
  if (value?.error) throw new Error(value.error);
  return value.result;
}
function error(value) {
  $("error").textContent = value?.message || String(value);
  $("error").hidden = false;
}
function preview(draft) {
  $("form").hidden = !draft;
  if (!draft) return;
  $("title").value = draft.title;
  $("text").value = draft.text;
  $("preview-image").hidden = !draft.image;
  $("preview-image").src = draft.image || "";
  $("source").textContent = "원본: " + draft.url;
  $("success").textContent = "";
}
async function busy(fn) {
  $("error").hidden = true;
  for (const button of document.querySelectorAll("button"))
    button.disabled = true;
  try {
    await fn();
  } catch (e) {
    error(e);
  } finally {
    for (const button of document.querySelectorAll("button"))
      button.disabled = false;
  }
}
$("capture").onclick = () =>
  busy(async () =>
    preview((await send({ action: "capture", mode: $("mode").value })).draft),
  );
$("settings").onclick = () => api.runtime.openOptionsPage();
$("form").onsubmit = (event) => {
  event.preventDefault();
  void busy(async () => {
    const result = await send({
      action: "save",
      title: $("title").value,
      text: $("text").value,
    });
    $("form").hidden = true;
    const link = document.createElement("a");
    link.href = result.url;
    link.target = "_blank";
    link.rel = "noreferrer";
    link.textContent = "저장한 문서 열기";
    $("success").replaceChildren(
      document.createTextNode("개인 인박스에 저장했습니다. "),
      link,
    );
  });
};
send({ action: "status" })
  .then((value) => {
    $("connection").textContent = value.connected
      ? "연결 서버: " + value.server
      : "연결 설정이 필요합니다. 저장할 내용을 먼저 가져올 수도 있습니다.";
    preview(value.draft);
  })
  .catch(error);
