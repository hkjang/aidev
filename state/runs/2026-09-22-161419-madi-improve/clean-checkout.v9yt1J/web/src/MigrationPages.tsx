import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowRight,
  CheckCircle2,
  FileUp,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { api, bytes, datetime } from "./api";
import { useApp } from "./context";
import { archiveFolder } from "./transfer-folder";
import "./transfer.css";
import ResumableMigration from "./migration/ResumableMigration";
import MigrationPolicy from "./migration/MigrationPolicy";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
type Row = Record<string, any>;
const formats = [
  ["markdown", "Markdown / ZIP"],
  ["obsidian", "Obsidian Vault"],
  ["notion", "Notion 내보내기"],
  ["html", "HTML 문서"],
  ["csv", "CSV → 데이터베이스"],
  ["json", "JSON 보관함 / 문서 목록"],
];
function stateLabel(row: Row) {
  if (row.job_status === "failed") return "작업 실패";
  if (row.status === "completed") return "가져오기 완료";
  if (row.status === "cancelled") return "취소";
  if (row.status === "ready") return "미리보기 준비";
  return row.status === "running" ? "가져오는 중" : "미리보기 처리 중";
}
export function MigrationPage() {
  const [params, setParams] = useSearchParams();
  const entry = params.get("source") || "file";
  const importID = params.get("import");
  const [legacyOpen, setLegacyOpen] = useState(!!importID);
  useEffect(() => {
    if (importID) setLegacyOpen(true);
  }, [importID]);
  return (
    <>
      <PageHeading
        eyebrow="PREPARE · REVIEW · COMMIT"
        title="가져오기"
        description="파일, 폴더 또는 다른 서비스의 지식을 한곳에서 가져옵니다."
        actions={
          <Link className="button migration-export-link" to="/app/export">
            내보내기
          </Link>
        }
      />
      <nav className="migration-entry-tabs" aria-label="가져오기 원본">
        <Button
          variant={entry === "file" ? "primary" : ""}
          onClick={() =>
            setParams((p) => {
              const n = new URLSearchParams(p);
              n.set("source", "file");
              return n;
            })
          }
        >
          파일에서 가져오기
        </Button>
        <Button
          variant={entry === "folder" ? "primary" : ""}
          onClick={() =>
            setParams((p) => {
              const n = new URLSearchParams(p);
              n.set("source", "folder");
              return n;
            })
          }
        >
          폴더에서 가져오기
        </Button>
        <Button
          variant={entry === "service" ? "primary" : ""}
          onClick={() => setParams({ source: "service" })}
        >
          다른 서비스에서 가져오기
        </Button>
      </nav>
      {entry === "service" ? (
        <section className="panel padded">
          <h2>다른 서비스의 지식 연결</h2>
          <p>
            관리자가 허용한 연결을 선택하고 대상 공간과 전송 범위를 확인하세요.
          </p>
          <div className="button-row">
            <Link className="button" to="/app/connectors">
              서비스 커넥터
            </Link>
            <Link className="button" to="/app/git-sync">
              Git 동기화
            </Link>
            <Button onClick={() => setParams({ source: "file" })}>
              Notion · Obsidian 내보낸 파일
            </Button>
          </div>
        </section>
      ) : (
        <>
          <ResumableMigration
            inputMode={entry === "folder" ? "folder" : "file"}
          />
          <details
            className="migration-legacy"
            open={legacyOpen}
            onToggle={(event) => setLegacyOpen(event.currentTarget.open)}
          >
            <summary>ZIP · 기존 JSON 형식 호환 가져오기</summary>
            <LegacyMigrationPage />
          </details>
          <MigrationPolicy />
        </>
      )}
    </>
  );
}
function LegacyMigrationPage() {
  const { workspace, notify, reload } = useApp();
  const [params, setParams] = useSearchParams();
  const [folderFiles, setFolderFiles] = useState<File[]>([]),
    [inputMode, setInputMode] = useState("file");
  const scope = workspace?.id || "",
    scopeRef = useRef(scope),
    operation = useRef(0),
    controller = useRef<AbortController | null>(null);
  scopeRef.current = scope;
  const alive = (op: number, start: string) =>
    operation.current === op && scopeRef.current === start;
  const [rows, setRows] = useState<Row[]>([]),
    [spaces, setSpaces] = useState<Row[]>([]),
    [space, setSpace] = useState(""),
    [format, setFormat] = useState("markdown"),
    [file, setFile] = useState<File | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [confirmation, setConfirmation] = useState("");
  const folderInput = useRef<HTMLInputElement>(null);
  const input = useRef<HTMLInputElement>(null),
    generation = useRef(0);
  const selected = rows.find((x) => x.id === params.get("import"));
  const writable = ["owner", "admin", "editor"].includes(workspace?.role || "");
  const load = useCallback(async () => {
    if (!workspace?.id || !writable) {
      setLoading(false);
      return;
    }
    const run = ++generation.current;
    try {
      const [r, s] = await Promise.all([
        api<Row[]>(`/migrations?workspace_id=${workspace.id}`),
        api<Row[]>(`/spaces?workspace_id=${workspace.id}`),
      ]);
      if (run === generation.current) {
        setRows(r);
        setSpaces(s.filter((x) => x.can_write));
        setError("");
      }
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, [workspace?.id, writable]);
  useEffect(() => {
    setLoading(true);
    setRows([]);
    setSpace("");
    operation.current++;
    controller.current?.abort();
    setFile(null);
    setFolderFiles([]);
    setConfirmation("");
    setBusy(false);
    setError("");
    void load();
    return () => {
      generation.current++;
      operation.current++;
      controller.current?.abort();
    };
  }, [load]);
  useEffect(() => {
    if (!rows.some((r) => ["pending", "running"].includes(r.job_status)))
      return;
    const timer = setInterval(() => void load(), 4000);
    return () => clearInterval(timer);
  }, [rows, load]);
  useEffect(() => setConfirmation(""), [selected?.id]);
  return (
    <>
      <PageHeading
        eyebrow="PREVIEW · VERIFY · IMPORT"
        title="가져오기 센터"
        description="파일을 먼저 검토하고, 확인한 데이터만 작업 큐에서 안전하게 가져옵니다."
        actions={
          <>
            <Link className="button" to="/app/export">
              내보내기 센터
            </Link>
            <Link className="button" to="/app/jobs">
              작업 이력
            </Link>
            <Button onClick={() => void load()}>
              <RefreshCw size={17} />
              새로고침
            </Button>
          </>
        }
      />
      <ErrorBox error={error} />
      {!writable ? (
        <Empty
          title="가져오기 권한이 필요합니다"
          text="워크스페이스 편집자 이상의 권한이 필요합니다."
        />
      ) : (
        <>
          <div className="notice subtle">
            <ShieldCheck size={22} />
            <span>
              미리보기는 문서를 변경하지 않습니다. 기존 동명 문서를 덮어쓰지
              않으며, 파일은 7일 보관 후 폐기합니다. 외부 스크립트나 웹 콘텐츠를
              실행하지 않습니다.
            </span>
          </div>
          <form
            className="panel padded"
            onSubmit={async (e) => {
              e.preventDefault();
              if ((!file && !folderFiles.length) || !workspace) return;
              const op = ++operation.current,
                start = scope;
              controller.current?.abort();
              const abort = new AbortController();
              controller.current = abort;
              setBusy(true);
              setError("");
              try {
                const form = new FormData();
                const selectedFile =
                  inputMode === "folder"
                    ? await archiveFolder(folderFiles, abort.signal)
                    : file;
                if (!alive(op, start) || !selectedFile) return;
                form.append("file", selectedFile);
                const stage = await api<Row>(
                  `/migrations?workspace_id=${workspace.id}&space_id=${space}&format=${format}`,
                  "POST",
                  form,
                );
                if (!alive(op, start)) return;
                await load();
                if (!alive(op, start)) return;
                setParams({ import: stage.id });
                setFile(null);
                setFolderFiles([]);
                if (folderInput.current) folderInput.current.value = "";
                if (input.current) input.current.value = "";
                notify(
                  "미리보기 작업을 등록했습니다. 아래 이력에서 결과를 확인하세요.",
                );
              } catch (e) {
                if (alive(op, start)) setError((e as Error).message);
              } finally {
                if (alive(op, start)) setBusy(false);
              }
            }}
          >
            <fieldset
              disabled={busy}
              style={{ border: 0, padding: 0, minWidth: 0 }}
            >
              <h2>
                <FileUp size={21} /> 가져올 파일
              </h2>
              <div className="form-grid">
                <Field label="원본 형식">
                  <select
                    value={format}
                    onChange={(e) => {
                      setFormat(e.target.value);
                      setFile(null);
                      setFolderFiles([]);
                      if (["csv", "json"].includes(e.target.value))
                        setInputMode("file");
                    }}
                  >
                    {formats.map(([id, name]) => (
                      <option value={id} key={id}>
                        {name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="대상 공간">
                  <select
                    value={space}
                    onChange={(e) => setSpace(e.target.value)}
                  >
                    <option value="">워크스페이스 기본 영역</option>
                    {spaces.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name}
                      </option>
                    ))}
                  </select>
                </Field>
              </div>
              {!["csv", "json"].includes(format) && (
                <Field label="파일 선택 방식">
                  <select
                    value={inputMode}
                    onChange={(e) => {
                      setInputMode(e.target.value);
                      setFile(null);
                      setFolderFiles([]);
                    }}
                  >
                    <option value="file">파일 또는 ZIP 선택</option>
                    <option value="folder">폴더 선택</option>
                  </select>
                </Field>
              )}
              {inputMode === "folder" ? (
                <Field
                  label="가져오기 폴더"
                  hint="원본 합계 48MB · 파일 5,000개 · 브라우저 별도 작업에서 ZIP 준비"
                >
                  <div className="transfer-file-picker">
                    <input
                      ref={folderInput}
                      className="sr-only"
                      type="file"
                      {...({ webkitdirectory: "", directory: "" } as Record<
                        string,
                        string
                      >)}
                      multiple
                      onChange={(e) =>
                        setFolderFiles(Array.from(e.target.files || []))
                      }
                    />
                    <Button
                      type="button"
                      onClick={() => folderInput.current?.click()}
                    >
                      폴더 선택
                    </Button>
                    <span>
                      {folderFiles.length
                        ? `선택한 파일 ${folderFiles.length}개`
                        : "선택한 폴더 없음"}
                    </span>
                  </div>
                </Field>
              ) : (
                <Field
                  label="가져오기 파일"
                  hint="최대 50MB · CSV 5,000행/100열 · ZIP 5,000개 항목/해제 100MB · 문서 트리 20단계"
                >
                  <div className="transfer-file-picker">
                    <input
                      ref={input}
                      className="sr-only"
                      type="file"
                      required
                      accept={
                        format === "html"
                          ? ".html,.htm,.zip"
                          : format === "csv"
                            ? ".csv"
                            : format === "json"
                              ? ".json"
                              : ".md,.markdown,.zip"
                      }
                      onChange={(e) => setFile(e.target.files?.[0] || null)}
                    />
                    <Button
                      type="button"
                      onClick={() => input.current?.click()}
                    >
                      파일 선택
                    </Button>
                    <span>{file?.name || "선택한 파일 없음"}</span>
                  </div>
                </Field>
              )}
              {format === "json" && (
                <p className="muted">
                  JSON 문서 목록은 title, markdown, tags, aliases 속성의 배열
                  또는 {"{documents: [...]}"} 객체를 사용합니다. 또는 내보내기
                  센터의 madi-json-vault 버전 1 파일을 사용하면 폴더·첨부·링크도
                  새 비공개 초안으로 복원합니다.
                </p>
              )}
              {format === "notion" && (
                <p className="muted">
                  Notion에서 내보낸 Markdown ZIP 또는 HTML ZIP을 선택하세요.
                  HTML은 안전한 Markdown으로 변환하며 외부 이미지를 다운로드하지
                  않습니다. 로컬 이미지와 문서 링크를 복원하며, CSV는 원본
                  첨부와 텍스트 속성 데이터베이스로 함께 가져옵니다(최대 20개).
                </p>
              )}
              <Button
                variant="primary"
                disabled={
                  busy || (inputMode === "folder" ? !folderFiles.length : !file)
                }
              >
                {busy ? "파일 준비 중…" : "미리보기 만들기"}
                <ArrowRight size={17} />
              </Button>
            </fieldset>
          </form>
          <section style={{ marginTop: 28 }}>
            <div className="section-heading">
              <h2>내 가져오기 이력</h2>
              <Badge>{rows.length}건</Badge>
            </div>
            {loading ? (
              <Loading />
            ) : rows.length ? (
              <div className="panel table-scroll migration-history-table">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>원본 파일</th>
                      <th>상태</th>
                      <th>생성 시각</th>
                      <th>작업</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.id}>
                        <td>
                          <strong>{row.filename}</strong>
                          <small className="muted" style={{ display: "block" }}>
                            {formats.find(([id]) => id === row.format)?.[1]} ·{" "}
                            {bytes(row.source_bytes)}
                          </small>
                        </td>
                        <td>
                          <Badge
                            tone={row.status === "completed" ? "green" : ""}
                          >
                            {stateLabel(row)}
                          </Badge>
                        </td>
                        <td>{datetime(row.created_at)}</td>
                        <td>
                          <Button onClick={() => setParams({ import: row.id })}>
                            미리보기 · 결과
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <Empty
                title="첫 가져오기를 준비하세요"
                text="Obsidian Vault, Notion 내보내기와 기존 문서 파일을 연결할 수 있습니다."
              />
            )}
          </section>
        </>
      )}
      <Modal
        open={!!selected}
        onOpenChange={(v) => {
          if (!v && !busy) setParams({});
        }}
        title="가져오기 미리보기와 결과"
        description={selected?.filename || ""}
        wide
      >
        {selected && (
          <>
            <div className="section-heading">
              <Badge tone={selected.status === "completed" ? "green" : ""}>
                {stateLabel(selected)}
              </Badge>
              <span className="muted">
                원본 만료: {datetime(selected.expires_at)}
              </span>
            </div>
            <ErrorBox
              error={selected.job_error || selected.report?.error || ""}
            />
            {selected.preview?.validated ? (
              <>
                <div className="migration-preview-stats">
                  {[
                    ["documents", "문서"],
                    ["folders", "폴더"],
                    ["attachments", "첨부파일"],
                    ["databases", "데이터베이스"],
                    ["rows", "데이터 행"],
                  ].map(([key, label]) => (
                    <article className="panel padded" key={key}>
                      <span className="muted">{label}</span>
                      <h2>{selected.preview[key] || 0}</h2>
                    </article>
                  ))}
                </div>
                {selected.preview.columns?.length > 0 && (
                  <p>열: {selected.preview.columns.join(" · ")}</p>
                )}
                {selected.preview.warnings?.map((text: string, i: number) => (
                  <p className="muted" key={i}>
                    {text}
                  </p>
                ))}
                {selected.preview.sample?.length > 0 && (
                  <div className="table-scroll">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>문서 제목 (최대 30개 미리보기)</th>
                          <th>원문 크기</th>
                          <th>동명 문서</th>
                        </tr>
                      </thead>
                      <tbody>
                        {selected.preview.sample.map((d: Row, i: number) => (
                          <tr key={i}>
                            <td>{d.title}</td>
                            <td>{bytes(d.bytes)}</td>
                            <td>{d.duplicates || 0}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </>
            ) : !selected.job_error && selected.status !== "cancelled" ? (
              <Loading />
            ) : null}
            {selected.status === "completed" ? (
              <div className="notice subtle">
                <CheckCircle2 size={22} />
                <span>
                  가져오기 완료: 문서 {selected.report.imported || 0}개 · 폴더{" "}
                  {selected.report.folders || 0}개 · 첨부{" "}
                  {selected.report.attachments || 0}개 · 데이터 행{" "}
                  {selected.report.rows || 0}개.{" "}
                  {selected.report.database_id && (
                    <Link to={`/app/databases/${selected.report.database_id}`}>
                      데이터베이스 열기
                    </Link>
                  )}
                </span>
              </div>
            ) : null}
            {selected.preview?.validated &&
              selected.source_bytes > 0 &&
              (["ready", "failed"].includes(selected.status) ||
                selected.job_status === "failed") && (
                <form
                  onSubmit={async (e) => {
                    e.preventDefault();
                    const op = ++operation.current,
                      start = scope;
                    setBusy(true);
                    setError("");
                    try {
                      await api(`/migrations/${selected.id}/run`, "POST", {
                        confirmation,
                      });
                      if (!alive(op, start)) return;
                      notify("가져오기 작업을 시작했습니다.");
                      await load();
                      if (alive(op, start)) await reload();
                    } catch (e) {
                      if (alive(op, start)) setError((e as Error).message);
                    } finally {
                      if (alive(op, start)) setBusy(false);
                    }
                  }}
                >
                  <Field
                    label="가져오기 확인"
                    hint="미리보기를 확인한 뒤 IMPORT를 입력하세요."
                  >
                    <input
                      value={confirmation}
                      onChange={(e) => setConfirmation(e.target.value)}
                      required
                      autoComplete="off"
                    />
                  </Field>
                  <Button
                    variant="primary"
                    disabled={busy || confirmation !== "IMPORT"}
                  >
                    확인한 데이터 가져오기
                  </Button>
                </form>
              )}
            {!["completed", "cancelled"].includes(selected.status) && (
              <Button
                style={{ marginTop: 16 }}
                disabled={busy}
                onClick={async () => {
                  if (
                    !confirm(
                      "이 가져오기 작업을 취소하고 보관된 원본 파일을 폐기하시겠습니까?",
                    )
                  )
                    return;
                  setBusy(true);
                  const op = ++operation.current,
                    start = scope;
                  try {
                    await api(`/migrations/${selected.id}`, "DELETE");
                    if (!alive(op, start)) return;
                    await load();
                    if (alive(op, start)) notify("작업을 취소했습니다.");
                  } catch (e) {
                    if (alive(op, start)) setError((e as Error).message);
                  } finally {
                    if (alive(op, start)) setBusy(false);
                  }
                }}
              >
                <Trash2 size={17} /> 작업 취소 · 원본 폐기
              </Button>
            )}
          </>
        )}
      </Modal>
    </>
  );
}
