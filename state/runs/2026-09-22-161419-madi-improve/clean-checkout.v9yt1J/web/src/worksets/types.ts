export type WorksetItem = {
  kind: "document" | "database" | "task";
  resource_id: string;
  context: Record<string, any>;
};
export type Workset = {
  id: string;
  name: string;
  workspace_id: string;
  owner_id: string;
  kind: "workset" | "reference";
  version: number;
  items: WorksetItem[];
};
export type Resolved = WorksetItem & {
  available: boolean;
  title?: string;
  version?: number;
  url?: string;
  restore_context?: Record<string, any>;
  context_changed?: boolean;
  context_notice?: string;
  tags?: string[];
  stale?: boolean;
  task_text?: string;
  done?: boolean;
  can_write?: boolean;
};
export type WorksetData = {
  workset: Workset;
  resolved: Resolved[];
  notice: string;
};
