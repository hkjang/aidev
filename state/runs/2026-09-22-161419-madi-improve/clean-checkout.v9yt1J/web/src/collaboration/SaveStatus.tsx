import type { MadiCollaborationProvider } from "./provider";

export default function CollaborationSaveStatus({
  provider,
}: {
  provider: MadiCollaborationProvider;
}) {
  const labels = {
    local: "로컬 변경 · 아직 서버에 미확정",
    committing: "서버 저장 확인 중…",
    confirmed: "서버 저장 확정",
    reconnecting: "재연결 중 · 로컬 초안 유지",
    recovery: "복구 확인 필요 · 자동 덮어쓰기 중지",
  };
  return (
    <span
      role="status"
      data-save-state={provider.saveState}
      title={
        provider.confirmedAt
          ? `마지막 서버 확인 ${new Date(provider.confirmedAt).toLocaleTimeString("ko-KR")}`
          : "DB 커밋 확인 후에만 서버 저장 확정으로 표시합니다."
      }
    >
      {labels[provider.saveState]}
      {provider.saveState === "confirmed" && provider.snapshot
        ? ` · v${provider.snapshot.version}`
        : ""}
    </span>
  );
}
