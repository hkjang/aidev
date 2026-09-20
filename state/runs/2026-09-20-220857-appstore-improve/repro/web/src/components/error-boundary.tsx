import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
import { Button, StatePanel } from "./ui";

interface Props {
  children: ReactNode;
}
interface State {
  error?: Error;
}

export class GlobalErrorBoundary extends Component<Props, State> {
  state: State = {};

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("Unhandled UI error", error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <main className="page" id="main-content">
        <StatePanel
          icon={<AlertTriangle />}
          title="예기치 않은 화면 오류가 발생했습니다"
          description="입력한 내용은 유지되지 않았을 수 있습니다. 새로고침 후 다시 시도해 주세요."
          actions={<Button onClick={() => location.reload()}>새로고침</Button>}
        />
      </main>
    );
  }
}
