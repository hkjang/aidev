import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { sha256 } from "@noble/hashes/sha2.js";
import { api, bytes, datetime } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { readSignedArchive } from "./distribution-archive";
import type { SignedArchive } from "./distribution-archive";
import DistributionReview from "./DistributionReview";
import "./evidence.css";
import "./distribution.css";

type Policy = {
  enabled: boolean;
  revision: number;
  instance_id: string;
  max_valid_days: number;
};
type Context = {
  policy: Policy;
  signing_keys: { id: string; label: string; fingerprint: string }[];
  notice: string;
};
type Distribution = {
  id: string;
  status: string;
  job_id: string;
  job_status: string;
  receiver_instance: string;
  expires_at: string;
  created_at: string;
  artifact_bytes: number;
  downloads: number;
};
const statuses: Record<string, string> = {
  queued: "서명 준비 중",
  awaiting_review: "배포 전체 검토 대기",
  ready: "반출 준비됨",
  failed: "준비 실패",
  revoked: "폐기됨",
  expired: "만료됨",
};
const wait = (signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    const abort = () => {
      clearTimeout(timer);
      reject(new DOMException("취소됨", "AbortError"));
    };
    const timer = setTimeout(() => {
      signal.removeEventListener("abort", abort);
      resolve();
    }, 1500);
    signal.addEventListener("abort", abort, { once: true });
  });
const checksum = (data: Uint8Array) =>
  Array.from(sha256(data))
    .map((v) => v.toString(16).padStart(2, "0"))
    .join("");

