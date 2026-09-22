import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import {
  ArrowRight,
  Clock,
  FilePlus2,
  FileText,
  Inbox,
  Search,
  LockKeyhole,
  BookOpen,
  CheckCircle2,
  Bell,
  RefreshCw,
} from "lucide-react";
import { api, date, type Doc, type DocSummary } from "../api";
import { useApp } from "../context";
import { Button, Empty, ErrorBox, Loading, PageHeading } from "../ui";
import { RecoveryNotice } from "../review/ChangeReview";
import { safeIDs } from "./DocumentActions";
import "./focused-home.css";
import RecentWorksets from "../worksets/RecentWorksets";
type Item = Record<string, any>;
export default function FocusedHome() {
  const { user, workspace, documents, reload, notify } = useApp(),
    navigate = useNavigate();
  const [query, setQuery] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [templates, setTemplates] = useState<Item[]>([]),
    [templateError, setTemplateError] = useState("");
  const current = useRef("");
  current.current = `${user.id}:${workspace?.id}`;
  const request = useRef({ scope: "", id: "" });
  const writable =
    !!workspace &&
    user.role !== "viewer" &&
    ["owner", "admin", "editor"].includes(workspace.role);
  const visible = documents.filter(
    (d) => d.workspace_id === workspace?.id && !d.deleted_at,
  );
  const recent = safeIDs(user.preferences?.recent_documents)
    .map((id) => visible.find((d) => d.id === id))
    .filter((d): d is DocSummary => !!d)
    .slice(0, 5);
  const continueDocuments = recent.length
    ? recent
    : [...visible]
        .sort((a, b) => b.updated_at.localeCompare(a.updated_at))
        .slice(0, 5);
  useEffect(() => {
    let active = true;
    setBusy(false);
    setError(null);
    setQuery("");
    setTemplates([]);
    setTemplateError("");
    if (!workspace) return;
    api<{ items: Item[] }>(`/templates?workspace_id=${workspace.id}&limit=3`)
      .then((data) => {
        if (active) setTemplates(data.items);
      })
      .catch((e) => {
        if (active) setTemplateError(e.message);
      });
    return () => {
      active = false;
    };
  }, [user.id, workspace?.id]);
  const capture = async () => {
    if (!workspace || busy || !writable) return;
    const scope = current.current;
    if (request.current.scope !== scope)
      request.current = { scope, id: crypto.randomUUID() };
    setBusy(true);
    setError(null);
    try {
      const doc = await api<Doc>("/captures", "POST", {
        workspace_id: workspace.id,
        title: "새 개인 메모",
        text: "",
        client_request_id: request.current.id,
      });
      if (current.current !== scope) return;
      request.current = { scope: "", id: "" };
      await reload();
      if (current.current !== scope) return;
      notify(
        "나만 볼 수 있는 메모를 만들었습니다. 분류는 나중에 수집함에서 할 수 있습니다.",
      );
      navigate(`/app/documents/${doc.id}?mode=edit`);
    } catch (e) {
      if (current.current === scope) setError(e);
    } finally {
      if (current.current === scope) setBusy(false);
    }
  };
  return (
    <div className="focused-home">
      <PageHeading
        eyebrow="오늘의 작업"
        title={`${user.name || "사용자"}님, 어디서 이어갈까요?`}
        description="기록하고, 찾고, 이어서 작업하세요."
      />
      <section className="home-start">
        <div>
          <span className="home-eyebrow">
            <LockKeyhole size={17} />
            나만의 빠른 기록
          </span>
          <h2>생각이 사라지기 전에.</h2>
          <p>
            제목이나 분류를 먼저 정하지 않아도 됩니다.
            <br />
            개인 메모를 열고 바로 적으세요.
          </p>
          <Button
            variant="primary"
            disabled={busy || !writable}
            onClick={() => void capture()}
          >
            <FilePlus2 size={19} />
            {busy ? "개인 메모 여는 중…" : "바로 메모"}
          </Button>
          {!writable && (
            <p className="muted">
              문서 작성 권한이 있으면 개인 메모를 만들 수 있습니다.
            </p>
          )}
        </div>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (query.trim())
              navigate("/app/search?q=" + encodeURIComponent(query.trim()));
          }}
        >
          <Search size={25} />
          <h3>필요한 지식 찾기</h3>
          <label htmlFor="home-search">제목, 키워드 또는 궁금한 내용</label>
          <div>
            <input
              id="home-search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="무엇을 찾고 있나요?"
            />
            <Button variant="primary" disabled={!query.trim()}>
              검색
            </Button>
          </div>
          <Link to="/app/search">
            상세 검색 열기 <ArrowRight size={16} />
          </Link>
        </form>
      </section>
      <RecoveryNotice error={error} onRetry={() => void capture()} />
      <div className="home-work-grid">
        <section className="panel padded">
          <div className="section-heading">
            <h2>
              <Clock size={21} />
              이어서 작업
            </h2>
            <Link to="/app/documents">모든 문서</Link>
          </div>
          {continueDocuments.length ? (
            <div className="continue-list">
              {continueDocuments.map((d) => (
                <Link to={`/app/documents/${d.id}`} key={d.id}>
                  <FileText size={20} />
                  <span>
                    <strong>{d.title}</strong>
                    <small>{date(d.updated_at)} 업데이트</small>
                  </span>
                  <ArrowRight size={17} />
                </Link>
              ))}
            </div>
          ) : (
            <Empty
              title="아직 열었던 문서가 없습니다"
              text="새 메모를 만들거나 팀 문서에서 작업을 시작하세요."
            />
          )}
          <p className="muted small-text">
            현재 열람 가능한 문서만 표시합니다.
          </p>
        </section>
        <section className="panel padded">
          <div className="section-heading">
            <h2>
              <Inbox size={21} />내 처리함
            </h2>
          </div>
          <p>미분류 메모와 내가 맡은 일을 한곳에서 확인하세요.</p>
          <div className="home-work-links">
            <Link to="/app/my-work">
              <Inbox size={20} />내 처리함 열기
              <ArrowRight size={17} />
            </Link>
            <Link to="/app/tasks?filter=mine">
              <CheckCircle2 size={20} />
              나의 할 일<ArrowRight size={17} />
            </Link>
            <Link to="/app/inbox">
              <LockKeyhole size={20} />
              수집한 개인 메모
              <ArrowRight size={17} />
            </Link>
          </div>
        </section>
      </div>
      <RecentWorksets />
      <section className="panel padded">
        <div className="section-heading">
          <h2>
            <BookOpen size={21} />
            템플릿으로 시작
          </h2>
          <Link to="/app/templates">템플릿 모두 보기</Link>
        </div>
        <p className="muted">
          최근 업데이트된 사용자 템플릿을 바로 열거나, 기본 회의록·의사결정·일일
          노트 양식을 선택하세요.
        </p>
        <ErrorBox error={templateError} />
        <div className="home-template-list">
          {templates.map((t) => (
            <Link key={t.id} to={`/app/templates?template=${t.id}`}>
              <BookOpen size={20} />
              <strong>{t.name}</strong>
              <small>
                {t.description || "내용을 검토한 뒤 새 문서로 사용"}
              </small>
            </Link>
          ))}
          <Link to="/app/templates?source=builtin">
            <BookOpen size={20} />
            <strong>기본 템플릿 모음</strong>
            <small>회의록 · 결정 기록 · 운영 절차 · 일일 노트</small>
          </Link>
        </div>
      </section>
    </div>
  );
}
export function MyWorkPage() {
  const { user, workspace, publicInfo } = useApp();
  const [data, setData] = useState<{
      captures: Item[];
      tasks: Item[];
      notices: Item[];
      approvals: Item[];
    } | null>(null),
    [error, setError] = useState<unknown>(null),
    [revision, setRevision] = useState(0);
  useEffect(() => {
    let active = true;
    setData(null);
    setError(null);
    if (!workspace) return;
    const load = () =>
      Promise.all([
        api<Item[]>(`/captures?workspace_id=${workspace.id}`),
        api<{ items: Item[] }>(`/tasks/board?workspace_id=${workspace.id}`),
        api<Item[]>("/notifications"),
        publicInfo.approval_enabled
          ? api<Item[]>(`/approvals/inbox?workspace_id=${workspace.id}`)
          : Promise.resolve([]),
      ])
        .then(([captures, tasks, notices, approvals]) => {
          if (active)
            setData({
              captures,
              tasks: tasks.items.filter(
                (t) =>
                  !t.done &&
                  (t.assignee_id === user.id ||
                    (!t.assignee_id && t.owner_id === user.id)),
              ),
              notices: notices.filter((n) => !n.read_at),
              approvals,
            });
        })
        .catch((e) => {
          if (active) {
            setData(null);
            setError(e);
          }
        });
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [user.id, workspace?.id, publicInfo.approval_enabled, revision]);
  return (
    <div className="my-work-page">
      <PageHeading
        eyebrow="확인하고 이어가기"
        title="내 처리함"
        description="수집한 메모, 나의 할 일, 계정 알림을 모았습니다. 검토함은 관리자가 활성화한 경우에만 표시됩니다."
        actions={
          <Button onClick={() => setRevision((v) => v + 1)}>
            <RefreshCw size={17} />
            새로 불러오기
          </Button>
        }
      />
      <ErrorBox error={error} />
      {!data && !error ? (
        <Loading />
      ) : (
        data && (
          <div className="my-work-grid">
            <WorkSection
              title="분류할 개인 메모"
              icon={<Inbox size={22} />}
              items={data.captures.slice(0, 5).map((d) => ({
                id: d.id,
                title: d.title,
                to: `/app/documents/${d.id}`,
              }))}
              empty="수집함이 비어 있습니다."
              more="/app/inbox"
            />
            <WorkSection
              title="내가 맡은 할 일"
              icon={<CheckCircle2 size={22} />}
              items={data.tasks.slice(0, 5).map((t, i) => ({
                id: t.task_id || String(i),
                title: t.text,
                to: `/app/tasks?filter=mine&document_id=${t.document_id}`,
              }))}
              empty="현재 나에게 할당된 미완료 작업이 없습니다."
              more="/app/tasks?filter=mine"
            />
            <WorkSection
              title="읽지 않은 계정 알림"
              description="모든 워크스페이스의 내 알림 중 현재 접근 가능한 내용"
              icon={<Bell size={22} />}
              items={data.notices.slice(0, 5).map((n) => ({
                id: n.id,
                title: n.title,
                to: n.document_id
                  ? `/app/documents/${n.document_id}`
                  : undefined,
                action: async () => {
                  await api(`/notifications/${n.id}/read`, "POST", {});
                  setRevision((v) => v + 1);
                },
              }))}
              empty="새 알림이 없습니다."
            />
            {publicInfo.approval_enabled && (
              <WorkSection
                title="내 검토가 필요한 요청"
                icon={<CheckCircle2 size={22} />}
                items={data.approvals.slice(0, 5).map((a) => ({
                  id: a.id,
                  title: a.title || "검토 요청",
                  to: `/app/approvals?request=${a.id}`,
                }))}
                empty="현재 검토할 요청이 없습니다."
                more="/app/approvals"
              />
            )}
          </div>
        )
      )}
    </div>
  );
}
function WorkSection({
  title,
  description,
  icon,
  items,
  empty,
  more,
}: {
  title: string;
  description?: string;
  icon: React.ReactNode;
  items: {
    id: string;
    title: string;
    to?: string;
    action?: () => Promise<void>;
  }[];
  empty: string;
  more?: string;
}) {
  const [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState("");
  return (
    <section className="panel padded">
      <div className="section-heading">
        <h2>
          {icon}
          {title}
        </h2>
        {more && <Link to={more}>모두 보기</Link>}
      </div>
      {description && <p className="muted small-text">{description}</p>}
      <ErrorBox error={error} />
      {items.length ? (
        <ul className="work-inbox-list">
          {items.map((item) => (
            <li key={item.id}>
              {item.to ? (
                <Link to={item.to}>{item.title}</Link>
              ) : (
                <span>{item.title}</span>
              )}
              {item.action && (
                <Button
                  disabled={!!busy}
                  onClick={async () => {
                    setBusy(item.id);
                    try {
                      await item.action!();
                    } catch (e) {
                      setError(e);
                    } finally {
                      setBusy("");
                    }
                  }}
                >
                  읽음 처리
                </Button>
              )}
            </li>
          ))}
        </ul>
      ) : (
        <p className="muted">{empty}</p>
      )}
    </section>
  );
}
