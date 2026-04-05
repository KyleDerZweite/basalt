import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import cytoscape from "cytoscape";
// @ts-expect-error cytoscape-dagre has no bundled types
import dagre from "cytoscape-dagre";
import { fontFamilies } from "../lib/typography";

cytoscape.use(dagre);

/** CSS variable resolver — reads computed values from :root so Cytoscape can use them. */
function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function buildStylesheet(): cytoscape.StylesheetStyle[] {
  return [
    // ── Base node
    {
      selector: "node",
      style: {
        "background-color": cssVar("--bg-elevated"),
        "border-color": cssVar("--border"),
        "border-width": 1.5,
        color: cssVar("--text-primary"),
        "font-family": fontFamilies.body,
        "font-size": 11,
        "text-valign": "center",
        "text-halign": "center",
        label: "data(label)",
        "text-max-width": "140px",
        "text-wrap": "wrap",
        width: 154,
        height: 64,
        "overlay-padding": 6,
        "transition-property": "background-color, border-color, border-width",
        "transition-duration": 150,
      },
    },

    // ── Root node (hexagon, amber)
    {
      selector: 'node[category = "root"]',
      style: {
        shape: "hexagon",
        "background-color": cssVar("--accent-soft"),
        "border-color": cssVar("--accent"),
        "border-width": 2.5,
        color: cssVar("--accent"),
        "font-size": 13,
        "font-weight": "bold",
        width: 140,
        height: 64,
      },
    },

    // ── Alias node (rounded rect, amber border)
    {
      selector: 'node[category = "aliases"]',
      style: {
        shape: "round-rectangle",
        "background-color": cssVar("--bg-elevated"),
        "border-color": cssVar("--accent-dim"),
        "border-width": 1.5,
        color: cssVar("--text-secondary"),
        width: 130,
        height: 48,
      },
    },

    // ── Account node
    {
      selector: 'node[category = "account"]',
      style: {
        shape: "round-rectangle",
        width: 140,
        height: 52,
      },
    },

    // ── Email node (ellipse)
    {
      selector: 'node[type = "email"]',
      style: {
        shape: "ellipse",
        "border-color": cssVar("--info"),
        "background-color": cssVar("--info-soft"),
        color: cssVar("--info"),
        width: 160,
        height: 44,
      },
    },

    // ── Domain node (diamond)
    {
      selector: 'node[type = "domain"]',
      style: {
        shape: "diamond",
        "border-color": cssVar("--warning"),
        "background-color": cssVar("--warning-soft"),
        color: cssVar("--warning"),
        width: 120,
        height: 60,
      },
    },

    // ── Username node (tag)
    {
      selector: 'node[type = "username"]',
      style: {
        shape: "round-tag",
        "border-color": cssVar("--border-strong"),
        width: 130,
        height: 44,
      },
    },

    // ── Website node
    {
      selector: 'node[type = "website"]',
      style: {
        shape: "round-rectangle",
        "border-color": cssVar("--success"),
        "background-color": cssVar("--success-soft"),
        color: cssVar("--success"),
      },
    },

    // ── Selected state
    {
      selector: "node:selected",
      style: {
        "border-color": cssVar("--accent"),
        "border-width": 2.5,
        "background-color": cssVar("--accent-soft"),
        color: cssVar("--accent"),
      },
    },

    // ── Hover (active) state
    {
      selector: "node:active",
      style: {
        "overlay-color": cssVar("--accent"),
        "overlay-opacity": 0.08,
      },
    },

    // ── Base edge
    {
      selector: "edge",
      style: {
        "line-color": cssVar("--border"),
        "target-arrow-color": cssVar("--border"),
        "target-arrow-shape": "triangle",
        "arrow-scale": 0.8,
        "curve-style": "bezier",
        width: 1.5,
        opacity: 0.7,
      },
    },

    // ── Warning edge (dashed red)
    {
      selector: 'edge[type = "warning"]',
      style: {
        "line-color": cssVar("--danger"),
        "target-arrow-color": cssVar("--danger"),
        "line-style": "dashed",
        "line-dash-pattern": [6, 3],
        width: 1.5,
        opacity: 0.9,
      },
    },

    // ── Alias edge (lighter)
    {
      selector: 'edge[type = "alias"]',
      style: {
        "line-color": cssVar("--accent-dim"),
        "target-arrow-color": cssVar("--accent-dim"),
        "line-style": "dashed",
        "line-dash-pattern": [4, 4],
        width: 1,
        opacity: 0.6,
      },
    },

    // ── Selected edge
    {
      selector: "edge:selected",
      style: {
        "line-color": cssVar("--accent"),
        "target-arrow-color": cssVar("--accent"),
        width: 2.5,
        opacity: 1,
      },
    },
  ];
}

