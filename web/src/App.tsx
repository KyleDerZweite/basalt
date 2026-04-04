import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  Position,
  ReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import ELK, { type ElkNode } from "elkjs/lib/elk.bundled.js";
import { Link, Route, Routes, useLocation, useNavigate, useParams } from "react-router-dom";

type Seed = { type: string; value: string };
type ModuleStatus = { name: string; description: string; status: string; message: string };
type TargetAlias = { id: string; target_id: string; seed_type: string; seed_value: string; label?: string; is_primary?: boolean };
type Target = { id: string; slug: string; display_name: string; notes?: string; aliases?: TargetAlias[] };
type RawNode = {
  id: string;
  type: string;
  label: string;
  source_module: string;
  confidence: number;
  properties?: Record<string, unknown>;
};
type RawEdge = { id: string; source: string; target: string; type: string; source_module: string };
type RawGraph = { nodes: RawNode[]; edges: RawEdge[] };
type InsightFinding = {
  title: string;
  summary: string;
  node_ids?: string[];
  profile_url?: string;
  confidence?: number;
  category?: string;
  source_label?: string;
};
type ScanInsights = {
  headline: string;
  top_findings?: InsightFinding[];
  high_confidence_accounts?: InsightFinding[];
  identity_signals?: string[];
  infrastructure_summary?: string[];
  warnings?: string[];
};
type ScanRecord = {
  id: string;
  target_id?: string;
  status: string;
  started_at: string;
  completed_at?: string;
  updated_at: string;
  seeds: Seed[];
  health?: ModuleStatus[];
  insights?: ScanInsights;
  node_count: number;
  edge_count: number;
  error_message?: string;
  graph?: RawGraph;
};
type WorkspaceNode = {
  id: string;
  label: string;
  type: string;
  category: string;
  raw_node_ids?: string[];
  raw_edge_ids?: string[];
  profile_url?: string;
  confidence?: number;
  collapsed_count?: number;
};
type WorkspaceEdge = { id: string; source: string; target: string; type: string; raw_edge_ids?: string[] };
type ScanWorkspace = {
  record: ScanRecord;
  target?: Target;
  insights?: ScanInsights;
  graph: { layout: string; nodes: WorkspaceNode[]; edges: WorkspaceEdge[] };
  raw_graph_available: boolean;
  raw_node_count: number;
  raw_edge_count: number;
};
type Bootstrap = {
  version: string;
  data_dir: string;
  default_config_path: string;
  api_base_path: string;
  base_url: string;
};
type Settings = { strict_mode: boolean; disabled_modules?: string[]; legal_accepted_at?: string | null };
type ScanEvent = { sequence: number; time: string; type: string; module?: string; message?: string; data?: Record<string, unknown> };
type ThemeMode = "dark" | "light";
type GraphNodeData = {
  label: string;
  category: string;
  type: string;
  confidence?: number;
  collapsedCount?: number;
  profileURL?: string;
};

const elk = new ELK();
const themeStorageKey = "basalt-theme";
const scanEventTypes = [
  "scan_queued",
  "scan_status",
  "module_verified",
  "verify_complete",
  "module_started",
  "module_finished",
  "node_discovered",
  "edge_discovered",
  "module_error",
  "scan_finished",
  "scan_failed",
] as const;
const nodeTypes = { workspaceNode: WorkspaceGraphNode };
const navItems = [
  { to: "/", label: "Home" },
  { to: "/targets", label: "Targets" },
  { to: "/new", label: "New Scan" },
  { to: "/settings", label: "Settings" },
];

