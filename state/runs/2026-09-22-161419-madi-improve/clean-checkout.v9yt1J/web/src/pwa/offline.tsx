import React, { useEffect, useState } from "react";
import ReactDOM from "react-dom/client";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import "@fontsource-variable/noto-sans-kr";
import "../styles.css";
import "./pwa.css";
import { Button, Empty, ErrorBox, Field, Modal } from "../ui";
import { downloadText } from "../api";
import { ensureSecureRandomUUID } from "./uuid";
import {
  listOfflineRecords,
  lockOfflineVault,
  saveOfflineRecord,
  unlockVault,
  vaultUnlocked,
  type OfflineRecord,
} from "./vault";
ensureSecureRandomUUID();
function OfflineApp() {
  const [opened, setOpened] = useState(vaultUnlocked());
  const [password, setPassword] = useState("");
  const [records, setRecords] = useState<OfflineRecord[]>([]);
  const [current, setCurrent] = useState<OfflineRecord | null>(null);
  const [title, setTitle] = useState("");
  const [text, setText] = useState("");
  const [workspaceID, setWorkspaceID] = useState(
    new URLSearchParams(location.search).get("workspace_id") || "",
  );
  const [capture, setCapture] = useState(false);
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState(false);
  const [message, setMessage] = useState("");
  async function refresh() {
    const open = vaultUnlocked();
    setOpened(open);
    if (open) setRecords(await listOfflineRecords());
    else {
      setRecords([]);
      setCurrent(null);
      setTitle("");
      setText("");
      setCapture(false);
    }
  }
  useEffect(() => {
    const changed = () => {
      void refresh().catch(setError);
    };
    changed();
    window.addEventListener("madi-vault-change", changed);
    return () => window.removeEventListener("madi-vault-change", changed);
  }, []);
  async function run(fn: () => Promise<void>) {
    setError(undefined);
    setMessage("");
    setBusy(true);
    try {
      await fn();
    } catch (err) {
      setError(err);
    } finally {
      setBusy(false);
    }
  }
  const dirty =
    !!current &&
    (title !==
      (current.kind === "document" ? current.doc.title : current.title) ||
      text !==
        (current.kind === "document" ? current.doc.markdown : current.text));
  useEffect(() => {
    const before = (event: BeforeUnloadEvent) => {
      if (dirty) {
        event.preventDefault();
      }
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [dirty]);
  function select(record: OfflineRecord) {
    if (
      dirty &&
      !window.confirm(
        "저장하지 않은 편집 내용을 버리고 다른 문서를 여시겠습니까?",
      )
    )
      return;
    setCurrent(record);
    setTitle(record.kind === "document" ? record.doc.title : record.title);
    setText(record.kind === "document" ? record.doc.markdown : record.text);
    setPreview(false);
  }
  return (
    <div className="offline-shell">
      <header>
        <img src="/favicon.svg" alt="" />
        <div>
          <h1>madi 기기 보관함</h1>
          <p className="muted">암호화된 내 문서 · 서버 자동 전송 없음</p>
        </div>
      </header>
      <div className="offline-note">
        현재 화면은 저장된 기기 사본만 엽니다. 첨부·외부 이미지·플러그인은
        실행하지 않습니다. 수정 내용은 ‘기기 초안 저장’을 누른 뒤, 온라인 상태의
        ‘기기와 오프라인’에서 직접 전송하세요.
      </div>
      <ErrorBox error={error} />
      {message && (
        <p role="status" className="notice">
          {message}
        </p>
      )}
      <main>
        {!opened ? (
          <section className="device-card">
            <h2>암호로 잠금 해제</h2>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void run(async () => {
                  await unlockVault(password);
                  setPassword("");
                });
              }}
            >
              <Field label="기기 보관함 암호">
                <input
                  type="password"
                  required
                  autoComplete="off"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </Field>
              <Button disabled={busy}>보관함 잠금 해제</Button>
            </form>
            <p>
              보관함이 없으면 서버에 연결한 뒤 ‘개인화 → 기기와 오프라인’에서
              먼저 만드세요.
            </p>
            <a href="/app/devices">서버에 연결하기</a>
          </section>
        ) : (
          <>
            <div className="offline-actions">
              <Button variant="secondary" onClick={() => lockOfflineVault()}>
                보관함 잠그기
              </Button>
              <Button onClick={() => setCapture(true)}>빠른 임시 기록</Button>
              <a className="button secondary" href="/app/devices">
                온라인 전송·관리 화면
              </a>
            </div>
            <div className="device-grid" style={{ marginTop: 24 }}>
              <section className="device-card">
                <h2>기기에 보관한 기록</h2>
                {records.length ? (
                  records.map((record) => (
                    <div className="offline-record" key={record.id}>
                      <div>
                        <strong>
                          {record.kind === "document"
                            ? record.doc.title
                            : record.title}
                        </strong>
                        <small>
                          {record.kind === "document"
                            ? record.edited
                              ? "수정 초안"
                              : "저장된 문서"
                            : "인박스 전송 대기"}
                        </small>
                      </div>
                      <Button variant="ghost" onClick={() => select(record)}>
                        열기
                      </Button>
                    </div>
                  ))
                ) : (
                  <Empty
                    title="아직 보관한 기록이 없습니다"
                    text="빠른 임시 기록을 쓰거나 온라인에서 문서를 선택해 저장하세요."
                  />
                )}
              </section>
              <section className="device-card">
                {current ? (
                  <>
                    <Field label="제목">
                      <input
                        maxLength={300}
                        value={title}
                        onChange={(event) => setTitle(event.target.value)}
                      />
                    </Field>
                    <div className="offline-actions">
                      <Button
                        variant="secondary"
                        onClick={() => setPreview(!preview)}
                      >
                        {preview ? "편집하기" : "읽기 미리보기"}
                      </Button>
                      <Button
                        variant="secondary"
                        onClick={() =>
                          downloadText(
                            (title.replace(/[^\p{L}\p{N}._ -]/gu, "_") ||
                              "madi") + ".md",
                            text,
                          )
                        }
                      >
                        Markdown 내보내기
                      </Button>
                    </div>
                    {preview ? (
                      <div className="offline-preview">
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          skipHtml
                          components={{
                            img: () => <span>[이미지 · 온라인에서 확인]</span>,
                            a: ({ children }) => (
                              <span>{children} [링크 · 온라인에서 확인]</span>
                            ),
                          }}
                        >
                          {text}
                        </ReactMarkdown>
                      </div>
                    ) : (
                      <Field label="Markdown 기기 초안">
                        <textarea
                          className="offline-editor"
                          value={text}
                          onChange={(event) => setText(event.target.value)}
                          maxLength={900000}
                        />
                      </Field>
                    )}
                    <Button
                      disabled={busy || !dirty}
                      onClick={() =>
                        void run(async () => {
                          const next: OfflineRecord =
                            current.kind === "document"
                              ? {
                                  ...current,
                                  doc: {
                                    ...current.doc,
                                    title,
                                    markdown: text,
                                  },
                                  edited: true,
                                }
                              : {
                                  ...current,
                                  title,
                                  text,
                                  client_request_id: crypto.randomUUID(),
                                };
                          await saveOfflineRecord(next);
                          setCurrent(next);
                          setMessage(
                            "이 기기에 암호화해 저장했습니다. 서버에는 아직 반영하지 않았습니다.",
                          );
                        })
                      }
                    >
                      기기 초안 저장
                    </Button>
                  </>
                ) : (
                  <Empty
                    title="읽을 기록을 선택하세요"
                    text="기기에서 편집해도 서버 원본은 자동 변경되지 않습니다."
                  />
                )}
              </section>
            </div>
          </>
        )}
      </main>
      <Modal
        open={capture}
        onOpenChange={setCapture}
        title="빠른 임시 기록"
        description="암호화해 보관한 뒤 온라인에서 개인 인박스로 전송할 수 있습니다."
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            const form = new FormData(event.currentTarget);
            void run(async () => {
              if (!/^[0-9a-f-]{36}$/i.test(workspaceID))
                throw new Error(
                  "워크스페이스 UUID를 입력하거나 온라인 기기 설정 화면에서 이 페이지를 여세요.",
                );
              const id = crypto.randomUUID();
              await saveOfflineRecord({
                kind: "capture",
                id: "capture:" + id,
                workspace_id: workspaceID,
                title: String(form.get("title") || "임시 기록"),
                text: String(form.get("text") || ""),
                url: "",
                client_request_id: id,
                created_at: new Date().toISOString(),
              });
              setCapture(false);
              setMessage("임시 기록을 암호화해 보관했습니다.");
            });
          }}
        >
          <Field label="워크스페이스 ID">
            <input
              required
              value={workspaceID}
              onChange={(event) => setWorkspaceID(event.target.value)}
            />
          </Field>
          <Field label="기록 제목">
            <input name="title" required maxLength={300} />
          </Field>
          <Field label="기록 내용">
            <textarea name="text" required maxLength={900000} rows={6} />
          </Field>
          <Button disabled={busy}>기기에 임시 기록 저장</Button>
        </form>
      </Modal>
    </div>
  );
}
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <OfflineApp />
  </React.StrictMode>,
);
