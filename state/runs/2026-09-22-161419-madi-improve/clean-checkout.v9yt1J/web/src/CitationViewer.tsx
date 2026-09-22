import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "./api";
import { Badge, ErrorBox, Loading, Modal } from "./ui";
import { positionName, type Position } from "./attachments/types";
export type CitationSource = {
  id: string;
  title: string;
  version?: number;
  citation_id?: string;
  start_line?: number;
  end_line?: number;
  url?: string;
  citation_url?: string;
  attachment_id?: string;
  attachment_position?: Position;
};
export default function CitationViewer({
  source,
  onClose,
  onNavigate,
}: {
  source: CitationSource | null;
  onClose: () => void;
  onNavigate: () => void;
}) {
  const [data, setData] = useState<{
      markdown: string;
      notice: string;
      source: CitationSource;
    } | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(false);
  useEffect(() => {
    setData(null);
    setError("");
    if (!source?.citation_url) return;
    let active = true;
    setLoading(true);
    void api(source.citation_url)
      .then((v) => {
        if (active) setData(v);
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [source?.citation_url]);
  return (
    <Modal
      open={!!source}
      onOpenChange={(v) => {
        if (!v) onClose();
      }}
      title="인용 원문 확인"
      description="AI가 받은 출처 조각을 현재 문서 권한·버전·원문 해시와 다시 대조합니다."
      wide
    >
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        data && (
          <>
            <div
              style={{
                display: "flex",
                gap: 10,
                alignItems: "center",
                flexWrap: "wrap",
                marginBottom: 14,
              }}
            >
              <strong>{data.source.title}</strong>
              <Badge>문서 v{data.source.version}</Badge>
              <Badge>
                {data.source.attachment_id
                  ? positionName(data.source.attachment_position || {})
                  : `${data.source.start_line}–${data.source.end_line}행`}
              </Badge>
            </div>
            <pre
              style={{
                whiteSpace: "pre-wrap",
                overflowWrap: "anywhere",
                fontSize: 16,
                lineHeight: 1.8,
                padding: 18,
                background: "var(--bg)",
                borderRadius: 12,
                maxHeight: "50vh",
                overflow: "auto",
              }}
            >
              {data.markdown}
            </pre>
            <p className="muted">{data.notice}</p>
            <Link
              to={
                data.source.url ||
                (data.source.attachment_id
                  ? `/app/attachments/${data.source.attachment_id}`
                  : `/app/documents/${data.source.id}`)
              }
              className="button primary"
              onClick={() => {
                onClose();
                onNavigate();
              }}
            >
              {data.source.attachment_id
                ? "첨부의 원본 위치로 이동"
                : "문서의 해당 줄로 이동"}
            </Link>
          </>
        )
      )}
    </Modal>
  );
}
