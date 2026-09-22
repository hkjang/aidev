import { useEffect, useState } from "react";
import { Save, ShieldCheck } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, PageHeading } from "../ui";
import "../channel-settings.css";
import "./style.css";
type Row = Record<string, any>;
const detectors = [
  ["rrn", "주민번호 형식"],
  ["email", "이메일"],
  ["phone", "전화번호 형식"],
  ["payment", "결제번호 형식"],
  ["account", "계좌번호 형식"],
];
const classes = [
  ["public", "공개"],
  ["internal", "내부"],
  ["confidential", "기밀"],
  ["restricted", "제한"],
];
const actionLabels: Record<string, string> = {
  "document.detected": "문서 검사",
  "document.masked": "문서 마스킹",
  "metadata.masked": "메타데이터 마스킹",
  "attachment.scan": "첨부 검사",
};
const modeLabels: Record<string, string> = {
  warn: "경고",
  block: "차단",
  mask: "마스킹",
  audit: "감사",
};
export default function ProtectionPage() {
  const { notify } = useApp();
  const [settings, setSettings] = useState<Row | null>(null),
    [revision, setRevision] = useState(0),
    [history, setHistory] = useState<Row[]>([]),
    [events, setEvents] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [terms, setTerms] = useState("");
  async function load() {
    const [policy, h, e] = await Promise.all([
      api<Row>("/admin/information-protection"),
      api<Row[]>("/admin/information-protection/history"),
      api<Row[]>("/admin/information-protection/events"),
    ]);
    setSettings(policy.settings);
    setTerms(policy.settings.custom_terms.join("\n"));
    setRevision(policy.revision);
    setHistory(h);
    setEvents(e);
  }
  useEffect(() => {
    let active = true;
    void load().catch((e) => active && setError(e.message));
    return () => {
      active = false;
    };
  }, []);
  const change = (key: string, value: any) =>
    settings && setSettings({ ...settings, [key]: value });
  return (
    <>
      <PageHeading
        eyebrow="INFORMATION PROTECTION"
        title="정보보호 정책"
        description="민감정보 탐지, 상속된 문서 등급과 외부 공유의 허용 경계를 관리합니다."
      />
      <ErrorBox error={error} />
      {!settings ? (
        <Loading />
      ) : (
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError("");
            try {
              const result = await api<Row>(
                "/admin/information-protection",
                "PUT",
                {
                  revision,
                  settings: {
                    ...settings,
                    custom_terms: terms
                      .split("\n")
                      .map((v) => v.trim())
                      .filter(Boolean),
                  },
                },
              );
              setRevision(result.revision);
              await load();
              notify("정보보호 정책을 저장했습니다");
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <section className="card">
            <h2>문서·첨부 민감정보</h2>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!settings.enabled}
                onChange={(e) => change("enabled", e.target.checked)}
              />
              새 문서 저장·첨부 검사 활성화
            </label>
            <Field label="민감정보 처리 방식">
              <select
                value={settings.mode}
                onChange={(e) => change("mode", e.target.value)}
              >
                <option value="warn">경고하고 저장</option>
                <option value="block">저장 차단</option>
                <option value="mask">마스킹 후 저장</option>
                <option value="audit">감사만 기록</option>
              </select>
            </Field>
            <div className="protection-options">
              {detectors.map(([key, label]) => (
                <label className="check-row" key={key}>
                  <input
                    type="checkbox"
                    checked={settings.detectors.includes(key)}
                    onChange={(e) =>
                      change(
                        "detectors",
                        e.target.checked
                          ? [...settings.detectors, key]
                          : settings.detectors.filter((v: string) => v !== key),
                      )
                    }
                  />
                  {label}
                </label>
              ))}
            </div>
            <Field label="사내 지정 민감어 (줄마다 하나)">
              <textarea
                value={terms}
                onChange={(e) => setTerms(e.target.value)}
                rows={4}
                placeholder="프로젝트 기밀 명칭"
              />
            </Field>
            <Field label="검사할 수 없는 첨부">
              <select
                value={settings.unscannable}
                onChange={(e) => change("unscannable", e.target.value)}
              >
                <option value="warn">검사 불가를 기록하고 허용</option>
                <option value="block">업로드 차단</option>
              </select>
            </Field>
            <p className="muted">
              UTF-8 텍스트 4MB까지 본문을 검사합니다. 이미지·PDF·압축·기타 이진
              파일과 큰 파일은 검사 불가로 구분합니다. 패턴 탐지는 오탐·누락이
              있으며 악성코드 검사나 OCR을 대신하지 않습니다.
            </p>
            <p className="muted">
              차단·마스킹에서는 숨은 Yjs 변경 이력을 검사할 수 없어 공동 편집을
              중지하고 원문 편집을 사용합니다. 기존 버전·백업의 과거 원문을 자동
              삭제하는 정책은 아닙니다.
            </p>
          </section>
          <section className="card">
            <h2>문서 등급과 워터마크</h2>
            <p>
              문서, 모든 상위 문서와 공간의 가장 높은 등급을 적용합니다. 낮은
              하위 등급으로 상위 정책을 우회할 수 없습니다.
            </p>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!settings.watermark_enabled}
                onChange={(e) => change("watermark_enabled", e.target.checked)}
              />
              화면·인쇄 워터마크 활성화
            </label>
            <Field label="워터마크 최소 등급">
              <select
                value={settings.watermark_min_classification}
                onChange={(e) =>
                  change("watermark_min_classification", e.target.value)
                }
              >
                {classes.map(([v, l]) => (
                  <option key={v} value={v}>
                    {l}
                  </option>
                ))}
              </select>
            </Field>
          </section>
          <section className="card">
            <h2>공개 링크 허용 정책</h2>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!settings.public_shares_enabled}
                onChange={(e) =>
                  change("public_shares_enabled", e.target.checked)
                }
              />
              문서 소유자의 명시적 외부 공유 허용
            </label>
            <div className="form-grid">
              <Field label="공유 가능한 최대 등급">
                <select
                  value={settings.public_share_max_classification}
                  onChange={(e) =>
                    change("public_share_max_classification", e.target.value)
                  }
                >
                  {classes.map(([v, l]) => (
                    <option key={v} value={v}>
                      {l}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="공유 최대 유효 기간 (일)">
                <input
                  type="number"
                  min={1}
                  max={365}
                  required
                  value={settings.public_share_max_days}
                  onChange={(e) =>
                    change("public_share_max_days", Number(e.target.value))
                  }
                />
              </Field>
            </div>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!settings.public_share_require_password}
                onChange={(e) =>
                  change("public_share_require_password", e.target.checked)
                }
              />
              공유 암호 필수
            </label>
            <label className="check-row">
              <input
                type="checkbox"
                checked={!!settings.public_share_allow_download}
                onChange={(e) =>
                  change("public_share_allow_download", e.target.checked)
                }
              />
              소유자가 공유 첨부 다운로드를 허용할 수 있음
            </label>
            <p className="muted">
              링크는 만료·허용 IP·현재 소유자 권한·현재 분류를 매번 확인하며
              검색엔진 색인을 허용하지 않습니다. 화면 복사 제한은 접근 통제나
              DRM이 아닙니다.
            </p>
          </section>
          <Button type="submit" variant="primary" disabled={busy}>
            <Save size={17} />
            정보보호 정책 저장
          </Button>
        </form>
      )}
      <section className="card">
        <h2>
          <ShieldCheck size={18} /> 최근 검출 감사
        </h2>
        <p className="muted">
          검출된 실제 값이나 문서 본문은 기록하지 않고 종류·개수만 보관합니다.
        </p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>시간</th>
                <th>조치</th>
                <th>검출 종류·수</th>
              </tr>
            </thead>
            <tbody>
              {events.map((v) => (
                <tr key={v.id}>
                  <td>{datetime(v.created_at)}</td>
                  <td>
                    {actionLabels[v.action] || v.action} ·{" "}
                    {modeLabels[v.mode] || v.mode}
                  </td>
                  <td>
                    {v.findings
                      .map(
                        (f: Row) =>
                          `${detectors.find(([id]) => id === f.kind)?.[1] || ({ custom_term: "사내 지정 민감어", unscannable: "검사 불가" } as Record<string, string>)[f.kind] || f.kind} ${f.count}건`,
                      )
                      .join(", ")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section className="card">
        <h2>정책 변경 이력</h2>
        {history.map((v) => (
          <div className="button-row" key={v.id}>
            <span>
              v{v.revision} · {datetime(v.created_at)} · {v.user_name}
            </span>
            <Button
              disabled={busy}
              onClick={() => {
                setSettings(v.data);
                setTerms(v.data.custom_terms.join("\n"));
                notify("이전 설정을 불러왔습니다. 확인 후 저장하세요");
              }}
            >
              이전 설정 불러오기
            </Button>
          </div>
        ))}
      </section>
    </>
  );
}
