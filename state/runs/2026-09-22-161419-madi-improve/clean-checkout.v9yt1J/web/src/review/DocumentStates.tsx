import { useEffect, useState } from "react";
import { Save, Users, CheckCircle2, Globe } from "lucide-react";
import { api, type Doc } from "../api";
import { statusNames } from "../ui";
import "./document-states.css";
export function DocumentStates({
  doc,
  dirty,
  saving,
  owner,
  onSharing,
  onPublishing,
  onExternal,
}: {
  doc: Doc;
  dirty: boolean;
  saving: boolean;
  owner: boolean;
  onSharing: () => void;
  onPublishing: () => void;
  onExternal: () => void;
}) {
  const [external, setExternal] = useState<number | null>(null),
    [unavailable, setUnavailable] = useState(false);
  useEffect(() => {
    let active = true;
    setExternal(null);
    setUnavailable(false);
    if (!owner) return;
    const load = () =>
      api<Record<string, any>[]>(`/documents/${doc.id}/public-shares`)
        .then((rows) => {
          if (active) {
            setExternal(
              rows.filter(
                (v) =>
                  !v.revoked_at &&
                  !v.source_owner_changed &&
                  new Date(v.expires_at) > new Date(),
              ).length,
            );
            setUnavailable(false);
          }
        })
        .catch(() => {
          if (active) {
            setExternal(null);
            setUnavailable(true);
          }
        });
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [doc.id, doc.version, owner]);
  return (
    <div className="document-states" aria-label="문서 저장과 공개 상태">
      <div>
        <Save size={16} />
        <span>
          <small>저장</small>
          <strong>
            {saving
              ? "서버 확인 중"
              : dirty
                ? "미확정 변경 있음"
                : `서버 버전 ${doc.version}`}
          </strong>
        </span>
      </div>
      <button type="button" onClick={onSharing} title="기본 공유 범위입니다. 상위 문서·공간의 접근 제한도 함께 적용됩니다.">
        <Users size={16} />
        <span>
          <small>내부 공유</small>
          <strong>
            {doc.visibility === "private"
              ? "나만 보기"
              : doc.visibility === "selected"
                ? "선택한 사용자"
                : "워크스페이스"}
          </strong>
        </span>
      </button>
      <button type="button" onClick={onPublishing}>
        <CheckCircle2 size={16} />
        <span>
          <small>게시 단계</small>
          <strong>{statusNames[doc.status] || doc.status}</strong>
        </span>
      </button>
      <button type="button" disabled={!owner} onClick={onExternal} title="미폐기·미만료 링크 수입니다. 현재 공유 정책과 비밀번호·IP 제한에 따라 접근 가능 여부가 달라집니다.">
        <Globe size={16} />
        <span>
          <small>외부 링크</small>
          <strong>
            {!owner
              ? "소유자만 확인"
              : unavailable
                ? "확인 필요"
                : external === null
                  ? "확인 중"
                  : external
                    ? `미만료 ${external}개`
                    : "없음"}
          </strong>
        </span>
      </button>
    </div>
  );
}
