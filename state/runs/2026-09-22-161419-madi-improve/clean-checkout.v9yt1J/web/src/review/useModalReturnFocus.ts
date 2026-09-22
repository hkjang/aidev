import { useRef } from "react";

/** Controlled portals have no Radix Trigger. Preserve only an element reference,
 * scoped to the current actor/workspace; never steal focus from a newer modal. */
export function useModalReturnFocus(
  open: boolean | string | null,
  scope: string,
  trigger?: HTMLElement | null,
) {
  const previous = useRef<typeof open>(false),
    current = useRef(scope),
    opener = useRef<{ element: HTMLElement; scope: string } | null>(null);
  current.current = scope;
  if (open && open !== previous.current) {
    const element = trigger ?? document.activeElement;
    if (element instanceof HTMLElement) opener.current = { element, scope };
  }
  previous.current = open;
  return (event?: Event) => {
    event?.preventDefault();
    const value = opener.current;
    requestAnimationFrame(() => {
      if (
        value &&
        current.current === value.scope &&
        value.element.isConnected &&
        !document.querySelector('[role="dialog"][data-state="open"]')
      )
        value.element.focus({ preventScroll: true });
    });
  };
}
