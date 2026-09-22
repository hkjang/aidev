import { useEffect, useRef, useState } from "react";
import { api, ApiError, type Property } from "../api";
import { useApp } from "../context";
import { Button, Field, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import type { DatabaseFilter, DatabaseSort } from "../DatabaseAdvanced";
import "./editing.css";
export type ViewState = {
  view: string;
  filters: DatabaseFilter[];
  sorts: DatabaseSort[];
  columns: string[];
  board_property_id: string;
  date_property_id: string;
};
type SavedView = {
  id: string;
  name: string;
  visibility: "private" | "workspace";
  data: ViewState;
  version: number;
  can_edit: boolean;
  compatible: boolean;
};
export default function DatabaseViews({
  databaseId,
  properties,
  current,
  selectedId,
  canWrite,
  onApply,
  onSelect,
  onDefault,
}: {
  databaseId: string;
  properties: Property[];
  current: ViewState;
  selectedId: string;
  canWrite: boolean;
  onApply: (data: ViewState, id: string) => void;
  onSelect: (id: string) => void;
  onDefault: (data: ViewState, id: string) => void;
}) {
  const { user, workspace, notify } = useApp(),
    origin = useRef(`${user.id}:${workspace?.id}:${databaseId}`),
    mounted = useRef(true),
    controller = useRef<AbortController | null>(null),
    first = useRef(true);
  const busyRef = useRef(false),
    selectedRef = useRef(selectedId);
  selectedRef.current = selectedId;
  const [views, setViews] = useState<SavedView[]>([]),
    [defaultId, setDefaultId] = useState(""),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [open, setOpen] = useState(false),
    [name, setName] = useState(""),
    [visibility, setVisibility] = useState("private"),
    [consent, setConsent] = useState(false),
    [saveMode, setSaveMode] = useState("new"),
    [snapshot, setSnapshot] = useState(current),
    [targetSnapshot, setTargetSnapshot] = useState<SavedView | null>(null),
    [remove, setRemove] = useState<SavedView | null>(null);
  const scope = `${user.id}:${workspace?.id}:${databaseId}`,
    scopeRef = useRef(scope);
  scopeRef.current = scope;
  const alive = () => mounted.current && scopeRef.current === origin.current;
  const selected = views.find((v) => v.id === selectedId);
  const load = async () => {
    controller.current?.abort();
    const abort = new AbortController();
    controller.current = abort;
    try {
      const result = await api<{ views: SavedView[]; default_view_id: string }>(
        `/databases/${databaseId}/views`,
        "GET",
        undefined,
        { signal: abort.signal },
      );
      if (!alive() || abort.signal.aborted) return;
      setViews(result.views);
      setDefaultId(result.default_view_id);
      if (
        selectedRef.current &&
        !result.views.some((v) => v.id === selectedRef.current)
      )
        onSelect("");
      if (first.current) {
        first.current = false;
        const v = result.views.find(
          (v) => v.id === result.default_view_id && v.compatible,
        );
        if (v) onDefault(v.data, v.id);
      }
    } catch (e) {
      if (alive() && !abort.signal.aborted) {
        setViews([]);
        setDefaultId("");
        setError(e);
      }
    }
  };
  useEffect(() => {
    mounted.current = true;
    void load();
    const timer = setInterval(() => {
      if (!busyRef.current) void load();
    }, 5000);
    return () => {
      mounted.current = false;
      controller.current?.abort();
      clearInterval(timer);
    };
  }, []);
  const run = async (fn: () => Promise<void>) => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      if (alive()) setError(e);
    } finally {
      busyRef.current = false;
      if (alive()) setBusy(false);
    }
  };
  if (scope !== origin.current) return null;
  return (
    <section className="database-views" aria-label="개인 및 팀 보기">
      <div className="database-view-actions">
        <Field label="저장된 보기">
          <select
            value={views.some((v) => v.id === selectedId) ? selectedId : ""}
            onChange={(e) => {
              const v = views.find((v) => v.id === e.target.value);
              if (v?.compatible) onApply(v.data, v.id);
              else if (!e.target.value) onSelect("");
            }}
          >
            <option value="">임시 보기 · 공유 설정은 바뀌지 않음</option>
            {views.map((v) => (
              <option key={v.id} value={v.id} disabled={!v.compatible}>
                {v.name} ·{" "}
                {v.visibility === "private" ? "나만 보기" : "팀 보기"}
                {!v.compatible ? " · 속성 재확인 필요" : ""}
                {defaultId === v.id ? " · 내 기본" : ""}
              </option>
            ))}
          </select>
        </Field>
        <Button
          onClick={() => {
            setName(selected?.name || "내 작업 보기");
            setVisibility("private");
            setSaveMode("new");
            setTargetSnapshot(selected ? structuredClone(selected) : null);
            setSnapshot(structuredClone(current));
            setConsent(false);
            setError(null);
            setOpen(true);
          }}
        >
          현재 보기 저장
        </Button>
        <Button
          disabled={!selected || busy || !selected.compatible}
          onClick={() =>
            void run(async () => {
              await api(`/databases/${databaseId}/view-preference`, "PUT", {
                view_id: selectedId,
              });
              if (!alive()) return;
              setDefaultId(selectedId);
              notify("이 데이터베이스의 내 기본 보기를 저장했습니다.");
            })
          }
        >
          내 기본 보기로 설정
        </Button>
        {defaultId && (
          <Button
            disabled={busy}
            onClick={() =>
              void run(async () => {
                await api(`/databases/${databaseId}/view-preference`, "PUT", {
                  view_id: "",
                });
                if (alive()) setDefaultId("");
              })
            }
          >
            내 기본 보기 해제
          </Button>
        )}
        {selected?.can_edit && (
          <Button disabled={busy} onClick={() => setRemove({ ...selected })}>
            선택한 보기 삭제
          </Button>
        )}
      </div>
      <p className="muted">
        탭·필터·정렬 변경은 현재 화면에만 적용됩니다. 팀 보기로 명시적으로
        저장하기 전에는 다른 사용자의 구성을 바꾸지 않습니다.
      </p>
      <details>
        <summary>표시 속성과 보드·달력 기준</summary>
        <div className="database-view-columns">
          {properties.map((p) => (
            <label key={p.id}>
              <input
                type="checkbox"
                checked={
                  !current.columns.length || current.columns.includes(p.id)
                }
                onChange={(e) => {
                  const before = current.columns.length
                      ? current.columns
                      : properties.map((p) => p.id),
                    columns = e.target.checked
                      ? [...before, p.id]
                      : before.filter((id) => id !== p.id);
                  if (!columns.length) {
                    notify("표시 속성을 하나 이상 유지하세요.", "error");
                    return;
                  }
                  onApply({ ...current, columns }, selectedId);
                }}
              />
              {p.name}
            </label>
          ))}
        </div>
        <div className="form-grid">
          <Field label="보드 상태 속성">
            <select
              value={current.board_property_id}
              onChange={(e) =>
                onApply(
                  { ...current, board_property_id: e.target.value },
                  selectedId,
                )
              }
            >
              <option value="">첫 선택/상태 속성 자동 선택</option>
              {properties
                .filter((p) => ["select", "status"].includes(p.type))
                .map((p) => (
                  <option value={p.id} key={p.id}>
                    {p.name}
                  </option>
                ))}
            </select>
          </Field>
          <Field label="달력 날짜 속성">
            <select
              value={current.date_property_id}
              onChange={(e) =>
                onApply(
                  { ...current, date_property_id: e.target.value },
                  selectedId,
                )
              }
            >
              <option value="">첫 날짜 속성 자동 선택</option>
              {properties
                .filter((p) => p.type === "date")
                .map((p) => (
                  <option value={p.id} key={p.id}>
                    {p.name}
                  </option>
                ))}
            </select>
          </Field>
        </div>
        {!properties.some((p) => ["select", "status"].includes(p.type)) && (
          <p className="notice subtle">
            보드 후보가 없습니다. 선택 또는 상태 속성을 먼저 추가하세요.
          </p>
        )}
        {!properties.some((p) => p.type === "date") && (
          <p className="notice subtle">
            달력 후보가 없습니다. 날짜 속성을 먼저 추가하세요.
          </p>
        )}
      </details>
      {!!error && !open && !remove && (
        <RecoveryNotice
          error={error}
          status={error instanceof ApiError ? error.status : undefined}
        />
      )}
      <Modal
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
        title="보기 구성 저장"
        description="보기는 필터·정렬·표시 속성과 레이아웃만 저장합니다. 문서나 행의 접근 권한을 변경하지 않습니다."
      >
        {!!error && (
          <RecoveryNotice
            error={error}
            status={error instanceof ApiError ? error.status : undefined}
          />
        )}
        <Field label="보기 저장 방식">
          <select
            value={saveMode}
            disabled={busy}
            onChange={(e) => {
              setSaveMode(e.target.value);
              if (e.target.value === "update" && targetSnapshot)
                setVisibility(targetSnapshot.visibility);
              setConsent(false);
            }}
          >
            <option value="new">새 보기로 저장</option>
            {targetSnapshot?.can_edit && (
              <option value="update">
                내가 만든 선택 보기 업데이트 · 버전 확인
              </option>
            )}
          </select>
        </Field>
        <Field label="보기 이름">
          <input
            maxLength={65}
            value={name}
            disabled={busy}
            onChange={(e) => setName(e.target.value)}
          />
        </Field>
        <Field label="보기 공개 범위">
          <select
            value={visibility}
            disabled={busy}
            onChange={(e) => {
              setVisibility(e.target.value);
              setConsent(false);
            }}
          >
            <option value="private">나만 보기</option>
            <option value="workspace" disabled={!canWrite}>
              팀 보기 · 데이터베이스 접근 가능한 사용자
            </option>
          </select>
        </Field>
        {visibility === "workspace" && (
          <label className="check-label">
            <input
              type="checkbox"
              checked={consent}
              disabled={busy}
              onChange={(e) => setConsent(e.target.checked)}
            />
            현재 필터·정렬·표시 구성을 팀에 공유합니다. 원문이나 행 권한은
            확대하지 않습니다.
          </label>
        )}
        <div className="modal-actions">
          <Button disabled={busy} onClick={() => setOpen(false)}>
            취소
          </Button>
          <Button
            disabled={
              busy || !name.trim() || (visibility === "workspace" && !consent)
            }
            onClick={() =>
              void run(async () => {
                const target =
                  saveMode === "update" ? targetSnapshot : undefined;
                const saved = await api<SavedView>(
                  `/databases/${databaseId}/views${target ? "/" + target.id : ""}`,
                  target ? "PUT" : "POST",
                  {
                    name,
                    visibility,
                    data: snapshot,
                    share_consent: visibility === "workspace" && consent,
                    ...(target ? { expected_version: target.version } : {}),
                  },
                );
                if (!alive()) return;
                await load();
                if (!alive()) return;
                onApply(saved.data, saved.id);
                setOpen(false);
                notify("보기 구성을 저장했습니다.");
              })
            }
          >
            확인한 보기 저장
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!remove}
        onOpenChange={(value) => {
          if (!value && !busy) setRemove(null);
        }}
        title="저장된 보기 삭제"
      >
        {!!error && (
          <RecoveryNotice
            error={error}
            status={error instanceof ApiError ? error.status : undefined}
          />
        )}
        {remove && (
          <ChangeReview
            title={`${remove.name} 보기 삭제`}
            changes={[
              {
                label: "영향",
                before:
                  remove.visibility === "workspace"
                    ? "팀에서 사용하는 보기"
                    : "나만 사용하는 보기",
                after: "보기만 삭제 · 문서/행 데이터는 유지",
              },
            ]}
            warnings={["이 보기를 기본으로 사용하던 개인 설정은 해제됩니다."]}
            busy={busy}
            confirmLabel="보기 삭제"
            onCancel={() => setRemove(null)}
            onConfirm={() =>
              void run(async () => {
                await api(
                  `/databases/${databaseId}/views/${remove.id}`,
                  "DELETE",
                  { expected_version: remove.version },
                );
                if (!alive()) return;
                onSelect("");
                setRemove(null);
                await load();
              })
            }
          />
        )}
      </Modal>
    </section>
  );
}