export function App() {
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [targets, setTargets] = useState<Target[]>([]);
  const [scans, setScans] = useState<ScanRecord[]>([]);
  const [health, setHealth] = useState<ModuleStatus[]>([]);
  const [error, setError] = useState("");
  const [theme, setTheme] = useState<ThemeMode>(() => {
    if (typeof window === "undefined") {
      return "dark";
    }
    const stored = window.localStorage.getItem(themeStorageKey);
    return stored === "light" ? "light" : "dark";
  });
  const location = useLocation();

  const refreshHome = useCallback(async () => {
    try {
      const [boot, nextSettings, nextTargets, nextScans, nextHealth] = await Promise.all([
        api<Bootstrap>("/app/bootstrap"),
        api<Settings>("/api/settings"),
        api<{ targets: Target[] }>("/api/targets"),
        api<{ scans: ScanRecord[] }>("/api/scans"),
        api<{ modules: ModuleStatus[] }>("/api/modules/health"),
      ]);
      setBootstrap(boot);
      setSettings(nextSettings);
      setTargets(nextTargets.targets);
      setScans(nextScans.scans);
      setHealth(nextHealth.modules);
      setError("");
    } catch (reason) {
      setError(asMessage(reason));
    }
  }, []);

  useEffect(() => {
    void refreshHome();
  }, [refreshHome]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  const pageMeta = useMemo(() => pageTitle(location.pathname), [location.pathname]);

  return (
    <div className="app-shell">
      <aside className="app-rail">
        <div className="rail-block">
          <div className="brand-mark">Basalt</div>
          <div className="brand-copy">
            <h1>Local investigation workspace</h1>
            <p>Graph-first OSINT scans with stored targets, aliases, evidence, and exports.</p>
          </div>
        </div>

        <nav className="rail-nav">
          {navItems.map((item) => {
            const active = location.pathname === item.to || (item.to !== "/" && location.pathname.startsWith(item.to));
            return (
              <Link key={item.to} className={active ? "rail-link active" : "rail-link"} to={item.to}>
                {item.label}
              </Link>
            );
          })}
        </nav>

        <div className="rail-block rail-footer">
          <button className="theme-toggle" type="button" onClick={() => setTheme((current) => (current === "dark" ? "light" : "dark"))}>
            {theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
          </button>
          <div className="meta-list">
            <div>
              <span>Version</span>
              <strong>{bootstrap?.version ?? "unknown"}</strong>
            </div>
            <div>
              <span>Data dir</span>
              <strong>{bootstrap?.data_dir ?? "unknown"}</strong>
            </div>
          </div>
        </div>
      </aside>

      <main className="app-main">
        <header className="topbar">
          <div>
            <div className="eyebrow">{pageMeta.kicker}</div>
            <h2>{pageMeta.title}</h2>
            <p className="muted">{pageMeta.summary}</p>
          </div>
          <div className="topbar-actions">
            <Link className="button secondary" to="/targets">Targets</Link>
            <Link className="button" to="/new">Start scan</Link>
          </div>
        </header>

        {error ? <div className="error-banner">{error}</div> : null}

        <Routes>
          <Route path="/" element={<HomePage scans={scans} targets={targets} health={health} settings={settings} />} />
          <Route path="/targets" element={<TargetsPage targets={targets} onRefresh={refreshHome} />} />
          <Route path="/new" element={<NewScanPage targets={targets} settings={settings} onCreated={refreshHome} />} />
          <Route path="/settings" element={<SettingsPage settings={settings} bootstrap={bootstrap} onRefresh={refreshHome} theme={theme} />} />
          <Route path="/scans/:scanID" element={<ScanWorkspacePage onRefreshHome={refreshHome} />} />
        </Routes>
      </main>
    </div>
  );
}

function HomePage({
  scans,
  targets,
  health,
  settings,
}: {
  scans: ScanRecord[];
  targets: Target[];
  health: ModuleStatus[];
  settings: Settings | null;
}) {
  const healthCounts = useMemo(() => summarizeHealth(health), [health]);
  const degraded = health.filter((item) => item.status !== "healthy");

  return (
    <section className="page">
      <div className="overview-grid">
        <StatCard title="Targets" value={targets.length} subtitle="Persisted people and aliases" />
        <StatCard title="Scans" value={scans.length} subtitle="Recent local investigations" />
        <StatCard title="Healthy modules" value={healthCounts.healthy} subtitle={`${healthCounts.degraded} degraded · ${healthCounts.offline} offline`} />
        <StatCard title="Strict mode" value={settings?.strict_mode ? "On" : "Off"} subtitle={`${(settings?.disabled_modules ?? []).length} modules disabled`} />
      </div>

      <div className="content-grid">
        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Recent scans</div>
              <h3>Open a workspace</h3>
            </div>
          </div>
          {scans.length === 0 ? (
            <div className="empty-state">No scans yet.</div>
          ) : (
            <div className="scan-list">
              {scans.slice(0, 8).map((scan) => (
                <Link className="scan-row" key={scan.id} to={`/scans/${scan.id}`}>
                  <div className="scan-row-main">
                    <strong>{scan.insights?.headline ?? `${scan.node_count} nodes · ${scan.edge_count} edges`}</strong>
                    <span className="muted">{scan.target_id ? `Target ${scan.target_id}` : "Ad hoc scan"}</span>
                  </div>
                  <div className="scan-row-side">
                    <StatusPill status={scan.status} />
                    <span className="mono">{formatDate(scan.started_at)}</span>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>

        <div className="stack-column">
          <section className="panel">
            <div className="section-heading">
              <div>
                <div className="eyebrow">Targets</div>
                <h3>Current alias sets</h3>
              </div>
              <Link className="button secondary" to="/targets">Manage</Link>
            </div>
            {targets.length === 0 ? (
              <div className="empty-state">No targets yet.</div>
            ) : (
              <div className="compact-list">
                {targets.slice(0, 6).map((target) => (
                  <div className="list-item" key={target.id}>
                    <div className="row-title">
                      <strong>{target.display_name}</strong>
                      <span className="chip">{target.slug}</span>
                    </div>
                    <div className="muted">
                      {(target.aliases ?? []).map((alias) => `${alias.seed_type}:${alias.seed_value}`).join(", ") || "No aliases"}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>

          <section className="panel">
            <div className="section-heading">
              <div>
                <div className="eyebrow">Module health</div>
                <h3>Current issues</h3>
              </div>
            </div>
            {degraded.length === 0 ? (
              <div className="empty-state">All modules are healthy.</div>
            ) : (
              <div className="compact-list">
                {degraded.map((item) => (
                  <div className="list-item" key={item.name}>
                    <div className="row-title">
                      <strong>{item.name}</strong>
                      <StatusPill status={item.status} />
                    </div>
                    <div className="muted">{item.message}</div>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>
      </div>
    </section>
  );
}

function TargetsPage({ targets, onRefresh }: { targets: Target[]; onRefresh: () => Promise<void> }) {
  const [displayName, setDisplayName] = useState("");
  const [slug, setSlug] = useState("");
  const [notes, setNotes] = useState("");
  const [selectedTarget, setSelectedTarget] = useState<Target | null>(null);
  const [aliasType, setAliasType] = useState("username");
  const [aliasValue, setAliasValue] = useState("");
  const [aliasLabel, setAliasLabel] = useState("");
  const [aliasPrimary, setAliasPrimary] = useState(false);
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    if (!selectedTarget && targets.length > 0) {
      setSelectedTarget(targets[0]);
    }
    if (selectedTarget) {
      const updated = targets.find((item) => item.id === selectedTarget.id);
      if (updated) {
        setSelectedTarget(updated);
      }
    }
  }, [selectedTarget, targets]);

  async function createTarget(event: React.FormEvent) {
    event.preventDefault();
    try {
      await api("/api/targets", {
        method: "POST",
        body: JSON.stringify({ display_name: displayName, slug, notes }),
      });
      setDisplayName("");
      setSlug("");
      setNotes("");
      setLocalError("");
      await onRefresh();
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  async function addAlias(event: React.FormEvent) {
    event.preventDefault();
    if (!selectedTarget) {
      return;
    }
    try {
      await api(`/api/targets/${encodeURIComponent(selectedTarget.slug)}/aliases`, {
        method: "POST",
        body: JSON.stringify({
          seed_type: aliasType,
          seed_value: aliasValue,
          label: aliasLabel,
          is_primary: aliasPrimary,
        }),
      });
      setAliasValue("");
      setAliasLabel("");
      setAliasPrimary(false);
      setLocalError("");
      await onRefresh();
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  async function removeAlias(aliasID: string) {
    if (!selectedTarget) {
      return;
    }
    try {
      await api(`/api/targets/${encodeURIComponent(selectedTarget.slug)}/aliases/${encodeURIComponent(aliasID)}`, {
        method: "DELETE",
      });
      await onRefresh();
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  return (
    <section className="page">
      {localError ? <div className="error-banner">{localError}</div> : null}
      <div className="content-grid">
        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Create</div>
              <h3>New target</h3>
            </div>
          </div>
          <form className="stack" onSubmit={createTarget}>
            <label className="field">
              <span>Name</span>
              <input value={displayName} onChange={(event) => setDisplayName(event.target.value)} required />
            </label>
            <label className="field">
              <span>Slug</span>
              <input value={slug} onChange={(event) => setSlug(event.target.value)} placeholder="kyle" />
            </label>
            <label className="field">
              <span>Notes</span>
              <textarea value={notes} onChange={(event) => setNotes(event.target.value)} placeholder="Optional notes" />
            </label>
            <button className="button" type="submit">Create target</button>
          </form>
        </section>

        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Directory</div>
              <h3>Existing targets</h3>
            </div>
          </div>
          {targets.length === 0 ? (
            <div className="empty-state">No targets yet.</div>
          ) : (
            <div className="target-grid">
              {targets.map((target) => (
                <button
                  className={selectedTarget?.id === target.id ? "target-card active" : "target-card"}
                  key={target.id}
                  type="button"
                  onClick={() => setSelectedTarget(target)}
                >
                  <strong>{target.display_name}</strong>
                  <span>{target.slug}</span>
                  <small>{(target.aliases ?? []).length} aliases</small>
                </button>
              ))}
            </div>
          )}
        </section>
      </div>

      <section className="panel">
        <div className="section-heading">
          <div>
            <div className="eyebrow">Aliases</div>
            <h3>{selectedTarget ? selectedTarget.display_name : "Select a target"}</h3>
          </div>
          {selectedTarget ? <Link className="button secondary" to="/new">Scan this target</Link> : null}
        </div>

        {selectedTarget ? (
          <div className="content-grid">
            <form className="stack" onSubmit={addAlias}>
              <label className="field">
                <span>Seed type</span>
                <select value={aliasType} onChange={(event) => setAliasType(event.target.value)}>
                  <option value="username">username</option>
                  <option value="email">email</option>
                  <option value="domain">domain</option>
                </select>
              </label>
              <label className="field">
                <span>Seed value</span>
                <input value={aliasValue} onChange={(event) => setAliasValue(event.target.value)} required />
              </label>
              <label className="field">
                <span>Label</span>
                <input value={aliasLabel} onChange={(event) => setAliasLabel(event.target.value)} placeholder="Main handle" />
              </label>
              <label className="checkbox">
                <input checked={aliasPrimary} onChange={(event) => setAliasPrimary(event.target.checked)} type="checkbox" />
                <span>Primary alias</span>
              </label>
              <button className="button" type="submit">Add alias</button>
            </form>

            <div className="stack-column">
              {selectedTarget.notes ? <div className="detail-note">{selectedTarget.notes}</div> : null}
              {(selectedTarget.aliases ?? []).length === 0 ? (
                <div className="empty-state">No aliases yet.</div>
              ) : (
                <div className="compact-list">
                  {(selectedTarget.aliases ?? []).map((alias) => (
                    <div className="list-item" key={alias.id}>
                      <div className="row-title">
                        <strong>{alias.label || alias.seed_value}</strong>
                        {alias.is_primary ? <span className="chip accent">Primary</span> : null}
                      </div>
                      <div className="muted">{alias.seed_type}:{alias.seed_value}</div>
                      <button className="button secondary small" onClick={() => void removeAlias(alias.id)} type="button">
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="empty-state">Select a target to manage aliases.</div>
        )}
      </section>
    </section>
  );
}

function NewScanPage({
  targets,
  settings,
  onCreated,
}: {
  targets: Target[];
  settings: Settings | null;
  onCreated: () => Promise<void>;
}) {
  const navigate = useNavigate();
  const [targetRef, setTargetRef] = useState("");
  const [seeds, setSeeds] = useState<Seed[]>([{ type: "username", value: "" }]);
  const [depth, setDepth] = useState(2);
  const [concurrency, setConcurrency] = useState(5);
  const [timeoutSeconds, setTimeoutSeconds] = useState(10);
  const [strictMode, setStrictMode] = useState(settings?.strict_mode ?? false);
  const [disabledModules, setDisabledModules] = useState((settings?.disabled_modules ?? []).join(", "));
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    setStrictMode(settings?.strict_mode ?? false);
    setDisabledModules((settings?.disabled_modules ?? []).join(", "));
  }, [settings]);

  const selectedTarget = useMemo(
    () => targets.find((target) => target.slug === targetRef) ?? null,
    [targetRef, targets],
  );

  async function createScan(event: React.FormEvent) {
    event.preventDefault();
    try {
      const payload = {
        target_ref: targetRef || undefined,
        seeds: seeds.filter((seed) => seed.value.trim() !== ""),
        depth,
        concurrency,
        timeout_seconds: timeoutSeconds,
        strict_mode: strictMode,
        disabled_modules: splitCommaList(disabledModules),
      };
      const created = await api<ScanRecord>("/api/scans", {
        method: "POST",
        body: JSON.stringify(payload),
      });
      await onCreated();
      navigate(`/scans/${created.id}`);
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  return (
    <section className="page">
      {localError ? <div className="error-banner">{localError}</div> : null}
      <form className="stack" onSubmit={createScan}>
        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Scope</div>
              <h3>Target and seeds</h3>
            </div>
          </div>
          <div className="content-grid">
            <div className="stack-column">
              <label className="field">
                <span>Target</span>
                <select value={targetRef} onChange={(event) => setTargetRef(event.target.value)}>
                  <option value="">No target</option>
                  {targets.map((target) => (
                    <option key={target.id} value={target.slug}>{target.display_name}</option>
                  ))}
                </select>
              </label>
              <div className="target-preview">
                <strong>Active target aliases</strong>
                <div className="muted">
                  {selectedTarget
                    ? (selectedTarget.aliases ?? []).map((alias) => `${alias.seed_type}:${alias.seed_value}`).join(", ") || "No aliases stored"
                    : "This scan will use only the explicit seeds below."}
                </div>
              </div>
            </div>

            <div className="stack-column">
              <div className="section-heading compact">
                <h4>Extra seeds</h4>
                <button className="button secondary small" type="button" onClick={() => setSeeds((current) => [...current, { type: "username", value: "" }])}>
                  Add seed
                </button>
              </div>
              <div className="stack">
                {seeds.map((seed, index) => (
                  <div className="seed-row" key={`${seed.type}-${index}`}>
                    <select value={seed.type} onChange={(event) => updateSeed(seeds, setSeeds, index, { type: event.target.value })}>
                      <option value="username">username</option>
                      <option value="email">email</option>
                      <option value="domain">domain</option>
                    </select>
                    <input value={seed.value} onChange={(event) => updateSeed(seeds, setSeeds, index, { value: event.target.value })} placeholder="identifier" />
                    <button className="button secondary small" type="button" onClick={() => setSeeds((current) => current.filter((_, itemIndex) => itemIndex !== index))}>
                      Remove
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Execution</div>
              <h3>Runtime settings</h3>
            </div>
          </div>
          <div className="config-grid">
            <label className="field">
              <span>Depth</span>
              <input type="number" min={1} value={depth} onChange={(event) => setDepth(Number(event.target.value) || 1)} />
            </label>
            <label className="field">
              <span>Concurrency</span>
              <input type="number" min={1} value={concurrency} onChange={(event) => setConcurrency(Number(event.target.value) || 1)} />
            </label>
            <label className="field">
              <span>Timeout</span>
              <input type="number" min={1} value={timeoutSeconds} onChange={(event) => setTimeoutSeconds(Number(event.target.value) || 1)} />
            </label>
          </div>
          <label className="field">
            <span>Disabled modules</span>
            <textarea value={disabledModules} onChange={(event) => setDisabledModules(event.target.value)} placeholder="github,reddit" />
          </label>
          <label className="checkbox">
            <input type="checkbox" checked={strictMode} onChange={(event) => setStrictMode(event.target.checked)} />
            <span>Strict mode</span>
          </label>
          <div className="topbar-actions">
            <button className="button" type="submit">Start scan</button>
            <Link className="button secondary" to="/targets">Manage targets</Link>
          </div>
        </section>
      </form>
    </section>
  );
}

function SettingsPage({
  settings,
  bootstrap,
  onRefresh,
  theme,
}: {
  settings: Settings | null;
  bootstrap: Bootstrap | null;
  onRefresh: () => Promise<void>;
  theme: ThemeMode;
}) {
  const [strictMode, setStrictMode] = useState(settings?.strict_mode ?? false);
  const [disabledModules, setDisabledModules] = useState((settings?.disabled_modules ?? []).join(", "));
  const [legalAcceptedAt, setLegalAcceptedAt] = useState(settings?.legal_accepted_at ?? "");
  const [localError, setLocalError] = useState("");

  useEffect(() => {
    setStrictMode(settings?.strict_mode ?? false);
    setDisabledModules((settings?.disabled_modules ?? []).join(", "));
    setLegalAcceptedAt(settings?.legal_accepted_at ?? "");
  }, [settings]);

  async function save(event: React.FormEvent) {
    event.preventDefault();
    try {
      await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify({
          strict_mode: strictMode,
          disabled_modules: splitCommaList(disabledModules),
          legal_accepted_at: legalAcceptedAt || null,
        }),
      });
      setLocalError("");
      await onRefresh();
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }

  return (
    <section className="page">
      {localError ? <div className="error-banner">{localError}</div> : null}
      <div className="content-grid">
        <form className="panel stack" onSubmit={save}>
          <div className="section-heading">
            <div>
              <div className="eyebrow">Defaults</div>
              <h3>Scan settings</h3>
            </div>
          </div>
          <label className="checkbox">
            <input type="checkbox" checked={strictMode} onChange={(event) => setStrictMode(event.target.checked)} />
            <span>Strict mode</span>
          </label>
          <label className="field">
            <span>Disabled modules</span>
            <textarea value={disabledModules} onChange={(event) => setDisabledModules(event.target.value)} />
          </label>
          <label className="field">
            <span>Legal accepted at</span>
            <input value={legalAcceptedAt} onChange={(event) => setLegalAcceptedAt(event.target.value)} placeholder="2026-04-04T12:00:00Z" />
          </label>
          <button className="button" type="submit">Save settings</button>
        </form>

        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Runtime</div>
              <h3>Environment</h3>
            </div>
          </div>
          <div className="compact-list">
            <div className="list-item">
              <div className="row-title"><strong>Theme</strong><span className="chip">{theme}</span></div>
              <div className="muted">Stored in the browser. Default is dark mode.</div>
            </div>
            <div className="list-item">
              <strong>Version</strong>
              <div className="muted mono">{bootstrap?.version ?? "unknown"}</div>
            </div>
            <div className="list-item">
              <strong>Data directory</strong>
              <div className="muted mono">{bootstrap?.data_dir ?? "unknown"}</div>
            </div>
            <div className="list-item">
              <strong>Config path</strong>
              <div className="muted mono">{bootstrap?.default_config_path ?? "unknown"}</div>
            </div>
          </div>
        </section>
      </div>
    </section>
  );
}

function ScanWorkspacePage({ onRefreshHome }: { onRefreshHome: () => Promise<void> }) {
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
      setWorkspace(nextWorkspace);
      setEvents(nextEvents.events);
      setSelectedNode((current) => {
        if (current && nextWorkspace.graph.nodes.some((node) => node.id === current)) {
          return current;
        }
        return nextWorkspace.graph.nodes[0]?.id ?? "";
      });
      setLocalError("");
    } catch (reason) {
      setLocalError(asMessage(reason));
    }
  }, [scanID]);

  useEffect(() => {
    void loadWorkspace();
  }, [loadWorkspace]);

  useEffect(() => {
    if (!workspace || !["queued", "verifying", "running"].includes(workspace.record.status)) {
      return;
    }
    const after = events.length > 0 ? events[events.length - 1].sequence : 0;
    const source = new EventSource(`/api/scans/${encodeURIComponent(scanID)}/events?stream=1&after=${after}`);
    const handleRefresh = () => {
      void loadWorkspace();
      void onRefreshHome();
    };
    for (const eventType of scanEventTypes) {
      source.addEventListener(eventType, handleRefresh);
    }
    source.onerror = () => source.close();
    return () => {
      for (const eventType of scanEventTypes) {
        source.removeEventListener(eventType, handleRefresh);
      }
      source.close();
    };
  }, [events, loadWorkspace, onRefreshHome, scanID, workspace]);

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
      setSelectedNode(workspaceID);
    }
  }

  if (!workspace) {
    return (
      <section className="page">
        {localError ? <div className="error-banner">{localError}</div> : null}
        <div className="empty-state">Loading workspace...</div>
      </section>
    );
  }

  const aliasCount = workspace.target?.aliases?.length ?? workspace.record.seeds.length;
  const warningCount = workspace.insights?.warnings?.length ?? 0;
  const topFindings = workspace.insights?.top_findings ?? [];

  return (
    <section className="page workspace-page">
      {localError ? <div className="error-banner">{localError}</div> : null}

      <section className="workspace-hero panel">
        <div className="hero-copy">
          <div className="eyebrow">Scan workspace</div>
          <h3>{workspace.target?.display_name ?? workspace.record.id}</h3>
          <p>{workspace.insights?.headline ?? `${workspace.raw_node_count} nodes and ${workspace.raw_edge_count} edges`}</p>
          <div className="hero-meta">
            <StatusPill status={workspace.record.status} />
            <span className="mono">Started {formatDate(workspace.record.started_at)}</span>
            <span className="mono">{workspace.raw_node_count} nodes</span>
            <span className="mono">{workspace.raw_edge_count} edges</span>
          </div>
        </div>
        <div className="hero-actions">
          <a className="button secondary" href={`/api/scans/${workspace.record.id}/export?format=json`}>Export JSON</a>
          <a className="button secondary" href={`/api/scans/${workspace.record.id}/export?format=csv`}>Export CSV</a>
          {["queued", "verifying", "running"].includes(workspace.record.status) ? (
            <button className="button danger" onClick={() => void cancelScan()} type="button">Cancel</button>
          ) : null}
        </div>
      </section>

      <div className="workspace-summary">
        <StatCard title="Aliases" value={aliasCount} subtitle="Curated identifiers in scope" />
        <StatCard title="Accounts" value={workspace.insights?.high_confidence_accounts?.length ?? 0} subtitle="High-confidence account matches" />
        <StatCard title="Warnings" value={warningCount} subtitle="Module issues and partial coverage" />
        <StatCard title="Events" value={events.length} subtitle="Progress and evidence trail" />
      </div>

      <div className="insight-row">
        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Top findings</div>
              <h3>Important results first</h3>
            </div>
          </div>
          {topFindings.length === 0 ? (
            <div className="empty-state">No high-signal findings yet.</div>
          ) : (
            <div className="finding-grid">
              {topFindings.slice(0, 4).map((finding, index) => (
                <button className="finding-card" key={`${finding.title}-${index}`} onClick={() => jumpToFinding(finding)} type="button">
                  <span className="chip">{finding.category ?? "finding"}</span>
                  <strong>{finding.title}</strong>
                  <p>{finding.summary}</p>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="panel">
          <div className="section-heading">
            <div>
              <div className="eyebrow">Coverage</div>
              <h3>Identity and warnings</h3>
            </div>
          </div>
          <div className="compact-list">
            <SummaryListItem label="Identity signals" items={workspace.insights?.identity_signals ?? []} />
            <SummaryListItem label="Infrastructure" items={workspace.insights?.infrastructure_summary ?? []} />
            <SummaryListItem label="Warnings" items={workspace.insights?.warnings ?? []} />
          </div>
        </section>
      </div>

      <div className="workspace-layout">
        <section className="panel graph-panel">
          <div className="section-heading compact">
            <div>
              <div className="eyebrow">Graph</div>
              <h3>Investigation map</h3>
            </div>
            <div className="muted">Graph is the main view. Raw nodes, edges, and events stay below.</div>
          </div>

          <div className="flow-wrap">
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
              <Background gap={20} size={1} variant={BackgroundVariant.Dots} />
            </ReactFlow>
          </div>
        </section>

        <aside className="panel inspector-panel">
          <div className="section-heading compact">
            <div>
              <div className="eyebrow">Inspector</div>
              <h3>{selectedWorkspaceNode?.label ?? "No node selected"}</h3>
            </div>
          </div>

          {selectedWorkspaceNode ? (
            <div className="stack">
              <div className="inspector-card">
                <div className="row-title">
                  <span className={`node-kind kind-${selectedWorkspaceNode.category}`}>{selectedWorkspaceNode.category}</span>
                  <span className="chip">{formatNodeType(selectedWorkspaceNode.type)}</span>
                </div>
                {typeof selectedWorkspaceNode.confidence === "number" && selectedWorkspaceNode.confidence > 0 ? (
                  <div className="muted">Confidence {Math.round(selectedWorkspaceNode.confidence * 100)}%</div>
                ) : null}
                {selectedWorkspaceNode.collapsed_count ? (
                  <div className="muted">{selectedWorkspaceNode.collapsed_count} additional items hidden behind this summary node.</div>
                ) : null}
                {selectedWorkspaceNode.profile_url ? (
                  <a className="inline-link" href={selectedWorkspaceNode.profile_url} rel="noreferrer" target="_blank">
                    Open profile
                  </a>
                ) : null}
              </div>

              <div className="inspector-card">
                <strong>Supporting evidence</strong>
                {selectedRawNodes.length === 0 ? (
                  <div className="empty-state">No raw nodes attached to this synthesized node.</div>
                ) : (
                  <div className="compact-list">
                    {selectedRawNodes.map((node) => (
                      <div className="list-item" key={node.id}>
                        <div className="row-title">
                          <strong>{node.label}</strong>
                          <span className="chip">{formatNodeType(node.type)}</span>
                        </div>
                        <div className="muted">{node.source_module} · {Math.round(node.confidence * 100)}%</div>
                        {renderPropertyPreview(node.properties)}
                        <details className="raw-details">
                          <summary>Raw payload</summary>
                          <pre>{JSON.stringify(node, null, 2)}</pre>
                        </details>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="inspector-card">
                <strong>Related raw edges</strong>
                {selectedRawEdges.length === 0 ? (
                  <div className="empty-state">No matching raw edges for this node.</div>
                ) : (
                  <div className="compact-list">
                    {selectedRawEdges.slice(0, 8).map((edge) => (
                      <div className="list-item" key={edge.id}>
                        <div className="row-title">
                          <strong>{edge.type}</strong>
                          <span className="chip">{edge.source_module}</span>
                        </div>
                        <div className="muted mono">{edge.source} → {edge.target}</div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="empty-state">Select a graph node to inspect evidence.</div>
          )}
        </aside>
      </div>

      <section className="panel">
        <div className="section-heading compact">
          <div>
            <div className="eyebrow">Secondary views</div>
            <h3>Lists and logs</h3>
          </div>
          <div className="tab-row">
            {(["findings", "nodes", "edges", "events"] as const).map((tab) => (
              <button
                className={activeTab === tab ? "tab active" : "tab"}
                key={tab}
                onClick={() => setActiveTab(tab)}
                type="button"
              >
                {tab}
              </button>
            ))}
          </div>
        </div>

        {activeTab === "findings" ? (
          <FindingList findings={topFindings} onSelect={jumpToFinding} />
        ) : null}
        {activeTab === "nodes" ? <RawNodesTable nodes={workspace.record.graph?.nodes ?? []} /> : null}
        {activeTab === "edges" ? <RawEdgesTable edges={workspace.record.graph?.edges ?? []} /> : null}
        {activeTab === "events" ? (
          <EventList events={events} />
        ) : null}
      </section>
    </section>
  );
}

function WorkspaceGraphNode({ data, selected }: NodeProps<Node<GraphNodeData>>) {
  return (
    <div className={selected ? `workspace-node is-selected ${data.category}` : `workspace-node ${data.category}`}>
      <Handle className="hidden-handle" position={Position.Left} type="target" />
      <div className="workspace-node-head">
        <span className="node-kind">{formatNodeCategory(data.category)}</span>
        <span className="node-type">{formatNodeType(data.type)}</span>
      </div>
      <strong>{data.label}</strong>
      <div className="workspace-node-meta">
        {typeof data.confidence === "number" && data.confidence > 0 ? (
          <span>{Math.round(data.confidence * 100)}%</span>
        ) : null}
        {data.collapsedCount ? <span>+{data.collapsedCount} hidden</span> : null}
      </div>
      <Handle className="hidden-handle" position={Position.Right} type="source" />
    </div>
  );
}

function useFlowGraph(graph: ScanWorkspace["graph"], selectedNode: string) {
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
        width: node.category === "root" ? 260 : node.category === "aliases" ? 210 : 230,
        height: node.category === "root" ? 106 : 96,
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
          "elk.layered.spacing.nodeNodeBetweenLayers": "150",
          "elk.spacing.nodeNode": "74",
          "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
        },
        children: baseNodes.map((node) => ({
          id: node.id,
          width: Number(node.width ?? 220),
          height: Number(node.height ?? 96),
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

function FindingList({
  findings,
  onSelect,
}: {
  findings: InsightFinding[];
  onSelect: (finding: InsightFinding) => void;
}) {
  if (findings.length === 0) {
    return <div className="empty-state">No top findings.</div>;
  }
  return (
    <div className="finding-list">
      {findings.map((finding, index) => (
        <button className="finding-row" key={`${finding.title}-${index}`} onClick={() => onSelect(finding)} type="button">
          <div className="row-title">
            <strong>{finding.title}</strong>
            <span className="chip">{finding.category ?? "finding"}</span>
          </div>
          <p>{finding.summary}</p>
        </button>
      ))}
    </div>
  );
}

function EventList({ events }: { events: ScanEvent[] }) {
  if (events.length === 0) {
    return <div className="empty-state">No events.</div>;
  }
  return (
    <div className="compact-list">
      {events.map((event) => (
        <div className="list-item" key={`${event.sequence}-${event.type}`}>
          <div className="row-title">
            <strong>{event.type}</strong>
            <span className="chip">{event.module || "scan"}</span>
          </div>
          <div className="muted">{event.message || "No message"}</div>
          <div className="mono">{formatDate(event.time)}</div>
        </div>
      ))}
    </div>
  );
}

function RawNodesTable({ nodes }: { nodes: RawNode[] }) {
  if (nodes.length === 0) {
    return <div className="empty-state">No raw nodes.</div>;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Label</th>
            <th>Type</th>
            <th>Module</th>
            <th>Confidence</th>
          </tr>
        </thead>
        <tbody>
          {nodes.map((node) => (
            <tr key={node.id}>
              <td>{node.label}</td>
              <td>{formatNodeType(node.type)}</td>
              <td>{node.source_module}</td>
              <td>{Math.round(node.confidence * 100)}%</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RawEdgesTable({ edges }: { edges: RawEdge[] }) {
  if (edges.length === 0) {
    return <div className="empty-state">No raw edges.</div>;
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Type</th>
            <th>Source</th>
            <th>Target</th>
            <th>Module</th>
          </tr>
        </thead>
        <tbody>
          {edges.map((edge) => (
            <tr key={edge.id}>
              <td>{edge.type}</td>
              <td className="mono">{edge.source}</td>
              <td className="mono">{edge.target}</td>
              <td>{edge.source_module}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function SummaryListItem({ label, items }: { label: string; items: string[] }) {
  return (
    <div className="list-item">
      <strong>{label}</strong>
      {items.length === 0 ? <div className="muted">None</div> : <div className="muted">{items.join(", ")}</div>}
    </div>
  );
}

function StatCard({ title, value, subtitle }: { title: string; value: ReactNode; subtitle: string }) {
  return (
    <article className="stat-card">
      <div className="eyebrow">{title}</div>
      <div className="stat-value">{value}</div>
      <div className="muted">{subtitle}</div>
    </article>
  );
}

function StatusPill({ status }: { status: string }) {
  return <span className={`status-pill status-${status}`}>{status}</span>;
}

function renderPropertyPreview(properties?: Record<string, unknown>) {
  if (!properties) {
    return null;
  }
  const entries = Object.entries(properties)
    .filter(([, value]) => value !== "" && value !== null && value !== undefined)
    .filter(([key]) => ["site_name", "profile_url", "full_name", "location", "website", "bio"].includes(key))
    .slice(0, 4);
  if (entries.length === 0) {
    return null;
  }
  return (
    <div className="meta-list compact">
      {entries.map(([key, value]) => (
        <div key={key}>
          <span>{formatNodeType(key)}</span>
          <strong>{String(value)}</strong>
        </div>
      ))}
    </div>
  );
}

function summarizeHealth(health: ModuleStatus[]) {
  return health.reduce(
    (summary, item) => {
      if (item.status === "healthy") {
        summary.healthy += 1;
      } else if (item.status === "degraded") {
        summary.degraded += 1;
      } else {
        summary.offline += 1;
      }
      return summary;
    },
    { healthy: 0, degraded: 0, offline: 0 },
  );
}

function pageTitle(pathname: string) {
  if (pathname.startsWith("/scans/")) {
    return {
      kicker: "Workspace",
      title: "Scan graph",
      summary: "Main graph view with evidence, findings, and supporting lists.",
    };
  }
  if (pathname === "/targets") {
    return {
      kicker: "Targets",
      title: "People and aliases",
      summary: "Curate multiple usernames, emails, and domains under one real target.",
    };
  }
  if (pathname === "/new") {
    return {
      kicker: "Launch",
      title: "Start a scan",
      summary: "Run a target-backed or ad hoc scan and open the graph workspace directly.",
    };
  }
  if (pathname === "/settings") {
    return {
      kicker: "Settings",
      title: "Local defaults",
      summary: "Saved settings are shared across CLI and web-triggered scans.",
    };
  }
  return {
    kicker: "Overview",
    title: "Basalt home",
    summary: "Recent scans, current module health, and quick access to saved targets.",
  };
}

async function api<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });
  const isJSON = (response.headers.get("content-type") ?? "").includes("application/json");
  const payload = isJSON ? (await response.json()) : await response.text();
  if (!response.ok) {
    if (isJSON && typeof payload === "object" && payload && "error" in payload) {
      throw new Error(String((payload as { error: string }).error));
    }
    throw new Error(typeof payload === "string" ? payload : response.statusText);
  }
  return payload as T;
}

function splitCommaList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function updateSeed(seeds: Seed[], setSeeds: (value: Seed[]) => void, index: number, next: Partial<Seed>) {
  setSeeds(
    seeds.map((seed, itemIndex) =>
      itemIndex === index
        ? { ...seed, ...next }
        : seed,
    ),
  );
}

function formatDate(value?: string) {
  if (!value) {
    return "n/a";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function asMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : String(reason);
}

function formatNodeType(value: string) {
  return value.replaceAll("_", " ");
}

function formatNodeCategory(value: string) {
  if (value === "web") {
    return "web";
  }
  return value.replaceAll("_", " ");
}
