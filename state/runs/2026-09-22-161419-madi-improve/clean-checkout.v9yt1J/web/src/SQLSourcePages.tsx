import "./channel-settings.css";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Database,
  Eye,
  Play,
  Plus,
  RefreshCw,
  Save,
  Settings2,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import { Text2SQLPanel } from "./Text2SQLPanel";
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
const engines = [
  ["postgres", "PostgreSQL", 5432],
  ["mysql", "MySQL", 3306],
  ["mariadb", "MariaDB", 3306],
  ["mssql", "SQL Server", 1433],
  ["oracle", "Oracle", 1521],
] as const;
const operators = [
  ["eq", "같음"],
  ["ne", "다름"],
  ["gt", "초과"],
  ["gte", "이상"],
  ["lt", "미만"],
  ["lte", "이하"],
  ["contains", "문자열 포함"],
  ["is_null", "값 없음"],
  ["not_null", "값 있음"],
];
const tableKey = (t: Row) => `${t.schema_name}.${t.table_name}`;
function newSource(wid: string, service: string) {
  return {
    workspace_id: wid,
    space_id: "",
    service_account_id: service,
    name: "",
    kind: "postgres",
    enabled: false,
    credentials: { username: "", password: "" },
    config: {
      host: "",
      port: 5432,
      database: "",
      tables: [],
      timeout_seconds: 15,
      max_rows: 500,
      allow_plaintext: false,
      insecure_tls: false,
      acknowledge_readonly: false,
      acknowledge_acl: false,
      allow_ai: false,
    },
  };
}
export function SQLSourcePage() {
  const { workspace, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const [rows, setRows] = useState<Row[]>([]),
    [spaces, setSpaces] = useState<Row[]>([]),
    [accounts, setAccounts] = useState<Row[]>([]),
    [detail, setDetail] = useState<Row | null>(null),
    [draft, setDraft] = useState<Row | null>(null),
    [queryDraft, setQueryDraft] = useState<Row | null>(null),
    [annotation, setAnnotation] = useState<Row | null>(null),
    [result, setResult] = useState<Row | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true);
  const generation = useRef(0),
    activeWorkspace = useRef(workspace?.id);
  activeWorkspace.current = workspace?.id;
  const manager = ["owner", "admin"].includes(workspace?.role || "");
  const source = rows.find((r) => r.id === params.get("source"));
  const activeSource = useRef(source?.id);
  activeSource.current = source?.id;
  const table = (detail?.tables || []).find(
    (t: Row) => tableKey(t) === params.get("table"),
  );
  const load = useCallback(async () => {
    if (!workspace?.id) return;
    const run = ++generation.current;
    try {
      const requests = [
        api<Row[]>(`/data-sources?workspace_id=${workspace.id}`),
        api<Row[]>(`/spaces?workspace_id=${workspace.id}`),
        manager
          ? api<Row[]>(`/connectors/options?workspace_id=${workspace.id}`)
          : Promise.resolve([]),
      ];
      const [r, s, a] = await Promise.all(requests);
      if (run !== generation.current) return;
      setRows(r);
      setSpaces(s.filter((x) => x.can_write));
      setAccounts(a);
      setError("");
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, [workspace?.id, manager]);
  const loadDetail = useCallback(async () => {
    if (!source) return;
    const id = source.id;
    const data = await api<Row>(`/data-sources/${id}`);
    if (
      activeWorkspace.current === source.workspace_id &&
      activeSource.current === id
    )
      setDetail(data);
  }, [source?.id, source?.revision]);
  useEffect(() => {
    setRows([]);
    setDetail(null);
    setDraft(null);
    setQueryDraft(null);
    setAnnotation(null);
    setResult(null);
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  useEffect(() => {
    setDetail(null);
    setResult(null);
    void loadDetail().catch((e) => setError(e.message));
  }, [loadDetail]);
  const run = async (action: () => Promise<any>, message?: string) => {
    setBusy(true);
    setError("");
    try {
      await action();
      if (message) notify(message);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const makeQuery = (t: Row) => {
    setQueryDraft({
      name: `${t.table_name} 조회`,
      enabled: true,
      plan: {
        schema: t.schema_name,
        table: t.table_name,
        columns: t.columns.map((c: Row) => c.name).slice(0, 10),
        filters: [],
        order: [],
        limit: Math.min(detail?.source?.config?.max_rows || 500, 100),
      },
    });
  };
  return (
    <>
      <PageHeading
        eyebrow="METADATA FIRST · READ ONLY"
        title="외부 데이터 소스"
        description="테이블을 지식으로 설명하고, 승인된 조회만 실행합니다. 외부 행 데이터는 madi 문서 DB에 복제하지 않습니다."
        actions={
          <>
            <Link className="button" to="/app/connectors">
              문서 커넥터
            </Link>
            <Button onClick={() => void load()}>
              <RefreshCw size={17} />
              새로고침
            </Button>
            {manager && (
              <Button
                variant="primary"
                onClick={() =>
                  setDraft(newSource(workspace!.id, accounts[0]?.id || ""))
                }
              >
                <Plus size={17} />
                데이터 소스 추가
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      <div className="notice">
        <ShieldCheck size={18} />
        <span>
          외부 연결 정책에서 정확한 DB 호스트를 허용해야 합니다. 원격 계정의
          실제 읽기 권한, 현재 사용자와 서비스 계정의 공간 권한, 허용 테이블
          목록을 함께 확인합니다.
        </span>
      </div>
      {loading ? (
        <Loading />
      ) : rows.length ? (
        <div className="card">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>데이터 소스</th>
                  <th>엔진</th>
                  <th>상태</th>
                  <th>관리</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={row.id}>
                    <td>
                      <Link to={`?source=${row.id}`}>{row.name}</Link>
                    </td>
                    <td>{engines.find((x) => x[0] === row.kind)?.[1]}</td>
                    <td>
                      <Badge>{row.enabled ? "활성" : "중지"}</Badge>
                    </td>
                    <td>
                      <div className="button-row">
                        <Button onClick={() => setParams({ source: row.id })}>
                          <Database size={16} />
                          테이블 보기
                        </Button>
                        {row.can_manage && (
                          <Button
                            onClick={() =>
                              setDraft({
                                ...row,
                                credentials: { username: "", password: "" },
                              })
                            }
                          >
                            <Settings2 size={16} />
                            설정
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <Empty
          title="외부 데이터베이스를 연결하세요"
          text="PostgreSQL·MySQL·MariaDB·Oracle·SQL Server를 읽기 전용으로 연결하고 테이블 설명과 관련 SQL을 관리할 수 있습니다."
        />
      )}
      {source && (
        <section className="card">
          <div className="card-header">
            <div>
              <h2>{source.name}</h2>
              <p>
                먼저 연결 진단으로 허용된 테이블의 컬럼과 읽기 전용 권한을
                확인합니다.
              </p>
            </div>
            {source.can_manage && (
              <Button
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await api(`/data-sources/${source.id}/inspect`, "POST", {});
                    await loadDetail();
                  }, "연결 권한과 테이블 메타데이터를 확인했습니다")
                }
              >
                <Eye size={17} />
                {busy ? "진단 중…" : "연결 진단 · 메타데이터 갱신"}
              </Button>
            )}
          </div>
          {!detail ? (
            <Loading />
          ) : (
            <>
              <div className="button-row">
                {detail.tables.map((t: Row) => (
                  <Button
                    key={tableKey(t)}
                    variant={table === t ? "primary" : ""}
                    onClick={() =>
                      setParams({ source: source.id, table: tableKey(t) })
                    }
                  >
                    {tableKey(t)}
                  </Button>
                ))}
              </div>
              {!detail.tables.length && (
                <Empty
                  title="아직 검사된 테이블이 없습니다"
                  text="관리자가 연결 진단을 실행하면 컬럼과 데이터 형식을 확인할 수 있습니다."
                />
              )}
              {table && (
                <>
                  <div className="card-header">
                    <div>
                      <h3>{tableKey(table)}</h3>
                      <p>
                        {table.description ||
                          "이 테이블의 목적과 주요 컬럼을 설명해 보세요."}
                      </p>
                      <small className="muted">
                        메타데이터 확인 {datetime(table.inspected_at)}
                      </small>
                    </div>
                    {source.can_manage && (
                      <div className="button-row">
                        <Button onClick={() => setAnnotation({ ...table })}>
                          설명 · 관련 문서
                        </Button>
                        <Button onClick={() => makeQuery(table)}>
                          <Plus size={17} />
                          조회 등록
                        </Button>
                      </div>
                    )}
                  </div>
                  <div className="table-wrap">
                    <table>
                      <thead>
                        <tr>
                          <th>컬럼</th>
                          <th>데이터 형식</th>
                          <th>빈 값 허용</th>
                        </tr>
                      </thead>
                      <tbody>
                        {table.columns.map((c: Row) => (
                          <tr key={c.name}>
                            <td>
                              <code>{c.name}</code>
                            </td>
                            <td>{c.type}</td>
                            <td>{c.nullable ? "허용" : "필수"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {table.document_ids?.length > 0 && (
                    <>
                      <h4>관련 지식</h4>
                      <ul>
                        {table.document_ids.map((id: string) => (
                          <li key={id}>
                            <Link to={`/app/documents/${id}`}>
                              {documents.find((d) => d.id === id)?.title ||
                                "관련 문서 열기"}
                            </Link>
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                  <h3>등록된 조회</h3>
                  {detail.queries
                    .filter(
                      (q: Row) =>
                        q.plan.schema === table.schema_name &&
                        q.plan.table === table.table_name,
                    )
                    .map((q: Row) => (
                      <div key={q.id} className="card-header">
                        <div>
                          <strong>{q.name}</strong>
                          <p>
                            {q.plan.columns.join(", ")} · 최대 {q.plan.limit}행
                            · {q.enabled ? "사용 중" : "중지"}
                          </p>
                        </div>
                        <div className="button-row">
                          {source.can_manage && !q.origin_proposal_id && (
                            <Button
                              onClick={() =>
                                setQueryDraft(JSON.parse(JSON.stringify(q)))
                              }
                            >
                              조회 편집
                            </Button>
                          )}
                          <Button
                            disabled={busy || !source.enabled || !q.enabled}
                            onClick={() =>
                              void run(async () => {
                                const scope = activeWorkspace.current;
                                const response = await api<Row>(
                                  `/data-sources/${source.id}/queries/${q.id}/execute`,
                                  "POST",
                                  {},
                                );
                                if (scope === activeWorkspace.current)
                                  setResult({ ...response, name: q.name });
                                await loadDetail();
                              })
                            }
                          >
                            <Play size={17} />
                            조회 실행
                          </Button>
                        </div>
                      </div>
                    ))}
                  <Text2SQLPanel
                    source={source}
                    table={table}
                    onChanged={loadDetail}
                  />
                </>
              )}
              {source.can_manage && detail.runs.length > 0 && (
                <details>
                  <summary>최근 조회 이력</summary>
                  <div className="table-wrap">
                    <table>
                      <thead>
                        <tr>
                          <th>실행 시간</th>
                          <th>결과</th>
                          <th>행 수</th>
                          <th>소요 시간</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detail.runs.map((r: Row) => (
                          <tr key={r.id}>
                            <td>{datetime(r.created_at)}</td>
                            <td>
                              {r.status === "succeeded" ? "성공" : "실패"}
                            </td>
                            <td>{r.row_count}</td>
                            <td>{r.duration_ms}ms</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </details>
              )}
            </>
          )}
        </section>
      )}
      {result && (
        <Modal
          open
          wide
          onOpenChange={() => setResult(null)}
          title={`${result.name} · 조회 결과`}
        >
          <p className="muted">
            {result.rows.length}행 · {result.duration_ms}ms · 결과는 서버 이력에
            저장하지 않습니다.
          </p>
          {result.truncated && (
            <div className="notice">
              조회 한도를 넘는 데이터가 있습니다. 저장된 조건을 좁혀서 다시
              조회하세요.
            </div>
          )}
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  {result.columns.map((c: string) => (
                    <th key={c}>{c}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {result.rows.map((r: Row, i: number) => (
                  <tr key={i}>
                    {result.columns.map((c: string) => (
                      <td key={c}>
                        {r[c] === null ? (
                          <span className="muted">NULL</span>
                        ) : (
                          String(r[c])
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <details>
            <summary>실행된 SELECT (값은 매개변수로 전달)</summary>
            <pre>{result.sql}</pre>
          </details>
        </Modal>
      )}
      {draft && (
        <SourceEditor
          draft={draft}
          setDraft={setDraft}
          accounts={accounts}
          spaces={spaces}
          busy={busy}
          error={error}
          onSave={() =>
            void run(async () => {
              await api(
                `/data-sources${draft.id ? `/${draft.id}` : ""}`,
                draft.id ? "PUT" : "POST",
                draft,
              );
              setDraft(null);
              await load();
            }, "데이터 소스 설정을 저장했습니다")
          }
        />
      )}
      {queryDraft && source && detail && (
        <QueryEditor
          draft={queryDraft}
          setDraft={setQueryDraft}
          tables={detail.tables}
          maximum={detail.source.config?.max_rows || 500}
          busy={busy}
          error={error}
          onSave={() =>
            void run(async () => {
              await api(
                `/data-sources/${source.id}/queries${queryDraft.id ? `/${queryDraft.id}` : ""}`,
                queryDraft.id ? "PUT" : "POST",
                queryDraft,
              );
              setQueryDraft(null);
              await loadDetail();
            }, "제한된 조회 계획을 저장했습니다")
          }
        />
      )}
      {annotation && source && (
        <Modal
          open
          onOpenChange={() => {
            if (!busy) setAnnotation(null);
          }}
          title="테이블 설명과 관련 지식"
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void run(async () => {
                await api(
                  `/data-sources/${source.id}/table`,
                  "PUT",
                  annotation,
                );
                setAnnotation(null);
                await loadDetail();
              }, "테이블 설명을 저장했습니다");
            }}
          >
            <ErrorBox error={error} />
            <Field label="테이블 설명">
              <textarea
                rows={5}
                maxLength={20000}
                value={annotation.description}
                onChange={(e) =>
                  setAnnotation({ ...annotation, description: e.target.value })
                }
              />
            </Field>
            <Field label="관련 문서">
              <div className="checkbox-options">
                {documents
                  .filter((d) => !d.deleted_at)
                  .map((d) => (
                    <label key={d.id}>
                      <input
                        type="checkbox"
                        checked={annotation.document_ids.includes(d.id)}
                        onChange={(e) =>
                          setAnnotation({
                            ...annotation,
                            document_ids: e.target.checked
                              ? [...annotation.document_ids, d.id]
                              : annotation.document_ids.filter(
                                  (id: string) => id !== d.id,
                                ),
                          })
                        }
                      />
                      {d.title}
                    </label>
                  ))}
              </div>
            </Field>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setAnnotation(null)}
              >
                취소
              </Button>
              <Button type="submit" variant="primary" disabled={busy}>
                설명 저장
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}

function SourceEditor({
  draft,
  setDraft,
  accounts,
  spaces,
  busy,
  error,
  onSave,
}: {
  draft: Row;
  setDraft: (v: Row | null) => void;
  accounts: Row[];
  spaces: Row[];
  busy: boolean;
  error: string;
  onSave: () => void;
}) {
  const config = (k: string, v: any) =>
    setDraft({ ...draft, config: { ...draft.config, [k]: v } });
  const [tables, setTables] = useState((draft.config.tables || []).join("\n"));
  return (
    <Modal
      open
      wide
      title={draft.id ? "데이터 소스 설정" : "데이터 소스 추가"}
      onOpenChange={() => {
        if (!busy) setDraft(null);
      }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSave();
        }}
      >
        <ErrorBox error={error} />
        <div className="form-grid">
          <Field label="데이터 소스 이름">
            <input
              required
              maxLength={120}
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </Field>
          <Field label="데이터베이스 엔진">
            <select
              disabled={!!draft.id}
              value={draft.kind}
              onChange={(e) =>
                setDraft({
                  ...draft,
                  kind: e.target.value,
                  config: {
                    ...draft.config,
                    port:
                      engines.find((x) => x[0] === e.target.value)?.[2] || 5432,
                  },
                })
              }
            >
              {engines.map(([k, l]) => (
                <option key={k} value={k}>
                  {l}
                </option>
              ))}
            </select>
          </Field>
          <Field label="DB 호스트">
            <input
              required
              value={draft.config.host}
              placeholder="db.example.internal"
              onChange={(e) => config("host", e.target.value)}
            />
          </Field>
          <Field label="DB 포트">
            <input
              required
              type="number"
              min={1}
              max={65535}
              value={draft.config.port}
              onChange={(e) => config("port", Number(e.target.value))}
            />
          </Field>
          <Field
            label={
              draft.kind === "oracle"
                ? "Oracle 서비스 이름"
                : "데이터베이스 이름"
            }
          >
            <input
              required
              value={draft.config.database}
              onChange={(e) => config("database", e.target.value)}
            />
          </Field>
          <Field label="대상 공간">
            <select
              disabled={!!draft.id}
              value={draft.space_id || ""}
              onChange={(e) => setDraft({ ...draft, space_id: e.target.value })}
            >
              <option value="">워크스페이스 루트</option>
              {spaces.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="전용 서비스 계정">
            <select
              required
              disabled={!!draft.id}
              value={draft.service_account_id}
              onChange={(e) =>
                setDraft({ ...draft, service_account_id: e.target.value })
              }
            >
              <option value="">선택하세요</option>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="원격 읽기 전용 사용자">
            <input
              autoComplete="off"
              required={!draft.id}
              value={draft.credentials.username}
              placeholder={draft.id ? "빈 값은 현재 계정 유지" : "madi_reader"}
              onChange={(e) =>
                setDraft({
                  ...draft,
                  credentials: {
                    ...draft.credentials,
                    username: e.target.value,
                  },
                })
              }
            />
          </Field>
          <Field label="원격 계정 암호">
            <input
              autoComplete="new-password"
              type="password"
              value={draft.credentials.password}
              placeholder={draft.id ? "빈 값은 현재 암호 유지" : ""}
              onChange={(e) =>
                setDraft({
                  ...draft,
                  credentials: {
                    ...draft.credentials,
                    password: e.target.value,
                  },
                })
              }
            />
          </Field>
          <Field label="조회 제한 시간 (초)">
            <input
              type="number"
              min={1}
              max={60}
              required
              value={draft.config.timeout_seconds}
              onChange={(e) =>
                config("timeout_seconds", Number(e.target.value))
              }
            />
          </Field>
          <Field label="최대 조회 행">
            <input
              type="number"
              min={1}
              max={2000}
              required
              value={draft.config.max_rows}
              onChange={(e) => config("max_rows", Number(e.target.value))}
            />
          </Field>
        </div>
        <Field label="허용 테이블 (줄마다 schema.table)">
          <textarea
            required
            rows={5}
            value={tables}
            placeholder={"public.orders\npublic.customers"}
            onChange={(e) => {
              setTables(e.target.value);
              config(
                "tables",
                e.target.value
                  .split(/[\n,]/)
                  .map((v) => v.trim())
                  .filter(Boolean),
              );
            }}
          />
        </Field>
        <p className="muted">
          최대 100개 테이블, 영문·숫자·밑줄 기반 식별자만 지원합니다.
          MySQL/MariaDB는 schema에 DB 이름, Oracle은 저장된 대문자
          소유자·테이블명을 사용하세요. 연결 진단은 컬럼 메타데이터만
          가져옵니다.
        </p>
        <Field label="사내 CA 인증서 PEM (선택)">
          <textarea
            rows={3}
            value={draft.config.ca_pem || ""}
            onChange={(e) => config("ca_pem", e.target.value)}
          />
        </Field>
        <label className="check-row">
          <input
            type="checkbox"
            checked={!!draft.config.allow_plaintext}
            onChange={(e) => config("allow_plaintext", e.target.checked)}
          />
          사내망 비암호화 연결 허용
        </label>
        <label className="check-row">
          <input
            type="checkbox"
            checked={!!draft.config.insecure_tls}
            onChange={(e) => config("insecure_tls", e.target.checked)}
          />
          서버 인증서 검증 생략 (테스트 전용)
        </label>
        <label className="check-row">
          <input
            type="checkbox"
            required
            checked={!!draft.config.acknowledge_readonly}
            onChange={(e) => config("acknowledge_readonly", e.target.checked)}
          />
          원격 계정에 관리자·DDL·쓰기 권한을 주지 않고 필요한 테이블 SELECT만
          허용했습니다.
        </label>
        <label className="check-row">
          <input
            type="checkbox"
            required
            checked={!!draft.config.acknowledge_acl}
            onChange={(e) => config("acknowledge_acl", e.target.checked)}
          />
          원격 데이터 조회 결과와 메타데이터를 대상 공간의 접근 권한으로
          제공함을 확인합니다.
        </label>
        <label className="check-row">
          <input
            type="checkbox"
            checked={!!draft.config.allow_ai}
            onChange={(e) => config("allow_ai", e.target.checked)}
          />
          AI 조회 계획 제안 허용 (메타데이터만 모델에 전송)
        </label>
        <label className="check-row">
          <input
            type="checkbox"
            checked={!!draft.enabled}
            onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })}
          />
          데이터 소스 활성화
        </label>
        <div className="notice">
          <ShieldCheck size={18} />
          <span>
            SQL Server는 읽기 전용 트랜잭션 모드가 없어 실제 읽기 권한 검사와
            제한된 SELECT 생성기를 함께 적용합니다. 다른 엔진은 읽기 전용
            트랜잭션도 적용합니다. 자유 SQL·함수·프로시저·쓰기 쿼리는 제공하지
            않습니다.
          </span>
        </div>
        <div className="modal-actions">
          <Button type="button" disabled={busy} onClick={() => setDraft(null)}>
            취소
          </Button>
          <Button type="submit" variant="primary" disabled={busy}>
            <Save size={17} />
            {busy ? "저장 중…" : "데이터 소스 저장"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
function QueryEditor({
  draft,
  setDraft,
  tables,
  maximum,
  busy,
  error,
  onSave,
}: {
  draft: Row;
  setDraft: (v: Row | null) => void;
  tables: Row[];
  maximum: number;
  busy: boolean;
  error: string;
  onSave: () => void;
}) {
  const p = draft.plan;
  const columns = (tables.find(
    (t) => t.schema_name === p.schema && t.table_name === p.table,
  )?.columns || []) as Row[];
  const plan = (key: string, value: any) =>
    setDraft({ ...draft, plan: { ...p, [key]: value } });
  const filter = (index: number, key: string, value: any) =>
    plan(
      "filters",
      p.filters.map((f: Row, i: number) =>
        i === index ? { ...f, [key]: value } : f,
      ),
    );
  return (
    <Modal
      open
      wide
      title="허용된 SELECT 등록"
      onOpenChange={() => {
        if (!busy) setDraft(null);
      }}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSave();
        }}
      >
        <ErrorBox error={error} />
        <p className="muted">
          {p.schema}.{p.table} · 선택한 컬럼과 조건은 매개변수화된 SELECT로만
          실행됩니다.
        </p>
        <Field label="저장 쿼리 이름">
          <input
            required
            maxLength={120}
            value={draft.name}
            onChange={(e) => setDraft({ ...draft, name: e.target.value })}
          />
        </Field>
        <Field label="조회 컬럼">
          <div className="checkbox-options">
            {columns.map((c) => (
              <label key={c.name}>
                <input
                  type="checkbox"
                  checked={p.columns.includes(c.name)}
                  onChange={(e) =>
                    plan(
                      "columns",
                      e.target.checked
                        ? [...p.columns, c.name]
                        : p.columns.filter((v: string) => v !== c.name),
                    )
                  }
                />
                {c.name} <small>{c.type}</small>
              </label>
            ))}
          </div>
        </Field>
        <h3>필터 조건 (모두 일치)</h3>
        {p.filters.map((f: Row, index: number) => (
          <div className="form-grid" key={index}>
            <Field label={`조건 ${index + 1} 컬럼`}>
              <select
                value={f.column}
                onChange={(e) => filter(index, "column", e.target.value)}
              >
                {columns.map((c) => (
                  <option key={c.name} value={c.name}>
                    {c.name}
                  </option>
                ))}
              </select>
            </Field>
            <Field label={`조건 ${index + 1} 연산`}>
              <select
                value={f.operator}
                onChange={(e) => filter(index, "operator", e.target.value)}
              >
                {operators.map(([k, l]) => (
                  <option key={k} value={k}>
                    {l}
                  </option>
                ))}
              </select>
            </Field>
            {!["is_null", "not_null"].includes(f.operator) && (
              <Field label={`조건 ${index + 1} 값`}>
                <input
                  maxLength={4096}
                  value={String(f.value ?? "")}
                  onChange={(e) => {
                    const type =
                      columns.find((c) => c.name === f.column)?.type || "";
                    const text = e.target.value;
                    filter(
                      index,
                      "value",
                      /int|numeric|decimal|float|double|real|number/i.test(
                        type,
                      ) &&
                        text !== "" &&
                        Number.isFinite(Number(text))
                        ? Number(text)
                        : /bool/i.test(type) && ["true", "false"].includes(text)
                          ? text === "true"
                          : text,
                    );
                  }}
                />
              </Field>
            )}
            <Button
              type="button"
              onClick={() =>
                plan(
                  "filters",
                  p.filters.filter((_: any, i: number) => i !== index),
                )
              }
            >
              <Trash2 size={16} />
              조건 삭제
            </Button>
          </div>
        ))}
        <Button
          type="button"
          disabled={p.filters.length >= 30 || !columns.length}
          onClick={() =>
            plan("filters", [
              ...p.filters,
              { column: columns[0].name, operator: "eq", value: "" },
            ])
          }
        >
          <Plus size={17} />
          조건 추가
        </Button>
        <div className="form-grid">
          <Field label="정렬 컬럼">
            <select
              value={p.order[0]?.column || ""}
              onChange={(e) =>
                plan(
                  "order",
                  e.target.value
                    ? [
                        {
                          column: e.target.value,
                          direction: p.order[0]?.direction || "asc",
                        },
                      ]
                    : [],
                )
              }
            >
              <option value="">정렬 없음</option>
              {columns.map((c) => (
                <option key={c.name} value={c.name}>
                  {c.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="정렬 방향">
            <select
              disabled={!p.order.length}
              value={p.order[0]?.direction || "asc"}
              onChange={(e) =>
                plan("order", [{ ...p.order[0], direction: e.target.value }])
              }
            >
              <option value="asc">오름차순</option>
              <option value="desc">내림차순</option>
            </select>
          </Field>
          <Field label="조회 행 제한">
            <input
              type="number"
              min={1}
              max={maximum}
              required
              value={p.limit}
              onChange={(e) => plan("limit", Number(e.target.value))}
            />
          </Field>
        </div>
        <label className="check-row">
          <input
            type="checkbox"
            checked={!!draft.enabled}
            onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })}
          />
          이 저장 쿼리 사용
        </label>
        <div className="modal-actions">
          <Button type="button" disabled={busy} onClick={() => setDraft(null)}>
            취소
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={busy || !p.columns.length}
          >
            <Save size={17} />
            조회 계획 저장
          </Button>
        </div>
      </form>
    </Modal>
  );
}
