import { Unzip, UnzipInflate } from "fflate";
import { sha256 } from "@noble/hashes/sha2.js";

const scope = self as unknown as {
  onmessage: ((event: MessageEvent<ArrayBuffer>) => void) | null;
  postMessage: (data: unknown, transfer?: Transferable[]) => void;
};
const MAX = 50 * 1024 * 1024,
  MANIFEST = 2 * 1024 * 1024;
const hex = (data: Uint8Array) =>
  Array.from(sha256(data))
    .map((v) => v.toString(16).padStart(2, "0"))
    .join("");
const safe = (name: string) =>
  name.length > 0 &&
  name.length <= 512 &&
  !name.startsWith("/") &&
  !/[\\\0:]/.test(name) &&
  name.split("/").every((v) => v && v !== "." && v !== "..");
scope.onmessage = (event) => {
  try {
    const input = new Uint8Array(event.data);
    if (input.length > MAX + MANIFEST + 2 * 1024 * 1024)
      throw new Error("서명 ZIP은 54MiB 이하여야 합니다.");
    let count = 0,
      total = 0;
    const names = new Set<string>(),
      files = new Map<string, Uint8Array>();
    const unzip = new Unzip((file) => {
      if (++count > 1002 || !safe(file.name) || names.has(file.name))
        throw new Error("중복·잘못된 경로 또는 파일 수 한도 초과입니다.");
      names.add(file.name);
      const limit =
        file.name === "madi-distribution.json"
          ? MANIFEST
          : file.name === "madi-distribution.sig"
            ? 128
            : MAX;
      if (file.originalSize !== undefined && file.originalSize > limit)
        throw new Error("압축 해제 파일 크기 한도를 초과합니다.");
      const chunks: Uint8Array[] = [];
      let size = 0;
      file.ondata = (error, chunk, final) => {
        if (error) throw error;
        size += chunk.length;
        total += chunk.length;
        if (size > limit || total > MAX + MANIFEST + 128)
          throw new Error("압축 해제 합계 크기 한도를 초과합니다.");
        chunks.push(chunk);
        if (final) {
          const data = new Uint8Array(size);
          let at = 0;
          for (const c of chunks) {
            data.set(c, at);
            at += c.length;
          }
          files.set(file.name, data);
        }
      };
      file.start();
    });
    unzip.register(UnzipInflate);
    // Small compressed chunks bound each synchronous decompression allocation.
    for (let at = 0; at < input.length; at += 1024)
      unzip.push(input.subarray(at, at + 1024), at + 1024 >= input.length);
    if (files.size !== count)
      throw new Error("압축 파일이 끝나기 전에 중단됐습니다.");
    const raw = files.get("madi-distribution.json"),
      signature = files.get("madi-distribution.sig");
    if (!raw || !signature)
      throw new Error("madi 서명 매니페스트와 서명 파일이 필요합니다.");
    const decoder = new TextDecoder("utf-8", { fatal: true });
    const manifest = JSON.parse(decoder.decode(raw));
    const uuid = (v: unknown) =>
      typeof v === "string" &&
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(v);
    if (
      !manifest ||
      typeof manifest !== "object" ||
      ![
        manifest.bundle_id,
        manifest.source_instance,
        manifest.source_workspace,
        manifest.receiver_instance,
        manifest.key_id,
      ].every(uuid) ||
      !Number.isSafeInteger(manifest.created_at) ||
      !Number.isSafeInteger(manifest.expires_at) ||
      manifest.created_at < 1 ||
      manifest.expires_at <= manifest.created_at ||
      manifest.expires_at > 253402300799 ||
      manifest.expires_at - manifest.created_at > 365 * 86400
    )
      throw new Error("배포 식별자와 유효기간 형식을 확인하세요.");
    files.delete("madi-distribution.json");
    files.delete("madi-distribution.sig");
    if (
      manifest.format !== "madi-distribution-v1" ||
      !Array.isArray(manifest.files) ||
      manifest.files.length < 1 ||
      manifest.files.length > 1000 ||
      manifest.files.length !== files.size
    )
      throw new Error("서명된 파일 목록이 일치하지 않습니다.");
    const declared = new Set<string>();
    for (const f of manifest.files) {
      if (
        !f ||
        typeof f.path !== "string" ||
        declared.has(f.path) ||
        !uuid(f.source_id) ||
        !["document", "attachment"].includes(f.kind) ||
        !Number.isInteger(f.source_version) ||
        f.source_version < 0 ||
        f.source_version > 2147483647 ||
        typeof f.metadata?.title !== "string" ||
        !Array.isArray(f.metadata?.tags) ||
        !f.metadata.tags.every((v: unknown) => typeof v === "string") ||
        !Array.isArray(f.metadata?.aliases) ||
        !f.metadata.aliases.every((v: unknown) => typeof v === "string")
      )
        throw new Error("서명된 경로가 중복됐습니다.");
      declared.add(f.path);
      const data = files.get(f.path);
      if (!data || data.length !== f.bytes || hex(data) !== f.sha256)
        throw new Error("실제 파일과 서명 매니페스트의 SHA256이 다릅니다.");
    }
    let binary = "";
    for (let at = 0; at < raw.length; at += 8192)
      binary += String.fromCharCode(...raw.subarray(at, at + 8192));
    const output = [...files].map(([path, data]) => ({ path, data }));
    scope.postMessage(
      {
        manifest,
        manifest_base64: btoa(binary),
        signature: decoder.decode(signature),
        manifest_hash: hex(raw),
        files: output,
      },
      output.map((f) => f.data.buffer as ArrayBuffer),
    );
  } catch (error) {
    scope.postMessage({
      error:
        error instanceof Error ? error.message : "서명 ZIP을 읽지 못했습니다.",
    });
  }
};
