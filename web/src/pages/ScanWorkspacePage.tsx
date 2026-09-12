import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, AlertTriangle, CircleDashed, ExternalLink, PanelRight, PanelRightClose } from "lucide-react";

import { EventTicker } from "../components/EventTicker";
import { FindingCard } from "../components/FindingCard";
import { NodeInspector } from "../components/NodeInspector";
import { PretextBlock } from "../components/PretextBlock";
import { StatusPill } from "../components/StatusPill";
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

type PanelTab = "inspector" | "events" | "raw";

/** Findings carry raw graph IDs while the workspace uses synthesized IDs. */
function resolveWorkspaceNode(nodes: WorkspaceNode[], id: string): WorkspaceNode | null {
  return (
    nodes.find(
      (n) => n.id === id || n.id === `raw:${id}` || (n.raw_node_ids ?? []).includes(id)
    ) ?? null
  );
}

function resolveByLabel(nodes: WorkspaceNode[], label: string): WorkspaceNode | null {
  return nodes.find((n) => n.label === label) ?? null;
}

export function ScanWorkspacePage({ onRefreshHome }: ScanWorkspacePageProps) {
  const { scanID } = useParams<{ scanID: string }>();
  const navigate = useNavigate();

  const [workspace, setWorkspace] = useState<ScanWorkspace | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState("");
  const [activeTab, setActiveTab] = useState<PanelTab>("inspector");
  const [panelHidden, setPanelHidden] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [cancelLoading, setCancelLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const isPanelOverlay = useMediaQuery("(max-width: 1199px)");
  const isPanelSheet = useMediaQuery("(max-width: 767px)");

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

  const nodes = useMemo(() => workspace?.graph.nodes ?? [], [workspace]);
  const edges = useMemo(() => workspace?.graph.edges ?? [], [workspace]);

  // Node selection: findings use raw IDs, lists use workspace IDs.
  const selectedNode: WorkspaceNode | null = selectedNodeId
    ? (resolveWorkspaceNode(nodes, selectedNodeId) ?? null)
    : null;

  const handleNodeClick = useCallback(
    (id: string) => {
      const resolved = resolveWorkspaceNode(nodes, id);
      setSelectedNodeId(resolved ? resolved.id : id);
      if (id) setActiveTab("inspector");
    },
    [nodes]
  );

  const handleLabelClick = useCallback(
    (label: string) => {
      const resolved = resolveByLabel(nodes, label);
      if (resolved) {
        setSelectedNodeId(resolved.id);
        setActiveTab("inspector");
      }
    },
    [nodes]
  );

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

  const query = searchQuery.trim().toLowerCase();
  const matchesQuery = useCallback(
    (text: string) => !query || text.toLowerCase().includes(query),
    [query]
  );

  const record = workspace?.record;
  const target = workspace?.target;
  const insights = workspace?.insights;

  const accounts = useMemo(
    () =>
      nodes
        .filter((n) => n.category === "accounts" || n.category === "account")
        .filter((n) => matchesQuery(`${n.label} ${(n.source_modules ?? []).join(" ")}`)),
    [nodes, matchesQuery]
  );
  const friends = useMemo(
    () =>
      nodes
        .filter((n) => n.category === "people")
        .filter((n) => matchesQuery(n.label)),
    [nodes, matchesQuery]
  );
  const findings = useMemo(
    () => (insights?.top_findings ?? []).filter((f) => matchesQuery(`${f.title} ${f.summary}`)),
    [insights, matchesQuery]
  );
  const profiles = useMemo(
    () =>
      (insights?.profiles ?? []).filter((p) =>
        matchesQuery(
          `${p.primary_handle} ${(p.usernames ?? []).join(" ")} ${(p.emails ?? []).join(" ")}`
        )
      ),
    [insights, matchesQuery]
  );

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
        {/* Evidence column */}
        <div className="evidence-scroll">
          {isActive && (
            <div className="evidence-progress" role="status">
              <div className="spinner spinner-sm" />
              <span>{isConnected ? "Scanning, results update live" : "Connecting"}</span>
              <span className="mono">{events.length} events</span>
            </div>
          )}

          <div className="evidence-searchrow">
            <input
              type="text"
              placeholder="Search findings, accounts, links"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {insights?.headline && (
            <PretextBlock
              className="insight-headline"
              text={insights.headline}
              font={pretextFonts.insightHeadline}
              lineHeight={lineHeights.body}
            />
          )}

                    {profiles.length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">People</span>
                <span className="mono" style={{ fontSize: 11, color: "var(--text-muted)" }}>
                  {profiles.length}
                </span>
              </div>
              <div className="flex-col gap-2">
                {profiles.map((profile) => (
                  <button
                    key={profile.id}
                    type="button"
                    className="profile-card"
                    onClick={() => {
                      const id = profile.node_ids?.[0];
                      if (id) handleNodeClick(id);
                    }}
                  >
                    {(profile.avatar_urls ?? [])[0] && (
                      <img
                        src={(profile.avatar_urls ?? [])[0]}
                        alt=""
                        className="profile-avatar"
                        loading="lazy"
                        referrerPolicy="no-referrer"
                        onError={(e) => {
                          e.currentTarget.style.display = "none";
                        }}
                      />
                    )}
                    <span className="profile-body">
                      <span className="profile-head">
                        <strong>{profile.primary_handle}</strong>
                        <span className="mono">{Math.round(profile.confidence * 100)}%</span>
                      </span>
                      {(profile.reasons ?? []).length > 0 && (
                        <span className="profile-reasons">{(profile.reasons ?? []).join(" · ")}</span>
                      )}
                      <span className="profile-chips">
                        {(profile.usernames ?? []).map((u) => (
                          <span className="signal-chip" key={u}>{u}</span>
                        ))}
                        {(profile.emails ?? []).map((e) => (
                          <span className="signal-chip" key={e}>{e}</span>
                        ))}
                        {(profile.modules ?? []).map((m) => (
                          <span className="type-badge" key={m}>{m}</span>
                        ))}
                      </span>
                    </span>
                  </button>
                ))}
              </div>
            </section>
          )}

          {findings.length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Top Findings</span>
              </div>              <div className="flex-col gap-2">
                {findings.map((f, i) => (
                  <FindingCard key={i} finding={f} onSelectNode={handleNodeClick} />
                ))}
              </div>
            </section>
          )}

          {(insights?.linked_identities ?? []).length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Linked Identities</span>
              </div>
              <div className="flex-col gap-2">
                {(insights?.linked_identities ?? []).map((link) => (
                  <div className="link-card" key={`${link.identifier_type}:${link.identifier}`}>
                    <div className="link-card-head">
                      <span className="link-identifier">{link.identifier}</span>
                      <span className="type-badge">{link.identifier_type}</span>
                    </div>
                    <div className="link-accounts">
                      {link.account_labels.map((label) => (
                        <button
                          key={label}
                          type="button"
                          className="link-account-chip"
                          onClick={() => handleLabelClick(label)}
                        >
                          {label}
                        </button>
                      ))}
                    </div>
                    {(link.modules ?? []).length > 0 && (
                      <div className="link-modules mono">via {(link.modules ?? []).join(", ")}</div>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )}

          {(insights?.possible_matches ?? []).length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Possible Matches</span>
              </div>
              <div className="flex-col gap-2">
                {(insights?.possible_matches ?? []).map((match, i) => (
                  <button
                    key={i}
                    type="button"
                    className="match-row"
                    onClick={() => {
                      const id = match.node_ids?.[0];
                      if (id) handleNodeClick(id);
                    }}
                  >
                    <span className="match-labels">
                      <strong>{match.label_a}</strong>
                      <span className="match-sep">≈</span>
                      <strong>{match.label_b}</strong>
                    </span>
                    <span className="match-reason">
                      {match.reason}
                      {match.confidence != null && ` · ${Math.round(match.confidence * 100)}%`}
                    </span>
                  </button>
                ))}
              </div>
            </section>
          )}

          {accounts.length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Accounts</span>
                <span className="mono" style={{ fontSize: 11, color: "var(--text-muted)" }}>
                  {accounts.length}
                </span>
              </div>
              <div className="flex-col gap-1">
                {accounts.map((node) => (
                  <button
                    key={node.id}
                    type="button"
                    className={`evidence-row${selectedNodeId === node.id ? " selected" : ""}`}
                    onClick={() => handleNodeClick(node.id)}
                  >
                    <span className="evidence-main">{node.label}</span>
                    <span className="evidence-sub">
                      {(node.source_modules ?? []).join(", ")}
                      {node.confidence != null && ` · ${Math.round(node.confidence * 100)}%`}
                    </span>
                    {node.profile_url && (
                      <a
                        href={node.profile_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="finding-link"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <ExternalLink size={11} />
                      </a>
                    )}
                  </button>
                ))}
              </div>
            </section>
          )}

          {friends.length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Friends &amp; Follows</span>
                <span className="mono" style={{ fontSize: 11, color: "var(--text-muted)" }}>
                  {friends.length}
                </span>
              </div>
              <div className="flex-col gap-1">
                {friends.map((node) => (
                  <button
                    key={node.id}
                    type="button"
                    className={`evidence-row${selectedNodeId === node.id ? " selected" : ""}`}
                    onClick={() => handleNodeClick(node.id)}
                  >
                    <span className="evidence-main">{node.label}</span>
                    <span className="evidence-sub">
                      {node.properties?.relationship ?? "connected"}
                      {node.properties?.platform_hint ? ` · ${node.properties.platform_hint}` : ""}
                    </span>
                  </button>
                ))}
              </div>
            </section>
          )}

          {(insights?.identity_signals ?? []).length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Identity Signals</span>
              </div>
              <div className="signal-chips">
                {(insights?.identity_signals ?? []).map((s, i) => (
                  <span className="signal-chip" key={i}>{s}</span>
                ))}
              </div>
            </section>
          )}

          {(insights?.warnings ?? []).length > 0 && (
            <section>
              <div className="section-head">
                <span className="section-title">Warnings</span>
              </div>
              <div className="warnings-list">
                {(insights?.warnings ?? []).map((w, i) => (
                  <div className="warning-item" key={i}><AlertTriangle size={13} /> {w}</div>
                ))}
              </div>
            </section>
          )}

          {!insights && !isActive && (
            <div className="empty-state" style={{ padding: "20px 0" }}>
              <div className="empty-state-icon"><CircleDashed size={24} /></div>
              <div className="empty-state-title">No insights</div>
              <div className="empty-state-desc">Insights are generated after the scan completes.</div>
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
            {(["inspector", "events", "raw"] as PanelTab[]).map((tab) => (
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
            {/* Inspector tab */}
            {activeTab === "inspector" && (
              <NodeInspector
                node={selectedNode}
                nodes={nodes}
                edges={edges}
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
