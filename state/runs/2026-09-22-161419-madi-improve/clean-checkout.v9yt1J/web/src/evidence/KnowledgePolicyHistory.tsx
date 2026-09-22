import { useEffect, useRef, useState } from "react";
import { api, datetime } from "../api";
import { Button, ErrorBox, Modal } from "../ui";
type Entry = {
  version: number;
  enabled: boolean;
  retention_days?: number;
  retention_hours?: number;
  token_counter?: string;
  allow_http?: boolean;
  created_at: string;
};
export default function KnowledgePolicyHistory({
  kind,
  version,
  onRestored,
}: {
  kind: "evidence" | "packages";
  version: number;
  onRestored: () => Promise<void>;
}) {
  const url =
    kind === "evidence"
      ? "/admin/evidence-policy"
      : "/admin/knowledge-packages/policy";
  const [history, setHistory] = useState<Entry[]>([]),
    [selected, setSelected] = useState<Entry | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    generation.current++;
    setSelected(null);
    setBusy(false);
    setHistory([]);
    let active = true;
    void api<Entry[]>(url + "/history")
      .then((value) => {
        if (active) {
          setHistory(value);
          setError("");
        }
      })
      .catch((e: Error) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
      generation.current++;
    };
  }, [url, version]);
  const restore = async () => {
    if (!selected || busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await api(`${url}/history/${selected.version}/restore`, "POST", {
        version,
        consent: true,
      });
      if (request === generation.current) {
        setSelected(null);
        await onRestored();
      }
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  return (
    <section className="card">
      <h2>설정 변경 이력</h2>
      <ErrorBox error={error} />
      <p>
        최근 100개 버전입니다. 이전 값으로 돌아갈 때도 새 버전을 기록하며,
        만료·삭제된 사본을 복구하지 않습니다.
      </p>
      {history.map((row) => (
        <div className="evidence-review" key={row.version}>
          <strong>
            v{row.version} · {row.enabled ? "활성화" : "비활성화"}
          </strong>{" "}
          · {datetime(row.created_at)}
          <p>
            {row.retention_days !== undefined
              ? `보존 ${row.retention_days}일`
              : `보존 ${row.retention_hours}시간 · ${row.token_counter === "responses" ? "모델 계산 허용" : "로컬 추정만"} · ${row.allow_http ? "HTTP 허용" : "HTTPS만"}`}
          </p>
          <Button
            disabled={busy || row.version === version}
            onClick={() => setSelected(row)}
          >
            이 값으로 새 설정 저장
          </Button>
        </div>
      ))}
      <Modal
        open={!!selected}
        onOpenChange={(open) => {
          if (!open && !busy) setSelected(null);
        }}
        title="이전 설정값으로 새 버전 저장"
      >
        <p>
          현재 설정 v{version}에서 이력 v{selected?.version}의 값으로
          변경합니다. 활성 여부와 모델 계산 허용도 함께 바뀝니다. 보존기간
          축소는 기존 사본에 즉시 적용하며, 확대해도 이미 만료된 사본은 되살리지
          않습니다.
        </p>
        <Button
          disabled={busy}
          variant="primary"
          onClick={() => void restore()}
        >
          변경 확인·설정 저장
        </Button>
      </Modal>
    </section>
  );
}
