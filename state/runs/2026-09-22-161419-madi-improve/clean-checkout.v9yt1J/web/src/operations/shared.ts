import { useEffect, useRef } from "react";

export function useOperationsGuard(dirty: boolean) {
  useEffect(() => {
    const leave = (event: BeforeUnloadEvent) => {
      if (dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    const internal = (event: Event) => {
      if (
        dirty &&
        !window.confirm("저장하지 않은 운영 설정을 버리고 이동할까요?")
      )
        event.preventDefault();
    };
    window.addEventListener("beforeunload", leave);
    window.addEventListener("madi:operations-leave", internal);
    return () => {
      window.removeEventListener("beforeunload", leave);
      window.removeEventListener("madi:operations-leave", internal);
    };
  }, [dirty]);
}
export function useOperationMounted() {
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  return mounted;
}
export function refreshPresentation() {
  window.dispatchEvent(new Event("madi:presentation-refresh"));
}
export function leaveOperations() {
  return window.dispatchEvent(
    new Event("madi:operations-leave", { cancelable: true }),
  );
}
