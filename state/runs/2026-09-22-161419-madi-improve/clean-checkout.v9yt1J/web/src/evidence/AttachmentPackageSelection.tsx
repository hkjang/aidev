import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { sha256 } from "@noble/hashes/sha2.js";
import { bytesToHex } from "@noble/hashes/utils.js";
import { api } from "../api";
import { ErrorBox, Field, Loading } from "../ui";
import {
  type Fragment,
  type Position,
  positionName,
} from "../attachments/types";

export type PackageAttachmentSource = {
  id: string;
  version: number;
  attachment_id: string;
  attachment_checksum: string;
  extraction_id: string;
  extraction_revision: number;
  fragment_id: string;
  fragment_hash: string;
  attachment_position: Position;
  start_byte: number;
  end_byte: number;
  content_hash: string;
};
export type AttachmentPackageChoice = {
  source: PackageAttachmentSource;
  mandatory: boolean;
  reason: string;
};
type Result = {
  extraction: {
    id: string;
    attachment_id: string;
    document_id: string;
    workspace_id: string;
    revision: number;
    checksum: string;
  };
  document_version: number;
  fragment: Fragment;
};

export default function AttachmentPackageSelection({
  extraction,
  fragment,
  workspaceID,
  value,
  onChange,
}: {
  extraction: string;
  fragment: string;
  workspaceID: string;
  value: AttachmentPackageChoice | null;
  onChange: (value: AttachmentPackageChoice | null) => void;
}) {
  const [data, setData] = useState<Result | null>(null),
    [error, setError] = useState("");
  const [start, setStart] = useState(1),
    [end, setEnd] = useState(0);
  const callback = useRef(onChange);
  callback.current = onChange;
  useEffect(() => {
    let active = true,
      running = false,
      signature = "";
    const controller = new AbortController();
    setData(null);
    setError("");
    callback.current(null);
    const load = async () => {
      if (running) return;
      running = true;
      try {
        const next = await api<Result>(
          `/attachment-extractions/${extraction}/fragments/${fragment}`,
          "GET",
          undefined,
          { signal: controller.signal },
        );
        if (!active) return;
        if (next.extraction.workspace_id !== workspaceID)
          throw new Error("첨부와 같은 워크스페이스를 선택하세요.");
        const current = JSON.stringify(next);
        if (signature !== current) {
          callback.current(null);
          setStart(1);
          setEnd(Array.from(next.fragment.text).length);
        }
        signature = current;
        setData(next);
        setError("");
      } catch (e) {
        if (active) {
          setData(null);
          callback.current(null);
          setError((e as Error).message);
        }
      } finally {
        running = false;
      }
    };
    void load();
    const timer = window.setInterval(() => void load(), 2000);
    return () => {
      active = false;
      controller.abort();
      clearInterval(timer);
      callback.current(null);
    };
  }, [extraction, fragment, workspaceID]);
  const span = useMemo(() => {
    if (!data) return null;
    const chars = Array.from(data.fragment.text);
    if (
      !Number.isSafeInteger(start) ||
      !Number.isSafeInteger(end) ||
      start < 1 ||
      end < start ||
      end > chars.length
    )
      return null;
    const encoder = new TextEncoder(),
      text = chars.slice(start - 1, end).join("");
    const bytes = encoder.encode(text),
      before = encoder.encode(chars.slice(0, start - 1).join(""));
    if (!bytes.length || bytes.length > 8192) return null;
    const source: PackageAttachmentSource = {
      id: data.extraction.document_id,
      version: data.document_version,
      attachment_id: data.extraction.attachment_id,
      attachment_checksum: data.extraction.checksum,
      extraction_id: data.extraction.id,
      extraction_revision: data.extraction.revision,
      fragment_id: data.fragment.id,
      fragment_hash: data.fragment.content_hash,
      attachment_position: data.fragment.position,
      start_byte: before.length,
      end_byte: before.length + bytes.length,
      content_hash: bytesToHex(sha256(bytes)),
    };
    return { source, text, bytes: bytes.length };
  }, [data, start, end]);
  return (
    <section className="card">
      <h3>첨부 구간을 확인하고 포함</h3>
      <p>
        추출한 텍스트는 원본 파일과 대조하세요. 자동 포함하거나 AI 서버에
        전송하지 않습니다. 필수 지정도 파일 전체가 아닌 아래에서 선택한 구간에만
        적용됩니다.
      </p>
      <ErrorBox error={error} />
      {!data && !error && <Loading />}
      {data && (
        <>
          <p>
            {positionName(data.fragment.position)} · 부모 문서 v
            {data.document_version} · 추출 v{data.extraction.revision}
          </p>
          <Link
            to={`/app/attachments/${data.extraction.attachment_id}?extraction=${extraction}&fragment=${fragment}`}
          >
            원본 위치와 대조
          </Link>
          <div className="evidence-compare">
            <Field label="시작 글자 (포함)">
              <input
                type="number"
                min={1}
                max={Array.from(data.fragment.text).length}
                value={start}
                onChange={(e) => {
                  callback.current(null);
                  setStart(Number(e.target.value));
                }}
              />
            </Field>
            <Field label="끝 글자 (포함)">
              <input
                type="number"
                min={start}
                max={Array.from(data.fragment.text).length}
                value={end}
                onChange={(e) => {
                  callback.current(null);
                  setEnd(Number(e.target.value));
                }}
              />
            </Field>
          </div>
          {span ? (
            <>
              <pre className="evidence-text">{span.text}</pre>
              <p>
                선택 {span.bytes.toLocaleString()} / 8,192바이트 · 이 추출 조각
                기준 {span.source.start_byte}~{span.source.end_byte}바이트
              </p>
            </>
          ) : (
            <p role="alert">
              올바른 글자 범위로 8KiB 이하를 선택하세요. 더 큰 원문을 조용히
              자르지 않습니다.
            </p>
          )}
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={!!value}
              disabled={!span}
              onChange={(e) =>
                callback.current(
                  e.target.checked && span
                    ? { source: span.source, mandatory: false, reason: "" }
                    : null,
                )
              }
            />
            이 첨부 구간을 패키지에 포함합니다
          </label>
          {value && (
            <>
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={value.mandatory}
                  onChange={(e) =>
                    callback.current({ ...value, mandatory: e.target.checked })
                  }
                />
                선택 구간 전체를 필수로 보존
              </label>
              <Field label="첨부 구간 포함 이유">
                <input
                  maxLength={250}
                  value={value.reason}
                  onChange={(e) =>
                    callback.current({ ...value, reason: e.target.value })
                  }
                />
              </Field>
            </>
          )}
        </>
      )}
    </section>
  );
}
