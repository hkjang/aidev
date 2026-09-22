import React, { useEffect, useState } from "react";
import ReactDOM from "react-dom/client";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import "@fontsource-variable/noto-sans-kr";
import "./style.css";
import { ensureSecureRandomUUID } from "../../../web/src/pwa/uuid";
import {
  clearVault,
  createVault,
  listOfflineRecords,
  lockOfflineVault,
  removeOfflineRecord,
  saveOfflineRecord,
  unlockVault,
  vaultMetadata,
  vaultSupported,
  vaultUnlocked,
  type OfflineRecord,
} from "../../../web/src/pwa/vault";
import type { Doc } from "../../../web/src/api";
ensureSecureRandomUUID();
type Config = { server: string; allow_http: boolean; remember_key: boolean };
type User = { id: string; name: string };
type Workspace = { id: string; name: string };
const request = <T,>(path: string, method = "GET", data?: unknown) =>
  invoke<T>("api_request", { path, method, data: data ?? null });
function App() {
  const [tab, setTab] = useState(
    localStorage.getItem("madi-desktop-tab") || "connect",
  );
  const [config, setConfig] = useState<Config>({
    server: "",
    allow_http: false,
    remember_key: false,
  });
  const [token, setToken] = useState("");
  const [user, setUser] = useState<User | null>(null);
  const [spaces, setSpaces] = useState<Workspace[]>([]);
  const [workspace, setWorkspace] = useState("");
  const [version, setVersion] = useState("0.1.0");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [title, setTitle] = useState("");
  const [text, setText] = useState("");
  const [captureID, setCaptureID] = useState(crypto.randomUUID());
  const [passphrase, setPassphrase] = useState("");
  const [meta, setMeta] = useState<{ userID: string } | null>(null);
  const [unlocked, setUnlocked] = useState(false);
  const [records, setRecords] = useState<OfflineRecord[]>([]);
  const [docs, setDocs] = useState<Doc[]>([]);
  const [docID, setDocID] = useState("");
  const [current, setCurrent] = useState<OfflineRecord | null>(null);
  const [draftTitle, setDraftTitle] = useState("");
  const [draftText, setDraftText] = useState("");
  const [deepLink, setDeepLink] = useState("");
  const owner = user ? config.server + "|" + user.id : meta?.userID;
  async function refreshVault() {
    if (!vaultSupported()) return;
    setMeta(await vaultMetadata());
    const open = vaultUnlocked();
    setUnlocked(open);
    setRecords(open ? await listOfflineRecords() : []);
    if (!open) {
      setCurrent(null);
      setDraftTitle("");
      setDraftText("");
    }
  }
  async function session() {
    const status = await invoke<{
      config: Config;
      connected: boolean;
      version: string;
      pending_link?: string;
      shortcut_available: boolean;
    }>("connection_status");
    setConfig(status.config);
    setVersion(status.version);
    if (status.pending_link) setDeepLink(status.pending_link);
    if (!status.shortcut_available)
      setMessage(
        "전역 단축키를 등록하지 못했습니다. 다른 앱의 단축키 충돌 또는 운영체제 권한을 확인하세요.",
      );
    const remembered = localStorage.getItem(
      "madi-desktop-workspace:" + status.config.server,
    );
    if (remembered) {
      setWorkspace(remembered);
      setSpaces([
        {
          id: remembered,
          name: "기억한 워크스페이스 · " + remembered.slice(0, 8),
        },
      ]);
    }
    if (status.connected) {
      const me = await request<User>("/auth/me");
      const list = await request<Workspace[]>("/workspaces");
      setUser(me);
      setSpaces(list);
      const selected = list.some((space) => space.id === remembered)
        ? remembered!
        : list[0]?.id || "";
      setWorkspace(selected);
      localStorage.setItem(
        "madi-desktop-workspace:" + status.config.server,
        selected,
      );
    }
  }
  useEffect(() => {
    void session().catch((err) => setError(String(err)));
    void refreshVault().catch((err) => setError(String(err)));
    const change = () => {
      void refreshVault().catch((err) => setError(String(err)));
    };
    window.addEventListener("madi-vault-change", change);
    const subscriptions = [
      listen("madi-capture", () => setTab("capture")),
      listen("madi-lock", () => lockOfflineVault()),
      listen<string>("madi-deep-link", (event) => setDeepLink(event.payload)),
    ];
    return () => {
      window.removeEventListener("madi-vault-change", change);
      for (const sub of subscriptions) void sub.then((unlisten) => unlisten());
    };
  }, []);
  useEffect(() => {
    localStorage.setItem("madi-desktop-tab", tab);
  }, [tab]);
  async function run(fn: () => Promise<void>) {
    setError("");
    setMessage("");
    setBusy(true);
    try {
      await fn();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }
  async function sync(record: OfflineRecord) {
    if (!user || !owner || !vaultUnlocked(owner))
      throw new Error(
        "보관함 소유자 계정으로 서버에 연결하고 잠금을 해제하세요.",
      );
    if (record.kind === "capture") {
      await request("/captures", "POST", {
        workspace_id: record.workspace_id,
        title: record.title,
        text: record.text,
        url: record.url,
        client_request_id: record.client_request_id,
      });
      await removeOfflineRecord(record.id);
      setMessage("개인 인박스로 전송했습니다.");
    } else {
      const latest = await request<Doc>("/documents/" + record.doc.id);
      if (record.edited) {
        if (!latest.can_write)
          throw new Error(
            "현재 문서 수정 권한이 없습니다. 기기 초안은 보존합니다.",
          );
        if (latest.version !== record.doc.version)
          throw new Error(
            "서버 문서가 바뀌었습니다. 초안을 내보내 비교하세요. 자동으로 덮어쓰지 않습니다.",
          );
        const saved = await request<Doc>("/documents/" + record.doc.id, "PUT", {
          version: latest.version,
          title: record.doc.title,
          markdown: record.doc.markdown,
          block_metadata: {},
        });
        await saveOfflineRecord({
          ...record,
          doc: saved,
          edited: false,
          saved_at: new Date().toISOString(),
        });
      } else
        await saveOfflineRecord({
          ...record,
          doc: latest,
          saved_at: new Date().toISOString(),
        });
      setMessage("현재 서버 권한과 버전을 확인해 동기화했습니다.");
    }
  }
  function edit(record: OfflineRecord) {
    if (
      current &&
      (draftTitle !==
        (current.kind === "document" ? current.doc.title : current.title) ||
        draftText !==
          (current.kind === "document"
            ? current.doc.markdown
            : current.text)) &&
      !confirm("기기에 저장하지 않은 편집을 버리고 다른 기록을 여시겠습니까?")
    )
      return;
    setCurrent(record);
    setDraftTitle(record.kind === "document" ? record.doc.title : record.title);
    setDraftText(
      record.kind === "document" ? record.doc.markdown : record.text,
    );
  }
  return (
    <div className="desktop-app">
      <aside>
        <a
          className="brand"
          href="#"
          onClick={(event) => {
            event.preventDefault();
            setTab("capture");
          }}
        >
          <img src="/favicon.svg" alt="" />
          madi <small>DESKTOP</small>
        </a>
        <p>
          기록은 빠르게,
          <br />
          지식은 안전하게.
        </p>
        <nav>
          {[
            ["capture", "빠른 기록"],
            ["offline", "오프라인 보관함"],
            ["connect", "연결과 기기 설정"],
          ].map(([id, label]) => (
            <button
              key={id}
              className={tab === id ? "selected" : ""}
              onClick={() => setTab(id)}
            >
              {label}
            </button>
          ))}
        </nav>
        <div className="aside-bottom">
          <strong>{user?.name || "사내 지식 워크스페이스"}</strong>
          <span>madi v{version}</span>
          <span>Ctrl / ⌘ + Shift + Space</span>
        </div>
      </aside>
      <main>
        <header>
          <div>
            <small>MY KNOWLEDGE · MY DEVICE</small>
            <h1>
              {tab === "capture"
                ? "생각이 사라지기 전에"
                : tab === "offline"
                  ? "연결 없이도 이어지는 기록"
                  : "나의 사내 워크스페이스"}
            </h1>
          </div>
          <button
            className="secondary"
            disabled={!config.server || busy}
            onClick={() =>
              void run(() => invoke("open_workspace", { path: "/app" }))
            }
          >
            서비스 화면 열기 ↗
          </button>
        </header>
        {error && (
          <div role="alert" className="error">
            {error}
          </div>
        )}
        {message && (
          <div role="status" className="success">
            {message}
          </div>
        )}
        {deepLink && (
          <section className="card">
            <h2>문서 링크를 여시겠습니까?</h2>
            <p>설정된 사내 서버에서만 엽니다: {deepLink}</p>
            <div className="actions">
              <button
                onClick={() =>
                  void run(async () => {
                    if (deepLink === "/app/inbox") setTab("capture");
                    else await invoke("open_workspace", { path: deepLink });
                    setDeepLink("");
                  })
                }
              >
                열기
              </button>
              <button className="secondary" onClick={() => setDeepLink("")}>
                취소
              </button>
            </div>
          </section>
        )}
        {tab === "connect" ? (
          <section className="card">
            <h2>서버 연결</h2>
            <p>
              기기 기능은 개인 API 키의 워크스페이스 권한 안에서만 동작합니다.
              서비스 화면에서는 별도로 로컬 계정 또는 SSO로 로그인하세요.
            </p>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void run(async () => {
                  lockOfflineVault();
                  await invoke<User>("connect", { config, token });
                  setToken("");
                  await session();
                  setMessage("사내 서버에 연결했습니다.");
                  setTab("capture");
                });
              }}
            >
              <label htmlFor="server">madi 서버 주소</label>
              <input
                id="server"
                type="url"
                required
                placeholder="https://madi.company.local"
                value={config.server}
                onChange={(event) =>
                  setConfig({ ...config, server: event.target.value })
                }
              />
              <label className="check">
                <input
                  type="checkbox"
                  checked={config.allow_http}
                  onChange={(event) =>
                    setConfig({ ...config, allow_http: event.target.checked })
                  }
                />
                신뢰된 사내 HTTP 사용 · API 키와 문서가 암호화되지 않는 위험을
                확인했습니다.
              </label>
              <label htmlFor="token">개인 API 키</label>
              <input
                id="token"
                type="password"
                required
                autoComplete="off"
                value={token}
                onChange={(event) => setToken(event.target.value)}
              />
              <small>
                document:read, document:write 권한 · 토큰은 웹페이지에 전달하지
                않습니다.
              </small>
              <label className="check">
                <input
                  type="checkbox"
                  checked={config.remember_key}
                  onChange={(event) =>
                    setConfig({ ...config, remember_key: event.target.checked })
                  }
                />
                운영체제 키체인에 API 키 보관 · 다음 실행에서 연결 복원
              </label>
              <div className="actions">
                <button disabled={busy}>연결 확인</button>
                <button
                  type="button"
                  className="secondary"
                  onClick={() =>
                    void run(async () => {
                      lockOfflineVault();
                      await invoke("disconnect");
                      setUser(null);
                      setSpaces([]);
                      setToken("");
                      setMessage(
                        "세션 키와 키체인 항목을 삭제하고 원격 창을 닫았습니다.",
                      );
                    })
                  }
                >
                  연결 해제·키 삭제
                </button>
              </div>
            </form>
            <div className="note">
              기본값은 메모리 세션 연결입니다. OS 키체인 저장 실패 시 평문
              파일에 저장하지 않습니다. TLS 인증서 검증을 끄지 않으며, 사내 CA는
              운영체제 신뢰 저장소에 등록하세요. 닫기 버튼은 트레이로 숨깁니다.
              완전히 종료하려면 트레이에서 ‘madi 종료’를 선택하세요.
            </div>
          </section>
        ) : tab === "capture" ? (
          <section className="card">
            <h2>내 인박스에 빠른 기록</h2>
            <p>
              짧은 생각, 회의 중 메모, 로컬 Markdown 파일을 먼저 모으고 나중에
              분류하세요.
            </p>
            <label htmlFor="workspace">워크스페이스</label>
            <select
              id="workspace"
              value={workspace}
              onChange={(event) => {
                setWorkspace(event.target.value);
                localStorage.setItem(
                  "madi-desktop-workspace:" + config.server,
                  event.target.value,
                );
                setCaptureID(crypto.randomUUID());
              }}
            >
              <option value="">서버 연결 후 선택하세요</option>
              {spaces.map((value) => (
                <option key={value.id} value={value.id}>
                  {value.name}
                </option>
              ))}
            </select>
            <label htmlFor="title">기록 제목</label>
            <input
              id="title"
              maxLength={250}
              value={title}
              onChange={(event) => {
                setTitle(event.target.value);
                setCaptureID(crypto.randomUUID());
              }}
            />
            <label htmlFor="text">기록 내용 · Markdown</label>
            <textarea
              id="text"
              rows={11}
              maxLength={900000}
              value={text}
              onChange={(event) => {
                setText(event.target.value);
                setCaptureID(crypto.randomUUID());
              }}
            />
            <div className="actions">
              <button
                disabled={busy || !user || !workspace || !title.trim()}
                onClick={() =>
                  void run(async () => {
                    const doc = await request<Doc>("/captures", "POST", {
                      workspace_id: workspace,
                      title,
                      text,
                      client_request_id: captureID,
                    });
                    setTitle("");
                    setText("");
                    setCaptureID(crypto.randomUUID());
                    setMessage("개인 인박스에 저장했습니다: " + doc.title);
                  })
                }
              >
                개인 인박스에 저장
              </button>
              <button
                className="secondary"
                disabled={busy || !unlocked || !workspace || !title.trim()}
                onClick={() =>
                  void run(async () => {
                    if (!owner || !vaultUnlocked(owner))
                      throw new Error("본인 보관함을 먼저 잠금 해제하세요.");
                    await saveOfflineRecord({
                      kind: "capture",
                      id: "capture:" + captureID,
                      workspace_id: workspace,
                      title,
                      text,
                      url: "",
                      client_request_id: captureID,
                      created_at: new Date().toISOString(),
                    });
                    setTitle("");
                    setText("");
                    setCaptureID(crypto.randomUUID());
                    setMessage(
                      "암호화 기기 보관함에 전송 대기로 저장했습니다.",
                    );
                  })
                }
              >
                기기에 전송 대기 보관
              </button>
            </div>
            <div className="actions">
              <button
                className="secondary"
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    const file = await invoke<{
                      title: string;
                      text: string;
                    } | null>("import_markdown");
                    if (file) {
                      setTitle(file.title);
                      setText(file.text);
                      setCaptureID(crypto.randomUUID());
                      setMessage(
                        "선택한 UTF-8 파일을 가져왔습니다. 저장 전에 확인하세요.",
                      );
                    }
                  })
                }
              >
                로컬 Markdown 파일 가져오기
              </button>
              <button
                className="secondary"
                disabled={busy || !text}
                onClick={() =>
                  void run(async () => {
                    if (await invoke("export_markdown", { title, text }))
                      setMessage("선택한 위치에 Markdown 파일을 저장했습니다.");
                  })
                }
              >
                Markdown 파일로 내보내기
              </button>
            </div>
            <div className="note">
              파일은 사용자가 선택한 한 개만 읽고, 내보내기는 OS 저장
              대화상자에서 지정한 위치에만 기록합니다. 기록 내용은 ‘저장’ 또는
              ‘기기 보관’을 누르기 전까지 메모리에만 유지됩니다.
            </div>
          </section>
        ) : (
          <>
            <section className="card">
              <h2>암호화 오프라인 보관함</h2>
              <p>
                선택한 문서와 미전송 기록만 AES-256-GCM으로 보관합니다.
                첨부·웹페이지·API 응답은 자동 저장하지 않습니다. 패스프레이즈는
                서버나 디스크에 저장하지 않으며, 재실행·잠금·5분 미사용 시 다시
                입력해야 합니다.
              </p>
              {!vaultSupported() ? (
                <p role="status">
                  현재 웹뷰가 안전한 WebCrypto 보관함을 지원하지 않습니다. 최신
                  운영체제 웹뷰를 사용하세요. 온라인 기록과 파일 가져오기는 계속
                  사용할 수 있습니다.
                </p>
              ) : unlocked ? (
                <div className="actions">
                  <button
                    className="secondary"
                    onClick={() => lockOfflineVault()}
                  >
                    보관함 잠그기
                  </button>
                </div>
              ) : (
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    void run(async () => {
                      if (meta) await unlockVault(passphrase, owner);
                      else {
                        if (!owner)
                          throw new Error(
                            "먼저 서버에 연결해 보관함 소유자를 확인하세요.",
                          );
                        await createVault(owner, passphrase);
                      }
                      setPassphrase("");
                    });
                  }}
                >
                  <label htmlFor="passphrase">
                    기기 보관함 암호 · 12자 이상
                  </label>
                  <input
                    id="passphrase"
                    type="password"
                    minLength={12}
                    required
                    autoComplete="off"
                    value={passphrase}
                    onChange={(event) => setPassphrase(event.target.value)}
                  />
                  <div className="actions">
                    <button disabled={busy}>
                      {meta ? "보관함 잠금 해제" : "암호화 보관함 만들기"}
                    </button>
                  </div>
                </form>
              )}
              <div className="note">
                오프라인 사본은 저장 이후 권한 회수를 실시간 확인할 수 없습니다.
                기밀 문서는 조직의 반출 정책을 확인하세요. 암호 분실 시 복구할
                수 없습니다. 보관함은 이 기기와 사내 서버 사용자 조합에
                귀속됩니다.
              </div>
              {meta && (
                <button
                  className="danger"
                  onClick={() => {
                    if (
                      prompt(
                        "미전송 기록을 포함해 기기 보관함을 영구 삭제합니다. 서버 원본은 바꾸지 않습니다. 확인: 기기 보관함 삭제",
                      ) === "기기 보관함 삭제"
                    )
                      void run(async () => {
                        await clearVault();
                        setMessage(
                          "기기 보관함을 영구 삭제했습니다. 복구할 수 없습니다.",
                        );
                      });
                  }}
                >
                  기기 보관함 삭제
                </button>
              )}
            </section>
            {unlocked && (
              <section className="card">
                <h2>명시적으로 저장한 기록</h2>
                <div className="actions">
                  <button
                    className="secondary"
                    disabled={busy || !user || !workspace}
                    onClick={() =>
                      void run(async () => {
                        setDocs(
                          await request<Doc[]>(
                            "/documents?workspace_id=" +
                              encodeURIComponent(workspace) +
                              "&limit=100",
                          ),
                        );
                      })
                    }
                  >
                    현재 권한으로 문서 목록 조회
                  </button>
                </div>
                <label htmlFor="document">기기에 보관할 문서</label>
                <select
                  id="document"
                  value={docID}
                  onChange={(event) => setDocID(event.target.value)}
                >
                  <option value="">문서를 선택하세요</option>
                  {docs.map((doc) => (
                    <option key={doc.id} value={doc.id}>
                      {doc.title}
                    </option>
                  ))}
                </select>
                <div className="actions">
                  <button
                    disabled={busy || !docID}
                    onClick={() =>
                      void run(async () => {
                        if (!owner || !vaultUnlocked(owner))
                          throw new Error("보관함 소유자 계정으로 연결하세요.");
                        if (
                          records.some(
                            (record) =>
                              record.kind === "document" &&
                              record.doc.id === docID &&
                              record.edited,
                          )
                        )
                          throw new Error(
                            "기존 기기 초안이 있습니다. 먼저 전송 또는 내보내기 후 삭제하세요.",
                          );
                        const doc = await request<Doc>("/documents/" + docID);
                        await saveOfflineRecord({
                          kind: "document",
                          id: "document:" + doc.id,
                          doc,
                          edited: false,
                          saved_at: new Date().toISOString(),
                        });
                        setMessage("선택한 문서를 암호화해 저장했습니다.");
                      })
                    }
                  >
                    선택 문서 기기에 보관
                  </button>
                </div>
                {records.map((record) => (
                  <div className="record" key={record.id}>
                    <div>
                      <strong>
                        {record.kind === "document"
                          ? record.doc.title
                          : record.title}
                      </strong>
                      <small>
                        {record.kind === "capture"
                          ? "인박스 전송 대기"
                          : record.edited
                            ? "오프라인 수정 초안"
                            : "문서 기기 사본"}
                      </small>
                    </div>
                    <button className="secondary" onClick={() => edit(record)}>
                      열기
                    </button>
                    <button
                      className="secondary"
                      disabled={busy || !user}
                      onClick={() => void run(() => sync(record))}
                    >
                      서버 동기화
                    </button>
                    <button
                      className="secondary"
                      onClick={() => {
                        if (
                          confirm(
                            "이 기기의 사본과 미전송 변경을 삭제하시겠습니까? 서버 원본은 유지됩니다.",
                          )
                        )
                          void run(() => removeOfflineRecord(record.id));
                      }}
                    >
                      삭제
                    </button>
                  </div>
                ))}
                {current && (
                  <div className="editor">
                    <label htmlFor="draft-title">기기 초안 제목</label>
                    <input
                      id="draft-title"
                      value={draftTitle}
                      onChange={(event) => setDraftTitle(event.target.value)}
                    />
                    <label htmlFor="draft-text">
                      기기 초안 · Markdown 원문
                    </label>
                    <textarea
                      id="draft-text"
                      rows={10}
                      value={draftText}
                      onChange={(event) => setDraftText(event.target.value)}
                    />
                    <div className="actions">
                      <button
                        disabled={busy}
                        onClick={() =>
                          void run(async () => {
                            const next: OfflineRecord =
                              current.kind === "document"
                                ? {
                                    ...current,
                                    doc: {
                                      ...current.doc,
                                      title: draftTitle,
                                      markdown: draftText,
                                    },
                                    edited: true,
                                  }
                                : {
                                    ...current,
                                    title: draftTitle,
                                    text: draftText,
                                    client_request_id: crypto.randomUUID(),
                                  };
                            await saveOfflineRecord(next);
                            setCurrent(next);
                            setMessage("기기 초안을 암호화해 저장했습니다.");
                          })
                        }
                      >
                        기기 초안 저장
                      </button>
                      <button
                        className="secondary"
                        onClick={() =>
                          void run(async () => {
                            await invoke("export_markdown", {
                              title: draftTitle,
                              text: draftText,
                            });
                          })
                        }
                      >
                        Markdown 내보내기
                      </button>
                    </div>
                  </div>
                )}
              </section>
            )}
          </>
        )}
      </main>
    </div>
  );
}
ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
