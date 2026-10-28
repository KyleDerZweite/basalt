import type { RawGraph, WorkspaceEdge, WorkspaceNode } from "../types";

/** Preserve the collected evidence topology independently of display categories. */
export function buildEvidenceGraph(
  graph: RawGraph | undefined,
  displayNodes: WorkspaceNode[] = [],
): { nodes: WorkspaceNode[]; edges: WorkspaceEdge[] } {
  const categories = new Map<string, string>();
  for (const node of displayNodes) {
    for (const id of node.raw_node_ids ?? []) categories.set(id, node.category);
  }
  return {
    nodes: (graph?.nodes ?? []).map((node) => ({
      id: node.id,
      label: node.label,
      type: node.type,
      category: categories.get(node.id) ?? node.type,
      raw_node_ids: [node.id],
      confidence: node.confidence,
      wave: node.wave,
      pivot: node.pivot,
      source_modules: node.source_module ? [node.source_module] : [],
      profile_url: typeof node.properties?.profile_url === "string" ? node.properties.profile_url : undefined,
      properties: Object.fromEntries(
        Object.entries(node.properties ?? {})
          .filter(([, value]) => value != null)
          .map(([key, value]) => [key, typeof value === "object" ? JSON.stringify(value) : String(value)]),
      ),
    })),
    edges: graph?.edges ?? [],
  };
}
