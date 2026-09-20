import {
  QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import {
  beginSilentSso,
  clearSilentSsoState,
  markSignedOut,
  shouldAttemptSilentSso,
} from "../features/auth/silent-sso";
import { useLocation } from "react-router-dom";
import { api, setCsrfToken } from "../lib/api";
import type { Session } from "../types";

export type ThemePreference = "light" | "dark" | "system";

interface ThemeContextValue {
  preference: ThemePreference;
  resolved: "light" | "dark";
  setPreference: (preference: ThemePreference) => void;
  toggle: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);
const AuthContext = createContext<{
  session?: Session;
  isPending: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
} | null>(null);

function initialTheme(): ThemePreference {
  const stored = localStorage.getItem("appstore.theme");
  return stored === "light" || stored === "dark" || stored === "system"
    ? stored
    : "system";
}

function ThemeProvider({ children }: PropsWithChildren) {
  const [preference, setPreferenceState] =
    useState<ThemePreference>(initialTheme);
  const publicConfig = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    staleTime: 60_000,
  });
  const [systemDark, setSystemDark] = useState(
    () => matchMedia("(prefers-color-scheme: dark)").matches,
  );
  const resolved =
    preference === "system" ? (systemDark ? "dark" : "light") : preference;

  useEffect(() => {
    if (localStorage.getItem("appstore.theme")) return;
    const configured = publicConfig.data?.theme;
    if (
      configured === "light" ||
      configured === "dark" ||
      configured === "system"
    ) {
      setPreferenceState(configured);
    }
  }, [publicConfig.data?.theme]);

  useEffect(() => {
    const query = matchMedia("(prefers-color-scheme: dark)");
    const listener = (event: MediaQueryListEvent) =>
      setSystemDark(event.matches);
    query.addEventListener("change", listener);
    return () => query.removeEventListener("change", listener);
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = resolved;
    document.documentElement.style.colorScheme = resolved;
  }, [resolved]);

  const setPreference = useCallback((next: ThemePreference) => {
    localStorage.setItem("appstore.theme", next);
    setPreferenceState(next);
  }, []);

  const value = useMemo<ThemeContextValue>(
    () => ({
      preference,
      resolved,
      setPreference,
      toggle: () => setPreference(resolved === "dark" ? "light" : "dark"),
    }),
    [preference, resolved, setPreference],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
}

export function AuthProvider({ children }: PropsWithChildren) {
  const queryClient = useQueryClient();
  const query = useQuery({
    queryKey: ["session"],
    queryFn: ({ signal }) => api.session(signal),
    retry: false,
    staleTime: 30_000,
  });

  useEffect(() => setCsrfToken(query.data?.csrfToken), [query.data?.csrfToken]);

  const value = useMemo(
    () => ({
      session: query.data,
      isPending: query.isPending,
      error: query.error,
      refresh: async () => {
        await queryClient.invalidateQueries({ queryKey: ["session"] });
      },
      logout: async () => {
        // Marked before the session goes away: a silent SSO attempt right
        // after signing out would make the sign-out look broken.
        markSignedOut();
        await api.logout();
        setCsrfToken();
        queryClient.setQueryData(["session"], {
          authenticated: false,
        } satisfies Session);
        await queryClient.invalidateQueries();
      },
    }),
    [query.data, query.error, query.isPending, queryClient],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false, staleTime: 15_000 },
    mutations: { retry: 0 },
  },
});

/**
 * Applies the configured favicon to the live document. The bundled icon link is
 * static, so an administrator's upload would otherwise never show in the tab.
 */
function FaviconEffect() {
  const config = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    staleTime: 60_000,
  });
  const faviconUrl = config.data?.faviconUrl;
  useEffect(() => {
    if (!faviconUrl) return;
    const link =
      document.querySelector<HTMLLinkElement>('link[rel="icon"]') ??
      document.head.appendChild(
        Object.assign(document.createElement("link"), { rel: "icon" }),
      );
    const previous = link.getAttribute("href");
    link.setAttribute("href", faviconUrl);
    return () => {
      if (previous === null) link.remove();
      else link.setAttribute("href", previous);
    };
  }, [faviconUrl]);
  return null;
}

/**
 * Tries a silent SSO sign-in once the session and public config are known.
 * Someone already signed in at the identity provider then lands on the page
 * they opened without seeing a login screen; someone who is not is sent back
 * to /login?sso=none by the callback and never bounced again.
 */
function SilentSsoEffect() {
  const auth = useAuth();
  const location = useLocation();
  const config = useQuery({
    queryKey: ["public-config"],
    queryFn: ({ signal }) => api.publicConfig(signal),
    staleTime: 60_000,
  });
  const authenticated = auth.session?.authenticated ?? false;
  const settled = !auth.isPending && !auth.error && !!config.data;
  useEffect(() => {
    if (!settled) return;
    if (authenticated) {
      // A session exists again, so a later sign-out may be followed by a
      // silent attempt in a new tab as usual.
      clearSilentSsoState();
      return;
    }
    if (
      !shouldAttemptSilentSso({
        config: config.data,
        pathname: location.pathname,
        search: location.search,
      })
    )
      return;
    beginSilentSso(location.pathname + location.search);
  }, [settled, authenticated, config.data, location.pathname, location.search]);
  return null;
}

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <FaviconEffect />
          <SilentSsoEffect />
          {children}
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export function useTheme(): ThemeContextValue {
  const value = useContext(ThemeContext);
  if (!value) throw new Error("ThemeProvider가 필요합니다.");
  return value;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthProvider가 필요합니다.");
  return value;
}
