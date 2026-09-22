import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api, bytes, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Loading, Modal } from "../ui";
import { ApprovalReview } from "../approval/ApprovalPanel";
import { type ApprovalStatus, statusNames } from "../approval/types";

type Review = {
  id: string;
  owner_id: string;
  status: string;
  manifest_sha256: string;
  notice: string;
  approval: ApprovalStatus;
  manifest: {
    receiver_instance: string;
    key_id: string;
    expires_at: number;
    files: {
      source_id: string;
      parent_source_id: string;
      kind: string;
      source_version: number;
      path: string;
      sha256: string;
      bytes: number;
      metadata: { title: string };
    }[];
  };
};
type Confirmation = {
  path: string;
  body: Record<string, unknown>;
  title: string;
  hash: string;
};

export default function DistributionReview({ id }: { id: string }) {
  const { user, workspace } = useApp();
  const [record, setRecord] = useState<Review | null>(null),
    [error, setError] = useState("");
  const [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const [review, setReview] = useState(""),
    [confirm, setConfirm] = useState<Confirmation | null>(null),
    [consent, setConsent] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    const g = ++generation.current,
      controller = new AbortController();
    let active = true,
      running = false,
      signature = "";
    setRecord(null);
    setBusy(false);
    setError("");
    setReview("");
    setConfirm(null);
    setConsent(false);
    const load = async () => {
      if (running) return;
      running = true;
      try {
        const next = await api<Review>(
          `/knowledge/distribution/exports/${id}/review`,
          "GET",
          undefined,
          { signal: controller.signal },
        );
        if (!active || g !== generation.current) return;
        const changed = `${next.manifest_sha256}:${next.status}:${next.approval.request?.id}:${next.approval.request?.version}:${next.approval.stale}`;
        if (signature && signature !== changed) {
          setConfirm(null);
          setConsent(false);
        }
        signature = changed;
        setRecord(next);
        setError("");
      } catch (e) {
        if (active && g === generation.current) {
          setRecord(null);
          setReview("");
          setConfirm(null);
          setConsent(false);
          setError((e as Error).message);
        }
      } finally {
        running = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      active = false;
      generation.current++;
      controller.abort();
      clearInterval(timer);
    };
  }, [id, user.id, workspace?.id, refresh]);
  const execute = async () => {
    if (
      !record ||
      !confirm ||
      !consent ||
      busy ||
      confirm.hash !== record.manifest_sha256
    )
      return;
    const g = generation.current;
    setBusy(true);
    setError("");
    try {
      await api(
        `/knowledge/distribution/exports/${id}/${confirm.path}`,
        "POST",
        confirm.body,
      );
      if (g === generation.current) {
        setConfirm(null);
        setConsent(false);
        setRefresh((v) => v + 1);
      }
    } catch (e) {
      if (g === generation.current) {
        setError((e as Error).message);
        setConsent(false);
      }
    } finally {
      if (g === generation.current) setBusy(false);
    }
  };
  const request = record?.approval.request;
  return (
    <section className="card">
      <h2>배포 전체 검토</h2>
      <Link to="/app/knowledge-distribution">배포 목록으로</Link>
      <ErrorBox error={error} />
      {!record && !error && <Loading />}
      {record && (
        <>
          <p className="notice">{record.notice}</p>
          <p>
            수신망{" "}
            <code className="distribution-id">
              {record.manifest.receiver_instance}
            </code>
          </p>
          <p>
            서명 키{" "}
            <code className="distribution-id">{record.manifest.key_id}</code> ·
            유효기간{" "}
            {datetime(
              new Date(record.manifest.expires_at * 1000).toISOString(),
            )}
          </p>
          <p>
            검토 상태:{" "}
            {request ? statusNames[request.status] || request.status : "미제출"}
            {record.approval.stale && " · 현재 자료로 다시 검토 필요"}
          </p>
          <p>{record.approval.reason}</p>
          <details>
            <summary>승인으로 고정할 매니페스트 SHA-256</summary>
            <code className="evidence-hash">{record.manifest_sha256}</code>
          </details>
          <div className="evidence-list">
            {record.manifest.files.map((f) => (
              <article className="card" key={f.source_id}>
                <h3>
                  {f.kind === "document" ? "문서" : "첨부"} · {f.metadata.title}
                </h3>
                <p>
                  {f.kind === "document"
                    ? `문서 v${f.source_version}`
                    : "원본 파일 전체"}{" "}
                  · {bytes(f.bytes)}
                </p>
                <code className="evidence-hash">{f.path}</code>
                <details>
                  <summary>원본 파일 SHA-256</summary>
                  <code className="evidence-hash">{f.sha256}</code>
                </details>
                {f.kind === "document" ? (
                  <Link to={`/app/documents/${f.source_id}`}>
                    현재 문서 확인
                  </Link>
                ) : (
                  <a href={`/api/v1/attachments/${f.source_id}`} download>
                    검토할 첨부 내려받기
                  </a>
                )}
              </article>
            ))}
          </div>
          <div className="button-row">
            {request && (
              <Button onClick={() => setReview(request.id)}>
                승인 정책·검토·결정
              </Button>
            )}
            {user.id === record.owner_id &&
              record.status === "awaiting_review" &&
              (!request ||
                (request.status !== "pending" &&
                  request.status !== "approved") ||
                record.approval.stale) && (
                <Button
                  disabled={busy}
                  onClick={() => {
                    setConsent(false);
                    setConfirm({
                      path: "approval",
                      body: {
                        manifest_sha256: record.manifest_sha256,
                        consent: true,
                      },
                      hash: record.manifest_sha256,
                      title: "배포 전체를 검토 요청",
                    });
                  }}
                >
                  문서·첨부 전체 검토 요청
                </Button>
              )}
            {user.id === record.owner_id &&
              record.status === "awaiting_review" &&
              request?.status === "approved" &&
              !record.approval.stale && (
                <Button
                  variant="primary"
                  disabled={busy}
                  onClick={() => {
                    setConsent(false);
                    setConfirm({
                      path: "sign",
                      body: {
                        request_id: request.id,
                        request_version: request.version,
                        consent: true,
                      },
                      hash: record.manifest_sha256,
                      title: "승인된 배포를 서명 준비",
                    });
                  }}
                >
                  승인 확인 후 서명 준비
                </Button>
              )}
            {user.id === record.owner_id && record.status === "ready" && (
              <a
                href={`/api/v1/knowledge/distribution/exports/${id}/download`}
                download
              >
                서명 ZIP 내려받기
              </a>
            )}
          </div>
        </>
      )}
      {review && (
        <ApprovalReview
          requestID={review}
          onClose={() => setReview("")}
          onChanged={() => setRefresh((v) => v + 1)}
        />
      )}
      <Modal
        open={!!confirm && !!record}
        title={confirm?.title || "배포 확인"}
        onOpenChange={(open) => {
          if (!open && !busy) {
            setConfirm(null);
            setConsent(false);
          }
        }}
      >
        <p>
          검토한 모든 문서·첨부의 정확한 해시와 위 수신망·유효기간을 확인하세요.
          실제 반입은 수신망의 비공개 검토로 진행되며 해당 망의 게시 승인 절차를
          대신하지 않습니다.
        </p>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={consent}
            disabled={busy}
            onChange={(e) => setConsent(e.target.checked)}
          />
          배포 범위를 확인했으며 이 작업을 요청합니다
        </label>
        <Button
          variant="primary"
          disabled={busy || !consent}
          onClick={() => void execute()}
        >
          확인하고 진행
        </Button>
      </Modal>
    </section>
  );
}
