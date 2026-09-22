import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import { ChangeReview } from "../review/ChangeReview";
import "./evidence.css";

type Period = { version: number; from: string; until: string };
type Registration = {
  protection_revision: number;
  document_id: string;
  document_version: number;
  revision: number;
  periods: Period[];
  notice: string;
  can_write: boolean;
  history: {
    revision: number;
    recorded_at: string;
    reason: string;
    periods: Period[];
  }[];
};
type Hit = {
  document_id: string;
  title: string;
  version: number;
  current_version: number;
  validity_revision: number;
  valid_from: string;
  valid_until: string | null;
  recorded_at: string;
  excerpt: string;
};
type SearchResult = {
  protection_revision: number;
  results: Hit[];
  next_after: string;
  has_more: boolean;
  date: string;
  notice: string;
  outcome: string;
  interpretation: { matched_terms: string[]; dictionary_revision: number };
};
type Historical = Hit & {
  protection_revision: number;
  markdown: string;
  content_hash: string;
  notice: string;
  date: string;
};
const localDay = () => {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
};
const periodText = (periods: Period[]) =>
  periods.length
    ? periods
        .map(
          (p) =>
            `v${p.version}: ${p.from}부터 ${p.until ? `${p.until} 전날까지` : "종료일 없이"}`,
        )
        .join("\n")
    : "명시적으로 등록된 유효기간 없음";

