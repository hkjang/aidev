import { serverOrigin } from "./common.js";
const api = globalThis.chrome || globalThis.browser;
const $ = (id) => document.getElementById(id);
async function send(message) {
  const value = await api.runtime.sendMessage(message);
  if (value?.error) throw new Error(value.error);
  return value.result;
}
function error(value) {
  $("error").textContent = value.message || String(value);
  $("error").hidden = false;
}
function workspaces(items, current = "") {
  $("workspace").replaceChildren();
  for (const workspace of items) {
    const option = document.createElement("option");
    option.value = workspace.id;
    option.textContent = workspace.name;
    $("workspace").append(option);
  }
  $("workspace").value = current || items[0]?.id || "";
  $("workspace-form").hidden = !items.length;
}
$("connect").onsubmit = async (event) => {
  event.preventDefault();
  $("error").hidden = true;
  try {
    const server = serverOrigin($("server").value, $("http").checked);
    if (!(await api.permissions.request({ origins: [server + "/*"] })))
      throw new Error("서버 연결 권한을 허용해야 사용할 수 있습니다.");
    const result = await send({
      action: "configure",
      server,
      token: $("token").value,
      allowHTTP: $("http").checked,
    });
    $("token").value = "";
    workspaces(result.workspaces);
    $("status").textContent =
      "서버에 연결했습니다. 기본 워크스페이스를 저장하세요.";
  } catch (e) {
    error(e);
  }
};
$("workspace-form").onsubmit = async (event) => {
  event.preventDefault();
  try {
    await send({ action: "workspace", id: $("workspace").value });
    $("status").textContent =
      "워크스페이스를 저장했습니다. 웹페이지에서 madi 확장 아이콘을 눌러보세요.";
  } catch (e) {
    error(e);
  }
};
$("disconnect").onclick = async () => {
  try {
    await send({ action: "disconnect" });
    $("token").value = "";
    $("workspace-form").hidden = true;
    $("status").textContent =
      "API 키·임시 미리보기를 삭제하고 서버 접근 권한을 회수했습니다.";
  } catch (e) {
    error(e);
  }
};
send({ action: "status" })
  .then(async (state) => {
    $("server").value = state.server || "";
    $("http").checked = !!state.allowHTTP;
    if (state.connected)
      workspaces(
        (await send({ action: "workspaces" })).workspaces,
        state.workspace_id,
      );
  })
  .catch(error);
