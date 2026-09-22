import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime } from "../api";
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
import "./evidence.css";
import KnowledgePolicyHistory from "./KnowledgePolicyHistory";
import { type Position, positionName } from "../attachments/types";

type Policy = { enabled: boolean; retention_days: number; version: number };
type Entry = {
  id: string;
  created_at: string;
  expires_at: string;
  source_count: number;
  review_status: string;
  version: number;
};
type Record = Entry & {
  question: string;
  answer: string;
  model: string;
  max_tokens: number;
  notice: string;
  settings_fingerprint: string;
  sources: {
    source: {
      id: string;
      title: string;
      version: number;
      start_line: number;
      end_line: number;
      content_hash: string;
      attachment_id?: string;
      attachment_position?: Position;
      extraction_id?: string;
      fragment_id?: string;
      start_byte: number;
      end_byte: number;
    };
    text: string;
    integrity: string;
    freshness: string;
    current_version: number;
    current_title: string;
    current_text: string;
    same_span: boolean;
  }[];
  reviews: {
    status: string;
    note: string;
    created_at: string;
    version: number;
  }[];
};
const labels: { [key: string]: string } = {
  unreviewed: "미검토",
  supported: "사용자가 근거 연결 확인",
  insufficient: "근거 부족",
  misinterpreted: "해석 오류",
};

