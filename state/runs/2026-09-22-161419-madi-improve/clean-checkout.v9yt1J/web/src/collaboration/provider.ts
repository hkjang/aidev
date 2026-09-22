import * as Y from "yjs";
import {
  Awareness,
  applyAwarenessUpdate,
  removeAwarenessStates,
} from "y-protocols/awareness";
import * as encoding from "lib0/encoding";

export type CollaborationStatus =
  | "connecting"
  | "syncing"
  | "connected"
  | "disconnected"
  | "conflict"
  | "error";
export type CollaborationSaveState =
  "local" | "committing" | "confirmed" | "reconnecting" | "recovery";
export type CollaborationSnapshot = {
  epoch: string;
  sequence: number;
  version: number;
  markdown: string;
  title: string;
  tags: string[];
  documentStatus: string;
  canWrite: boolean;
  user: { id: string; name: string; color: string };
};
type WireMessage = {
  type: string;
  schema?: string;
  epoch?: string;
  sequence?: number;
  version?: number;
  state?: string;
  markdown?: string;
  title?: string;
  tags?: string[];
  status?: string;
  can_write?: boolean;
  id?: number;
  code?: string;
  error?: string;
  user?: CollaborationSnapshot["user"];
  presence?: {
    client_id: number;
    clock: number;
    state: Record<string, unknown>;
  }[];
  committed?: boolean;
  reset_reason?: string;
  state_mode?: "full" | "delta";
  from_sequence?: number;
};
type Listener = (...args: any[]) => void;
const schema = "madi-tiptap-v1";

export function encodeUpdate(update: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < update.length; i += 8192)
    binary += String.fromCharCode(...update.subarray(i, i + 8192));
  return btoa(binary);
}
export function decodeUpdate(value: string): Uint8Array {
  return Uint8Array.from(atob(value), (character) => character.charCodeAt(0));
}

/**
 * Native Yjs transport backed by madi's Go/PostgreSQL server. This is not the
 * y-websocket wire protocol. It supplies the standard Y.Doc + Awareness objects
 * consumed by the open-source TipTap Collaboration and CollaborationCaret.
 *
 * A saved indicator means the server committed BOTH CRDT state and canonical
 * Markdown. A REST replacement starts a new epoch; we retain a recovery update
 * and stop, never automatically merge a stale editor over that replacement.
 */
export class MadiCollaborationProvider {
  readonly document = new Y.Doc();
  readonly awareness = new Awareness(this.document);
  readonly documentId: string;
  status: CollaborationStatus = "disconnected";
  snapshot: CollaborationSnapshot | null = null;
  synced = false;
  error = "";
  recoveryUpdate: Uint8Array | null = null;
  confirmedAt: number | null = null;
  resetReason = "";
  private listeners = new Map<string, Set<Listener>>();
  private socket: WebSocket | null = null;
  private destroyed = false;
  private retry: ReturnType<typeof setTimeout> | null = null;
  private flushTimer: ReturnType<typeof setTimeout> | null = null;
  private awarenessTimer: ReturnType<typeof setTimeout> | null = null;
  private retryCount = 0;
  private updates: Uint8Array[] = [];
  private request = 0;
  private revision = 0;
  private savedRevision = 0;
  private inFlight = new Map<number, number>();
  private remoteClients = new Set<number>();
  private canSend = false;
  private seeding = false;
  private initialized = false;
  private lastAwareness = 0;

  constructor(documentId: string) {
    this.documentId = documentId;
    this.document.on("update", this.onDocumentUpdate);
    this.awareness.on("update", this.onAwarenessUpdate);
  }

  get hasUnsavedChanges(): boolean {
    return this.revision > this.savedRevision || this.seeding;
  }
  get saveState(): CollaborationSaveState {
    if (this.status === "conflict" || this.status === "error")
      return "recovery";
    if (this.status !== "connected") return "reconnecting";
    if (this.inFlight.size > 0) return "committing";
    return this.hasUnsavedChanges ? "local" : "confirmed";
  }
  insertMarkdown(markdown: string): void {
    this.emit("insert", markdown);
  }
  rejectUnsupported(reason: string): void {
    this.fail(reason);
  }
  on(event: string, listener: Listener): this {
    let set = this.listeners.get(event);
    if (!set) this.listeners.set(event, (set = new Set()));
    set.add(listener);
    return this;
  }
  off(event: string, listener: Listener): this {
    this.listeners.get(event)?.delete(listener);
    return this;
  }
  private emit(event: string, ...args: any[]): void {
    for (const fn of this.listeners.get(event) || []) fn(...args);
  }
  private setStatus(status: CollaborationStatus): void {
    this.status = status;
    this.emit("status", { status });
    this.emit("change");
  }

