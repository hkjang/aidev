import type {
  AiStreamEvent,
  AiModelLimit,
  AppDocument,
  ApiProblem,
  BrandingAsset,
  AuditEntry,
  Category,
  KeyPermissionDefinition,
  KeyPermissionOptions,
  KeyPermissionTemplate,
  OidcTestResult,
  PageResult,
  PersonalKey,
  PublicConfig,
  Review,
  ReviewDetail,
  SecurityCheckSettings,
  SecurityCheckTestResult,
  SecurityCheckView,
  Session,
  StoreApp,
  User,
  UserPreferences,
  VersionInfo,
} from "../types";

const DEFAULT_TIMEOUT_MS = 15_000;
let csrfToken = "";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  readonly details?: unknown;

  constructor(status: number, problem: ApiProblem) {
    super(problem.message);
    this.name = "ApiError";
    this.status = status;
    this.code = problem.code;
    this.requestId = problem.requestId;
    this.details = problem.details;
  }
}

export function setCsrfToken(value?: string): void {
  csrfToken = value ?? "";
}

type ApiOptions = RequestInit & { timeoutMs?: number };

function combineSignals(
  signal: AbortSignal | null | undefined,
  timeoutMs: number,
): {
  signal: AbortSignal;
  dispose: () => void;
} {
  const controller = new AbortController();
  const timeout = window.setTimeout(
    () =>
      controller.abort(
        new DOMException("요청 시간이 초과되었습니다.", "TimeoutError"),
      ),
    timeoutMs,
  );
  const onAbort = () => controller.abort(signal?.reason);
  signal?.addEventListener("abort", onAbort, { once: true });
  return {
    signal: controller.signal,
    dispose: () => {
      window.clearTimeout(timeout);
      signal?.removeEventListener("abort", onAbort);
    },
  };
}

function unwrap<T>(payload: unknown): T {
  if (payload && typeof payload === "object" && "data" in payload) {
    return (payload as { data: T }).data;
  }
  return payload as T;
}

function problemFrom(payload: unknown, status: number): ApiProblem {
  const fallback =
    status === 401
      ? "로그인이 필요합니다."
      : status === 403
        ? "접근 권한이 없습니다."
        : "요청을 처리하지 못했습니다.";
  if (payload && typeof payload === "object") {
    const body = payload as {
      error?: Partial<ApiProblem>;
      message?: string;
      code?: string;
      requestId?: string;
    };
    const source = body.error ?? body;
    return {
      code: source.code ?? `HTTP_${status}`,
      message: source.message ?? fallback,
      requestId: source.requestId,
      details: "details" in source ? source.details : undefined,
    };
  }
  return { code: `HTTP_${status}`, message: fallback };
}

export async function apiFetch<T>(
  path: string,
  options: ApiOptions = {},
): Promise<T> {
  const {
    timeoutMs = DEFAULT_TIMEOUT_MS,
    headers,
    body,
    signal: callerSignal,
    ...init
  } = options;
  const combined = combineSignals(callerSignal, timeoutMs);
  const requestHeaders = new Headers(headers);
  requestHeaders.set("Accept", "application/json");
  if (
    body &&
    !(body instanceof FormData) &&
    !requestHeaders.has("Content-Type")
  ) {
    requestHeaders.set("Content-Type", "application/json");
  }
  if (
    csrfToken &&
    !["GET", "HEAD"].includes((init.method ?? "GET").toUpperCase())
  ) {
    requestHeaders.set("X-CSRF-Token", csrfToken);
  }

  try {
    const response = await fetch(path, {
      ...init,
      body,
      headers: requestHeaders,
      signal: combined.signal,
      credentials: "include",
    });
    const contentType = response.headers.get("content-type") ?? "";
    const payload: unknown =
      response.status === 204
        ? undefined
        : contentType.includes("json")
          ? await response.json()
          : await response.text();
    if (!response.ok)
      throw new ApiError(
        response.status,
        problemFrom(payload, response.status),
      );
    return unwrap<T>(payload);
  } finally {
    combined.dispose();
  }
}

function listFrom<T>(payload: unknown, keys: string[]): T[] {
  if (Array.isArray(payload)) return payload as T[];
  if (payload && typeof payload === "object") {
    const object = payload as Record<string, unknown>;
    for (const key of ["items", ...keys]) {
      if (Array.isArray(object[key])) return object[key] as T[];
    }
  }
  return [];
}

function pageFrom<T>(payload: unknown, keys: string[]): PageResult<T> {
  const object =
    payload && typeof payload === "object"
      ? (payload as Record<string, unknown>)
      : {};
  const items = listFrom<T>(payload, keys);
  return {
    items,
    total: Number(object.total ?? object.totalCount ?? items.length),
    page: Number(
      object.page ??
        (Number(object.limit) > 0
          ? Math.floor(Number(object.offset ?? 0) / Number(object.limit)) + 1
          : 1),
    ),
    pageSize: Number(object.pageSize ?? object.limit ?? items.length),
  };
}

