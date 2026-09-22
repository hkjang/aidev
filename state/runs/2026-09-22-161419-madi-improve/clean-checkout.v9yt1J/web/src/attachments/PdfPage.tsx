import { useEffect, useRef, useState } from "react";
import { getDocument, GlobalWorkerOptions } from "pdfjs-dist";
import workerURL from "pdfjs-dist/build/pdf.worker.min.mjs?url";
import { ErrorBox, Loading } from "../ui";
import type { Position } from "./types";

GlobalWorkerOptions.workerSrc = workerURL;
export default function PdfPage({
  attachmentID,
  page,
  position,
}: {
  attachmentID: string;
  page: number;
  position?: Position;
}) {
  const host = useRef<HTMLDivElement>(null);
  const canvas = useRef<HTMLCanvasElement>(null);
  const [width, setWidth] = useState(760),
    [busy, setBusy] = useState(true),
    [error, setError] = useState<unknown>(null);
  useEffect(() => {
    const el = host.current;
    if (!el) return;
    const observer = new ResizeObserver(([entry]) =>
      setWidth(Math.max(240, Math.min(1100, entry.contentRect.width))),
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    let active = true;
    setBusy(true);
    setError(null);
    const loading = getDocument({
      url: `/api/v1/attachments/${attachmentID}`,
      httpHeaders: { "X-Madi-Request": "1" },
      withCredentials: true,
      cMapUrl: "/assets/pdfjs/cmaps/",
      cMapPacked: true,
      standardFontDataUrl: "/assets/pdfjs/standard_fonts/",
      wasmUrl: "/assets/pdfjs/wasm/",
      enableXfa: false,
      maxImageSize: 16_000_000,
      canvasMaxAreaInBytes: 64 << 20,
      disableAutoFetch: true,
      disableStream: true,
    });
    let rendering: { cancel: () => void } | undefined;
    void (async () => {
      const pdf = await loading.promise;
      if (!active) return;
      const source = await pdf.getPage(
        Math.max(1, Math.min(page, pdf.numPages)),
      );
      if (!active || !canvas.current) return;
      const natural = source.getViewport({ scale: 1 });
      const viewport = source.getViewport({ scale: width / natural.width });
      const el = canvas.current;
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      el.width = Math.floor(viewport.width * ratio);
      el.height = Math.floor(viewport.height * ratio);
      el.style.width = "100%";
      el.style.height = "auto";
      const task = source.render({
        canvas: el,
        viewport,
        transform: ratio === 1 ? undefined : [ratio, 0, 0, ratio, 0, 0],
      });
      rendering = task;
      await task.promise;
      if (active) setBusy(false);
    })().catch((e) => {
      if (active) {
        setError(
          e instanceof Error
            ? new Error(
                "PDF 원본을 표시하지 못했습니다. 파일 권한·암호화 여부를 확인하거나 원본을 다운로드하세요.",
              )
            : e,
        );
        setBusy(false);
      }
    });
    return () => {
      active = false;
      rendering?.cancel();
      void loading.destroy();
    };
  }, [attachmentID, page, width]);
  const b = position?.bounds,
    size = position?.page_size;
  const highlight =
    b?.length === 4 &&
    size?.length === 2 &&
    size[0] > 0 &&
    size[1] > 0 &&
    position?.page === page;
  return (
    <div ref={host} className="extract-pdf">
      <ErrorBox error={error} />
      {busy && <Loading />}
      <div className="extract-canvas">
        <canvas ref={canvas} aria-label={`PDF ${page}쪽 원본`} />
        {highlight && (
          <span
            className="extract-highlight"
            aria-label="선택한 근거의 원본 위치"
            style={{
              left: `${(b![0] / size![0]) * 100}%`,
              top: `${(b![1] / size![1]) * 100}%`,
              width: `${((b![2] - b![0]) / size![0]) * 100}%`,
              height: `${((b![3] - b![1]) / size![1]) * 100}%`,
            }}
          />
        )}
      </div>
    </div>
  );
}
