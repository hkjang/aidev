import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  History,
  Settings,
  Sparkles,
  Trash2,
  RefreshCw,
  Square,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
import { AISaveDraft } from "./AIActions";
import "./search-history.css";

type Entry = {
  id: string;
  revision: number;
  query: string;
  filters: Record<string, string>;
  searches: number;
  zero_results: number;
  last_count: number;
  last_seen: string;
  expires_at: string;
};
type Preferences = {
  enabled: boolean;
  retention_days: number;
  revision: number;
};
type Gap = {
  title: string;
  reason: string;
  outline: string;
  references: number[];
};
type Proposal = { notice: string; suggestions: Gap[]; sources: Entry[] };
const filterLabels: Record<string, string> = {
  type: "검색 종류",
  space_id: "공간 ID",
  author_id: "소유자 ID",
  tag: "태그",
  status: "문서 상태",
  from: "시작일",
  to: "종료일",
  has_attachment: "첨부 여부",
  sort: "정렬",
};
const filterValues: Record<string, Record<string, string>> = {
  type: {
    document: "문서",
    block: "블록",
    code: "코드",
    task: "할 일",
    attachment: "파일",
    comment: "댓글",
    user: "사용자",
    tag: "태그",
    database: "데이터베이스",
    row: "데이터 행",
    ai: "AI 대화",
  },
  status: {
    draft: "초안",
    review: "검토 중",
    published: "게시됨",
    stale: "검토 필요",
    archived: "보관됨",
    rejected: "반려됨",
  },
  sort: {
    relevance: "관련도",
    newest: "최근 수정",
    oldest: "오래된 수정",
    title: "제목",
  },
  has_attachment: {
    "1": "첨부 있음",
    true: "첨부 있음",
    "0": "첨부 없음",
    false: "첨부 없음",
  },
};
export default function SearchHistoryPage() {
  const { workspace, user, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [preferences, setPreferences] = useState<Preferences | null>(null),
    [settingsOpen, setSettingsOpen] = useState(false),
    [enabled, setEnabled] = useState(false),
    [days, setDays] = useState(30),
    [items, setItems] = useState<Entry[]>([]),
    [more, setMore] = useState(false),
    [selected, setSelected] = useState<Entry[]>([]),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [refresh, setRefresh] = useState(0),
    [clear, setClear] = useState(false);
  const [analyze, setAnalyze] = useState(false),
    [consent, setConsent] = useState(false),
    [running, setRunning] = useState(false),
    [raw, setRaw] = useState(""),
    [proposal, setProposal] = useState<Proposal | null>(null),
    [analysisError, setAnalysisError] = useState("");
  const generation = useRef(0),
    controller = useRef<AbortController | null>(null);
  const zero = params.get("zero") !== "0",
    offset = Math.max(0, Number(params.get("offset")) || 0);
  useEffect(() => {
    const n = ++generation.current;
    setLoading(true);
    setItems([]);
    setSelected([]);
    setError(null);
    controller.current?.abort();
    controller.current = null;
    setAnalyze(false);
    setClear(false);
    setSettingsOpen(false);
    setBusy(false);
    setPreferences(null);
    setProposal(null);
    setRunning(false);
    if (!workspace) {
      setLoading(false);
      return;
    }
    void Promise.all([
      api<Preferences>("/profile/search-history/settings"),
      api<{ items: Entry[]; has_more: boolean }>(
        `/search/history?workspace_id=${workspace.id}&zero=${zero ? "1" : "0"}&offset=${offset}`,
      ),
    ])
      .then(([p, v]) => {
        if (generation.current !== n) return;
        setPreferences(p);
        setItems(v.items);
        setMore(v.has_more);
      })
      .catch((e) => {
        if (generation.current === n) setError(e);
      })
      .finally(() => {
        if (generation.current === n) setLoading(false);
      });
    return () => {
      generation.current++;
      controller.current?.abort();
    };
  }, [workspace?.id, user.id, zero, offset, refresh]);
  function closeAnalysis(open: boolean) {
    setAnalyze(open);
    if (!open) {
      controller.current?.abort();
      controller.current = null;
      setRunning(false);
      setRaw("");
      setProposal(null);
      setConsent(false);
      setAnalysisError("");
    }
  }
  async function generate() {
    if (!workspace || !consent || !selected.length || running) return;
    const request = new AbortController();
    controller.current = request;
    setRunning(true);
    setRaw("");
    setProposal(null);
    setAnalysisError("");
    try {
      const r = await fetch("/api/v1/search/history/gaps", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
        body: JSON.stringify({
          workspace_id: workspace.id,
          entries: selected.map(({ id, revision }) => ({ id, revision })),
          consent: true,
        }),
        signal: request.signal,
      });
      if (!r.ok) {
        const e = await r
          .json()
          .catch(() => ({ error: "분석 요청을 처리하지 못했습니다" }));
        throw new Error(e.error);
      }
      if (!r.body) throw new Error("스트리밍 응답이 없습니다");
      const reader = r.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        done = false,
        result: Proposal | null = null;
      while (true) {
        const read = await reader.read();
        if (controller.current !== request || request.signal.aborted) return;
        if (read.done) break;
        buffer += decoder.decode(read.value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const raw = line.slice(5).trim();
          if (!raw) continue;
          if (raw === "[DONE]") {
            done = true;
            continue;
          }
          const event = JSON.parse(raw);
          if (event.retract) {
            result = null;
            setRaw("");
            setProposal(null);
          }
          if (event.error) throw new Error(event.error);
          if (event.text) setRaw((v) => v + event.text);
          if (event.proposal) result = event.proposal;
        }
      }
      if (!done || !result)
        throw new Error("완료되지 않은 제안은 저장할 수 없습니다");
      setProposal(result);
    } catch (e) {
      if (controller.current === request && !request.signal.aborted) {
        setAnalysisError((e as Error).message);
        setProposal(null);
      }
    } finally {
      if (controller.current === request) {
        controller.current = null;
        setRunning(false);
      }
    }
  }
  if (!workspace)
    return (
      <div className="page">
        <Empty title="워크스페이스를 선택하세요" />
      </div>
    );
  return (
    <div className="page search-history-page">
      <PageHeading
        eyebrow="PERSONAL KNOWLEDGE"
        title="내 검색 기록"
        description="개인 동의로 저장한 검색어만 보관합니다. 결과 문서 본문과 다른 사용자의 검색 기록은 포함하지 않습니다."
        actions={
          <>
            <Button onClick={() => setRefresh((v) => v + 1)} disabled={loading}>
              <RefreshCw size={17} />
              새로고침
            </Button>
            <Button
              onClick={() => {
                if (preferences) {
                  setEnabled(preferences.enabled);
                  setDays(preferences.retention_days);
                  setSettingsOpen(true);
                }
              }}
              disabled={!preferences}
            >
              <Settings size={17} />
              기록 설정
            </Button>
          </>
        }
      />
      <ErrorBox error={error} />
      {preferences && (
        <div className="notice">
          <History size={20} />
          <span>
            {preferences.enabled
              ? `검색어 저장 중 · ${preferences.retention_days}일 보존 · 개인 전체 최대 2,000건`
              : "검색어 저장 꺼짐 · 기본값입니다. 기록 설정에서 동의해야 이후 검색부터 저장합니다."}
            <br />
            저장을 꺼도 기존 기록은 보존 기간까지 남습니다. 즉시 지우려면 아래
            삭제를 사용하세요.
          </span>
        </div>
      )}
      <div className="search-history-toolbar">
        <label className="check-label">
          <input
            type="checkbox"
            checked={zero}
            onChange={(e) => setParams({ zero: e.target.checked ? "1" : "0" })}
          />
          결과가 없었던 검색만
        </label>
        <Button disabled={!items.length || busy} onClick={() => setClear(true)}>
          <Trash2 size={17} />이 워크스페이스 기록 삭제
        </Button>
        <Button
          variant="primary"
          disabled={!preferences?.enabled || !selected.length}
          onClick={() => {
            setAnalyze(true);
            setConsent(false);
            setProposal(null);
            setRaw("");
            setAnalysisError("");
          }}
        >
          <Sparkles size={17} />
          선택 {selected.length}/30건 지식 보완 제안
        </Button>
      </div>
      {loading ? (
        <Loading />
      ) : !items.length ? (
        <Empty
          title="저장된 검색 기록이 없습니다"
          text="저장 동의 후 검색을 실행해 보세요. 필터를 해제하면 성공한 검색도 확인할 수 있습니다."
        />
      ) : (
        <div className="search-history-list">
          {items.map((entry) => (
            <article className="panel padded" key={entry.id}>
              <label className="check-label">
                <input
                  type="checkbox"
                  aria-label={`${entry.query} 분석에 선택`}
                  checked={selected.some((v) => v.id === entry.id)}
                  disabled={
                    !entry.zero_results ||
                    (selected.length >= 30 &&
                      !selected.some((v) => v.id === entry.id))
                  }
                  onChange={(e) =>
                    setSelected((old) =>
                      e.target.checked
                        ? [...old, entry]
                        : old.filter((v) => v.id !== entry.id),
                    )
                  }
                />
                <strong>{entry.query}</strong>
              </label>
              <p>
                검색 {entry.searches}회 · 결과 없음 {entry.zero_results}회 ·
                마지막 표시 결과 {entry.last_count}개
              </p>
              <small>
                최근 {datetime(entry.last_seen)} · 자동 삭제{" "}
                {datetime(entry.expires_at)}
              </small>
              {Object.keys(entry.filters).length > 0 && (
                <details>
                  <summary>당시 검색 필터</summary>
                  <dl>
                    {Object.entries(entry.filters).map(([k, v]) => (
                      <div key={k}>
                        <dt>{filterLabels[k] || k}</dt>
                        <dd>{filterValues[k]?.[v] || v}</dd>
                      </div>
                    ))}
                  </dl>
                </details>
              )}
              <Link
                to={`/app/search?${new URLSearchParams({ ...entry.filters, q: entry.query })}`}
              >
                현재 권한으로 다시 검색 →
              </Link>
            </article>
          ))}
        </div>
      )}
      {(offset > 0 || more) && (
        <div className="modal-actions">
          <Button
            disabled={!offset}
            onClick={() =>
              setParams({
                zero: zero ? "1" : "0",
                offset: String(Math.max(0, offset - 40)),
              })
            }
          >
            이전
          </Button>
          <Button
            disabled={!more}
            onClick={() =>
              setParams({ zero: zero ? "1" : "0", offset: String(offset + 40) })
            }
          >
            다음
          </Button>
        </div>
      )}
      <Modal
        open={settingsOpen}
        onOpenChange={(v) => {
          if (!busy) setSettingsOpen(v);
        }}
        title="개인 검색 기록 설정"
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!preferences || busy) return;
            const request = generation.current;
            setBusy(true);
            try {
              const value = await api<Preferences>(
                "/profile/search-history/settings",
                "PUT",
                {
                  enabled,
                  retention_days: days,
                  revision: preferences.revision,
                  consent: enabled,
                },
              );
              if (request !== generation.current) return;
              setPreferences(value);
              setSettingsOpen(false);
              setRefresh((v) => v + 1);
              notify("검색 기록 설정을 저장했습니다");
            } catch (e) {
              if (request === generation.current) setError(e);
            } finally {
              if (request === generation.current) setBusy(false);
            }
          }}
        >
          <p>
            검색어·검색 필터·횟수·결과 수를 개인 기록으로 PostgreSQL과 백업에
            보관합니다. 서비스 관리자 API에는 공개하지 않지만 DB·백업 운영자에
            대한 종단간 암호화는 아닙니다. 입력한 개인정보는 서비스 정보보호
            정책을 적용합니다. API 키·플러그인 검색은 기록하지 않습니다.
          </p>
          <label className="check-label">
            <input
              type="checkbox"
              checked={enabled}
              disabled={busy}
              onChange={(e) => setEnabled(e.target.checked)}
            />
            이후 검색어를 내 개인 기록에 저장하는 데 동의합니다.
          </label>
          <Field label="보존 기간 (일)">
            <input
              type="number"
              required
              min={7}
              max={365}
              step={1}
              value={days}
              disabled={busy}
              onChange={(e) => setDays(Number(e.target.value))}
            />
          </Field>
          <p className="muted">
            기간을 줄이면 기존 기록에도 짧은 기한을 적용합니다. 기간을 늘려도
            기존 기록의 삭제일을 자동 연장하지 않습니다. 백업 복원 후 저장
            동의는 꺼집니다.
          </p>
          <ErrorBox error={error} />
          <div className="modal-actions">
            <Button variant="primary" disabled={busy}>
              설정 저장
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={clear}
        onOpenChange={(v) => {
          if (!busy) setClear(v);
        }}
        title="개인 검색 기록 삭제"
      >
        <p>
          현재 워크스페이스의 내 검색어 기록을 모두 삭제합니다. 앱에서 되돌릴 수
          없으며 기존 백업 사본은 운영 보존 정책에 따릅니다.
        </p>
        <div className="modal-actions">
          <Button
            variant="danger"
            disabled={busy}
            onClick={async () => {
              if (busy) return;
              const request = generation.current;
              setBusy(true);
              try {
                await api("/search/history", "DELETE", {
                  workspace_id: workspace.id,
                  confirmation: "DELETE_ALL",
                });
                if (request !== generation.current) return;
                setClear(false);
                setRefresh((v) => v + 1);
                notify("개인 검색 기록을 삭제했습니다");
              } catch (e) {
                if (request === generation.current) setError(e);
              } finally {
                if (request === generation.current) setBusy(false);
              }
            }}
          >
            확인하고 모두 삭제
          </Button>
        </div>
      </Modal>
      <Modal
        open={analyze}
        onOpenChange={closeAnalysis}
        title="검색 실패에서 지식 보완하기"
        wide
      >
        <p>
          선택한 개인 검색어와 필터·실패 횟수 {selected.length}건만 분석합니다.
          검색 실패는 필터·권한·용어 문제일 수도 있으므로 실제 문서 부재나 조직
          전체의 지식 부족을 뜻하지 않습니다.
        </p>
        <label className="check-label">
          <input
            type="checkbox"
            checked={consent}
            disabled={running}
            onChange={(e) => setConsent(e.target.checked)}
          />
          선택한 개인 검색 기록을 AI 공급자에 전송하는 데 동의합니다.
        </label>
        <ErrorBox error={analysisError} />
        <div className="modal-actions">
          {running ? (
            <Button
              onClick={() => {
                controller.current?.abort();
                controller.current = null;
                setRunning(false);
                setRaw("");
                setProposal(null);
              }}
            >
              <Square size={16} />
              분석 중단
            </Button>
          ) : (
            <Button
              variant="primary"
              disabled={!consent}
              onClick={() => void generate()}
            >
              <Sparkles size={17} />
              보완할 지식 제안
            </Button>
          )}
        </div>
        {raw && !proposal && (
          <details>
            <summary>AI 스트리밍 응답</summary>
            <pre className="search-history-stream">{raw}</pre>
          </details>
        )}
        {proposal && (
          <section aria-label="지식 보완 제안">
            <p>{proposal.notice}</p>
            {proposal.suggestions.map((item, i) => (
              <article className="panel padded" key={i}>
                <h3>{item.title}</h3>
                <p>{item.reason}</p>
                <p>
                  근거 검색 기록:{" "}
                  {item.references
                    .map(
                      (n) =>
                        `[${n}] ${proposal.sources[n - 1]?.query || "기록 확인 필요"}`,
                    )
                    .join(" · ")}
                </p>
                <pre>{item.outline}</pre>
                <AISaveDraft
                  onSaved={() => closeAnalysis(false)}
                  answer={`# ${item.title}\n\n${item.outline}\n\n## 제안의 범위\n\n${item.reason}\n\n선택한 개인 검색 실패 이력 기반 초안입니다. 실제 자료 부재를 확정하지 않으며 운영 사실·담당자·기한은 별도 확인이 필요합니다.`}
                  question={item.title}
                  sources={[]}
                />
              </article>
            ))}
          </section>
        )}
      </Modal>
    </div>
  );
}
