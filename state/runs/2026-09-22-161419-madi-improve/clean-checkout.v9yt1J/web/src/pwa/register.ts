export async function registerPWA() {
  if (!window.isSecureContext || !("serviceWorker" in navigator)) return;
  try {
    const registration = await navigator.serviceWorker.register("/sw.js", {
      scope: "/",
    });
    const signal = () => {
      if (registration.waiting && navigator.serviceWorker.controller)
        window.dispatchEvent(new Event("madi-pwa-update"));
    };
    signal();
    registration.addEventListener("updatefound", () =>
      registration.installing?.addEventListener("statechange", signal),
    );
    window.addEventListener("online", () => {
      void registration.update().catch(() => {});
    });
  } catch {
    /* Installation is optional; normal service use remains available. */
  }
}
