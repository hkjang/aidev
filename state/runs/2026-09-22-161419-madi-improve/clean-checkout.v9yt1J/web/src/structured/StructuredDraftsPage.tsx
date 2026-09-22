import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Database, FileText, RefreshCw } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, Empty, Field, Loading, PageHeading } from "../ui";
import { RecoveryNotice } from "../review/ChangeReview";
import StructuredComposer from "./StructuredComposer";
import StructuredReview from "./StructuredReview";
import type { StructuredListItem } from "./types";
import "./style.css";
export default function StructuredDraftsPage() {
  const { user, workspace, documents } = useApp(),
    [params, setParams] = useSearchParams();
  const id = params.get("id") || "",
    doc = params.get("document_id") || "",
    db = params.get("database_id") || "";
  const [databases, setDatabases] = useState<
      { id: string; name: string; can_write?: boolean }[]
    >([]),
    [items, setItems] = useState<StructuredListItem[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState<unknown>(null),
    [after, setAfter] = useState(""),
    [next, setNext] = useState(""),
    [retry, setRetry] = useState(0);
  const scope = `${user.id}:${workspace?.id}:${id}:${after}`,
    current = useRef(scope),
    loaded = useRef("");
  current.current = scope;
  useEffect(() => {
    setAfter("");
  }, [workspace?.id, user.id]);
  useEffect(() => {
    const controller = new AbortController();
    let pending = false;
    setItems([]);
    setDatabases([]);
    setError(null);
    setLoading(true);
    const valid = () => !controller.signal.aborted && current.current === scope;
    const load = async () => {
      if (!workspace || pending) return;
      pending = true;
      try {
        const [dbs, list] = await Promise.all([
          api<typeof databases>(
            `/databases?workspace_id=${workspace.id}`,
            "GET",
            undefined,
            { signal: controller.signal },
          ),
          api<{ items: StructuredListItem[]; next_after: string }>(
            `/knowledge/structured-drafts?workspace_id=${workspace.id}${after ? `&after=${after}` : ""}`,
            "GET",
            undefined,
            { signal: controller.signal },
          ),
        ]);
        if (valid()) {
          loaded.current = scope;
          setDatabases(dbs);
          setItems(list.items);
          setNext(list.next_after);
          setError(null);
        }
      } catch (e) {
        if (valid()) {
          setItems([]);
          setDatabases([]);
          setError(e);
        }
      } finally {
        pending = false;
        if (valid()) setLoading(false);
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      controller.abort();
      clearInterval(timer);
    };
  }, [scope, retry, workspace]);
  const selectedScope = `${user.id}:${workspace?.id}:${doc}:${db}`;
  return (
    <div className="page structured-page">
      <PageHeading
        title="문서에서 데이터 정리"
        description="원문의 선택 구간을 일반 속성 값으로 제안받고, 근거와 공유 대상을 사람이 확인한 뒤 새 행을 만듭니다."
        actions={
          <Button onClick={() => setRetry((v) => v + 1)}>
            <RefreshCw size={18} />
            목록 다시 읽기
          </Button>
        }
      />
      <div className="notice structured-notice">
        <div>
          <strong>
            제안 생성 → 개인 검토함 → 근거 수정·미리보기 → 별도 공유
          </strong>
          <p>
            자동으로 행을 저장하지 않습니다. 원문 비공개 권한은 데이터베이스에
            복사한 값에 상속되지 않으므로 마지막 단계에서 공유 범위를
            확인합니다.
          </p>
        </div>
      </div>
      <RecoveryNotice error={error} onRetry={() => setRetry((v) => v + 1)} />
      {id ? (
        <>
          <Button onClick={() => setParams({})}>개인 검토함 목록</Button>
          {workspace && (
            <StructuredReview
              key={`${user.id}:${workspace.id}:${id}`}
              id={id}
              workspaceId={workspace.id}
              onDeleted={() => setParams({})}
            />
          )}
        </>
      ) : (
        <>
          <section className="panel structured-section">
            <h2>원문과 데이터베이스 선택</h2>
            <div className="structured-pickers">
              <Field label="구조화 원문 문서">
                <select
                  value={doc}
                  disabled={loading}
                  onChange={(e) =>
                    setParams({
                      ...Object.fromEntries(params),
                      document_id: e.target.value,
                    })
                  }
                >
                  <option value="">문서를 선택하세요</option>
                  {documents.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="값을 담을 데이터베이스">
                <select
                  value={db}
                  disabled={loading}
                  onChange={(e) =>
                    setParams({
                      ...Object.fromEntries(params),
                      database_id: e.target.value,
                    })
                  }
                >
                  <option value="">데이터베이스를 선택하세요</option>
                  {(loaded.current === scope ? databases : [])
                    .filter((d) => d.can_write !== false)
                    .map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.name}
                      </option>
                    ))}
                </select>
              </Field>
            </div>
            <p className="muted">
              같은 워크스페이스의 현재 읽을 수 있는 문서와 작성 가능한
              데이터베이스만 사용합니다. 계산·관계·사용자 속성은 AI 구조화에
              포함하지 않습니다.
            </p>
          </section>
          {workspace && doc && db && (
            <StructuredComposer
              key={selectedScope}
              documentId={doc}
              databaseId={db}
              workspaceId={workspace.id}
              onSaved={(id) => setParams({ id })}
            />
          )}
          <section className="panel structured-section">
            <div className="structured-section-heading">
              <h2>내 검토함</h2>
              <span className="badge">본인만 보기 · 미반영 초안 24시간</span>
            </div>
            {loading ? (
              <Loading />
            ) : loaded.current === scope && items.length ? (
              <div className="structured-draft-list">
                {items.map((item) => (
                  <button
                    className="structured-draft-card"
                    key={item.id}
                    onClick={() => setParams({ id: item.id })}
                  >
                    <FileText size={22} />
                    <span>
                      <strong>{item.title}</strong>
                      <span>
                        <Database size={16} />
                        {item.database_name}
                      </span>
                      <small>
                        {item.state === "committed"
                          ? "새 행 생성 완료"
                          : "개인 검토 대기"}{" "}
                        ·{" "}
                        {item.fresh
                          ? "현재 기준과 일치"
                          : "기준 변경·이력 확인"}{" "}
                        · {datetime(item.created_at)}
                      </small>
                    </span>
                  </button>
                ))}
              </div>
            ) : (
              <Empty
                title="보관한 개인 구조화 초안이 없습니다"
                text="AI 제안을 확인하고 개인 보관에 동의하면 이곳에서 검토할 수 있습니다."
              />
            )}
            <div className="button-row">
              {after && <Button onClick={() => setAfter("")}>처음 목록</Button>}
              {next && (
                <Button onClick={() => setAfter(next)}>다음 목록</Button>
              )}
            </div>
          </section>
        </>
      )}
    </div>
  );
}
