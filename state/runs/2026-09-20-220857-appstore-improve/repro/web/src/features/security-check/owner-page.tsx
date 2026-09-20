import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  CheckCircle2,
  Copy,
  ExternalLink,
  RefreshCcw,
  Send,
  ShieldCheck,
} from "lucide-react";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api } from "../../lib/api";
import { formatDateTime } from "../../lib/utils";
import {
  Button,
  Card,
  ErrorState,
  Field,
  Input,
  LoadingState,
  PageHeader,
} from "../../components/ui";
import type { SecurityCheckView } from "../../types";

// SecCheck's own wording for where a review stands, so an owner reading both
// screens sees the same words.
const REMOTE_STATUS: Record<string, string> = {
  UNVERIFIED: "미확인",
  DRAFT: "작성 중",
  CHANGE_REQUESTED: "보완 요청",
  SUBMITTED: "제출됨",
  RESUBMITTED: "재제출됨",
  REVIEWING: "검토 중",
  APPROVAL_PENDING: "승인 대기",
  APPROVED: "승인됨",
  REJECTED: "반려됨",
  CANCELLED: "취소됨",
  CLOSED: "종료됨",
};

export function AppSecurityCheckPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const client = useQueryClient();
  const [reviewId, setReviewId] = useState("");
  const [copied, setCopied] = useState(false);
  const check = useQuery({
    queryKey: ["app-security-check", id],
    queryFn: ({ signal }) => api.appSecurityCheck(id, signal),
    enabled: !!id,
  });
  const verify = useMutation({
    mutationFn: (value: string) => api.verifyAppSecurityCheck(id, value),
    onSuccess: async (result) => {
      client.setQueryData(["app-security-check", id], result);
      await client.invalidateQueries({ queryKey: ["my-apps"] });
    },
  });
  const submit = useMutation({
    mutationFn: () => api.submitApp(id),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["my-apps"] });
      await client.invalidateQueries({ queryKey: ["app-security-check", id] });
      navigate("/my/apps");
    },
  });

  if (check.isPending)
    return (
      <div className="page">
        <LoadingState label="보안 심의 상태를 불러오는 중입니다" />
      </div>
    );
  if (check.error)
    return (
      <div className="page">
        <ErrorState error={check.error} retry={() => void check.refetch()} />
      </div>
    );
  const view = check.data as SecurityCheckView;
  const waiting = view.status !== "UNVERIFIED" && !view.verified;

  return (
    <div className="page">
      <PageHeader
        eyebrow="Security review"
        title="보안 심의"
        description={`${view.appName} · SecCheck에서 이 앱의 보안 심의를 받고, 승인 결과를 여기에서 확인합니다.`}
        actions={
          <Link className="button button-secondary" to="/my/apps">
            <ArrowLeft size={17} /> 내 앱
          </Link>
        }
      />

      {!view.enabled && (
        <div className="notice mt-5">
          이 조직은 보안 심의를 필수로 적용하고 있지 않습니다. 심의를 마치면
          앱에 <strong>보안 심의 완료</strong> 라벨이 붙습니다.
        </div>
      )}
      {view.enabled && !view.configured && (
        <div className="notice notice-danger mt-5" role="alert">
          SecCheck 연동이 아직 설정되지 않았습니다. 관리자에게 문의하세요.
        </div>
      )}

      <div className="detail-body">
        <div className="detail-main">
          <Card className="form-section">
            <h2 className="security-step">1. SecCheck에 심의 등록</h2>
            <p className="field-help">
              SecCheck에서 <strong>서비스명을 앱 이름과 똑같이</strong>{" "}
              입력하고, 아래 연동 정보를 <strong>심의 설명</strong>에 그대로
              붙여 넣으세요. 이 정보가 있어야 어떤 앱의 심의인지 확인할 수
              있습니다.
            </p>
            <pre className="security-binding">{view.bindingText}</pre>
            <div className="branding-actions">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => {
                  void navigator.clipboard
                    ?.writeText(view.bindingText)
                    .then(() => setCopied(true))
                    .catch(() => setCopied(false));
                }}
              >
                <Copy size={16} /> {copied ? "복사됨" : "연동 정보 복사"}
              </Button>
              {view.newReviewUrl && (
                <a
                  className="button button-secondary button-sm"
                  href={view.newReviewUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <ExternalLink size={16} /> SecCheck에서 심의 등록
                </a>
              )}
            </div>
            <p className="field-help mt-4">
              앱 정보를 수정하면 연동 정보가 새로 발급됩니다. 수정한 뒤에는 이
              화면에서 다시 복사해 심의에 반영하세요.
            </p>
          </Card>

          <Card className="form-section">
            <h2 className="security-step">2. 승인 결과 확인</h2>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                verify.mutate(reviewId.trim() || (view.reviewId ?? ""));
              }}
            >
              <Field
                label="심의 ID"
                id="security-review-id"
                help="SecCheck 심의 주소 끝의 ID입니다. 한 번 확인한 뒤에는 비워 두면 같은 심의를 다시 조회합니다."
              >
                <Input
                  id="security-review-id"
                  value={reviewId}
                  placeholder={
                    view.reviewId || "00000000-0000-0000-0000-000000000000"
                  }
                  onChange={(event) => setReviewId(event.target.value)}
                  maxLength={64}
                />
              </Field>
              {verify.error && (
                <div className="notice notice-danger mt-4" role="alert">
                  {verify.error.message}
                </div>
              )}
              <div className="form-actions mt-5">
                <Button
                  type="submit"
                  variant="secondary"
                  disabled={
                    verify.isPending || (!reviewId.trim() && !view.reviewId)
                  }
                >
                  <RefreshCcw size={17} />{" "}
                  {verify.isPending ? "확인 중…" : "심의 결과 확인"}
                </Button>
              </div>
            </form>
          </Card>

          <Card className="form-section">
            <h2 className="security-step">3. 등록 제출</h2>
            {view.verified ? (
              <>
                <div className="notice notice-positive">
                  <CheckCircle2 size={19} aria-hidden="true" /> 보안 심의가
                  확인되었습니다. 이제 앱을 제출할 수 있습니다.
                </div>
                {submit.error && (
                  <div className="notice notice-danger mt-4" role="alert">
                    {submit.error.message}
                  </div>
                )}
                <div className="form-actions mt-5">
                  <Button
                    disabled={submit.isPending}
                    onClick={() => submit.mutate()}
                  >
                    <Send size={17} />{" "}
                    {submit.isPending ? "제출 중…" : "검토 제출"}
                  </Button>
                </div>
              </>
            ) : (
              <p className="field-help">
                최종 승인된 심의가 확인되면 여기에서 검토 제출 버튼이 열립니다.
                조건부 승인이나 종료된 심의로는 제출할 수 없습니다.
              </p>
            )}
          </Card>
        </div>

        <Card className="prose-card">
          <h2>심의 상태</h2>
          <div className="badge-row">
            {view.verified ? (
              <span className="badge badge-positive">
                <ShieldCheck size={14} aria-hidden="true" /> 보안 심의 완료
              </span>
            ) : (
              <span className="badge badge-warning">
                {waiting ? "심의 진행 중" : "미확인"}
              </span>
            )}
          </div>
          <dl className="meta-list mt-5">
            <div className="meta-row">
              <dt>SecCheck 상태</dt>
              <dd>{REMOTE_STATUS[view.status] ?? view.status}</dd>
            </div>
            <div className="meta-row">
              <dt>심의 번호</dt>
              <dd>{view.reviewNumber || "—"}</dd>
            </div>
            <div className="meta-row">
              <dt>최종 결과</dt>
              <dd>{view.finalResult || "—"}</dd>
            </div>
            <div className="meta-row">
              <dt>승인 시각</dt>
              <dd>{view.approvedAt ? formatDateTime(view.approvedAt) : "—"}</dd>
            </div>
            <div className="meta-row">
              <dt>마지막 확인</dt>
              <dd>{view.checkedAt ? formatDateTime(view.checkedAt) : "—"}</dd>
            </div>
            <div className="meta-row">
              <dt>앱 상태</dt>
              <dd>{view.appStatus}</dd>
            </div>
          </dl>
          {view.reviewUrl && (
            <a
              className="button button-secondary w-full mt-5"
              href={view.reviewUrl}
              target="_blank"
              rel="noopener noreferrer"
            >
              <ExternalLink size={17} /> SecCheck에서 열기
            </a>
          )}
        </Card>
      </div>
    </div>
  );
}
