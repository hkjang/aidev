import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams, Link } from "react-router-dom";
import { api, ApiError, bytes, datetime } from "../api";
import { useApp } from "../context";
import { Badge, Button, Empty, Field, Loading, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import "./resumable.css";
import { sha256 } from "@noble/hashes/sha2.js";

type Row = Record<string, any>;
type Source = {
  file: File;
  path: string;
  source_id: string;
  parent_source_id?: string;
  kind: string;
  metadata?: Row;
};
const names: Record<string, string> = {
  uploading: "파일 받는 중",
  preparing: "준비·검증 중",
  ready: "검토 대기",
  committing: "원자적 확정 중",
  completed: "가져오기 완료",
  failed: "검토 필요",
  cancelled: "취소됨",
  new: "신규",
  changed: "변경",
  unchanged: "동일",
  conflict: "충돌",
  exact: "원문 보존",
  links_rewritten: "링크 복원",
  converted: "변환됨",
  policy_masked: "정보보호 적용",
  binary_preserved: "바이너리 보존",
  type_review_required: "자료형 확인 필요",
};
const types = [
  ["text", "텍스트"],
  ["number", "숫자"],
  ["checkbox", "체크박스"],
  ["date", "날짜"],
  ["select", "선택"],
  ["url", "URL"],
  ["email", "이메일"],
  ["phone", "전화번호"],
];
const hash = async (file: Blob) => {
  const h = sha256.create();
  for (let at = 0; at < file.size; at += 1048576)
    h.update(new Uint8Array(await file.slice(at, at + 1048576).arrayBuffer()));
  return Array.from(h.digest())
    .map((v) => v.toString(16).padStart(2, "0"))
    .join("");
};
function safePath(v: string) {
  return (
    !!v &&
    v.length <= 512 &&
    !v.startsWith("/") &&
    !v.includes("\\") &&
    !v.includes("\0") &&
    v.split("/").every((p) => p && p !== "." && p !== ".." && !p.includes(":"))
  );
}

async function sourceFiles(files: File[]): Promise<Source[]> {
  if (
    files.length === 1 &&
    files[0].name.toLowerCase().endsWith(".json") &&
    !files[0].webkitRelativePath
  ) {
    if (files[0].size > 50 * 1024 * 1024)
      throw new Error(
        "JSON 컨테이너는 50MB 이하로 나누거나 내보낸 폴더를 선택하세요.",
      );
    const data = JSON.parse(await files[0].text());
    if (data?.format === "madi-json-vault") {
      if (
        data.version !== 1 ||
        data.manifest?.format !== "madi-vault" ||
        data.manifest.version !== 1 ||
        !Array.isArray(data.files) ||
        data.files.length > 100000
      )
        throw new Error("지원하는 JSON 보관함은 madi-json-vault 버전 1입니다.");
      let total = 0;
      const expanded: File[] = [];
      for (const item of data.files) {
        if (!safePath(item.path) || typeof item.data_base64 !== "string")
          throw new Error("JSON 보관함 파일 경로·데이터를 확인하세요.");
        const decoded = Uint8Array.from(atob(item.data_base64), (c) =>
          c.charCodeAt(0),
        );
        total += decoded.length;
        if (total > 100 * 1024 * 1024 || decoded.length > 50 * 1024 * 1024)
          throw new Error(
            "JSON 펼친 합계는 100MB 이하입니다. 더 큰 자료는 원본 폴더를 선택하세요.",
          );
        expanded.push(new File([decoded], item.path));
      }
      expanded.push(
        new File([JSON.stringify(data.manifest)], "madi-manifest.json", {
          type: "application/json",
        }),
      );
      return sourceFiles(expanded);
    }
    const documents = Array.isArray(data)
      ? data
      : Array.isArray(data?.documents)
        ? data.documents
        : null;
    if (documents) {
      if (documents.length > 100000)
        throw new Error("문서 항목 수 한도를 초과했습니다.");
      const seen = new Set<string>();
      return documents.map((d: Row, i: number) => {
        if (
          !d ||
          typeof d.title !== "string" ||
          typeof d.markdown !== "string" ||
          Object.keys(d).some(
            (k) =>
              !["title", "markdown", "tags", "aliases", "source_id"].includes(
                k,
              ),
          )
        )
          throw new Error(
            "JSON 문서는 title/markdown/tags/aliases/source_id만 사용할 수 있습니다.",
          );
        const id =
          typeof d.source_id === "string"
            ? d.source_id
            : `${files[0].name}#${i + 1}`;
        if (seen.has(id)) throw new Error("JSON 원본 ID가 중복되었습니다.");
        seen.add(id);
        const file = new File(
          [
            JSON.stringify({
              format: "madi-migration-document",
              version: 1,
              title: d.title,
              markdown: d.markdown,
              tags: d.tags || [],
              aliases: d.aliases || [],
            }),
          ],
          `${i + 1}.json`,
          { type: "application/json" },
        );
        if (file.size > 4 * 1024 * 1024)
          throw new Error("JSON 개별 문서는 4MB 이하입니다.");
        return {
          file,
          path: `${i + 1}.json`,
          source_id: id,
          kind: "document",
          metadata: {},
        };
      });
    }
  }
  const folder = files.some((f) => f.webkitRelativePath),
    root = folder ? files[0].webkitRelativePath.split("/")[0] + "/" : "";
  const paths = files
    .map((file) => ({
      file,
      path: folder ? file.webkitRelativePath.slice(root.length) : file.name,
    }))
    .filter(
      (v) =>
        ![".git", ".obsidian", "__MACOSX"].includes(v.path.split("/")[0]) &&
        !v.path.endsWith("/.DS_Store"),
    );
  let manifest: Row | undefined;
  const mf = paths.find((v) => v.path === "madi-manifest.json");
  if (mf) {
    if (mf.file.size > 4 * 1024 * 1024)
      throw new Error("보관함 manifest는 4MB 이하입니다.");
    manifest = JSON.parse(await mf.file.text());
    if (manifest?.format !== "madi-vault" || manifest?.version !== 1)
      throw new Error("지원하는 manifest는 madi-vault 버전 1입니다.");
  }
  const meta = new Map<string, Row>();
  for (const d of [
    ...(manifest?.documents || []),
    ...(manifest?.attachments || []),
  ]) {
    if (
      !d ||
      typeof d.file !== "string" ||
      !safePath(d.file) ||
      meta.has(d.file)
    )
      throw new Error("manifest 경로가 중복되거나 잘못되었습니다.");
    meta.set(d.file, d);
  }
  const seen = new Set<string>(),
    ids = new Set<string>();
  return paths
    .filter((v) => v.path !== "madi-manifest.json")
    .map(({ file, path }) => {
      if (!safePath(path) || seen.has(path.toLocaleLowerCase()))
        throw new Error("안전하지 않거나 중복된 파일 경로입니다.");
      seen.add(path.toLocaleLowerCase());
      const ext = path.split(".").pop()?.toLowerCase() || "",
        kind = ["md", "markdown", "html", "htm", "json"].includes(ext)
          ? "document"
          : ext === "csv"
            ? "csv"
            : "attachment";
      if (ext === "zip")
        throw new Error(
          "재개 이관은 압축을 푼 폴더를 선택하세요. ZIP은 아래 호환 파일 가져오기를 사용할 수 있습니다.",
        );
      if (file.size > (kind === "document" ? 4 : 50) * 1024 * 1024)
        throw new Error(
          `${path}: 문서 4MB · 개별 CSV/첨부 50MB 제한을 확인하세요.`,
        );
      const m = meta.get(path),
        source_id = typeof m?.id === "string" ? m.id : path;
      if (ids.has(source_id)) throw new Error("원본 ID가 중복되었습니다.");
      ids.add(source_id);
      const metadata: Row = {};
      for (const key of ["title", "tags", "aliases", "icon"])
        if (m && key in m) metadata[key] = m[key];
      return {
        file,
        path,
        source_id,
        parent_source_id: typeof m?.parent_id === "string" ? m.parent_id : "",
        kind,
        metadata,
      };
    });
}

export default function ResumableMigration({
  inputMode,
}: {
  inputMode: "file" | "folder";
}) {
  const { workspace, user, notify, reload } = useApp(),
    [params, setParams] = useSearchParams();
  const scope = workspace?.id || "";
  const [runs, setRuns] = useState<Row[]>([]),
    [run, setRun] = useState<Row | null>(null),
    [items, setItems] = useState<Row[]>([]),
    [more, setMore] = useState(false),
    [offset, setOffset] = useState(0),
    [spaces, setSpaces] = useState<Row[]>([]);
  const [files, setFiles] = useState<File[]>([]),
    [sourceKey, setSourceKey] = useState(""),
    [label, setLabel] = useState(""),
    [space, setSpace] = useState(""),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [uploading, setUploading] = useState(false),
    [loading, setLoading] = useState(true),
    [progress, setProgress] = useState(""),
    [confirm, setConfirm] = useState(""),
    [review, setReview] = useState<Row | null>(null),
    [columns, setColumns] = useState<Row[]>([]),
    [typesConfirmed, setTypesConfirmed] = useState(false),
    [showConfirm, setShowConfirm] = useState(false);
  const input = useRef<HTMLInputElement>(null),
    generation = useRef(0),
    operation = useRef(0),
    scopeRef = useRef(scope),
    abort = useRef<AbortController | null>(null),
    uploadSession = useRef(""),
    offsetRef = useRef(offset),
    selectedRef = useRef(params.get("session") || "");
  scopeRef.current = scope;
  offsetRef.current = offset;
  selectedRef.current = params.get("session") || "";
  const selected = params.get("session") || "",
    writable =
      user?.role !== "viewer" &&
      ["owner", "admin", "editor"].includes(workspace?.role || "");
  const alive = (op: number, start: string) =>
    operation.current === op && scopeRef.current === start;
  const load = useCallback(async () => {
    if (!scope || !writable) {
      setLoading(false);
      return;
    }
    const gen = ++generation.current;
    const wanted = selectedRef.current;
    try {
      const [list, spaces, current] = await Promise.all([
        api<Row[]>(`/migrations/sessions?workspace_id=${scope}`),
        api<Row[]>(`/spaces?workspace_id=${scope}`),
        wanted
          ? api<Row>(`/migrations/sessions/${wanted}`)
          : Promise.resolve(null),
      ]);
      let page: Row = { items: [], has_more: false };
      if (current)
        page = await api<Row>(
          `/migrations/sessions/${wanted}/items?offset=${offsetRef.current}`,
        );
      if (
        gen !== generation.current ||
        wanted !== selectedRef.current ||
        scope !== scopeRef.current
      )
        return;
      if (current && current.workspace_id !== scope)
        throw new ApiError("현재 워크스페이스의 이관을 선택하세요.", 404);
      setRuns(list);
      setSpaces(spaces.filter((s) => s.can_write));
      setRun(current);
      setItems(page.items);
      setMore(page.has_more);
    } catch (e) {
      if (gen === generation.current) setError(e);
    } finally {
      if (gen === generation.current) setLoading(false);
    }
  }, [scope, selected, offset, writable]);
  useEffect(() => {
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    operation.current++;
    abort.current?.abort();
    setFiles([]);
    setRun(null);
    setItems([]);
    setReview(null);
    setColumns([]);
    setBusy(false);
    setUploading(false);
    setConfirm("");
    setError(null);
    setProgress("");
    setSourceKey("");
    setLabel("");
    setSpace("");
    return () => {
      operation.current++;
      abort.current?.abort();
    };
  }, [scope]);
  useEffect(() => {
    if (!selected || selected !== uploadSession.current) {
      operation.current++;
      abort.current?.abort();
      setBusy(false);
    }
    setReview(null);
    setColumns([]);
    setConfirm("");
    setShowConfirm(false);
  }, [selected]);
  useEffect(() => {
    if (!run || !["preparing", "committing"].includes(run.status)) return;
    const t = setInterval(() => void load(), 2000);
    return () => clearInterval(t);
  }, [run?.status, load]);
  const select = (id: string) => {
    selectedRef.current = id;
    offsetRef.current = 0;
    setParams((p) => {
      const next = new URLSearchParams(p);
      id ? next.set("session", id) : next.delete("session");
      next.delete("import");
      return next;
    });
    setOffset(0);
  };
  const act = async (work: (op: number, start: string) => Promise<void>) => {
    const op = ++operation.current,
      start = scope;
    setBusy(true);
    setError(null);
    try {
      await work(op, start);
    } catch (e) {
      if (alive(op, start)) setError(e);
    } finally {
      if (alive(op, start)) {
        uploadSession.current = "";
        setBusy(false);
        setUploading(false);
      }
    }
  };
  const upload = () =>
    void act(async (op, start) => {
      setProgress("");
      setUploading(true);
      const controller = new AbortController();
      abort.current = controller;
      const list = await sourceFiles(files);
      if (!alive(op, start)) return;
      if (!list.length) throw new Error("가져올 파일을 선택하세요.");
      let id = run?.status === "uploading" ? run.id : "";
      if (!id) {
        const v = await api<Row>("/migrations/sessions", "POST", {
          workspace_id: scope,
          space_id: space,
          source_key: sourceKey,
          label: label || sourceKey,
          format: inputMode === "folder" ? "obsidian" : "markdown",
        });
        if (!alive(op, start)) return;
        id = v.id;
        selectedRef.current = id;
        uploadSession.current = id;
        select(id);
      }
      for (let i = 0; i < list.length; i++) {
        if (!alive(op, start) || controller.signal.aborted) return;
        const f = list[i];
        setProgress(`${i + 1}/${list.length} · ${f.path} 검증`);
        const sha256 = await hash(f.file);
        if (!alive(op, start)) return;
        const registration = await api<Row>(
          `/migrations/sessions/${id}/items`,
          "POST",
          {
            items: [
              {
                source_id: f.source_id,
                path: f.path,
                kind: f.kind,
                bytes: f.file.size,
                sha256,
                parent_source_id: f.parent_source_id,
                metadata: f.metadata || {},
              },
            ],
          },
        );
        if (!alive(op, start)) return;
        const item = registration.items[0];
        const checkpoints = await api<Row[]>(
          `/migrations/sessions/${id}/items/${item.id}/chunks`,
        );
        const have = new Map(checkpoints.map((c) => [c.ordinal, c.checksum]));
        for (let byte = 0; byte < f.file.size; byte += 1048576) {
          if (!alive(op, start)) return;
          const chunk = f.file.slice(byte, byte + 1048576),
            ordinal = Math.floor(byte / 1048576),
            checksum = await hash(chunk);
          if (have.get(ordinal) === checksum) continue;
          const response = await fetch(
            `/api/v1/migrations/sessions/${id}/items/${item.id}/chunks/${ordinal}`,
            {
              method: "PUT",
              credentials: "same-origin",
              headers: {
                "X-Madi-Request": "1",
                "Content-Type": "application/octet-stream",
              },
              body: chunk,
              signal: controller.signal,
            },
          );
          if (!response.ok) {
            const body = await response
              .json()
              .catch(() => ({ error: "청크 전송에 실패했습니다" }));
            throw new ApiError(body.error, response.status);
          }
          if (alive(op, start))
            setProgress(
              `${i + 1}/${list.length} · ${f.path} · ${bytes(Math.min(byte + 1048576, f.file.size))}/${bytes(f.file.size)}`,
            );
        }
      }
      if (!alive(op, start)) return;
      select(id);
      setFiles([]);
      if (input.current) input.current.value = "";
      setProgress("업로드가 완료되었습니다. 준비·검증을 실행하세요.");
      notify("암호화 원본을 받았습니다. 아직 문서는 생성하지 않았습니다.");
      await load();
    });
  const openItem = (item: Row) =>
    void act(async (op, start) => {
      const detail = await api<Row>(
        `/migrations/sessions/${selected}/items/${item.id}`,
      );
      if (!alive(op, start) || selectedRef.current !== selected) return;
      setReview(detail);
      setColumns(
        (item.types_confirmed ? item.csv_types : detail.inferred_columns) || [],
      );
      setTypesConfirmed(false);
    });
  if (!writable)
    return (
      <Empty
        title="가져오기 권한이 필요합니다"
        text="워크스페이스 편집자 이상의 권한으로 사용할 수 있습니다."
      />
    );
  return (
    <section className="resume-migration">
      <div className="notice subtle">
        1MB 암호화 청크 → 항목별 준비 → 원문·변환 비교 → 명시적 확정. 새 문서는
        비공개 초안이며, 모든 항목이 성공해야 한 번에 반영됩니다.
      </div>
      <RecoveryNotice
        error={error}
        busy={busy}
        onReview={() => void load()}
        onRetry={() => void load()}
        onReauthenticate={() => location.assign("/login")}
      />
      <div className="resume-columns">
        <aside className="panel padded">
          <div className="section-heading">
            <h2>재개 가능한 이관</h2>
            <Button disabled={busy} onClick={() => select("")}>
              새 이관
            </Button>
          </div>
          {loading && !runs.length ? (
            <Loading />
          ) : (
            runs.map((v) => (
              <Button
                className={`resume-run ${v.id === selected ? "selected" : ""}`}
                key={v.id}
                disabled={busy}
                onClick={() => select(v.id)}
              >
                <strong>{v.label}</strong>
                <small>
                  {names[v.status]} · {v.item_count}개
                </small>
              </Button>
            ))
          )}
          {!runs.length && !loading && (
            <p className="muted">준비한 이관이 없습니다.</p>
          )}
        </aside>
        <div>
          {(!run || run.status === "uploading") && (
            <form
              className="panel padded"
              onSubmit={(e) => {
                e.preventDefault();
                upload();
              }}
            >
              <fieldset disabled={busy} className="resume-fieldset">
                <h2>{run ? "중단 지점부터 이어받기" : "원본 선택"}</h2>
                <p className="muted">
                  {run
                    ? "같은 원본 파일 또는 폴더를 다시 선택하세요. 서버와 해시가 일치하는 청크는 보내지 않습니다."
                    : "원본 식별자를 유지하면 다음 이관에서 신규·변경·동일 항목을 구분합니다. 기존 사용자 수정은 덮어쓰지 않습니다."}
                </p>
                <div className="form-grid">
                  <Field
                    label="원본 식별자"
                    hint="예: 팀위키-2026 · 같은 원본을 다시 가져올 때 유지"
                  >
                    <input
                      required
                      value={run?.source_key || sourceKey}
                      readOnly={!!run}
                      onChange={(e) => setSourceKey(e.target.value)}
                      maxLength={512}
                    />
                  </Field>
                  <Field label="이관 이름">
                    <input
                      value={run?.label || label}
                      readOnly={!!run}
                      onChange={(e) => setLabel(e.target.value)}
                      maxLength={200}
                      placeholder="예: 운영팀 위키 이관"
                    />
                  </Field>
                </div>
                <Field label="대상 공간">
                  <select
                    value={run?.space_id || space}
                    disabled={!!run}
                    onChange={(e) => setSpace(e.target.value)}
                  >
                    <option value="">기본 영역</option>
                    {spaces.map((s) => (
                      <option key={s.id} value={s.id}>
                        {s.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field
                  label={
                    inputMode === "folder" ? "재개 이관 폴더" : "재개 이관 파일"
                  }
                  hint="문서 4MB · 개별 첨부/CSV 50MB · 기본 세션 합계 1GB/50,000항목 (관리자 조정)"
                >
                  <div className="transfer-file-picker">
                    <input
                      ref={input}
                      className="sr-only"
                      type="file"
                      multiple
                      {...(inputMode === "folder"
                        ? { webkitdirectory: "", directory: "" }
                        : {})}
                      onChange={(e) => {
                        const list = Array.from(e.target.files || []);
                        setFiles(list);
                        if (!sourceKey && list.length) {
                          const name =
                            list[0].webkitRelativePath.split("/")[0] ||
                            list[0].name;
                          setSourceKey(name);
                          setLabel(name);
                        }
                      }}
                    />
                    <Button
                      type="button"
                      onClick={() => input.current?.click()}
                    >
                      {inputMode === "folder" ? "폴더 선택" : "파일 선택"}
                    </Button>
                    <span>
                      {files.length
                        ? `${files.length}개 · ${bytes(files.reduce((n, f) => n + f.size, 0))}`
                        : "선택한 파일 없음"}
                    </span>
                  </div>
                </Field>
                <Button
                  variant="primary"
                  disabled={!files.length || (!sourceKey && !run) || busy}
                >
                  {busy
                    ? "청크 전송 중…"
                    : run
                      ? "체크포인트에서 재개"
                      : "암호화 원본 업로드"}
                </Button>
              </fieldset>
            </form>
          )}
          {progress && <p role="status">{progress}</p>}
          {uploading && (
            <Button
              onClick={() => {
                operation.current++;
                abort.current?.abort();
                uploadSession.current = "";
                setUploading(false);
                setBusy(false);
                setProgress(
                  "업로드를 중단했습니다. 같은 파일을 선택하면 확인된 청크부터 이어받습니다.",
                );
                void load();
              }}
            >
              업로드 중단 · 체크포인트 유지
            </Button>
          )}
          {run && (
            <section className="panel padded">
              <div className="section-heading">
                <div>
                  <h2>{run.label}</h2>
                  <p className="muted">
                    {run.source_key} · 비교 데이터 보관:{" "}
                    {datetime(run.expires_at)}
                  </p>
                </div>
                <Badge>{names[run.status]}</Badge>
              </div>
              {run.error && (
                <RecoveryNotice
                  error={run.error}
                  onReview={() => void load()}
                />
              )}
              <div className="resume-stats">
                {[
                  ["uploaded_bytes", "받은 원본"],
                  ["item_count", "항목"],
                  ["prepared_count", "준비 완료"],
                ].map(([key, label]) => (
                  <div key={key}>
                    <span>{label}</span>
                    <strong>
                      {key === "uploaded_bytes" ? bytes(run[key]) : run[key]}
                    </strong>
                  </div>
                ))}
              </div>
              <div className="button-row">
                {["uploading", "failed"].includes(run.status) && (
                  <Button
                    disabled={
                      busy ||
                      !run.item_count ||
                      run.uploaded_bytes !== run.declared_bytes
                    }
                    onClick={() =>
                      void act(async (op, start) => {
                        await api(
                          `/migrations/sessions/${selected}/prepare`,
                          "POST",
                          { revision: run.revision },
                        );
                        if (alive(op, start)) await load();
                      })
                    }
                  >
                    준비·검증 실행
                  </Button>
                )}
                <Button disabled={busy} onClick={() => void load()}>
                  체크포인트 새로고침
                </Button>
                {!["completed", "cancelled"].includes(run.status) && (
                  <Button
                    disabled={busy}
                    onClick={() => {
                      if (
                        window.confirm(
                          "이관 원본과 준비 데이터를 폐기할까요? 아직 확정되지 않은 문서는 생성되지 않습니다.",
                        )
                      )
                        void act(async (op, start) => {
                          await api(
                            `/migrations/sessions/${selected}`,
                            "DELETE",
                          );
                          if (alive(op, start)) await load();
                        });
                    }}
                  >
                    이관 취소
                  </Button>
                )}
              </div>
              {run.status === "committing" && (
                <p className="notice subtle">
                  {run.report?.phase === "publication_lock"
                    ? "권한 잠금과 최종 전체 반영을 진행합니다. 이 단계에서는 관리자 권한 변경이 잠시 기다릴 수 있습니다."
                    : "검토한 첨부를 비공개 저장소에 준비하고 있습니다."}{" "}
                  잠금 대기는 5초, 개별 DB 명령은 10분으로 제한하며 실패 시 일부
                  자료를 게시하지 않습니다.
                </p>
              )}
              {run.report?.storage_bytes !== undefined && (
                <p className="muted">
                  예상 첨부 저장 {bytes(run.report.storage_bytes)} · 독립 파일{" "}
                  {run.report.storage_objects}개
                </p>
              )}
              {!!items.length && (
                <div className="table-scroll resume-items">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>원본 경로</th>
                        <th>변경 판정</th>
                        <th>호환성·체크포인트</th>
                        <th>검토</th>
                      </tr>
                    </thead>
                    <tbody>
                      {items.map((item) => (
                        <tr key={item.id}>
                          <td>
                            <strong>{item.file_path}</strong>
                            <small>
                              {bytes(item.source_bytes)} ·{" "}
                              {item.kind === "csv"
                                ? "데이터베이스"
                                : item.kind === "attachment"
                                  ? "첨부"
                                  : "문서"}
                            </small>
                          </td>
                          <td>
                            <Badge
                              tone={
                                item.disposition === "conflict" ? "red" : ""
                              }
                            >
                              {item.metadata.decision === "skip"
                                ? "제외"
                                : names[item.disposition]}
                            </Badge>
                          </td>
                          <td>
                            {names[item.compatibility.status] || item.status}
                            <small>
                              청크 {bytes(item.received_bytes)} · 단계{" "}
                              {item.checkpoint}/2
                            </small>
                          </td>
                          <td>
                            <Button
                              disabled={busy || item.status !== "prepared"}
                              onClick={() => openItem(item)}
                            >
                              원문 · 변환 검토
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <div className="button-row">
                <Button
                  disabled={busy || offset === 0}
                  onClick={() => setOffset((v) => Math.max(0, v - 100))}
                >
                  이전 100개
                </Button>
                <Button
                  disabled={busy || !more}
                  onClick={() => setOffset((v) => v + 100)}
                >
                  다음 100개
                </Button>
              </div>
              {run.status === "ready" && (
                <>
                  <div className="resume-stats">
                    {[
                      ["new", "신규"],
                      ["changed", "변경"],
                      ["unchanged", "동일"],
                      ["conflicts", "충돌"],
                      ["csv_unconfirmed", "자료형 미확인"],
                      ["broken_dependencies", "제외한 자료의 참조"],
                    ].map(([key, name]) => (
                      <div key={key}>
                        <span>{name}</span>
                        <strong>{run.report[key] || 0}</strong>
                      </div>
                    ))}
                  </div>
                  <Button
                    variant="primary"
                    disabled={
                      busy ||
                      !!run.report.conflicts ||
                      !!run.report.csv_unconfirmed ||
                      !!run.report.broken_dependencies
                    }
                    onClick={() => {
                      setShowConfirm(true);
                      setConfirm("");
                    }}
                  >
                    변경 목록 확인 후 확정
                  </Button>
                </>
              )}
              {run.status === "completed" && (
                <div className="notice subtle">
                  한 번에 반영했습니다. 신규 {run.report.new || 0} · 변경{" "}
                  {run.report.changed || 0} · 동일 {run.report.unchanged || 0} ·
                  제외 {run.report.skipped || 0}.{" "}
                  <Link to="/app/documents">문서 목록 열기</Link>
                </div>
              )}
            </section>
          )}
        </div>
      </div>
      <Modal
        open={!!review}
        onOpenChange={(v) => {
          if (!v && !busy) setReview(null);
        }}
        title="원문과 변환 품질 검토"
        description={review?.item?.file_path}
        wide
      >
        {review && (
          <>
            <RecoveryNotice error={error} />
            <p>
              <Badge>{names[review.item.compatibility.status]}</Badge> · 원본 ID{" "}
              {review.item.source_id}
            </p>
            {review.item.compatibility.warnings?.map((w: string) => (
              <p className="notice subtle" key={w}>
                {w}
              </p>
            ))}
            {review.item.compatibility.links && (
              <p>
                복원 링크 {review.item.compatibility.links.resolved} · 미해결{" "}
                {review.item.compatibility.links.unresolved}
              </p>
            )}
            {review.item.compatibility.conversion_losses && (
              <ul>
                {Object.entries(
                  review.item.compatibility.conversion_losses as Record<
                    string,
                    number
                  >,
                )
                  .filter(([, count]) => count > 0)
                  .map(([key, count]) => (
                    <li key={key}>
                      {(
                        {
                          removed_executable: "실행 요소 제거",
                          remote_images_omitted: "원격·미지원 이미지 제외",
                          merged_cells_flattened: "병합 셀 단순화",
                          style_attributes_omitted: "스타일 제외",
                          interactive_elements_flattened:
                            "상호작용 요소 단순화",
                        } as Record<string, string>
                      )[key] || key}
                      : {count}개
                    </li>
                  ))}
              </ul>
            )}
            {review.item.kind === "csv" ? (
              <>
                <p>
                  {review.row_count}개 행. 전화번호의 선행 0과 큰 정수는 기본
                  텍스트로 보존합니다. 추론이 맞는지 확인하세요.
                </p>
                <div className="table-scroll">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>열 이름</th>
                        <th>자료형</th>
                        <th>검증 표본</th>
                      </tr>
                    </thead>
                    <tbody>
                      {columns.map((col, i) => (
                        <tr key={i}>
                          <td>{col.name}</td>
                          <td>
                            <select
                              aria-label={`${col.name} 자료형`}
                              value={col.type}
                              disabled={busy || run?.status !== "ready"}
                              onChange={(e) => {
                                setColumns((v) =>
                                  v.map((c, j) =>
                                    j === i
                                      ? { ...c, type: e.target.value }
                                      : c,
                                  ),
                                );
                                setTypesConfirmed(false);
                              }}
                            >
                              {types.map(([id, name]) => (
                                <option key={id} value={id}>
                                  {name}
                                </option>
                              ))}
                            </select>
                            {col.type === "select" && (
                              <Field
                                label={`${col.name} 선택 옵션`}
                                hint="한 줄에 하나 · 최대 100개 · 원문과 같은 값"
                              >
                                <textarea
                                  value={(col.options || []).join("\n")}
                                  disabled={busy || run?.status !== "ready"}
                                  onChange={(e) => {
                                    const options = e.target.value.split("\n");
                                    setColumns((v) =>
                                      v.map((c, j) =>
                                        j === i ? { ...c, options } : c,
                                      ),
                                    );
                                    setTypesConfirmed(false);
                                  }}
                                />
                              </Field>
                            )}
                          </td>
                          <td>{(col.samples || []).join(" · ")}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                <Field label="CSV 자료형 명시 확인">
                  <input
                    type="checkbox"
                    checked={typesConfirmed}
                    disabled={busy || run?.status !== "ready"}
                    onChange={(e) => setTypesConfirmed(e.target.checked)}
                  />
                </Field>
              </>
            ) : review.item.kind !== "attachment" ? (
              <>
                <div className="resume-source-comparison">
                  <section>
                    <h3>원본</h3>
                    <pre>{review.source}</pre>
                    {review.source_truncated && (
                      <p>앞부분 256KB만 표시합니다.</p>
                    )}
                  </section>
                  <section>
                    <h3>변환 정본</h3>
                    <pre>{review.canonical}</pre>
                    {review.canonical_truncated && (
                      <p>앞부분 256KB만 표시합니다.</p>
                    )}
                  </section>
                </div>
                <p>
                  추가 {review.diff?.added || 0}줄 · 삭제{" "}
                  {review.diff?.removed || 0}줄
                  {review.diff?.truncated ? " · 비교 일부 생략" : ""}
                </p>
                <div className="button-row">
                  <a
                    className="button"
                    href={`/api/v1/migrations/sessions/${selected}/items/${review.item.id}/download`}
                  >
                    원본 전체 내려받기
                  </a>
                  <a
                    className="button"
                    href={`/api/v1/migrations/sessions/${selected}/items/${review.item.id}/download?view=canonical`}
                  >
                    변환 Markdown 내려받기
                  </a>
                </div>
                <details>
                  <summary>줄 단위 변환 비교</summary>
                  <div className="resume-diff" aria-label="원문 변환 비교">
                    {review.diff?.rows?.map((line: any, index: number) => (
                      <pre key={index} className={`diff-${line.kind}`}>
                        {line.kind === "add"
                          ? "+ "
                          : line.kind === "remove"
                            ? "- "
                            : "  "}
                        {line.text}
                      </pre>
                    ))}
                  </div>
                </details>
                {review.current_diff && (
                  <details>
                    <summary>현재 문서와 반영 예정 원문 비교</summary>
                    <div className="resume-diff">
                      {review.current_diff.rows?.map(
                        (line: any, index: number) => (
                          <pre key={index} className={`diff-${line.kind}`}>
                            {line.kind === "add"
                              ? "+ "
                              : line.kind === "remove"
                                ? "- "
                                : "  "}
                            {line.text}
                          </pre>
                        ),
                      )}
                    </div>
                  </details>
                )}
              </>
            ) : (
              <p>바이너리 원본을 보존하며 실행하지 않습니다.</p>
            )}
            {run?.status === "ready" && (
              <ChangeReview
                title="이 항목 처리 방법"
                changes={[
                  {
                    label: review.item.file_path,
                    before: names[review.item.disposition],
                    after:
                      review.item.kind === "csv"
                        ? "확인한 자료형으로 데이터베이스 생성"
                        : review.item.metadata?.decision === "skip"
                          ? "검토한 정본 사용"
                          : "이 항목 반영 제외",
                  },
                ]}
                warnings={
                  review.item.disposition === "conflict"
                    ? [
                        "현재 문서가 이관 이후 변경되었습니다. 덮어쓰지 않고 이 항목을 제외할 수 있습니다.",
                      ]
                    : []
                }
                confirmLabel={
                  review.item.kind === "csv"
                    ? "자료형 확인 저장"
                    : review.item.metadata?.decision === "skip"
                      ? "이 항목 다시 포함"
                      : "이 항목 제외"
                }
                disabled={
                  (review.item.kind === "csv" && !typesConfirmed) ||
                  review.item.kind === "folder"
                }
                busy={busy}
                onCancel={() => setReview(null)}
                onConfirm={() =>
                  void act(async (op, start) => {
                    await api(
                      `/migrations/sessions/${selected}/items/${review.item.id}/review`,
                      "PUT",
                      {
                        revision: run.revision,
                        plan_hash: run.plan_hash,
                        decision:
                          review.item.kind === "csv" ||
                          review.item.metadata?.decision === "skip"
                            ? "apply"
                            : "skip",
                        types: columns,
                        confirm_types: typesConfirmed,
                      },
                    );
                    if (alive(op, start)) {
                      setReview(null);
                      await load();
                    }
                  })
                }
              />
            )}
            {run?.status === "ready" && review.item.kind === "csv" && (
              <Button
                disabled={busy}
                onClick={() =>
                  void act(async (op, start) => {
                    await api(
                      `/migrations/sessions/${selected}/items/${review.item.id}/review`,
                      "PUT",
                      {
                        revision: run.revision,
                        plan_hash: run.plan_hash,
                        decision: "skip",
                      },
                    );
                    if (alive(op, start)) {
                      setReview(null);
                      await load();
                    }
                  })
                }
              >
                CSV 항목 제외
              </Button>
            )}
          </>
        )}
      </Modal>
      <Modal
        open={showConfirm}
        onOpenChange={(v) => {
          if (!busy) setShowConfirm(v);
        }}
        title="이관 변경 확정"
        wide
      >
        {run && (
          <ChangeReview
            title="검토한 항목만 한 번에 반영"
            changes={[
              {
                label: "문서·데이터베이스",
                before: "비공개 staging",
                after: `신규 ${run.report.new || 0} · 변경 ${run.report.changed || 0} · 동일 ${run.report.unchanged || 0} · 제외 ${run.report.skipped || 0}`,
              },
            ]}
            warnings={[
              "신규 문서는 비공개 초안입니다. 현재 사용자 수정·권한·보호 정책이 바뀌면 전체 확정을 중단합니다.",
              "확정 후 취소 버튼으로 기존 문서를 삭제하지 않습니다.",
            ]}
            confirmLabel="이관 확정"
            disabled={confirm !== "IMPORT"}
            busy={busy}
            onCancel={() => setShowConfirm(false)}
            onConfirm={() =>
              void act(async (op, start) => {
                await api(`/migrations/sessions/${selected}/commit`, "POST", {
                  revision: run.revision,
                  plan_hash: run.plan_hash,
                  confirmation: confirm,
                });
                if (alive(op, start)) {
                  setShowConfirm(false);
                  await load();
                  await reload();
                }
              })
            }
          >
            <Field label="확정 문구 IMPORT">
              <input
                autoComplete="off"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
              />
            </Field>
          </ChangeReview>
        )}
      </Modal>
    </section>
  );
}
