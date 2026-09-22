import { useEffect, useState } from "react";
import {
  Download,
  HardDrive,
  Lock,
  Monitor,
  RefreshCw,
  ShieldCheck,
  Smartphone,
  Trash2,
} from "lucide-react";
import { useApp } from "../context";
import { api, ApiError, type Doc } from "../api";
import { Button, Empty, ErrorBox, Field, Modal, PageHeading } from "../ui";
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
} from "./vault";
import "./pwa.css";
export default function DevicesPage() {
  const { user, workspace, documents, notify } = useApp();
  const [meta, setMeta] = useState<{ userID: string } | null>(null);
  const [unlocked, setUnlocked] = useState(vaultUnlocked(user.id));
  const [password, setPassword] = useState("");
  const [records, setRecords] = useState<OfflineRecord[]>([]);
  const [documentID, setDocumentID] = useState("");
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [erase, setErase] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const refresh = async () => {
    if (!vaultSupported()) return;
    setMeta(await vaultMetadata());
    const open = vaultUnlocked(user.id);
    setUnlocked(open);
    setRecords(open ? await listOfflineRecords() : []);
  };
  useEffect(() => {
    if (vaultUnlocked() && !vaultUnlocked(user.id)) lockOfflineVault();
    void refresh().catch(setError);
    const change = () => {
      void refresh().catch(setError);
    };
    window.addEventListener("madi-vault-change", change);
    return () => window.removeEventListener("madi-vault-change", change);
  }, [user.id]);
  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError(undefined);
    try {
      await fn();
      await refresh();
    } catch (err) {
      setError(err);
    } finally {
      setBusy(false);
    }
  }
  async function sync(record: OfflineRecord) {
    if (!navigator.onLine) throw new Error("서버에 연결한 뒤 다시 시도하세요.");
    if (record.kind === "capture") {
      await api("/captures", "POST", {
        workspace_id: record.workspace_id,
        title: record.title,
        text: record.text,
        url: record.url,
        client_request_id: record.client_request_id,
      });
      await removeOfflineRecord(record.id);
      notify("임시 기록을 개인 인박스에 저장했습니다.");
      return;
    }
    const current = await api<Doc>("/documents/" + record.doc.id);
    if (record.edited) {
      if (!current.can_write)
        throw new Error(
          "현재 문서 수정 권한이 없습니다. 로컬 초안은 그대로 보관합니다.",
        );
      if (current.version !== record.doc.version)
        throw new Error(
          "서버 문서가 변경되었습니다. 오프라인 초안을 Markdown으로 내보낸 뒤 현재 문서와 비교해 주세요. 자동으로 덮어쓰지 않습니다.",
        );
      const saved = await api<Doc>("/documents/" + record.doc.id, "PUT", {
        title: record.doc.title,
        markdown: record.doc.markdown,
        version: record.doc.version,
        block_metadata: {},
      });
      await saveOfflineRecord({
        ...record,
        doc: saved,
        edited: false,
        saved_at: new Date().toISOString(),
      });
      notify("오프라인 초안을 서버에 반영했습니다.");
    } else {
      await saveOfflineRecord({
        ...record,
        doc: current,
        saved_at: new Date().toISOString(),
      });
      notify("현재 권한을 확인하고 기기 사본을 갱신했습니다.");
    }
  }
  return (
    <>
      <PageHeading
        eyebrow="PERSONAL · DEVICES"
        title="기기와 오프라인"
        description="이 기기에서만 여는 암호화 보관함. 원하는 문서만 직접 선택해 보관하세요."
      />
      <ErrorBox error={error} />
      <div className="device-grid">
        <section className="device-card">
          <ShieldCheck size={30} />
          <h2>내 기기 보관함</h2>
          <p>
            문서 제목과 본문, 임시 기록을 AES-256-GCM으로 암호화합니다. 암호는
            서버에 보내지 않으며, 재접속·로그아웃·5분 미사용 시 보관함을
            잠급니다.
          </p>
          <div className="offline-note">
            기기 사본은 저장 시점의 문서입니다. 오프라인에서는 이후 권한 회수를
            확인할 수 없으므로, 기밀 문서는 조직의 반출 정책을 먼저 확인하세요.
            첨부파일과 서버 응답은 자동 저장하지 않습니다.
          </div>
          {!vaultSupported() ? (
            <p role="status">
              HTTPS 또는 localhost에서 오프라인 암호화와 앱 설치를 사용할 수
              있습니다. 현재 주소에서도 일반 서비스 기능은 정상 사용 가능합니다.
            </p>
          ) : unlocked ? (
            <div className="offline-actions">
              <Button variant="secondary" onClick={() => lockOfflineVault()}>
                <Lock size={17} />
                보관함 잠그기
              </Button>
              <a className="button secondary" href="/offline.html">
                오프라인 화면 열기
              </a>
            </div>
          ) : (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void run(async () => {
                  if (meta) await unlockVault(password, user.id);
                  else await createVault(user.id, password);
                  setPassword("");
                });
              }}
            >
              <Field
                label="기기 보관함 암호"
                hint="12자 이상 · 로그인 암호와 다르게 설정 · 분실 시 복구할 수 없습니다."
              >
                <input
                  type="password"
                  minLength={12}
                  autoComplete="off"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </Field>
              <Button disabled={busy || (!!meta && meta.userID !== user.id)}>
                {meta ? "보관함 잠금 해제" : "암호화 보관함 만들기"}
              </Button>
              {meta && meta.userID !== user.id && (
                <p>
                  다른 사용자 보관함이 있습니다. 해당 계정으로 로그인하거나 기기
                  사본을 삭제하세요.
                </p>
              )}
            </form>
          )}
          {meta && (
            <Button variant="ghost" onClick={() => setErase(true)}>
              <Trash2 size={16} />이 기기의 보관함 삭제
            </Button>
          )}
        </section>
        <section className="device-card">
          <Smartphone size={30} />
          <h2>앱으로 빠르게 연결</h2>
          <p>
            Chrome·Edge의 주소창 설치 버튼 또는 모바일 브라우저의 ‘홈 화면에
            추가’를 이용하세요. 외부 CDN이나 인터넷 연결 없이 사내 서버에서 설치
            파일을 제공합니다.
          </p>
          <div className="offline-note">
            설치형 앱도 사내 서버 연결이 필요합니다. 연결이 끊기면 미리 보관한
            문서를 읽고 초안을 작성할 수 있습니다. 전송은 온라인에서 현재 사용자
            권한을 다시 확인한 뒤 직접 실행합니다.
          </div>
          <div className="offline-actions">
            <a
              className="button secondary"
              href="/manifest.webmanifest"
              download="madi.webmanifest"
            >
              <Download size={16} />
              설치 정보 보기
            </a>
            <a
              className="button secondary"
              href="https://hkjang.github.io/madi/guide.html"
              target="_blank"
              rel="noreferrer"
            >
              <Monitor size={16} />
              기기 사용 가이드
            </a>
          </div>
          <p className="muted">
            폐쇄망에서는 저장소의 sdk/desktop, sdk/clipper 가이드를 함께
            배포하세요. 외부 링크는 사용자가 직접 열 때만 연결됩니다.
          </p>
        </section>
      </div>
      <section className="device-card" style={{ marginTop: 24 }}>
        <h2>
          <HardDrive size={23} /> 명시적으로 저장한 문서와 임시 기록
        </h2>
        {unlocked ? (
          <>
            <div className="offline-actions">
              <Field label="오프라인에 보관할 문서">
                <select
                  value={documentID}
                  onChange={(event) => setDocumentID(event.target.value)}
                >
                  <option value="">문서를 선택하세요</option>
                  {documents
                    .filter((doc) => !doc.deleted_at)
                    .map((doc) => (
                      <option value={doc.id} key={doc.id}>
                        {doc.title}
                      </option>
                    ))}
                </select>
              </Field>
              <Button
                disabled={busy || !documentID}
                onClick={() =>
                  void run(async () => {
                    const doc = await api<Doc>("/documents/" + documentID);
                    if (
                      records.some(
                        (record) =>
                          record.kind === "document" &&
                          record.doc.id === doc.id &&
                          record.edited,
                      )
                    )
                      throw new Error(
                        "이 문서에 오프라인 초안이 있습니다. 먼저 전송하거나 내보낸 후 기기 사본을 삭제하세요.",
                      );
                    await saveOfflineRecord({
                      kind: "document",
                      id: "document:" + doc.id,
                      doc,
                      saved_at: new Date().toISOString(),
                      edited: false,
                    });
                    notify("선택한 문서를 이 기기에 암호화해 보관했습니다.");
                  })
                }
              >
                문서 보관
              </Button>
            </div>
            {records.length ? (
              records.map((record) => (
                <div className="offline-record" key={record.id}>
                  <div>
                    <strong>
                      {record.kind === "document"
                        ? record.doc.title
                        : record.title || "제목 없는 임시 기록"}
                    </strong>
                    <small>
                      {record.kind === "capture"
                        ? "인박스 전송 대기"
                        : record.edited
                          ? "오프라인 수정 · 서버에 아직 반영되지 않음"
                          : "읽기용 기기 사본"}
                    </small>
                  </div>
                  <Button
                    variant="secondary"
                    disabled={busy}
                    onClick={() => void run(() => sync(record))}
                  >
                    <RefreshCw size={16} />
                    {record.kind === "capture" || record.edited
                      ? "서버에 전송"
                      : "권한 확인·갱신"}
                  </Button>
                  <Button
                    variant="ghost"
                    aria-label={
                      (record.kind === "document"
                        ? record.doc.title
                        : record.title) + " 기기 사본 삭제"
                    }
                    disabled={busy}
                    onClick={() =>
                      void run(() => removeOfflineRecord(record.id))
                    }
                  >
                    <Trash2 size={16} />
                  </Button>
                </div>
              ))
            ) : (
              <Empty
                title="저장한 기기 사본이 없습니다"
                text="보관할 문서를 선택하세요. 원본의 첨부파일은 복사하지 않습니다."
              />
            )}
            <div className="offline-actions">
              <a
                className="button secondary"
                href={
                  "/offline.html" +
                  (workspace ? "?workspace_id=" + workspace.id : "")
                }
              >
                오프라인 읽기·임시 기록 작성
              </a>
            </div>
          </>
        ) : (
          <Empty
            title="보관함이 잠겨 있습니다"
            text="기기 보관함을 만들거나 암호로 잠금 해제하세요."
          />
        )}
      </section>
      <Modal
        open={erase}
        onOpenChange={setErase}
        title="이 기기의 보관함 삭제"
        description="기기에 저장한 모든 문서 사본과 아직 전송하지 않은 임시 기록·초안이 영구 삭제됩니다. 서버 원본은 변경하지 않습니다."
      >
        <Field label="확인 문구" hint="기기 보관함 삭제를 입력하세요.">
          <input
            value={confirmation}
            onChange={(event) => setConfirmation(event.target.value)}
          />
        </Field>
        <div className="offline-actions">
          <Button variant="secondary" onClick={() => setErase(false)}>
            취소
          </Button>
          <Button
            variant="danger"
            disabled={busy || confirmation !== "기기 보관함 삭제"}
            onClick={() =>
              void run(async () => {
                await clearVault();
                setErase(false);
                setConfirmation("");
                notify(
                  "이 기기의 암호화 보관함을 삭제했습니다. 복구할 수 없습니다.",
                );
              })
            }
          >
            기기 사본 영구 삭제
          </Button>
        </div>
      </Modal>
    </>
  );
}
