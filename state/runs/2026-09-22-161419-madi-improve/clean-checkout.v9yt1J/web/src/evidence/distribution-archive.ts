export type SignedFile = {
  source_id: string;
  path: string;
  kind: "document" | "attachment";
  bytes: number;
  sha256: string;
  parent_source_id: string;
  source_version: number;
  metadata: { title: string; tags: string[]; aliases: string[]; icon: string };
  metadata_hash: string;
  approval: { required: boolean; request_id: string; resource_hash: string };
};
export type SignedArchive = {
  manifest: {
    format: string;
    bundle_id: string;
    source_instance: string;
    source_workspace: string;
    receiver_instance: string;
    key_id: string;
    protection_revision: number;
    created_at: number;
    expires_at: number;
    files: SignedFile[];
  };
  manifest_base64: string;
  signature: string;
  manifest_hash: string;
  files: { path: string; data: Uint8Array }[];
};
export async function readSignedArchive(
  file: File,
  signal: AbortSignal,
): Promise<SignedArchive> {
  if (file.size > 54 * 1024 * 1024)
    throw new Error("서명 ZIP은 54MiB 이하여야 합니다.");
  const buffer = await file.arrayBuffer();
  if (signal.aborted) throw new DOMException("취소됨", "AbortError");
  return new Promise((resolve, reject) => {
    const worker = new Worker(
      new URL("./distribution-archive.worker.ts", import.meta.url),
      { type: "module" },
    );
    const cleanup = () => {
      worker.terminate();
      signal.removeEventListener("abort", abort);
    };
    const abort = () => {
      cleanup();
      reject(new DOMException("취소됨", "AbortError"));
    };
    signal.addEventListener("abort", abort, { once: true });
    worker.onerror = () => {
      cleanup();
      reject(new Error("서명 ZIP 검사 작업을 완료하지 못했습니다."));
    };
    worker.onmessage = (
      event: MessageEvent<SignedArchive & { error?: string }>,
    ) => {
      cleanup();
      event.data.error
        ? reject(new Error(event.data.error))
        : resolve(event.data);
    };
    worker.postMessage(buffer, [buffer]);
  });
}
