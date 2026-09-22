import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, PageHeading } from "../ui";
import type { Policy } from "./types";
import { formatTime } from "./types";
import "./system-status.css";
type Settings = {
  policy: Policy;
  history: { revision: number; policy: Policy; created_at: string }[];
  notice: string;
};
export default function SystemStatusAdminPage() {
  const { user } = useApp();
  const actor = useRef(user?.id);
  actor.current = user?.id;
  const [state, setState] = useState<Settings | null>(null),
    [draft, setDraft] = useState<Policy | null>(null),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [confirm, setConfirm] = useState(false),
    [busy, setBusy] = useState(false);
  const seq = useRef(0);
  const load = useCallback(async () => {
    const uid = user?.id,
      token = ++seq.current;
    setError("");
    try {
      const value = await api<Settings>("/admin/system-status/policy");
      if (uid !== actor.current || token !== seq.current) return;
      setState(value);
      setDraft(value.policy);
      setConfirm(false);
    } catch (e) {
      if (uid !== actor.current || token !== seq.current) return;
      setState(null);
      setDraft(null);
      setError(
        e instanceof Error ? e.message : "보고 정책을 불러오지 못했습니다",
      );
    }
  }, [user?.id]);
  useEffect(() => {
    setState(null);
    setDraft(null);
    setConfirm(false);
    setNotice("");
    void load();
    return () => {
      seq.current++;
    };
  }, [load]);
  const change = (patch: Partial<Policy>) => {
    setDraft((v) => (v ? { ...v, ...patch } : v));
    setConfirm(false);
  };
  const save = async () => {
    if (!draft || !confirm || busy || !state) return;
    const uid = user?.id;
    setBusy(true);
    setError("");
    try {
      await api("/admin/system-status/policy", "PUT", {
        ...draft,
        confirm: true,
      });
      if (uid !== actor.current) return;
      await load();
      if (uid === actor.current)
        setNotice(
          "보고 정책을 저장했습니다. 기존 기대값·관측이 실제 배포 확인으로 바뀌지는 않습니다.",
        );
    } catch (e) {
      if (uid === actor.current) {
        setError(e instanceof Error ? e.message : "저장하지 못했습니다");
        setConfirm(false);
      }
    } finally {
      if (uid === actor.current) setBusy(false);
    }
  };
  return (
    <section className="system-status-page">
      <PageHeading
        title="시스템 운영 보고 정책"
        eyebrow="SERVICE ADMIN"
        description="보고 수신과 신선도·보존 범위를 설정합니다. 대상 시스템에 직접 접속하거나 명령을 실행하지 않습니다."
        actions={
          <Link className="button" to="/app/system-status">
            운영 카드로 이동
          </Link>
        }
      />
      <ErrorBox error={error} />
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      {!draft ? (
        error ? (
          <Button onClick={() => void load()}>다시 불러오기</Button>
        ) : (
          <Loading />
        )
      ) : (
        <>
          <div className="status-authority-notice">
            <p>{state?.notice}</p>
          </div>
          <div className="status-policy-form">
            <label className="checkbox-row">
              <input
                type="checkbox"
                checked={draft.enabled}
                disabled={busy}
                onChange={(e) => change({ enabled: e.target.checked })}
              />
              운영 카드 등록과 관측 보고 수신을 허용합니다
            </label>
            <Field label="최대 관측 TTL · 초">
              <input
                type="number"
                min={60}
                max={604800}
                value={draft.max_ttl_seconds}
                disabled={busy}
                onChange={(e) =>
                  change({ max_ttl_seconds: Number(e.target.value) })
                }
              />
            </Field>
            <Field label="허용할 최대 과거 관측 · 초">
              <input
                type="number"
                min={60}
                max={604800}
                value={draft.max_observation_age_seconds}
                disabled={busy}
                onChange={(e) =>
                  change({
                    max_observation_age_seconds: Number(e.target.value),
                  })
                }
              />
            </Field>
            <Field label="관측 보고 보존 · 일">
              <input
                type="number"
                min={8}
                max={3650}
                value={draft.retention_days}
                disabled={busy}
                onChange={(e) =>
                  change({ retention_days: Number(e.target.value) })
                }
              />
            </Field>
            <p className="muted">
              TTL과 과거 관측 범위는 60초~7일입니다. 미래 관측은 서버 시각 오차
              60초까지만 허용하고 만료는 관측+TTL과 수신+TTL 중 이른 시각입니다.
              관측 보고는 8~3650일 보존하며 유지관리 배치에서 만료 자료를
              삭제합니다. 보존 축소로 삭제된 이력은 백업이 없다면 복구할 수
              없습니다.
            </p>
            <p className="muted">
              전역 허용만으로 보고 권한이 생기지 않습니다. 문서별 작성자가 보고
              주체를 지정하고 계정·키·워크스페이스·문서 권한을 모두 만족해야
              합니다. 비활성화하면 과거 이력을 현재 상태로 확인하지 않습니다.
              백업 복원 후에는 정책이 꺼지고 모든 카드의 검증 세대가 갱신됩니다.
            </p>
            <label className="checkbox-row">
              <input
                type="checkbox"
                checked={confirm}
                disabled={busy}
                onChange={(e) => setConfirm(e.target.checked)}
              />
              보고 수신·만료·보존 범위와 삭제 영향을 확인했습니다.
            </label>
            <div className="status-actions">
              <Button
                variant="primary"
                disabled={!confirm || busy}
                onClick={() => void save()}
              >
                {busy ? "저장 중…" : "보고 정책 저장"}
              </Button>
              <Button disabled={busy} onClick={() => void load()}>
                현재 정책 다시 읽기
              </Button>
            </div>
          </div>
          <h2>정책 변경 이력</h2>
          <div className="status-history-list">
            {state?.history.map((item) => (
              <article className="status-history" key={item.revision}>
                <strong>
                  revision {item.revision} ·{" "}
                  {item.policy.enabled ? "보고 허용" : "보고 비활성"}
                </strong>
                <p>
                  최대 TTL {item.policy.max_ttl_seconds}초 · 최대 과거{" "}
                  {item.policy.max_observation_age_seconds}초 · 보존{" "}
                  {item.policy.retention_days}일
                </p>
                <time>{formatTime(item.created_at)}</time>
              </article>
            ))}
          </div>
        </>
      )}
    </section>
  );
}
