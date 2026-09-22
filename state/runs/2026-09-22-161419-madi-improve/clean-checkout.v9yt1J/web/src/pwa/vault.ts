import type { Doc } from "../api";

type Envelope = { id: string; iv: Uint8Array<ArrayBuffer>; data: ArrayBuffer };
type VaultMeta = {
  id: "vault";
  userID: string;
  salt: Uint8Array<ArrayBuffer>;
  check: Envelope;
};
export type OfflineDocument = {
  kind: "document";
  id: string;
  doc: Doc;
  saved_at: string;
  edited: boolean;
};
export type OfflineCapture = {
  kind: "capture";
  id: string;
  workspace_id: string;
  title: string;
  text: string;
  url: string;
  client_request_id: string;
  created_at: string;
};
export type OfflineRecord = OfflineDocument | OfflineCapture;
let activeKey: CryptoKey | null = null;
let activeUser = "";
let idle: ReturnType<typeof setTimeout> | undefined;
let generation = 0;
const DB_NAME = "madi-offline-v1";
const control =
  typeof BroadcastChannel === "function"
    ? new BroadcastChannel("madi-vault-control")
    : null;
control?.addEventListener("message", (event) => {
  if (event.data === "lock") lockOfflineVault(false);
});
export const vaultSupported = () =>
  !!(
    globalThis.isSecureContext &&
    globalThis.crypto?.subtle &&
    globalThis.indexedDB
  );