function toQuery(
  params: Record<string, string | number | boolean | undefined>,
): string {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== "") query.set(key, String(value));
  });
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}

function pagedParams(
  params: Record<string, string | number | boolean | undefined>,
): Record<string, string | number | boolean | undefined> {
  const page = Math.max(1, Number(params.page ?? 1) || 1);
  const limit = Math.max(
    1,
    Number(params.pageSize ?? params.limit ?? 24) || 24,
  );
  const { page: _page, pageSize: _pageSize, ...rest } = params;
  return { ...rest, limit, offset: params.offset ?? (page - 1) * limit };
}

export const api = {
  version: (signal?: AbortSignal) =>
    apiFetch<VersionInfo>("/api/version", { signal }),
  publicConfig: (signal?: AbortSignal) =>
    apiFetch<PublicConfig>("/api/v1/public/config", { signal }),
  apps: async (
    params: Record<string, string | number | boolean | undefined>,
    signal?: AbortSignal,
  ) => {
    const payload = await apiFetch<unknown>(
      `/api/v1/apps${toQuery(pagedParams(params))}`,
      { signal },
    );
    return pageFrom<StoreApp>(payload, ["apps", "applications"]);
  },
  app: (slug: string, signal?: AbortSignal) =>
    apiFetch<StoreApp>(`/api/v1/apps/${encodeURIComponent(slug)}`, { signal }),
  categories: async (signal?: AbortSignal) =>
    listFrom<Category>(
      await apiFetch<unknown>("/api/v1/categories", { signal }),
      ["categories"],
    ),
  session: async (signal?: AbortSignal): Promise<Session> => {
    try {
      return await apiFetch<Session>("/api/v1/auth/session", { signal });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401)
        return { authenticated: false };
      throw error;
    }
  },
  bootstrapLogin: (username: string, password: string) =>
    apiFetch<Session>("/api/v1/auth/bootstrap/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  logout: () => apiFetch<void>("/api/v1/auth/logout", { method: "POST" }),
  me: (signal?: AbortSignal) => apiFetch<User>("/api/v1/me", { signal }),
  myApps: async (signal?: AbortSignal) =>
    listFrom<StoreApp>(await apiFetch<unknown>("/api/v1/me/apps", { signal }), [
      "apps",
    ]),
  myKeys: async (signal?: AbortSignal) =>
    listFrom<PersonalKey>(
      await apiFetch<unknown>("/api/v1/me/keys", { signal }),
      ["keys"],
    ),
  myActivity: async (signal?: AbortSignal) =>
    listFrom<Record<string, unknown>>(
      await apiFetch<unknown>("/api/v1/me/activity", { signal }),
      ["activity", "items"],
    ),
  mySettings: (signal?: AbortSignal) =>
    apiFetch<UserPreferences>("/api/v1/me/settings", { signal }),
  updateMySettings: (input: UserPreferences) =>
    apiFetch<UserPreferences>("/api/v1/me/settings", {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  keyPermissionOptions: (signal?: AbortSignal) =>
    apiFetch<KeyPermissionOptions>("/api/v1/me/key-permissions", { signal }),
  createKey: (input: unknown) =>
    apiFetch<PersonalKey>("/api/v1/me/keys", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  rotateKey: (id: string) =>
    apiFetch<PersonalKey>(`/api/v1/me/keys/${encodeURIComponent(id)}/rotate`, {
      method: "POST",
    }),
  revokeKey: (id: string) =>
    apiFetch<void>(`/api/v1/me/keys/${encodeURIComponent(id)}/revoke`, {
      method: "POST",
    }),
  createApp: (input: unknown) =>
    apiFetch<StoreApp>("/api/v1/apps", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateApp: (id: string, input: unknown) =>
    apiFetch<StoreApp>(`/api/v1/apps/${encodeURIComponent(id)}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  appSecurityCheck: (appId: string, signal?: AbortSignal) =>
    apiFetch<SecurityCheckView>(
      `/api/v1/apps/${encodeURIComponent(appId)}/security-check`,
      { signal },
    ),
  verifyAppSecurityCheck: (appId: string, reviewId: string) =>
    apiFetch<SecurityCheckView>(
      `/api/v1/apps/${encodeURIComponent(appId)}/security-check/verify`,
      { method: "POST", body: JSON.stringify({ reviewId }), timeoutMs: 60_000 },
    ),
  submitApp: (appId: string) =>
    apiFetch<StoreApp>(`/api/v1/apps/${encodeURIComponent(appId)}/submit`, {
      method: "POST",
      timeoutMs: 60_000,
    }),
  securityCheckSettings: (signal?: AbortSignal) =>
    apiFetch<SecurityCheckSettings>("/api/v1/admin/security-check", { signal }),
  updateSecurityCheckSettings: (input: {
    enabled: boolean;
    baseUrl: string;
    apiKey?: string;
    timeoutSeconds: number;
  }) =>
    apiFetch<SecurityCheckSettings>("/api/v1/admin/security-check", {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  testSecurityCheck: () =>
    apiFetch<SecurityCheckTestResult>("/api/v1/admin/security-check/test", {
      method: "POST",
      timeoutMs: 60_000,
    }),
  appDocuments: async (app: string, signal?: AbortSignal) =>
    listFrom<AppDocument>(
      await apiFetch<unknown>(
        `/api/v1/apps/${encodeURIComponent(app)}/documents`,
        { signal },
      ),
      ["documents"],
    ),
  uploadAppDocument: (app: string, file: File, title?: string) => {
    const body = new FormData();
    body.append("file", file);
    if (title?.trim()) body.append("title", title.trim());
    // A 20MB manual over a company VPN needs more than the default budget.
    return apiFetch<AppDocument>(
      `/api/v1/apps/${encodeURIComponent(app)}/documents`,
      { method: "POST", body, timeoutMs: 120_000 },
    );
  },
  deleteAppDocument: (app: string, id: string) =>
    apiFetch<void>(
      `/api/v1/apps/${encodeURIComponent(app)}/documents/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    ),
  reviews: async (signal?: AbortSignal) =>
    listFrom<Review>(await apiFetch<unknown>("/api/v1/reviews", { signal }), [
      "reviews",
    ]),
  review: (id: string, signal?: AbortSignal) =>
    apiFetch<ReviewDetail>(`/api/v1/reviews/${encodeURIComponent(id)}`, {
      signal,
    }),
  approveReview: (id: string, comment = "") =>
    apiFetch<Review>(`/api/v1/reviews/${encodeURIComponent(id)}/approve`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    }),
  rejectReview: (id: string, comment: string) =>
    apiFetch<Review>(`/api/v1/reviews/${encodeURIComponent(id)}/reject`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    }),
  admin: <T>(resource: string, signal?: AbortSignal) =>
    apiFetch<T>(`/api/v1/admin/${resource}`, { signal }),
  adminApps: async (
    params: Record<string, string | number | boolean | undefined>,
    signal?: AbortSignal,
  ) => {
    const payload = await apiFetch<unknown>(
      `/api/v1/admin/apps${toQuery(pagedParams(params))}`,
      { signal },
    );
    return pageFrom<StoreApp>(payload, ["apps", "applications"]);
  },
  uploadBranding: (kind: "logo" | "favicon", file: File) => {
    const body = new FormData();
    body.append("file", file);
    return apiFetch<BrandingAsset>(
      `/api/v1/admin/branding/${encodeURIComponent(kind)}`,
      { method: "POST", body, timeoutMs: 30_000 },
    );
  },
  importBranding: (kind: "logo" | "favicon", sourceUrl: string) =>
    apiFetch<BrandingAsset>(
      `/api/v1/admin/branding/${encodeURIComponent(kind)}`,
      {
        method: "POST",
        body: JSON.stringify({ sourceUrl }),
        timeoutMs: 30_000,
      },
    ),
  deleteBranding: (kind: "logo" | "favicon") =>
    apiFetch<void>(`/api/v1/admin/branding/${encodeURIComponent(kind)}`, {
      method: "DELETE",
    }),
  createAdminApp: (input: unknown) =>
    apiFetch<StoreApp>("/api/v1/admin/apps", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  adminApp: (id: string, signal?: AbortSignal) =>
    apiFetch<StoreApp>(`/api/v1/admin/apps/${encodeURIComponent(id)}`, {
      signal,
    }),
  updateAdminApp: (id: string, input: unknown) =>
    apiFetch<StoreApp>(`/api/v1/admin/apps/${encodeURIComponent(id)}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  deleteAdminApp: (id: string) =>
    apiFetch<void>(`/api/v1/admin/apps/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),
  testOidc: (input: { issuerUrl?: string; clientId?: string }) =>
    apiFetch<OidcTestResult>("/api/v1/admin/authentication/test", {
      method: "POST",
      body: JSON.stringify(input),
      timeoutMs: 25_000,
    }),
  updateAdmin: <T>(resource: string, input: unknown) =>
    apiFetch<T>(`/api/v1/admin/${resource}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  setAdminAppStatus: (id: string, status: string) =>
    apiFetch<StoreApp>(`/api/v1/admin/apps/${encodeURIComponent(id)}/status`, {
      method: "PUT",
      body: JSON.stringify({ status }),
    }),
  adminAction: <T>(resource: string, input?: unknown) =>
    apiFetch<T>(`/api/v1/admin/${resource}`, {
      method: "POST",
      body: input === undefined ? undefined : JSON.stringify(input),
    }),
  deleteAdmin: (resource: string) =>
    apiFetch<void>(`/api/v1/admin/${resource}`, { method: "DELETE" }),
  updateKeyPermission: (key: string, input: KeyPermissionDefinition) =>
    apiFetch<KeyPermissionDefinition>(
      `/api/v1/admin/security/permissions/${encodeURIComponent(key)}`,
      { method: "PUT", body: JSON.stringify(input) },
    ),
  createKeyTemplate: (input: Omit<KeyPermissionTemplate, "id">) =>
    apiFetch<KeyPermissionTemplate>("/api/v1/admin/security/templates", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateKeyTemplate: (id: string, input: Omit<KeyPermissionTemplate, "id">) =>
    apiFetch<KeyPermissionTemplate>(
      `/api/v1/admin/security/templates/${encodeURIComponent(id)}`,
      { method: "PUT", body: JSON.stringify(input) },
    ),
  deleteKeyTemplate: (id: string) =>
    apiFetch<void>(
      `/api/v1/admin/security/templates/${encodeURIComponent(id)}`,
      { method: "DELETE" },
    ),
  aiModels: async (providerId: string, signal?: AbortSignal) =>
    listFrom<AiModelLimit>(
      await apiFetch<unknown>(
        `/api/v1/admin/ai/models${toQuery({ providerId })}`,
        { signal },
      ),
      ["models"],
    ),
  upsertAiModel: (input: AiModelLimit) =>
    apiFetch<AiModelLimit>("/api/v1/admin/ai/models", {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  deleteAiModel: (id: string) =>
    apiFetch<void>(`/api/v1/admin/ai/models/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),
  adminUsers: async (
    params: Record<string, string | number | boolean | undefined>,
    signal?: AbortSignal,
  ) => {
    const payload = await apiFetch<unknown>(
      `/api/v1/admin/users${toQuery(pagedParams(params))}`,
      { signal },
    );
    return pageFrom<User>(payload, ["users"]);
  },
  audit: async (
    params: Record<string, string | number | boolean | undefined>,
    signal?: AbortSignal,
  ) => {
    const payload = await apiFetch<unknown>(
      `/api/v1/admin/audit${toQuery(pagedParams(params))}`,
      { signal },
    );
    return pageFrom<AuditEntry>(payload, ["logs", "auditLogs"]);
  },
};

export async function streamAiChat(
  input: unknown,
  onEvent: (event: AiStreamEvent) => void,
  signal: AbortSignal,
): Promise<void> {
  const response = await fetch("/api/v1/ai/chat/stream", {
    method: "POST",
    headers: {
      Accept: "text/event-stream",
      "Content-Type": "application/json",
      ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
    },
    credentials: "include",
    body: JSON.stringify(input),
    signal,
  });
  if (!response.ok) {
    const payload: unknown = await response.json().catch(() => undefined);
    throw new ApiError(response.status, problemFrom(payload, response.status));
  }
  if (!response.body) throw new Error("스트리밍 응답 본문이 없습니다.");

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, "\n");
    let boundary = buffer.indexOf("\n\n");
    while (boundary >= 0) {
      const block = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);
      emitSseBlock(block, onEvent);
      boundary = buffer.indexOf("\n\n");
    }
  }
  // The stream ended without a blank line after the last block. Flush the
  // decoder and deliver that block only if its payload is complete (valid
  // JSON or [DONE]); a fragment cut off mid-event is discarded, not shown.
  buffer += decoder.decode().replace(/\r\n/g, "\n");
  if (buffer.trim()) emitSseBlock(buffer, onEvent, { completeOnly: true });
}

function emitSseBlock(
  block: string,
  onEvent: (event: AiStreamEvent) => void,
  options: { completeOnly?: boolean } = {},
): void {
  let eventName: AiStreamEvent["event"] = "message";
  const dataLines: string[] = [];
  for (const line of block.split("\n")) {
    if (line.startsWith("event:"))
      eventName = line.slice(6).trim() as AiStreamEvent["event"];
    if (line.startsWith("data:")) dataLines.push(line.slice(5).trimStart());
  }
  const raw = dataLines.join("\n");
  if (raw && raw !== "[DONE]") {
    let data: unknown = raw;
    try {
      data = JSON.parse(raw);
    } catch {
      // Text chunks are valid inside a terminated block; an unterminated
      // tail that is not JSON may be a truncated event, so drop it.
      if (options.completeOnly) return;
    }
    onEvent({ event: eventName, data });
  }
  if (raw === "[DONE]") onEvent({ event: "finish", data: null });
}
