import { useEffect, useRef, useState } from "react";
import { api, type User } from "../api";
import { useApp } from "../context";
import "./style.css";

// Serialize background preference patches so an older response cannot overwrite
// a more recent local navigation choice. The server merges only supplied keys.
const queues = new Map<string, Promise<unknown>>();
export function patchPreferences(
  userID: string,
  patch: Record<string, unknown>,
): Promise<User> {
  const pending = (queues.get(userID) || Promise.resolve())
    .catch(() => {})
    .then(() =>
      api<User>("/profile", "PUT", {
        preferences: patch,
        expected_user_id: userID,
      }),
    );
  queues.set(userID, pending);
  void pending
    .finally(() => {
      if (queues.get(userID) === pending) queues.delete(userID);
    })
    .catch(() => {});
  return pending;
}
export function usePreferenceWriter() {
  const { user, setUser } = useApp();
  const active = useRef(true),
    actor = useRef(user.id);
  actor.current = user.id;
  useEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  return async (patch: Record<string, unknown>) => {
    const id = user.id;
    const updated = await patchPreferences(id, patch);
    if (active.current && actor.current === id && updated.id === id)
      setUser(updated);
    return updated;
  };
}
export function useSidebarState() {
  const { user, setUser, notify } = useApp();
  const write = usePreferenceWriter();
  const [collapsed, setLocal] = useState(!!user.preferences?.sidebar_collapsed);
  useEffect(
    () => setLocal(!!user.preferences?.sidebar_collapsed),
    [user.id, user.preferences?.sidebar_collapsed],
  );
  const setCollapsed = (value: boolean) => {
    setLocal(value);
    void write({ sidebar_collapsed: value }).catch((e) =>
      notify(e.message, "error"),
    );
  };
  return [collapsed, setCollapsed] as const;
}
export function SidebarResize() {
  const { user, setUser, notify } = useApp();
  const write = usePreferenceWriter();
  const width = Math.min(
    420,
    Math.max(220, Number(user.preferences?.sidebar_width) || 268),
  );
  const draft = useRef(width);
  const [dragging, setDragging] = useState(false);
  const save = (value: number) =>
    void write({ sidebar_width: value }).catch((e) =>
      notify(e.message, "error"),
    );
  return (
    <div
      role="separator"
      aria-label="사이드바 너비 조정"
      aria-orientation="vertical"
      aria-valuemin={220}
      aria-valuemax={420}
      aria-valuenow={width}
      tabIndex={0}
      className={`sidebar-resizer ${dragging ? "dragging" : ""}`}
      onPointerDown={(e) => {
        if (e.button !== 0) return;
        draft.current = width;
        setDragging(true);
        e.currentTarget.setPointerCapture(e.pointerId);
      }}
      onPointerMove={(e) => {
        if (!dragging) return;
        draft.current = Math.min(420, Math.max(220, Math.round(e.clientX)));
        e.currentTarget
          .closest<HTMLElement>(".app-layout")
          ?.style.setProperty("--sidebar-width", `${draft.current}px`);
      }}
      onPointerUp={(e) => {
        if (!dragging) return;
        setDragging(false);
        e.currentTarget.releasePointerCapture(e.pointerId);
        save(draft.current);
      }}
      onPointerCancel={() => {
        setDragging(false);
        save(draft.current);
      }}
      onKeyDown={(e) => {
        if (e.key === "ArrowLeft" || e.key === "ArrowRight") {
          e.preventDefault();
          save(
            Math.min(
              420,
              Math.max(220, width + (e.key === "ArrowLeft" ? -10 : 10)),
            ),
          );
        }
      }}
    />
  );
}
