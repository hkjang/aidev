import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  AlertTriangle,
  LoaderCircle,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { api } from "./api";
import { useApp } from "./context";
import { Button, ErrorBox } from "./ui";
import { pluginWorkerSource } from "./plugins/sdk";

export type Contribution = {
  id: string;
  title: string;
  description?: string;
  extensions?: string[];
  provider?: string;
};
export type PluginManifest = {
  id: string;
  name: string;
  version: string;
  api_version: number;
  description: string;
  author: string;
  entry: string;
  style?: string;
  capabilities: string[];
  contributions: Record<string, Contribution[]>;
};
export type PluginInfo = {
  id: string;
  manifest: PluginManifest;
  capabilities: string[];
  version: number;
  grant_version: number;
  enabled?: boolean;
  global_enabled?: boolean;
};
type Runtime = PluginInfo & {
  files: Record<string, { mime: string; data: string }>;
};
export type PluginFrameHandle = {
  invoke: (kind: string, id: string, input?: unknown) => Promise<any>;
};
type Props = {
  pluginId: string;
  workspaceId: string;
  onRegistered?: (kind: string, id: string) => void;
  initialData?: unknown;
  compact?: boolean;
};

export const PluginFrame = forwardRef<PluginFrameHandle, Props>(
  function PluginFrame(
    { pluginId, workspaceId, onRegistered, initialData, compact = false },
    ref,
  ) {
    const { user, notify } = useApp(),
      navigate = useNavigate();
    const [runtime, setRuntime] = useState<Runtime | null>(null),
      [error, setError] = useState(""),
      [running, setRunning] = useState(false),
      [revision, setRevision] = useState(0);
    const iframe = useRef<HTMLIFrameElement>(null),
      requests = useRef(new Map<string, AbortController>()),
      invocations = useRef(
        new Map<
          string,
          {
            resolve: (value: any) => void;
            reject: (e: Error) => void;
            timer: ReturnType<typeof setTimeout>;
          }
        >(),
      );
    const callbacks = useRef({
      onRegistered,
      notify,
      navigate,
      initialData,
      user,
    });
    callbacks.current = { onRegistered, notify, navigate, initialData, user };
    const nonce = useMemo(
      () => crypto.randomUUID().replaceAll("-", ""),
      [pluginId, workspaceId, revision],
    );
    const send = useCallback(
      (payload: Record<string, unknown>) =>
        iframe.current?.contentWindow?.postMessage(
          { channel: "madi-plugin-v1", nonce, ...payload },
          "*",
        ),
      [nonce],
    );
    const stop = useCallback(() => {
      for (const controller of requests.current.values()) controller.abort();
      requests.current.clear();
      for (const task of invocations.current.values()) {
        clearTimeout(task.timer);
        task.reject(new Error("플러그인 실행이 중단되었습니다"));
      }
      invocations.current.clear();
    }, []);
    useEffect(() => {
      let active = true;
      setRuntime(null);
      setError("");
      setRunning(false);
      api<Runtime>(`/plugins/${pluginId}/runtime?workspace_id=${workspaceId}`)
        .then((value) => {
          if (active) setRuntime(value);
        })
        .catch((e) => {
          if (active) setError(e.message);
        });
      return () => {
        active = false;
        stop();
      };
    }, [pluginId, workspaceId, revision, stop]);
    useEffect(() => {
      if (!runtime) return;
      let active = true;
      const timer = setInterval(async () => {
        try {
          const plugins = await api<PluginInfo[]>(
            `/plugins?workspace_id=${workspaceId}`,
          );
          if (!active) return;
          const latest = plugins.find((p) => p.id === pluginId);
          if (!latest) {
            stop();
            setRuntime(null);
            setError("플러그인 권한이 취소되어 실행을 중단했습니다.");
          } else if (
            latest.version !== runtime.version ||
            latest.grant_version !== runtime.grant_version
          ) {
            stop();
            setRevision((value) => value + 1);
          }
        } catch {
          if (active) {
            stop();
            setRuntime(null);
            setError(
              "현재 플러그인 권한을 확인할 수 없어 실행을 중단했습니다.",
            );
          }
        }
      }, 5000);
      return () => {
        active = false;
        clearInterval(timer);
      };
    }, [runtime, workspaceId, pluginId, stop]);
    useImperativeHandle(
      ref,
      () => ({
        invoke: (kind, id, input) =>
          new Promise((resolve, reject) => {
            if (!runtime || !running) {
              reject(new Error("플러그인이 아직 준비되지 않았습니다"));
              return;
            }
            if (
              !runtime.manifest.contributions[kind]?.some(
                (item) => item.id === id,
              )
            ) {
              reject(new Error("등록되지 않은 확장 기능입니다"));
              return;
            }
            if (invocations.current.size >= 16) {
              reject(new Error("동시 실행 한도를 초과했습니다"));
              return;
            }
            const callID = crypto.randomUUID();
            const timer = setTimeout(() => {
              invocations.current.delete(callID);
              reject(new Error("확장 기능 실행 시간이 초과되었습니다"));
            }, 180000);
            invocations.current.set(callID, { resolve, reject, timer });
            send({
              type: "invoke",
              id: callID,
              kind,
              contributionId: id,
              input,
            });
          }),
      }),
      [runtime, running, send],
    );
    useEffect(() => {
      if (!runtime) return;
      async function onMessage(event: MessageEvent) {
        const data = event.data;
        // Sandboxed frames have opaque origins. Source identity and an unguessable
        // per-mount nonce are required; accepting origin="null" alone is unsafe.
        if (
          event.source !== iframe.current?.contentWindow ||
          !data ||
          typeof data !== "object" ||
          data.channel !== "madi-plugin-v1" ||
          data.nonce !== nonce
        )
          return;
        let size = 0;
        try {
          size = JSON.stringify(data).length;
        } catch {
          return;
        }
        if (size > 2 << 20) {
          setError("플러그인 메시지 크기 한도를 초과했습니다");
          return;
        }
        if (data.type === "ready") {
          send({
            type: "init",
            workerSource: pluginWorkerSource(
              nonce,
              runtime!.files[runtime!.manifest.entry]?.data || "",
            ),
            style: runtime!.manifest.style
              ? runtime!.files[runtime!.manifest.style]?.data || ""
              : "",
            context: {
              workspace_id: workspaceId,
              user: {
                id: callbacks.current.user.id,
                name: callbacks.current.user.name,
              },
              manifest: runtime!.manifest,
              capabilities: runtime!.capabilities,
              assets: runtime!.files,
              initial_data: callbacks.current.initialData,
            },
          });
          setRunning(true);
          return;
        }
        if (data.type === "registered") {
          if (
            typeof data.kind === "string" &&
            typeof data.id === "string" &&
            runtime!.manifest.contributions[data.kind]?.some(
              (item) => item.id === data.id,
            )
          )
            callbacks.current.onRegistered?.(data.kind, data.id);
          return;
        }
        if (data.type === "plugin-error") {
          stop();
          setRunning(false);
          setRuntime(null);
          setError(
            String(data.error || "플러그인 오류가 발생했습니다").slice(0, 1000),
          );
          return;
        }
        if (data.type === "invocation-result") {
          const task = invocations.current.get(data.id);
          if (task) {
            clearTimeout(task.timer);
            invocations.current.delete(data.id);
            data.error
              ? task.reject(new Error(String(data.error).slice(0, 1000)))
              : task.resolve(data.result);
          }
          return;
        }
        if (data.type === "cancel") {
          requests.current.get(data.id)?.abort();
          return;
        }
        if (
          data.type !== "request" ||
          typeof data.id !== "string" ||
          data.id.length > 100 ||
          typeof data.operation !== "string" ||
          !data.args ||
          typeof data.args !== "object" ||
          Array.isArray(data.args)
        )
          return;
        if (requests.current.has(data.id) || requests.current.size >= 16) {
          send({
            type: "response",
            id: data.id,
            error: "동시 요청 한도를 초과했습니다",
          });
          return;
        }
        const controller = new AbortController();
        requests.current.set(data.id, controller);
        try {
          const response = await fetch(`/api/v1/plugins/${pluginId}/bridge`, {
            method: "POST",
            credentials: "same-origin",
            headers: {
              "Content-Type": "application/json",
              "X-Madi-Request": "1",
            },
            body: JSON.stringify({
              workspace_id: workspaceId,
              operation: data.operation,
              args: data.args,
            }),
            signal: controller.signal,
          });
          if (!response.ok) {
            const detail = await response
              .json()
              .catch(() => ({ error: `요청 실패 (${response.status})` }));
            throw new Error(
              detail.error || "플러그인 요청을 처리하지 못했습니다",
            );
          }
          let result: any;
          if (data.operation === "ai.chat") {
            const reader = response.body?.getReader();
            if (!reader) throw new Error("AI 응답 스트림을 열지 못했습니다");
            const decoder = new TextDecoder();
            let pending = "",
              text = "",
              sources: unknown[] = [],
              done = false;
            while (true) {
              const part = await reader.read();
              if (part.done) break;
              pending += decoder.decode(part.value, { stream: true });
              const lines = pending.split("\n");
              pending = lines.pop() || "";
              for (const line of lines) {
                if (!line.startsWith("data:")) continue;
                const raw = line.slice(5).trim();
                if (raw === "[DONE]") {
                  done = true;
                  continue;
                }
                if (!raw) continue;
                const chunk = JSON.parse(raw);
                if (chunk.error) throw new Error(chunk.error);
                if (chunk.text) {
                  text += chunk.text;
                  if (text.length > 2 << 20)
                    throw new Error(
                      "AI 출력이 플러그인 크기 한도를 초과했습니다",
                    );
                }
                if (chunk.sources) sources = chunk.sources;
                send({ type: "chunk", id: data.id, chunk });
              }
            }
            if (!done) throw new Error("AI 스트림이 중단되었습니다");
            result = { text, sources };
          } else result = await response.json();
          if (data.operation === "ui.notify")
            callbacks.current.notify(result.message);
          if (data.operation === "ui.navigate")
            callbacks.current.navigate(result.path);
          send({ type: "response", id: data.id, result });
        } catch (e) {
          send({
            type: "response",
            id: data.id,
            error:
              e instanceof Error ? e.message : "플러그인 요청에 실패했습니다",
          });
        } finally {
          requests.current.delete(data.id);
        }
      }
      window.addEventListener("message", onMessage);
      return () => window.removeEventListener("message", onMessage);
    }, [runtime, pluginId, workspaceId, nonce, send, stop]);
    return (
      <div className={`plugin-frame-wrapper ${compact ? "compact" : ""}`}>
        {!compact && (
          <div className="plugin-frame-bar">
            <span>
              <ShieldCheck size={16} /> 격리 실행 · 승인된 권한만 사용
            </span>
            <Button
              variant="ghost"
              onClick={() => {
                stop();
                setRevision((v) => v + 1);
              }}
            >
              <RefreshCw size={15} />
              다시 실행
            </Button>
          </div>
        )}
        {error && <ErrorBox error={error} />}
        {!runtime && !error && (
          <div className="empty-state">
            <LoaderCircle className="spin" />
            <p>플러그인 권한을 확인하고 있습니다.</p>
          </div>
        )}
        {runtime && (
          <iframe
            ref={iframe}
            title={`${runtime.manifest.name} 플러그인`}
            src={`/plugin-sandbox.html#${nonce}`}
            onLoad={() => send({ type: "probe" })}
            sandbox="allow-scripts"
            referrerPolicy="no-referrer"
            allow="camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'; payment 'none'; usb 'none'"
            className="plugin-frame"
          />
        )}
        {!runtime && error && (
          <div className="notice">
            <AlertTriangle size={18} /> 관리자가 플러그인을 다시 승인한 후
            재실행하세요.
          </div>
        )}
        <style>{`.plugin-frame-wrapper{border:1px solid var(--border);border-radius:12px;background:var(--surface);overflow:hidden}.plugin-frame-wrapper>.notice{margin:14px}.plugin-frame-bar{display:flex;align-items:center;justify-content:space-between;padding:10px 18px;border-bottom:1px solid var(--border);gap:12px}.plugin-frame-bar>span{display:flex;align-items:center;gap:8px;font-size:14px;color:var(--muted)}.plugin-frame{display:block;width:100%;height:590px;border:0;background:#fbfcf8}.plugin-frame-wrapper.compact .plugin-frame{height:340px}@media(max-width:640px){.plugin-frame-bar{padding:10px;flex-wrap:wrap}.plugin-frame{height:650px}}`}</style>
      </div>
    );
  },
);
