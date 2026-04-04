import { useMemo } from "react";
import { Link } from "react-router-dom";

import { EmptyState } from "../components/EmptyState";
import { MetricLine } from "../components/MetricLine";
import { SectionHeading } from "../components/SectionHeading";
import { StatusPill } from "../components/StatusPill";
import { formatDate, formatSeed, summarizeHealth } from "../lib/format";
import type { ModuleStatus, ScanRecord, Settings, Target } from "../types";

type HomePageProps = {
  scans: ScanRecord[];
  targets: Target[];
  health: ModuleStatus[];
  settings: Settings | null;
};

export function HomePage({ scans, targets, health, settings }: HomePageProps) {
  const healthCounts = useMemo(() => summarizeHealth(health), [health]);
  const degraded = health.filter((item) => item.status !== "healthy");
  const featuredScan = scans[0] ?? null;
  const activeTarget = targets[0] ?? null;

  return (
    <section className="page page-home">
      <section className="home-hero">
        <div className="hero-copy">
          <div className="section-kicker">Local investigation workspace</div>
          <h1>BASALT</h1>
          <p>
            Graph-first investigations, saved target dossiers, and evidence trails running on localhost without leaving the browser.
          </p>
          <div className="hero-actions">
            <Link className="button" to="/new">
              Start a scan
            </Link>
            <Link className="button secondary" to="/targets">
              Open target dossiers
            </Link>
          </div>
        </div>

        <div className="hero-stage">
          <div className="hero-stage-copy">
            <span className="section-kicker">Live canvas</span>
            <strong>{featuredScan?.insights?.headline ?? "Prime the workspace with a first scan"}</strong>
            <p>
              {featuredScan
                ? `${featuredScan.target_id ? `Target ${featuredScan.target_id}` : "Ad hoc scan"} · ${featuredScan.node_count} nodes · ${featuredScan.edge_count} edges`
                : activeTarget
                  ? `Saved dossier ready: ${activeTarget.display_name}`
                  : "Saved targets, graph layouts, and exports appear here as soon as you launch a case."}
            </p>
          </div>

          <div className="hero-signal-grid" aria-hidden="true">
            <span />
            <span />
            <span />
            <span />
            <span />
            <span />
          </div>

          <dl className="hero-stage-ledger">
            <div>
              <dt>Seeds</dt>
              <dd>{featuredScan ? featuredScan.seeds.map(formatSeed).join(" · ") || "target-only" : "username · email · domain"}</dd>
            </div>
            <div>
              <dt>Coverage</dt>
              <dd>
                {healthCounts.healthy} healthy · {healthCounts.degraded} degraded · {healthCounts.offline} offline
              </dd>
            </div>
            <div>
              <dt>Mode</dt>
              <dd>{settings?.strict_mode ? "strict collection" : "standard collection"}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section className="surface">
        <SectionHeading
          kicker="Operational snapshot"
          title="Current footing"
          summary="Basalt keeps the high-level state readable before you drill down into an active case."
        />
        <div className="metric-grid">
          <MetricLine label="Saved targets" value={targets.length} note="Curated people, aliases, and notes ready for reuse." />
          <MetricLine label="Scan history" value={scans.length} note="Recent investigations that can be reopened instantly." />
          <MetricLine
            label="Module readiness"
            value={`${healthCounts.healthy}/${health.length || 0}`}
            note={`${healthCounts.degraded} degraded · ${healthCounts.offline} offline`}
          />
          <MetricLine
            label="Defaults"
            value={settings?.strict_mode ? "Strict" : "Standard"}
            note={`${(settings?.disabled_modules ?? []).length} disabled modules in saved settings`}
          />
        </div>
      </section>

      <div className="section-grid">
        <section className="surface">
          <SectionHeading
            kicker="Recent scans"
            title="Return to an investigation"
            summary="Open the graph workspace directly from the most recent result set."
            action={
              <Link className="button secondary" to="/new">
                Launch new
              </Link>
            }
          />
          {scans.length === 0 ? (
            <EmptyState title="No scans yet" detail="Start a scan to populate the evidence graph, event stream, and exports." />
          ) : (
            <div className="ledger-list">
              {scans.slice(0, 7).map((scan) => (
                <Link className="ledger-row" key={scan.id} to={`/scans/${scan.id}`}>
                  <div className="ledger-copy">
                    <strong>{scan.insights?.headline ?? `${scan.node_count} nodes · ${scan.edge_count} edges`}</strong>
                    <p>{scan.target_id ? `Target ${scan.target_id}` : "Ad hoc scan"} · {scan.seeds.map(formatSeed).join(" · ") || "no explicit seeds"}</p>
                  </div>
                  <div className="ledger-meta">
                    <StatusPill status={scan.status} />
                    <span className="mono">{formatDate(scan.started_at)}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>

        <section className="surface">
          <SectionHeading
            kicker="Module health"
            title={degraded.length === 0 ? "All modules ready" : "Coverage issues to watch"}
            summary="Health verification is surfaced here so you know whether a thin result set is a data problem or a runtime problem."
          />
          {degraded.length === 0 ? (
            <EmptyState title="Healthy across the board" detail="Every registered module reported healthy status during the last refresh." />
          ) : (
            <div className="status-stack-list">
              {degraded.map((item) => (
                <div className="status-row" key={item.name}>
                  <div>
                    <strong>{item.name}</strong>
                    <p>{item.message}</p>
                  </div>
                  <StatusPill status={item.status} />
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      <section className="surface">
        <SectionHeading
          kicker="Target dossiers"
          title="Saved people, handles, and domains"
          summary="Use persistent targets when you want repeatable scans and a cleaner alias ledger."
          action={
            <Link className="button secondary" to="/targets">
              Manage dossiers
            </Link>
          }
        />
        {targets.length === 0 ? (
          <EmptyState title="No targets saved" detail="Create a dossier to keep aliases together and launch repeatable scans from one place." />
        ) : (
          <div className="directory-list">
            {targets.slice(0, 6).map((target) => (
              <div className="directory-row" key={target.id}>
                <div>
                  <strong>{target.display_name}</strong>
                  <p>{target.notes || "No notes saved for this dossier."}</p>
                </div>
                <div className="directory-meta">
                  <span className="chip">{target.slug}</span>
                  <span>{(target.aliases ?? []).length} aliases</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>
    </section>
  );
}
