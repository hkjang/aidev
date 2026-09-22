import { useEffect, useMemo, useRef, useState } from "react";
import { Play, Puzzle } from "lucide-react";
import { useApp } from "./context";
import { Button, ErrorBox } from "./ui";
import { PluginFrame, type PluginFrameHandle } from "./PluginFrame";

export default function PluginBlockPreview({
  source,
  workspaceId,
}: {
  source: string;
  workspaceId?: string;
}) {
  const { workspace } = useApp();
  const wid = workspaceId || workspace?.id;
  const parsed = useMemo(() => {
    try {
      if (source.length > 65536) return null;
      const value = JSON.parse(source);
      if (
        !value ||
        typeof value !== "object" ||
        !/^[a-z][a-z0-9-]{2,63}$/.test(value.plugin_id) ||
        !/^[a-z][a-z0-9_-]{0,63}$/.test(value.block_id)
      )
        return null;
      return value as { plugin_id: string; block_id: string; data: unknown };
    } catch {
      return null;
    }
  }, [source]);
  const [started, setStarted] = useState(false),
    [registered, setRegistered] = useState(false),
    [error, setError] = useState("");
  const frame = useRef<PluginFrameHandle>(null);
  useEffect(() => {
    setStarted(false);
    setRegistered(false);
    setError("");
  }, [source, wid]);
  useEffect(() => {
    let active = true;
    if (registered && parsed)
      frame.current
        ?.invoke("blocks", parsed.block_id, parsed.data)
        .catch((e) => {
          if (active) setError(e.message);
        });
    return () => {
      active = false;
    };
  }, [registered, parsed]);
  return (
    <div className="plugin-block-preview">
      <ErrorBox error={error} />
      {!parsed ? (
        <div className="notice error">
          플러그인 블록 JSON의 plugin_id·block_id·data 형식을 확인하세요.
        </div>
      ) : started && wid ? (
        <PluginFrame
          ref={frame}
          pluginId={parsed.plugin_id}
          workspaceId={wid}
          initialData={parsed.data}
          compact
          onRegistered={(kind, id) => {
            if (kind === "blocks" && id === parsed.block_id)
              setRegistered(true);
          }}
        />
      ) : (
        <div className="notice">
          <Puzzle size={20} />
          <div>
            <strong>플러그인 블록 · {parsed.plugin_id}</strong>
            <p>
              확장은 자동 실행되지 않습니다. 승인된 권한으로 격리 실행하려면
              버튼을 누르세요.
            </p>
            <Button
              variant="secondary"
              disabled={!wid}
              onClick={() => setStarted(true)}
            >
              <Play size={16} />
              플러그인 블록 실행
            </Button>
          </div>
        </div>
      )}
      <details>
        <summary>원본 플러그인 블록</summary>
        <pre>
          <code>{source}</code>
        </pre>
      </details>
      <style>{`.plugin-block-preview{margin:20px 0}.plugin-block-preview>.notice{align-items:flex-start}.plugin-block-preview>.notice p{font-size:14px;line-height:1.7;margin:8px 0}.plugin-block-preview details{font-size:13px;color:var(--muted);margin-top:10px}.plugin-block-preview summary{cursor:pointer}.plugin-block-preview pre{max-height:240px;overflow:auto}`}</style>
    </div>
  );
}
