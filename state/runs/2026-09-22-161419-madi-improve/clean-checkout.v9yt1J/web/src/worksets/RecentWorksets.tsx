import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Layers3 } from "lucide-react";
import { api } from "../api";
import { useApp } from "../context";
import { RecoveryNotice } from "../review/ChangeReview";
export default function RecentWorksets() {
  const { user, workspace } = useApp(),
    scope = `${user.id}:${workspace?.id}`;
  const [state, setState] = useState<{
    scope: string;
    items: { id: string; name: string; kind: string; item_count: number }[];
  }>({ scope: "", items: [] });
  const [error, setError] = useState<unknown>(null);
  useEffect(() => {
    const ctl = new AbortController();
    setError(null);
    if (workspace)
      void api(`/worksets?workspace_id=${workspace.id}`, "GET", undefined, {
        signal: ctl.signal,
      })
        .then((data) => {
          if (!ctl.signal.aborted)
            setState({ scope, items: data.items.slice(0, 3) });
        })
        .catch((e) => {
          if (!ctl.signal.aborted) {
            setState({ scope, items: [] });
            setError(e);
          }
        });
    return () => ctl.abort();
  }, [scope]);
  const items = state.scope === scope ? state.items : [];
  return (
    <section className="panel padded" aria-label="최근 작업 묶음">
      <div className="section-heading">
        <h2>
          <Layers3 size={21} />
          작업 묶음 이어가기
        </h2>
        <Link to="/app/worksets">내 묶음 모두 보기</Link>
      </div>
      <p className="muted">
        문서 위치·데이터베이스 보기·할 일을 함께 보관한 개인 업무 맥락입니다.
      </p>
      <RecoveryNotice error={error} />
      <div className="home-template-list">
        {items.map((item) => (
          <Link key={item.id} to={`/app/worksets/${item.id}`}>
            <Layers3 size={20} />
            <strong>{item.name}</strong>
            <small>
              {item.kind === "reference" ? "참고 선반" : "작업 묶음"} ·{" "}
              {item.item_count}개 참조
            </small>
          </Link>
        ))}
        {!items.length && (
          <Link to="/app/worksets">
            <Layers3 size={20} />
            <strong>나만의 작업 묶음 만들기</strong>
            <small>
              원문이나 공개 범위를 바꾸지 않고 필요한 참조만 모으세요.
            </small>
          </Link>
        )}
      </div>
    </section>
  );
}