const bytes = (value: string) => new TextEncoder().encode(value);
function changed() {
  window.dispatchEvent(new Event("madi-vault-change"));
}
export function lockOfflineVault(broadcast = true) {
  activeKey = null;
  activeUser = "";
  generation++;
  clearTimeout(idle);
  changed();
  if (broadcast) control?.postMessage("lock");
}
function touch() {
  clearTimeout(idle);
  idle = setTimeout(lockOfflineVault, 5 * 60_000);
}
export function vaultUnlocked(userID?: string) {
  return !!activeKey && (!userID || activeUser === userID);
}
export async function vaultMetadata(): Promise<{ userID: string } | null> {
  const meta = await read<VaultMeta>("meta", "vault");
  return meta ? { userID: meta.userID } : null;
}
function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => {
      request.result.createObjectStore("meta", { keyPath: "id" });
      request.result.createObjectStore("records", { keyPath: "id" });
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () =>
      reject(
        new Error(
          "기기 저장 공간을 열 수 없습니다. 브라우저 저장 권한을 확인하세요.",
        ),
      );
  });
}
async function read<T>(store: string, id?: string): Promise<T | null> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction(store, "readonly");
    const req = id
      ? tx.objectStore(store).get(id)
      : tx.objectStore(store).getAll();
    req.onsuccess = () => resolve(req.result ?? null);
    req.onerror = () => reject(req.error);
    tx.oncomplete = () => db.close();
    tx.onabort = () => db.close();
  });
}
async function write(
  store: string,
  value: unknown,
  action: "put" | "delete" | "clear" | "add" = "put",
) {
  const db = await openDB();
  return new Promise<void>((resolve, reject) => {
    const tx = db.transaction(store, "readwrite");
    const object = tx.objectStore(store);
    if (action === "clear") object.clear();
    else if (action === "delete") object.delete(value as string);
    else if (action === "add") object.add(value);
    else object.put(value);
    tx.oncomplete = () => {
      db.close();
      resolve();
    };
    tx.onabort = tx.onerror = () => {
      db.close();
      reject(new Error("기기 저장 공간이 부족하거나 저장 권한이 없습니다."));
    };
  });
}
async function derive(passphrase: string, salt: Uint8Array<ArrayBuffer>) {
  const material = await crypto.subtle.importKey(
    "raw",
    bytes(passphrase),
    "PBKDF2",
    false,
    ["deriveKey"],
  );
  return crypto.subtle.deriveKey(
    { name: "PBKDF2", hash: "SHA-256", salt, iterations: 310_000 },
    material,
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt", "decrypt"],
  );
}
async function encrypt(
  key: CryptoKey,
  user: string,
  id: string,
  value: unknown,
): Promise<Envelope> {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  return {
    id,
    iv,
    data: await crypto.subtle.encrypt(
      { name: "AES-GCM", iv, additionalData: bytes(`${user}:${id}:1`) },
      key,
      bytes(JSON.stringify(value)),
    ),
  };
}
async function decrypt<T>(
  key: CryptoKey,
  user: string,
  value: Envelope,
): Promise<T> {
  const data = await crypto.subtle.decrypt(
    {
      name: "AES-GCM",
      iv: value.iv,
      additionalData: bytes(`${user}:${value.id}:1`),
    },
    key,
    value.data,
  );
  return JSON.parse(new TextDecoder().decode(data));
}
export async function createVault(userID: string, passphrase: string) {
  if (!vaultSupported())
    throw new Error(
      "오프라인 암호화 보관함은 HTTPS 또는 localhost에서 사용할 수 있습니다.",
    );
  if (passphrase.length < 12)
    throw new Error(
      "보관함 암호는 12자 이상 입력하세요. 서버 로그인 암호와 다르게 설정하세요.",
    );
  if (await vaultMetadata())
    throw new Error("기존 보관함을 먼저 잠금 해제하거나 삭제하세요.");
  const attempt = ++generation;
  const salt = crypto.getRandomValues(new Uint8Array(32));
  const key = await derive(passphrase, salt);
  const check = await encrypt(key, userID, "check", { madi: "offline-v1" });
  if (attempt !== generation) throw new Error("보관함 열기가 취소되었습니다.");
  await write(
    "meta",
    { id: "vault", userID, salt, check } satisfies VaultMeta,
    "add",
  );
  if (attempt !== generation) throw new Error("보관함 열기가 취소되었습니다.");
  activeKey = key;
  activeUser = userID;
  touch();
  changed();
}
export async function unlockVault(passphrase: string, userID?: string) {
  if (!vaultSupported())
    throw new Error(
      "이 브라우저에서는 안전한 오프라인 암호화를 지원하지 않습니다.",
    );
  const attempt = ++generation;
  const meta = await read<VaultMeta>("meta", "vault");
  if (!meta)
    throw new Error(
      "저장된 보관함이 없습니다. 온라인 상태에서 먼저 보관함을 만드세요.",
    );
  if (userID && meta.userID !== userID)
    throw new Error(
      "다른 사용자의 보관함입니다. 해당 계정으로 로그인하거나 기기 보관함을 삭제하세요.",
    );
  const key = await derive(passphrase, meta.salt);
  try {
    const check = await decrypt<{ madi: string }>(key, meta.userID, meta.check);
    if (check.madi !== "offline-v1") throw new Error();
  } catch {
    throw new Error(
      "보관함 암호가 올바르지 않거나 저장 데이터가 손상되었습니다.",
    );
  }
  if (attempt !== generation) throw new Error("보관함 열기가 취소되었습니다.");
  activeKey = key;
  activeUser = meta.userID;
  touch();
  changed();
}
export async function clearVault() {
  lockOfflineVault();
  const db = await openDB();
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(["meta", "records"], "readwrite");
    tx.objectStore("meta").clear();
    tx.objectStore("records").clear();
    tx.oncomplete = () => {
      db.close();
      resolve();
    };
    tx.onabort = () => {
      db.close();
      reject(tx.error);
    };
  });
  changed();
}
function keyContext() {
  if (!activeKey) throw new Error("먼저 오프라인 보관함을 잠금 해제하세요.");
  touch();
  return { key: activeKey, user: activeUser, epoch: generation };
}
export async function listOfflineRecords(): Promise<OfflineRecord[]> {
  const { key, user, epoch } = keyContext();
  const rows = (await read<Envelope[]>("records")) || [];
  const records = await Promise.all(
    rows.map((row) => decrypt<OfflineRecord>(key, user, row)),
  );
  if (epoch !== generation) throw new Error("보관함이 잠겼습니다.");
  return records;
}
export async function saveOfflineRecord(value: OfflineRecord) {
  const { key, user, epoch } = keyContext();
  if (bytes(JSON.stringify(value)).length > 1_048_576)
    throw new Error("문서 하나당 오프라인 보관 한도는 1 MB입니다.");
  const all = (await read<Envelope[]>("records")) || [];
  if (all.length >= 100 && !all.some((row) => row.id === value.id))
    throw new Error(
      "기기에는 최대 100개의 문서와 임시 기록을 보관할 수 있습니다.",
    );
  if (
    all.reduce((total, row) => total + row.data.byteLength, 0) >
    50 * 1_048_576
  )
    throw new Error("오프라인 보관함의 50 MB 한도에 도달했습니다.");
  const envelope = await encrypt(key, user, value.id, value);
  if (epoch !== generation) throw new Error("보관함이 잠겼습니다.");
  await write("records", envelope);
  changed();
}
export async function removeOfflineRecord(id: string) {
  keyContext();
  await write("records", id, "delete");
  changed();
}
window.addEventListener("pagehide", () => lockOfflineVault());
