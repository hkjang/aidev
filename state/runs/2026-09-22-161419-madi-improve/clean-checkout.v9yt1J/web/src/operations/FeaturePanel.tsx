import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { History, Save, RefreshCw, ShieldCheck } from "lucide-react";
import { api, datetime, type User } from "../api";
import { useApp } from "../context";
import { Badge, Button, ErrorBox, Field, Loading, Modal } from "../ui";
import {
  leaveOperations,
  refreshPresentation,
  useOperationMounted,
  useOperationsGuard,
} from "./shared";

type Feature = {
  id: string;
  name: string;
  description: string;
  stop_policy: string;
};
type Snapshot = {
  data: Record<string, boolean>;
  version?: number;
  revision?: string;
};
export function FeaturePanel({ workspaceID }: { workspaceID?: string }) {
  const { user, notify } = useApp(),
    mounted = useOperationMounted();
  const generation = useRef(0);
  const [users, setUsers] = useState<User[]>([]),
    [target, setTarget] = useState("service"),
    [catalogue, setCatalogue] = useState<Feature[]>([]),
    [snapshot, setSnapshot] = useState<Snapshot | null>(null),
    [draft, setDraft] = useState<Record<string, boolean>>({}),
    [effective, setEffective] = useState<Record<string, boolean>>({}),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true),
    [history, setHistory] = useState<Record<string, any>[] | null>(null);
  const userMode = !workspaceID && target !== "service",
    dirty =
      !!snapshot && JSON.stringify(snapshot.data) !== JSON.stringify(draft);
  useOperationsGuard(dirty);
  const path = workspaceID
    ? `/workspaces/${workspaceID}/settings`
    : userMode
      ? `/admin/features/users/${target}`
      : "/admin/settings";
  async function load() {
    const ticket = ++generation.current;
    setLoading(true);
    setError(null);
    try {
      const [settings, features] = await Promise.all([
        api(path),
        api(
          workspaceID
            ? `/workspaces/${workspaceID}/features`
            : "/admin/features/catalogue",
        ),
      ]);
      if (!mounted.current || ticket !== generation.current) return;
      const data =
        (userMode
          ? settings.data
          : workspaceID
            ? settings.data?.feature_flags
            : settings.feature_flags) || {};
      setSnapshot({
        data,
        version: settings.version,
        revision: settings.settings_revision,
      });
      setDraft({ ...data });
      setCatalogue(workspaceID ? features.catalogue : features);
      setEffective(workspaceID ? features.effective : {});
    } catch (e) {
      if (mounted.current && ticket === generation.current) setError(e);
    } finally {
      if (mounted.current && ticket === generation.current) setLoading(false);
    }
  }
  useEffect(() => {
    void load();
    return () => {
      generation.current++;
    };
  }, [path]);
  useEffect(() => {
    if (workspaceID) return;
    let active = true;
    api<User[]>("/admin/users")
      .then((data) => {
        if (active) setUsers(data);
      })
      .catch((e) => {
        if (active) setError(e);
      });
    return () => {
      active = false;
    };
  }, [workspaceID]);
  async function save() {
    if (!snapshot) return;
    setBusy(true);
    setError(null);
    try {
      await api(
        path,
        "PUT",
        workspaceID
          ? { version: snapshot.version, data: { feature_flags: draft } }
          : userMode
            ? { version: snapshot.version, data: draft }
            : {
                feature_flags: draft,
                expected_settings_revision: snapshot.revision,
              },
      );
      if (!mounted.current) return;
      refreshPresentation();
      notify("기능 정책을 저장했습니다");
      await load();
    } catch (e) {
      if (mounted.current) setError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  return (
    <section className="operations-feature">
      <div className="notice">
        <ShieldCheck size={20} />
        <span>
          서비스 → 워크스페이스 → 사용자 정책을 모두 확인합니다. 한 단계라도
          꺼짐이면 실행할 수 없습니다. 하위 단계의 켜짐은 상위 차단이나 문서
          권한을 우회하지 않습니다.
        </span>
      </div>
      {!workspaceID && (
        <Field label="정책 적용 대상">
          <select
            value={target}
            disabled={busy}
            onChange={(e) => {
              if (leaveOperations()) {
                setTarget(e.target.value);
                setHistory(null);
              }
            }}
          >
            <option value="service">서비스 전체 기본 정책</option>
            {users.map((u) => (
              <option key={u.id} value={u.id}>
                {u.name} · {u.email}
                {u.id === user.id ? " (나)" : ""}
              </option>
            ))}
          </select>
        </Field>
      )}
      <div className="operations-toolbar">
        <p className="muted">
          {workspaceID
            ? "이 워크스페이스에만 적용합니다. 실제 적용 표시는 현재 로그인 사용자 기준입니다."
            : userMode
              ? "관리자가 지정하는 사용자 제한입니다. 사용자는 개인화 화면에서 이 정책을 변경할 수 없습니다."
              : "기본 기능은 모두 켜져 있습니다. 정책을 끄더라도 문서·캔버스·설치 파일 등 원본을 삭제하지 않습니다."}
        </p>
        <div className="actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => {
              if (!dirty || leaveOperations()) void load();
            }}
          >
            <RefreshCw size={16} />
            다시 불러오기
          </Button>
          {userMode ? (
            <Button
              variant="secondary"
              disabled={busy}
              onClick={async () => {
                try {
                  setHistory(await api(path + "/history"));
                } catch (e) {
                  setError(e);
                }
              }}
            >
              <History size={16} />
              설정 이력
            </Button>
          ) : (
            <Link
              className="button secondary"
              to={
                workspaceID
                  ? "/app/workspace-settings"
                  : "/admin/settings?tab=history"
              }
              onClick={(e) => {
                if (!leaveOperations()) e.preventDefault();
              }}
            >
              <History size={16} />
              설정 이력
            </Link>
          )}
        </div>
      </div>
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        <>
          <div className="operations-feature-grid">
            {catalogue.map((f) => (
              <article
                className="panel padded operations-feature-card"
                key={f.id}
              >
                <div className="operations-card-title">
                  <h2>{f.name}</h2>
                  {workspaceID && (
                    <Badge tone={effective[f.id] ? "green" : "amber"}>
                      {effective[f.id] ? "현재 사용 가능" : "현재 비활성"}
                    </Badge>
                  )}
                </div>
                <p>{f.description}</p>
                <Field label={`${f.name} 정책`}>
                  <select
                    disabled={busy}
                    value={
                      Object.hasOwn(draft, f.id)
                        ? draft[f.id]
                          ? "on"
                          : "off"
                        : "inherit"
                    }
                    onChange={(e) => {
                      const value = e.target.value;
                      setDraft((old) => {
                        const next = { ...old };
                        if (value === "inherit") delete next[f.id];
                        else next[f.id] = value === "on";
                        return next;
                      });
                    }}
                  >
                    <option value="inherit">
                      {workspaceID || userMode
                        ? "상위 정책 상속"
                        : "기본값 사용 (켜짐)"}
                    </option>
                    <option value="on">켜짐 · 기존 권한 유지</option>
                    <option value="off">꺼짐 · 실행 차단</option>
                  </select>
                </Field>
                <small>{f.stop_policy}</small>
              </article>
            ))}
          </div>
          <footer className="operations-save">
            <span className="muted">
              {dirty
                ? "저장하지 않은 변경사항이 있습니다."
                : "모든 변경사항이 저장되어 있습니다."}
            </span>
            <Button disabled={busy || !dirty} onClick={() => void save()}>
              <Save size={17} />
              {busy ? "저장 중…" : "기능 정책 저장"}
            </Button>
          </footer>
        </>
      )}
      <Modal
        title="사용자 기능 정책 이력"
        description="복원하면 새 버전으로 기록합니다. 다른 관리자가 변경한 경우 자동으로 덮어쓰지 않습니다."
        open={history !== null}
        onOpenChange={(open) => {
          if (!open) setHistory(null);
        }}
      >
        {history?.length === 0 ? (
          <p>저장된 이력이 없습니다.</p>
        ) : (
          history?.map((h) => (
            <div className="operations-history" key={h.version}>
              <div>
                <strong>버전 {h.version}</strong>
                <p className="muted">
                  {datetime(h.created_at)} · {h.actor_name}
                </p>
                <small>
                  {catalogue
                    .filter((f) => Object.hasOwn(h.data || {}, f.id))
                    .map((f) => `${f.name}: ${h.data[f.id] ? "켜짐" : "꺼짐"}`)
                    .join(", ") || "모든 기능 상속"}
                </small>
              </div>
              <Button
                variant="secondary"
                disabled={busy || h.version === snapshot?.version}
                onClick={async () => {
                  if (
                    !window.confirm(
                      `버전 ${h.version}의 기능 정책을 복원할까요?`,
                    )
                  )
                    return;
                  setBusy(true);
                  try {
                    await api(path + `/history/${h.version}/restore`, "POST", {
                      version: snapshot?.version,
                    });
                    if (!mounted.current) return;
                    setHistory(null);
                    refreshPresentation();
                    await load();
                    notify("기능 정책 이력을 복원했습니다");
                  } catch (e) {
                    setError(e);
                  } finally {
                    if (mounted.current) setBusy(false);
                  }
                }}
              >
                복원
              </Button>
            </div>
          ))
        )}
      </Modal>
    </section>
  );
}
