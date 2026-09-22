import { useEffect, useRef, useState } from "react";
import { BookmarkPlus } from "lucide-react";
import { api } from "../api";
import { useApp } from "../context";
import { Button, Field, Modal } from "../ui";
import { RecoveryNotice } from "../review/ChangeReview";
import { useModalReturnFocus } from "../review/useModalReturnFocus";
import type { WorksetItem, WorksetData } from "./types";
export default function SaveToWorkset({
  item,
  label = "작업 묶음에 보관",
  disabled = false,
}: {
  item: WorksetItem | (() => WorksetItem);
  label?: string;
  disabled?: boolean;
}) {
  const { user, workspace, notify } = useApp();
  const [open, setOpen] = useState(false),
    [snapshot, setSnapshot] = useState<WorksetItem | null>(null),
    [options, setOptions] = useState<any[]>([]),
    [target, setTarget] = useState(""),
    [name, setName] = useState("내 작업 묶음"),
    [kind, setKind] = useState("workset"),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null);
  const scope = `${user.id}:${workspace?.id}`,
    current = useRef(scope),
    opened = useRef("");
  current.current = scope;
  const trigger = useRef<HTMLElement | null>(null);
  const returnFocus = useModalReturnFocus(open, scope, trigger.current);
  const alive = useRef(true);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    setOpen(false);
    setSnapshot(null);
    setBusy(false);
    setError(null);
  }, [scope]);
  useEffect(() => {
    if (!open || !workspace) return;
    const controller = new AbortController();
    setOptions([]);
    setError(null);
    void api(`/worksets?workspace_id=${workspace.id}`, "GET", undefined, {
      signal: controller.signal,
    })
      .then((v) => {
        if (!controller.signal.aborted && current.current === scope)
          setOptions(v.items);
      })
      .catch((e) => {
        if (!controller.signal.aborted && current.current === scope)
          setError(e);
      });
    return () => controller.abort();
  }, [open, scope]);
  async function save() {
    if (!snapshot || !workspace || busy) return;
    setBusy(true);
    setError(null);
    try {
      if (target) {
        const data = await api<WorksetData>(`/worksets/${target}`);
        if (current.current !== scope || !alive.current) return;
        await api(`/worksets/${target}`, "PUT", {
          name: data.workset.name,
          kind: data.workset.kind,
          expected_version: data.workset.version,
          items: [...data.workset.items, snapshot],
        });
      } else {
        await api("/worksets", "POST", {
          workspace_id: workspace.id,
          name,
          kind,
          items: [snapshot],
        });
      }
      if (current.current === scope && alive.current) {
        setOpen(false);
        notify("원문을 복사하지 않고 개인 작업 묶음에 참조를 보관했습니다.");
      }
    } catch (e) {
      if (current.current === scope && alive.current) setError(e);
    } finally {
      if (current.current === scope && alive.current) setBusy(false);
    }
  }
  return (
    <>
      <Button
        type="button"
        disabled={disabled}
        title={
          disabled ? "문서 저장을 확인한 뒤 현재 위치를 보관하세요." : undefined
        }
        onMouseDown={(e) => e.preventDefault()}
        onClick={(event) => {
          trigger.current = event.currentTarget;
          setSnapshot(typeof item === "function" ? item() : item);
          setTarget("");
          setKind("workset");
          opened.current = scope;
          setOpen(true);
        }}
      >
        <BookmarkPlus size={18} />
        {label}
      </Button>
      <Modal
        open={open && opened.current === scope}
        onCloseAutoFocus={returnFocus}
        onOpenChange={(v) => {
          if (!busy) setOpen(v);
        }}
        title="개인 작업 묶음에 보관"
        description="문서나 공유 범위를 바꾸지 않고 내 계정에 위치·참조만 저장합니다."
      >
        <Field label="보관할 작업 묶음">
          <select
            value={target}
            disabled={busy}
            onChange={(e) => setTarget(e.target.value)}
          >
            <option value="">새 개인 묶음</option>
            {options
              .filter(
                (o) => o.kind !== "reference" || snapshot?.kind === "document",
              )
              .map((o) => (
                <option value={o.id} key={o.id}>
                  {o.name} · {o.item_count}개
                </option>
              ))}
          </select>
        </Field>
        {!target && (
          <>
            <Field label="새 묶음 이름">
              <input
                value={name}
                disabled={busy}
                maxLength={200}
                onChange={(e) => setName(e.target.value)}
              />
            </Field>
            <Field label="묶음 종류">
              <select
                value={kind}
                disabled={busy}
                onChange={(e) => setKind(e.target.value)}
              >
                <option value="workset">작업 묶음 · 최대20개</option>
                {snapshot?.kind === "document" && (
                  <option value="reference">참고 선반 · 문서 최대6개</option>
                )}
              </select>
            </Field>
          </>
        )}
        <RecoveryNotice error={error} />
        <div className="modal-actions">
          <Button type="button" disabled={busy} onClick={() => setOpen(false)}>
            취소
          </Button>
          <Button
            type="button"
            variant="primary"
            disabled={busy || (!target && !name.trim())}
            onClick={() => void save()}
          >
            참조 보관
          </Button>
        </div>
      </Modal>
    </>
  );
}
