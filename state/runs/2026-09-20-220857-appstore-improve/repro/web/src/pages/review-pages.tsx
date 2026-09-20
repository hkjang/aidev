import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Check,
  Download,
  ExternalLink,
  FileText,
  Layers,
  ServerCog,
  ShieldCheck,
  ShieldAlert,
  X,
} from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../lib/api";
import { formatDate, formatDateTime } from "../lib/utils";
import { formatFileSize } from "../features/apps/guide-documents";
import {
  AppIcon,
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorState,
  Field,
  LoadingState,
  PageHeader,
  Textarea,
} from "../components/ui";
import type { Review, ReviewDetail } from "../types";

const DECISION_TONE = {
  pending: "warning",
  approved: "positive",
  rejected: "danger",
  cancelled: undefined,
} as const;

const DECISION_LABEL: Record<string, string> = {
  pending: "검토 대기",
  approved: "승인",
  rejected: "반려",
  cancelled: "취소됨",
};

function DecisionBadge({ status }: { status: string }) {
  return (
    <Badge tone={DECISION_TONE[status as keyof typeof DECISION_TONE]}>
      {DECISION_LABEL[status] ?? status}
    </Badge>
  );
}

export function ReviewQueuePage() {
  const reviews = useQuery({
    queryKey: ["reviews"],
    queryFn: ({ signal }) => api.reviews(signal),
  });
  return (
    <div className="page">
      <PageHeader
        eyebrow="Team review"
        title="검토 대기"
        description="Workflow가 활성화된 경우에만 등록 건이 이 목록에 나타납니다."
      />
      {reviews.isPending && <LoadingState />}
      {reviews.error && (
        <ErrorState
          error={reviews.error}
          retry={() => void reviews.refetch()}
        />
      )}
      {reviews.data && !reviews.data.length && (
        <EmptyState
          title="검토 대기 항목이 없습니다"
          description="모든 등록 요청을 처리했습니다."
        />
      )}
      {reviews.data && !!reviews.data.length && (
        <Card className="data-card">
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>앱</th>
                  <th>등록자</th>
                  <th>팀</th>
                  <th>단계</th>
                  <th>상태</th>
                  <th>요청 시각</th>
                  <th>검토</th>
                </tr>
              </thead>
              <tbody>
                {reviews.data.map((review) => (
                  <tr key={review.id}>
                    <td>
                      <strong>
                        {review.appName || review.appSlug || review.appId}
                      </strong>
                    </td>
                    <td>{review.submitterName || "—"}</td>
                    <td>{review.team || "—"}</td>
                    <td>{review.level ? `${review.level}단계` : "—"}</td>
                    <td>
                      <DecisionBadge status={review.status} />
                    </td>
                    <td>{formatDateTime(review.createdAt)}</td>
                    <td>
                      <Link
                        className="button button-secondary button-sm"
                        to={`/review/${review.id}`}
                      >
                        열기
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}

export function ReviewDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const client = useQueryClient();
  const review = useQuery({
    queryKey: ["review", id],
    queryFn: ({ signal }) => api.review(id, signal),
  });
  const [comment, setComment] = useState("");
  const [needsComment, setNeedsComment] = useState(false);
  const settle = async () => {
    await client.invalidateQueries({ queryKey: ["reviews"] });
    await client.invalidateQueries({ queryKey: ["apps"] });
    navigate("/review");
  };
  const approve = useMutation({
    mutationFn: () => api.approveReview(id, comment.trim()),
    onSuccess: settle,
  });
  const reject = useMutation({
    mutationFn: () => api.rejectReview(id, comment.trim()),
    onSuccess: settle,
  });

  if (review.isPending)
    return (
      <div className="page">
        <LoadingState label="검토 내용을 불러오는 중입니다" />
      </div>
    );
  if (review.error)
    return (
      <div className="page">
        <ErrorState error={review.error} retry={() => void review.refetch()} />
      </div>
    );
  if (!review.data) return null;

  const detail = review.data as ReviewDetail;
  const app = detail.app;
  const documents = detail.documents ?? [];
  const security = detail.securityCheck;
  // The current review is in the history too; the rest is what came before.
  const history = (detail.history ?? []).filter(
    (item) => item.id !== detail.id,
  );
  const pending = detail.status === "pending";
  const busy = approve.isPending || reject.isPending;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Review request"
        title={detail.appName || detail.appSlug || "앱 등록 검토"}
        description={`${detail.submitterName || "등록자"} 제출 · ${formatDateTime(detail.createdAt)}${detail.level ? ` · ${detail.level}단계 검토` : ""}`}
        actions={
          <>
            <DecisionBadge status={detail.status} />
            {app && (
              <Link
                className="button button-secondary"
                to={`/apps/${encodeURIComponent(app.slug)}`}
              >
                <ExternalLink size={17} /> 스토어에서 보기
              </Link>
            )}
          </>
        }
      />

      <div className="detail-body">
        <div className="detail-main">
          {app ? (
            <Card className="prose-card">
              <header className="detail-head">
                <AppIcon app={app} />
                <div className="detail-head-content">
                  <h2 className="!mt-0">{app.name}</h2>
                  <p className="detail-summary">{app.summary}</p>
                  <div className="badge-row">
                    {(app.category?.name || app.categoryName) && (
                      <Badge>{app.category?.name || app.categoryName}</Badge>
                    )}
                    {app.language && <Badge>{app.language}</Badge>}
                    {app.supportsMcp && (
                      <Badge tone="primary">
                        <ServerCog size={14} /> MCP
                      </Badge>
                    )}
                    {app.supportsApi && <Badge tone="positive">API</Badge>}
                    <Badge>
                      {app.visibility === "private" ? "Private" : "Public"}
                    </Badge>
                  </div>
                </div>
              </header>
              <h2 className="!mt-8">앱 소개</h2>
              <p className="whitespace-pre-line">
                {app.description || app.summary}
              </p>
              {!!app.tags?.length && (
                <>
                  <h2 className="!mt-8">태그</h2>
                  <div className="badge-row">
                    {app.tags.map((tag) => (
                      <Badge key={tag}>{tag}</Badge>
                    ))}
                  </div>
                </>
              )}
              {!!app.screenshots?.length && (
                <>
                  <h2 className="!mt-8">Screenshot</h2>
                  <ul className="document-list">
                    {app.screenshots.map((shot) => (
                      <li className="document-row" key={shot}>
                        <ExternalLink size={16} aria-hidden="true" />
                        <a
                          className="document-name"
                          href={shot}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          {shot}
                        </a>
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </Card>
          ) : (
            <Card className="prose-card">
              <h2>앱 정보</h2>
              <p>앱 정보를 불러오지 못했습니다. 앱이 삭제되었을 수 있습니다.</p>
            </Card>
          )}

          <Card className="prose-card">
            <h2>가이드 문서</h2>
            {documents.length ? (
              <ul className="document-list mt-4">
                {documents.map((document) => (
                  <li className="document-row" key={document.id}>
                    <FileText size={18} aria-hidden="true" />
                    <span className="document-name">
                      <strong>{document.title || document.fileName}</strong>
                      <span className="document-meta">
                        {document.fileName} · {formatFileSize(document.size)} ·{" "}
                        {formatDate(document.createdAt)}
                      </span>
                    </span>
                    <a
                      className="button button-secondary button-sm"
                      href={document.downloadUrl ?? ""}
                      download={document.fileName}
                    >
                      <Download size={16} /> 내려받기
                    </a>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="field-help">첨부된 가이드 문서가 없습니다.</p>
            )}
          </Card>

          <Card className="form-section">
            <h2 className="security-step">검토 결정</h2>
            <Field
              label="검토 의견"
              id="review-comment"
              help="승인할 때는 선택이고, 반려할 때는 필수입니다. 등록자가 내 앱과 수정 화면에서 그대로 읽습니다."
              error={needsComment ? "반려하려면 사유를 입력하세요." : undefined}
            >
              <Textarea
                id="review-comment"
                value={comment}
                onChange={(event) => {
                  setComment(event.target.value);
                  if (event.target.value.trim()) setNeedsComment(false);
                }}
                maxLength={2000}
                rows={5}
                disabled={!pending || busy}
                placeholder="확인한 내용, 보완이 필요한 부분을 적습니다."
              />
            </Field>
            {(approve.error || reject.error) && (
              <div className="notice notice-danger mt-4" role="alert">
                {(approve.error ?? reject.error)?.message}
              </div>
            )}
            {!pending && (
              <div className="notice mt-4">
                이미 처리된 검토입니다 — {DECISION_LABEL[detail.status]}
                {detail.decidedAt
                  ? ` · ${formatDateTime(detail.decidedAt)}`
                  : ""}
                {detail.reviewerName ? ` · ${detail.reviewerName}` : ""}
              </div>
            )}
            <div className="form-actions mt-5">
              <Button
                variant="danger"
                disabled={!pending || busy}
                onClick={() => {
                  if (!comment.trim()) {
                    setNeedsComment(true);
                    return;
                  }
                  reject.mutate();
                }}
              >
                <X size={18} /> {reject.isPending ? "반려 중…" : "반려"}
              </Button>
              <Button
                disabled={!pending || busy}
                onClick={() => approve.mutate()}
              >
                <Check size={18} />{" "}
                {approve.isPending ? "승인 중…" : "승인 및 게시"}
              </Button>
            </div>
          </Card>
        </div>

        <div className="detail-main">
          <Card className="prose-card">
            <h2>제출 정보</h2>
            <dl className="meta-list">
              <div className="meta-row">
                <dt>등록자</dt>
                <dd>{detail.submitterName || "—"}</dd>
              </div>
              <div className="meta-row">
                <dt>담당팀</dt>
                <dd>{detail.team || app?.team || "—"}</dd>
              </div>
              <div className="meta-row">
                <dt>검토 단계</dt>
                <dd>{detail.level ? `${detail.level}단계` : "—"}</dd>
              </div>
              <div className="meta-row">
                <dt>버전</dt>
                <dd>
                  {app?.version ? `v${app.version.replace(/^v/, "")}` : "—"}
                </dd>
              </div>
              <div className="meta-row">
                <dt>Framework</dt>
                <dd>{app?.framework || "—"}</dd>
              </div>
              <div className="meta-row">
                <dt>최근 수정</dt>
                <dd>{app ? formatDateTime(app.updatedAt) : "—"}</dd>
              </div>
              <div className="meta-row">
                <dt>Slug</dt>
                <dd>{app?.slug || detail.appSlug || "—"}</dd>
              </div>
            </dl>
            {app?.serviceUrl && (
              <a
                className="button button-secondary w-full mt-5"
                href={app.serviceUrl}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink size={17} /> 서비스 URL 열기
              </a>
            )}
          </Card>

          {security && (
            <Card className="prose-card">
              <h2>보안 심의</h2>
              <div className="badge-row">
                {security.verified ? (
                  <Badge tone="positive">
                    <ShieldCheck size={14} aria-hidden="true" /> 보안 심의 완료
                  </Badge>
                ) : (
                  <Badge tone="warning">
                    <ShieldAlert size={14} aria-hidden="true" /> 미확인
                  </Badge>
                )}
              </div>
              <dl className="meta-list mt-4">
                <div className="meta-row">
                  <dt>심의 번호</dt>
                  <dd>{security.reviewNumber || "—"}</dd>
                </div>
                <div className="meta-row">
                  <dt>승인 시각</dt>
                  <dd>
                    {security.approvedAt
                      ? formatDateTime(security.approvedAt)
                      : "—"}
                  </dd>
                </div>
              </dl>
              {security.reviewUrl && (
                <a
                  className="button button-secondary w-full mt-4"
                  href={security.reviewUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ExternalLink size={17} /> SecCheck 심의 열기
                </a>
              )}
            </Card>
          )}

          <Card className="prose-card">
            <h2>
              <Layers size={17} aria-hidden="true" /> 이전 검토
            </h2>
            {history.length ? (
              <ul className="review-history">
                {history.map((item) => (
                  <ReviewHistoryItem key={item.id} review={item} />
                ))}
              </ul>
            ) : (
              <p className="field-help">이 앱의 첫 검토입니다.</p>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}

function ReviewHistoryItem({ review }: { review: Review }) {
  return (
    <li className="review-history-item">
      <div className="badge-row !mt-0">
        <DecisionBadge status={review.status} />
        {!!review.level && <Badge>{review.level}단계</Badge>}
      </div>
      <div className="document-meta">
        {formatDateTime(review.decidedAt || review.createdAt)}
        {review.reviewerName ? ` · ${review.reviewerName}` : ""}
      </div>
      {review.reason && (
        <p className="review-history-comment">{review.reason}</p>
      )}
    </li>
  );
}
