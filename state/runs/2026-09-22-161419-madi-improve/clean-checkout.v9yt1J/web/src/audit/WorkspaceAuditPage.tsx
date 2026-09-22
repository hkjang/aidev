import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { History, RefreshCw, ShieldCheck } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  PageHeading,
  statusNames,
} from "../ui";
import "./style.css";
type Row = {
  id: string;
  created_at: string;
  action: string;
  actor_name: string;
  resource_kind: string;
  resource_id: string;
  resource_title: string;
  changes: Record<string, unknown>;
};
type Data = { items: Row[]; next_cursor: string; notice: string };
const actions: Record<string, string> = {
  DOCUMENT_CREATE: "문서 만들기",
  DOCUMENT_UPDATE: "문서 수정",
  DOCUMENT_DELETE: "문서 휴지통 이동",
  DOCUMENT_RESTORE: "문서 복구",
  DOCUMENT_READ: "문서 읽기",
  SHARE_CREATE: "공유 권한 변경",
  WORKSPACE_SETTINGS_UPDATE: "팀 설정 변경",
  WORKSPACE_MEMBER_UPDATE: "팀 구성원 변경",
  DATABASE_CREATE: "데이터베이스 만들기",
  DATABASE_UPDATE: "데이터베이스 설정 변경",
  DATABASE_ROW_CREATE: "데이터 행 만들기",
  DATABASE_ROW_UPDATE: "데이터 행 수정",
  TEMPLATE_CREATE: "템플릿 만들기",
  TEMPLATE_UPDATE: "템플릿 수정",
  FILE_UPLOAD: "첨부파일 올리기",
  FILE_DOWNLOAD: "첨부파일 내려받기",
  KEY_CREATE: "API 키 발급",
  KEY_ROTATE: "API 키 회전",
  KEY_REVOKE: "API 키 폐기",
};
const kinds: Record<string, string> = {
  document: "문서",
  database: "데이터베이스",
  space: "공간",
  template: "템플릿",
  canvas: "캔버스",
  key: "API 키",
  storage: "저장소",
  workspace: "워크스페이스",
};
function link(row: Row) {
  switch (row.resource_kind) {
    case "document":
      return "/app/documents/" + row.resource_id + "?mode=read";
    case "database":
      return "/app/databases/" + row.resource_id;
    case "canvas":
      return "/app/canvases/" + row.resource_id;
    case "template":
      return "/app/templates";
    case "space":
      return "/app/spaces";
    default:
      return null;
  }
}
function label(action: string) {
  if (actions[action]) return actions[action];
  const [kind, ...rest] = action.split("_");
  const first: Record<string, string> = {
    DOCUMENT: "문서",
    DATABASE: "데이터베이스",
    TEMPLATE: "템플릿",
    CANVAS: "캔버스",
    WORKSPACE: "팀",
    SPACE: "공간",
    KEY: "키",
    FILE: "첨부",
    COMMENT: "댓글",
    STORAGE: "저장소",
    AI: "AI",
  };
  const last: Record<string, string> = {
    CREATE: "생성",
    UPDATE: "변경",
    DELETE: "삭제",
    RESTORE: "복구",
    READ: "조회",
    CONNECT: "접속",
    ERROR: "오류",
    QUERY: "질의",
    CHANGE: "변경",
    TEST: "진단",
    ASSIGNMENT: "할당",
    ROTATE: "회전",
    REVOKE: "폐기",
  };
  return `${first[kind] || "관리"} ${last[rest.at(-1) || ""] || "동작"} · ${action}`;
}
export default function WorkspaceAuditPage() {
  const { workspace, user } = useApp();
  const [params, setParams] = useSearchParams();
  const [result, setData] = useState<(Data & { scope: string }) | null>(null),
    [error, setError] = useState<unknown>(null),
    [refresh, setRefresh] = useState(0);
  const generation = useRef(0);
  const days = params.get("days") || "30",
    action = params.get("action") || "",
    cursor = params.get("cursor") || "";
  const scope = `${user.id}:${workspace?.id}:${days}:${action}:${cursor}`;
  const liveScope = useRef(scope);
  liveScope.current = scope;
  const data = result?.scope === scope ? result : null;
  useEffect(() => {
    const ticket = ++generation.current;
    setData(null);
    setError(null);
    if (!workspace) return;
    api<Data>(
      `/workspaces/${workspace.id}/audit?${new URLSearchParams({ days, action, cursor })}`,
    )
      .then((value) => {
        if (ticket === generation.current && liveScope.current === scope)
          setData({ ...value, scope });
      })
      .catch((e) => {
        if (ticket === generation.current && liveScope.current === scope)
          setError(e);
      });
    return () => {
      generation.current++;
    };
  }, [scope, refresh]);
  const update = (key: string, value: string) =>
    setParams((current) => {
      const next = new URLSearchParams(current);
      next.delete("cursor");
      value ? next.set(key, value) : next.delete(key);
      return next;
    });
  return (
    <div className="workspace-audit">
      <PageHeading
        eyebrow="WORKSPACE AUDIT"
        title="팀 감사로그"
        description="현재 접근할 수 있는 팀 자료의 주요 관리 동작을 확인합니다."
        actions={
          <Button variant="secondary" onClick={() => setRefresh((v) => v + 1)}>
            <RefreshCw size={17} />
            다시 불러오기
          </Button>
        }
      />
      <div className="panel padded audit-filters">
        <Field label="조회 기간">
          <select value={days} onChange={(e) => update("days", e.target.value)}>
            {[1, 7, 30, 90, 365].map((n) => (
              <option key={n} value={n}>
                최근 {n}일
              </option>
            ))}
            {![1, 7, 30, 90, 365].includes(Number(days)) && (
              <option value={days}>{days}일</option>
            )}
          </select>
        </Field>
        <Field label="감사 동작">
          <select
            value={action}
            onChange={(e) => update("action", e.target.value)}
          >
            <option value="">모든 동작</option>
            {Object.entries(actions).map(([key, name]) => (
              <option key={key} value={key}>
                {name}
              </option>
            ))}
            {action && !actions[action] && (
              <option value={action}>{action}</option>
            )}
          </select>
        </Field>
        {cursor && (
          <Button variant="secondary" onClick={() => update("cursor", "")}>
            첫 페이지
          </Button>
        )}
      </div>
      <ErrorBox error={error} />
      {!data && !error ? (
        <Loading />
      ) : (
        data && (
          <>
            <p className="notice">
              <ShieldCheck size={19} />
              {data.notice}
            </p>
            {data.items.length === 0 ? (
              <Empty
                title="표시할 감사 기록이 없습니다"
                text="기간·동작 필터를 조정하세요. 현재 접근 권한이 없는 자료의 기록은 표시하지 않습니다."
              />
            ) : (
              <div className="panel audit-table">
                <table>
                  <thead>
                    <tr>
                      <th>동작과 시간</th>
                      <th>사용자</th>
                      <th>현재 자료</th>
                      <th>안전한 변경 정보</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((row) => (
                      <tr key={row.id}>
                        <td>
                          <strong>{label(row.action)}</strong>
                          <small>{datetime(row.created_at)}</small>
                        </td>
                        <td>{row.actor_name || "삭제된 계정"}</td>
                        <td>
                          <Badge>{kinds[row.resource_kind] || "자료"}</Badge>
                          {link(row) ? (
                            <Link to={link(row)!}>{row.resource_title}</Link>
                          ) : (
                            <span>{row.resource_title}</span>
                          )}
                        </td>
                        <td>
                          {Object.entries(row.changes).map(([key, value]) => (
                            <span className="audit-change" key={key}>
                              {(
                                {
                                  version: "버전",
                                  enabled: "활성",
                                  restored: "복원",
                                  rows: "행 수",
                                  status: "상태",
                                } as Record<string, string>
                              )[key] || key}
                              :{" "}
                              {typeof value === "boolean"
                                ? value
                                  ? "예"
                                  : "아니요"
                                : key === "status"
                                  ? {
                                      ...statusNames,
                                      approved: "승인됨",
                                      pending: "대기",
                                      running: "실행 중",
                                      succeeded: "완료",
                                      failed: "실패",
                                      cancelled: "취소됨",
                                    }[String(value)] || String(value)
                                  : String(value)}
                            </span>
                          ))}
                          {!Object.keys(row.changes).length && "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {data.next_cursor && (
              <Button
                variant="secondary"
                onClick={() =>
                  setParams((current) => {
                    const next = new URLSearchParams(current);
                    next.set("cursor", data.next_cursor);
                    return next;
                  })
                }
              >
                <History size={17} />
                이전 기록 100개
              </Button>
            )}
          </>
        )
      )}
    </div>
  );
}
