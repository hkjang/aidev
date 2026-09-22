import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  CheckCircle2,
  RefreshCw,
  ShieldCheck,
  Sparkles,
  Square,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import { Badge, Button, ErrorBox, Field, Modal } from "./ui";
type Row = Record<string, any>;
const stateNames: Record<string, string> = {
  proposed: "실행 전 제안",
  review: "검토 중",
  approved: "조회 등록됨",
  rejected: "반려",
  cancelled: "취소",
};
export function Text2SQLPanel({
  source,
  table,
  onChanged,
}: {
  source: Row;
  table: Row;
  onChanged: () => Promise<void>;
}) {
  const { notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [prompt, setPrompt] = useState(""),
    [output, setOutput] = useState(""),
    [rows, setRows] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [generating, setGenerating] = useState(false),
    [busy, setBusy] = useState(false),
    [approval, setApproval] = useState(false),
    [enabled, setEnabled] = useState(false),
    [confirmation, setConfirmation] = useState("");
  const controller = useRef<AbortController | null>(null),
    generation = useRef(0);
  const selected = rows.find((r) => r.id === params.get("proposal"));
  const load = useCallback(async () => {
    const run = ++generation.current;
    const data = await api<Row>(`/data-sources/${source.id}/ai/proposals`);
    if (run !== generation.current) return;
    setRows(data.proposals);
    setApproval(data.approval_required);
    setEnabled(data.ai_enabled);
  }, [source.id]);
  useEffect(() => {
    setRows([]);
    setOutput("");
    setPrompt("");
    setError("");
    void load().catch((e) => setError(e.message));
    return () => {
      generation.current++;
      controller.current?.abort();
    };
  }, [load, table.schema_name, table.table_name]);
  useEffect(() => {
    if (!rows.some((r) => r.status === "review")) return;
    const timer = setInterval(() => {
      void load()
        .then(() => onChanged())
        .catch((e) => setError(e.message));
    }, 4000);
    return () => clearInterval(timer);
  }, [rows, load, onChanged]);
  useEffect(() => setConfirmation(""), [selected?.id]);
  const select = (id: string | null) =>
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      if (id) next.set("proposal", id);
      else next.delete("proposal");
      return next;
    });
  const generate = async () => {
    setGenerating(true);
    setError("");
    setOutput("");
    const abort = new AbortController();
    controller.current = abort;
    let stored = false;
    try {
      const response = await fetch(
        `/api/v1/data-sources/${source.id}/ai/proposals`,
        {
          method: "POST",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            "X-Madi-Request": "1",
          },
          body: JSON.stringify({
            prompt,
            schema: table.schema_name,
            table: table.table_name,
          }),
          signal: abort.signal,
        },
      );
      if (!response.ok) {
        const v = await response.json();
        throw Error(v.error || "AI 계획을 생성하지 못했습니다");
      }
      if (!response.body) throw Error("스트리밍 응답이 없습니다");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const chunk = await reader.read();
        if (chunk.done) break;
        buffer += decoder.decode(chunk.value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const text = line.slice(5).trim();
          if (!text || text === "[DONE]") continue;
          const value = JSON.parse(text);
          if (value.error) throw Error(value.error);
          if (value.text) setOutput((old) => old + value.text);
          if (value.stored && value.proposal_id) {
            stored = true;
            await load();
            select(value.proposal_id);
          }
        }
      }
      if (!stored)
        throw Error("계획이 완료되지 않아 제안을 저장하지 않았습니다");
      notify("AI 계획을 저장했습니다. 조건과 출처를 확인하세요");
    } catch (e) {
      if ((e as Error).name !== "AbortError") setError((e as Error).message);
      else
        setError("AI 생성을 중지했습니다. 등록된 쿼리는 변경하지 않았습니다.");
    } finally {
      setGenerating(false);
      controller.current = null;
    }
  };
  return (
    <section style={{ marginTop: "1.5rem" }}>
      <div className="card-header">
        <div>
          <h3>
            <Sparkles size={18} /> AI 조회 계획
          </h3>
          <p>
            질문을 검증 가능한 SELECT 계획으로 바꿉니다. 확인이나 승인만으로
            실제 데이터 조회를 실행하지 않습니다.
          </p>
        </div>
        <Button
          disabled={generating || busy}
          onClick={() => void load().catch((e) => setError(e.message))}
        >
          <RefreshCw size={16} />
          이력 새로고침
        </Button>
      </div>
      <ErrorBox error={error} />
      <div className="notice">
        <ShieldCheck size={18} />
        <span>
          모델에는 선택한 테이블의 컬럼 메타데이터와 질문만 보냅니다. DB
          암호·원격 행 데이터는 보내지 않습니다. 출력은 64KB·최대 16,384토큰으로
          제한합니다.
        </span>
      </div>
      {enabled ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void generate();
          }}
        >
          <Field label="AI 데이터 조회 질문">
            <textarea
              rows={3}
              required
              maxLength={16000}
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="금액이 높은 업무 두 건을 이름과 상태와 함께 보고 싶어요."
            />
          </Field>
          <div className="button-row">
            <Button
              variant="primary"
              type="submit"
              disabled={generating || !prompt.trim()}
            >
              <Sparkles size={17} />
              {generating ? "계획 생성 중…" : "조회 계획 제안받기"}
            </Button>
            {generating && (
              <Button type="button" onClick={() => controller.current?.abort()}>
                <Square size={16} />
                생성 중지
              </Button>
            )}
          </div>
        </form>
      ) : (
        <p className="muted">
          데이터 소스 관리자가 설정에서 AI 조회 계획 제안을 허용하면 사용할 수
          있습니다.
        </p>
      )}
      {output && (
        <details open={generating}>
          <summary>모델 스트리밍 응답</summary>
          <pre
            style={{ maxHeight: 260, overflow: "auto", whiteSpace: "pre-wrap" }}
          >
            {output}
          </pre>
        </details>
      )}
      {rows.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>내 제안</th>
                <th>상태</th>
                <th>생성 시각</th>
                <th>확인</th>
              </tr>
            </thead>
            <tbody>
              {rows
                .filter(
                  (r) =>
                    r.plan.schema === table.schema_name &&
                    r.plan.table === table.table_name,
                )
                .map((r) => (
                  <tr key={r.id}>
                    <td>{r.name}</td>
                    <td>
                      <Badge>
                        {!approval && r.status === "review"
                          ? "보류"
                          : stateNames[r.status] || r.status}
                      </Badge>
                      {r.stale && <Badge tone="warning">소스 변경됨</Badge>}
                    </td>
                    <td>{datetime(r.created_at)}</td>
                    <td>
                      <Button onClick={() => select(r.id)}>계획 보기</Button>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}
      {selected && (
        <Modal
          open
          wide
          title={`${selected.name} · 실행 전 계획`}
          onOpenChange={() => {
            if (!busy) select(null);
          }}
        >
          <ErrorBox error={error} />
          <p>{selected.explanation}</p>
          <div className="notice">
            <ShieldCheck size={18} />
            <span>
              이 내용은 실제 데이터 조회 결과가 아닙니다. 아래 계획을 확인하여
              등록한 뒤, 저장 쿼리의 '조회 실행'을 별도로 선택하세요.
            </span>
          </div>
          <h3>메타데이터 출처</h3>
          <ul>
            {selected.citations.map((c: Row) => (
              <li key={c.source_id + "." + c.schema + "." + c.table}>
                <Link
                  to={`/app/data-sources?source=${c.source_id}&table=${encodeURIComponent(c.schema + "." + c.table)}`}
                >
                  {c.source_name} · {c.schema}.{c.table}
                </Link>
              </li>
            ))}
          </ul>
          <dl>
            <dt>대상 테이블</dt>
            <dd>
              {selected.plan.schema}.{selected.plan.table}
            </dd>
            <dt>조회 컬럼</dt>
            <dd>{selected.plan.columns.join(", ")}</dd>
            <dt>조회 한도</dt>
            <dd>{selected.plan.limit}행</dd>
          </dl>
          <details open>
            <summary>검증된 조회 계획 JSON</summary>
            <pre style={{ maxHeight: 360, overflow: "auto" }}>
              {JSON.stringify(selected.plan, null, 2)}
            </pre>
          </details>
          {selected.stale && (
            <ErrorBox error="데이터 소스 설정이 변경되었습니다. 현재 테이블에서 새 계획을 생성하세요." />
          )}
          {selected.status === "approved" && (
            <div className="notice">
              <CheckCircle2 size={18} />
              <span>
                확인한 계획이 저장 쿼리로 등록되었습니다. 이 창을 닫고 등록된
                조회 목록에서 실행할 수 있습니다.
              </span>
            </div>
          )}
          {approval && selected.approval_id && (
            <Link
              className="button"
              to={`/app/approvals?request=${selected.approval_id}`}
            >
              검토 요청과 승인 이력 보기
            </Link>
          )}
          {enabled &&
            !selected.stale &&
            ["proposed", "rejected", "cancelled"].includes(selected.status) && (
              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  setBusy(true);
                  setError("");
                  try {
                    const response = await api<Row>(
                      `/data-sources/${source.id}/ai/proposals/${selected.id}/confirm`,
                      "POST",
                      { confirmation, version: selected.version },
                    );
                    await load();
                    await onChanged();
                    notify(
                      response.approval_required
                        ? "확인한 계획의 검토를 요청했습니다"
                        : "확인한 조회를 등록했습니다",
                    );
                    setConfirmation("");
                  } catch (e) {
                    setError((e as Error).message);
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                <Field label="SQL 계획 확인 (CREATE QUERY 입력)">
                  <input
                    autoComplete="off"
                    required
                    value={confirmation}
                    onChange={(e) => setConfirmation(e.target.value)}
                    placeholder="CREATE QUERY"
                  />
                </Field>
                {approval && (
                  <p className="muted">
                    관리자 설정에 따라 다른 검토 담당자의 승인이 필요하며 자기
                    승인은 허용되지 않습니다.
                  </p>
                )}
                <Button
                  type="submit"
                  variant="primary"
                  disabled={busy || confirmation !== "CREATE QUERY"}
                >
                  {busy
                    ? "처리 중…"
                    : approval
                      ? "확인한 계획 검토 요청"
                      : "확인한 조회 등록"}
                </Button>
              </form>
            )}
        </Modal>
      )}
    </section>
  );
}
