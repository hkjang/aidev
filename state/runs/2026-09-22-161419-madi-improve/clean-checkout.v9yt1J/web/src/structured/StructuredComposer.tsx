import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Sparkles, Square } from "lucide-react";
import { api, datetime } from "../api";
import { Button, Field, Loading } from "../ui";
import { RecoveryNotice } from "../review/ChangeReview";
import { streamStructuredProposal } from "./stream";
import {
  propertyLabels,
  type StructuredContext,
  type StructuredProposal,
} from "./types";

/** Textarea normalizes CRLF. Map its UTF-16 selection back to exact source. */
export function textareaSourceRange(
  markdown: string,
  start: number,
  end: number,
) {
  if (!Number.isInteger(start) || !Number.isInteger(end) || start < 0)
    return null;
  let normalized = 0,
    index = 0,
    from = -1,
    to = -1;
  while (index <= markdown.length) {
    if (normalized === start && from < 0) from = index;
    if (normalized === end) {
      to = index;
      break;
    }
    if (index === markdown.length) break;
    if (markdown[index] === "\r" && markdown[index + 1] === "\n") index += 2;
    else index++;
    normalized++;
  }
  if (from < 0 || to <= from) return null;
  const splitsSurrogate = (offset: number) =>
    offset > 0 &&
    /[\uD800-\uDBFF]/.test(markdown[offset - 1]) &&
    /[\uDC00-\uDFFF]/.test(markdown[offset] || "");
  if (splitsSurrogate(from) || splitsSurrogate(to)) return null;
  const encoder = new TextEncoder();
  return {
    text: markdown.slice(from, to),
    start_byte: encoder.encode(markdown.slice(0, from)).length,
    end_byte: encoder.encode(markdown.slice(0, to)).length,
  };
}
export default function StructuredComposer({
  documentId,
  databaseId,
  workspaceId,
  onSaved,
}: {
  documentId: string;
  databaseId: string;
  workspaceId: string;
  onSaved: (id: string) => void;
}) {
  const [context, setContext] = useState<StructuredContext | null>(null),
    [error, setError] = useState<unknown>(null),
    [loading, setLoading] = useState(true),
    [retry, setRetry] = useState(0);
  const [selected, setSelected] =
      useState<ReturnType<typeof textareaSourceRange>>(null),
    [properties, setProperties] = useState<string[]>([]),
    [consent, setConsent] = useState(false),
    [storageConsent, setStorageConsent] = useState(false);
  const [busy, setBusy] = useState(false),
    [saving, setSaving] = useState(false),
    [proposal, setProposal] = useState<StructuredProposal | null>(null);
  const [startLine, setStartLine] = useState("1"),
    [endLine, setEndLine] = useState("1");
  const source = useRef<HTMLTextAreaElement>(null),
    stream = useRef<AbortController | null>(null),
    mounted = useRef(false),
    baseline = useRef<StructuredContext | null>(null);
  const discardOutput = () => {
    stream.current?.abort();
    setBusy(false);
    setProposal(null);
    setStorageConsent(false);
    setConsent(false);
  };
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      stream.current?.abort();
    };
  }, []);
  useEffect(() => {
    const abort = new AbortController();
    let pending = false,
      blocked = false;
    baseline.current = null;
    setContext(null);
    setSelected(null);
    setProperties([]);
    setLoading(true);
    setError(null);
    discardOutput();
    const load = async () => {
      if (pending || blocked || abort.signal.aborted) return;
      pending = true;
      try {
        const next = await api<StructuredContext>(
          `/documents/${documentId}/structured-context?database_id=${databaseId}`,
          "GET",
          undefined,
          { signal: abort.signal },
        );
        if (abort.signal.aborted) return;
        if (
          next.workspace_id !== workspaceId ||
          next.document_id !== documentId ||
          next.database_id !== databaseId
        )
          throw Error("현재 워크스페이스의 원문과 대상을 다시 선택하세요.");
        const old = baseline.current;
        if (
          old &&
          (old.version !== next.version ||
            old.schema_hash !== next.schema_hash ||
            old.destination_hash !== next.destination_hash ||
            old.provider.fingerprint !== next.provider.fingerprint ||
            old.provider.configured !== next.provider.configured)
        )
          throw Error(
            "원문·속성·공유 대상·AI 공급자가 바뀌어 선택과 제안·동의를 지웠습니다. 현재 상태를 다시 읽으세요.",
          );
        if (!old) {
          baseline.current = next;
          setContext(next);
          setError(null);
        }
      } catch (e) {
        if (!abort.signal.aborted) {
          blocked = true;
          baseline.current = null;
          setContext(null);
          setSelected(null);
          setProperties([]);
          discardOutput();
          setError(e);
        }
      } finally {
        pending = false;
        if (!abort.signal.aborted) setLoading(false);
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      abort.abort();
      clearInterval(timer);
      stream.current?.abort();
    };
  }, [documentId, databaseId, workspaceId, retry]);
  const acceptSelection = (
    selection: ReturnType<typeof textareaSourceRange>,
  ) => {
    discardOutput();
    if (!selection || !selection.text.trim()) {
      setSelected(null);
      setError(Error("원문에서 사용할 연속 구간을 먼저 선택하세요."));
      return;
    }
    if (selection.end_byte - selection.start_byte > 32768) {
      setSelected(null);
      setError(Error("선택 구간은 UTF-8 기준 32KiB 이하여야 합니다."));
      return;
    }
    setSelected(selection);
    setError(null);
  };
  const useSelection = () => {
    if (!source.current || !context) return;
    acceptSelection(
      textareaSourceRange(
        context.markdown,
        source.current.selectionStart,
        source.current.selectionEnd,
      ),
    );
  };
  const useLines = () => {
    if (!context) return;
    const lines = context.markdown.replace(/\r\n?/g, "\n").split("\n"),
      first = Number(startLine),
      last = Number(endLine);
    if (
      !Number.isInteger(first) ||
      !Number.isInteger(last) ||
      first < 1 ||
      last < first ||
      last > lines.length
    ) {
      discardOutput();
      setSelected(null);
      setError(
        Error(
          `시작 줄과 끝 줄을 1~${lines.length} 사이의 올바른 순서로 입력하세요.`,
        ),
      );
      return;
    }
    const start = lines
        .slice(0, first - 1)
        .reduce((sum, line) => sum + line.length + 1, 0),
      end = start + lines.slice(first - 1, last).join("\n").length;
    acceptSelection(textareaSourceRange(context.markdown, start, end));
  };
  const generate = async () => {
    if (!context || !selected || busy || saving || !consent) return;
    const snapshot = context;
    const abort = new AbortController();
    stream.current?.abort();
    stream.current = abort;
    setBusy(true);
    setProposal(null);
    setStorageConsent(false);
    setError(null);
    const fresh = () =>
      mounted.current && !abort.signal.aborted && baseline.current === snapshot;
    try {
      const result = await streamStructuredProposal(
        `/documents/${documentId}/structured-draft`,
        {
          database_id: databaseId,
          expected_version: snapshot.version,
          start_byte: selected.start_byte,
          end_byte: selected.end_byte,
          selected_text: selected.text,
          property_ids: properties,
          provider_fingerprint: snapshot.provider.fingerprint,
          schema_hash: snapshot.schema_hash,
          destination_hash: snapshot.destination_hash,
          consent,
        },
        abort.signal,
      );
      if (result.source_version !== snapshot.version)
        throw Error("원문 버전과 제안이 일치하지 않습니다.");
      if (fresh()) setProposal(result);
    } catch (e) {
      if (fresh()) {
        setProposal(null);
        setError(e);
      }
    } finally {
      if (mounted.current && stream.current === abort) setBusy(false);
    }
  };
  const save = async () => {
    if (!proposal || !storageConsent || saving || !context) return;
    const snapshot = baseline.current;
    setSaving(true);
    setError(null);
    try {
      const result = await api<{ id: string }>(
        "/knowledge/structured-drafts",
        "POST",
        { ticket: proposal.draft_ticket, consent: storageConsent },
      );
      if (mounted.current && snapshot === baseline.current) onSaved(result.id);
    } catch (e) {
      if (mounted.current && snapshot === baseline.current) {
        setProposal(null);
        setStorageConsent(false);
        setError(e);
      }
    } finally {
      if (mounted.current) setSaving(false);
    }
  };
  if (loading) return <Loading />;
  return (
    <div className="structured-composer">
      <RecoveryNotice error={error} onRetry={() => setRetry((v) => v + 1)} />
      {context && (
        <>
          <section className="panel structured-section">
            <h2>1. 원문과 전송 범위 선택</h2>
            <p>
              <strong>{context.title}</strong> · 저장 버전 {context.version} →{" "}
              <strong>{context.database_name}</strong>
            </p>
            <p className="muted">
              원문에서 필요한 연속 구간을 드래그하거나 키보드로 선택한 뒤 ‘선택
              구간 사용’을 누르세요. 문서 전체를 자동 전송하지 않습니다.
            </p>
            <Field label="구조화할 저장 원문">
              <textarea
                className="structured-source"
                ref={source}
                readOnly
                value={context.markdown}
                rows={12}
                spellCheck={false}
              />
            </Field>
            <Button
              onMouseDown={(e) => e.preventDefault()}
              onClick={useSelection}
              disabled={busy || saving}
            >
              선택 구간 사용
            </Button>
            <details className="structured-line-selection">
              <summary>키보드로 줄 범위 지정</summary>
              <p className="muted">
                첫 줄은 1입니다. 지정한 줄의 원문만 선택하며, 아래 전송
                미리보기에서 내용을 확인할 수 있습니다.
              </p>
              <fieldset
                disabled={busy || saving}
                className="structured-pickers"
              >
                <Field label="구조화 시작 줄">
                  <input
                    type="number"
                    min="1"
                    value={startLine}
                    onChange={(event) => setStartLine(event.target.value)}
                  />
                </Field>
                <Field label="구조화 끝 줄">
                  <input
                    type="number"
                    min="1"
                    value={endLine}
                    onChange={(event) => setEndLine(event.target.value)}
                  />
                </Field>
              </fieldset>
              <Button onClick={useLines} disabled={busy || saving}>
                줄 범위를 선택 구간으로 사용
              </Button>
            </details>
            {selected && (
              <section className="structured-selection">
                <strong>
                  전송할 구간 · {selected.end_byte - selected.start_byte}바이트
                  / 32KiB
                </strong>
                <p className="muted">
                  UTF-8 {selected.start_byte}–{selected.end_byte}바이트
                </p>
                <pre>{selected.text}</pre>
              </section>
            )}
            <fieldset
              disabled={busy || saving}
              className="structured-properties"
            >
              <legend>추출할 일반 속성 · {properties.length} / 32개</legend>
              {context.properties.map((p) => (
                <label key={p.id}>
                  <input
                    type="checkbox"
                    checked={properties.includes(p.id)}
                    disabled={
                      !properties.includes(p.id) && properties.length >= 32
                    }
                    onChange={(e) => {
                      discardOutput();
                      setProperties((values) =>
                        e.target.checked
                          ? [...values, p.id]
                          : values.filter((v) => v !== p.id),
                      );
                    }}
                  />
                  <span>
                    {p.name}
                    <small>
                      {propertyLabels[p.type] || p.type}
                      {p.options?.length ? ` · ${p.options.join(", ")}` : ""}
                    </small>
                  </span>
                </label>
              ))}
            </fieldset>
            {!context.properties.length && (
              <p className="notice">
                계산·관계·사용자 속성은 구조화하지 않습니다. 대상 데이터베이스에
                일반 입력 속성을 먼저 만드세요.
              </p>
            )}
          </section>
          <section className="panel structured-section">
            <h2>2. AI 제안 생성</h2>
            <p>
              선택 구간과 선택한 속성의 이름·타입·옵션만 전송합니다. 문서
              제목·나머지 원문·DB 이름·다른 행·선택하지 않은 속성은 보내지
              않습니다.
            </p>
            <dl className="structured-provider">
              <dt>AI 공급자</dt>
              <dd>{context.provider.base_url || "미설정"}</dd>
              <dt>모델</dt>
              <dd>{context.provider.model || "미설정"}</dd>
            </dl>
            {context.provider.base_url.startsWith("http://") && (
              <p className="notice warning">
                HTTP 공급자는 전송을 암호화하지 않습니다. 신뢰하는 내부망인지
                확인하세요.
              </p>
            )}
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={consent}
                disabled={busy || saving || !selected || !properties.length}
                onChange={(e) => setConsent(e.target.checked)}
              />
              위 구간과 선택 속성을 표시된 AI 공급자에게 전송하는 데 동의합니다.
            </label>
            <div className="button-row">
              <Button
                variant="primary"
                disabled={
                  busy ||
                  saving ||
                  !selected ||
                  !properties.length ||
                  !consent ||
                  !context.provider.configured
                }
                onClick={generate}
              >
                <Sparkles size={18} />
                {busy ? "제안을 생성하고 확인하는 중…" : "구조화 제안 생성"}
              </Button>
              {busy && (
                <Button
                  onClick={() => {
                    discardOutput();
                    setError(
                      Error(
                        "생성을 중단했습니다. 중간 결과는 보관·반영하지 않습니다.",
                      ),
                    );
                  }}
                >
                  <Square size={17} />
                  생성 중단
                </Button>
              )}
            </div>
            {!context.provider.configured && (
              <p className="notice">
                관리자가 AI 공급자와 모델을 활성화해야 합니다. 일반 문서와
                데이터베이스는 그대로 사용할 수 있습니다.
              </p>
            )}
            {busy && !proposal && (
              <p role="status" aria-live="polite" className="structured-notice">
                AI 제안을 생성하고 있습니다. 결과의 형식과 민감정보를 확인한 뒤
                표시합니다. 완료 전에는 보관하거나 반영할 수 없습니다.
              </p>
            )}
          </section>
          {proposal && (
            <section className="panel structured-section">
              <h2>3. 완료된 제안을 개인 검토함에 보관</h2>
              <p>
                AI 제안은 사실성 확인이 아닙니다. 타입·선택 옵션·인용이 맞지
                않는 항목도 오류와 함께 보존하며, 다음 단계에서 수정하거나
                제외합니다.
              </p>
              <div className="structured-proposal-summary">
                {proposal.fields.map((field, index) => (
                  <article key={`${field.property_id}:${index}`}>
                    <strong>
                      {context.properties.find(
                        (p) => p.id === field.property_id,
                      )?.name || field.property_id}
                    </strong>
                    <pre>
                      {typeof field.value === "string"
                        ? field.value
                        : JSON.stringify(field.value)}
                    </pre>
                    <blockquote>{field.quote || "인용 없음"}</blockquote>
                    <p className={field.valid ? "muted" : "notice warning"}>
                      {field.valid
                        ? "타입·구간 검사 통과 · 의미는 직접 확인"
                        : field.issue || "직접 확인이 필요한 제안"}
                    </p>
                  </article>
                ))}
              </div>
              <p className="muted">
                보관 요청 유효 기한 {datetime(proposal.expires_at)} · 보관한
                개인 초안은 24시간 동안 검토할 수 있습니다.
              </p>
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={storageConsent}
                  disabled={saving}
                  onChange={(e) => setStorageConsent(e.target.checked)}
                />
                값과 근거 구간을 내 검토함에 암호화 보관하는 데 동의합니다. 아직
                데이터베이스에 공유하지 않습니다.
              </label>
              <Button
                variant="primary"
                disabled={saving || !storageConsent}
                onClick={save}
              >
                {saving ? "개인 초안 보관 중…" : "개인 검토함에 보관"}
              </Button>
            </section>
          )}
          <Link
            className="button"
            to={`/app/documents/${documentId}?mode=preview`}
          >
            원문으로 돌아가기
          </Link>
        </>
      )}
    </div>
  );
}
