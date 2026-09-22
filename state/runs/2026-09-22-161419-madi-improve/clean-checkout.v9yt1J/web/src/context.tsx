import { createContext, useContext } from "react";
import type { Doc, DocSummary, User, Workspace } from "./api";
export type AppContextType = {
  user: User;
  setUser: (u: User) => void;
  workspaces: Workspace[];
  workspace: Workspace | null;
  setWorkspace: (id: string) => void;
  documents: DocSummary[];
  reload: () => Promise<void>;
  createDocument: (
    title?: string,
    markdown?: string,
    extra?: Record<string, any>,
  ) => Promise<Doc | undefined>;
  notify: (text: string, type?: "success" | "error") => void;
  publicInfo: Record<string, any>;
  refreshPublic: () => Promise<void>;
};
export const AppContext = createContext<AppContextType>(null!);
export const useApp = () => useContext(AppContext);
