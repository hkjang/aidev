import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  Database,
  RefreshCw,
  ShieldCheck,
  Square,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { api, ApiError } from "./api";
import { useApp } from "./context";
import { Badge, Button, ErrorBox, Loading, Modal } from "./ui";
import "./rag-index.css";

type Provider = {
  configured: boolean;
  base_url: string;
  model: string;
  fingerprint: string;
  rerank: {
    enabled: boolean;
    base_url: string;
    model: string;
    fingerprint: string;
  };
};
type Grant = {
  id: string;
  revision: number;
  active: boolean;
  auto_reindex: boolean;
  allow_rerank: boolean;
  provider_fingerprint: string;
  rerank_fingerprint: string;
  requires_reconsent: boolean;
  actor_id: string;
};
type Index = {
  job_id: string;
  status: "pending" | "running" | "succeeded" | "failed" | "cancelled";
  index_status: "building" | "ready";
  document_version: number;
  total_chunks: number;
  indexed_chunks: number;
  dimensions: number;
  error?: string;
  can_retry: boolean;
};
type Status = {
  document_id: string;
  document_version: number;
  title: string;
  visibility: string;
  can_index: boolean;
  enabled: boolean;
  provider: Provider | null;
  grant: Grant | null;
  index: Index | null;
  notice?: string;
};
const names: Record<Index["status"], string> = {
  pending: "작업 대기",
  running: "색인 생성 중",
  succeeded: "작업 완료",
  failed: "색인 실패",
  cancelled: "작업 취소됨",
};

export default function DocumentRAGIndex({
  documentID,
  generationID,
  onClose,
}: {
  documentID: string;
  generationID?: string;
  onClose: () => void;
}) {
  const { user } = useApp();
  return (
    <IndexConsent
      key={`${user.id}:${documentID}:${generationID || "active"}`}
      documentID={documentID}
      generationID={generationID}
      onClose={onClose}
    />
  );
}

