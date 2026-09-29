let csrfToken = "";
export const setCSRF = (value: string) => {
  csrfToken = value;
};

/**
 * 세션이 끊겼을 때 알림을 받을 곳. 세션은 기본 8시간이라 화면을 열어 둔 채
 * 만료되는 일이 흔한데, 그때 각 화면은 "요청 실패 (401)"만 띄우고 사용자는
 * 다시 로그인해야 한다는 사실을 알 수 없었다.
 */
type SessionEndedHandler = (reason: string) => void;
let onSessionEnded: SessionEndedHandler | null = null;
export const setSessionEndedHandler = (handler: SessionEndedHandler | null) => {
  onSessionEnded = handler;
};

// 서버가 세션 없음/만료를 구분해 내려주는 코드. 로그인 실패(invalid_credentials)나
// 비밀번호 확인 실패는 세션 문제가 아니므로 여기에 넣지 않는다.
const SESSION_ENDED_CODES = new Set([
  "session_expired",
  "authentication_required",
]);

type APIErrorShape = { error?: { code?: string; message?: string } };
export class APIError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData))
    headers.set("Content-Type", "application/json");
  if (csrfToken && !["GET", "HEAD"].includes(init.method ?? "GET"))
    headers.set("X-CSRF-Token", csrfToken);
  const response = await fetch(path, {
    ...init,
    headers,
    credentials: "same-origin",
  });
  if (!response.ok) {
    let data: APIErrorShape = {};
    try {
      data = await response.json();
    } catch {
      /* empty */
    }
    const code = data.error?.code ?? "request_failed";
    const message = data.error?.message ?? `요청 실패 (${response.status})`;
    if (response.status === 401 && SESSION_ENDED_CODES.has(code))
      onSessionEnded?.(message);
    throw new APIError(response.status, code, message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export function postJSON<T>(path: string, body: unknown) {
  return api<T>(path, { method: "POST", body: JSON.stringify(body) });
}
export function putJSON<T>(path: string, body: unknown) {
  return api<T>(path, { method: "PUT", body: JSON.stringify(body) });
}
export function patchJSON<T>(path: string, body: unknown) {
  return api<T>(path, { method: "PATCH", body: JSON.stringify(body) });
}
