export type GraphNode = {
  id: string;
  title: string;
  tags: string[];
  icon?: string;
  space_id: string | null;
  parent_id: string | null;
  owner_id: string;
  indexed: boolean;
  can_write?: boolean;
  version?: number;
};
export type GraphEdge = {
  source: string;
  target: string;
  type: "reference" | "related" | "parent" | "policy" | "execution" | "data";
  origin: "wiki" | "manual" | "tree";
};
export type GraphData = {
  nodes: GraphNode[];
  edges: GraphEdge[];
  unresolved: {
    source: string;
    target: string;
    reason: "unresolved" | "ambiguous";
  }[];
  diagnostics: {
    limit: number;
    truncated: boolean;
    pending: number;
    notice?: string;
  };
};
export type GraphFilters = {
  q: string;
  tag: string;
  space: string;
  owner: string;
  focus: string;
  depth: number;
  type: string;
};
export const emptyGraph: GraphData = {
  nodes: [],
  edges: [],
  unresolved: [],
  diagnostics: { limit: 2000, truncated: false, pending: 0 },
};

export function filterGraph(data: GraphData, filters: GraphFilters) {
  const q = filters.q.trim().toLocaleLowerCase("ko-KR");
  let nodes = data.nodes.filter(
    (node) =>
      node.id === filters.focus ||
      ((!q ||
        node.title.toLocaleLowerCase("ko-KR").includes(q) ||
        node.tags?.some((tag) => tag.toLocaleLowerCase("ko-KR").includes(q))) &&
        (!filters.tag || node.tags?.includes(filters.tag)) &&
        (!filters.space ||
          (filters.space === "none"
            ? !node.space_id
            : node.space_id === filters.space)) &&
        (!filters.owner || node.owner_id === filters.owner)),
  );
  let ids = new Set(nodes.map((node) => node.id));
  let edges = data.edges.filter(
    (edge) =>
      ids.has(edge.source) &&
      ids.has(edge.target) &&
      (!filters.type || edge.type === filters.type),
  );
  if (filters.focus) {
    const reachable = new Set(ids.has(filters.focus) ? [filters.focus] : []);
    const adjacent = new Map<string, Set<string>>();
    for (const edge of edges) {
      if (!adjacent.has(edge.source)) adjacent.set(edge.source, new Set());
      if (!adjacent.has(edge.target)) adjacent.set(edge.target, new Set());
      adjacent.get(edge.source)!.add(edge.target);
      adjacent.get(edge.target)!.add(edge.source);
    }
    let frontier = [...reachable];
    for (let step = 0; step < Math.min(5, Math.max(1, filters.depth)); step++) {
      const next: string[] = [];
      for (const id of frontier)
        for (const neighbor of adjacent.get(id) || [])
          if (!reachable.has(neighbor)) {
            reachable.add(neighbor);
            next.push(neighbor);
          }
      frontier = next;
      if (!frontier.length) break;
    }
    nodes = nodes.filter((node) => reachable.has(node.id));
    ids = new Set(nodes.map((node) => node.id));
    edges = edges.filter(
      (edge) => ids.has(edge.source) && ids.has(edge.target),
    );
  }
  const connected = new Set(
    edges.flatMap((edge) => [edge.source, edge.target]),
  );
  return {
    nodes,
    edges,
    unresolved: (data.unresolved || []).filter((link) => ids.has(link.source)),
    isolated: nodes.filter(
      (node) => node.indexed !== false && !connected.has(node.id),
    ),
  };
}
