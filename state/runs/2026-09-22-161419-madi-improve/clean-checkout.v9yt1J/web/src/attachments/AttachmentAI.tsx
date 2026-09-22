import { useEffect, useRef, useState } from "react";
import { sha256 } from "@noble/hashes/sha2.js";
import { bytesToHex } from "@noble/hashes/utils.js";
import { api, ApiError } from "../api";
import { useApp } from "../context";
import { Button, Field, Modal } from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import CitationViewer, { type CitationSource } from "../CitationViewer";
import SaveEvidence from "../evidence/SaveEvidence";
import { MarkdownContent } from "../editor/MarkdownContent";
import { type Context, type Fragment, type Run, positionName } from "./types";
import { attachmentSelectionRange } from "./selection";

type Provider = {
  configured: boolean;
  base_url: string;
  model: string;
  fingerprint: string;
  max_tokens: number;
};
type Answer = {
  answer: string;
  sources: CitationSource[];
  history_ticket: string;
};
function firstRange(text: string) {
  let bytes = 0,
    end = 0;
  for (const char of text) {
    const n = new TextEncoder().encode(char).length;
    if (bytes + n > 8192) break;
    bytes += n;
    end += char.length;
  }
  return [0, end] as const;
}
export default function AttachmentAI({
  context,
  fragment,
  run,
}: {
  context: Context;
  fragment: Fragment;
  run: Run;
}) {
  const { user, workspace } = useApp();
  const scope = `${user.id}:${workspace?.id}:${context.id}:${context.document_version}:${run.id}:${run.revision}:${fragment.id}`;
  const current = useRef(scope);
  current.current = scope;
  const controller = useRef<AbortController | null>(null);
  const sourceInput = useRef<HTMLTextAreaElement | null>(null);
  const [open, setOpen] = useState(false),
    [provider, setProvider] = useState<Provider | null>(null),
    [range, setRange] = useState<readonly [number, number]>(() =>
      firstRange(fragment.text),
    ),
    [prompt, setPrompt] = useState(""),
    [text, setText] = useState(""),
    [answer, setAnswer] = useState<Answer | null>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [citation, setCitation] = useState<CitationSource | null>(null);
  useEffect(() => {
    setOpen(false);
    setAnswer(null);
    setText("");
    setPrompt("");
    setError(null);
    setBusy(false);
    setRange(firstRange(fragment.text));
    setProvider(null);
    return () => controller.current?.abort();
  }, [scope]);
  useEffect(() => {
    if (!open) return;
    const abort = new AbortController();
    api<{ provider: Provider }>(
      `/documents/${context.document_id}/ai-selection`,
      "GET",
      undefined,
      { signal: abort.signal },
    )
      .then((v) => {
        if (!abort.signal.aborted) setProvider(v.provider);
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e);
      });
    return () => abort.abort();
  }, [open, scope]);
  const selected = fragment.text.slice(range[0], range[1]);
  const selectedBytes = new TextEncoder().encode(selected);
  const characters = Array.from(fragment.text);
  const firstCharacter = Array.from(fragment.text.slice(0, range[0])).length + 1;
  const lastCharacter = Array.from(fragment.text.slice(0, range[1])).length;
  const selectCharacters = (first: number, last: number) => {
    if (!Number.isSafeInteger(first) || !Number.isSafeInteger(last) || first < 1 || last < first || last > characters.length) return;
    setRange([characters.slice(0, first - 1).join("").length, characters.slice(0, last).join("").length]);
  };
  const start = async () => {
    if (
      !provider?.configured ||
      busy ||
      selectedBytes.length === 0 ||
      selectedBytes.length > 8192
    )
      return;
    const abort = new AbortController();
    controller.current = abort;
    const alive = () => !abort.signal.aborted && current.current === scope;
    setBusy(true);
    setText("");
    setAnswer(null);
    setError(null);
    let result: Answer | null = null,
      output = "",
      done = false;
    try {
      const response = await fetch(
        `/api/v1/attachment-extractions/${run.id}/ai`,
        {
          method: "POST",
          credentials: "same-origin",
          signal: abort.signal,
          headers: {
            "Content-Type": "application/json",
            "X-Madi-Request": "1",
          },
          body: JSON.stringify({
            fragment_id: fragment.id,
            document_version: context.document_version,
            revision: run.revision,
            start_byte: new TextEncoder().encode(
              fragment.text.slice(0, range[0]),
            ).length,
            end_byte: new TextEncoder().encode(fragment.text.slice(0, range[1]))
              .length,
            hash: bytesToHex(sha256(selectedBytes)),
            provider_fingerprint: provider.fingerprint,
            consent: true,
            prompt,
          }),
        },
      );
      if (!response.ok) {
        const v = await response.json().catch(() => ({}));
        throw new ApiError(
          v.error || "첨부 AI 요청에 실패했습니다",
          response.status,
        );
      }
      if (
        !response.body ||
        !response.headers.get("Content-Type")?.includes("text/event-stream")
      )
        throw new Error("AI 스트리밍 응답이 필요합니다");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const chunk = await reader.read();
        if (chunk.done) break;
        buffer += decoder.decode(chunk.value, { stream: true });
        if (buffer.length > 300000)
          throw new Error("AI 이벤트가 허용 크기를 초과했습니다");
        let end: number;
        while ((end = buffer.indexOf("\n\n")) >= 0) {
          const raw = buffer
            .slice(0, end)
            .split("\n")
            .filter((l) => l.startsWith("data:"))
            .map((l) => l.slice(5).trimStart())
            .join("\n");
          buffer = buffer.slice(end + 2);
          if (!raw) continue;
          if (raw === "[DONE]") {
            done = true;
            continue;
          }
          const value = JSON.parse(raw);
          if (value.error || value.retract)
            throw new Error(
              value.error || "권한·원본 변경으로 답변을 회수했습니다",
            );
          if (typeof value.text === "string") {
            output += value.text;
            if (new TextEncoder().encode(output).length > 65536)
              throw new Error("AI 답변은 64KiB 이하여야 합니다");
            if (alive()) setText(output);
          }
          if (value.proposal) {
            if (
              value.proposal.answer !== output ||
              !Array.isArray(value.proposal.sources) ||
              value.proposal.sources.length !== 1 ||
              value.proposal.sources[0].attachment_id !== context.id
            )
              throw new Error("AI 결과의 첨부 근거를 확인하지 못했습니다");
            result = value.proposal;
          }
        }
      }
      if (!done || !result)
        throw new Error("AI 응답이 완료되지 않아 임시 내용을 지웠습니다");
      if (alive()) {
        setAnswer(result);
        setOpen(false);
      }
    } catch (e) {
      if (current.current === scope) {
        setText("");
        setAnswer(null);
        if (!abort.signal.aborted) setError(e);
      }
    } finally {
      abort.abort();
      if (current.current === scope) setBusy(false);
    }
  };
  return (
    <section className="extract-ai">
      <Button
        onClick={() => {
          setError(null);
          setOpen(true);
        }}
      >
        이 근거로 AI 질문
      </Button>
      <RecoveryNotice
        error={error}
        onReview={() => {
          setProvider(null);
          setOpen(false);
        }}
      />
      {text && (
        <div className="card">
          <h3>{busy ? "답변 작성 중…" : "첨부 근거 기반 답변"}</h3>
          <MarkdownContent markdown={text} />
          {answer && (
            <>
              <Button onClick={() => setCitation(answer.sources[0])}>
                [1] 원본 위치·해시 확인
              </Button>
              <SaveEvidence
                ticket={answer.history_ticket}
                onNavigate={() => setCitation(null)}
              />
            </>
          )}
        </div>
      )}
      <CitationViewer
        source={citation}
        onClose={() => setCitation(null)}
        onNavigate={() => setCitation(null)}
      />
      <Modal
        open={open}
        onOpenChange={(v) => {
          if (!v) {
            controller.current?.abort();
            setBusy(false);
            setText("");
            setAnswer(null);
          }
          setOpen(v);
        }}
        title="선택한 첨부 본문 전송 확인"
        wide
      >
        <ChangeReview
          title="이 요청에만 첨부 본문 전송 동의"
          changes={[
            {
              label: "근거 범위",
              before: positionName(fragment.position),
              after: `${selectedBytes.length.toLocaleString()}바이트 · 최대 8KiB`,
            },
            {
              label: "AI 공급자",
              before: "서버 로컬 추출",
              after: provider
                ? `${provider.base_url} · ${provider.model} · 출력 최대 ${provider.max_tokens.toLocaleString()}토큰`
                : "확인 중",
            },
          ]}
          warnings={[
            "문서 색인 동의와 별개인 단일 요청입니다. 아래 선택 텍스트와 질문만 전송하며 원본 파일·다른 본문·부모 문서·파일명은 보내지 않습니다.",
            "OCR·표 변환은 틀릴 수 있습니다. 답변이 아니라 원본의 쪽·영역·셀을 최종 근거로 확인하세요.",
          ]}
          disabled={
            !provider?.configured ||
            !selectedBytes.length ||
            selectedBytes.length > 8192
          }
          busy={busy}
          confirmLabel="선택 본문 전송·답변 요청"
          onConfirm={() => void start()}
          onCancel={() => {
            controller.current?.abort();
            setOpen(false);
          }}
        >
          <RecoveryNotice error={error} />
          <fieldset disabled={busy}>
            <Field
              label="추출 원문 · 드래그로 부분 선택"
              hint="기본 선택은 최대 8KiB 앞부분입니다. 드래그한 뒤 ‘선택 구간 사용’을 누르거나 아래 글자 범위를 지정하세요."
            >
              <textarea
                ref={sourceInput}
                readOnly
                rows={8}
                value={fragment.text}
              />
            </Field>
            <Button type="button" onMouseDown={(e) => e.preventDefault()} onClick={() => {
              const el = sourceInput.current;
              const next = el && attachmentSelectionRange(fragment.text, el.selectionStart, el.selectionEnd);
              if (next) { setRange(next); setError(null); }
              else setError(new Error("추출 원문에서 구간을 선택하거나 시작·끝 글자를 지정하세요."));
            }}>선택 구간 사용</Button>
            <div className="evidence-compare">
              <Field label="전송 시작 글자 (포함)"><input type="number" min={1} max={lastCharacter} value={firstCharacter} onChange={(e) => selectCharacters(Number(e.target.value), lastCharacter)} /></Field>
              <Field label="전송 끝 글자 (포함)"><input type="number" min={firstCharacter} max={characters.length} value={lastCharacter} onChange={(e) => selectCharacters(firstCharacter, Number(e.target.value))} /></Field>
            </div>
            <details open>
              <summary>실제 전송할 텍스트</summary>
              <pre className="extract-text">{selected}</pre>
            </details>
            <Field
              label="이 근거에 질문"
              hint="비워 두면 핵심 내용을 요약합니다."
            >
              <textarea
                maxLength={2048}
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                rows={3}
              />
            </Field>
          </fieldset>
          {busy && (
            <Button
              onClick={() => {
                controller.current?.abort();
                setText("");
                setAnswer(null);
                setBusy(false);
              }}
            >
              AI 요청 중지
            </Button>
          )}
        </ChangeReview>
      </Modal>
    </section>
  );
}
