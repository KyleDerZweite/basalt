import { startTransition, useCallback, useEffect, useEffectEvent, useMemo, useState } from "react";
import { Background, BackgroundVariant, Controls, ReactFlow } from "@xyflow/react";
import { useParams } from "react-router-dom";

import { EventList } from "../components/EventList";
import { FindingList } from "../components/FindingList";
import { EmptyState } from "../components/EmptyState";
import { PropertyPreview } from "../components/PropertyPreview";
import { RawEdgesTable } from "../components/RawEdgesTable";
import { RawNodesTable } from "../components/RawNodesTable";
import { SectionHeading } from "../components/SectionHeading";
import { StatusPill } from "../components/StatusPill";
import { SummaryListItem } from "../components/SummaryListItem";
import { WorkspaceGraphNode } from "../components/WorkspaceGraphNode";
import { useFlowGraph } from "../hooks/useFlowGraph";
import { api } from "../lib/api";
import { scanEventTypes } from "../lib/constants";
import { asMessage, formatDate, formatNodeCategory, formatNodeType } from "../lib/format";
import type { InsightFinding, ScanEvent, ScanWorkspace } from "../types";

const nodeTypes = { workspaceNode: WorkspaceGraphNode };

type ScanWorkspacePageProps = {
  onRefreshHome: () => Promise<void>;
};

