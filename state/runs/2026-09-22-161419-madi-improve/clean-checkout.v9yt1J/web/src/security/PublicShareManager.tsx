import { useEffect, useState } from "react";
import { Copy, Globe, KeyRound, Plus, ShieldX } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Modal } from "../ui";
import "../channel-settings.css";
import "./style.css";
type Row = Record<string, any>;
export default function PublicShareManager({
  documentID,
}: {
  documentID: string;
}) {
  const { notify } = useApp();
  const [open, setOpen] = useState(false),
    [rows, setRows] = useState<Row[]>([]),
    [policy, setPolicy] = useState<Row>({}),
    [draft, setDraft] = useState<Row | null>(null),
    [link, setLink] = useState(""),
    [visits, setVisits] = useState<Row[] | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [revoke, setRevoke] = useState<Row | null>(null),
    [rotate, setRotate] = useState<Row | null>(null);
  const endpoint = `/documents/${documentID}/public-shares`;
  async function load() {
    const [shares, protection] = await Promise.all([
      api<Row[]>(endpoint),
      api<Row>(`/documents/${documentID}/protection`),
    ]);
    setRows(shares);
    setPolicy({
      ...protection.public_policy,
      enabled: protection.public_shares_enabled,
    });
  }
  useEffect(() => {
    if (!open) return;
    void load().catch((e) => setError(e.message));
  }, [open, documentID]);
  const newDraft = () => ({
    expires_at: new Date(
      Date.now() + Math.min(policy.max_days || 7, 7) * 86400000,
    )
      .toISOString()
      .slice(0, 16),
    password: "",
    ip_allowlist: "",
    allow_download: false,
    allow_copy: true,
    confirm_public: false,
    clear_password: false,
  });
  return (
    <>
      <Button
        onClick={() => {
          setOpen(true);
          setError("");
        }}
      >
        <Globe size={16} />
        공개 링크
      </Button>
      <Modal
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
        title="문서 공개 링크"
        wide
      >
        <ErrorBox error={error} />
        <p>
          비공개 문서도 이 링크를 가진 방문자에게 전달됩니다. 현재 소유자의
          명시적인 동의가 필요하며, 상위 등급이나 권한 제한을 우회할 수
          없습니다.
        </p>
        {!policy.enabled && (
          <p className="notice">
            관리자가 공개 링크 생성을 허용하지 않았습니다. 기존 링크 폐기는 계속
            가능합니다.
          </p>
        )}
        <Button
          disabled={!policy.enabled || busy}
          onClick={() => setDraft(newDraft())}
        >
          <Plus size={16} />
          공개 링크 만들기
        </Button>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>만료</th>
                <th>보호</th>
                <th>상태</th>
                <th>관리</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((v) => (
                <tr key={v.id}>
                  <td>{datetime(v.expires_at)}</td>
                  <td>
                    {v.password_required ? "암호 있음" : "암호 없음"} ·{" "}
                    {v.ip_allowlist.length ? "IP 제한" : "모든 IP"} ·{" "}
                    {v.allow_download ? "첨부 허용" : "첨부 차단"}
                  </td>
                  <td>
                    {v.revoked_at
                      ? "폐기됨"
                      : v.source_owner_changed
                        ? "소유자 변경 · 폐기 필요"
                        : new Date(v.expires_at) < new Date()
                          ? "만료됨"
                          : "활성"}
                  </td>
                  <td>
                    <div className="button-row">
                      <Button
                        disabled={
                          !!v.revoked_at ||
                          v.source_owner_changed ||
                          busy ||
                          !policy.enabled
                        }
                        onClick={() =>
                          setDraft({
                            ...v,
                            expires_at: new Date(v.expires_at)
                              .toISOString()
                              .slice(0, 16),
                            password: "",
                            clear_password: false,
                            confirm_public: false,
                            ip_allowlist: v.ip_allowlist.join("\n"),
                          })
                        }
                      >
                        설정
                      </Button>
                      <Button
                        disabled={
                          !!v.revoked_at || v.source_owner_changed || busy
                        }
                        onClick={() => setRotate(v)}
                      >
                        <KeyRound size={15} />
                        링크 회전
                      </Button>
                      <Button
                        disabled={!!v.revoked_at || busy}
                        onClick={() => setRevoke(v)}
                      >
                        <ShieldX size={15} />
                        폐기
                      </Button>
                      <Button
                        disabled={busy}
                        onClick={async () => {
                          try {
                            setVisits(
                              await api<Row[]>(`${endpoint}/${v.id}/visits`),
                            );
                          } catch (e) {
                            setError((e as Error).message);
                          }
                        }}
                      >
                        접근 이력
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="muted">
          원래 링크 비밀은 저장하지 않아 다시 조회할 수 없습니다. 링크 회전은
          이전 주소를 즉시 폐기합니다.
        </p>
      </Modal>
      {draft && (
        <Modal
          open
          title={draft.id ? "공개 링크 설정" : "새 공개 링크"}
          onOpenChange={() => {
            if (!busy) setDraft(null);
          }}
        >
          <ErrorBox error={error} />
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                const body = {
                  ...draft,
                  expires_at: new Date(draft.expires_at + "Z").toISOString(),
                  ip_allowlist: draft.ip_allowlist
                    .split("\n")
                    .map((v: string) => v.trim())
                    .filter(Boolean),
                };
                const result = await api<Row>(
                  endpoint + (draft.id ? `/${draft.id}` : ""),
                  draft.id ? "PUT" : "POST",
                  body,
                );
                setDraft(null);
                if (result.url) setLink(location.origin + result.url);
                await load();
                notify("공개 공유 설정을 저장했습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Field label="공유 만료 (UTC)">
              <input
                type="datetime-local"
                required
                value={draft.expires_at}
                onChange={(e) =>
                  setDraft({ ...draft, expires_at: e.target.value })
                }
              />
            </Field>
            <Field
              label={draft.id ? "새 공유 암호 (빈 값은 유지)" : "공유 암호"}
            >
              <input
                type="password"
                autoComplete="new-password"
                required={!draft.id && policy.require_password}
                value={draft.password}
                onChange={(e) =>
                  setDraft({ ...draft, password: e.target.value })
                }
              />
            </Field>
            {draft.id && !policy.require_password && (
              <label className="check-row">
                <input
                  type="checkbox"
                  checked={draft.clear_password}
                  onChange={(e) =>
                    setDraft({ ...draft, clear_password: e.target.checked })
                  }
                />
                기존 공유 암호 제거
              </label>
            )}
            <Field label="허용 IP / CIDR (줄마다 하나)">
              <textarea
                rows={3}
                value={draft.ip_allowlist}
                onChange={(e) =>
                  setDraft({ ...draft, ip_allowlist: e.target.value })
                }
                placeholder="192.0.2.0/24"
              />
            </Field>
            <label className="check-row">
              <input
                type="checkbox"
                disabled={!policy.allow_download}
                checked={draft.allow_download}
                onChange={(e) =>
                  setDraft({ ...draft, allow_download: e.target.checked })
                }
              />
              첨부 다운로드 허용
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={draft.allow_copy}
                onChange={(e) =>
                  setDraft({ ...draft, allow_copy: e.target.checked })
                }
              />
              화면 텍스트 복사 허용
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                required
                checked={draft.confirm_public}
                onChange={(e) =>
                  setDraft({ ...draft, confirm_public: e.target.checked })
                }
              />
              링크를 가진 방문자에게 이 문서를 공개함을 확인했습니다
            </label>
            <Button type="submit" variant="primary" disabled={busy}>
              공유 설정 저장
            </Button>
          </form>
        </Modal>
      )}
      {link && (
        <Modal
          open
          title="공개 링크 · 한 번만 표시"
          onOpenChange={() => setLink("")}
        >
          <p>
            주소의 # 뒤 값까지 포함해야 합니다. 외부 전달 전 암호와 만료 시각을
            확인하세요.
          </p>
          <Field label="새 공개 링크">
            <input readOnly value={link} />
          </Field>
          <Button
            onClick={async () => {
              try {
                await navigator.clipboard.writeText(link);
                notify("공개 링크를 복사했습니다");
              } catch {
                notify("링크를 직접 선택해 복사하세요", "error");
              }
            }}
          >
            <Copy size={16} />
            링크 복사
          </Button>
        </Modal>
      )}
      {rotate && (
        <Modal
          open
          title="공개 링크 회전 확인"
          onOpenChange={() => {
            if (!busy) setRotate(null);
          }}
        >
          <ErrorBox error={error} />
          <p>
            기존 주소와 암호 확인 세션이 즉시 무효화됩니다. 새 주소를 전달할
            준비가 되었는지 확인하세요.
          </p>
          <Button
            disabled={busy}
            variant="primary"
            onClick={async () => {
              setBusy(true);
              setError("");
              try {
                const value = await api<Row>(
                  `${endpoint}/${rotate.id}/rotate`,
                  "POST",
                  { revision: rotate.revision },
                );
                setRotate(null);
                setLink(location.origin + value.url);
                await load();
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            기존 링크 폐기하고 회전
          </Button>
        </Modal>
      )}
      {revoke && (
        <Modal
          open
          title="공개 링크 폐기"
          onOpenChange={() => {
            if (!busy) setRevoke(null);
          }}
        >
          <ErrorBox error={error} />
          <p>
            이 링크와 기존 암호 확인 세션을 즉시 사용할 수 없게 합니다. 이미
            다운로드한 자료는 회수할 수 없습니다.
          </p>
          <Button
            disabled={busy}
            variant="danger"
            onClick={async () => {
              setBusy(true);
              try {
                await api(`${endpoint}/${revoke.id}`, "DELETE");
                setRevoke(null);
                await load();
                notify("공개 링크를 폐기했습니다");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            공개 링크 폐기 확인
          </Button>
        </Modal>
      )}
      {visits && (
        <Modal
          open
          title="공개 링크 접근 이력"
          onOpenChange={() => setVisits(null)}
        >
          <p>IP 원문은 저장하지 않고 해시로 구분합니다.</p>
          {visits.length ? (
            visits.map((v) => (
              <p key={v.id}>
                {datetime(v.created_at)} ·{" "}
                {{
                  unlock: "암호 확인",
                  read: "문서 열람",
                  download: "첨부 다운로드",
                }[v.action as string] || v.action}{" "}
                · {v.result === "allowed" ? "허용" : "거부"}
              </p>
            ))
          ) : (
            <p>접근 이력이 없습니다.</p>
          )}
        </Modal>
      )}
    </>
  );
}
