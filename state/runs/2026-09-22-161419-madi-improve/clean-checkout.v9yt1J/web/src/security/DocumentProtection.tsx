import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { api } from "../api";
import { useApp } from "../context";
import { RecoveryNotice } from "../review/ChangeReview";
import "./style.css";
const classes: Record<string, string> = {
  public: "공개",
  internal: "내부",
  confidential: "기밀",
  restricted: "제한",
};
export function DocumentWatermark({
  viewer,
  classification,
  at,
}: {
  viewer: string;
  classification: string;
  at?: string;
}) {
  const [time, setTime] = useState(() => new Date(at || Date.now()));
  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 30000);
    return () => clearInterval(timer);
  }, []);
  const text = `${viewer} · ${classes[classification] || classification} · ${time.toLocaleString("ko-KR")}`;
  return (
    <div className="protection-watermark" aria-hidden="true">
      {text}
      <br />
      {text}
      <br />
      {text}
    </div>
  );
}
export default function DocumentProtection({
  documentID,
  version,
}: {
  documentID: string;
  version?: number;
}) {
  const { user, workspace } = useApp();
  const scope = `${user.id}:${workspace?.id}:${documentID}:${version}`;
  const [data, setData] = useState<{ scope: string; value: Record<string, any> } | null>(null),
    [error, setError] = useState<{ scope: string; value: unknown } | null>(null),
    [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true, running = false;
    const controller = new AbortController();
    const load = () => {
      if (running) return;
      running = true;
      return api<Record<string, any>>(`/documents/${documentID}/protection`, "GET", undefined, { signal: controller.signal })
        .then((v) => {
          if (active) {
            setData({ scope, value: v });
            setError(null);
          }
        })
        .catch((e) => {
          if (active) {
            setData(null);
            setError({ scope, value: e instanceof TypeError ? new Error("네트워크 연결을 확인한 뒤 정보보호 정책을 다시 확인하세요. 확인 실패를 보호 정책 해제로 취급하지 않습니다.") : e });
          }
        }).finally(() => { running = false; });
    };
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => {
      active = false;
      controller.abort();
      clearInterval(timer);
    };
  }, [scope, documentID, attempt]);
  if (error?.scope === scope)
    return <RecoveryNotice error={error.value} onRetry={() => setAttempt((n) => n + 1)} onReview={() => setAttempt((n) => n + 1)} />;
  if (data?.scope !== scope) return null;
  const policy = data.value;
  return (
    <>
      <div className="protection-summary">
        <ShieldCheck size={16} />
        <span>상속 적용 등급: {classes[policy.classification]}</span>
        {policy.watermark && <span>화면·인쇄 워터마크</span>}
        {policy.enabled && (
          <span>
            민감정보{" "}
            {
              (
                {
                  warn: "경고",
                  block: "차단",
                  mask: "마스킹",
                  audit: "감사",
                } as Record<string, string>
              )[policy.mode]
            }
          </span>
        )}
      </div>
      {policy.watermark && (
        <DocumentWatermark
          viewer={policy.viewer}
          classification={policy.classification}
        />
      )}
    </>
  );
}