export interface CytoscapeGraphHandle {
  exportPNG: () => string;
  fit: () => void;
}

interface CytoscapeGraphProps {
  elements: cytoscape.ElementDefinition[];
  selectedNodeId?: string;
  onNodeClick: (id: string) => void;
  resizeKey?: string;
}

export const CytoscapeGraph = forwardRef<CytoscapeGraphHandle, CytoscapeGraphProps>(
  function CytoscapeGraph({ elements, selectedNodeId, onNodeClick, resizeKey }, ref) {
    const containerRef = useRef<HTMLDivElement>(null);
    const cyRef = useRef<cytoscape.Core | null>(null);

    // Imperative handle for parent components
    useImperativeHandle(ref, () => ({
      exportPNG: () => cyRef.current?.png({ full: true, scale: 2 }) ?? "",
      fit: () => cyRef.current?.fit(undefined, 40),
    }));

    // Mount Cytoscape once
    useEffect(() => {
      if (!containerRef.current) return;

      const cy = cytoscape({
        container: containerRef.current,
        elements: [],
        style: buildStylesheet(),
        layout: { name: "preset" },
        userZoomingEnabled: true,
        userPanningEnabled: true,
        boxSelectionEnabled: false,
        autounselectify: false,
        wheelSensitivity: 0.3,
        minZoom: 0.1,
        maxZoom: 4,
      });

      cy.on("tap", "node", (evt) => {
        onNodeClick(evt.target.id() as string);
      });

      // Click on background deselects
      cy.on("tap", (evt) => {
        if (evt.target === cy) {
          onNodeClick("");
        }
      });

      cyRef.current = cy;

      const observer = new ResizeObserver(() => {
        cy.resize();
      });
      observer.observe(containerRef.current);

      return () => {
        observer.disconnect();
        cy.destroy();
        cyRef.current = null;
      };
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // Keep onNodeClick handler current without remounting
    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) return;
      cy.removeListener("tap", "node");
      cy.on("tap", "node", (evt) => {
        onNodeClick(evt.target.id() as string);
      });
    }, [onNodeClick]);

    // Update elements and re-run layout when graph data changes
    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) return;

      cy.elements().remove();
      cy.add(elements);

      if (elements.length > 0) {
        cy.layout({
          name: "dagre",
          rankDir: "LR",
          nodeSep: 80,
          rankSep: 160,
          edgeSep: 20,
          ranker: "network-simplex",
          animate: false,
          padding: 40,
        } as cytoscape.LayoutOptions).run();

        cy.fit(undefined, 40);
      }
    }, [elements]);

    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) {
        return;
      }

      cy.resize();
      cy.fit(undefined, 40);
    }, [resizeKey]);

    // Sync selection highlighting
    useEffect(() => {
      const cy = cyRef.current;
      if (!cy) return;
      cy.elements().unselect();
      if (selectedNodeId) {
        cy.$id(selectedNodeId).select();
      }
    }, [selectedNodeId]);

    return (
      <div
        ref={containerRef}
        style={{ width: "100%", height: "100%", position: "absolute", inset: 0 }}
      />
    );
  }
);
