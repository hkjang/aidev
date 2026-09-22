import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime, type Doc } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { ChangeReview } from "../review/ChangeReview";
import type { ConflictCandidate } from "../conflicts/KnowledgeConflictsPage";
import "./evidence.css";
type Proposal = {
  id: string;
  document_id: string;
  document_title: string;
  workspace_id: string;
  owner_id: string;
  owner_name: string;
  base_version: number;
  current_version: number;
  provenance: string;
  status: string;
  revision: number;
  merged_version: number;
  can_write: boolean;
  created_at: string;
};
type Detail = Proposal & {
  markdown: string;
  reason: string;
  base_markdown: string;
  current_markdown: string;
  stale: boolean;
  notice: string;
  protection_revision: number;
  events: { revision: number; status: string; note: string }[];
  origin_stale?: boolean;
  origin?: {
    kind: string;
    stale: boolean;
    notice: string;
    run_id?: string;
    candidate_id?: string;
  };
  diff: {
    rows: { kind: string; text: string; count: number }[];
    notice: string;
  };
};
const labels: Record<string, string> = {
  open: "열린 변경안",
  merged: "반영 완료",
  rejected: "반려",
  withdrawn: "철회",
};
export default function KnowledgeProposalsPage() {
  const { user, workspace, documents, notify, publicInfo } = useApp();
  const [params, setParams] = useSearchParams();
  const id = params.get("id") || "",
    documentID = params.get("document_id") || "",
    conflictID = params.get("conflict_id") || "";
  const [origin, setOrigin] = useState<ConflictCandidate | null>(null),
    [originConsent, setOriginConsent] = useState(false);
  const [items, setItems] = useState<Proposal[]>([]),
    [record, setRecord] = useState<Detail | null>(null),
    [source, setSource] = useState<Doc | null>(null),
    [markdown, setMarkdown] = useState(""),
    [reason, setReason] = useState(""),
    [provenance, setProvenance] = useState("human"),
    [consent, setConsent] = useState(false);
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [refresh, setRefresh] = useState(0),
    [compare, setCompare] = useState(false),
    [decision, setDecision] = useState(""),
    [note, setNote] = useState("");
  const generation = useRef(0);
  const [sourceProtection, setSourceProtection] = useState<number | null>(null),
    [sourceStale, setSourceStale] = useState(false);
  const dirty = !!source && (markdown !== source.markdown || !!reason.trim());
  const dirtyRef = useRef(dirty);
  dirtyRef.current = dirty;
  const discard = () =>
    !dirty || window.confirm("공유하지 않은 변경안 입력을 버리고 이동할까요?");
  useEffect(() => {
    const leave = (event: BeforeUnloadEvent) => {
      if (dirtyRef.current) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, []);
  useEffect(() => {
    const request = ++generation.current;
    let active = true;
    setRecord(null);
    setSource(null);
    setOrigin(null);
    setOriginConsent(false);
    setSourceProtection(null);
    setSourceStale(false);
    setItems([]);
    setMarkdown("");
    setReason("");
    setConsent(false);
    setCompare(false);
    setDecision("");
    setNote("");
    setError("");
    setLoading(true);
    setBusy(false);
    const load = async () => {
      try {
        if (!workspace) return;
        if (id) {
          const next = await api<Detail>(`/knowledge/proposals/${id}`);
          if (active && request === generation.current) setRecord(next);
        } else {
          const list = await api<Proposal[]>(
            `/knowledge/proposals?workspace_id=${workspace.id}`,
          );
          if (active && request === generation.current) setItems(list);
          if (documentID) {
            const before = await api<{
              version: number;
              protection_revision: number;
            }>(`/documents/${documentID}/proposal-context`);
            const doc = await api<Doc>(`/documents/${documentID}`);
            const after = await api<{
              version: number;
              protection_revision: number;
            }>(`/documents/${documentID}/proposal-context`);
            const comparison = conflictID
              ? await api<ConflictCandidate>(
                  `/knowledge/conflict-candidates/${conflictID}`,
                )
              : null;
            if (
              comparison &&
              (comparison.stale ||
                comparison.workspace_id !== workspace.id ||
                ![
                  comparison.left.document_id,
                  comparison.right.document_id,
                ].includes(documentID))
            )
              throw Error(
                "비교 원문 또는 판단이 변경되었습니다. 본인 보고서에서 다시 분석하세요.",
              );
            if (
              before.version !== doc.version ||
              after.version !== doc.version ||
              after.protection_revision !== before.protection_revision
            )
              throw Error(
                "원문 또는 보호 정책이 변경되었습니다. 다시 확인하세요.",
              );
            if (active && request === generation.current) {
              if (doc.workspace_id !== workspace.id)
                throw Error("현재 워크스페이스 문서만 선택하세요");
              setSource(doc);
              setMarkdown(doc.markdown);
              setSourceProtection(after.protection_revision);
              setOrigin(comparison);
            }
          }
        }
      } catch (e) {
        if (active && request === generation.current) {
          setRecord(null);
          setSource(null);
          setError((e as Error).message);
        }
      } finally {
        if (active && request === generation.current) setLoading(false);
      }
    };
    void load();
    return () => {
      active = false;
      generation.current++;
    };
  }, [id, documentID, conflictID, user.id, workspace?.id, refresh]);
  useEffect(() => {
    if (!source || sourceProtection === null) return;
    let active = true,
      pending = false;
    const check = async () => {
      if (pending) return;
      pending = true;
      try {
        const v = await api<{ version: number; protection_revision: number }>(
          `/documents/${source.id}/proposal-context`,
        );
        if (origin) {
          const next = await api<ConflictCandidate>(
            `/knowledge/conflict-candidates/${origin.id}`,
          );
          if (next.stale || next.revision !== origin.revision)
            throw Error(
              "비교 출처 또는 판단이 변경되어 제안 입력을 지웠습니다. 개인 보고서에서 다시 분석하세요.",
            );
        }
        if (v.protection_revision !== sourceProtection)
          throw Error(
            "보호 정책이 변경되어 표시 중인 원문과 제안 입력을 지웠습니다. 현재 권한으로 다시 확인하세요.",
          );
        if (active && v.version !== source.version) {
          setSourceStale(true);
          setCompare(false);
          setError(
            "기준 원문이 바뀌었습니다. 제안 입력은 이 탭에만 유지하고, 현재 원문을 다시 비교한 후 새 변경안을 작성하세요.",
          );
        }
      } catch (e) {
        if (active) {
          setSource(null);
          setOrigin(null);
          setOriginConsent(false);
          setMarkdown("");
          setReason("");
          setCompare(false);
          setConsent(false);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
      }
    };
    const timer = setInterval(() => void check(), 2000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [source, sourceProtection, origin]);
  useEffect(() => {
    if (!record) return;
    let active = true,
      pending = false;
    const check = async () => {
      if (pending) return;
      pending = true;
      try {
        const next = await api<{
          revision: number;
          current_version: number;
          protection_revision: number;
          origin_stale?: boolean;
        }>(`/knowledge/proposals/${record.id}/check`);
        if (
          active &&
          (next.revision !== record.revision ||
            next.current_version !== record.current_version ||
            !!next.origin_stale !== !!record.origin_stale ||
            next.protection_revision !== record.protection_revision)
        ) {
          setRecord(null);
          setCompare(false);
          setDecision("");
          setError(
            "변경안·현재 원문·보호 정책이 달라졌습니다. 다시 불러와 비교하세요.",
          );
        }
      } catch (e) {
        if (active) {
          setRecord(null);
          setCompare(false);
          setDecision("");
          setError((e as Error).message);
        }
      } finally {
        pending = false;
      }
    };
    const timer = setInterval(() => void check(), 2000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [record, user.id, workspace?.id]);
  const perform = async (action: () => Promise<void>) => {
    if (busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  const create = () =>
    perform(async () => {
      if (!source) return;
      const request = generation.current;
      const result = await api<{ id: string }>(
        `/documents/${source.id}/proposals`,
        "POST",
        {
          base_version: source.version,
          markdown,
          reason,
          provenance,
          consent,
          ...(origin
            ? {
                conflict_id: origin.id,
                conflict_revision: origin.revision,
                conflict_consent: originConsent,
              }
            : {}),
        },
      );
      if (request === generation.current) {
        setParams({ id: result.id });
        notify("원문은 유지하고 변경안을 공유했습니다");
      }
    });
  const merge = () =>
    perform(async () => {
      if (!record) return;
      const request = generation.current;
      await api(`/knowledge/proposals/${record.id}/merge`, "POST", {
        revision: record.revision,
        consent: true,
        note,
      });
      if (request === generation.current) {
        setRefresh((v) => v + 1);
        notify(
          "변경안과 문서 버전을 함께 저장했습니다. 게시 상태를 확인하세요",
        );
      }
    });
  const decide = () =>
    perform(async () => {
      if (!record) return;
      const request = generation.current;
      await api(`/knowledge/proposals/${record.id}/decision`, "POST", {
        revision: record.revision,
        status: decision,
        note,
      });
      if (request === generation.current) {
        setRefresh((v) => v + 1);
        notify("변경안 처리 상태를 저장했습니다");
      }
    });
  return (
    <div className="page evidence-page">
      <PageHeading
        title="지식 변경 제안"
        description="사람과 AI의 수정 초안을 원문과 비교하고, 현재 버전에서만 명시적으로 반영합니다."
      />
      <ErrorBox error={error} />
      <div className="button-row">
        <Button
          disabled={busy}
          onClick={() => {
            if (discard()) setRefresh((v) => v + 1);
          }}
        >
          서버 내용 다시 읽기
        </Button>
        {id && <Button onClick={() => setParams({})}>변경안 목록</Button>}
      </div>
      {loading ? (
        <Loading />
      ) : id ? (
        record && (
          <>
            <section className="card">
              <h2>{record.document_title}</h2>
              <p>
                {record.owner_name} · 기준 v{record.base_version} · 현재 v
                {record.current_version} · {labels[record.status]}
              </p>
              <p className="notice">{record.notice}</p>
              <p>
                작성 방식(작성자 표기):{" "}
                {record.provenance === "ai_assisted"
                  ? "AI 도움을 받아 작성"
                  : "사람이 작성"}
              </p>
              <h3>변경 이유</h3>
              <pre className="evidence-text">{record.reason}</pre>
              <Link to={`/app/documents/${record.document_id}`}>
                현재 문서 전체 확인
              </Link>
              {record.stale && (
                <p role="note">
                  기준 원문이 바뀌었습니다. 자동으로 병합하지 않습니다. 현재
                  원문과 제안을 비교해 새 변경안을 작성하세요.
                </p>
              )}
            </section>
            <section className="card">
              <h2>변경 내용 비교</h2>
              {record.origin && (
                <div className="notice" style={{ display: "block" }}>
                  <strong>문서 차이 검토에서 연결한 변경안</strong>
                  <p>{record.origin.notice}</p>
                  {record.origin_stale && (
                    <p>
                      비교 출처 또는 판단이 바뀌어 원문 반영이 중단되었습니다.
                    </p>
                  )}
                  {record.origin.run_id && (
                    <Link
                      to={`/app/knowledge-conflicts?id=${record.origin.run_id}&candidate=${record.origin.candidate_id}`}
                    >
                      내 비교 보고서 확인
                    </Link>
                  )}
                </div>
              )}
              <div className="evidence-compare">
                <div>
                  <h3>기준 원문 v{record.base_version}</h3>
                  <pre className="evidence-text">{record.base_markdown}</pre>
                </div>
                <div>
                  <h3>제안 원문</h3>
                  <pre className="evidence-text">{record.markdown}</pre>
                </div>
              </div>
              <details>
                <summary>차이 구간</summary>
                <p>{record.diff.notice}</p>
                <pre className="evidence-text">
                  {record.diff.rows
                    .map((row) =>
                      row.kind === "skip"
                        ? `… ${row.count}줄 생략 …\n`
                        : `${row.kind === "add" ? "+" : row.kind === "remove" ? "−" : " "} ${row.text}`,
                    )
                    .join("")}
                </pre>
              </details>
              {record.stale && (
                <details>
                  <summary>현재 문서 원문 v{record.current_version}</summary>
                  <pre className="evidence-text">{record.current_markdown}</pre>
                </details>
              )}
            </section>
            {record.status === "open" && (
              <section className="card">
                <Field label="변경안 처리 의견">
                  <textarea
                    value={note}
                    maxLength={1000}
                    rows={3}
                    onChange={(e) => setNote(e.target.value)}
                  />
                </Field>
                <div className="button-row">
                  <Button
                    variant="primary"
                    disabled={busy || !record.can_write || record.stale}
                    onClick={() => setCompare(true)}
                  >
                    비교 후 변경안 반영
                  </Button>
                  {record.owner_id === user.id && (
                    <Button
                      disabled={busy || !record.can_write}
                      onClick={() => setDecision("withdrawn")}
                    >
                      내 변경안 철회
                    </Button>
                  )}
                  {publicInfo.approval_enabled && record.can_write && (
                    <Button
                      disabled={busy}
                      onClick={() => setDecision("rejected")}
                    >
                      검토 후 반려
                    </Button>
                  )}
                  <Link
                    to={`/app/knowledge-proposals?document_id=${record.document_id}`}
                  >
                    현재 원문에서 새 변경안
                  </Link>
                </div>
              </section>
            )}
            {!!record.events.length && (
              <section className="card">
                <h2>변경안 처리 이력</h2>
                {record.events.map((event) => (
                  <div className="evidence-review" key={event.revision}>
                    <strong>{labels[event.status]}</strong> · revision{" "}
                    {event.revision}
                    <pre className="evidence-text">{event.note}</pre>
                  </div>
                ))}
              </section>
            )}
          </>
        )
      ) : (
        <>
          <section className="card">
            <h2>원문을 유지하며 수정 제안</h2>
            <Field label="변경안을 작성할 문서">
              <select
                value={documentID}
                onChange={(e) =>
                  discard() &&
                  setParams(
                    e.target.value ? { document_id: e.target.value } : {},
                  )
                }
              >
                <option value="">문서를 선택하세요</option>
                {documents
                  .filter((d) => d.can_write)
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
              </select>
            </Field>
            {source && (
              <>
                <p>
                  기준 문서 v{source.version} · 이 화면은 공동 편집 원문을 바로
                  변경하지 않습니다.
                </p>
                <p>
                  이 입력은 아직 서버에 공유되지 않았고 이 화면을 떠나면
                  사라집니다. 권한 또는 보호 정책이 바뀌면 원문과 입력 표시를
                  제거합니다.
                </p>
                <Field label="변경 이유">
                  <textarea
                    value={reason}
                    maxLength={1000}
                    rows={2}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </Field>
                {origin && (
                  <div className="notice" style={{ display: "block" }}>
                    <strong>비교 후보를 변경안의 출처로 연결</strong>
                    <p>
                      두 원문의 현재 접근 권한과 분석 버전이 모두 확인되어야
                      열람·반영할 수 있습니다. 다른 원문의 제목·구간이나 개인
                      판단 메모를 변경 이유에 자동 복사하지 않습니다.
                    </p>
                    <Link
                      to={`/app/knowledge-conflicts?id=${origin.run_id}&candidate=${origin.id}`}
                    >
                      내 비교 구간 확인
                    </Link>
                    <label className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={originConsent}
                        onChange={(e) => setOriginConsent(e.target.checked)}
                      />
                      비교 출처를 연결하고 두 원문 접근 권한을 함께 적용하는 데
                      동의합니다.
                    </label>
                  </div>
                )}
                <Field label="작성 방식">
                  <select
                    value={provenance}
                    onChange={(e) => setProvenance(e.target.value)}
                  >
                    <option value="human">사람이 작성</option>
                    <option value="ai_assisted">
                      AI 도움을 받아 작성 (작성자 표기)
                    </option>
                  </select>
                </Field>
                <Field label="제안 Markdown 원문">
                  <textarea
                    value={markdown}
                    onChange={(e) => setMarkdown(e.target.value)}
                    rows={18}
                    spellCheck={false}
                  />
                </Field>
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={consent}
                    onChange={(e) => setConsent(e.target.checked)}
                  />
                  현재 문서 열람자가 이 제안과 변경 이유를 볼 수 있음에
                  동의합니다.
                </label>
                <Button
                  variant="primary"
                  disabled={
                    busy ||
                    sourceStale ||
                    !consent ||
                    (!!origin && !originConsent) ||
                    !reason.trim() ||
                    !source.can_write ||
                    markdown === source.markdown
                  }
                  onClick={() => setCompare(true)}
                >
                  공유 전 변경 비교
                </Button>
              </>
            )}
          </section>
          <section className="card">
            <h2>최근 변경안</h2>
            <p>
              현재 접근 가능한 최신 200건 · 개인별 열린 변경안은 최대
              100개입니다.
            </p>
            {!items.length ? (
              <Empty
                title="표시할 변경안이 없습니다"
                text="문서를 선택하고 원문을 유지한 채 수정 제안을 작성하세요."
              />
            ) : (
              items.map((item) => (
                <div className="evidence-review" key={item.id}>
                  <Link to={`?id=${item.id}`}>{item.document_title}</Link>
                  <p>
                    {labels[item.status]} · 기준 v{item.base_version} ·{" "}
                    {item.owner_name} · {datetime(item.created_at)}
                  </p>
                </div>
              ))
            )}
          </section>
        </>
      )}
      <Modal
        open={compare}
        onOpenChange={(open) => {
          if (!busy) setCompare(open);
        }}
        title={
          record ? "현재 원문에 변경안 반영" : "원문을 유지하고 변경안 공유"
        }
      >
        <ChangeReview
          title={record ? "원문 반영 확인" : "공유할 변경 내용"}
          changes={[
            {
              label: "문서 본문",
              before: (
                <pre className="evidence-text">
                  {record?.current_markdown ?? source?.markdown}
                </pre>
              ),
              after: (
                <pre className="evidence-text">
                  {record?.markdown ?? markdown}
                </pre>
              ),
            },
          ]}
          warnings={[
            record
              ? "기준 버전이 바뀌면 반영하지 않습니다. 관리자 승인 정책이 켜져 있으면 반영 후에도 별도 게시 승인이 필요합니다."
              : "제안은 문서의 현재 열람 권한을 따릅니다. 원문은 변경하지 않습니다.",
          ]}
          confirmLabel={
            record ? "현재 버전에 변경안 반영" : "동의한 범위로 변경안 공유"
          }
          busy={busy}
          onCancel={() => setCompare(false)}
          onConfirm={() => void (record ? merge() : create())}
        />
      </Modal>
      <Modal
        open={!!decision}
        onOpenChange={(open) => {
          if (!open && !busy) setDecision("");
        }}
        title={decision === "rejected" ? "변경안 반려" : "내 변경안 철회"}
      >
        <p>
          문서 원문은 바꾸지 않고 이 변경안을 닫습니다. 처리 의견을 남겨 주세요.
        </p>
        <Field label="처리 사유">
          <textarea
            value={note}
            maxLength={1000}
            onChange={(e) => setNote(e.target.value)}
          />
        </Field>
        <Button disabled={busy || !note.trim()} onClick={() => void decide()}>
          변경안 처리 확인
        </Button>
      </Modal>
    </div>
  );
}
