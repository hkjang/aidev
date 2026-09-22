import { useEffect, useRef, useState } from "react";
import {
  ArrowDownToLine,
  FileDiff,
  History,
  RefreshCw,
  RotateCcw,
} from "lucide-react";
import { api, datetime, downloadText, type Doc, type DocSummary } from "../api";
import { Button, Empty, ErrorBox, Field, Loading, Modal, Badge } from "../ui";
import { MarkdownContent } from "../editor/MarkdownContent";
import "./style.css";
type Version = {
  version: number;
  title: string;
  created_at: string;
  user_name: string;
  markdown_bytes: number;
  current_version: number;
};
type Snapshot = Version & {
  markdown: string;
  tags: string[];
  block_metadata?: Doc["block_metadata"];
};
type Diff = {
  from: number;
  to: number;
  current_version: number;
  title: { before: string; after: string };
  tags: { before: string[]; after: string[] };
  diff: {
    rows: {
      kind: "equal" | "add" | "remove" | "skip";
      text: string;
      old_line?: number;
      new_line?: number;
      count?: number;
    }[];
    added: number;
    removed: number;
    coarse: boolean;
    truncated: boolean;
    notice: string;
    before_bytes: number;
    after_bytes: number;
  };
};

export default function DocumentHistory({
  documentID,
  currentVersion,
  canWrite,
  documents,
  onClose,
  onRestored,
}: {
  documentID: string;
  currentVersion: number;
  canWrite: boolean;
  documents: DocSummary[];
  onClose: () => void;
  onRestored: () => Promise<void>;
}) {
  const [versions, setVersions] = useState<Version[]>([]),
    [expected, setExpected] = useState(currentVersion),
    [selected, setSelected] = useState<number | null>(null),
    [snapshot, setSnapshot] = useState<Snapshot | null>(null),
    [from, setFrom] = useState(0),
    [to, setTo] = useState(0),
    [diff, setDiff] = useState<Diff | null>(null),
    [tab, setTab] = useState<"list" | "preview" | "diff">("list"),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [more, setMore] = useState(false),
    [confirm, setConfirm] = useState(false),
    [source, setSource] = useState(false);
  const active = useRef(true),
    generation = useRef(0),
    selectionGeneration = useRef(0),
    busyRef = useRef(false);
  const path = `/documents/${documentID}/versions`;
  async function load(older = false) {
    const ticket = ++generation.current;
    setLoading(true);
    setError(null);
    try {
      const query =
        older && versions.length ? `?before=${versions.at(-1)!.version}` : "";
      const rows = await api<Version[]>(path + query);
      if (!active.current || ticket !== generation.current) return;
      setVersions((old) => (older ? [...old, ...rows] : rows));
      setMore(rows.length === 50);
      if (!older) {
        setExpected(rows[0]?.current_version || currentVersion);
        setFrom(rows[1]?.version || rows[0]?.version || 0);
        setTo(rows[0]?.version || 0);
        setDiff(null);
        setSnapshot(null);
        setSelected(null);
        setTab("list");
      }
    } catch (e) {
      if (active.current && ticket === generation.current) setError(e);
    } finally {
      if (active.current && ticket === generation.current) setLoading(false);
    }
  }
  useEffect(() => {
    active.current = true;
    void load();
    return () => {
      active.current = false;
      generation.current++;
      selectionGeneration.current++;
    };
  }, [documentID]);
  async function show(version: number) {
    const ticket = ++selectionGeneration.current;
    setSelected(version);
    setSnapshot(null);
    setTab("preview");
    setError(null);
    try {
      const value = await api<Snapshot>(path + "/" + version);
      if (active.current && ticket === selectionGeneration.current)
        setSnapshot(value);
    } catch (e) {
      if (active.current && ticket === selectionGeneration.current) setError(e);
    }
  }
  async function compare() {
    const ticket = ++selectionGeneration.current;
    setTab("diff");
    setDiff(null);
    setError(null);
    try {
      const result = await api<Diff>(path + `/diff?from=${from}&to=${to}`);
      if (active.current && ticket === selectionGeneration.current)
        setDiff(result);
    } catch (e) {
      if (active.current && ticket === selectionGeneration.current) setError(e);
    }
  }
  async function restore() {
    if (!selected || busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError(null);
    try {
      await api(path + `/${selected}/restore`, "POST", {
        expected_version: expected,
      });
      if (!active.current) return;
      await onRestored();
      if (active.current) onClose();
    } catch (e) {
      if (active.current) {
        setError(e);
        setConfirm(false);
      }
    } finally {
      busyRef.current = false;
      if (active.current) setBusy(false);
    }
  }
  return (
    <Modal
      open
      title="문서 변경 이력"
      description={`이력 확인 기준은 현재 버전 ${expected}입니다. 원문은 선택한 버전만 불러오며, 복원은 새로운 문서 버전으로 저장합니다.`}
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
      wide
    >
      <div className="document-history">
        <div className="history-toolbar">
          <div className="actions">
            <Button
              variant="secondary"
              disabled={busy}
              onClick={() => {
                selectionGeneration.current++;
                setTab("list");
                setSnapshot(null);
                setSelected(null);
                setDiff(null);
              }}
            >
              <History size={16} />
              버전 목록
            </Button>
            <Button
              variant="secondary"
              disabled={busy || loading}
              onClick={() => void load()}
            >
              <RefreshCw size={16} />
              최신 이력 다시 불러오기
            </Button>
          </div>
          <Badge>현재 v{expected}</Badge>
        </div>
        <ErrorBox error={error} />
        {versions.length > 0 && (
          <div className="history-comparison">
            <Field label="비교 이전 버전">
              <select
                value={from}
                disabled={busy}
                onChange={(e) => {
                  selectionGeneration.current++;
                  setFrom(Number(e.target.value));
                  setDiff(null);
                  setTab("list");
                }}
              >
                {versions.map((v) => (
                  <option key={v.version} value={v.version}>
                    버전 {v.version} · {v.title}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="비교 다음 버전">
              <select
                value={to}
                disabled={busy}
                onChange={(e) => {
                  selectionGeneration.current++;
                  setTo(Number(e.target.value));
                  setDiff(null);
                  setTab("list");
                }}
              >
                {versions.map((v) => (
                  <option key={v.version} value={v.version}>
                    버전 {v.version} · {v.title}
                  </option>
                ))}
              </select>
            </Field>
            <Button
              variant="secondary"
              disabled={busy || !from || !to}
              onClick={() => void compare()}
            >
              <FileDiff size={16} />
              변경 비교
            </Button>
          </div>
        )}
        {tab === "list" ? (
          <>
            {loading && versions.length === 0 ? (
              <Loading />
            ) : versions.length === 0 ? (
              <Empty
                title="변경 이력이 없습니다"
                text="문서 저장과 공동 편집의 변경 내용이 버전으로 기록됩니다."
              />
            ) : (
              <div className="history-version-list">
                {versions.map((v) => (
                  <article key={v.version}>
                    <div>
                      <strong>버전 {v.version}</strong>
                      <span>{v.title}</span>
                      <small>
                        {datetime(v.created_at)} · {v.user_name} ·{" "}
                        {(v.markdown_bytes / 1024).toFixed(1)}KB
                      </small>
                    </div>
                    <Button
                      variant="secondary"
                      disabled={busy}
                      aria-label={`버전 ${v.version} 원문 보기`}
                      onClick={() => void show(v.version)}
                    >
                      원문 보기
                    </Button>
                  </article>
                ))}
              </div>
            )}
            {more && (
              <Button
                variant="secondary"
                disabled={loading || busy}
                onClick={() => void load(true)}
              >
                {loading ? "불러오는 중…" : "이전 이력 더 보기"}
              </Button>
            )}
          </>
        ) : tab === "preview" ? (
          snapshot ? (
            <section className="history-preview">
              <div className="history-toolbar">
                <h2>
                  버전 {snapshot.version} · {snapshot.title}
                </h2>
                <div className="actions">
                  <Button
                    variant="secondary"
                    onClick={() => setSource((v) => !v)}
                  >
                    {source ? "문서 미리보기" : "Markdown 원문"}
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() =>
                      downloadText(
                        `${snapshot.title}-v${snapshot.version}.md`,
                        snapshot.markdown,
                      )
                    }
                  >
                    <ArrowDownToLine size={16} />
                    원문 다운로드
                  </Button>
                  {canWrite && (
                    <Button
                      disabled={busy || snapshot.version === expected}
                      onClick={() => setConfirm(true)}
                    >
                      <RotateCcw size={16} />이 버전으로 복원
                    </Button>
                  )}
                </div>
              </div>
              {source ? (
                <pre className="history-source">{snapshot.markdown}</pre>
              ) : (
                <MarkdownContent
                  markdown={snapshot.markdown}
                  metadata={snapshot.block_metadata}
                  documents={documents}
                />
              )}
            </section>
          ) : error ? null : (
            <Loading />
          )
        ) : diff ? (
          <section className="history-diff">
            <div className="history-toolbar">
              <h2>
                버전 {diff.from} → {diff.to}
              </h2>
              <div className="actions">
                <Badge tone="green">추가 {diff.diff.added}줄</Badge>
                <Badge tone="amber">삭제 {diff.diff.removed}줄</Badge>
              </div>
            </div>
            {diff.title.before !== diff.title.after && (
              <p>
                제목: {diff.title.before} → {diff.title.after}
              </p>
            )}
            {JSON.stringify(diff.tags.before) !==
              JSON.stringify(diff.tags.after) && (
              <p>
                태그: {(diff.tags.before || []).join(", ") || "없음"} →{" "}
                {(diff.tags.after || []).join(", ") || "없음"}
              </p>
            )}
            {diff.diff.notice && (
              <div className="notice warning">{diff.diff.notice}</div>
            )}
            {diff.diff.rows.length === 0 ? (
              <p>
                {diff.diff.truncated
                  ? "표시 한도를 초과했습니다. 각 버전 원문을 내려받아 비교하세요."
                  : "Markdown 원문이 같습니다."}
              </p>
            ) : (
              <div
                className="history-diff-table"
                role="table"
                aria-label="문서 원문 줄 변경 비교"
              >
                <div role="row" className="history-diff-header">
                  <span role="columnheader">이전</span>
                  <span role="columnheader">다음</span>
                  <span role="columnheader">변경 원문</span>
                </div>
                {diff.diff.rows.map((row, i) => (
                  <div
                    role="row"
                    key={i}
                    className={`history-diff-row ${row.kind}`}
                  >
                    <span role="cell">{row.old_line || ""}</span>
                    <span role="cell">{row.new_line || ""}</span>
                    <code role="cell">
                      <b
                        aria-label={
                          row.kind === "add"
                            ? "추가"
                            : row.kind === "remove"
                              ? "삭제"
                              : "동일"
                        }
                      >
                        {row.kind === "add"
                          ? "+"
                          : row.kind === "remove"
                            ? "−"
                            : " "}
                      </b>
                      {row.kind === "skip" ? (
                        `동일한 ${row.count}줄 접음`
                      ) : row.text.endsWith("\n") ? (
                        row.text.slice(0, -1)
                      ) : (
                        <>
                          {row.text}
                          <em> [끝 줄바꿈 없음]</em>
                        </>
                      )}
                    </code>
                  </div>
                ))}
              </div>
            )}
          </section>
        ) : error ? null : (
          <Loading />
        )}
        <Modal
          open={confirm}
          onOpenChange={(value) => {
            if (!busy) setConfirm(value);
          }}
          title="문서 버전 복원"
          description="현재 내용과 비교한 뒤 복원하세요. 이력 확인 이후 다른 사용자가 저장한 경우 복원을 중단합니다."
        >
          <p>
            버전 {selected}의 제목·Markdown·태그·블록 정보를 새 버전으로
            저장합니다. 현재 공유 범위·소유권은 유지하고 기존 승인·정보보호
            정책을 다시 적용합니다.
          </p>
          <div className="actions">
            <Button
              variant="secondary"
              disabled={busy}
              onClick={() => setConfirm(false)}
            >
              돌아가기
            </Button>
            <Button disabled={busy} onClick={() => void restore()}>
              {busy ? "복원 중…" : "복원 확인"}
            </Button>
          </div>
        </Modal>
      </div>
    </Modal>
  );
}