  connect(): void {
    if (this.destroyed || this.status === "conflict" || this.socket) return;
    const url = new URL(
      `/api/v1/documents/${encodeURIComponent(this.documentId)}/collaboration`,
      window.location.href,
    );
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    this.setStatus("connecting");
    const socket = new WebSocket(url);
    this.socket = socket;
    socket.onopen = () => {
      this.setStatus("syncing");
    };
    socket.onmessage = (event) => {
      try {
        this.receive(JSON.parse(event.data) as WireMessage);
      } catch (error) {
        this.fail(
          error instanceof Error
            ? error.message
            : "편집 상태를 읽지 못했습니다",
        );
      }
    };
    socket.onclose = (event) => {
      if (this.socket !== socket) return;
      this.socket = null;
      this.canSend = false;
      this.synced = false;
      removeAwarenessStates(this.awareness, [...this.remoteClients], this);
      this.remoteClients.clear();
      if (
        this.destroyed ||
        this.status === "conflict" ||
        this.status === "error"
      )
        return;
      if (event.code === 1008) {
        this.fail(event.reason || "문서 권한 또는 세션이 만료되었습니다");
        return;
      }
      this.setStatus("disconnected");
      // The Y.Doc remains intact: reconnect merges and resends unacknowledged
      // operations only if the server is still in the exact same epoch.
      this.retry = setTimeout(
        () => {
          this.retry = null;
          this.connect();
        },
        Math.min(15000, 500 * 2 ** Math.min(this.retryCount++, 5)),
      );
    };
    socket.onerror = () => {
      this.error =
        "공동 편집 연결을 확인하고 있습니다. 로컬 편집 내용은 보관됩니다";
      this.emit("change");
    };
  }

  /** Call after the editor has setContent(seed.markdown, {contentType:'markdown'}). */
  seedFromCurrentDocument(): void {
    if (
      !this.snapshot ||
      this.initialized ||
      !this.snapshot.canWrite ||
      this.seeding
    )
      return;
    this.seeding = true;
    this.revision++;
    this.sendUpdate("seed", Y.encodeStateAsUpdate(this.document));
    this.emit("change");
  }

