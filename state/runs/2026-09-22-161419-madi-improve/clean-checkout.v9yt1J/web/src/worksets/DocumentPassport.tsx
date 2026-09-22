import { useEffect, useRef, useState } from "react";
import { api, ApiError, datetime } from "../api";
import { useApp } from "../context";
import { Badge, Button, Field, Loading, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { useModalReturnFocus } from "../review/useModalReturnFocus";
import "./style.css";
type Preview = {
  version: number;
  before: { title: string; tags: string[]; markdown: string };
  after: { title: string; tags: string[]; markdown: string };
  title_candidates: string[];
  tag_candidates: string[];
  front_matter_reformatted: boolean;
  notice: string;
};
export default function DocumentPassport({
  documentID,
  onClose,
  allowCleanup = true,
  onUpdated,
}: {
  documentID: string | null;
  onClose: () => void;
  allowCleanup?: boolean;
  onUpdated?: () => void;
}) {
  const { user, workspace } = useApp(),
    [record, setData] = useState<any>(null),
    [error, setError] = useState<unknown>(null),
    [cleanup, setCleanup] = useState(false);
  const scope = `${user.id}:${workspace?.id}:${documentID}`,
    current = useRef(scope),
    loaded = useRef("");
  current.current = scope;
  const data = loaded.current === scope ? record : null;
  const returnFocus = useModalReturnFocus(
    documentID,
    `${user.id}:${workspace?.id}`,
  );
  useEffect(() => {
    setData(null);
    setCleanup(false);
    setError(null);
    if (!documentID) return;
    const ctl = new AbortController();
    let pending = false;
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const value = await api(
          `/documents/${documentID}/passport`,
          "GET",
          undefined,
          { signal: ctl.signal },
        );
        if (!ctl.signal.aborted && current.current === scope) {
          if (value.workspace_id !== workspace?.id)
            throw new Error("문서의 현재 워크스페이스를 선택해 주세요.");
          loaded.current = scope;
          setData(value);
          setError(null);
        }
      } catch (e) {
        if (!ctl.signal.aborted && current.current === scope) {
          setData(null);
          setCleanup(false);
          setError(e);
        }
      } finally {
        pending = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      ctl.abort();
      clearInterval(timer);
    };
  }, [scope]);
  return (
    <>
      <Modal
        open={!!documentID && !cleanup}
        onOpenChange={(v) => {
          if (!v) onClose();
        }}
        title="문서 여권"
        onCloseAutoFocus={returnFocus}
        description="현재 소유·버전·분류와 업무 유효기간을 확인합니다."
      >
        <RecoveryNotice error={error} />
        {!data && !error ? (
          <Loading />
        ) : (
          data && (
            <div className="document-passport">
              <h3>{data.title}</h3>
              <dl>
                {[
                  ["소유자", data.owner_name],
                  ["버전", `v${data.version}`],
                  [
                    "게시 상태",
                    (
                      {
                        draft: "초안",
                        published: "게시",
                        review: "검토 중",
                        approved: "승인",
                        rejected: "반려",
                        archived: "보관",
                        stale: "오래된 문서",
                      } as any
                    )[data.status] || data.status,
                  ],
                  [
                    "내부 공유",
                    (
                      {
                        private: "나만 보기",
                        selected: "선택한 사용자",
                        workspace: "워크스페이스",
                      } as any
                    )[data.visibility],
                  ],
                  [
                    "문서 등급",
                    (
                      {
                        public: "공개",
                        internal: "내부",
                        confidential: "기밀",
                        restricted: "제한",
                      } as any
                    )[data.classification],
                  ],
                  ["마지막 변경", datetime(data.updated_at)],
                ].map(([name, value]) => (
                  <div style={{ display: "contents" }} key={name}>
                    <dt>{name}</dt>
                    <dd>{value}</dd>
                  </div>
                ))}
              </dl>
              <h4>업무 유효기간</h4>
              {data.validity_periods.length ? (
                data.validity_periods.map((p: any, i: number) => (
                  <p key={i}>
                    v{p.version} · {p.from}부터{" "}
                    {p.until ? `${p.until} 미포함` : "종료일 없음"}
                  </p>
                ))
              ) : (
                <p className="muted">
                  등록된 기간이 없습니다. 유효하다고 추정하지 않습니다.
                </p>
              )}
              <p className="notice subtle">{data.notice}</p>
              {data.can_write && (
                <Button
                  disabled={!allowCleanup}
                  onClick={() => setCleanup(true)}
                >
                  본문에서 정리 후보 확인
                </Button>
              )}
              {!allowCleanup && (
                <p className="muted">
                  편집 중인 내용을 먼저 저장한 뒤 정리 후보를 확인하세요.
                </p>
              )}
            </div>
          )
        )}
      </Modal>
      {cleanup && data && documentID && (
        <Cleanup
          documentID={documentID}
          version={data.version}
          onClose={() => setCleanup(false)}
          onApplied={() => {
            onClose();
            onUpdated?.();
            returnFocus();
          }}
        />
      )}
    </>
  );
}
function Cleanup({
  documentID,
  version,
  onClose,
  onApplied,
}: {
  documentID: string;
  version: number;
  onClose: () => void;
  onApplied: () => void;
}) {
  const { user, workspace, reload, notify } = useApp(),
    [base, setBase] = useState<Preview | null>(null),
    [review, setReview] = useState<Preview | null>(null),
    [title, setTitle] = useState(""),
    [tags, setTags] = useState<string[]>([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [stale, setStale] = useState(false),
    [consent, setConsent] = useState(false);
  const scope = `${user.id}:${workspace?.id}:${documentID}`,
    current = useRef(scope);
  current.current = scope;
  const alive = useRef(true);
  useEffect(() => {
    alive.current = true;
    const ctl = new AbortController();
    void api<Preview>(
      `/documents/${documentID}/cleanup-preview`,
      "POST",
      { expected_version: version },
      { signal: ctl.signal },
    )
      .then((v) => {
        if (!ctl.signal.aborted && current.current === scope) {
          setBase(v);
          setTitle(v.after.title);
          setTags(v.after.tags);
        }
      })
      .catch((e) => {
        if (!ctl.signal.aborted && current.current === scope) setError(e);
      });
    return () => {
      alive.current = false;
      ctl.abort();
    };
  }, [scope]);
  useEffect(() => {
    if (version !== base?.version && base) {
      setStale(true);
      setReview(null);
      setError(
        new Error(
          "원문 버전이 바뀌었습니다. 닫고 현재 문서에서 다시 확인하세요.",
        ),
      );
    }
  }, [version, base?.version]);
  const fresh = () => alive.current && current.current === scope;
  async function preview() {
    if (!base || busy || stale) return;
    setBusy(true);
    setError(null);
    setConsent(false);
    try {
      const result = await api<Preview>(
        `/documents/${documentID}/cleanup-preview`,
        "POST",
        { expected_version: base.version, title, tags },
      );
      if (fresh()) setReview(result);
    } catch (e) {
      if (fresh()) {
        setError(e);
        setReview(null);
      }
    } finally {
      if (fresh()) setBusy(false);
    }
  }
  async function apply() {
    if (!review || !consent || busy || stale) return;
    setBusy(true);
    setError(null);
    try {
      await api(`/documents/${documentID}`, "PUT", {
        version: review.version,
        title: review.after.title,
        tags: review.after.tags,
        markdown: review.after.markdown,
      });
      if (!fresh()) return;
      await reload();
      if (fresh()) {
        notify("선택한 제목과 태그를 현재 정책으로 저장했습니다.");
        onApplied();
      }
    } catch (e) {
      if (fresh()) {
        setError(e);
        if (e instanceof ApiError && e.status === 409) setStale(true);
      }
    } finally {
      if (fresh()) setBusy(false);
    }
  }
  return (
    <Modal
      open
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
      title="본문 기반 정리 제안"
      description="규칙 기반 후보를 고르고 원문 변경을 비교합니다. 자동 이동·공유·게시하지 않습니다."
      wide
    >
      <div className="workset-cleanup">
        <RecoveryNotice error={error} dirty={!!review} />
        {!base && !error ? (
          <Loading />
        ) : base && !review ? (
          <>
            <p className="notice subtle">{base.notice}</p>
            <Field label="본문 기반 제목 후보">
              <select
                value={title}
                disabled={busy || stale}
                onChange={(e) => setTitle(e.target.value)}
              >
                {base.title_candidates.map((t) => (
                  <option value={t} key={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>
            <fieldset>
              <legend>본문 기반 태그 후보</legend>
              <div className="checkbox-options">
                {base.tag_candidates.map((t) => (
                  <label key={t}>
                    <input
                      type="checkbox"
                      disabled={busy || stale}
                      checked={tags.includes(t)}
                      onChange={(e) =>
                        setTags((old) =>
                          e.target.checked
                            ? [...old, t]
                            : old.filter((v) => v !== t),
                        )
                      }
                    />
                    {t}
                  </label>
                ))}
              </div>
            </fieldset>
            <div className="modal-actions">
              <Button disabled={busy || stale} onClick={() => void preview()}>
                선택한 변경 비교
              </Button>
            </div>
          </>
        ) : (
          review && (
            <ChangeReview
              title="제목·태그와 원문 변경 확인"
              changes={[
                {
                  label: "제목",
                  before: review.before.title,
                  after: review.after.title,
                },
                {
                  label: "태그",
                  before: review.before.tags.join(", "),
                  after: review.after.tags.join(", "),
                },
                {
                  label: "Markdown 정본",
                  before: <pre>{review.before.markdown}</pre>,
                  after: <pre>{review.after.markdown}</pre>,
                },
              ]}
              warnings={[
                "현재 문서 버전이 달라지면 적용하지 않습니다. 게시 승인과 민감정보 정책을 그대로 따릅니다.",
                ...(review.front_matter_reformatted
                  ? [
                      "Front Matter의 tags를 바꾸며 YAML 서식을 다시 직렬화합니다. 다른 키·주석과 Front Matter 뒤 본문은 보존합니다.",
                    ]
                  : []),
              ]}
              confirmLabel="확인한 정리 적용"
              disabled={!consent || stale}
              busy={busy}
              onCancel={() => {
                setReview(null);
                setConsent(false);
              }}
              onConfirm={() => void apply()}
            >
              <label className="consent-check">
                <input
                  type="checkbox"
                  checked={consent}
                  disabled={busy || stale}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                변경 전후를 확인했고 이 제목과 태그를 저장합니다.
              </label>
            </ChangeReview>
          )
        )}
      </div>
    </Modal>
  );
}
