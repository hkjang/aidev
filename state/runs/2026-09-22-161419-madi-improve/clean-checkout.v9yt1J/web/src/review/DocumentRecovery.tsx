import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Undo2, X } from "lucide-react";
import { api, type Doc } from "../api";
import { useApp } from "../context";
import { Button } from "../ui";
import { RecoveryNotice } from "./ChangeReview";
import "./document-recovery.css";
type TrashedDocument = {
  id: string;
  version: number;
  title: string;
  actor: string;
  workspace: string;
};
export function offerDocumentUndo(doc: TrashedDocument) {
  window.dispatchEvent(
    new CustomEvent("madi:document-trashed", { detail: doc }),
  );
}
export default function DocumentRecovery() {
  const { user, workspace, reload, notify } = useApp(),
    [doc, setDoc] = useState<TrashedDocument | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null);
  const scope = `${user.id}:${workspace?.id}`,
    active = useRef(scope),
    generation = useRef(0);
  active.current = scope;
  useEffect(() => {
    setDoc(null);
    setBusy(false);
    setError(null);
    generation.current++;
    const onTrash = (event: Event) => {
      const d = (event as CustomEvent).detail as TrashedDocument;
      if (
        !d ||
        d.actor !== user.id ||
        d.workspace !== workspace?.id ||
        typeof d.title !== "string" ||
        !Number.isInteger(d.version) ||
        d.version < 1 ||
        !/^[a-f0-9-]{36}$/i.test(d.id)
      )
        return;
      generation.current++;
      setDoc(d);
      setBusy(false);
      setError(null);
    };
    window.addEventListener("madi:document-trashed", onTrash);
    return () => {
      generation.current++;
      window.removeEventListener("madi:document-trashed", onTrash);
    };
  }, [scope]);
  const close = () => {
    generation.current++;
    setDoc(null);
    setError(null);
  };
  useEffect(() => {
    if (!doc || busy) return;
    const timer = setTimeout(close, 120000);
    return () => clearTimeout(timer);
  }, [doc, busy]);
  if (!doc || doc.actor !== user.id || doc.workspace !== workspace?.id)
    return null;
  const undo = async () => {
    if (busy) return;
    const revision = generation.current;
    setBusy(true);
    setError(null);
    const current = () =>
      active.current === scope && generation.current === revision;
    try {
      await api<Doc>(`/documents/${doc.id}/restore`, "POST", {
        expected_version: doc.version,
      });
      if (!current()) return;
      await reload();
      if (!current()) return;
      close();
      notify("휴지통 이동을 취소하고 문서를 복원했습니다.");
    } catch (e) {
      if (current()) setError(e);
    } finally {
      if (current()) setBusy(false);
    }
  };
  return (
    <section className="document-recovery" aria-label="휴지통 이동 실행 취소">
      <header>
        <div role="status">
          <strong>문서를 휴지통으로 이동했습니다.</strong>
          <span>{doc.title}</span>
        </div>
        <button
          className="icon-button"
          aria-label="실행 취소 안내 닫기"
          disabled={busy}
          onClick={close}
        >
          <X size={18} />
        </button>
      </header>
      <p>
        삭제 직후의 버전만 되돌립니다. 다른 변경이 있으면 자동 복원하지
        않습니다.
      </p>
      {error ? <RecoveryNotice error={error} /> : null}
      <footer>
        <Link to="/app/trash" onClick={close}>
          휴지통 확인
        </Link>
        <Button disabled={busy || !!error} onClick={() => void undo()}>
          <Undo2 size={17} />
          {busy ? "현재 버전 확인 중…" : "휴지통 이동 실행 취소"}
        </Button>
      </footer>
    </section>
  );
}