export function ScanWorkspacePage({ onRefreshHome }: ScanWorkspacePageProps) {
  const { scanID = "" } = useParams();
  const [workspace, setWorkspace] = useState<ScanWorkspace | null>(null);
  const [events, setEvents] = useState<ScanEvent[]>([]);
  const [selectedNode, setSelectedNode] = useState("");
  const [activeTab, setActiveTab] = useState<"findings" | "nodes" | "edges" | "events">("findings");
  const [localError, setLocalError] = useState("");

  const loadWorkspace = useCallback(async () => {
    if (!scanID) {
      return;
    }

    try {
      const [nextWorkspace, nextEvents] = await Promise.all([
        api<ScanWorkspace>(`/api/scans/${encodeURIComponent(scanID)}/workspace`),
        api<{ events: ScanEvent[] }>(`/api/scans/${encodeURIComponent(scanID)}/events`),
      ]);
      startTransition(() => {
        setWorkspace(nextWorkspace);
        setEvents(nextEvents.events);
        setSelectedNode((current) => {
          if (current && nextWorkspace.graph.nodes.some((node) => node.id === current)) {
            return current;
          }
          return nextWorkspace.graph.nodes[0]?.id ?? "";
        });
        setLocalError("");
      });
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }, [scanID]);

  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);

  const refreshStream = useEffectEvent(() => {
    void loadWorkspace();
    void onRefreshHome();
  });

  useEffect(() => {
    if (!workspace || !["queued", "verifying", "running"].includes(workspace.record.status)) {
      return;
    }

    const after = events.length > 0 ? events[events.length - 1].sequence : 0;
    const source = new EventSource(`/api/scans/${encodeURIComponent(scanID)}/events?stream=1&after=${after}`);

    for (const eventType of scanEventTypes) {
      source.addEventListener(eventType, refreshStream as EventListener);
    }

    source.onerror = () => source.close();

    return () => {
      for (const eventType of scanEventTypes) {
        source.removeEventListener(eventType, refreshStream as EventListener);
      }
      source.close();
    };
  }, [events, refreshStream, scanID, workspace]);

  const workspaceNodeByRawID = useMemo(() => {
    const index = new Map<string, string>();
    for (const node of workspace?.graph.nodes ?? []) {
      for (const rawID of node.raw_node_ids ?? []) {
        if (!index.has(rawID)) {
          index.set(rawID, node.id);
        }
      }
    }
    return index;
  }, [workspace]);

  const selectedWorkspaceNode = workspace?.graph.nodes.find((node) => node.id === selectedNode) ?? null;
  const selectedRawNodes = (workspace?.record.graph?.nodes ?? []).filter((node) => selectedWorkspaceNode?.raw_node_ids?.includes(node.id));
  const selectedRawNodeIDs = new Set(selectedRawNodes.map((node) => node.id));
  const selectedRawEdges = (workspace?.record.graph?.edges ?? []).filter(
    (edge) => selectedRawNodeIDs.has(edge.source) || selectedRawNodeIDs.has(edge.target),
  );
  const flowGraph = useFlowGraph(workspace?.graph ?? { layout: "mindmap", nodes: [], edges: [] }, selectedNode);

  async function cancelScan() {
    if (!workspace) {
      return;
    }

    try {
      await api(`/api/scans/${encodeURIComponent(workspace.record.id)}/cancel`, { method: "POST" });
      await loadWorkspace();
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  function jumpToFinding(finding: InsightFinding) {
    const rawID = finding.node_ids?.[0];
    if (!rawID) {
      return;
    }

    const workspaceID = workspaceNodeByRawID.get(rawID);
    if (workspaceID) {
      startTransition(() => {
        setSelectedNode(workspaceID);
      });
    }
  }

  if (!workspace) {
    return (
      <section className="page">
        {localError ? <div className="error-banner">{localError}</div> : null}
        <EmptyState title="Loading workspace" detail="Basalt is assembling the investigation graph, findings, and event stream." />
      </section>
    );
  }

  const aliasCount = workspace.target?.aliases?.length ?? workspace.record.seeds.length;
  const warningCount = workspace.insights?.warnings?.length ?? 0;
  const topFindings = workspace.insights?.top_findings ?? [];
  const leadFinding = topFindings[0] ?? null;

  return (
    <section className="page workspace-page">
      {localError ? <div className="error-banner">{localError}</div> : null}

      <section className="workspace-banner">
        <div className="workspace-banner-copy">
          <div className="section-kicker">Case workspace</div>
          <h1>{workspace.target?.display_name ?? workspace.record.id}</h1>
          <p>{workspace.insights?.headline ?? `${workspace.raw_node_count} nodes and ${workspace.raw_edge_count} edges`}</p>
          <div className="workspace-meta">
            <StatusPill status={workspace.record.status} />
            <span className="mono">Started {formatDate(workspace.record.started_at)}</span>
            <span className="mono">{workspace.raw_node_count} raw nodes</span>
            <span className="mono">{workspace.raw_edge_count} raw edges</span>
          </div>
          <div className="hero-actions">
            <a className="button secondary" href={`/api/scans/${workspace.record.id}/export?format=json`}>
              Export JSON
            </a>
            <a className="button secondary" href={`/api/scans/${workspace.record.id}/export?format=csv`}>
              Export CSV
            </a>
            {["queued", "verifying", "running"].includes(workspace.record.status) ? (
              <button className="button danger" onClick={() => void cancelScan()} type="button">
                Cancel scan
              </button>
            ) : null}
          </div>
        </div>

        <div className="workspace-signal">
          <div className="mini-label">Lead signal</div>
          <strong>{leadFinding?.title ?? "Awaiting stronger evidence"}</strong>
          <p>{leadFinding?.summary ?? "Top findings will rise here once the scan has enough evidence to synthesize a clearer headline."}</p>
          <dl className="brief-list compact">
            <div>
              <dt>Aliases</dt>
              <dd>{aliasCount}</dd>
            </div>
            <div>
              <dt>Accounts</dt>
              <dd>{workspace.insights?.high_confidence_accounts?.length ?? 0}</dd>
            </div>
            <div>
              <dt>Warnings</dt>
              <dd>{warningCount}</dd>
            </div>
            <div>
              <dt>Events</dt>
              <dd>{events.length}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section className="surface">
        <SectionHeading
          kicker="Lead findings"
          title="High-signal results"
          summary="Select a finding to focus the matching node inside the graph."
        />
        <FindingList findings={topFindings} onSelect={jumpToFinding} />
      </section>

      <div className="workspace-layout">
        <section className="surface graph-stage">
          <SectionHeading
            kicker="Graph"
            title="Investigation map"
            summary="This synthesized graph is the main working surface. Raw evidence and logs stay secondary."
          />
          <div className="graph-canvas">
            <ReactFlow
              nodes={flowGraph.nodes}
              edges={flowGraph.edges}
              fitView
              fitViewOptions={{ padding: 0.16, duration: 450 }}
              nodeTypes={nodeTypes}
              onNodeClick={(_, node) => setSelectedNode(node.id)}
              nodesDraggable={false}
              nodesConnectable={false}
              elementsSelectable
              proOptions={{ hideAttribution: true }}
            >
              <Controls showInteractive={false} />
              <Background gap={22} size={1} variant={BackgroundVariant.Dots} />
            </ReactFlow>
          </div>
        </section>

        <aside className="surface inspector-stage">
          <SectionHeading
            kicker="Inspector"
            title={selectedWorkspaceNode?.label ?? "No node selected"}
            summary="Use the inspector to move from synthesized graph nodes back to the underlying evidence."
          />

          {selectedWorkspaceNode ? (
            <div className="stack">
              <div className="inspector-block">
                <div className="node-headline">
                  <span className={`node-kind kind-${selectedWorkspaceNode.category}`}>{formatNodeCategory(selectedWorkspaceNode.category)}</span>
                  <span className="chip">{formatNodeType(selectedWorkspaceNode.type)}</span>
                </div>
                {typeof selectedWorkspaceNode.confidence === "number" && selectedWorkspaceNode.confidence > 0 ? (
                  <p>Confidence {Math.round(selectedWorkspaceNode.confidence * 100)}%</p>
                ) : null}
                {selectedWorkspaceNode.collapsed_count ? (
                  <p>{selectedWorkspaceNode.collapsed_count} additional items are collapsed behind this synthesized node.</p>
                ) : null}
                {selectedWorkspaceNode.profile_url ? (
                  <a className="inline-link" href={selectedWorkspaceNode.profile_url} rel="noreferrer" target="_blank">
                    Open profile
                  </a>
                ) : null}
              </div>

              <div className="inspector-block">
                <strong>Supporting raw nodes</strong>
                {selectedRawNodes.length === 0 ? (
                  <EmptyState title="No attached raw nodes" detail="This synthesized node does not currently expose raw evidence rows." />
                ) : (
                  <div className="stack dense">
                    {selectedRawNodes.map((node) => (
                      <div className="evidence-row" key={node.id}>
                        <div className="row-title">
                          <strong>{node.label}</strong>
                          <span className="chip">{formatNodeType(node.type)}</span>
                        </div>
                        <p>
                          {node.source_module} · {Math.round(node.confidence * 100)}%
                        </p>
                        <PropertyPreview properties={node.properties} />
                        <details className="raw-details">
                          <summary>Raw payload</summary>
                          <pre>{JSON.stringify(node, null, 2)}</pre>
                        </details>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="inspector-block">
                <strong>Related raw edges</strong>
                {selectedRawEdges.length === 0 ? (
                  <EmptyState title="No matching raw edges" detail="Raw evidence edges will appear here when this node has linked graph relationships." />
                ) : (
                  <div className="stack dense">
                    {selectedRawEdges.slice(0, 8).map((edge) => (
                      <div className="evidence-row" key={edge.id}>
                        <div className="row-title">
                          <strong>{edge.type}</strong>
                          <span className="chip">{edge.source_module}</span>
                        </div>
                        <p className="mono">
                          {edge.source} → {edge.target}
                        </p>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <EmptyState title="Select a node" detail="Choose a node in the graph to inspect the evidence that produced it." />
          )}
        </aside>
      </div>

      <div className="section-grid">
        <section className="surface">
          <SectionHeading
            kicker="Coverage"
            title="Identity, infrastructure, warnings"
            summary="These lists stay concise so you can scan the result shape before dropping to raw evidence."
          />
          <div className="summary-groups">
            <SummaryListItem label="Identity signals" items={workspace.insights?.identity_signals ?? []} />
            <SummaryListItem label="Infrastructure" items={workspace.insights?.infrastructure_summary ?? []} />
            <SummaryListItem label="Warnings" items={workspace.insights?.warnings ?? []} />
          </div>
        </section>

        <section className="surface">
          <SectionHeading
            kicker="Records"
            title="Evidence tables and event stream"
            summary="Raw nodes, edges, and events stay available without competing with the main graph view."
            action={
              <div className="tab-row">
                {(["findings", "nodes", "edges", "events"] as const).map((tab) => (
                  <button
                    className={activeTab === tab ? "tab is-active" : "tab"}
                    key={tab}
                    onClick={() => setActiveTab(tab)}
                    type="button"
                  >
                    {formatNodeType(tab)}
                  </button>
                ))}
              </div>
            }
          />

          {activeTab === "findings" ? <FindingList findings={topFindings} onSelect={jumpToFinding} /> : null}
          {activeTab === "nodes" ? <RawNodesTable nodes={workspace.record.graph?.nodes ?? []} /> : null}
          {activeTab === "edges" ? <RawEdgesTable edges={workspace.record.graph?.edges ?? []} /> : null}
          {activeTab === "events" ? <EventList events={events} /> : null}
        </section>
      </div>
    </section>
  );
}
