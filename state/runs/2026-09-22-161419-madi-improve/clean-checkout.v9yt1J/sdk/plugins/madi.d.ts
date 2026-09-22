/** madi Plugin SDK v1 — Worker globals only; no DOM or fetch API is permitted. */
type MadiUINode = {
  type:
    | "stack"
    | "row"
    | "text"
    | "heading"
    | "button"
    | "input"
    | "textarea"
    | "select"
    | "checkbox"
    | "code"
    | "image"
    | "divider"
    | "badge";
  text?: string;
  label?: string;
  placeholder?: string;
  value?: string | boolean;
  asset?: string;
  disabled?: boolean;
  options?: (string | { value: string; label: string })[];
  children?: MadiUINode[];
  onClick?: () => unknown | Promise<unknown>;
  onChange?: (value: string | boolean) => unknown | Promise<unknown>;
};
declare const madi: {
  ready(): Promise<{
    workspace_id: string;
    user: { id: string; name: string };
    manifest: unknown;
    capabilities: string[];
    initial_data?: unknown;
  }>;
  ui: { render(node: MadiUINode): void };
  documents: {
    list(args?: { q?: string; tag?: string }): Promise<any[]>;
    get(args: { id: string }): Promise<any>;
    create(args: {
      title: string;
      markdown?: string;
      visibility?: "private" | "workspace";
      space_id?: string;
    }): Promise<any>;
    update(args: {
      id: string;
      version: number;
      title: string;
      markdown: string;
    }): Promise<any>;
    delete(args: { id: string }): Promise<any>;
  };
  databases: {
    list(args?: {}): Promise<any[]>;
    get(args: { id: string }): Promise<any>;
    query(args: {
      id: string;
      filters?: unknown[];
      sorts?: unknown[];
      limit?: number;
    }): Promise<any>;
    createRow(args: {
      id: string;
      values: Record<string, unknown>;
    }): Promise<any>;
    updateRow(args: {
      id: string;
      row_id: string;
      values: Record<string, unknown>;
    }): Promise<any>;
    deleteRow(args: { id: string; row_id: string }): Promise<any>;
  };
  ai: {
    chat(
      args: { prompt: string; document_id?: string; max_tokens?: number },
      onChunk?: (chunk: {
        text?: string;
        sources?: { id: string; title: string }[];
      }) => void,
    ): Promise<{ text: string; sources: { id: string; title: string }[] }>;
  };
  storage: {
    get(key: string): Promise<unknown>;
    set(key: string, value: unknown): Promise<{ ok: boolean }>;
    delete(key: string): Promise<{ ok: boolean }>;
  };
  notify(message: string): Promise<unknown>;
  navigate(path: string): Promise<unknown>;
  asset(path: string): string;
  registerBlock(
    id: string,
    handler: (data: any) => unknown | Promise<unknown>,
  ): void;
  registerCommand(
    id: string,
    handler: (input: any) => unknown | Promise<unknown>,
  ): void;
  registerSidebar(
    id: string,
    handler: (input: any) => unknown | Promise<unknown>,
  ): void;
  registerMenu(
    id: string,
    handler: (input: any) => unknown | Promise<unknown>,
  ): void;
  registerImporter(
    id: string,
    handler: (file: {
      name: string;
      content: string;
    }) => unknown | Promise<unknown>,
  ): void;
  registerExporter(
    id: string,
    handler: () =>
      | { name: string; content: string }
      | Promise<{ name: string; content: string }>,
  ): void;
  registerAIProvider(id: string): void;
};
