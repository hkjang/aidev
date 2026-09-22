import { useEffect, useRef, useState } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import { Home, Search, FilePlus2 } from "lucide-react";
import { useApp } from "../context";
import { api, type Doc } from "../api";
import "../personalization/mobile.css";
export default function MobileActions({
  blocked = false,
}: {
  blocked?: boolean;
}) {
  const { user, workspace, reload, notify } = useApp(),
    navigate = useNavigate();
  const [obscured, setObscured] = useState(false),
    [busy, setBusy] = useState(false);
  const scope = `${user.id}:${workspace?.id}`,
    current = useRef(scope),
    alive = useRef(true),
    request = useRef({ scope: "", id: "" });
  current.current = scope;
  const writable =
    !!workspace &&
    user.role !== "viewer" &&
    ["owner", "admin", "editor"].includes(workspace.role);
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  useEffect(() => {
    setBusy(false);
  }, [scope]);
  useEffect(() => {
    let frame = 0;
    const measure = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        const el = document.activeElement as HTMLElement | null;
        const editing = !!el?.matches(
          'input:not([type="checkbox"]):not([type="radio"]),textarea,[contenteditable="true"],[role="textbox"]',
        );
        const viewport = window.visualViewport;
        const keyboard =
          !!viewport &&
          viewport.scale === 1 &&
          window.innerHeight - viewport.height > 140;
        setObscured(
          editing ||
            keyboard ||
            !!document.querySelector(
              '[role="dialog"][aria-modal="true"], [role="alertdialog"], [role="menu"][data-state="open"]',
            ),
        );
      });
    };
    const observer = new MutationObserver(measure);
    observer.observe(document.body, { childList: true, subtree: true });
    document.addEventListener("focusin", measure);
    document.addEventListener("focusout", measure);
    window.visualViewport?.addEventListener("resize", measure);
    measure();
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      document.removeEventListener("focusin", measure);
      document.removeEventListener("focusout", measure);
      window.visualViewport?.removeEventListener("resize", measure);
    };
  }, []);
  useEffect(() => {
    document.documentElement.dataset.mobileActions =
      blocked || obscured ? "hidden" : "visible";
    return () => {
      delete document.documentElement.dataset.mobileActions;
    };
  }, [blocked, obscured]);
  const capture = async () => {
    if (!workspace || !writable || busy) return;
    if (request.current.scope !== scope)
      request.current = { scope, id: crypto.randomUUID() };
    setBusy(true);
    try {
      const doc = await api<Doc>("/captures", "POST", {
        workspace_id: workspace.id,
        title: "새 개인 메모",
        text: "",
        client_request_id: request.current.id,
      });
      if (!alive.current || current.current !== scope) return;
      request.current = { scope: "", id: "" };
      await reload();
      if (!alive.current || current.current !== scope) return;
      navigate(`/app/documents/${doc.id}?mode=edit`);
      notify("나만 볼 수 있는 메모를 열었습니다.");
    } catch (e) {
      if (alive.current && current.current === scope)
        notify((e as Error).message, "error");
    } finally {
      if (alive.current && current.current === scope) setBusy(false);
    }
  };
  return (
    <nav
      className="mobile-core-actions"
      aria-label="모바일 핵심 동작"
      hidden={blocked || obscured}
    >
      <NavLink to="/app" end>
        <Home size={21} />
        <span>홈</span>
      </NavLink>
      <NavLink to="/app/search">
        <Search size={21} />
        <span>검색</span>
      </NavLink>
      <button
        type="button"
        disabled={!writable || busy}
        onClick={() => void capture()}
      >
        <FilePlus2 size={21} />
        <span>{busy ? "여는 중…" : "개인 메모"}</span>
      </button>
    </nav>
  );
}
