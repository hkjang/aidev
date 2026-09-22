import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { Archive } from "lucide-react";
import { api } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Modal } from "../ui";

export default function SaveEvidence({ ticket, onNavigate }: { ticket: string; onNavigate: () => void }) {
  const { user, workspace } = useApp();
  const [open, setOpen] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState(""), [saved, setSaved] = useState("");
  const generation = useRef(0);
  useEffect(() => { generation.current++; setOpen(false); setBusy(false); setError(""); setSaved(""); return () => { generation.current++; }; }, [ticket, user.id, workspace?.id]);
  const save = async () => {
    if (busy) return;
    const request = generation.current;
    setBusy(true); setError("");
    try {
      const result = await api<{ id: string }>("/ai/evidence", "POST", { ticket, consent: true });
      if (request !== generation.current) return;
      setSaved(result.id); setOpen(false);
    } catch (e) { if (request === generation.current) setError((e as Error).message); }
    finally { if (request === generation.current) setBusy(false); }
  };
  return <>
    {saved ? <Link className="button" to={`/app/evidence?id=${saved}`} onClick={onNavigate}><Archive size={16} /> 보관한 근거 보기</Link>
      : <Button type="button" onClick={() => setOpen(true)}><Archive size={16} /> 당시 근거 보관</Button>}
    <Modal open={open} onOpenChange={(value) => { if (!busy) setOpen(value); }} title="당시 AI 근거 보관" description="답변의 사실성을 인증하는 기능이 아닙니다.">
      <p>질문·완료 답변·모델 설정 식별값·실제 사용한 원문 구간을 본인의 별도 보관함에 암호화하여 저장합니다. 원문이 바뀌어도 당시 근거와 현재 구간을 비교할 수 있습니다.</p>
      <p className="notice">관리자가 보관 정책을 활성화해야 합니다. 현재 문서 권한·민감정보·보존기간 정책이 계속 적용되며, 다른 사용자와 관리자는 내용을 볼 수 없습니다. 개인 대화 기록과 독립적으로 보관되고 서비스 백업에 포함됩니다.</p>
      <ErrorBox error={error} />
      <div className="modal-actions"><Button disabled={busy} onClick={() => setOpen(false)}>취소</Button><Button variant="primary" disabled={busy} onClick={() => void save()}>{busy ? "근거 확인 중…" : "동의하고 근거 보관"}</Button></div>
    </Modal>
  </>;
}
