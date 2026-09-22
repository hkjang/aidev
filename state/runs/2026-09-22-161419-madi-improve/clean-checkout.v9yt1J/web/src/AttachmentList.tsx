import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Download, Paperclip } from "lucide-react";
import { api, bytes } from "./api";
import { Button, ErrorBox } from "./ui";
import { useApp } from "./context";
import "./attachments.css";
export default function AttachmentList({
  documentID,
  revision,
}: {
  documentID: string;
  revision: number;
}) {
  const { user } = useApp();
  const scope = `${user.id}:${documentID}:${revision}`;
  const current = useRef(scope);
  current.current = scope;
  const [files, setFiles] = useState<Record<string, any>[]>([]),
    [error, setError] = useState<unknown>(null),
    [more, setMore] = useState(false),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    setFiles([]);
    setError(null);
    setBusy(false);
    api<Record<string, any>[]>(`/documents/${documentID}/attachments`)
      .then((rows) => {
        if (active) {
          setFiles(rows);
          setMore(rows.length === 200);
        }
      })
      .catch((e) => {
        if (active) setError(e);
      });
    return () => {
      active = false;
    };
  }, [documentID, revision, user.id]);
  if (!files.length && !error) return null;
  return (
    <section className="attachment-index">
      <h3>
        <Paperclip size={17} />
        문서 첨부파일 · {files.length}
        {more ? "+" : ""}개
      </h3>
      <ErrorBox error={error} />
      <ul>
        {files.map((f) => (
          <li key={f.id}>
            <a href={f.url} download={f.name}>
              <Download size={16} />
              <span>{f.name}</span>
              <small>{bytes(f.size)}</small>
            </a>
            <Link className="button small" to={`/app/attachments/${f.id}`}>
              본문·위치 보기
            </Link>
          </li>
        ))}
      </ul>
      {more && (
        <Button
          disabled={busy}
          onClick={async () => {
            const captured = scope;
            setBusy(true);
            try {
              const rows = await api<Record<string, any>[]>(
                `/documents/${documentID}/attachments?after=${files.at(-1)?.id}`,
              );
              if (current.current !== captured) return;
              setFiles((prev) => [...prev, ...rows]);
              setMore(rows.length === 200);
            } catch (e) {
              if (current.current === captured) setError(e);
            } finally {
              if (current.current === captured) setBusy(false);
            }
          }}
        >
          이전 첨부파일 더 보기
        </Button>
      )}
    </section>
  );
}
