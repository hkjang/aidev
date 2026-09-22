import { useSearchParams } from "react-router-dom";
import { Activity, Flag, Palette, Radio } from "lucide-react";
import { useApp } from "../context";
import { Empty, PageHeading } from "../ui";
import { FeaturePanel } from "./FeaturePanel";
import { BrandingPanel } from "./BrandingPanel";
import { HealthPanel, TelemetryPanel } from "./TelemetryPanel";
import { leaveOperations } from "./shared";
import "./style.css";

export default function OperationsPage({
  workspaceMode = false,
}: {
  workspaceMode?: boolean;
}) {
  const { user, workspace } = useApp();
  const [params, setParams] = useSearchParams();
  const allowed = workspaceMode
    ? !!workspace &&
      ["owner", "admin"].includes(workspace.role) &&
      user.role !== "viewer"
    : user.role === "admin";
  const tabs = workspaceMode
    ? [
        { id: "branding", label: "브랜딩", icon: Palette },
        { id: "features", label: "기능 정책", icon: Flag },
      ]
    : [
        { id: "health", label: "상태와 오류", icon: Activity },
        { id: "telemetry", label: "OpenTelemetry", icon: Radio },
        { id: "features", label: "기능 정책", icon: Flag },
      ];
  const tab = tabs.find((t) => t.id === params.get("tab"))?.id || tabs[0].id;
  if (!allowed)
    return (
      <div className="page">
        <Empty
          title="운영 설정 권한이 필요합니다"
          text={
            workspaceMode
              ? "현재 워크스페이스 소유자 또는 관리자로 접근하세요."
              : "서비스 관리자로 접근하세요."
          }
        />
      </div>
    );
  return (
    <div className="page operations-page">
      <PageHeading
        eyebrow={workspaceMode ? "WORKSPACE OPERATIONS" : "SERVICE OPERATIONS"}
        title={workspaceMode ? "팀 운영 설정" : "서비스 운영"}
        description={
          workspaceMode
            ? "팀 브랜딩과 기능 공개 범위를 관리합니다. 인증·권한·승인·정보보호 정책은 그대로 적용합니다."
            : "실제 서비스 상태를 확인하고, 내부망 환경에 맞게 진단 전송과 기능 정책을 제어합니다."
        }
      />
      <nav className="operations-tabs" aria-label="운영 설정 메뉴">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            className={id === tab ? "active" : ""}
            aria-current={id === tab ? "page" : undefined}
            onClick={() => {
              if (tab === id || !leaveOperations()) return;
              const next = new URLSearchParams(window.location.search);
              next.set("tab", id);
              setParams(next);
            }}
          >
            <Icon size={18} />
            {label}
          </button>
        ))}
      </nav>
      <div
        key={`${user.id}:${workspaceMode ? workspace?.id : "service"}:${tab}`}
      >
        {tab === "features" ? (
          <FeaturePanel
            workspaceID={workspaceMode ? workspace!.id : undefined}
          />
        ) : tab === "branding" ? (
          <BrandingPanel workspaceID={workspace!.id} />
        ) : tab === "telemetry" ? (
          <TelemetryPanel />
        ) : (
          <HealthPanel />
        )}
      </div>
    </div>
  );
}
