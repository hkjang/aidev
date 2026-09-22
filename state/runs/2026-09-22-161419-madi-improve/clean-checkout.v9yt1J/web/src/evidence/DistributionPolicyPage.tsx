import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, Modal, PageHeading } from "../ui";
import "./evidence.css";
import "./distribution.css";

type Policy = {
  enabled: boolean;
  revision: number;
  instance_id: string;
  max_valid_days: number;
};
type Key = {
  id: string;
  kind: "trusted" | "signing";
  label: string;
  source_instance: string;
  source_key_id: string;
  public_key: string;
  fingerprint: string;
  revision: number;
  revoked_at: string | null;
  created_at: string;
};
type Settings = {
  policy: Policy;
  keys: Key[];
  history: {
    revision: number;
    enabled: boolean;
    max_valid_days: number;
    created_at: string;
  }[];
  notice: string;
  max_files: number;
  max_bytes: number;
};
type Confirmation = {
  path: string;
  method: string;
  body: Record<string, unknown>;
  title: string;
  description: string;
};

export default function DistributionPolicyPage() {
  const { user, notify } = useApp();
  const [data, setData] = useState<Settings | null>(null),
    [enabled, setEnabled] = useState(false),
    [days, setDays] = useState(7);
  const [kind, setKind] = useState<"signing" | "trusted">("signing"),
    [label, setLabel] = useState(""),
    [source, setSource] = useState(""),
    [sourceKey, setSourceKey] = useState(""),
    [publicKey, setPublicKey] = useState(""),
    [fingerprint, setFingerprint] = useState("");
  const [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0),
    [confirm, setConfirm] = useState<Confirmation | null>(null),
    [consent, setConsent] = useState(false),
    [selected, setSelected] = useState<Key | null>(null),
    [retrust, setRetrust] = useState("");
  const generation = useRef(0);
  useEffect(() => {
    const current = ++generation.current;
    let active = true;
    setData(null);
    setSelected(null);
    setConfirm(null);
    setConsent(false);
    setError("");
    void api<Settings>("/admin/knowledge-distribution")
      .then((v) => {
        if (active && current === generation.current) {
          setData(v);
          setEnabled(v.policy.enabled);
          setDays(v.policy.max_valid_days);
        }
      })
      .catch((e) => {
        if (active && current === generation.current) setError(e.message);
      });
    return () => {
      active = false;
      generation.current++;
    };
  }, [user?.id, refresh]);
  const ask = (value: Confirmation) => {
    setConsent(false);
    setConfirm(value);
  };
  const submit = async () => {
    if (!confirm || !consent || busy) return;
    const pending = confirm,
      current = generation.current;
    setBusy(true);
    setError("");
    try {
      await api(pending.path, pending.method, pending.body);
      if (current === generation.current) {
        setConfirm(null);
        setConsent(false);
        setRefresh((v) => v + 1);
        notify("배포 운영 설정을 저장했습니다.");
      }
    } catch (e) {
      if (current === generation.current) {
        setConfirm(null);
        setConsent(false);
        setError((e as Error).message);
      }
    } finally {
      if (current === generation.current) setBusy(false);
    }
  };
  if (user?.role !== "admin") return <p>서비스 관리자 권한이 필요합니다.</p>;
  return (
    <div className="page evidence-page distribution-page">
      <PageHeading
        title="망별 지식 배포 설정"
        description="서비스 운영 정책과 공개키 신뢰를 관리합니다. 개인 문서 작성·반출은 사용자 화면에서 진행합니다."
        actions={
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => {
              if (
                window.confirm(
                  "입력 중인 설정을 버리고 현재 서버 설정을 다시 불러올까요?",
                )
              )
                setRefresh((v) => v + 1);
            }}
          >
            현재 설정 다시 읽기
          </Button>
        }
      />
      <Link to="/app/knowledge-distribution">내 지식 반출·반입으로 이동</Link>
      <ErrorBox error={error} />
      {!data ? (
        <Loading />
      ) : (
        <>
          <p className="notice">{data.notice}</p>
          <section className="card">
            <h2>배포 운영 정책</h2>
            <p>
              망 식별자는 이 설치의 식별자입니다. 데이터베이스 복원은 같은
              식별자를 유지하되, 정책을 끄고 모든 키를 철회합니다. 반출 키는
              새로 생성하고 외부 신뢰 키는 지문을 다시 확인해야 합니다.
            </p>
            <Field label="현재 망 식별자">
              <input value={data.policy.instance_id} readOnly />
            </Field>
            <label className="check">
              <input
                type="checkbox"
                checked={enabled}
                disabled={busy}
                onChange={(e) => setEnabled(e.target.checked)}
              />
              서명 반출·반입 사용
            </label>
            <Field
              label="최대 패키지 유효기간(일)"
              hint="1~365일. 이미 생성한 결과의 설정 revision이 바뀌면 새 반출을 준비해야 합니다."
            >
              <input
                type="number"
                min={1}
                max={365}
                value={days}
                onChange={(e) => setDays(Number(e.target.value))}
                disabled={busy}
              />
            </Field>
            <p>
              1개 패키지의 파일 합계는 50MiB, 최대 {data.max_files}개입니다. ZIP
              파일은 매니페스트·서명·압축 부가 정보를 포함합니다. 원본 내보내기
              결과와 배포 결과를 각각 개인별 200MiB까지 임시 보관합니다.
            </p>
            <Button
              disabled={
                busy || !Number.isInteger(days) || days < 1 || days > 365
              }
              onClick={() =>
                ask({
                  path: "/admin/knowledge-distribution",
                  method: "PUT",
                  body: {
                    revision: data.policy.revision,
                    enabled,
                    max_valid_days: days,
                    consent: true,
                  },
                  title: "배포 정책 변경",
                  description: `${enabled ? "서명 배포를 사용" : "서명 배포를 중지"}하고 최대 유효기간을 ${days}일로 설정합니다. 기존 반출 준비 결과는 새 설정으로 다시 생성해야 합니다. 만료·이미 전달된 사본의 회수를 보장하지 않습니다.`,
                })
              }
            >
              정책 변경 확인
            </Button>
            <details>
              <summary>정책 변경 이력</summary>
              <p>
                과거 값을 선택해도 즉시 변경하지 않습니다. 현재 revision에서
                새로운 설정으로 확인·저장하세요.
              </p>
              {data.history.map((h) => (
                <p key={h.revision}>
                  v{h.revision} · {datetime(h.created_at)} ·{" "}
                  {h.enabled ? "사용" : "중지"} · {h.max_valid_days}일{" "}
                  <Button
                    variant="secondary"
                    disabled={busy}
                    onClick={() => {
                      setEnabled(h.enabled);
                      setDays(h.max_valid_days);
                    }}
                  >
                    이 값을 입력란에 불러오기
                  </Button>
                </p>
              ))}
            </details>
          </section>
          <section className="card">
            <h2>키 생성·등록</h2>
            <p>
              새 반출 키를 만들면 기존 키가 자동으로 철회되지 않습니다. 수신망에
              새 공개키를 전달하고 확인한 뒤 이전 키를 명시적으로 철회하세요.
              개인키는 서버에서 암호화하며 화면이나 API 응답으로 내보내지
              않습니다.
            </p>
            <div className="form-grid">
              <Field label="키 용도">
                <select
                  value={kind}
                  onChange={(e) =>
                    setKind(e.target.value as "signing" | "trusted")
                  }
                  disabled={busy}
                >
                  <option value="signing">이 망의 반출 서명 키 생성</option>
                  <option value="trusted">외부 망의 공개키 신뢰 등록</option>
                </select>
              </Field>
              <Field label="키 이름">
                <input
                  value={label}
                  onChange={(e) => setLabel(e.target.value)}
                  maxLength={200}
                  disabled={busy}
                />
              </Field>
            </div>
            {kind === "trusted" && (
              <>
                <p role="note">
                  패키지 안에 들어 있는 키만 보고 신뢰하지 마세요. 반출망
                  관리자와 별도 경로로 망 식별자·키 ID·공개키·SHA256 지문을
                  확인한 뒤 입력하세요.
                </p>
                <Field label="반출망 식별자">
                  <input
                    value={source}
                    onChange={(e) => setSource(e.target.value)}
                    disabled={busy}
                  />
                </Field>
                <Field label="반출망 키 ID">
                  <input
                    value={sourceKey}
                    onChange={(e) => setSourceKey(e.target.value)}
                    disabled={busy}
                  />
                </Field>
                <Field label="공개키(base64)">
                  <input
                    value={publicKey}
                    onChange={(e) => setPublicKey(e.target.value)}
                    disabled={busy}
                    autoComplete="off"
                  />
                </Field>
                <Field label="별도 확인한 공개키 SHA256 지문">
                  <input
                    value={fingerprint}
                    onChange={(e) => setFingerprint(e.target.value)}
                    disabled={busy}
                    autoComplete="off"
                  />
                </Field>
              </>
            )}
            <Button
              disabled={
                busy ||
                !label.trim() ||
                (kind === "trusted" &&
                  (!source.trim() ||
                    !sourceKey.trim() ||
                    !publicKey.trim() ||
                    !fingerprint.trim()))
              }
              onClick={() =>
                ask({
                  path: "/admin/knowledge-distribution/keys",
                  method: "POST",
                  body: {
                    kind,
                    label: label.trim(),
                    source_instance: source.trim(),
                    source_key_id: sourceKey.trim(),
                    public_key: publicKey.trim(),
                    fingerprint: fingerprint.trim(),
                    consent: true,
                  },
                  title:
                    kind === "signing"
                      ? "새 반출 서명 키 생성"
                      : "외부 반출망 공개키 신뢰 등록",
                  description:
                    kind === "signing"
                      ? `“${label}” 이름으로 Ed25519 키를 생성합니다. 기존 키는 유지됩니다.`
                      : `“${label}” · ${source} 망의 키 ${sourceKey}를 신뢰합니다. 별도로 확인한 SHA256 지문은 ${fingerprint}입니다. 이 키의 패키지도 수신망 정책과 이관 검토를 거칩니다.`,
                })
              }
            >
              {kind === "signing" ? "생성 내용 확인" : "신뢰 등록 내용 확인"}
            </Button>
          </section>
          <section className="card">
            <h2>현재 키와 철회 이력</h2>
            <p>
              반출 키 철회는 앞으로의 서명·내려받기를 차단합니다. 수신망의 신뢰
              철회는 새 반입을 차단합니다. 이미 반입·전달된 사본을 삭제하지는
              않습니다.
            </p>
            {data.keys.map((k) => (
              <article className="distribution-run" key={k.id}>
                <strong>
                  {k.label} · {k.kind === "signing" ? "반출 서명" : "외부 신뢰"}{" "}
                  · {k.revoked_at ? "철회됨" : "사용 중"}
                </strong>
                <code>{k.fingerprint}</code>
                <span>
                  v{k.revision} · {datetime(k.created_at)}
                </span>
                <div className="button-row">
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setSelected(k);
                      setRetrust("");
                    }}
                  >
                    공개 정보·관리
                  </Button>
                  {!k.revoked_at && (
                    <Button
                      variant="secondary"
                      disabled={busy}
                      onClick={() =>
                        ask({
                          path: `/admin/knowledge-distribution/keys/${k.id}/revoke`,
                          method: "POST",
                          body: {
                            revision: k.revision,
                            confirmation: "REVOKE",
                          },
                          title: "키 사용 철회",
                          description: `“${k.label}” 키를 철회합니다. ${k.kind === "signing" ? "새 서명과 기존 배포 결과 다운로드" : "이 키로 검증하는 새 반입"}를 차단하지만 이미 전달·반입된 자료는 삭제하지 않습니다.`,
                        })
                      }
                    >
                      키 철회
                    </Button>
                  )}
                </div>
              </article>
            ))}
          </section>
        </>
      )}
      <Modal
        open={!!selected}
        onOpenChange={(v) => !v && setSelected(null)}
        title="공개키 정보와 신뢰 상태"
      >
        {selected && (
          <>
            <p>
              {selected.label} · {selected.revoked_at ? "철회됨" : "사용 중"}
            </p>
            <Field label="반출망 식별자">
              <input readOnly value={selected.source_instance} />
            </Field>
            <Field label="반출망 키 ID">
              <input readOnly value={selected.source_key_id} />
            </Field>
            <Field label="공개키(base64)">
              <textarea readOnly value={selected.public_key} />
            </Field>
            <Field label="공개키 SHA256 지문">
              <textarea readOnly value={selected.fingerprint} />
            </Field>
            {selected.kind === "trusted" && selected.revoked_at && (
              <>
                <p>
                  신뢰를 다시 활성화하려면 반출망과 별도로 지문을 재확인하세요.
                  유출·침해로 철회한 키는 재신뢰하지 말고 새 공개키를
                  등록하세요.
                </p>
                <Field label="재확인한 전체 SHA256 지문">
                  <input
                    value={retrust}
                    onChange={(e) => setRetrust(e.target.value)}
                  />
                </Field>
                <Button
                  disabled={busy || retrust.trim() !== selected.fingerprint}
                  onClick={() => {
                    const k = selected;
                    setSelected(null);
                    ask({
                      path: `/admin/knowledge-distribution/keys/${k.id}/trust`,
                      method: "POST",
                      body: {
                        revision: k.revision,
                        fingerprint: retrust.trim(),
                        confirmation: "TRUST",
                      },
                      title: "외부 공개키 신뢰 재확인",
                      description: `“${k.label}” 키의 SHA256 ${retrust.trim()}를 별도 경로로 확인하고 새 반입을 허용합니다. 서명 내용 자체의 사실성이나 게시 권한을 부여하지 않습니다.`,
                    });
                  }}
                >
                  신뢰 재확인 내용 검토
                </Button>
              </>
            )}
          </>
        )}
      </Modal>
      <Modal
        open={!!confirm}
        onOpenChange={(v) => {
          if (!v) {
            setConfirm(null);
            setConsent(false);
          }
        }}
        title={confirm?.title || "설정 변경 확인"}
      >
        <p>{confirm?.description}</p>
        <label className="check">
          <input
            type="checkbox"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
          />
          변경 범위와 영향을 확인했습니다.
        </label>
        <Button disabled={busy || !consent} onClick={() => void submit()}>
          확인한 설정 저장
        </Button>
      </Modal>
    </div>
  );
}
