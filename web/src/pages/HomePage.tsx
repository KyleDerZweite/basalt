import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight, CircleDashed } from "lucide-react";

import { StatusPill } from "../components/StatusPill";
import { EmptyState } from "../components/EmptyState";
import { ModuleHealthList } from "../components/ModuleHealthList";
import { PretextBlock } from "../components/PretextBlock";
import { formatDate } from "../lib/format";
import { ACTIVE_STATUSES } from "../lib/constants";
import { lineHeights, pretextFonts } from "../lib/typography";
import type { ModuleStatus, ScanRecord, Target } from "../types";

interface HomePageProps {
  scans: ScanRecord[];
  targets: Target[];
  health: ModuleStatus[];
}

export function HomePage({ scans, targets, health }: HomePageProps) {
  const navigate = useNavigate();
  const [quickSeed, setQuickSeed] = useState("");

  const activeScans = scans.filter((s) => (ACTIVE_STATUSES as readonly string[]).includes(s.status));
  const recentScans = scans.slice(0, 12);

  function getScanLabel(scan: ScanRecord): string {
    const target = targets.find((t) => t.id === scan.target_id);
    if (target) return target.display_name;
    if (scan.seeds.length > 0) return scan.seeds.map((s) => s.value).join(", ");
    return "Unknown";
  }

  function handleQuickLaunch(e: React.FormEvent) {
    e.preventDefault();
    if (!quickSeed.trim()) return;
    navigate(`/new?seed=${encodeURIComponent(quickSeed.trim())}`);
  }

  const healthyCount = health.filter((m) => m.status === "healthy").length;

  return (
    <div>
      {/* Page header */}
      <div className="page-header">
        <div className="page-header-kicker">Intelligence Platform</div>
        <PretextBlock
          as="h1"
          className="page-header-title"
          text="Dashboard"
          font={pretextFonts.pageTitle}
          lineHeight={lineHeights.title}
        />
        <PretextBlock
          as="p"
          className="page-header-desc"
          text="Overview of recent investigations and module health."
          font={pretextFonts.pageDescription}
          lineHeight={lineHeights.body}
        />
      </div>

      <div className="home-grid">
        {/* Left: scan list */}
        <div className="flex-col gap-6">
          {/* Active scans highlight */}
          {activeScans.length > 0 && (
            <div className="card">
              <div className="card-header">
                <span className="card-title">Active Scans</span>
                <span className="live-badge">
                  <span className="live-dot" style={{ width: 5, height: 5 }} />
                  Live
                </span>
              </div>
              <div className="card-body-sm">
                <div className="scan-list">
                  {activeScans.map((scan) => (
                    <div
                      key={scan.id}
                      className="scan-row"
                      onClick={() => navigate(`/scans/${scan.id}`)}
                    >
                      <StatusPill status={scan.status} />
                      <span className="scan-row-name">{getScanLabel(scan)}</span>
                      <div className="scan-row-meta">
                        <span title={`${scan.node_count} nodes discovery`}>{scan.node_count}n</span>
                        <span title={`${scan.edge_count} relationships (edges)`}>{scan.edge_count}e</span>
                      </div>
                      <button className="btn btn-ghost btn-sm">Open <ArrowRight size={12} /></button>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Recent scans */}
          <div className="card">
            <div className="card-header">
              <span className="card-title">Recent Scans</span>
              <button className="btn btn-ghost btn-sm" onClick={() => navigate("/new")}>
                New Scan
              </button>
            </div>
            <div className="card-body-sm">
              {recentScans.length === 0 ? (
                <EmptyState
                  icon={<CircleDashed size={24} />}
                  title="No scans yet"
                  desc="Launch your first investigation to get started."
                  action={
                    <button className="btn btn-primary btn-sm" onClick={() => navigate("/new")}>
                      New Scan
                    </button>
                  }
                />
              ) : (
                <div className="scan-list">
                  {recentScans.map((scan) => (
                    <div
                      key={scan.id}
                      className="scan-row"
                      onClick={() => navigate(`/scans/${scan.id}`)}
                    >
                      <StatusPill status={scan.status} />
                      <span className="scan-row-name">{getScanLabel(scan)}</span>
                      <div className="scan-row-meta">
                        <span title={`${scan.node_count} nodes discovery`}>{scan.node_count}n</span>
                        <span title={`${scan.edge_count} relationships (edges)`}>{scan.edge_count}e</span>
                        <span>{formatDate(scan.started_at)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right: sidebar cards */}
        <div className="flex-col gap-4">
          {/* Quick launch */}
          <div className="card">
            <div className="card-header">
              <span className="card-title">Quick Scan</span>
            </div>
            <div className="card-body">
              <form className="quick-launch" onSubmit={handleQuickLaunch}>
                <input
                  type="text"
                  placeholder="username, email, or domain…"
                  value={quickSeed}
                  onChange={(e) => setQuickSeed(e.target.value)}
                />
                <button className="btn btn-primary btn-full" type="submit">
                  Configure Scan <ArrowRight size={14} />
                </button>
              </form>
            </div>
          </div>

          {/* Module health */}
          <div className="card">
            <div className="card-header">
              <span className="card-title">Modules</span>
              <span className="mono" style={{ fontSize: 11 }}>
                {healthyCount}/{health.length}
              </span>
            </div>
            <div className="card-body-sm">
              <ModuleHealthList modules={health} limit={6} />
            </div>
          </div>

          {/* Targets summary */}
          {targets.length > 0 && (
            <div className="card">
              <div className="card-header">
                <span className="card-title">Targets</span>
                <button className="btn btn-ghost btn-sm" onClick={() => navigate("/targets")}>
                  Manage <ArrowRight size={12} />
                </button>
              </div>
              <div className="card-body-sm">
                <div className="scan-list">
                  {targets.slice(0, 5).map((t) => (
                    <div
                      key={t.id}
                      className="scan-row"
                      onClick={() => navigate(`/new?target=${t.slug}`)}
                    >
                      <span className="scan-row-name">{t.display_name}</span>
                      <span style={{ fontSize: 11, color: "var(--text-muted)", fontFamily: "var(--font-mono)" }}>
                        {(t.aliases ?? []).length} alias{(t.aliases ?? []).length !== 1 ? "es" : ""}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