  /** Optional explicit save button; regular edits are automatically batched. */
  flush(): void {
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }
    if (!this.canSend || !this.snapshot?.canWrite || !this.updates.length)
      return;
    const update = Y.mergeUpdates(this.updates);
    this.updates = [];
    this.sendUpdate("update", update);
  }

  async waitForSaved(timeout = 12000): Promise<void> {
    this.flush();
    if (!this.hasUnsavedChanges) return;
    if (this.status === "conflict" || this.status === "error")
      throw new Error(this.error);
    await new Promise<void>((resolve, reject) => {
      const finish = () => {
        if (this.status === "conflict" || this.status === "error") {
          cleanup();
          reject(new Error(this.error));
        } else if (!this.hasUnsavedChanges) {
          cleanup();
          resolve();
        }
      };
      const timer = setTimeout(() => {
        cleanup();
        reject(
          new Error(
            "저장 확인을 기다리는 중입니다. 연결을 확인하고 로컬 초안을 보관하세요",
          ),
        );
      }, timeout);
      const cleanup = () => {
        clearTimeout(timer);
        this.off("change", finish);
      };
      this.on("change", finish);
    });
  }

  private sendUpdate(type: "seed" | "update", update: Uint8Array): void {
    if (this.socket?.readyState !== WebSocket.OPEN || !this.snapshot) return;
    const id = ++this.request;
    this.inFlight.set(id, this.revision);
    this.socket.send(
      JSON.stringify({
        type,
        schema,
        id,
        epoch: this.snapshot.epoch,
        version: this.snapshot.version,
        state: encodeUpdate(update),
      }),
    );
  }

  private onDocumentUpdate = (update: Uint8Array, origin: unknown): void => {
    if (origin === this || this.destroyed) return;
    // The editor's initial empty paragraph/seed transaction is not a user save.
    if (!this.initialized) return;
    this.revision++;
    this.updates.push(update);
    if (!this.flushTimer) this.flushTimer = setTimeout(() => this.flush(), 120);
    this.emit("change");
  };

  private onAwarenessUpdate = (_changes: unknown, origin: unknown): void => {
    if (origin === this || this.destroyed) return;
    if (this.awarenessTimer) return;
    this.awarenessTimer = setTimeout(() => {
      this.awarenessTimer = null;
      if (this.socket?.readyState !== WebSocket.OPEN || !this.snapshot) return;
      const local = this.awareness.getLocalState();
      const clock =
        this.awareness.meta.get(this.document.clientID)?.clock ||
        ++this.lastAwareness;
      this.socket.send(
        JSON.stringify({
          type: "awareness",
          epoch: this.snapshot.epoch,
          client_id: this.document.clientID,
          clock,
          awareness: local || {},
        }),
      );
    }, 100);
  };

  private receive(message: WireMessage): void {
    if (message.type === "error") {
      if (message.code === "missing_state" && this.canSend) {
        this.sendUpdate("update", Y.encodeStateAsUpdate(this.document));
        return;
      }
      this.fail(message.error || "공동 편집을 저장하지 못했습니다");
      return;
    }
    if (message.type === "presence") {
      const encoder = encoding.createEncoder();
      const entries = (message.presence || []).filter(
        (entry) => entry.client_id !== this.document.clientID,
      );
      encoding.writeVarUint(encoder, entries.length);
      const clients = new Set<number>();
      for (const entry of entries) {
        clients.add(entry.client_id);
        encoding.writeVarUint(encoder, entry.client_id);
        encoding.writeVarUint(encoder, entry.clock);
        encoding.writeVarString(encoder, JSON.stringify(entry.state));
      }
      applyAwarenessUpdate(
        this.awareness,
        encoding.toUint8Array(encoder),
        this,
      );
      removeAwarenessStates(
        this.awareness,
        [...this.remoteClients].filter((id) => !clients.has(id)),
        this,
      );
      this.remoteClients = clients;
      this.emit("presence");
      return;
    }
    if (!["hello", "sync", "ack", "reset"].includes(message.type)) return;
    if (message.schema !== schema || !message.epoch) {
      this.fail("편집기 스키마 버전이 다릅니다. 화면을 새로고침하세요");
      return;
    }
    if (
      message.state_mode === "delta" &&
      (!this.initialized ||
        this.snapshot?.epoch !== message.epoch ||
        this.snapshot.sequence !== (message.from_sequence || 0))
    ) {
      // A missed notification/reordered connection cannot be assumed delivered.
      // Reopen without discarding the Y.Doc; hello supplies a durable full state.
      this.socket?.close(1012, "증분 저장 순서 재동기화");
      return;
    }
    if (
      message.type === "reset" ||
      (this.snapshot && this.snapshot.epoch !== message.epoch)
    ) {
      if (message.committed && message.id) {
        this.savedRevision = Math.max(
          this.savedRevision,
          this.inFlight.get(message.id) || 0,
        );
        this.confirmedAt = Date.now();
        this.inFlight.delete(message.id);
      }
      this.resetReason = message.reset_reason || "source_replaced";
      this.recoveryUpdate = Y.encodeStateAsUpdate(this.document);
      this.error = ["history_compacted", "manual_compaction"].includes(
        this.resetReason,
      )
        ? "장시간 편집 이력을 압축하여 새 편집 기준이 생겼습니다. 현재 초안을 보관하고 최신 문서로 다시 연결하세요. 오래된 초안은 자동으로 덮어쓰지 않습니다."
        : "다른 편집 방식에서 문서가 변경되었습니다. 현재 초안을 보관하고 최신 문서로 다시 연결하세요";
      this.canSend = false;
      this.setStatus("conflict");
      this.emit("reset", {
        serverMarkdown: message.markdown || "",
        recoveryUpdate: this.recoveryUpdate,
        hasUnsavedChanges: this.hasUnsavedChanges,
      });
      this.socket?.close();
      return;
    }
    const previouslyInitialized = this.initialized;
    this.snapshot = {
      epoch: message.epoch,
      sequence: message.sequence || 0,
      version: message.version || 1,
      markdown: message.markdown || "",
      canWrite: message.can_write === true,
      title: message.title || "",
      tags: message.tags || [],
      documentStatus: message.status || "draft",
      user: message.user ||
        this.snapshot?.user || { id: "", name: "사용자", color: "#0f766e" },
    };
    this.awareness.setLocalStateField("user", this.snapshot.user);
    if (message.state) {
      Y.applyUpdate(this.document, decodeUpdate(message.state), this);
      this.initialized = true;
      this.canSend = this.snapshot.canWrite;
      this.synced = true;
      this.seeding = false;
      this.error = "";
      this.retryCount = 0;
      this.setStatus("connected");
      if (message.type === "ack") {
        this.savedRevision = Math.max(
          this.savedRevision,
          this.inFlight.get(message.id || 0) || 0,
        );
        this.confirmedAt = Date.now();
        this.inFlight.delete(message.id || 0);
        this.emit("saved", this.snapshot);
      }
      if (
        message.type === "hello" &&
        previouslyInitialized &&
        this.hasUnsavedChanges &&
        this.canSend
      ) {
        this.updates = [];
        this.inFlight.clear();
        this.sendUpdate("update", Y.encodeStateAsUpdate(this.document));
      }
      this.emit("synced", { state: true });
    } else if (message.type === "hello") {
      this.setStatus("syncing");
      if (this.snapshot.canWrite) this.emit("seed", this.snapshot);
      else {
        this.error = "작성자가 공동 편집을 시작하면 실시간으로 연결됩니다";
        this.emit("change");
      }
    }
    this.emit("snapshot", this.snapshot);
    this.emit("change");
  }

  private fail(message: string): void {
    this.error = message;
    this.recoveryUpdate = Y.encodeStateAsUpdate(this.document);
    this.canSend = false;
    this.setStatus("error");
    this.emit("error", { message });
    this.socket?.close();
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;
    if (this.retry) clearTimeout(this.retry);
    if (this.flushTimer) clearTimeout(this.flushTimer);
    if (this.awarenessTimer) clearTimeout(this.awarenessTimer);
    this.document.off("update", this.onDocumentUpdate);
    this.awareness.off("update", this.onAwarenessUpdate);
    this.awareness.destroy();
    this.socket?.close();
    this.socket = null;
    this.document.destroy();
    this.listeners.clear();
  }
}