export default function EvidencePage() {
  const { user, workspace } = useApp();
  const [params, setParams] = useSearchParams(),
    id = params.get("id") || "";
  const [items, setItems] = useState<Entry[]>([]),
    [record, setRecord] = useState<Record | null>(null),
    [policy, setPolicy] = useState<Policy | null>(null);
  const [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const [review, setReview] = useState("unreviewed"),
    [note, setNote] = useState(""),
    [remove, setRemove] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    const request = ++generation.current;
    let active = true,
      pending = false;
    setItems([]);
    setRecord(null);
    setPolicy(null);
    setError("");
    setLoading(true);
    setBusy(false);
    setRemove(false);
    setNote("");
    setReview("unreviewed");
    const load = async (initial = false) => {
      if (!workspace || pending) return;
      pending = true;
      try {
        const nextPolicy = await api<Policy>("/ai/evidence/policy");
        const data = id
          ? await api<Record>(`/ai/evidence/${id}`)
          : await api<Entry[]>(`/ai/evidence?workspace_id=${workspace.id}`);
        if (!active || request !== generation.current) return;
        setPolicy(nextPolicy);
        if (id) setRecord(data as Record);
        else setItems(data as Entry[]);
        setError("");
      } catch (e) {
        if (active && request === generation.current) {
          setRecord(null);
          setItems([]);
          setRemove(false);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
        if (initial && active && request === generation.current)
          setLoading(false);
      }
    };
    void load(true);
    const timer = window.setInterval(() => void load(), 2000);
    return () => {
      active = false;
      generation.current++;
      window.clearInterval(timer);
    };
  }, [id, user.id, workspace?.id, refresh]);
  const mutate = async (deleting = false) => {
    if (!record || busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await api(
        `/ai/evidence/${record.id}${deleting ? "" : "/reviews"}`,
        deleting ? "DELETE" : "POST",
        deleting
          ? { version: record.version, confirmation: "DELETE" }
          : { version: record.version, status: review, note },
      );
      if (request !== generation.current) return;
      if (deleting) setParams({});
      else setRefresh((v) => v + 1);
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  return (
    <div className="page evidence-page">
      <PageHeading
        title="AI 근거 보관함"
        description="당시 원문의 무결성, 현재 최신성, 사람의 해석 검토를 구분합니다."
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        <>
          {id && <Button onClick={() => setParams({})}>보관 목록으로</Button>}
          {!id && (
            <>
              <p className="notice">
                {policy?.enabled
                  ? `완료된 AI 답변의 ‘당시 근거 보관’에서 저장합니다. 보존기간은 최대 ${policy.retention_days}일이며 본인만 열람할 수 있습니다.`
                  : "관리자가 근거 보관 정책을 활성화해야 합니다. 비활성화 상태에서는 보관 기록을 표시하지 않습니다."}
              </p>
              {!items.length ? (
                <Empty
                  title="표시할 근거가 없습니다"
                  text="문서 출처가 있는 AI 답변에서 명시적으로 보관하세요. 접근 권한이 없거나 만료된 사본은 표시하지 않습니다."
                />
              ) : (
                <div className="evidence-list">
                  {items.map((item) => (
                    <Link className="card" key={item.id} to={`?id=${item.id}`}>
                      <strong>{datetime(item.created_at)} 근거</strong>
                      <span>
                        출처 {item.source_count}개 ·{" "}
                        {labels[item.review_status]}
                      </span>
                      <small>보존 종료 {datetime(item.expires_at)}</small>
                    </Link>
                  ))}
                </div>
              )}
            </>
          )}
          {record && (
            <>
              <section className="card">
                <h2>당시 질문과 답변</h2>
                <p className="notice">{record.notice}</p>
                <h3>질문</h3>
                <pre className="evidence-text">{record.question}</pre>
                <h3>AI 답변</h3>
                <pre className="evidence-text">{record.answer}</pre>
                <p>
                  모델 {record.model} · 응답 최대{" "}
                  {record.max_tokens.toLocaleString()} 토큰 ·{" "}
                  {labels[record.review_status]}
                </p>
                <details>
                  <summary>설정 식별값·보존 정보</summary>
                  <code className="evidence-hash">
                    {record.settings_fingerprint}
                  </code>
                  <p>
                    저장 {datetime(record.created_at)} · 만료{" "}
                    {datetime(record.expires_at)}
                  </p>
                </details>
              </section>
              <h2>근거별 원문 비교</h2>
              {record.sources.map((quote, i) => (
                <section className="card" key={`${quote.source.id}:${i}`}>
                  <h3>
                    {i + 1}. {quote.source.title}
                  </h3>
                  <div className="evidence-status">
                    <span>원문 무결성: 일치</span>
                    <span>
                      {quote.freshness === "current"
                        ? "최신 버전"
                        : `변경됨 · 현재 v${quote.current_version}`}
                    </span>
                    <span>주장 연결: {labels[record.review_status]}</span>
                  </div>
                  <div className="evidence-compare">
                    <div>
                      <h4>
                        {quote.source.attachment_id
                          ? `당시 첨부 추출 · ${positionName(quote.source.attachment_position || {})} · 조각 기준 ${quote.source.start_byte}~${quote.source.end_byte}바이트`
                          : `당시 v${quote.source.version} · ${quote.source.start_line}~${quote.source.end_line}행`}
                      </h4>
                      <pre className="evidence-text">{quote.text}</pre>
                    </div>
                    <div>
                      <h4>현재 같은 바이트 구간</h4>
                      <p className="muted">
                        편집으로 위치가 이동했을 수 있습니다.{" "}
                        {quote.same_span
                          ? "이 구간의 내용은 같습니다."
                          : "같은 위치의 내용이 달라졌거나 구간을 찾을 수 없습니다."}
                      </p>
                      <pre className="evidence-text">
                        {quote.current_text ||
                          "표시 가능한 동일 좌표 구간 없음"}
                      </pre>
                      <Link
                        to={
                          quote.source.attachment_id
                            ? `/app/attachments/${quote.source.attachment_id}?extraction=${quote.source.extraction_id}&fragment=${quote.source.fragment_id}`
                            : `/app/documents/${quote.source.id}`
                        }
                      >
                        현재 원본 위치 확인
                      </Link>
                    </div>
                  </div>
                  <details>
                    <summary>원문 SHA-256</summary>
                    <code className="evidence-hash">
                      {quote.source.content_hash}
                    </code>
                  </details>
                </section>
              ))}
              <section className="card">
                <h2>주장과 근거 검토</h2>
                <p>
                  개인 검토 기록입니다. 해시 일치만으로 AI 답변을 올바른 것으로
                  판단하지 마세요.
                </p>
                <Field label="검토 결과">
                  <select
                    value={review}
                    onChange={(event) => setReview(event.target.value)}
                  >
                    {Object.entries(labels).map(([value, label]) => (
                      <option key={value} value={value}>
                        {label}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="검토 의견">
                  <textarea
                    value={note}
                    maxLength={2000}
                    onChange={(event) => setNote(event.target.value)}
                    rows={3}
                  />
                </Field>
                <Button
                  disabled={busy}
                  variant="primary"
                  onClick={() => void mutate()}
                >
                  검토 기록 저장
                </Button>
                {record.reviews.map((entry) => (
                  <div className="evidence-review" key={entry.version}>
                    <strong>{labels[entry.status]}</strong> ·{" "}
                    {datetime(entry.created_at)}
                    <pre className="evidence-text">{entry.note}</pre>
                  </div>
                ))}
              </section>
              <Button onClick={() => setRemove(true)}>이 근거 사본 삭제</Button>
            </>
          )}
        </>
      )}
      <Modal
        open={remove && !!record}
        onOpenChange={(value) => {
          if (!busy) setRemove(value);
        }}
        title="근거 사본 삭제"
      >
        <p>
          이 보관함의 근거 사본과 검토 이력을 삭제합니다. 원문과 개인 대화
          기록은 유지되며, 이미 생성된 백업의 사본은 백업 보존 정책을 따릅니다.
          원문에 보존 의무가 있으면 삭제가 제한됩니다.
        </p>
        <Button
          disabled={busy}
          variant="danger"
          onClick={() => void mutate(true)}
        >
          삭제 확인
        </Button>
      </Modal>
    </div>
  );
}

export function EvidencePolicyPage() {
  const { notify } = useApp();
  const [policy, setPolicy] = useState<Policy | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = async () => {
    try {
      setPolicy(await api<Policy>("/admin/evidence-policy"));
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
  };
  useEffect(() => {
    let active = true;
    void api<Policy>("/admin/evidence-policy")
      .then((value) => {
        if (active) setPolicy(value);
      })
      .catch((e: Error) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, []);
  const save = async () => {
    if (!policy || busy) return;
    setBusy(true);
    setError("");
    try {
      setPolicy(await api<Policy>("/admin/evidence-policy", "PUT", policy));
      notify("AI 근거 보관 정책을 저장했습니다");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page evidence-page">
      <PageHeading
        title="AI 근거 보관 정책"
        description="관리자는 정책을 설정하지만 개인 근거 본문을 열람하지 않습니다."
      />
      <ErrorBox error={error} />
      {policy ? (
        <section className="card">
          <Field label="근거 보관 활성화">
            <select
              value={String(policy.enabled)}
              onChange={(e) =>
                setPolicy({ ...policy, enabled: e.target.value === "true" })
              }
            >
              <option value="false">비활성화</option>
              <option value="true">활성화</option>
            </select>
          </Field>
          <Field
            label="최대 보존기간 (일)"
            hint="1~3650일. 기간 축소는 기존 사본에도 적용되며 늘려도 만료된 자료가 복구되지 않습니다."
          >
            <input
              type="number"
              min={1}
              max={3650}
              step={1}
              value={policy.retention_days}
              onChange={(e) =>
                setPolicy({ ...policy, retention_days: Number(e.target.value) })
              }
            />
          </Field>
          <p className="notice">
            비활성화하면 신규 저장과 열람을 차단합니다. 사본은 암호화되어 백업에
            포함됩니다. 만료 사본은 시간별 유지관리에서 최대 500개씩 삭제하며,
            원문 보존 의무가 있으면 본문은 숨기고 물리 삭제를 보류합니다.
          </p>
          <div className="modal-actions">
            <Button disabled={busy} onClick={() => void load()}>
              서버 설정 다시 읽기
            </Button>
            <Button
              disabled={busy}
              variant="primary"
              onClick={() => void save()}
            >
              정책 저장
            </Button>
          </div>
        </section>
      ) : (
        !error && <Loading />
      )}
      {policy && (
        <KnowledgePolicyHistory
          kind="evidence"
          version={policy.version}
          onRestored={load}
        />
      )}
    </div>
  );
}
