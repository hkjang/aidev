import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { Button, ErrorBox, Modal } from "../ui";
import "./diagnostics.css";

type Diagnostics = {
  version: number;
  epoch: string;
  sequence: number;
  snapshot_sequence: number;
  snapshot_bytes: number;
  pending_updates: number;
  pending_bytes: number;
  active_connections: number;
  can_compact: boolean;
  listener_status: string;
  snapshot_at: string | null;
  reset_reason: string;
};
type Group = {
  id: string;
  first_version: number;
  last_version: number;
  changes: number;
  user_name: string;
  updated_at: string;
};
const bytes = (value: number) =>
  `${(value / 1024).toLocaleString("ko-KR", { maximumFractionDigits: 1 })} KiB`;

export default function CollaborationDiagnostics({
  documentId,
  open,
  onOpenChange,
}: {
  documentId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [data, setData] = useState<Diagnostics | null>(null),
    [groups, setGroups] = useState<Group[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [confirmed, setConfirmed] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    const run = ++generation.current;
    setData(null);
    setGroups([]);
    setError(null);
    setConfirmed(false);
    setBusy(false);
    if (!open) return;
    let stopped = false;
    const refresh = async () => {
      try {
        const [next, history] = await Promise.all([
          api<Diagnostics>(
            `/documents/${documentId}/collaboration/diagnostics`,
          ),
          api<Group[]>(`/documents/${documentId}/collaboration/history`),
        ]);
        if (stopped || generation.current !== run) return;
        setData((previous) => {
          if (
            previous?.epoch !== next.epoch ||
            previous?.version !== next.version
          )
            setConfirmed(false);
          return next;
        });
        setGroups(history);
        setError(null);
      } catch (e) {
        if (!stopped && generation.current === run) {
          setData(null);
          setGroups([]);
          setConfirmed(false);
          setError(e);
        }
      }
    };
    void refresh();
    const timer = setInterval(() => {
      if (document.visibilityState === "visible") void refresh();
    }, 5000);
    return () => {
      stopped = true;
      clearInterval(timer);
      generation.current++;
    };
  }, [documentId, open]);
  async function compact() {
    if (!data || !confirmed || busy) return;
    const run = generation.current;
    setBusy(true);
    setError(null);
    try {
      await api(`/documents/${documentId}/collaboration/compact`, "POST", {
        expected_version: data.version,
        expected_epoch: data.epoch,
        confirm: true,
      });
      if (generation.current !== run) return;
      const next = await api<Diagnostics>(
        `/documents/${documentId}/collaboration/diagnostics`,
      );
      if (generation.current === run) {
        setConfirmed(false);
        setData(next);
      }
    } catch (e) {
      if (generation.current === run) setError(e);
    } finally {
      if (generation.current === run) setBusy(false);
    }
  }
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="공동 편집 진단"
      description="서버 확정 상태와 장시간 편집 이력을 확인합니다. 본문과 개별 문서 버전은 압축으로 삭제되지 않습니다."
    >
      <ErrorBox error={error} />
      {data && (
        <div className="collaboration-diagnostics">
          <dl>
            <dt>내용 버전 / 저장 순서</dt>
            <dd>
              v{data.version} / {data.sequence}
            </dd>
            <dt>체크포인트</dt>
            <dd>
              순서 {data.snapshot_sequence} · {bytes(data.snapshot_bytes)}
            </dd>
            <dt>이후 증분 로그</dt>
            <dd>
              {data.pending_updates}개 · {bytes(data.pending_bytes)}
            </dd>
            <dt>현재 연결</dt>
            <dd>{data.active_connections}개</dd>
            <dt>변경 알림 연결</dt>
            <dd>
              {{
                connected: "연결됨",
                reconnecting: "재연결 중 · DB 재조회 유지",
                idle: "유휴 상태",
              }[data.listener_status] || data.listener_status}
            </dd>
          </dl>
          <p className="muted">
            증분 64개·1 MiB 또는 다음 저장 시 60초 경과를 기준으로 체크포인트를
            만듭니다. 알림 유실 시 DB 순서로 복구하며, 권한 검사는 별도로
            반복합니다. 검사 간격은 최대 지연 보장이 아닙니다.
          </p>
          {data.can_compact && data.epoch && (
            <section>
              <h3>편집 이력 압축</h3>
              <p>
                새 편집 기준을 만듭니다. 연결된 모든 편집자는 초안을 보관한 뒤
                다시 연결해야 합니다. 오래된 오프라인 초안은 자동 병합하지
                않습니다.
              </p>
              <label className="checkbox-row">
                <input
                  type="checkbox"
                  checked={confirmed}
                  onChange={(e) => setConfirmed(e.target.checked)}
                />
                미확정 초안을 먼저 다운로드했으며 다른 편집자의 재연결이
                필요함을 확인했습니다.
              </label>
              <Button
                disabled={!confirmed || busy}
                onClick={() => void compact()}
              >
                {busy ? "압축 중…" : "원문을 보존하고 편집 이력 압축"}
              </Button>
            </section>
          )}
          <h3>연속 공동 편집 이력</h3>
          <p className="muted">
            같은 작성자의 연속 변경을 30초 간격·최대 5분으로 묶습니다. 개별 버전
            조회와 복원은 문서 변경 이력에서 계속 사용할 수 있습니다.
          </p>
          {groups.length ? (
            <ul>
              {groups.map((g) => (
                <li key={g.id}>
                  {g.user_name} · v{g.first_version}–v{g.last_version} ·{" "}
                  {g.changes}회 ·{" "}
                  {new Date(g.updated_at).toLocaleString("ko-KR")}
                </li>
              ))}
            </ul>
          ) : (
            <p>아직 공동 편집 변경 이력이 없습니다.</p>
          )}
        </div>
      )}
    </Modal>
  );
}
