import "./channel-settings.css";
import { useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import {
  Database,
  FileText,
  Network,
  Plus,
  Search,
  ShieldCheck,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
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
export const entityTypes = [
  ["person", "사람"],
  ["team", "팀"],
  ["project", "프로젝트"],
  ["system", "시스템"],
  ["server", "서버"],
  ["application", "애플리케이션"],
  ["database", "데이터베이스"],
  ["technology", "기술"],
  ["vendor", "공급사"],
  ["model", "모델"],
  ["policy", "정책"],
];
function sourceURL(row: Row) {
  return `/app/data-sources?source=${row.source_id}&table=${encodeURIComponent(`${row.schema_name || row.schema}.${row.table_name || row.table}`)}`;
}
export function EnterprisePage() {
  const { workspace } = useApp();
  const [params, setParams] = useSearchParams();
  const [query, setQuery] = useState(params.get("q") || ""),
    [data, setData] = useState<Row | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const generation = useRef(0);
  const kind = params.get("kind") || "all";
  useEffect(() => setQuery(params.get("q") || ""), [params.get("q")]);
  useEffect(() => {
    if (!workspace?.id) return;
    const run = ++generation.current;
    setLoading(true);
    api<Row>(
      `/enterprise/search?workspace_id=${workspace.id}&q=${encodeURIComponent(params.get("q") || "")}`,
    )
      .then((v) => {
        if (run === generation.current) {
          setData(v);
          setError("");
        }
      })
      .catch((e) => {
        if (run === generation.current) setError(e.message);
      })
      .finally(() => {
        if (run === generation.current) setLoading(false);
      });
    return () => {
      generation.current++;
    };
  }, [workspace?.id, params.get("q")]);
  const show = (k: string) => kind === "all" || kind === k;
  return (
    <>
      <PageHeading
        eyebrow="ENTERPRISE KNOWLEDGE CATALOG"
        title="지식 소스 탐색"
        description="문서, 연결된 원격 자료, 테이블 설명, 등록된 조회를 같은 권한 경계 안에서 찾습니다."
        actions={
          <>
            <Link className="button" to="/app/entities">
              <Network size={17} />
              엔터티
            </Link>
            <Link className="button" to="/app/data-sources">
              데이터 소스
            </Link>
          </>
        }
      />
      <ErrorBox error={error} />
      <form
        className="search-bar"
        onSubmit={(e) => {
          e.preventDefault();
          setParams({ q: query, kind });
        }}
      >
        <Search size={20} />
        <input
          aria-label="지식 소스 검색"
          placeholder="운영 정책, 고객 테이블, 프로젝트 이름…"
          maxLength={500}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Button type="submit" variant="primary">
          검색
        </Button>
      </form>
      <div className="button-row" style={{ margin: "1rem 0" }}>
        {[
          ["all", "전체"],
          ["documents", "문서·원격 자료"],
          ["tables", "외부 테이블"],
          ["queries", "저장 쿼리"],
        ].map(([k, label]) => (
          <Button
            key={k}
            variant={kind === k ? "primary" : ""}
            onClick={() => setParams({ q: params.get("q") || "", kind: k })}
          >
            {label}
          </Button>
        ))}
      </div>
      <div className="notice">
        <ShieldCheck size={18} />
        <span>
          검색 대상은 현재 접근 가능한 자료입니다. 원격 행 데이터는 검색 색인에
          복사하지 않으며, 각 범주에서 최대 100개를 표시합니다.
        </span>
      </div>
      {loading ? (
        <Loading />
      ) : (
        data && (
          <>
            {show("documents") && (
              <section className="card">
                <div className="card-header">
                  <h2>문서와 연결된 자료</h2>
                  <Badge>{data.documents.length}</Badge>
                </div>
                {!data.documents.length ? (
                  <Empty
                    title="일치하는 문서가 없습니다"
                    text="다른 이름이나 용어로 검색해 보세요."
                  />
                ) : (
                  <div className="document-list">
                    {data.documents.map((r: Row) => (
                      <Link
                        key={r.id}
                        className="document-row"
                        to={
                          r.kind === "entity"
                            ? `/app/entities/${r.id}`
                            : `/app/documents/${r.id}`
                        }
                      >
                        <FileText size={21} />
                        <div>
                          <strong>{r.title}</strong>
                          <p className="muted">
                            {r.connector
                              ? `${r.connector} · ${r.connector_kind}`
                              : r.kind === "entity"
                                ? "엔터티 지식"
                                : "워크스페이스 문서"}{" "}
                            · {datetime(r.updated_at)}
                          </p>
                        </div>
                      </Link>
                    ))}
                  </div>
                )}
              </section>
            )}
            {show("tables") && (
              <section className="card">
                <div className="card-header">
                  <h2>외부 테이블</h2>
                  <Badge>{data.tables.length}</Badge>
                </div>
                {!data.tables.length ? (
                  <Empty
                    title="일치하는 테이블이 없습니다"
                    text="데이터 소스에서 메타데이터를 검사하고 테이블 설명을 등록하세요."
                  />
                ) : (
                  <div className="document-list">
                    {data.tables.map((r: Row) => (
                      <Link
                        className="document-row"
                        key={
                          r.source_id + ":" + r.schema_name + "." + r.table_name
                        }
                        to={sourceURL(r)}
                      >
                        <Database size={21} />
                        <div>
                          <strong>
                            {r.schema_name}.{r.table_name}
                          </strong>
                          <p className="muted">
                            {r.source_name} · {r.engine} · {r.column_count}개
                            컬럼
                          </p>
                          <p>{r.description}</p>
                        </div>
                      </Link>
                    ))}
                  </div>
                )}
              </section>
            )}
            {show("queries") && (
              <section className="card">
                <div className="card-header">
                  <h2>등록된 읽기 전용 쿼리</h2>
                  <Badge>{data.queries.length}</Badge>
                </div>
                {!data.queries.length ? (
                  <Empty
                    title="일치하는 쿼리가 없습니다"
                    text="테이블 페이지에서 제한된 SELECT 계획을 등록할 수 있습니다."
                  />
                ) : (
                  <div className="document-list">
                    {data.queries.map((r: Row) => (
                      <Link
                        className="document-row"
                        key={r.id}
                        to={sourceURL(r)}
                      >
                        <Search size={21} />
                        <div>
                          <strong>{r.name}</strong>
                          <p className="muted">
                            {r.source_name} · {r.schema}.{r.table} ·{" "}
                            {r.enabled ? "사용 중" : "중지"}
                          </p>
                        </div>
                      </Link>
                    ))}
                  </div>
                )}
              </section>
            )}
          </>
        )
      )}
    </>
  );
}
export function EntityListPage() {
  const { workspace, notify, reload } = useApp();
  const [params, setParams] = useSearchParams();
  const [rows, setRows] = useState<Row[]>([]),
    [spaces, setSpaces] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [draft, setDraft] = useState<Row | null>(null);
  const generation = useRef(0);
  const writable = ["owner", "admin", "editor"].includes(workspace?.role || "");
  const load = async () => {
    if (!workspace?.id) return;
    const run = ++generation.current;
    try {
      const [r, s] = await Promise.all([
        api<Row[]>(`/enterprise/entities?workspace_id=${workspace.id}`),
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
  };
  useEffect(() => {
    setLoading(true);
    setRows([]);
    setDraft(null);
    void load();
    return () => {
      generation.current++;
    };
  }, [workspace?.id]);
  const visible = rows.filter(
    (r) =>
      (!params.get("type") || r.entity_type === params.get("type")) &&
      r.title.toLowerCase().includes((params.get("q") || "").toLowerCase()),
  );
  return (
    <>
      <PageHeading
        eyebrow="PEOPLE · SYSTEMS · KNOWLEDGE"
        title="엔터티"
        description="사람, 팀, 기술, 시스템을 중심으로 문서와 데이터 소스의 의미를 연결합니다."
        actions={
          <>
            <Link className="button" to="/app/enterprise">
              지식 소스 탐색
            </Link>
            {writable && (
              <Button
                variant="primary"
                onClick={() =>
                  setDraft({
                    title: "",
                    entity_type: "technology",
                    description: "",
                    space_id: "",
                    visibility: "workspace",
                  })
                }
              >
                <Plus size={17} />
                엔터티 만들기
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      <div className="form-grid">
        <Field label="엔터티 이름 검색">
          <input
            value={params.get("q") || ""}
            onChange={(e) =>
              setParams({ q: e.target.value, type: params.get("type") || "" })
            }
          />
        </Field>
        <Field label="엔터티 종류 필터">
          <select
            value={params.get("type") || ""}
            onChange={(e) =>
              setParams({ q: params.get("q") || "", type: e.target.value })
            }
          >
            <option value="">모든 종류</option>
            {entityTypes.map(([k, l]) => (
              <option key={k} value={k}>
                {l}
              </option>
            ))}
          </select>
        </Field>
      </div>
      {loading ? (
        <Loading />
      ) : visible.length ? (
        <div className="document-grid">
          {visible.map((r) => (
            <Link
              className="card document-card"
              key={r.id}
              to={`/app/entities/${r.id}`}
            >
              <Network size={25} />
              <Badge>
                {entityTypes.find((x) => x[0] === r.entity_type)?.[1] || "지식"}
              </Badge>
              <h2>{r.title}</h2>
              <p className="muted">{datetime(r.updated_at)}</p>
            </Link>
          ))}
        </div>
      ) : (
        <Empty
          title="연결의 중심이 될 지식을 등록하세요"
          text="문서에서 [[엔터티 이름]]을 참조하거나 SQL 테이블의 관련 문서로 연결하면 관계를 함께 볼 수 있습니다."
        />
      )}
      {draft && (
        <Modal
          open
          title="엔터티 만들기"
          onOpenChange={() => {
            if (!busy) setDraft(null);
          }}
        >
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api("/enterprise/entities", "POST", {
                  ...draft,
                  workspace_id: workspace!.id,
                });
                setDraft(null);
                await load();
                await reload();
                notify("엔터티 문서를 만들었습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <Field label="엔터티 이름">
              <input
                required
                maxLength={300}
                value={draft.title}
                onChange={(e) => setDraft({ ...draft, title: e.target.value })}
              />
            </Field>
            <Field label="엔터티 종류">
              <select
                value={draft.entity_type}
                onChange={(e) =>
                  setDraft({ ...draft, entity_type: e.target.value })
                }
              >
                {entityTypes.map(([k, l]) => (
                  <option key={k} value={k}>
                    {l}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="설명">
              <textarea
                rows={5}
                maxLength={20000}
                value={draft.description}
                onChange={(e) =>
                  setDraft({ ...draft, description: e.target.value })
                }
              />
            </Field>
            <Field label="대상 공간">
              <select
                value={draft.space_id}
                onChange={(e) =>
                  setDraft({ ...draft, space_id: e.target.value })
                }
              >
                <option value="">워크스페이스 루트</option>
                {spaces.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="공개 범위">
              <select
                value={draft.visibility}
                onChange={(e) =>
                  setDraft({ ...draft, visibility: e.target.value })
                }
              >
                <option value="workspace">대상 공간 사용자</option>
                <option value="private">나만 보기</option>
              </select>
            </Field>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setDraft(null)}
              >
                취소
              </Button>
              <Button type="submit" disabled={busy} variant="primary">
                엔터티 생성
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
export function EntityPage() {
  const { id } = useParams();
  const { notify } = useApp();
  const [data, setData] = useState<Row | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    setData(null);
    api<Row>(`/enterprise/entities/${id}`)
      .then((v) => {
        if (active) {
          setData(v);
          setError("");
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [id]);
  return (
    <>
      <ErrorBox error={error} />
      {!data ? (
        error ? (
          <Empty
            title="엔터티를 불러올 수 없습니다"
            text="주소와 현재 문서 접근 권한을 확인하세요."
          />
        ) : (
          <Loading />
        )
      ) : (
        <>
          <PageHeading
            eyebrow="ENTITY KNOWLEDGE PROFILE"
            title={data.document.title}
            description="문서 링크와 데이터 소스의 명시적인 연결을 모은 지식 프로필입니다."
            actions={
              <>
                <Link className="button" to="/app/entities">
                  모든 엔터티
                </Link>
                <Link className="button" to={`/app/documents/${id}`}>
                  {data.document.can_write ? "문서 편집" : "원문 보기"}
                </Link>
              </>
            }
          />
          <section className="card">
            <div className="card-header">
              <div>
                <Badge>
                  {entityTypes.find(
                    (x) => x[0] === data.metadata.system_metadata?.entity_type,
                  )?.[1] || "지식"}
                </Badge>
                <p className="muted">
                  담당자 {data.metadata.owner_name} · 문서 버전{" "}
                  {data.document.version}
                </p>
              </div>
              {data.metadata.can_manage && (
                <Field label="엔터티 종류 변경">
                  <select
                    disabled={busy}
                    value={
                      data.metadata.system_metadata?.entity_type || "technology"
                    }
                    onChange={async (e) => {
                      const entity_type = e.target.value;
                      setBusy(true);
                      setError("");
                      setData({
                        ...data,
                        metadata: {
                          ...data.metadata,
                          system_metadata: {
                            ...data.metadata.system_metadata,
                            entity_type,
                          },
                        },
                      });
                      try {
                        await api(`/enterprise/entities/${id}`, "PUT", {
                          entity_type,
                        });
                        notify("엔터티 종류를 변경했습니다");
                      } catch (e) {
                        setData(data);
                        setError((e as Error).message);
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    {entityTypes.map(([k, l]) => (
                      <option key={k} value={k}>
                        {l}
                      </option>
                    ))}
                  </select>
                </Field>
              )}
            </div>
            <p style={{ whiteSpace: "pre-wrap" }}>
              {data.document.markdown
                .replace(/^# [^\n]+\n*/, "")
                .slice(0, 3000)}
            </p>
          </section>
          <div className="stats-grid">
            <div className="stat-card">
              <span>관련 문서</span>
              <strong>{data.related_documents.length}</strong>
            </div>
            <div className="stat-card">
              <span>관련 테이블</span>
              <strong>{data.related_tables.length}</strong>
            </div>
            <div className="stat-card">
              <span>연결된 등록 쿼리</span>
              <strong>
                {data.related_tables.reduce(
                  (sum: number, t: Row) => sum + Number(t.query_count || 0),
                  0,
                )}
              </strong>
            </div>
          </div>
          <section className="card">
            <div className="card-header">
              <h2>이 엔터티를 참조하는 문서</h2>
            </div>
            {data.related_documents.length ? (
              <div className="document-list">
                {data.related_documents.map((d: Row) => (
                  <Link
                    className="document-row"
                    key={d.id}
                    to={`/app/documents/${d.id}`}
                  >
                    <FileText size={21} />
                    <strong>{d.title}</strong>
                  </Link>
                ))}
              </div>
            ) : (
              <Empty
                title="참조 문서를 연결해 보세요"
                text={`문서 본문에 [[${data.document.title}]] 링크를 추가하면 여기에 표시됩니다.`}
              />
            )}
            <p className="muted">
              현재 접근 가능한 연결 후보 최대{" "}
              {data.backlink_scan_limit.toLocaleString()}
              개를 확인합니다. 색인 대기 원문 분석은 JSON 기준 16MiB로 제한하며,
              코드 예시의 링크는 제외합니다.
              {data.backlink_truncated &&
                " 표시 범위를 초과한 후보가 있습니다. 전체 결과가 아닙니다."}
            </p>
          </section>
          <section className="card">
            <div className="card-header">
              <h2>관련 데이터와 SQL</h2>
              <Link className="button" to="/app/data-sources">
                연결 관리
              </Link>
            </div>
            {data.related_tables.length ? (
              <div className="document-list">
                {data.related_tables.map((t: Row) => (
                  <Link
                    className="document-row"
                    key={sourceURL(t)}
                    to={sourceURL(t)}
                  >
                    <Database size={21} />
                    <div>
                      <strong>
                        {t.schema_name}.{t.table_name}
                      </strong>
                      <p className="muted">
                        {t.source_name} · {t.engine} · 활성 조회 {t.query_count}
                        개
                      </p>
                      <p>{t.description}</p>
                    </div>
                  </Link>
                ))}
              </div>
            ) : (
              <Empty
                title="연결된 데이터 소스가 없습니다"
                text="테이블의 '설명 · 관련 문서'에서 이 엔터티를 선택하세요. 현재 사용자에게 허용된 소스만 표시됩니다."
              />
            )}
          </section>
        </>
      )}
    </>
  );
}
