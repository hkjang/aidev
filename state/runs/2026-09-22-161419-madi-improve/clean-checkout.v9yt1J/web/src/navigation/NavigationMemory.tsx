import { useEffect, useRef } from "react";
import { Link, useLocation } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { useApp } from "../context";

type Position = {
  y: number;
  x: number;
  selection?: [number, number];
  editorY?: number;
  version?: string;
  range?: {
    start: number[];
    end: number[];
    startOffset: number;
    endOffset: number;
  };
  at: number;
};
const relevant = (path: string) =>
  /^\/app\/(documents\/[^/]+|graph|tasks|evidence)(?:\/|$)/.test(path);
const safeOrigin = (value: unknown): value is string =>
  typeof value === "string" &&
  /^\/app\/documents\/[a-f0-9-]+(?:\?[^#]*)?(?:#[^]*)?$/i.test(value) &&
  value.length < 2000;
function read(key: string): Record<string, Position> {
  try {
    const parsed = JSON.parse(sessionStorage.getItem(key) || "{}");
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed))
      return {};
    return Object.fromEntries(
      Object.entries(parsed).filter(
        ([route, v]) =>
          route.length < 2000 &&
          relevant(route) &&
          !!v &&
          typeof v === "object" &&
          ["x", "y", "at"].every(
            (field) =>
              Number.isFinite((v as Record<string, number>)[field]) &&
              (v as Record<string, number>)[field] >= 0,
          ),
      ),
    ) as Record<string, Position>;
  } catch {
    return {};
  }
}
function nodePath(root: Node, node: Node): number[] | null {
  const result: number[] = [];
  let current: Node | null = node;
  while (current && current !== root) {
    const parent: ParentNode | null = current.parentNode;
    if (!parent || result.length >= 32) return null;
    result.unshift(Array.prototype.indexOf.call(parent.childNodes, current));
    current = parent;
  }
  return current === root ? result : null;
}
function rangeSnapshot() {
  const root = document.querySelector(".editor-area"),
    selection = window.getSelection();
  if (!root || !selection?.rangeCount) return undefined;
  const range = selection.getRangeAt(0),
    start = nodePath(root, range.startContainer),
    end = nodePath(root, range.endContainer);
  return start && end
    ? { start, end, startOffset: range.startOffset, endOffset: range.endOffset }
    : undefined;
}
function restoreRange(value: NonNullable<Position["range"]>) {
  const root = document.querySelector(".editor-area");
  if (!root) return false;
  const find = (path: number[]) =>
    path.reduce<Node | null>(
      (node, index) => node?.childNodes[index] || null,
      root,
    );
  const start = find(value.start),
    end = find(value.end);
  if (!start || !end) return false;
  try {
    const range = document.createRange();
    range.setStart(
      start,
      Math.min(
        value.startOffset,
        start.nodeType === Node.TEXT_NODE
          ? start.textContent?.length || 0
          : start.childNodes.length,
      ),
    );
    range.setEnd(
      end,
      Math.min(
        value.endOffset,
        end.nodeType === Node.TEXT_NODE
          ? end.textContent?.length || 0
          : end.childNodes.length,
      ),
    );
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
    return true;
  } catch {
    return false;
  }
}
export function captureWorksetDocumentContext(version: number) {
  const source =
    document.querySelector<HTMLTextAreaElement>(".markdown-source");
  const line = source
    ? source.value.slice(0, source.selectionStart).split("\n").length
    : Number(new URLSearchParams(location.search).get("line")) || undefined;
  return {
    version,
    mode: "read",
    scroll_y: Math.max(0, Math.round(window.scrollY)),
    ...(line ? { line } : {}),
    ...(!source && rangeSnapshot() ? { range: rangeSnapshot() } : {}),
  };
}
export function restoreWorksetPosition(
  actor: string,
  workspace: string,
  route: string,
  value: Record<string, any>,
) {
  if (!safeOrigin(route)) return;
  const key = `madi.position.${actor}.${workspace}`,
    all = read(key);
  all[route] = {
    at: Date.now(),
    x: 0,
    y: Number(value.scroll_y) || 0,
    version: String(value.version || ""),
    ...(value.range ? { range: value.range } : {}),
  };
  try {
    sessionStorage.setItem(key, JSON.stringify(all));
  } catch {}
}
/** Positions and numeric selection ranges only. Never cache document or AI content. */
export function useNavigationMemory() {
  const { user, workspace } = useApp(),
    location = useLocation();
  const route = location.pathname + location.search + location.hash;
  const documentID = /^\/app\/documents\/([^/]+)$/.exec(location.pathname)?.[1];
  const key = `madi.position.${user.id}.${workspace?.id || ""}`;
  const current = useRef({ key, route });
  current.current = { key, route };
  useEffect(() => {
    if (!relevant(location.pathname)) return;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const save = (departing = false) => {
      // A queued timer can run after a navigation commit but before this old
      // passive effect is cleaned up. Never replace its document's position
      // with a loading screen or the next route's DOM. A real popstate/pagehide
      // may capture the outgoing DOM after the browser URL has already moved.
      if (
        current.current.key !== key ||
        current.current.route !== route ||
        (!departing &&
          window.location.pathname +
            window.location.search +
            window.location.hash !==
            route)
      )
        return;
      const documentElement =
        document.querySelector<HTMLElement>("[data-document-id]");
      if (
        documentID &&
        (documentElement?.dataset.documentId !== documentID ||
          !documentElement.dataset.documentVersion)
      )
        return;
      const input =
        document.querySelector<HTMLTextAreaElement>(".markdown-source");
      const all = read(key);
      const version = document.querySelector<HTMLElement>(
        "[data-document-version]",
      )?.dataset.documentVersion;
      all[route] = {
        x: window.scrollX,
        y: window.scrollY,
        at: Date.now(),
        version,
        range:
          rangeSnapshot() ||
          (version === all[route]?.version ? all[route]?.range : undefined),
        ...(input
          ? {
              selection: [input.selectionStart, input.selectionEnd] as [
                number,
                number,
              ],
              editorY: input.scrollTop,
            }
          : {}),
      };
      const trimmed = Object.fromEntries(
        Object.entries(all)
          .filter(([, v]) => Date.now() - v.at < 86400000)
          .sort((a, b) => b[1].at - a[1].at)
          .slice(0, 60),
      );
      try {
        sessionStorage.setItem(key, JSON.stringify(trimmed));
      } catch {}
    };
    const schedule = () => {
      clearTimeout(timer);
      timer = setTimeout(save, 100);
    };
    const saveDeparture = () => save(true);
    const depart = (event: MouseEvent) => {
      const anchor = (event.target as Element)?.closest<HTMLAnchorElement>(
        "a[href]",
      );
      if (!anchor) return;
      const target = new URL(anchor.href, window.location.origin);
      if (target.origin !== window.location.origin) return;
      save();
      if (
        /^\/app\/documents\/[^/]+$/.test(location.pathname) &&
        /^\/app\/(graph|tasks|evidence)(?:\/|$)/.test(target.pathname)
      ) {
        try {
          sessionStorage.setItem(key + ".origin", route);
        } catch {}
      }
    };
    document.addEventListener("scroll", schedule, true);
    document.addEventListener("selectionchange", schedule);
    document.addEventListener("click", depart, true);
    window.addEventListener("pagehide", saveDeparture);
    window.addEventListener("popstate", saveDeparture);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("scroll", schedule, true);
      document.removeEventListener("selectionchange", schedule);
      document.removeEventListener("click", depart, true);
      window.removeEventListener("pagehide", saveDeparture);
      window.removeEventListener("popstate", saveDeparture);
    };
  }, [key, route]);
  useEffect(() => {
    if (
      !relevant(location.pathname) ||
      location.hash ||
      new URLSearchParams(location.search).has("line")
    )
      return;
    const remembered = read(key)[route];
    if (!remembered || Date.now() - remembered.at > 86400000) return;
    let cancelled = false,
      selectionRestored = false,
      done = false;
    const restore = () => {
      if (
        cancelled ||
        done ||
        current.current.key !== key ||
        current.current.route !== route ||
        window.location.pathname +
          window.location.search +
          window.location.hash !==
          route
      )
        return;
      const documentPage = location.pathname.startsWith("/app/documents/");
      if (documentPage && !document.querySelector(".document-body")) return;
      if (documentID) {
        const documentElement =
          document.querySelector<HTMLElement>("[data-document-id]");
        if (
          documentElement?.dataset.documentId !== documentID ||
          !documentElement.dataset.documentVersion
        )
          return;
      }
      const input =
        document.querySelector<HTMLTextAreaElement>(".markdown-source");
      if (remembered.selection && documentPage && !input) return;
      const sameVersion =
        remembered.version ===
        document.querySelector<HTMLElement>("[data-document-version]")?.dataset
          .documentVersion;
      // A taller viewport or shorter side panel can make the old window scroll
      // position unreachable. That must not block a valid same-version editor
      // selection, or repeatedly overwrite it while later layout is loading.
      if (!selectionRestored) {
        if (sameVersion && remembered.range && !restoreRange(remembered.range))
          return;
        selectionRestored = true;
        if (sameVersion && input && remembered.selection) {
          const [start, end] = remembered.selection;
          input.setSelectionRange(
            Math.min(start, input.value.length),
            Math.min(end, input.value.length),
          );
          input.scrollTop = remembered.editorY || 0;
        }
      }
      const maxScroll = Math.max(
        0,
        document.documentElement.scrollHeight - window.innerHeight,
      );
      window.scrollTo(remembered.x, Math.min(remembered.y, maxScroll));
      if (maxScroll < remembered.y - 4) return;
      done = true;
      observer.disconnect();
    };
    const observer = new MutationObserver(restore);
    observer.observe(document.body, { childList: true, subtree: true });
    const frame = requestAnimationFrame(restore);
    const timeout = setTimeout(() => {
      cancelled = true;
      observer.disconnect();
    }, 8000);
    const stop = () => {
      cancelled = true;
      observer.disconnect();
    };
    window.addEventListener("wheel", stop, { once: true });
    window.addEventListener("touchstart", stop, { once: true });
    window.addEventListener("resize", restore);
    return () => {
      cancelled = true;
      cancelAnimationFrame(frame);
      clearTimeout(timeout);
      observer.disconnect();
      window.removeEventListener("wheel", stop);
      window.removeEventListener("touchstart", stop);
      window.removeEventListener("resize", restore);
    };
  }, [key, route]);
}
export function NavigationReturn() {
  const { user, workspace } = useApp(),
    location = useLocation();
  if (!/^\/app\/(graph|tasks|evidence)(?:\/|$)/.test(location.pathname))
    return null;
  let origin: unknown;
  try {
    origin = sessionStorage.getItem(
      `madi.position.${user.id}.${workspace?.id || ""}.origin`,
    );
  } catch {}
  return safeOrigin(origin) ? (
    <Link className="navigation-return text-button" to={origin}>
      <ArrowLeft size={17} />
      문서로 돌아가기 <span>이전 위치 복원</span>
    </Link>
  ) : null;
}
