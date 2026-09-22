import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { FileSearch, RefreshCw } from "lucide-react";
import { api, bytes, datetime } from "../api";
import { useApp } from "../context";
import {
  Badge,
  Button,
  Empty,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import {
  type Context,
  type Fragment,
  type Run,
  positionName,
  statusName,
} from "./types";
import "./style.css";
const PdfPage = lazy(() => import("./PdfPage"));
const AttachmentAI = lazy(() => import("./AttachmentAI"));

export default function AttachmentPage() {
  const { id = "" } = useParams();
  const { user, workspace } = useApp();
  const [params, setParams] = useSearchParams();
  const scope = `${user.id}:${workspace?.id}:${id}`;
  const current = useRef(scope);
  current.current = scope;
  const generation = useRef(0);
  const [data, setData] = useState<Context | null>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [reload, setReload] = useState(0);
  const [fragments, setFragments] = useState<Fragment[]>([]),
    [selected, setSelected] = useState<Fragment | null>(null),
    [more, setMore] = useState(false),
    [next, setNext] = useState(0),
    [offset, setOffset] = useState(0);
  const [confirm, setConfirm] = useState<"extract" | "ocr" | null>(null),
    [ocr, setOCR] = useState<number[]>([]);
  const [page, setPage] = useState(1);
  const explicit = params.get("extraction"),
    fragmentID = params.get("fragment");
  const run = data?.runs.find(
    (r) =>
      r.id === (explicit || data?.runs.find((r) => r.status === "ready")?.id),
  );
  useEffect(() => {
    const controller = new AbortController();
    const g = ++generation.current;
    setData(null);
    setError(null);
    setFragments([]);
    setSelected(null);
    setConfirm(null);
    setOCR([]);
    setBusy(false);
    setOffset(0);
    const load = async () => {
      try {
        const value = await api<Context>(
          `/attachments/${id}/extraction-context`,
          "GET",
          undefined,
          { signal: controller.signal },
        );
        if (current.current !== scope || g !== generation.current) return;
        if (value.workspace_id !== workspace?.id) {
          setData(null);
          setError(
            new Error("첨부파일의 워크스페이스를 선택한 뒤 다시 여세요."),
          );
          return;
        }
        setData(value);
        setError(null);
      } catch (e) {
        if (
          !controller.signal.aborted &&
          current.current === scope &&
          g === generation.current
        ) {
          setData(null);
          setFragments([]);
          setSelected(null);
          setError(e);
        }
      }
    };
    void load();
    const timer = setInterval(() => void load(), 3000);
    return () => {
      controller.abort();
      clearInterval(timer);
      generation.current++;
    };
  }, [scope, reload]);
  useEffect(() => {
    setOffset(0);
    setFragments([]);
    setSelected(null);
  }, [explicit, scope]);
  useEffect(() => {
    if (!run || run.status !== "ready") {
      setFragments([]);
      setSelected(null);
      return;
    }
    const controller = new AbortController();
    setError(null);
    setFragments([]);
    setSelected(null);
    void api<{ fragments: Fragment[]; has_more: boolean; next_offset: number }>(
      `/attachment-extractions/${run.id}/fragments?offset=${offset}`,
      "GET",
      undefined,
      { signal: controller.signal },
    )
      .then(async (value) => {
        if (controller.signal.aborted || current.current !== scope) return;
        setFragments(value.fragments);
        setMore(value.has_more);
        setNext(value.next_offset);
        let f = value.fragments.find((f) => f.id === fragmentID) || null;
        if (fragmentID && !f) {
          const detail = await api<{ fragment: Fragment }>(
            `/attachment-extractions/${run.id}/fragments/${fragmentID}`,
            "GET",
            undefined,
            { signal: controller.signal },
          );
          f = detail.fragment;
        }
        if (controller.signal.aborted || current.current !== scope) return;
        setSelected(f);
        if (f?.position.page) setPage(f.position.page);
      })
      .catch((e) => {
        if (!controller.signal.aborted && current.current === scope) {
          setError(e);
          setFragments([]);
          setSelected(null);
        }
      });
    return () => controller.abort();
  }, [scope, run?.id, run?.status, run?.revision, fragmentID, offset]);
  const request = async () => {
    if (!data || !confirm || busy) return;
    const captured = scope,
      mode = confirm;
    setBusy(true);
    setError(null);
    try {
      const value = await api<{ id: string }>(
        `/attachments/${id}/extractions`,
        "POST",
        {
          document_version: data.document_version,
          checksum: data.checksum,
          ocr_pages: mode === "ocr" ? ocr : [],
          confirmation: mode === "ocr" ? "OCR" : "",
        },
      );
      if (current.current !== captured) return;
      setConfirm(null);
      setParams({ extraction: value.id });
      setReload((v) => v + 1);
    } catch (e) {
      if (current.current === captured) setError(e);
    } finally {
      if (current.current === captured) setBusy(false);
    }
  };
  const cancel = async (v: Run) => {
    if (busy) return;
    const captured = scope;
    setBusy(true);
    try {
      await api(`/attachment-extractions/${v.id}`, "DELETE");
      if (current.current === captured) setReload((v) => v + 1);
    } catch (e) {
      if (current.current === captured) setError(e);
    } finally {
      if (current.current === captured) setBusy(false);
    }
  };
  const choose = (f: Fragment) => {
    const p = new URLSearchParams(params);
    p.set("extraction", f.extraction_id);
    p.set("fragment", f.id);
    setParams(p);
  };
  const reviewing = !!confirm;
  if (!data)
    return (
      <main className="page extract-page">
        <PageHeading title="첨부 본문·위치" />
        <RecoveryNotice
          error={error}
          onRetry={() => setReload((v) => v + 1)}
          onReview={() => setReload((v) => v + 1)}
        />
        {!error && <Loading />}
      </main>
    );
  const pending = data.runs.some(
    (r) => r.status === "queued" || r.status === "running",
  );
  return (
    <main className="page extract-page">
      <PageHeading
        eyebrow="첨부 지식"
        title={data.name}
        description={`${data.document_title} · ${bytes(data.size)} · 원본 첨부는 변경하지 않습니다`}
        actions={
          <>
            <Link className="button" to={`/app/documents/${data.document_id}`}>
              부모 문서
            </Link>
            <a
              className="button"
              href={`/api/v1/attachments/${id}`}
              download={data.name}
            >
              원본 다운로드
            </a>
            <Button
              aria-label="첨부 새로고침"
              onClick={() => setReload((v) => v + 1)}
            >
              <RefreshCw size={18} />
            </Button>
          </>
        }
      />
      <RecoveryNotice
        error={error}
        busy={busy}
        onRetry={() => setReload((v) => v + 1)}
        onReview={() => {
          setConfirm(null);
          setReload((v) => v + 1);
        }}
      />
      <section className="card extract-actions">
        <div>
          <h2>
            <FileSearch size={20} /> 검색 가능한 본문
          </h2>
          <p>
            로컬 PDF·Office·UTF-8 텍스트를 추출합니다. 파일의
            지시문·매크로·수식은 실행하지 않으며 AI 제공자에게 자동 전송하지
            않습니다.
          </p>
        </div>
        <Button
          variant="primary"
          disabled={
            busy ||
            pending ||
            !data.can_write ||
            !data.policy.data.enabled ||
            !data.format
          }
          onClick={() => setConfirm("extract")}
        >
          {pending ? "추출 작업 진행 중" : "본문 추출 검토"}
        </Button>
        {!data.policy.data.enabled && (
          <p className="notice">
            관리자가 첨부 추출을 켠 뒤 사용할 수 있습니다.
          </p>
        )}
        {!data.format && (
          <p className="notice">
            이 형식은 지원하지 않습니다. PDF, DOCX, PPTX, XLSX, TXT, Markdown,
            CSV를 사용하세요.
          </p>
        )}
      </section>
      {run?.result?.warnings?.map((warning, i) => (
        <p className="notice" key={i}>
          {warning}
        </p>
      ))}
      {run &&
        run.status === "ready" &&
        data.format === "pdf" &&
        !!run.result.empty_pages?.length && (
          <section className="card">
            <h2>텍스트 없는 쪽</h2>
            <p>
              {run.result.empty_pages.join(", ")}쪽 · 스캔 이미지 여부를
              원본에서 확인한 뒤 필요한 쪽만 OCR로 추가 추출합니다.
            </p>
            <Button
              disabled={
                !data.can_write ||
                busy ||
                pending ||
                !data.policy.data.enabled ||
                !data.policy.data.ocr_enabled
              }
              onClick={() => {
                setOCR([]);
                setConfirm("ocr");
              }}
            >
              OCR 쪽 선택
            </Button>
          </section>
        )}
      <div className="extract-layout">
        <section className="card extract-body">
          <h2>추출 근거와 위치</h2>
          {run?.status === "ready" ? (
            <>
              <div className="extract-fragments">
                {fragments.map((f) => (
                  <button
                    key={f.id}
                    className={`extract-fragment ${selected?.id === f.id ? "selected" : ""}`}
                    onClick={() => choose(f)}
                  >
                    <strong>{positionName(f.position)}</strong>
                    <span>{f.text}</span>
                  </button>
                ))}
              </div>
              {!fragments.length && !error && (
                <Empty
                  title="추출된 텍스트가 없습니다"
                  text="PDF의 텍스트 없는 쪽은 선택 OCR을 검토하세요. 이미지·도형만 있는 Office 내용은 텍스트로 복원되지 않습니다."
                />
              )}
              <div className="extract-paging">
                <Button
                  disabled={!offset}
                  onClick={() => setOffset(Math.max(0, offset - 100))}
                >
                  이전 근거
                </Button>
                <span>
                  {offset + 1}–{offset + fragments.length}
                </span>
                <Button disabled={!more} onClick={() => setOffset(next)}>
                  다음 근거
                </Button>
              </div>
            </>
          ) : (
            <Empty
              title="아직 추출 본문이 없습니다"
              text="작업을 요청하면 대기 상태가 표시됩니다. 완료 전에는 본문 일부가 게시되지 않습니다."
            />
          )}
        </section>
        <section className="card extract-original">
          <h2>원본 위치</h2>
          {data.format === "pdf" ? (
            <>
              <Field label="PDF 쪽">
                <input
                  type="number"
                  min={1}
                  max={Math.max(
                    1,
                    Number(run?.result.pages || data.policy.data.max_pages),
                  )}
                  value={page}
                  onChange={(e) =>
                    setPage(
                      Math.max(
                        1,
                        Math.min(
                          Number(
                            run?.result.pages || data.policy.data.max_pages,
                          ),
                          Number(e.target.value) || 1,
                        ),
                      ),
                    )
                  }
                />
              </Field>
              <Suspense fallback={<Loading />}>
                <PdfPage
                  key={`${scope}:${data.checksum}`}
                  attachmentID={id}
                  page={page}
                  position={selected?.position}
                />
              </Suspense>
            </>
          ) : selected ? (
            <>
              <Badge>{positionName(selected.position)}</Badge>
              <p className="muted">
                {data.format === "text" ? "UTF-8 원문의 문단 위치입니다. 정보보호 정책이 적용된 결과는 원본 파일과 표시가 다를 수 있습니다." : "Office의 서식·도형을 재현한 화면이 아니라, 원본 내 위치가 연결된 텍스트입니다. 원본 파일에서 해당 위치를 대조하세요."}
              </p>
              <pre className="extract-text">{selected.text}</pre>
            </>
          ) : (
            <p>
              왼쪽 근거를 선택하면 슬라이드·시트·셀·문단 위치를 확인할 수
              있습니다.
            </p>
          )}
          {selected && (
            <p className="extract-checksum">
              근거 #{selected.ordinal + 1} · {positionName(selected.position)}
              <br />
              SHA-256 {selected.content_hash}
            </p>
          )}
          {selected && run && (
            <Suspense fallback={<Loading />}>
              <AttachmentAI
                key={`${scope}:${selected.id}`}
                context={data}
                fragment={selected}
                run={run}
              />
            </Suspense>
          )}
          {selected && run && (
            <Link
              className="button"
              to={`/app/knowledge-packages?extraction=${run.id}&fragment=${selected.id}`}
            >
              이 구간을 AI 컨텍스트 패키지에서 검토
            </Link>
          )}
        </section>
      </div>
      <section className="card">
        <h2>추출 이력</h2>
        <p>
          최근 50건. 정책·원본이 바뀐 이전 결과는 검색·인용에 사용하지 않습니다.
        </p>
        <div className="extract-history">
          {data.runs.map((v) => (
            <article key={v.id}>
              <div>
                <Badge>{statusName[v.status] || v.status}</Badge>{" "}
                <span>{datetime(v.created_at)}</span>
                {v.error && <p className="notice error">{v.error}</p>}
              </div>
              <div className="extract-history-actions">
                <Button
                  onClick={() => {
                    setParams({ extraction: v.id });
                    setOffset(0);
                  }}
                >
                  결과 보기
                </Button>
                {v.actor_id === user.id &&
                  (v.status === "queued" || v.status === "running") && (
                    <Button disabled={busy} onClick={() => void cancel(v)}>
                      작업 취소
                    </Button>
                  )}
              </div>
            </article>
          ))}
        </div>
      </section>
      <Modal
        open={reviewing}
        onOpenChange={(v) => {
          if (!v && !busy) setConfirm(null);
        }}
        title={confirm === "ocr" ? "선택한 쪽 OCR 검토" : "첨부 본문 추출 검토"}
        wide
      >
        <ChangeReview
          title="원본은 그대로, 검색용 본문을 별도로 생성"
          changes={[
            {
              label: "전송 범위",
              before: "원본 첨부",
              after: "현재 서버의 격리된 로컬 추출기만 사용",
            },
            {
              label: "검색·인용",
              before: "첨부 이름 검색",
              after: "현재 문서 권한을 따르는 본문·위치 검색",
            },
            {
              label: "원본 확인",
              before: `부모 문서 v${data.document_version}`,
              after: `SHA-256 ${data.checksum || "확정 전 안전한 체크섬 계산"}`,
            },
          ]}
          warnings={[
            "자동 추출과 OCR은 오인식·누락이 있습니다. 표·열·병합 셀·수식 표시가 원본과 다를 수 있습니다.",
            "새 작업은 기존 활성 추출 결과를 원자적으로 교체합니다. 완료 전 원본·정책·권한이 바뀌면 결과를 게시하지 않습니다.",
            "엄격한 개인정보 정책은 추출 본문·시트 이름에 적용되지만 원본 파일의 과거 민감정보까지 지우지는 않습니다.",
          ]}
          confirmLabel={
            confirm === "ocr" ? "선택 쪽 OCR 요청" : "본문 추출 요청"
          }
          busy={busy}
          disabled={confirm === "ocr" && !ocr.length}
          onConfirm={() => void request()}
          onCancel={() => setConfirm(null)}
        >
          {confirm === "ocr" && (
            <fieldset disabled={busy}>
              <legend>OCR할 텍스트 없는 쪽 · 최대 20쪽</legend>
              <div className="extract-page-options">
                {run?.result.empty_pages?.map((n) => (
                  <label key={n}>
                    <input
                      type="checkbox"
                      checked={ocr.includes(n)}
                      disabled={!ocr.includes(n) && ocr.length >= 20}
                      onChange={(e) =>
                        setOCR((v) =>
                          e.target.checked
                            ? [...v, n]
                            : v.filter((p) => p !== n),
                        )
                      }
                    />
                    {n}쪽
                  </label>
                ))}
              </div>
              <p>
                영어·한국어 모델을 사용합니다. 텍스트가 이미 있는 쪽에는 OCR을
                실행하지 않습니다.
              </p>
            </fieldset>
          )}
        </ChangeReview>
      </Modal>
    </main>
  );
}