export default function KnowledgeTimePage() {
  const { user, workspace, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const selected = params.get("document_id") || "",
    date = params.get("date") || localDay(),
    q = params.get("q") || "",
    after = params.get("after") || "";
  const [inputDate, setInputDate] = useState(date),
    [inputQuery, setInputQuery] = useState(q);
  const [data, setData] = useState<SearchResult | null>(null),
    [record, setRecord] = useState<Registration | null>(null),
    [periods, setPeriods] = useState<Period[]>([]),
    [reason, setReason] = useState(""),
    [confirm, setConfirm] = useState(false);
  const [preview, setPreview] = useState<Historical | null>(null),
    [versions, setVersions] = useState<{ version: number; title: string }[]>(
      [],
    ),
    [older, setOlder] = useState(false);
  const [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const generation = useRef(0),
    previewGeneration = useRef(0),
    dirty =
      !!record &&
      (JSON.stringify(periods) !== JSON.stringify(record.periods) ||
        !!reason.trim());
  const dirtyRef = useRef(dirty);
  const previewOpener = useRef<HTMLElement | null>(null);
  dirtyRef.current = dirty;
  useEffect(() => {
    const leave = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", leave);
    return () => window.removeEventListener("beforeunload", leave);
  }, []);
  const changeParams = (next: Record<string, string>) => {
    if (
      dirty &&
      !window.confirm("등록하지 않은 유효기간 변경을 버리고 이동할까요?")
    )
      return;
    setParams(next);
  };
  useEffect(() => {
    let active = true;
    const request = ++generation.current;
    previewGeneration.current++;
    setPreview(null);
    setConfirm(false);
    setRecord(null);
    setPeriods([]);
    setReason("");
    setVersions([]);
    setOlder(false);
    setData(null);
    setError("");
    setLoading(true);
    setBusy(false);
    setInputDate(date);
    setInputQuery(q);
    const load = async () => {
      try {
        if (!workspace) return;
        const result = await api<SearchResult>(
          `/knowledge/time-search?${new URLSearchParams({ workspace_id: workspace.id, date, q, after })}`,
        );
        if (!active || request !== generation.current) return;
        setData(result);
        if (selected) {
          const [registered, vs] = await Promise.all([
            api<Registration>(`/documents/${selected}/validity`),
            api<{ version: number; title: string }[]>(
              `/documents/${selected}/versions?limit=50`,
            ),
          ]);
          if (active && request === generation.current) {
            setRecord(registered);
            setPeriods(registered.periods);
            setVersions(vs);
            setOlder(vs.length === 50);
          }
        }
      } catch (e) {
        if (active && request === generation.current) {
          setData(null);
          setRecord(null);
          setPeriods([]);
          setError((e as Error).message);
        }
      } finally {
        if (active && request === generation.current) setLoading(false);
      }
    };
    void load();
    return () => {
      active = false;
      generation.current++;
      previewGeneration.current++;
    };
  }, [user.id, workspace?.id, selected, date, q, after, refresh]);
  // No historical text or registration reasons are persisted in browser storage.
  // Read current ACL/revisions while results are visible; failure clears copies.
  useEffect(() => {
    if (!data && !record && !preview) return;
    let active = true,
      pending = false;
    const check = async () => {
      if (pending) return;
      pending = true;
      try {
        const expected = new Map<
          string,
          { revision: number; version: number }
        >();
        for (const h of data?.results || [])
          expected.set(h.document_id, {
            revision: h.validity_revision,
            version: h.current_version,
          });
        if (record)
          expected.set(record.document_id, {
            revision: record.revision,
            version: record.document_version,
          });
        if (preview)
          expected.set(preview.document_id, {
            revision: preview.validity_revision,
            version: preview.current_version,
          });
        const checked = await api<{ valid: boolean }>(
          "/knowledge/time-check",
          "POST",
          {
            workspace_id: workspace?.id,
            protection_revision:
              data?.protection_revision ??
              record?.protection_revision ??
              preview?.protection_revision,
            documents: [...expected].map(([id, value]) => ({ id, ...value })),
          },
        );
        if (!checked.valid)
          throw Error(
            "현재 권한·문서·유효기간 또는 보호 정책이 변경되었습니다. 다시 검색하세요.",
          );
      } catch (e) {
        if (active) {
          setData(null);
          setRecord(null);
          setPeriods([]);
          setReason("");
          setPreview(null);
          setConfirm(false);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
      }
    };
    const timer = setInterval(() => void check(), 3000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [data, record, preview]);
  const perform = async (action: () => Promise<void>) => {
    if (busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  const save = () =>
    perform(async () => {
      if (!record) return;
      const request = generation.current;
      await api(`/documents/${record.document_id}/validity`, "PUT", {
        revision: record.revision,
        document_version: record.document_version,
        periods,
        reason,
        consent: true,
      });
      if (request === generation.current) {
        setConfirm(false);
        setReason("");
        setRefresh((v) => v + 1);
        notify("업무 유효기간과 변경 이력을 함께 등록했습니다");
      }
    });
  const open = async (hit: Hit) => {
    previewOpener.current =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    const request = ++previewGeneration.current,
      scope = generation.current;
    setPreview(null);
    setError("");
    try {
      const next = await api<Historical>(
        `/documents/${hit.document_id}/valid-at?${new URLSearchParams({ date, revision: String(hit.validity_revision) })}`,
      );
      if (request === previewGeneration.current && scope === generation.current)
        setPreview(next);
    } catch (e) {
      if (request === previewGeneration.current && scope === generation.current)
        setError((e as Error).message);
    }
  };
  const loadOlder = () =>
    perform(async () => {
      const request = generation.current;
      const next = await api<{ version: number; title: string }[]>(
        `/documents/${selected}/versions?limit=50&before=${versions.at(-1)?.version || 2147483647}`,
      );
      if (request === generation.current) {
        setVersions((v) => [...v, ...next]);
        setOlder(next.length === 50);
      }
    });
  return (
    <div className="page evidence-page">
      <PageHeading
        title="시점 기준 지식"
        description="수정 날짜가 아닌, 명시된 업무 유효기간으로 당시 문서 버전을 찾습니다."
      />
      <ErrorBox error={error} />
      <section className="card">
        <h2>업무 기준일로 찾기</h2>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            changeParams({
              date: inputDate,
              q: inputQuery,
              ...(selected ? { document_id: selected } : {}),
            });
          }}
        >
          <div className="evidence-compare">
            <Field label="업무 기준일">
              <input
                type="date"
                value={inputDate}
                required
                onChange={(e) => setInputDate(e.target.value)}
              />
            </Field>
            <Field label="당시 원문 검색어">
              <input
                value={inputQuery}
                onChange={(e) => setInputQuery(e.target.value)}
                placeholder="예: K8s 장애 대응"
              />
            </Field>
          </div>
          <Button type="submit" variant="primary" disabled={busy}>
            기준일 검색
          </Button>
        </form>
        <p role="note">
          미등록 날짜는 ‘유효’로 추정하지 않습니다. 오늘 알고 있는 등록 내용과
          현재 용어 사전을 기준으로 검색하며, 그 당시의 접근 권한을 복원하지
          않습니다.
        </p>
        <Button
          disabled={busy}
          onClick={() => {
            if (
              !dirty ||
              window.confirm(
                "등록하지 않은 변경을 버리고 서버 내용을 다시 읽을까요?",
              )
            )
              setRefresh((v) => v + 1);
          }}
        >
          서버 내용 다시 읽기
        </Button>
      </section>
      {loading ? (
        <Loading />
      ) : (
        data && (
          <section className="card">
            <h2>{data.date}의 등록 문서</h2>
            <p>{data.notice}</p>
            {!!data.interpretation.matched_terms.length && (
              <p>
                적용한 조직 용어:{" "}
                {data.interpretation.matched_terms.join(" · ")}
              </p>
            )}
            {!data.results.length ? (
              <Empty
                title="이 조건으로 등록된 문서가 없습니다"
                text="검색어와 기준일을 확인하세요. 미등록 문서나 접근할 수 없는 문서의 유무는 안내하지 않습니다."
              />
            ) : (
              <div className="evidence-list">
                {data.results.map((h) => (
                  <article className="card" key={h.document_id}>
                    <h3>{h.title}</h3>
                    <p>
                      기준일 버전 v{h.version} · 현재 v{h.current_version}
                    </p>
                    <p>
                      {h.valid_from}부터{" "}
                      {h.valid_until
                        ? `${h.valid_until} 전날까지`
                        : "종료일 없이"}{" "}
                      · 원문 기록 시각 {datetime(h.recorded_at)}
                    </p>
                    <p className="evidence-text">{h.excerpt}</p>
                    <div className="button-row">
                      <Button onClick={() => void open(h)}>
                        당시 원문 미리보기
                      </Button>
                      <Link to={`/app/documents/${h.document_id}`}>
                        현재 문서 열기
                      </Link>
                      <Button
                        onClick={() =>
                          changeParams({ date, q, document_id: h.document_id })
                        }
                      >
                        유효기간·등록 이력
                      </Button>
                    </div>
                  </article>
                ))}
              </div>
            )}
            {data.has_more && (
              <Button
                onClick={() =>
                  changeParams({
                    date,
                    q,
                    after: data.next_after,
                    ...(selected ? { document_id: selected } : {}),
                  })
                }
              >
                다음 문서
              </Button>
            )}
            {after && (
              <Button
                onClick={() =>
                  changeParams({
                    date,
                    q,
                    ...(selected ? { document_id: selected } : {}),
                  })
                }
              >
                첫 결과로
              </Button>
            )}
          </section>
        )
      )}
      <section className="card">
        <h2>문서별 유효기간 등록</h2>
        <Field label="유효기간을 확인할 문서">
          <select
            value={selected}
            onChange={(e) =>
              changeParams({
                date,
                q,
                ...(e.target.value ? { document_id: e.target.value } : {}),
              })
            }
          >
            <option value="">문서를 선택하세요</option>
            {documents.map((d) => (
              <option key={d.id} value={d.id}>
                {d.title}
              </option>
            ))}
          </select>
        </Field>
        {record && (
          <>
            <p>{record.notice}</p>
            <p>
              현재 문서 v{record.document_version} · 등록 revision{" "}
              {record.revision}
            </p>
            {periods.map((p, i) => (
              <div className="card validity-period" key={i}>
                <Field label={`기간 ${i + 1} 원문 버전`}>
                  <select
                    disabled={!record.can_write}
                    value={p.version}
                    onChange={(e) =>
                      setPeriods((v) =>
                        v.map((p, n) =>
                          n === i
                            ? { ...p, version: Number(e.target.value) }
                            : p,
                        ),
                      )
                    }
                  >
                    {![...versions].some((v) => v.version === p.version) && (
                      <option value={p.version}>
                        v{p.version} (현재 등록)
                      </option>
                    )}
                    {versions.map((v) => (
                      <option key={v.version} value={v.version}>
                        v{v.version} · {v.title}
                      </option>
                    ))}
                  </select>
                </Field>
                <div className="evidence-compare">
                  <Field label={`기간 ${i + 1} 시작일 (포함)`}>
                    <input
                      type="date"
                      disabled={!record.can_write}
                      required
                      value={p.from}
                      onChange={(e) =>
                        setPeriods((v) =>
                          v.map((p, n) =>
                            n === i ? { ...p, from: e.target.value } : p,
                          ),
                        )
                      }
                    />
                  </Field>
                  <Field label={`기간 ${i + 1} 종료일 (제외)`}>
                    <input
                      type="date"
                      disabled={!record.can_write}
                      value={p.until}
                      onChange={(e) =>
                        setPeriods((v) =>
                          v.map((p, n) =>
                            n === i ? { ...p, until: e.target.value } : p,
                          ),
                        )
                      }
                    />
                  </Field>
                </div>
                {record.can_write && (
                  <Button
                    onClick={() =>
                      setPeriods((v) => v.filter((_, n) => n !== i))
                    }
                  >
                    기간 {i + 1} 제거
                  </Button>
                )}
              </div>
            ))}
            {record.can_write && (
              <>
                <div className="button-row">
                  <Button
                    disabled={periods.length >= 100}
                    onClick={() =>
                      setPeriods((v) => [
                        ...v,
                        {
                          version: record.document_version,
                          from: date,
                          until: "",
                        },
                      ])
                    }
                  >
                    유효기간 추가
                  </Button>
                  {older && (
                    <Button disabled={busy} onClick={() => void loadOlder()}>
                      더 오래된 버전 불러오기
                    </Button>
                  )}
                </div>
                <Field label="유효기간 변경 이유">
                  <textarea
                    rows={3}
                    value={reason}
                    maxLength={1000}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </Field>
                <Button
                  disabled={busy || !dirty || !reason.trim()}
                  variant="primary"
                  onClick={() => setConfirm(true)}
                >
                  등록 전 변경 비교
                </Button>
                <p>
                  이 입력은 아직 서버에 등록되지 않았으며 이 화면을 떠나면
                  사라집니다. 등록은 문서 게시나 승인 상태를 변경하지 않습니다.
                </p>
              </>
            )}
            {!!record.history.length && (
              <details>
                <summary>등록 이력 (최근 50개)</summary>
                {record.history.map((h) => (
                  <div className="evidence-review" key={h.revision}>
                    <h3>등록 revision {h.revision}</h3>
                    <p>
                      등록 시각 {datetime(h.recorded_at)} — 업무 유효일과 별개
                    </p>
                    <pre className="evidence-text">{periodText(h.periods)}</pre>
                    <pre className="evidence-text">{h.reason}</pre>
                  </div>
                ))}
              </details>
            )}
          </>
        )}
      </section>
      <Modal
        open={confirm && !!record}
        title="유효기간 변경 확인"
        onOpenChange={setConfirm}
      >
        {record && (
          <ChangeReview
            title="문서 열람자에게 등록 내용 공유"
            description="기간 전체를 교체합니다. 삭제된 기간은 검색에서 제외되며 등록 이력에는 남습니다."
            changes={[
              {
                label: "업무 유효기간",
                before: (
                  <pre className="evidence-text">
                    {periodText(record.periods)}
                  </pre>
                ),
                after: (
                  <pre className="evidence-text">{periodText(periods)}</pre>
                ),
              },
            ]}
            warnings={[
              "시작일 포함 · 종료일 제외. 이 표시는 사실성이나 게시 승인 인증이 아닙니다.",
            ]}
            onCancel={() => setConfirm(false)}
            onConfirm={() => void save()}
            confirmLabel="동의하고 유효기간 등록"
            busy={busy}
          />
        )}
      </Modal>
      <Modal
        open={!!preview}
        title="업무 기준일의 원문"
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          if (previewOpener.current?.isConnected)
            previewOpener.current.focus({ preventScroll: true });
        }}
        onOpenChange={(open) => {
          if (!open) {
            previewGeneration.current++;
            setPreview(null);
          }
        }}
      >
        {preview && (
          <>
            <h2>
              {preview.title} · v{preview.version}
            </h2>
            <p>
              {preview.date} 기준 등록 · 현재 v{preview.current_version}
            </p>
            <p>{preview.notice}</p>
            <pre className="evidence-text">{preview.markdown}</pre>
            <details>
              <summary>원문 내용 해시</summary>
              <p className="evidence-hash">{preview.content_hash}</p>
              <p>
                내용의 동일성을 비교하는 값입니다. 사실성·서명·승인 검증이
                아닙니다.
              </p>
            </details>
            <Link to={`/app/documents/${preview.document_id}`}>
              현재 문서 전체 열기
            </Link>
          </>
        )}
      </Modal>
    </div>
  );
}
