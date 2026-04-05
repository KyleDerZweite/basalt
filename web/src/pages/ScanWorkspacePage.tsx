import { useCallback, useEffect, useRef, useState } from "react";
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

  // Auto-switch to events tab when scan starts
  useEffect(() => {
    if (isActive) setActiveTab("events");
  }, [isActive]);

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

  // Cytoscape elements
  const elements = useCytoscapeGraph(
    workspace?.graph ?? { layout: "", nodes: [], edges: [] }
  );

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
          <CytoscapeGraph
            ref={graphRef}
            elements={elements}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            resizeKey={`${panelHidden}-${isPanelOverlay}-${isPanelSheet}`}
          />

          {/* Active scan overlay */}
          {isActive && (
            <div className="graph-building-overlay">
              <div className="spinner" />
              <div className="build-label">
                {isConnected ? "SCANNING - BUILDING GRAPH" : "CONNECTING…"}
              </div>
            </div>
          )}

          {/* Graph stats */}
          {!isActive && workspace.raw_node_count > 0 && (
            <div className="graph-stats-bar">
              <div className="graph-stat-chip">
                <strong>{workspace.raw_node_count}</strong> raw nodes
              </div>
              <div className="graph-stat-chip">
                <strong>{workspace.raw_edge_count}</strong> raw edges
              </div>
              <div className="graph-stat-chip">
                <strong>{workspace.graph.nodes.length}</strong> synthesized
              </div>
            </div>
          )}
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
              <NodeInspector node={selectedNode} />
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
                </div>

                <div className="flex-col gap-2">
                  <div className="section-title">Export</div>
                  <button className="btn btn-ghost btn-full" onClick={() => handleExport("json")}>
                    Download JSON
                  </button>
                  <button className="btn btn-ghost btn-full" onClick={() => handleExport("csv")}>
                    Download CSV
                  </button>
                  {workspace.raw_graph_available && (
                    <button className="btn btn-ghost btn-full" onClick={handleExportPNG}>
                      Export Graph PNG
                    </button>
                  )}
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
