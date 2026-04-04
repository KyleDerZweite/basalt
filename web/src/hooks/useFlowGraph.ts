import { useEffect, useState } from "react";
import { Position, type Edge, type Node } from "@xyflow/react";
import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";

import type { GraphNodeData, ScanWorkspace } from "../types";

const elk = new ELK();

export function useFlowGraph(graph: ScanWorkspace["graph"], selectedNode: string) {
  const [layouted, setLayouted] = useState<{ nodes: Node<GraphNodeData>[]; edges: Edge[] }>({ nodes: [], edges: [] });

  useEffect(() => {
    let mounted = true;

    async function layout() {
      const baseNodes: Node<GraphNodeData>[] = graph.nodes.map((node) => ({
        id: node.id,
        type: "workspaceNode",
        position: { x: 0, y: 0 },
        data: {
          label: node.label,
          type: node.type,
          category: node.category,
          confidence: node.confidence,
          collapsedCount: node.collapsed_count,
          profileURL: node.profile_url,
        },
        sourcePosition: Position.Right,
        targetPosition: Position.Left,
        width: node.category === "root" ? 320 : node.category === "aliases" ? 240 : 264,
        height: node.category === "root" ? 116 : 102,
        selected: node.id === selectedNode,
      }));

      const baseEdges: Edge[] = graph.edges.map((edge) => ({
        id: edge.id,
        source: edge.source,
        target: edge.target,
        type: "smoothstep",
        className: `flow-edge ${edge.type}`,
        animated: edge.type === "warning",
      }));

      if (baseNodes.length === 0) {
        if (mounted) {
          setLayouted({ nodes: [], edges: [] });
        }
        return;
      }

      const elkGraph: ElkNode = {
        id: "workspace-root",
        layoutOptions: {
          "elk.algorithm": "layered",
          "elk.direction": "RIGHT",
          "elk.layered.spacing.nodeNodeBetweenLayers": "168",
          "elk.spacing.nodeNode": "84",
          "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
        },
        children: baseNodes.map((node) => ({
          id: node.id,
          width: Number(node.width ?? 240),
          height: Number(node.height ?? 104),
        })),
        edges: baseEdges.map((edge) => ({ id: edge.id, sources: [edge.source], targets: [edge.target] })),
      };

      const layoutedGraph = await elk.layout(elkGraph);
      const nextNodes = baseNodes.map((node) => {
        const positioned = layoutedGraph.children?.find((child) => child.id === node.id);
        return {
          ...node,
          selected: node.id === selectedNode,
          position: { x: positioned?.x ?? 0, y: positioned?.y ?? 0 },
        };
      });

      if (mounted) {
        setLayouted({ nodes: nextNodes, edges: baseEdges });
      }
    }

    void layout();
    return () => {
      mounted = false;
    };
  }, [graph, selectedNode]);

  return layouted;
}
