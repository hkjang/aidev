import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Download, WifiOff, RefreshCw, X } from "lucide-react";
import { Button } from "../ui";
import "./pwa.css";
type InstallPrompt = Event & {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: string }>;
};
export default function PWAControls() {
  const [online, setOnline] = useState(navigator.onLine);
  const [prompt, setPrompt] = useState<InstallPrompt | null>(null);
  const [update, setUpdate] = useState(false);
  const [hidden, setHidden] = useState(false);
  useEffect(() => {
    const network = () => setOnline(navigator.onLine);
    const install = (event: Event) => {
      event.preventDefault();
      setPrompt(event as InstallPrompt);
    };
    const newer = () => setUpdate(true);
    window.addEventListener("online", network);
    window.addEventListener("offline", network);
    window.addEventListener("beforeinstallprompt", install);
    window.addEventListener("madi-pwa-update", newer);
    void navigator.serviceWorker
      ?.getRegistration()
      .then((reg) =>
        setUpdate(!!reg?.waiting && !!navigator.serviceWorker.controller),
      );
    return () => {
      window.removeEventListener("online", network);
      window.removeEventListener("offline", network);
      window.removeEventListener("beforeinstallprompt", install);
      window.removeEventListener("madi-pwa-update", newer);
    };
  }, []);
  if (hidden || (online && !prompt && !update)) return null;
  return (
    <aside className="pwa-banner" aria-label="기기 및 연결 상태">
      {!online ? (
        <>
          <WifiOff size={18} />
          <span>
            서버 연결이 끊겼습니다. 저장된 문서는{" "}
            <a href="/offline.html">기기 보관함</a>에서 확인하세요.
          </span>
        </>
      ) : update ? (
        <>
          <RefreshCw size={18} />
          <span>
            새 버전을 받을 준비가 되었습니다. 편집 내용을 저장한 뒤 앱을 모두
            닫고 다시 열어 주세요.
          </span>
        </>
      ) : (
        <>
          <Download size={18} />
          <span>madi를 이 기기에 설치해 빠르게 열어 보세요.</span>
          <Button
            variant="secondary"
            onClick={async () => {
              await prompt?.prompt();
              await prompt?.userChoice;
              setPrompt(null);
            }}
          >
            앱 설치
          </Button>
          <Link to="/app/devices">기기 설정</Link>
        </>
      )}
      <button
        className="icon-button"
        aria-label="기기 안내 닫기"
        onClick={() => setHidden(true)}
      >
        <X size={17} />
      </button>
    </aside>
  );
}