function IndexConsent({
  documentID,
  generationID,
  onClose,
}: {
  documentID: string;
  generationID?: string;
  onClose: () => void;
}) {
  const { workspace, user, notify } = useApp();
  const [status, setStatus] = useState<Status | null>(null);
  const [error, setError] = useState<unknown>(null),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false);
  const [consent, setConsent] = useState(false),
    [automatic, setAutomatic] = useState(false),
    [rerank, setRerank] = useState(false);
  const [confirm, setConfirm] = useState<"cancel" | "revoke" | null>(null);
  const mounted = useRef(true),
    sequence = useRef(0),
    busyRef = useRef(false);
  const generationQuery = generationID
    ? `?generation_id=${encodeURIComponent(generationID)}`
    : "";
  const basePath = `/documents/${documentID}/rag-index`;
  const path = basePath + generationQuery;
  const identity = status
    ? `${status.document_version}:${status.can_index}:${status.enabled}:${status.provider?.fingerprint}:${status.provider?.rerank?.fingerprint}:${status.grant?.revision}:${status.grant?.active}`
    : "";
  useEffect(() => {
    setConsent(false);
    setAutomatic(false);
    setRerank(false);
    setConfirm(null);
  }, [identity]);
  const refresh = useCallback(
    async (clearError = false) => {
      const request = ++sequence.current;
      try {
        const data = await api<Status>(path);
        if (!mounted.current || sequence.current !== request) return;
        setStatus(data);
        if (clearError) setError(null);
      } catch (e) {
        if (!mounted.current || sequence.current !== request) return;
        // Do not continue displaying an old provider, job or consent after ACL
        // revocation, and never show its controls if status cannot be verified.
        setStatus(null);
        setConsent(false);
        setError(e);
      } finally {
        if (mounted.current && sequence.current === request) setLoading(false);
      }
    },
    [path],
  );
  useEffect(() => {
    mounted.current = true;
    void refresh(true);
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      if (!mounted.current) return;
      if (!busyRef.current && document.visibilityState === "visible")
        await refresh();
      if (mounted.current) timer = setTimeout(poll, 2000);
    };
    timer = setTimeout(poll, 2000);
    return () => {
      mounted.current = false;
      sequence.current++;
      clearTimeout(timer);
    };
  }, [refresh]);
  const running =
    status?.index && ["pending", "running"].includes(status.index.status);
  const canStart =
    !!status?.can_index &&
    status.enabled &&
    !!status.provider?.configured &&
    !running;
  const mutate = async (action: "start" | "cancel" | "revoke") => {
    if (busyRef.current || !status?.can_index) return;
    if (action === "start" && (!canStart || !consent || !status.provider))
      return;
    busyRef.current = true;
    setBusy(true);
    setError(null);
    sequence.current++;
    const current = status;
    try {
      if (action === "start")
        await api(path, "POST", {
          expected_version: current.document_version,
          provider_fingerprint: current.provider!.fingerprint,
          consent: true,
          auto_reindex: automatic,
          allow_rerank: rerank && current.provider!.rerank.enabled,
          ...(rerank && current.provider!.rerank.enabled
            ? { rerank_fingerprint: current.provider!.rerank.fingerprint }
            : {}),
        });
      else if (action === "revoke" && current.grant)
        await api(path, "DELETE", { grant_revision: current.grant.revision });
      else if (action === "cancel" && current.index)
        await api(basePath + "/cancel" + generationQuery, "POST", {
          job_id: current.index.job_id,
        });
      else return;
      if (!mounted.current) return;
      setConsent(false);
      setAutomatic(false);
      setRerank(false);
      setConfirm(null);
      await refresh(true);
      if (mounted.current)
        notify(
          action === "start"
            ? "검색 색인 작업을 요청했습니다. 처리 상태를 확인하세요."
            : action === "revoke"
              ? "색인 동의를 철회하고 파생 벡터를 삭제했습니다."
              : "실행 중인 색인과 자동 재색인을 중지했습니다.",
        );
    } catch (e) {
      if (!mounted.current) return;
      setError(e);
      setConsent(false);
      setConfirm(null);
      if (e instanceof ApiError && [403, 404, 409].includes(e.status))
        await refresh();
    } finally {
      busyRef.current = false;
      if (mounted.current) setBusy(false);
    }
  };
  const index = status?.index;
  const usable =
    !!status?.enabled &&
    !!status.grant?.active &&
    !status.grant.requires_reconsent &&
    index?.index_status === "ready" &&
    index.document_version === status.document_version;
  const total = Math.max(0, index?.total_chunks || 0),
    indexed = Math.max(0, Math.min(total, index?.indexed_chunks || 0));
  const canSettings =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  return (
    <>
      <Modal
        open
        onOpenChange={(open) => {
          if (!open) onClose();
        }}
        title="문서 검색 색인"
        description="이 문서의 저장된 내용만 대상으로 검색 색인 전송 동의를 관리합니다. 저장하지 않은 초안은 포함하지 않습니다."
        wide
      >
        <div className="rag-index-panel">
          <ErrorBox error={error} />
          {generationID && (
            <p className="notice subtle">
              선택한 색인 세대의 동의와 파생 벡터만 관리합니다. 다른 세대는
              영향을 받지 않습니다. 자동 재색인은 이 세대가 활성 검색 세대가 된
              이후의 문서 변경부터 적용됩니다.
            </p>
          )}
          {loading && <Loading />}
          {!status && !loading && (
            <Button
              variant="secondary"
              onClick={() => {
                setLoading(true);
                void refresh(true);
              }}
            >
              <RefreshCw size={17} />
              상태 다시 확인
            </Button>
          )}
          {status && (
            <>
              <div className="rag-index-document">
                <strong>{status.title}</strong>
                <span>
                  저장본 v{status.document_version} ·{" "}
                  {(
                    {
                      private: "나만 보기",
                      selected: "선택한 사용자",
                      workspace: "워크스페이스 공유",
                    } as Record<string, string>
                  )[status.visibility] || "접근 제한 문서"}
                </span>
              </div>
              {status.notice && <p className="notice">{status.notice}</p>}
              {!status.can_index ? (
                <div className="notice">
                  <ShieldCheck size={20} />
                  <span>
                    이 문서의 색인을 관리할 권한이 없습니다. 문서 작성 권한과 AI
                    실행 권한이 필요합니다.
                  </span>
                </div>
              ) : (
                <>
                  {!status.enabled && (
                    <div className="notice warning">
                      <TriangleAlert size={20} />
                      <span>
                        검색 AI가 비활성화되어 새 색인을 시작할 수 없습니다.
                        기존 동의는 아래에서 철회할 수 있습니다.
                      </span>
                    </div>
                  )}
                  {status.provider && (
                    <section className="rag-index-provider">
                      <h3>
                        <ShieldCheck size={18} />
                        내용이 전송될 임베딩 공급자
                      </h3>
                      <dl>
                        <div>
                          <dt>API 주소</dt>
                          <dd>{status.provider.base_url || "설정되지 않음"}</dd>
                        </div>
                        <div>
                          <dt>모델</dt>
                          <dd>{status.provider.model || "설정되지 않음"}</dd>
                        </div>
                      </dl>
                      <p className="muted">
                        인증정보 노출을 막기 위해 표시 주소에서는 쿼리 문자열을
                        생략합니다. 실제 연결 설정은 관리자가 확인할 수
                        있습니다.
                      </p>
                      {/^http:/i.test(status.provider.base_url) && (
                        <p className="rag-index-warning">
                          HTTP 연결입니다. 문서 내용과 API 키가 전송 중
                          암호화되지 않습니다.
                        </p>
                      )}
                      {!status.provider.configured && (
                        <p className="rag-index-warning">
                          공급자 연결 설정이 필요합니다.
                        </p>
                      )}
                      {status.provider.rerank.enabled && (
                        <div className="rag-rerank-provider">
                          <h4>선택적 재정렬 공급자</h4>
                          <p>
                            {status.provider.rerank.base_url || "주소 미설정"}
                          </p>
                          <p>
                            모델:{" "}
                            {status.provider.rerank.model || "모델 미설정"}
                          </p>
                          <small>
                            별도로 동의한 경우에만 검색 결과의 관련 문서 조각을
                            재정렬 공급자에 전송합니다.
                          </small>
                        </div>
                      )}
                    </section>
                  )}
                  {status.grant?.requires_reconsent && (
                    <div className="notice warning">
                      <TriangleAlert size={20} />
                      <span>
                        공급자 설정이 변경되어 재동의가 필요합니다. 기존 동의가
                        새 공급자 전송을 자동으로 허용하지 않습니다.
                      </span>
                    </div>
                  )}
                  <section className="rag-index-status">
                    <h3>
                      <Database size={19} />
                      색인 상태
                    </h3>
                    {index ? (
                      <>
                        <div className="rag-index-status-heading">
                          <Badge
                            tone={
                              index.status === "failed"
                                ? "red"
                                : usable
                                  ? "green"
                                  : ""
                            }
                          >
                            {names[index.status] || "상태 확인 필요"}
                          </Badge>
                          <span>색인 대상 v{index.document_version}</span>
                        </div>
                        {total > 0 ? (
                          <>
                            <progress
                              max={total}
                              value={indexed}
                              aria-label="문서 색인 진행률"
                            />
                            <p>
                              {total.toLocaleString("ko-KR")}개 조각 중{" "}
                              {indexed.toLocaleString("ko-KR")}개 저장
                              {index.dimensions
                                ? ` · ${index.dimensions.toLocaleString("ko-KR")}차원`
                                : ""}
                            </p>
                          </>
                        ) : (
                          <p>
                            {running
                              ? "문서를 분할하고 작업을 준비하고 있습니다."
                              : "저장된 문서 조각이 없습니다."}
                          </p>
                        )}
                        {index.index_status === "ready" && (
                          <p>
                            {usable
                              ? "현재 문서와 동의 상태에 맞는 색인입니다."
                              : "기존 작업의 색인 저장본입니다. 문서·공급자·전송 동의 또는 활성 설정이 변경되어 현재 검색에는 사용하지 않습니다."}
                          </p>
                        )}
                        {index.error && <ErrorBox error={index.error} />}
                      </>
                    ) : (
                      <p>아직 이 문서의 색인 작업이 없습니다.</p>
                    )}
                    <p className="rag-index-grant">
                      {status.grant?.active
                        ? `전송 동의 활성 · 변경 시 자동 재색인 ${status.grant.auto_reindex ? "허용" : "허용 안 함"} · 재정렬 전송 ${status.grant.allow_rerank ? "허용" : "허용 안 함"}`
                        : "활성화된 전송 동의가 없습니다."}
                    </p>
                    <div className="rag-index-actions">
                      {running && (
                        <Button
                          variant="secondary"
                          disabled={busy}
                          onClick={() => setConfirm("cancel")}
                        >
                          <Square size={16} />
                          색인 중지
                        </Button>
                      )}
                      {status.grant?.active && (
                        <Button
                          variant="danger"
                          disabled={busy}
                          onClick={() => setConfirm("revoke")}
                        >
                          <Trash2 size={16} />
                          동의 철회 및 색인 삭제
                        </Button>
                      )}
                      <Button
                        variant="secondary"
                        disabled={busy}
                        onClick={() => void refresh(true)}
                      >
                        <RefreshCw size={16} />
                        새로고침
                      </Button>
                    </div>
                  </section>
                  {canStart && (
                    <form
                      className="rag-index-consent"
                      onSubmit={(event) => {
                        event.preventDefault();
                        void mutate("start");
                      }}
                    >
                      <h3>
                        {index?.can_retry
                          ? "현재 저장본으로 다시 색인"
                          : "저장된 문서 색인하기"}
                      </h3>
                      <p>
                        원본 문서의 접근 권한은 검색에도 적용됩니다. 색인 허용은
                        문서를 다른 사용자에게 공개하는 동의가 아닙니다.
                      </p>
                      <label>
                        <input
                          type="checkbox"
                          checked={consent}
                          disabled={busy}
                          onChange={(e) => setConsent(e.target.checked)}
                          required
                        />
                        <span>
                          위 공급자에게 이 문서의 저장된 내용을 전송해 임베딩을
                          생성하는 데 동의합니다.{" "}
                          {status.visibility === "private"
                            ? "‘나만 보기’ 문서도 선택한 공급자에게는 전송됩니다."
                            : "문서에 포함된 민감정보와 전송 주소를 확인했습니다."}
                        </span>
                      </label>
                      <label>
                        <input
                          type="checkbox"
                          checked={automatic}
                          disabled={busy}
                          onChange={(e) => setAutomatic(e.target.checked)}
                        />
                        <span>
                          이 문서가 변경되면 같은 공급자에 변경된 내용을
                          자동으로 다시 전송해 색인합니다.{" "}
                          <small>
                            선택 사항 · 기본은 이번 저장본 한 번만 색인합니다.
                          </small>
                        </span>
                      </label>
                      {status.provider?.rerank.enabled && (
                        <label>
                          <input
                            type="checkbox"
                            checked={rerank}
                            disabled={busy}
                            onChange={(e) => setRerank(e.target.checked)}
                          />
                          <span>
                            위 재정렬 공급자에게도 검색 시 관련 문서 조각을
                            전송하는 데 동의합니다.{" "}
                            <small>
                              선택 사항 · 임베딩 공급자와 다른 시스템일 수
                              있습니다.
                            </small>
                          </span>
                        </label>
                      )}
                      <Button disabled={busy || !consent}>
                        <Database size={18} />
                        {busy ? "요청 중…" : "동의하고 색인 시작"}
                      </Button>
                    </form>
                  )}
                </>
              )}
              {canSettings && (
                <Link
                  className="text-button"
                  to="/app/search-ai-settings"
                  onClick={onClose}
                >
                  워크스페이스 검색 AI 설정 →
                </Link>
              )}
              {user.role === "admin" && (
                <Link
                  className="text-button"
                  to="/admin/search-ai"
                  onClick={onClose}
                >
                  서비스 검색 AI 설정 →
                </Link>
              )}
            </>
          )}
        </div>
      </Modal>
      <Modal
        open={!!confirm}
        onOpenChange={(open) => {
          if (!open) setConfirm(null);
        }}
        title={
          confirm === "revoke"
            ? "전송 동의를 철회하고 색인을 삭제할까요?"
            : "색인 작업을 중지할까요?"
        }
        description={
          confirm === "revoke"
            ? "진행 중인 작업을 취소하고 이 문서에서 생성된 벡터를 삭제합니다. 원본 문서는 삭제하지 않습니다. 이미 공급자에 전송된 내용은 회수할 수 없습니다."
            : "진행 중인 색인 작업과 이후 자동 재색인을 중지합니다. 기존 전송 동의는 유지됩니다. 이미 전송한 내용은 회수할 수 없습니다."
        }
      >
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setConfirm(null)}
          >
            돌아가기
          </Button>
          <Button
            variant="danger"
            disabled={busy}
            onClick={() => confirm && void mutate(confirm)}
          >
            {busy
              ? "처리 중…"
              : confirm === "revoke"
                ? "동의 철회 및 삭제"
                : "색인 중지"}
          </Button>
        </div>
      </Modal>
    </>
  );
}
