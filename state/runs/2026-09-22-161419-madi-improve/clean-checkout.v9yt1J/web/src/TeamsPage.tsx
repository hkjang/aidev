import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Plus, Trash2, Users } from "lucide-react";
import { api } from "./api";
import { useApp } from "./context";
import { Button, Empty, ErrorBox, Field, Modal, PageHeading } from "./ui";
type Row = Record<string, any>;
export default function TeamsPage() {
  const { workspace, user, notify } = useApp(),
    [query, setQuery] = useSearchParams();
  const id = query.get("team") || "";
  const [teams, setTeams] = useState<Row[]>([]),
    [members, setMembers] = useState<Row[]>([]),
    [people, setPeople] = useState<Row[]>([]),
    [name, setName] = useState(""),
    [selected, setSelected] = useState(""),
    [editing, setEditing] = useState<Row | null>(null),
    [deleting, setDeleting] = useState<Row | null>(null),
    [editName, setEditName] = useState(""),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false);
  const manager =
      !!workspace &&
      ["owner", "admin"].includes(workspace.role) &&
      user.role !== "viewer",
    team = teams.find((t) => t.id === id);
  const load = useCallback(async () => {
    if (!workspace) return;
    try {
      const [t, p] = await Promise.all([
        api<Row[]>(`/teams?workspace_id=${workspace.id}`),
        api<Row[]>(`/workspaces/${workspace.id}/members`),
      ]);
      setTeams(t);
      setPeople(p);
      if (id) setMembers(await api<Row[]>(`/teams/${id}/members`));
      else setMembers([]);
      setError(null);
    } catch (e) {
      setError(e);
    }
  }, [workspace?.id, id]);
  useEffect(() => {
    void load();
  }, [load]);
  const run = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
      await load();
      return true;
    } catch (e) {
      setError(e);
      return false;
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page">
      <PageHeading
        eyebrow="TEAMS"
        title="팀과 멘션 그룹"
        description="문서에서 함께 언급할 팀을 관리합니다. 팀 가입만으로 문서 접근 권한이 생기지는 않습니다."
      />
      <ErrorBox error={error} />
      {manager && (
        <form
          className="panel padded"
          style={{ marginTop: 24 }}
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              const t = await api("/teams", "POST", {
                workspace_id: workspace?.id,
                name,
              });
              setName("");
              setQuery({ team: t.id });
              notify("팀을 만들었습니다");
            } catch (e) {
              setError(e);
            } finally {
              setBusy(false);
            }
          }}
        >
          <div className="button-row">
            <Field label="새 팀 이름">
              <input
                required
                maxLength={70}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="예: 플랫폼 운영팀"
              />
            </Field>
            <Button variant="primary" disabled={busy}>
              <Plus size={16} />팀 만들기
            </Button>
          </div>
        </form>
      )}
      <div className="spaces-layout">
        <section className="panel spaces-list">
          {teams.length === 0 ? (
            <Empty title="아직 팀이 없습니다" />
          ) : (
            teams.map((t) => (
              <button
                className={`space-list-item ${id === t.id ? "active" : ""}`}
                key={t.id}
                onClick={() => {
                  setQuery({ team: t.id });
                  setSelected("");
                }}
              >
                <Users size={20} />
                <span>
                  <strong>{t.name}</strong>
                  <small>등록 멤버 {t.member_count}명</small>
                </span>
              </button>
            ))
          )}
        </section>
        <section className="panel space-content">
          {team ? (
            <>
              <div className="space-heading">
                <div>
                  <h2>{team.name}</h2>
                  <p className="muted">
                    활성 워크스페이스 멤버 {members.length}명
                  </p>
                </div>
                {manager && (
                  <div className="space-actions">
                    <Button
                      disabled={busy}
                      onClick={() => {
                        setEditing(team);
                        setEditName(team.name);
                      }}
                    >
                      이름 변경
                    </Button>
                    <Button disabled={busy} onClick={() => setDeleting(team)}>
                      <Trash2 size={16} />팀 삭제
                    </Button>
                  </div>
                )}
              </div>
              {manager && (
                <form
                  style={{ marginTop: 24 }}
                  onSubmit={async (e) => {
                    e.preventDefault();
                    if (
                      await run(() =>
                        api(`/teams/${id}/members`, "PUT", {
                          user_id: selected,
                        }),
                      )
                    ) {
                      setSelected("");
                      notify("팀 멤버를 추가했습니다");
                    }
                  }}
                >
                  <Field label="추가할 워크스페이스 멤버">
                    <select
                      value={selected}
                      onChange={(e) => setSelected(e.target.value)}
                      required
                    >
                      <option value="">멤버 선택</option>
                      {people
                        .filter((p) => !members.some((m) => m.id === p.id))
                        .map((p) => (
                          <option key={p.id} value={p.id}>
                            {p.name} · {p.email}
                          </option>
                        ))}
                    </select>
                  </Field>
                  <Button disabled={busy || !selected}>
                    <Plus size={16} />
                    멤버 추가
                  </Button>
                </form>
              )}
              <div className="space-member-list">
                {members.map((m) => (
                  <div key={m.id}>
                    <span>
                      <strong>{m.name}</strong>
                      <small>{m.email}</small>
                    </span>
                    {manager && (
                      <Button
                        disabled={busy}
                        onClick={() =>
                          void run(() =>
                            api(`/teams/${id}/members`, "PUT", {
                              user_id: m.id,
                              remove: true,
                            }),
                          )
                        }
                        aria-label={`${m.name} 팀에서 제외`}
                      >
                        제외
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </>
          ) : (
            <Empty title="살펴볼 팀을 선택하세요" />
          )}
        </section>
      </div>
      <Modal
        open={!!editing}
        onOpenChange={(v) => !v && setEditing(null)}
        title="팀 이름 변경"
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (
              await run(() =>
                api(`/teams/${editing?.id}`, "PUT", { name: editName }),
              )
            )
              setEditing(null);
          }}
        >
          <Field label="팀 이름">
            <input
              required
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              maxLength={70}
            />
          </Field>
          <Button variant="primary" disabled={busy}>
            이름 저장
          </Button>
        </form>
      </Modal>
      <Modal
        open={!!deleting}
        onOpenChange={(v) => !v && setDeleting(null)}
        title="팀을 삭제할까요?"
        description="팀과 멘션 그룹 구성만 제거됩니다. 사용자 계정과 기존 문서는 삭제하지 않습니다."
      >
        <p>{deleting?.name}</p>
        <Button
          variant="danger"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              await api(`/teams/${deleting?.id}`, "DELETE");
              setDeleting(null);
              setQuery({});
              notify("팀을 삭제했습니다");
            } catch (e) {
              setError(e);
            } finally {
              setBusy(false);
            }
          }}
        >
          팀 삭제 확인
        </Button>
      </Modal>
    </div>
  );
}
