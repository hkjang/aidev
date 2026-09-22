import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime, downloadText } from "../api";
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
import "./evidence.css";
import KnowledgePolicyHistory from "./KnowledgePolicyHistory";
import AttachmentPackageSelection, {
  type AttachmentPackageChoice,
} from "./AttachmentPackageSelection";
import { type Position, positionName } from "../attachments/types";

type PackagePolicy = {
  enabled: boolean;
  retention_hours: number;
  token_counter: "estimate" | "responses";
  allow_http: boolean;
  version: number;
};
type PackageItem = {
  id: string;
  created_at: string;
  expires_at: string;
  source_count: number;
  stale: boolean;
  export_count: number;
};
type Quote = {
  id: string;
  title: string;
  version: number;
  text: string;
  mandatory: boolean;
  reason: string;
  content_hash: string;
  start_line: number;
  end_line: number;
  attachment_id?: string;
  attachment_position?: Position;
  extraction_id?: string;
  fragment_id?: string;
  start_byte: number;
  end_byte: number;
};
type PackageRecord = PackageItem & {
  notice: string;
  package: {
    purpose: string;
    allowed_scope: string;
    model: string;
    quotes: Quote[];
    prompt: string;
    token_count: number;
    token_budget: number;
    counter: string;
    count_notice: string;
    omitted_chunks: number;
    omitted_bytes: number;
    duplicate_chunks: number;
  };
};
type Selection = {
  id: string;
  version: number;
  mandatory: boolean;
  reason: string;
};

