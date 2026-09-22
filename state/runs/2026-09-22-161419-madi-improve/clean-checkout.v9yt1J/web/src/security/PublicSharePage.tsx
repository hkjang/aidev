import { useEffect, useRef, useState } from "react";
import { useLocation, useParams } from "react-router-dom";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Download, LockKeyhole } from "lucide-react";
import { Button, ErrorBox, Field, Loading } from "../ui";
import { DocumentWatermark } from "./DocumentProtection";
import "./style.css";
type Row = Record<string, any>;
export default function PublicSharePage() {
  const { id } = useParams();
  const token = useLocation().hash.slice(1),
    identity = `${id}#${token}`;
  const identityRef = useRef(identity);
  identityRef.current = identity;
  const generation = useRef(0);
  const grant = useRef("");
  const [rawData, setData] = useState<Row | null>(null),
    [loadedIdentity, setLoadedIdentity] = useState(""),
    [error, setError] = useState(""),
    [passwordRequired, setPasswordRequired] = useState(false),
    [password, setPassword] = useState(""),
    [busy, setBusy] = useState(false),
    [loading, setLoading] = useState(true);
  const data = loadedIdentity === identity ? rawData : null;
  async function request(suffix = "", body?: Row) {
    const response = await fetch(`/api/v1/public-shares/${id}${suffix}`, {
      method: body ? "POST" : "GET",
      headers: {
        "X-Madi-Share-Token": token,
        "X-Madi-Share-Access": grant.current,
        ...(body ? { "Content-Type": "application/json" } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
      cache: "no-store",
      credentials: "omit",
    });
    if (identityRef.current !== identity)
      throw new DOMException("공유 대상 변경", "AbortError");
    if (!response.ok) {
      const problem = await response.json();
      if (problem.code === "share_password_required") {
        setPasswordRequired(true);
        setData(null);
        return null;
      }
      throw new Error(problem.error || "공유 문서 요청에 실패했습니다");
    }
    return response;
  }
  async function load() {
    const run = ++generation.current;
    setError("");
    try {
      const response = await request();
      if (response) {
        const value = await response.json();
        if (run === generation.current && identityRef.current === identity) {
          setData(value);
          setLoadedIdentity(identity);
          setPasswordRequired(false);
        }
      }
    } catch (e) {
      if (run === generation.current && identityRef.current === identity) {
        setData(null);
        setError((e as Error).message);
      }
    } finally {
      if (run === generation.current && identityRef.current === identity)
        setLoading(false);
    }
  }
  useEffect(() => {
    grant.current = "";
    setData(null);
    setLoading(true);
    setPasswordRequired(false);
    setPassword("");
    void load();
    const timer = setInterval(() => void load(), 15000);
    return () => {
      generation.current++;
      clearInterval(timer);
    };
  }, [id, token]);
  return (
    <main className="public-share-page">
      <header className="public-share-brand">
        <img src="/favicon.svg" alt="" />
        <strong>madi · 제한된 공개 공유</strong>
      </header>
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : passwordRequired ? (
        <section className="public-share-unlock">
          <h1>
            <LockKeyhole size={28} /> 공유 암호 확인
          </h1>
          <p>문서 소유자가 전달한 암호를 입력하세요.</p>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                const response = await request("/unlock", { password });
                if (response) {
                  grant.current = (await response.json()).access_token;
                  setPassword("");
                  await load();
                }
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Field label="공유 암호">
              <input
                type="password"
                autoComplete="off"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </Field>
            <Button type="submit" variant="primary" disabled={busy}>
              공유 문서 열기
            </Button>
          </form>
        </section>
      ) : data ? (
        <>
          <h1>{data.title}</h1>
          <p className="muted">
            공유 만료: {new Date(data.expires_at).toLocaleString("ko-KR")}
          </p>
          <article
            style={{ userSelect: data.allow_copy ? "text" : "none" }}
            onCopy={(e) => {
              if (!data.allow_copy) e.preventDefault();
            }}
            onContextMenu={(e) => {
              if (!data.allow_copy) e.preventDefault();
            }}
          >
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              skipHtml
              components={{
                img: ({ alt }) => (
                  <span>[이미지: {alt || "첨부 목록을 확인하세요"}]</span>
                ),
                a: ({ href, children }) =>
                  href && /^https?:\/\//i.test(href) ? (
                    <a
                      href={href}
                      target="_blank"
                      rel="noopener noreferrer nofollow"
                    >
                      {children}
                    </a>
                  ) : (
                    <span>{children}</span>
                  ),
              }}
            >
              {data.markdown}
            </ReactMarkdown>
          </article>
          {data.watermark && (
            <DocumentWatermark
              viewer="공유 방문자"
              classification={data.classification}
              at={data.viewed_at}
            />
          )}
          <section className="public-share-controls">
            <h2>공유 첨부</h2>
            {data.allow_download ? (
              data.attachments.map((file: Row) => (
                <Button
                  key={file.id}
                  disabled={busy}
                  onClick={async () => {
                    setBusy(true);
                    try {
                      const response = await request(`/attachments/${file.id}`);
                      if (response) {
                        const url = URL.createObjectURL(await response.blob()),
                          a = document.createElement("a");
                        a.href = url;
                        a.download = file.name;
                        a.click();
                        setTimeout(() => URL.revokeObjectURL(url), 1000);
                      }
                    } catch (e) {
                      setError((e as Error).message);
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  <Download size={16} />
                  {file.name}
                </Button>
              ))
            ) : (
              <p>문서 소유자가 첨부 다운로드를 허용하지 않았습니다.</p>
            )}
          </section>
          <footer className="public-share-footer">
            이 문서는 링크를 가진 방문자에게 명시적으로 공유되었습니다. 자동
            외부 이미지 로딩과 검색엔진 색인은 사용하지 않습니다.{" "}
            {!data.allow_copy &&
              "복사 제한은 편의적 억제로 화면 캡처나 이미 전달된 자료의 보관을 막지는 못합니다."}
          </footer>
        </>
      ) : null}
    </main>
  );
}
