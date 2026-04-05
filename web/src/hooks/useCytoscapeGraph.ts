import { useMemo } from "react";
import type cytoscape from "cytoscape";
import type { ScanWorkspace } from "../types";

/** Transforms workspace graph data into Cytoscape element definitions. */
export function useCytoscapeGraph(graph: ScanWorkspace["graph"]): cytoscape.ElementDefinition[] {
  return useMemo(() => {
    const nodes: cytoscape.ElementDefinition[] = graph.nodes.map((node) => ({
      data: {
        id: node.id,
        label: node.label,
        category: node.category,
        type: node.type,
        confidence: node.confidence ?? 0,
        collapsedCount: node.collapsed_count ?? 0,
        profileURL: node.profile_url ?? "",
      },
    }));

    const edges: cytoscape.ElementDefinition[] = graph.edges.map((edge) => ({
      data: {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        type: edge.type,
      },
    }));

    return [...nodes, ...edges];
  }, [graph]);
}
