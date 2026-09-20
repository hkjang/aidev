import {
  createContext,
  useContext,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";
import { safeJsonParse } from "../../lib/utils";

const KEY = "appstore.favorites";
const FavoritesContext = createContext<{
  slugs: string[];
  isFavorite: (slug: string) => boolean;
  toggle: (slug: string) => void;
} | null>(null);

export function FavoritesProvider({ children }: PropsWithChildren) {
  const [slugs, setSlugs] = useState<string[]>(() =>
    safeJsonParse(localStorage.getItem(KEY), []),
  );
  const value = useMemo(
    () => ({
      slugs,
      isFavorite: (slug: string) => slugs.includes(slug),
      toggle: (slug: string) =>
        setSlugs((current) => {
          const next = current.includes(slug)
            ? current.filter((item) => item !== slug)
            : [...current, slug];
          localStorage.setItem(KEY, JSON.stringify(next));
          return next;
        }),
    }),
    [slugs],
  );
  return (
    <FavoritesContext.Provider value={value}>
      {children}
    </FavoritesContext.Provider>
  );
}

export function useFavorites() {
  const value = useContext(FavoritesContext);
  if (!value) throw new Error("FavoritesProvider가 필요합니다.");
  return value;
}
