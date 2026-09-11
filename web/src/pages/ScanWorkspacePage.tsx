import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, AlertTriangle, CircleDashed, PanelRight, PanelRightClose } from "lucide-react";

import { CytoscapeGraph, type CytoscapeGraphHandle } from "../components/CytoscapeGraph";
import { EventTicker } from "../components/EventTicker";
import { FindingCard } from "../components/FindingCard";
import { NodeInspector } from "../components/NodeInspector";
import { PretextBlock } from "../components/PretextBlock";
import { StatusPill } from "../components/StatusPill";
import { useCytoscapeGraph } from "../hooks/useCytoscapeGraph";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useScanEvents } from "../hooks/useScanEvents";
import { api } from "../lib/api";
import { asMessage } from "../lib/format";
import { ACTIVE_STATUSES } from "../lib/constants";
import { lineHeights, pretextFonts } from "../lib/typography";
import type { ScanWorkspace, WorkspaceNode } from "../types";

interface ScanWorkspacePageProps {
  onRefreshHome: () => void;
}

type PanelTab = "insights" | "inspector" | "events" | "raw";

export function ScanWorkspacePage({ onRefreshHome }: ScanWorkspacePageProps) {
  const { scanID } = useParams<{ scanID: string }>();
  const navigate = useNavigate();

  const [workspace, setWorkspace] = useState<ScanWorkspace | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState("");
  const [activeTab, setActiveTab] = useState<PanelTab>("insights");
  const [panelHidden, setPanelHidden] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [cancelLoading, setCancelLoading] = useState(false);
  const [filterQuery, setFilterQuery] = useState("");
  const [filterType, setFilterType] = useState("all");
  const [minConfidence, setMinConfidence] = useState(0);
  const [focusSelected, setFocusSelected] = useState(false);
  const isPanelOverlay = useMediaQuery("(max-width: 1199px)");
  const isPanelSheet = useMediaQuery("(max-width: 767px)");

  const graphRef = useRef<CytoscapeGraphHandle>(null);

  const fetchWorkspace = useCallback(async () => {
    if (!scanID) return;
    try {
      const ws = await api<ScanWorkspace>(`/api/scans/${scanID}/workspace`);
      setWorkspace(ws);
      onRefreshHome();
    } catch (reason) {
      setLoadError(asMessage(reason));
    }
  }, [scanID, onRefreshHome]);

  useEffect(() => {
    void fetchWorkspace();
  }, [fetchWorkspace]);

  const isActive = workspace
    ? (ACTIVE_STATUSES as readonly string[]).includes(workspace.record.status)
    : false;

  const { events, isConnected } = useScanEvents({
    scanId: scanID ?? "",
    scanStatus: workspace?.record.status ?? "",
    onWorkspaceUpdate: fetchWorkspace,
  });

  // The events tab keeps its unread count badge, but the panel stays where
  // the user left it so live progress never pulls focus from the findings.

  useEffect(() => {
    if (isPanelOverlay) {
      setPanelHidden(true);
    }
  }, [isPanelOverlay]);

  // Node selection: find WorkspaceNode by ID
  const selectedNode: WorkspaceNode | null = workspace
    ? (workspace.graph.nodes.find((n) => n.id === selectedNodeId) ?? null)
    : null;

  const handleNodeClick = useCallback((id: string) => {
    setSelectedNodeId(id);
    if (id) setActiveTab("inspector");
  }, []);

  const handleCancel = async () => {
    if (!scanID) return;
    setCancelLoading(true);
    try {
      await api(`/api/scans/${scanID}/cancel`, { method: "POST" });
      await fetchWorkspace();
    } catch {
      // ignore
    } finally {
      setCancelLoading(false);
    }
  };

  const handleExport = (format: "json" | "csv") => {
    window.open(`/api/scans/${scanID}/export?format=${format}`, "_blank");
  };

  const handleExportPNG = () => {
    const dataURL = graphRef.current?.exportPNG();
    if (!dataURL) return;
    const a = document.createElement("a");
    a.href = dataURL;
    a.download = `basalt-scan-${scanID?.slice(0, 8)}.png`;
    a.click();
  };

  // Cytoscape elements, narrowed by the filter bar and optional focus mode.
  const visibleGraph = useMemo(() => {
    const graph = workspace?.graph ?? { layout: "", nodes: [], edges: [] };
    const query = filterQuery.trim().toLowerCase();
    const keep = new Set<string>();
    for (const node of graph.nodes) {
      if (filterType !== "all" && node.type !== filterType) continue;
      if ((node.confidence ?? 0) < minConfidence) continue;
      if (query && !`${node.label} ${node.type}`.toLowerCase().includes(query)) continue;
      keep.add(node.id);
    }
    if (focusSelected && selectedNodeId && keep.has(selectedNodeId)) {
      const neighbors = new Set<string>([selectedNodeId]);
      for (const edge of graph.edges) {
        if (edge.source === selectedNodeId && keep.has(edge.target)) neighbors.add(edge.target);
        if (edge.target === selectedNodeId && keep.has(edge.source)) neighbors.add(edge.source);
      }
      for (const id of [...keep]) {
        if (!neighbors.has(id)) keep.delete(id);
      }
    }
    return {
      layout: graph.layout,
      nodes: graph.nodes.filter((n) => keep.has(n.id)),
      edges: graph.edges.filter((e) => keep.has(e.source) && keep.has(e.target)),
    };
  }, [workspace, filterQuery, filterType, minConfidence, focusSelected, selectedNodeId]);

  const elements = useCytoscapeGraph(visibleGraph);

  const availableTypes = useMemo(() => {
    const types = new Set<string>();
    for (const node of workspace?.graph.nodes ?? []) types.add(node.type);
    return [...types].sort();
  }, [workspace]);

  const record = workspace?.record;
  const target = workspace?.target;
  const insights = workspace?.insights;

  const breadcrumb = target?.display_name ?? record?.seeds.map((s) => s.value).join(", ") ?? "Scan";

  if (loadError) {
    return (
      <div className="workspace-layout">
        <div style={{ padding: 24 }}>
          <div className="error-banner">{loadError}</div>
          <button className="btn btn-ghost btn-sm" onClick={() => navigate(-1)}><ArrowLeft size={12} /> Back</button>
        </div>
      </div>
    );
  }

  if (!workspace) {
    return (
      <div className="workspace-layout" style={{ alignItems: "center", justifyContent: "center" }}>
        <div className="spinner" />
      </div>
    );
  }

  return (
    <div className={`workspace-layout${isPanelOverlay ? " workspace-layout-overlay" : ""}${isPanelSheet ? " workspace-layout-sheet" : ""}`}>
      {/* Top bar */}
      <div className="topbar workspace-topbar">
        <div className="topbar-breadcrumb">
          <span className="topbar-crumb" style={{ cursor: "pointer" }} onClick={() => navigate("/")}>
            Basalt
          </span>
          <span className="topbar-crumb-sep">›</span>
          <span className="topbar-title">{breadcrumb}</span>
        </div>

        {/* Meta stats */}
        <div className="topbar-meta">
          <StatusPill status={record?.status ?? ""} />
          <div className="topbar-meta-item">
            <span className="topbar-meta-value">{record?.node_count ?? 0}</span>
            <span>nodes</span>
          </div>
          <div className="topbar-meta-item">
            <span className="topbar-meta-value">{record?.edge_count ?? 0}</span>
            <span>edges</span>
          </div>
        </div>

        {/* Actions */}
        <div className="topbar-actions">
          {isActive && (
            <button
              className="btn btn-danger btn-sm"
              onClick={handleCancel}
              disabled={cancelLoading}
            >
              {cancelLoading ? "Canceling…" : "Cancel Scan"}
            </button>
          )}
          <button className="btn btn-ghost btn-sm" onClick={() => handleExport("json")}>
            Export JSON
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => handleExport("csv")}>
            Export CSV
          </button>
          <button className="btn btn-ghost btn-sm" onClick={handleExportPNG}>
            PNG
          </button>
          <button
            className="btn btn-ghost btn-sm btn-icon"
            onClick={() => setPanelHidden((h) => !h)}
            title={panelHidden ? "Show panel" : "Hide panel"}
          >
            {panelHidden ? <PanelRight size={14} /> : <PanelRightClose size={14} />}
          </button>
        </div>
      </div>

      {/* Workspace body */}
        <div className="workspace-body">
        {/* Graph canvas */}
        <div className="graph-canvas">
          {/* Slim progress bar keeps the graph inspectable while a scan runs */}
          {isActive && (
            <div className="graph-progress" role="status">
              <div className="spinner spinner-sm" />
              <span>{isConnected ? "Scanning, graph updates live" : "Connecting"}</span>
              <span className="mono">{events.length} events</span>
            </div>
          )}
          <div className="graph-filterbar">
            <input
              type="text"
              className="graph-filter-input"
              placeholder="Filter nodes"
              value={filterQuery}
              onChange={(e) => setFilterQuery(e.target.value)}
            />
            <select
              className="graph-filter-select"
              value={filterType}
              onChange={(e) => setFilterType(e.target.value)}
              title="Node type"
            >
              <option value="all">All types</option>
              {availableTypes.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
            <label className="graph-filter-conf" title="Minimum confidence">
              <span>≥{Math.round(minConfidence * 100)}%</span>
              <input
                type="range"
                min={0}
                max={0.9}
                step={0.1}
                value={minConfidence}
                onChange={(e) => setMinConfidence(Number(e.target.value))}
              />
            </label>
            <button
              type="button"
              className={`btn btn-ghost btn-sm${focusSelected ? " active" : ""}`}
              onClick={() => setFocusSelected((f) => !f)}
              disabled={!selectedNodeId}
              title="Show only the selected node and its direct connections"
            >
              Focus
            </button>
          </div>
          <CytoscapeGraph
            ref={graphRef}
            elements={elements}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            resizeKey={`${panelHidden}-${isPanelOverlay}-${isPanelSheet}`}
          />

          {/* Graph stats */}
          {!isActive && workspace.raw_node_count > 0 && (
            <div className="graph-stats-bar">
              <div className="graph-stat-chip">
                Showing <strong>{visibleGraph.nodes.length}</strong> of <strong>{workspace.graph.nodes.length}</strong> nodes
              </div>
              <div className="graph-stat-chip">
                <strong>{workspace.raw_node_count}</strong> raw evidence nodes
              </div>
            </div>
          )}
          <div className="graph-legend">
            <span className="legend-item legend-root">Target</span>
            <span className="legend-item legend-seed">Seed</span>
            <span className="legend-item legend-account">Account</span>
            <span className="legend-item legend-email">Email</span>
            <span className="legend-item legend-domain">Domain</span>
            <span className="legend-item legend-website">Website</span>
            <span className="legend-item legend-username">Username</span>
          </div>
        </div>

        {!panelHidden && isPanelOverlay && (
          <button
            type="button"
            className="workspace-panel-backdrop"
            aria-label="Close panel"
            onClick={() => setPanelHidden(true)}
          />
        )}

        {/* Right panel */}
        <div className={`right-panel${panelHidden ? " hidden" : ""}${isPanelOverlay ? " overlay" : ""}${isPanelSheet ? " sheet" : ""}`}>
          <div className="panel-tabs">
            {(["insights", "inspector", "events", "raw"] as PanelTab[]).map((tab) => (
              <button
                key={tab}
                className={`panel-tab${activeTab === tab ? " active" : ""}`}
                onClick={() => setActiveTab(tab)}
              >
                {tab.charAt(0).toUpperCase() + tab.slice(1)}
                {tab === "events" && events.length > 0 && (
                  <span style={{ marginLeft: 5, fontFamily: "var(--font-mono)", fontSize: 10, color: "var(--accent)" }}>
                    {events.length}
                  </span>
                )}
              </button>
            ))}
          </div>

          <div className="panel-body">
            {/* Insights tab */}
            {activeTab === "insights" && (
              <>
                {insights?.headline && (
                  <PretextBlock
                    className="insight-headline"
                    text={insights.headline}
                    font={pretextFonts.insightHeadline}
                    lineHeight={lineHeights.body}
                  />
                )}

                {(insights?.top_findings ?? []).length > 0 && (
                  <div>
                    <div className="section-head" style={{ marginBottom: 10 }}>
                      <span className="section-title">Top Findings</span>
                    </div>
                    <div className="flex-col gap-2">
                      {insights!.top_findings!.map((f, i) => (
                        <FindingCard key={i} finding={f} onSelectNode={handleNodeClick} />
                      ))}
                    </div>
                  </div>
                )}

                {(insights?.identity_signals ?? []).length > 0 && (
                  <div>
                    <div className="section-head" style={{ marginBottom: 6 }}>
                      <span className="section-title">Identity Signals</span>
                    </div>
                    <div className="signal-chips">
                      {insights!.identity_signals!.map((s, i) => (
                        <span className="signal-chip" key={i}>{s}</span>
                      ))}
                    </div>
                  </div>
                )}

                {(insights?.warnings ?? []).length > 0 && (
                  <div>
                    <div className="section-head" style={{ marginBottom: 6 }}>
                      <span className="section-title">Warnings</span>
                    </div>
                    <div className="warnings-list">
                      {insights!.warnings!.map((w, i) => (
                        <div className="warning-item" key={i}><AlertTriangle size={13} /> {w}</div>
                      ))}
                    </div>
                  </div>
                )}

                {!insights && !isActive && (
                  <div className="empty-state" style={{ padding: "20px 0" }}>
                    <div className="empty-state-icon"><CircleDashed size={24} /></div>
                    <div className="empty-state-title">No insights</div>
                    <div className="empty-state-desc">Insights are generated after the scan completes.</div>
                  </div>
                )}
              </>
            )}

            {/* Inspector tab */}
            {activeTab === "inspector" && (
              <NodeInspector
                node={selectedNode}
                nodes={workspace.graph.nodes}
                edges={workspace.graph.edges}
                onSelectNode={handleNodeClick}
              />
            )}

            {/* Events tab */}
            {activeTab === "events" && (
              <EventTicker events={events} isConnected={isConnected} />
            )}

            {/* Raw tab */}
            {activeTab === "raw" && (
              <div className="flex-col gap-4">
                <div className="flex-col gap-2">
                  <div className="section-title">Raw Graph</div>
                  <div className="flex gap-3">
                    <div className="card" style={{ flex: 1, padding: "12px 16px", textAlign: "center" }}>
                      <div style={{ fontSize: 22, fontWeight: 700, fontFamily: "var(--font-mono)", color: "var(--text-primary)" }}>
                        {workspace.raw_node_count}
                      </div>
                      <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 2 }}>Nodes</div>
                    </div>
                    <div className="card" style={{ flex: 1, padding: "12px 16px", textAlign: "center" }}>
                      <div style={{ fontSize: 22, fontWeight: 700, fontFamily: "var(--font-mono)", color: "var(--text-primary)" }}>
                        {workspace.raw_edge_count}
                      </div>
                      <div style={{ fontSize: 11, color: "var(--text-muted)", marginTop: 2 }}>Edges</div>
                    </div>
                  </div>
                  <span className="form-hint">Exports live in the top bar. The raw graph is available through the JSON export.</span>
                </div>

                {record?.error_message && (
                  <div className="error-banner">{record.error_message}</div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