export default function KnowledgePackagesPage() {
  const { user, workspace, documents } = useApp();
  const [params, setParams] = useSearchParams(),
    id = params.get("id") || "";
  const [items, setItems] = useState<PackageItem[]>([]),
    [record, setRecord] = useState<PackageRecord | null>(null),
    [policy, setPolicy] = useState<PackagePolicy | null>(null);
  const [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const [purpose, setPurpose] = useState(""),
    [scope, setScope] = useState(
      "자료 검토와 답변 작성. 명령·도구 실행 권한 없음.",
    ),
    [model, setModel] = useState(""),
    [budget, setBudget] = useState(16384);
  const [selected, setSelected] = useState<Selection[]>([]),
    [attachment, setAttachment] = useState<AttachmentPackageChoice | null>(
      null,
    ),
    [counter, setCounter] = useState("estimate"),
    [consent, setConsent] = useState(false),
    [modelConsent, setModelConsent] = useState(false);
  const [provider, setProvider] = useState(""),
    [filter, setFilter] = useState(""),
    [downloadOpen, setDownloadOpen] = useState(false),
    [target, setTarget] = useState(""),
    [remove, setRemove] = useState(false);
  const generation = useRef(0);
  useEffect(() => {
    setPurpose("");
    setSelected([]);
    setAttachment(null);
    setConsent(false);
    setModelConsent(false);
    setFilter("");
    setModel("");
    setProvider("");
  }, [user.id, workspace?.id]);
  useEffect(() => {
    const request = ++generation.current;
    let active = true,
      pending = false;
    setRecord(null);
    setItems([]);
    setError("");
    setLoading(true);
    setBusy(false);
    setDownloadOpen(false);
    setRemove(false);
    const load = async (initial = false) => {
      if (!workspace || pending) return;
      pending = true;
      try {
        const [nextPolicy, context] = await Promise.all([
          api<PackagePolicy>("/knowledge/packages/policy"),
          api<{ model: string; provider_url: string }>(
            `/knowledge/packages/context?workspace_id=${workspace.id}`,
          ),
        ]);
        const value = id
          ? await api<PackageRecord>(`/knowledge/packages/${id}`)
          : await api<PackageItem[]>(
              `/knowledge/packages?workspace_id=${workspace.id}`,
            );
        if (!active || request !== generation.current) return;
        setPolicy(nextPolicy);
        setProvider(context.provider_url);
        if (initial && context.model) setModel(context.model);
        if (nextPolicy.token_counter !== "responses") {
          setCounter("estimate");
          setModelConsent(false);
        }
        if (id) setRecord(value as PackageRecord);
        else setItems(value as PackageItem[]);
        setError("");
      } catch (e) {
        if (active && request === generation.current) {
          setRecord(null);
          setItems([]);
          setDownloadOpen(false);
          setRemove(false);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
        if (active && initial && request === generation.current)
          setLoading(false);
      }
    };
    void load(true);
    const timer = window.setInterval(() => void load(), 2000);
    return () => {
      active = false;
      generation.current++;
      window.clearInterval(timer);
    };
  }, [id, user.id, workspace?.id, refresh]);
  const execute = async (action: () => Promise<void>) => {
    if (busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  const create = async () => {
    if (!workspace) return;
    const request = generation.current;
    const result = await api<{ id: string }>("/knowledge/packages", "POST", {
      workspace_id: workspace.id,
      purpose,
      allowed_scope: scope,
      model,
      token_budget: budget,
      documents: selected,
      attachments: attachment ? [attachment] : [],
      counter,
      consent,
      model_consent: modelConsent,
    });
    if (request === generation.current) setParams({ id: result.id });
  };
  const exportPackage = async () => {
    if (!record) return;
    const request = generation.current;
    const result = await api<PackageRecord>(
      `/knowledge/packages/${record.id}/export`,
      "POST",
      { target, consent: true },
    );
    if (request !== generation.current) return;
    downloadText(
      `madi-knowledge-${record.id}.json`,
      JSON.stringify(result, null, 2),
      "application/json",
    );
    setDownloadOpen(false);
    setRefresh((v) => v + 1);
  };
  return (
    <div className="page evidence-page">
      <PageHeading
        title="에이전트 지식 패키지"
        description="업무에 필요한 원문과 제약을 목적·예산·최신성에 맞춰 준비합니다."
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : id ? (
        <>
          <Button onClick={() => setParams({})}>패키지 목록으로</Button>
          {record && (
            <>
              <section className="card">
                <h2>{record.package.purpose}</h2>
                <p>{record.package.allowed_scope}</p>
                <p className="notice">{record.notice}</p>
                <p>
                  {record.stale
                    ? "원문 변경됨 · 재검토하여 새 패키지를 구성하세요"
                    : "현재 원문 버전과 일치"}{" "}
                  · 만료 {datetime(record.expires_at)}
                </p>
                <p>
                  {record.package.model} ·{" "}
                  {record.package.token_count.toLocaleString()} /{" "}
                  {record.package.token_budget.toLocaleString()}{" "}
                  {record.package.counter === "estimate"
                    ? "추정 예산 단위"
                    : "공급자 계산 토큰"}
                </p>
                <p className="muted">{record.package.count_notice}</p>
                <p>
                  중복 제외 {record.package.duplicate_chunks}구간 · 예산/검사
                  범위 생략 {record.package.omitted_chunks}구간 (
                  {record.package.omitted_bytes.toLocaleString()}바이트)
                </p>
                <Button
                  variant="primary"
                  disabled={record.stale || busy}
                  onClick={() => setDownloadOpen(true)}
                >
                  전달 조건 확인·내보내기
                </Button>
              </section>
              {record.package.quotes.map((quote, index) => (
                <section className="card" key={`${quote.id}:${index}`}>
                  <h3>
                    {quote.mandatory ? "필수 정책" : "선택 근거"} ·{" "}
                    {quote.title}
                  </h3>
                  <p>포함 이유(사람의 설명): {quote.reason}</p>
                  <p>
                    {quote.attachment_id
                      ? `첨부 추출 구간 · ${positionName(quote.attachment_position || {})} · 조각 기준 ${quote.start_byte}~${quote.end_byte}바이트 · 부모 v${quote.version}`
                      : `문서 원문 인용 · v${quote.version} · ${quote.start_line}~${quote.end_line}행`}
                  </p>
                  <pre className="evidence-text">{quote.text}</pre>
                  <Link
                    to={
                      quote.attachment_id
                        ? `/app/attachments/${quote.attachment_id}?extraction=${quote.extraction_id}&fragment=${quote.fragment_id}`
                        : `/app/documents/${quote.id}`
                    }
                  >
                    현재 원본 위치 확인
                  </Link>
                  <details>
                    <summary>원문 해시</summary>
                    <code className="evidence-hash">{quote.content_hash}</code>
                  </details>
                </section>
              ))}
              <details className="card">
                <summary>에이전트에 전달할 입력 전체 확인</summary>
                <pre className="evidence-text">{record.package.prompt}</pre>
              </details>
              <Button onClick={() => setRemove(true)}>패키지 사본 삭제</Button>
            </>
          )}
        </>
      ) : (
        <>
          <section className="card">
            <h2>목적에 맞는 패키지 구성</h2>
            <p className="notice">
              필수 정책은 임의로 자르지 않습니다. 선택 문서는 앞 8개 구간을
              후보로 검사하고 목적 검색어가 일치한 구간을 우선합니다. AI 요약을
              원문으로 가장하지 않습니다. 다른 대화·도구·출력 토큰은 예산에
              별도로 남겨 두세요.
            </p>
            <Field label="업무 목적">
              <input
                value={purpose}
                maxLength={1000}
                onChange={(e) => setPurpose(e.target.value)}
                placeholder="예: vLLM 운영 변경안 검토"
              />
            </Field>
            <Field label="허용된 작업 범위">
              <textarea
                value={scope}
                maxLength={1000}
                onChange={(e) => setScope(e.target.value)}
                rows={2}
              />
            </Field>
            <div className="evidence-compare">
              <Field label="사용 모델">
                <input
                  value={model}
                  maxLength={256}
                  onChange={(e) => setModel(e.target.value)}
                  placeholder="사내 모델 식별자"
                />
              </Field>
              <Field label="입력 예산 (512~262144)">
                <input
                  type="number"
                  min={512}
                  max={262144}
                  step={1}
                  value={budget}
                  onChange={(e) => setBudget(Number(e.target.value))}
                />
              </Field>
            </div>
            <Field label="토큰 계산">
              <select
                value={counter}
                onChange={(e) => {
                  setCounter(e.target.value);
                  setModelConsent(false);
                }}
              >
                <option value="estimate">
                  로컬 보수적 추정 (원문 전송 없음)
                </option>
                {policy?.token_counter === "responses" && (
                  <option value="responses">
                    설정된 모델 공급자의 input_tokens 계산
                  </option>
                )}
              </select>
            </Field>
            {counter === "responses" && (
              <>
                <p>
                  전송 대상: {provider || "설정되지 않음"} · 모델 {model}
                </p>
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={modelConsent}
                    onChange={(e) => setModelConsent(e.target.checked)}
                  />
                  후보 원문과 업무 설명을 이 모델 서버에 보내 토큰 수를 계산하는
                  데 동의합니다.
                </label>
              </>
            )}
            <Field label="포함할 문서 찾기">
              <input
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="현재 접근 가능한 문서 제목"
              />
            </Field>
            <div className="package-document-list">
              {documents
                .filter((d) =>
                  d.title.toLowerCase().includes(filter.toLowerCase()),
                )
                .slice(0, 100)
                .map((doc) => {
                  const choice = selected.find((d) => d.id === doc.id);
                  return (
                    <div className="package-choice" key={doc.id}>
                      <label>
                        <input
                          type="checkbox"
                          checked={!!choice}
                          disabled={
                            !choice &&
                            selected.length + (attachment ? 1 : 0) >= 32
                          }
                          onChange={(e) =>
                            setSelected((old) =>
                              e.target.checked
                                ? [
                                    ...old,
                                    {
                                      id: doc.id,
                                      version: doc.version,
                                      mandatory: false,
                                      reason: "",
                                    },
                                  ]
                                : old.filter((d) => d.id !== doc.id),
                            )
                          }
                        />
                        {doc.title}
                      </label>
                      {choice && (
                        <>
                          <label>
                            <input
                              type="checkbox"
                              checked={choice.mandatory}
                              onChange={(e) =>
                                setSelected((old) =>
                                  old.map((d) =>
                                    d.id === doc.id
                                      ? { ...d, mandatory: e.target.checked }
                                      : d,
                                  ),
                                )
                              }
                            />
                            필수 정책 (전체 원문 보존)
                          </label>
                          <input
                            aria-label={`${doc.title} 포함 이유`}
                            placeholder="포함 이유"
                            maxLength={250}
                            value={choice.reason}
                            onChange={(e) =>
                              setSelected((old) =>
                                old.map((d) =>
                                  d.id === doc.id
                                    ? { ...d, reason: e.target.value }
                                    : d,
                                ),
                              )
                            }
                          />
                        </>
                      )}
                    </div>
                  );
                })}
            </div>
            {params.get("extraction") &&
              params.get("fragment") &&
              workspace && (
                <AttachmentPackageSelection
                  key={`${user.id}:${workspace.id}`}
                  extraction={params.get("extraction")!}
                  fragment={params.get("fragment")!}
                  workspaceID={workspace.id}
                  value={attachment}
                  onChange={setAttachment}
                />
              )}
            <p>
              {selected.length + (attachment ? 1 : 0)}/32개 선택 · 문서/첨부
              구간 합계. 목록은 현재 불러온 문서에서 최대 100개 표시합니다.
            </p>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={consent}
                onChange={(e) => setConsent(e.target.checked)}
              />
              현재 권한이 적용되는 암호화 사본을 최대{" "}
              {policy?.retention_hours || 24}시간 보관하는 데 동의합니다.
            </label>
            <Button
              variant="primary"
              disabled={
                busy ||
                !consent ||
                (!selected.length && !attachment) ||
                selected.length + (attachment ? 1 : 0) > 32 ||
                !policy?.enabled
              }
              onClick={() => void execute(create)}
            >
              {busy ? "권한·원문·예산 확인 중…" : "지식 패키지 구성"}
            </Button>
          </section>
          <h2>내 패키지</h2>
          {items.length ? (
            <div className="evidence-list">
              {items.map((item) => (
                <Link className="card" key={item.id} to={`?id=${item.id}`}>
                  <strong>{datetime(item.created_at)} 패키지</strong>
                  <span>
                    문서 {item.source_count}개 ·{" "}
                    {item.stale ? "원문 변경됨" : "현재 버전"} · 전달{" "}
                    {item.export_count}회
                  </span>
                  <small>만료 {datetime(item.expires_at)}</small>
                </Link>
              ))}
            </div>
          ) : (
            <Empty
              title="표시할 패키지가 없습니다"
              text="현재 권한이 있고 만료되지 않은 본인의 패키지만 표시합니다."
            />
          )}
        </>
      )}
      <Modal
        open={downloadOpen && !!record}
        onOpenChange={(value) => {
          if (!busy) setDownloadOpen(value);
        }}
        title="에이전트 전달 조건"
      >
        <p>
          현재 원문 버전과 권한을 다시 확인합니다. 이미 내려받은 사본은 이후
          권한 변경으로 회수할 수 없습니다. 조직의 반출·모델 전송 정책을
          따르세요. 이 패키지는 실행 권한을 부여하지 않습니다.
        </p>
        <Field label="전달 대상">
          <input
            value={target}
            maxLength={200}
            onChange={(e) => setTarget(e.target.value)}
            placeholder="예: 사내 운영 검토 에이전트"
          />
        </Field>
        <Button
          variant="primary"
          disabled={busy || !target.trim()}
          onClick={() => void execute(exportPackage)}
        >
          동의하고 JSON 내려받기
        </Button>
      </Modal>
      <Modal
        open={remove && !!record}
        onOpenChange={(value) => {
          if (!busy) setRemove(value);
        }}
        title="패키지 사본 삭제"
      >
        <p>
          서버 사본을 삭제합니다. 이미 내보낸 사본과 백업은 회수되지 않으며 원문
          보존 정책에 따라 삭제가 제한될 수 있습니다.
        </p>
        <Button
          disabled={busy}
          onClick={() =>
            void execute(async () => {
              if (!record) return;
              const request = generation.current;
              await api(`/knowledge/packages/${record.id}`, "DELETE", {
                confirmation: "DELETE",
              });
              if (request === generation.current) setParams({});
            })
          }
        >
          삭제 확인
        </Button>
      </Modal>
    </div>
  );
}

export function KnowledgePackagePolicyPage() {
  const { notify } = useApp();
  const [policy, setPolicy] = useState<PackagePolicy | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    void api<PackagePolicy>("/admin/knowledge-packages/policy")
      .then((value) => {
        if (active) setPolicy(value);
      })
      .catch((e: Error) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, []);
  const save = async () => {
    if (!policy || busy) return;
    setBusy(true);
    setError("");
    try {
      setPolicy(
        await api<PackagePolicy>(
          "/admin/knowledge-packages/policy",
          "PUT",
          policy,
        ),
      );
      notify("지식 패키지 정책을 저장했습니다");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page evidence-page">
      <PageHeading
        title="지식 패키지 정책"
        description="개인별 지식 묶음과 모델 토큰 계산의 허용 범위를 관리합니다."
      />
      <ErrorBox error={error} />
      {policy ? (
        <section className="card">
          <Field label="지식 패키지">
            <select
              value={String(policy.enabled)}
              onChange={(e) =>
                setPolicy({ ...policy, enabled: e.target.value === "true" })
              }
            >
              <option value="true">활성화</option>
              <option value="false">비활성화</option>
            </select>
          </Field>
          <Field label="최대 보존 시간 (1~168시간)">
            <input
              type="number"
              min={1}
              max={168}
              value={policy.retention_hours}
              onChange={(e) =>
                setPolicy({
                  ...policy,
                  retention_hours: Number(e.target.value),
                })
              }
            />
          </Field>
          <Field label="허용할 토큰 계산">
            <select
              value={policy.token_counter}
              onChange={(e) =>
                setPolicy({
                  ...policy,
                  token_counter: e.target
                    .value as PackagePolicy["token_counter"],
                })
              }
            >
              <option value="estimate">로컬 추정만</option>
              <option value="responses">로컬 추정 + 모델 input_tokens</option>
            </select>
          </Field>
          <Field label="모델 계산의 내부 HTTP 허용">
            <select
              value={String(policy.allow_http)}
              onChange={(e) =>
                setPolicy({ ...policy, allow_http: e.target.value === "true" })
              }
            >
              <option value="false">HTTPS만 허용</option>
              <option value="true">HTTP도 허용 (암호화되지 않음)</option>
            </select>
          </Field>
          <p className="notice">
            모델 계산은 현재 워크스페이스 AI 주소·키·모델을 사용하며 원문 전송
            동의를 따로 받습니다. 공급자가 Responses input_tokens를 지원하지
            않으면 오류로 안내하며 다른 서버나 추정치로 자동 전환하지 않습니다.
            패키지 조회·MCP에도 현재 문서 권한을 적용합니다.
          </p>
          <Button disabled={busy} variant="primary" onClick={() => void save()}>
            패키지 정책 저장
          </Button>
        </section>
      ) : (
        !error && <Loading />
      )}
      {policy && (
        <KnowledgePolicyHistory
          kind="packages"
          version={policy.version}
          onRestored={async () =>
            setPolicy(
              await api<PackagePolicy>("/admin/knowledge-packages/policy"),
            )
          }
        />
      )}
    </div>
  );
}