export default function KnowledgeDistributionPage() {
  const { workspace, user, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const session = params.get("session") || "",
    mode = params.get("mode") === "import" ? "import" : "export";
  const [context, setContext] = useState<Context | null>(null),
    [runs, setRuns] = useState<Distribution[]>([]),
    [selected, setSelected] = useState<string[]>([]);
  const [key, setKey] = useState(""),
    [receiver, setReceiver] = useState(""),
    [days, setDays] = useState(7),
    [archive, setArchive] = useState<SignedArchive | null>(null);
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [progress, setProgress] = useState(""),
    [confirm, setConfirm] = useState<"export" | "import" | null>(null),
    [consent, setConsent] = useState(false),
    [revoke, setRevoke] = useState<Distribution | null>(null);
  const [receipt, setReceipt] = useState<Record<string, any> | null>(null);
  const [exportSelection, setExportSelection] = useState<
    { id: string; version: number; title: string }[]
  >([]);
  const generation = useRef(0),
    abort = useRef<AbortController | null>(null),
    policyFingerprint = useRef("");
  const writable =
    user?.role !== "viewer" &&
    ["owner", "admin", "editor"].includes(workspace?.role || "");
  useEffect(() => {
    const current = ++generation.current;
    let active = true,
      pending = false;
    abort.current?.abort();
    setArchive(null);
    setReceipt(null);
    setSelected([]);
    setKey("");
    setConfirm(null);
    setConsent(false);
    setContext(null);
    setRuns([]);
    setError("");
    setBusy(false);
    setProgress("");
    policyFingerprint.current = "";
    const load = async () => {
      if (!workspace || pending) return;
      pending = true;
      try {
        const [next, rows] = await Promise.all([
          api<Context>(
            `/knowledge/distribution/context?workspace_id=${workspace.id}`,
          ),
          api<Distribution[]>(
            `/knowledge/distribution/exports?workspace_id=${workspace.id}`,
          ),
        ]);
        if (!active || current !== generation.current) return;
        const fingerprint = JSON.stringify(next);
        if (
          policyFingerprint.current &&
          fingerprint !== policyFingerprint.current
        ) {
          setConfirm(null);
          setConsent(false);
        }
        policyFingerprint.current = fingerprint;
        setContext(next);
        setRuns(rows);
        setKey((previous) =>
          next.signing_keys.some((k) => k.id === previous) ? previous : "",
        );
      } catch (e) {
        if (active && current === generation.current) {
          setContext(null);
          setRuns([]);
          setConfirm(null);
          setConsent(false);
          setReceipt(null);
          setError((e as Error).message);
          abort.current?.abort();
        }
      } finally {
        pending = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 3000);
    return () => {
      active = false;
      generation.current++;
      clearInterval(timer);
      abort.current?.abort();
    };
  }, [workspace?.id, user?.id]);
  useEffect(() => {
    if (!session || !workspace || !writable) {
      setReceipt(null);
      return;
    }
    let active = true,
      pending = false;
    const controller = new AbortController();
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const v = await api(
          `/migrations/sessions/${session}/signed-source`,
          "GET",
          undefined,
          { signal: controller.signal },
        );
        if (active) setReceipt(v);
      } catch {
        if (active) setReceipt(null);
      } finally {
        pending = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 4000);
    return () => {
      active = false;
      controller.abort();
      clearInterval(timer);
    };
  }, [session, workspace?.id, user?.id, writable]);
  useEffect(() => {
    if (
      confirm === "export" &&
      exportSelection.some(
        (ref) =>
          !documents.some(
            (d) =>
              d.id === ref.id &&
              d.version === ref.version &&
              d.status === "published",
          ),
      )
    ) {
      setConfirm(null);
      setConsent(false);
      setExportSelection([]);
      setError(
        "반출 범위를 확인하는 동안 원문이 변경됐습니다. 현재 문서를 다시 확인하세요.",
      );
    }
  }, [documents, confirm, exportSelection]);
  const action = async (
    fn: (signal: AbortSignal, current: number) => Promise<void>,
  ) => {
    if (busy) return;
    setBusy(true);
    setError("");
    const controller = new AbortController();
    abort.current = controller;
    const current = generation.current;
    try {
      await fn(controller.signal, current);
    } catch (e) {
      if (
        current === generation.current &&
        !(e instanceof DOMException && e.name === "AbortError")
      )
        setError((e as Error).message);
    } finally {
      if (current === generation.current) {
        setBusy(false);
        abort.current = null;
      }
    }
  };
  const choose = (file?: File) => {
    if (!file) return;
    setArchive(null);
    setConsent(false);
    void action(async (signal, current) => {
      setProgress("이 브라우저에서 압축 파일·해시 검사 중");
      const result = await readSignedArchive(file, signal);
      if (current === generation.current && !signal.aborted) {
        setArchive(result);
        setProgress(
          "파일 해시 비교 완료 · 서버 서명 검증은 아직 수행하지 않음",
        );
      }
    });
  };
  const exportBundle = () => {
    const source = exportSelection.map((d) => ({
      id: d.id,
      version: d.version,
    }));
    const signer = key,
      target = receiver.trim(),
      validDays = days;
    setConfirm(null);
    setConsent(false);
    void action(async (signal, current) => {
      if (!workspace || !source.length) return;
      setProgress("선택한 원문·첨부의 내보내기 작업 준비");
      const base = await api<{ id: string }>(
        "/exports",
        "POST",
        {
          workspace_id: workspace.id,
          format: "markdown",
          document_ids: source.map((d) => d.id),
        },
        { signal },
      );
      while (!signal.aborted && current === generation.current) {
        const state = await api<{
          status: string;
          report?: { error?: string };
        }>(`/exports/${base.id}`, "GET", undefined, { signal });
        if (state.status === "ready") break;
        if (!["queued", "running"].includes(state.status))
          throw new Error(
            state.report?.error ||
              "원문 내보내기가 완료되지 않았습니다. 이관 센터 작업 이력을 확인하세요.",
          );
        await wait(signal);
      }
      if (signal.aborted || current !== generation.current) return;
      const out = await api<{ id: string }>(
        "/knowledge/distribution/exports",
        "POST",
        {
          export_id: base.id,
          signing_key_id: signer,
          receiver_instance: target,
          valid_days: validDays,
          documents: source,
          consent: true,
        },
        { signal },
      );
      if (current !== generation.current) return;
      setProgress(`서명 작업 요청됨 · ${out.id}`);
      notify("원문·승인·현재 정책을 다시 확인하는 서명 작업을 요청했습니다.");
    });
  };
  const importBundle = () => {
    const source = archive;
    if (!source || !workspace) return;
    const wid = workspace.id;
    setConfirm(null);
    setConsent(false);
    void action(async (signal, current) => {
      const m = source.manifest;
      let id = session;
      if (!id) {
        const created = await api<{ id: string }>(
          "/migrations/sessions",
          "POST",
          {
            workspace_id: wid,
            source_key: `signed:${m.source_instance}:${m.source_workspace}`,
            label: `서명 반입 ${m.bundle_id}`,
            format: "markdown",
          },
          { signal },
        );
        id = created.id;
        if (current !== generation.current || signal.aborted) return;
        setParams({ mode: "import", session: id });
      }
      setProgress("수신망 정책·등록 공개키·서명 확인 중");
      await api(
        `/migrations/sessions/${id}/signed-source`,
        "POST",
        {
          manifest_base64: source.manifest_base64,
          signature: source.signature,
          consent: true,
        },
        { signal },
      );
      const files = new Map(source.files.map((f) => [f.path, f.data]));
      for (let index = 0; index < m.files.length; index++) {
        if (current !== generation.current || signal.aborted) return;
        const f = m.files[index],
          body = files.get(f.path)!;
        setProgress(
          `서명 확인됨 · ${index + 1}/${m.files.length} ${f.path} 업로드`,
        );
        const registered = await api<{ items: { id: string }[] }>(
          `/migrations/sessions/${id}/items`,
          "POST",
          { items: [{ ...f, bytes: body.length, metadata: f.metadata }] },
          { signal },
        );
        const item = registered.items[0];
        const checkpoints = await api<{ ordinal: number; checksum: string }[]>(
          `/migrations/sessions/${id}/items/${item.id}/chunks`,
          "GET",
          undefined,
          { signal },
        );
        const have = new Map(checkpoints.map((c) => [c.ordinal, c.checksum]));
        for (let at = 0; at < body.length; at += 1048576) {
          const chunk = body.subarray(at, at + 1048576),
            ordinal = Math.floor(at / 1048576);
          if (have.get(ordinal) === checksum(chunk)) continue;
          const response = await fetch(
            `/api/v1/migrations/sessions/${id}/items/${item.id}/chunks/${ordinal}`,
            {
              method: "PUT",
              credentials: "same-origin",
              headers: {
                "X-Madi-Request": "1",
                "Content-Type": "application/octet-stream",
              },
              body: new Blob([chunk as BlobPart]),
              signal,
            },
          );
          if (!response.ok) {
            const error = await response.json().catch(() => ({}));
            throw new Error(
              error.error ||
                "파일 업로드를 완료하지 못했습니다. 같은 ZIP으로 이어 올릴 수 있습니다.",
            );
          }
        }
      }
      if (current !== generation.current || signal.aborted) return;
      setProgress(
        "서명 원본 업로드 완료 · 이관 센터에서 준비·차이 검토·반영을 진행하세요.",
      );
      setArchive(null);
      notify(
        "원문은 아직 반영하지 않았습니다. 이관 센터에서 변환 결과를 검토하세요.",
      );
    });
  };
  return (
    <div className="page evidence-page distribution-page">
      <PageHeading
        title="망별 지식 배포"
        description="게시된 원문과 첨부를 서명하고, 신뢰를 확인한 망에서 비공개로 검토합니다."
      />
      {user?.role === "admin" && (
        <Link to="/admin/knowledge-distribution">배포 정책·신뢰 키 관리</Link>
      )}
      <ErrorBox error={error} />
      {!context ? (
        <Loading />
      ) : (
        <>
          <p className="notice">{context.notice}</p>
          <p>
            현재 망 식별자{" "}
            <code className="distribution-id">
              {context.policy.instance_id}
            </code>{" "}
            ·{" "}
            {context.policy.enabled
              ? "배포 정책 활성"
              : "관리자가 배포 정책을 활성화해야 합니다"}
          </p>
          <div className="button-row">
            <Button
              variant={mode === "export" ? "primary" : "secondary"}
              disabled={busy}
              onClick={() => {
                setParams({ mode: "export" });
                setConfirm(null);
              }}
            >
              반출 준비
            </Button>
            <Button
              variant={mode === "import" ? "primary" : "secondary"}
              disabled={busy}
              onClick={() => {
                setParams({ mode: "import" });
                setConfirm(null);
              }}
            >
              서명 반입
            </Button>
          </div>
          {params.get("review") ? (
            <DistributionReview id={params.get("review")!} />
          ) : mode === "export" ? (
            <>
              <section className="card">
                <h2>게시 문서 선택</h2>
                <p>
                  선택한 버전과 첨부 전체를 포함합니다. 승인 기능이 켜져 있으면
                  현재 승인된 게시 문서와 첨부 전체·수신망·유효기간을 별도로
                  검토한 후 원신청자가 서명을 확인합니다. 승인은 관리자의
                  명시적인 망간 지식 배포 정책을 따릅니다.
                </p>
                <div className="distribution-docs">
                  {documents
                    .filter((d) => d.status === "published")
                    .map((d) => (
                      <label className="check" key={d.id}>
                        <input
                          type="checkbox"
                          checked={selected.includes(d.id)}
                          disabled={busy}
                          onChange={(e) => {
                            setSelected((v) =>
                              e.target.checked
                                ? [...v, d.id]
                                : v.filter((id) => id !== d.id),
                            );
                            setConsent(false);
                          }}
                        />
                        <span>
                          {d.title} · v{d.version}
                        </span>
                        <Link to={`/app/documents/${d.id}`}>원문</Link>
                      </label>
                    ))}
                </div>
                {!documents.some((d) => d.status === "published") && (
                  <Empty
                    title="현재 목록에 게시 문서가 없습니다"
                    text="먼저 원문을 게시하고, 승인 기능이 활성화돼 있다면 검토를 완료하세요."
                  />
                )}
                <div className="form-grid">
                  <Field label="서명 키">
                    <select
                      value={key}
                      onChange={(e) => {
                        setKey(e.target.value);
                        setConsent(false);
                      }}
                      disabled={busy}
                    >
                      <option value="">키를 선택하세요</option>
                      {context.signing_keys.map((k) => (
                        <option key={k.id} value={k.id}>
                          {k.label} · {k.fingerprint.slice(0, 12)}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <Field label="수신망 식별자">
                    <input
                      value={receiver}
                      onChange={(e) => {
                        setReceiver(e.target.value);
                        setConsent(false);
                      }}
                      disabled={busy}
                      placeholder="수신망 관리자가 제공한 UUID"
                    />
                  </Field>
                  <Field label="패키지 유효기간(일)">
                    <input
                      type="number"
                      min={1}
                      max={context.policy.max_valid_days}
                      value={days}
                      disabled={busy}
                      onChange={(e) => {
                        setDays(Number(e.target.value));
                        setConsent(false);
                      }}
                    />
                  </Field>
                </div>
                <Button
                  disabled={
                    busy ||
                    !context.policy.enabled ||
                    !key ||
                    !receiver.trim() ||
                    selected.length === 0 ||
                    !Number.isInteger(days) ||
                    days < 1 ||
                    days > context.policy.max_valid_days
                  }
                  onClick={() => {
                    setConsent(false);
                    setExportSelection(
                      documents
                        .filter((d) => selected.includes(d.id))
                        .map((d) => ({
                          id: d.id,
                          version: d.version,
                          title: d.title,
                        })),
                    );
                    setConfirm("export");
                  }}
                >
                  반출 범위 확인
                </Button>
              </section>
              <section className="card">
                <h2>내 배포 이력</h2>
                <p>
                  목록은 최근 100개입니다. 파일 내려받기는 원본 내보내기가
                  유지되는 최대24시간 안에 가능하며, 반입 유효기간과 다릅니다.
                  이미 전달한 파일은 회수되지 않습니다.
                </p>
                {!runs.length && (
                  <Empty
                    title="배포 이력이 없습니다"
                    text="선택한 게시 문서를 확인하고 반출을 준비하세요."
                  />
                )}
                {runs.map((v) => (
                  <article className="distribution-run" key={v.id}>
                    <strong>{statuses[v.status] || "상태 확인 필요"}</strong>
                    <code>{v.id}</code>
                    <span>수신망 {v.receiver_instance}</span>
                    <span>
                      유효 {datetime(v.expires_at)} · {bytes(v.artifact_bytes)}{" "}
                      · 내려받기 {v.downloads}회
                    </span>
                    <div className="button-row">
                      <Link to={`/app/jobs?id=${v.job_id}`}>작업 이력</Link>
                      {v.status === "awaiting_review" && (
                        <Link to={`?review=${v.id}`}>배포 전체 검토 열기</Link>
                      )}
                      {v.status === "ready" && (
                        <a
                          href={`/api/v1/knowledge/distribution/exports/${v.id}/download`}
                          download
                        >
                          서명 ZIP 내려받기
                        </a>
                      )}
                      {["queued", "awaiting_review", "ready"].includes(
                        v.status,
                      ) && (
                        <Button
                          variant="secondary"
                          onClick={() => setRevoke(v)}
                          disabled={busy}
                        >
                          서버 사본 폐기
                        </Button>
                      )}
                    </div>
                  </article>
                ))}
              </section>
            </>
          ) : (
            <section className="card">
              <h2>서명 ZIP으로 반입 준비</h2>
              <p>
                파일 선택 시 압축과 파일 해시만 이 브라우저에서 검사합니다. 등록
                공개키를 통한 실제 서명 검사는 서버에서 별도로 수행합니다.
                반출망의 승인 사실이 수신망 게시 권한이 되지는 않습니다.
              </p>
              {!writable ? (
                <p>
                  이관을 준비하려면 현재 워크스페이스 작성 권한이 필요합니다.
                </p>
              ) : (
                <>
                  <Field label="서명 ZIP 파일">
                    <input
                      type="file"
                      accept=".zip,application/zip"
                      disabled={busy || !context.policy.enabled}
                      onChange={(e) => {
                        choose(e.target.files?.[0]);
                        e.target.value = "";
                      }}
                    />
                  </Field>
                  {session && (
                    <p>
                      이어서 준비하는 세션 <code>{session}</code>{" "}
                      <Button
                        variant="secondary"
                        disabled={busy}
                        onClick={() => {
                          setParams({ mode: "import" });
                          setArchive(null);
                          setProgress("");
                        }}
                      >
                        새 반입으로 시작
                      </Button>
                    </p>
                  )}
                  {archive && (
                    <>
                      <dl className="distribution-summary">
                        <dt>반출망</dt>
                        <dd>{archive.manifest.source_instance}</dd>
                        <dt>수신망</dt>
                        <dd>{archive.manifest.receiver_instance}</dd>
                        <dt>파일</dt>
                        <dd>
                          {archive.manifest.files.length}개 ·{" "}
                          {bytes(
                            archive.manifest.files.reduce(
                              (n, f) => n + f.bytes,
                              0,
                            ),
                          )}
                        </dd>
                        <dt>유효기간</dt>
                        <dd>
                          {datetime(
                            new Date(
                              archive.manifest.expires_at * 1000,
                            ).toISOString(),
                          )}
                        </dd>
                        <dt>매니페스트 SHA256</dt>
                        <dd>{archive.manifest_hash}</dd>
                      </dl>
                      <p>파일 해시 비교 완료 · 서버 서명 미확인</p>
                      <details>
                        <summary>포함된 파일·버전 보기</summary>
                        {archive.manifest.files.map((f) => (
                          <p key={f.source_id}>
                            {f.path} ·{" "}
                            {f.kind === "document"
                              ? `v${f.source_version}`
                              : "첨부"}{" "}
                            · {bytes(f.bytes)}
                          </p>
                        ))}
                      </details>
                      <Button
                        disabled={
                          busy ||
                          !context.policy.enabled ||
                          archive.manifest.receiver_instance !==
                            context.policy.instance_id
                        }
                        onClick={() => {
                          setConsent(false);
                          setConfirm("import");
                        }}
                      >
                        신뢰 확인·비공개 업로드
                      </Button>
                      {archive.manifest.receiver_instance !==
                        context.policy.instance_id && (
                        <p role="alert">
                          현재 망을 수신자로 지정한 패키지가 아닙니다.
                        </p>
                      )}
                    </>
                  )}
                  {session && (
                    <p>
                      <Link to={`/app/migrations?session=${session}`}>
                        이관 센터에서 준비·원문 비교·반영
                      </Link>
                    </p>
                  )}
                  {receipt && (
                    <details>
                      <summary>서명 반입 기록</summary>
                      <p>
                        {receipt.imported_at
                          ? `반영 완료 ${datetime(receipt.imported_at)}`
                          : "서명 원본 결합됨 · 원문 반영 미완료"}
                      </p>
                      <p>{receipt.notice}</p>
                      {receipt.mapping?.map((m: Record<string, any>) => (
                        <p key={`${m.source_id}:${m.target_id || "skip"}`}>
                          {m.source_id} v{m.source_version} →{" "}
                          {m.decision === "skip" ? (
                            "명시적으로 제외됨"
                          ) : (
                            <Link
                              to={
                                m.kind === "document"
                                  ? `/app/documents/${m.target_id}`
                                  : m.target_document_id
                                    ? `/app/documents/${m.target_document_id}`
                                    : `/app/migrations?session=${session}`
                              }
                            >
                              {m.target_id} v{m.target_version}
                              {m.disposition === "reference_copy"
                                ? " · 문서별 첨부 사본"
                                : ""}
                            </Link>
                          )}
                        </p>
                      ))}
                    </details>
                  )}
                </>
              )}
            </section>
          )}
        </>
      )}
      <p role="status" aria-live="polite">
        {progress}
      </p>
      {busy && (
        <Button
          variant="secondary"
          onClick={() => {
            abort.current?.abort();
            setProgress(
              "이 화면의 대기·업로드를 중단했습니다. 이미 요청한 서버 작업은 작업 이력에서 확인하고 필요하면 별도로 취소하세요.",
            );
          }}
        >
          대기·업로드 중단
        </Button>
      )}
      <Modal
        open={!!confirm}
        onOpenChange={(v) => {
          if (!v) {
            setConfirm(null);
            setConsent(false);
          }
        }}
        title={
          confirm === "export"
            ? "반출 대상과 영향을 확인하세요"
            : "서명 확인 후 비공개로 준비합니다"
        }
      >
        <p>
          {confirm === "export"
            ? `${selected.length}개 문서와 첨부를 수신망 ${receiver}에서 ${days}일 동안 반입할 수 있는 패키지로 준비합니다. 원문 버전이 바뀌면 다시 확인해야 합니다.`
            : "등록된 키와 실제 서명을 확인하고 파일별로 업로드합니다. 업로드만으로 문서를 게시하지 않습니다. 변환·링크 복원·정보 보호 결과는 이관 센터에서 다시 검토하세요."}
        </p>
        {confirm === "export" && (
          <ul>
            {exportSelection.map((d) => (
              <li key={d.id}>
                {d.title} · v{d.version}
              </li>
            ))}
          </ul>
        )}
        <label className="check">
          <input
            type="checkbox"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
          />
          원문과 범위를 확인했으며, 반출된 사본은 회수할 수 없음을 이해했습니다.
        </label>
        <Button
          disabled={!consent || busy}
          onClick={confirm === "export" ? exportBundle : importBundle}
        >
          {confirm === "export"
            ? "확인한 버전으로 서명 준비"
            : "서명 확인·업로드 시작"}
        </Button>
      </Modal>
      <Modal
        open={!!revoke}
        onOpenChange={(v) => !v && setRevoke(null)}
        title="서버의 배포 결과를 폐기할까요?"
      >
        <p>내려받은 파일이나 다른 망의 자료는 삭제되지 않습니다.</p>
        <Button
          disabled={busy}
          onClick={() => {
            const id = revoke?.id;
            setRevoke(null);
            if (id)
              void action(async (signal) => {
                await api(
                  `/knowledge/distribution/exports/${id}/revoke`,
                  "POST",
                  { confirmation: "REVOKE" },
                  { signal },
                );
                notify(
                  "서버 배포 사본을 폐기했습니다. 외부 사본은 회수되지 않습니다.",
                );
              });
          }}
        >
          서버 사본 폐기 확인
        </Button>
      </Modal>
    </div>
  );
}
