import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { History, ImagePlus, Save, ShieldCheck } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading } from "../ui";
import {
  leaveOperations,
  refreshPresentation,
  useOperationMounted,
  useOperationsGuard,
} from "./shared";
type Asset = {
  url: string;
  digest: string;
  width: number;
  height: number;
  size_bytes: number;
  created_at: string;
};
type Key = "site_name" | "theme_primary" | "logo_url" | "favicon_url";
export function BrandingPanel({ workspaceID }: { workspaceID: string }) {
  const { notify, workspace } = useApp(),
    mounted = useOperationMounted();
  const generation = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);
  const [snapshot, setSnapshot] = useState<{
      version: number;
      data: Partial<Record<Key, string>>;
    } | null>(null),
    [patch, setPatch] = useState<Partial<Record<Key, string | null>>>({}),
    [assets, setAssets] = useState<Asset[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [file, setFile] = useState<File | null>(null);
  const path = `/workspaces/${workspaceID}`,
    dirty = Object.keys(patch).length > 0;
  useOperationsGuard(dirty);
  const value = (key: Key) =>
    Object.hasOwn(patch, key) ? patch[key] || "" : snapshot?.data[key] || "";
  const change = (key: Key, v: string) =>
    setPatch((old) => ({ ...old, [key]: v || null }));
  useEffect(() => {
    const ticket = ++generation.current;
    Promise.all([
      api(path + "/settings"),
      api<Asset[]>(path + "/branding/assets"),
    ])
      .then(([settings, items]) => {
        if (mounted.current && ticket === generation.current) {
          setSnapshot(settings);
          setAssets(items);
        }
      })
      .catch((e) => {
        if (mounted.current && ticket === generation.current) setError(e);
      });
    return () => {
      generation.current++;
    };
  }, [path]);
  async function save() {
    if (!snapshot) return;
    setBusy(true);
    setError(null);
    try {
      const next = await api(path + "/settings", "PUT", {
        version: snapshot.version,
        data: patch,
      });
      if (!mounted.current) return;
      setSnapshot(next);
      setPatch({});
      refreshPresentation();
      notify("워크스페이스 브랜딩을 적용했습니다");
    } catch (e) {
      if (mounted.current) setError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  async function upload() {
    if (!file) return;
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("file", file);
      const result = await api<Asset>(path + "/branding/assets", "POST", form);
      if (!mounted.current) return;
      setAssets((old) => [
        result,
        ...old.filter((a) => a.digest !== result.digest),
      ]);
      change("logo_url", result.url);
      setFile(null);
      notify("안전한 PNG로 변환했습니다. 브랜딩을 저장하면 적용됩니다");
    } catch (e) {
      if (mounted.current) setError(e);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }
  return (
    <section>
      <div className="notice">
        <ShieldCheck size={21} />
        <span>
          이미지는 서비스 내부에 보관합니다. PNG/JPEG를 읽은 후 최대 512px PNG로
          다시 만들어 위치정보·메타데이터를 제거합니다. 외부 URL, SVG, 사용자
          CSS·스크립트는 허용하지 않습니다.
        </span>
      </div>
      <ErrorBox error={error} />
      {!snapshot ? (
        <Loading />
      ) : (
        <div className="operations-brand-grid">
          <aside className="panel padded">
            <div
              className="operations-brand-preview"
              style={{
                backgroundColor: /^#[a-f\d]{6}$/i.test(value("theme_primary"))
                  ? value("theme_primary")
                  : "#176d60",
              }}
            >
              <img
                src={value("logo_url") || "/favicon.svg"}
                alt="워크스페이스 로고 미리보기"
              />
              <strong>{value("site_name") || workspace?.name || "madi"}</strong>
              <span>함께 연결하는 지식</span>
            </div>
            <p className="muted">
              미리보기는 아직 저장되지 않은 값을 포함합니다. 색상은 흰색 글자와
              최소 4.5:1 대비를 만족해야 저장할 수 있습니다.
            </p>
            <Link
              className="button secondary"
              to="/app/workspace-settings"
              onClick={(e) => {
                if (!leaveOperations()) e.preventDefault();
              }}
            >
              <History size={16} />
              워크스페이스 설정 이력
            </Link>
          </aside>
          <form
            className="panel padded"
            onSubmit={(e) => {
              e.preventDefault();
              void save();
            }}
          >
            <h2>팀의 이름과 인상</h2>
            <Field
              label="팀 서비스 이름"
              hint="비우면 서비스 기본 이름을 상속합니다."
            >
              <input
                maxLength={100}
                value={value("site_name")}
                disabled={busy}
                onChange={(e) => change("site_name", e.target.value)}
                placeholder="서비스 기본 이름"
              />
            </Field>
            <Field
              label="대표 색상"
              hint="예: #176d60 · 비우면 기본 색상. 밝은 색은 가독성 검증에서 안내합니다."
            >
              <input
                value={value("theme_primary")}
                maxLength={7}
                pattern="#[a-fA-F0-9]{6}"
                disabled={busy}
                onChange={(e) => change("theme_primary", e.target.value)}
                placeholder="#176d60"
              />
            </Field>
            <div className="form-grid">
              {(["logo_url", "favicon_url"] as const).map((key) => (
                <Field
                  key={key}
                  label={key === "logo_url" ? "로고 이미지" : "파비콘 이미지"}
                >
                  <select
                    value={value(key)}
                    disabled={busy}
                    onChange={(e) => change(key, e.target.value)}
                  >
                    <option value="">서비스 기본 이미지</option>
                    <option value="/favicon.svg">madi 기본 아이콘</option>
                    {assets.map((a) => (
                      <option key={a.digest} value={a.url}>
                        {a.width}×{a.height} PNG · {a.digest.slice(0, 8)}
                      </option>
                    ))}
                  </select>
                </Field>
              ))}
            </div>
            <div className="operations-upload">
              <Field
                label="새 브랜딩 이미지"
                hint="1MB 이하 PNG/JPEG, 최대 2048×2048px · 워크스페이스당 40개 보관"
              >
                <div className="operations-file-picker">
                  <input
                    ref={fileInput}
                    className="sr-only"
                    key={file ? "selected" : "empty"}
                    type="file"
                    accept="image/png,image/jpeg"
                    disabled={busy}
                    onChange={(e) => setFile(e.target.files?.[0] || null)}
                  />
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={busy}
                    onClick={() => fileInput.current?.click()}
                  >
                    이미지 파일 선택
                  </Button>
                  <span>{file?.name || "선택한 파일 없음"}</span>
                </div>
              </Field>
              <Button
                type="button"
                variant="secondary"
                disabled={busy || !file}
                onClick={() => void upload()}
              >
                <ImagePlus size={16} />
                이미지 등록
              </Button>
            </div>
            {assets.length > 0 && (
              <div
                className="operations-asset-list"
                aria-label="등록된 브랜딩 이미지"
              >
                {assets.map((a) => (
                  <button
                    type="button"
                    key={a.digest}
                    disabled={busy}
                    onClick={() => change("logo_url", a.url)}
                    aria-label={`${a.digest.slice(0, 8)} 이미지를 로고로 선택`}
                    className={value("logo_url") === a.url ? "selected" : ""}
                  >
                    <img src={a.url} alt="" />
                    <small>
                      {a.width}×{a.height}
                    </small>
                    <span className="sr-only">
                      {a.created_at ? datetime(a.created_at) : "새 이미지"}
                    </span>
                  </button>
                ))}
              </div>
            )}
            <p className="muted">
              이미지 원본은 설정 이력 복원을 위해 보관합니다. 기본 이미지로
              변경해도 등록된 파일을 삭제하지 않습니다.
            </p>
            <footer className="operations-save">
              <span className="muted">
                {dirty
                  ? "저장 후 구성원 화면에 적용됩니다."
                  : "저장된 브랜딩입니다."}
              </span>
              <Button type="submit" disabled={busy || !dirty}>
                <Save size={17} />
                {busy ? "처리 중…" : "브랜딩 저장"}
              </Button>
            </footer>
          </form>
        </div>
      )}
    </section>
  );
}
