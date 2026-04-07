import { Suspense, lazy, startTransition, useCallback, useEffect, useMemo, useState } from "react";
import { Route, Routes, useLocation } from "react-router-dom";
import { Menu, X } from "lucide-react";

import { Sidebar } from "./components/Sidebar";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { api } from "./lib/api";
import { asMessage } from "./lib/format";
import { ACTIVE_STATUSES, getRouteLabel, themeStorageKey } from "./lib/constants";
import { HomePage } from "./pages/HomePage";
import type { Bootstrap, ModuleStatus, ScanRecord, Settings, Target, ThemeMode } from "./types";

const NewScanPage = lazy(() =>
  import("./pages/NewScanPage").then((module) => ({ default: module.NewScanPage }))
);

const ScanWorkspacePage = lazy(() =>
  import("./pages/ScanWorkspacePage").then((module) => ({ default: module.ScanWorkspacePage }))
);

const SettingsPage = lazy(() =>
  import("./pages/SettingsPage").then((module) => ({ default: module.SettingsPage }))
);

const TargetsPage = lazy(() =>
  import("./pages/TargetsPage").then((module) => ({ default: module.TargetsPage }))
);

export function App() {
  const [bootstrap, setBootstrap] = useState<Bootstrap | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [targets, setTargets] = useState<Target[]>([]);
  const [scans, setScans] = useState<ScanRecord[]>([]);
  const [health, setHealth] = useState<ModuleStatus[]>([]);
  const [error, setError] = useState("");
  const [theme, setTheme] = useState<ThemeMode>(() => {
    if (typeof window === "undefined") return "dark";
    const stored = window.localStorage.getItem(themeStorageKey);
    return stored === "light" ? "light" : "dark";
  });

  const location = useLocation();
  const isMobileShell = useMediaQuery("(max-width: 1023px)");
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  const liveScanCount = useMemo(
    () => scans.filter((s) => (ACTIVE_STATUSES as readonly string[]).includes(s.status)).length,
    [scans]
  );

  const refreshHome = useCallback(async () => {
    try {
      const [boot, nextSettings, nextTargets, nextScans, nextHealth] = await Promise.all([
        api<Bootstrap>("/app/bootstrap"),
        api<Settings>("/api/settings"),
        api<{ targets: Target[] }>("/api/targets"),
        api<{ scans: ScanRecord[] }>("/api/scans"),
        api<{ modules: ModuleStatus[] }>("/api/modules/health"),
      ]);

      startTransition(() => {
        setBootstrap(boot);
        setSettings(nextSettings);
        setTargets(nextTargets.targets);
        setScans(nextScans.scans);
        setHealth(nextHealth.modules);
        setError("");
      });
    } catch (reason) {
      setError(asMessage(reason));
    }
  }, []);

  const refreshActivity = useCallback(async () => {
    try {
      const [boot, nextSettings, nextTargets, nextScans] = await Promise.all([
        api<Bootstrap>("/app/bootstrap"),
        api<Settings>("/api/settings"),
        api<{ targets: Target[] }>("/api/targets"),
        api<{ scans: ScanRecord[] }>("/api/scans"),
      ]);

      startTransition(() => {
        setBootstrap(boot);
        setSettings(nextSettings);
        setTargets(nextTargets.targets);
        setScans(nextScans.scans);
        setError("");
      });
    } catch (reason) {
      setError(asMessage(reason));
    }
  }, []);

  useEffect(() => {
    void refreshHome();

    const timer = setInterval(() => {
      void refreshActivity();
    }, 10000);

    return () => clearInterval(timer);
  }, [refreshActivity, refreshHome]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  useEffect(() => {
    if (isMobileShell) {
      setMobileNavOpen(false);
    }
  }, [isMobileShell, location.pathname]);

  // Workspace is full-bleed - no page padding
  const isWorkspace = location.pathname.startsWith("/scans/");
  const routeLabel = getRouteLabel(location.pathname);
  const routeFallback = (
    <div
      style={{
        minHeight: isWorkspace ? "calc(100vh - 56px)" : 240,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      <div
        className="card"
        style={{
          padding: "14px 18px",
          display: "inline-flex",
          alignItems: "center",
          gap: 12,
        }}
      >
        <div className="spinner" />
        <span className="mono" style={{ fontSize: 11, color: "var(--text-muted)" }}>
          Loading route…
        </span>
      </div>
    </div>
  );

  return (
    <div className="app-layout">
      <Sidebar
        liveScanCount={liveScanCount}
        bootstrap={bootstrap}
        mobileOpen={mobileNavOpen}
        isMobile={isMobileShell}
        onRequestClose={() => setMobileNavOpen(false)}
      />

      <div className="app-main">
        {isMobileShell && !isWorkspace && (
          <header className="mobile-topbar">
            <button
              className="mobile-topbar-btn"
              type="button"
              onClick={() => setMobileNavOpen((current) => !current)}
              aria-label={mobileNavOpen ? "Close navigation" : "Open navigation"}
            >
              {mobileNavOpen ? <X size={16} /> : <Menu size={16} />}
            </button>
            <div className="mobile-topbar-copy">
              <span className="mobile-topbar-brand">Basalt</span>
              <span className="mobile-topbar-route">{routeLabel}</span>
            </div>
          </header>
        )}

        {error && (
          <div style={{ padding: "8px 20px 0" }}>
            <div className="error-banner">{error}</div>
          </div>
        )}

        <div className={`page-content${isWorkspace ? " no-padding" : ""}`}>
          <Suspense fallback={routeFallback}>
            <Routes>
              <Route
                path="/"
                element={
                  <HomePage
                    scans={scans}
                    targets={targets}
                    health={health}
                  />
                }
              />
              <Route
                path="/targets"
                element={<TargetsPage targets={targets} onRefresh={refreshHome} />}
              />
              <Route
                path="/new"
                element={
                  <NewScanPage
                    targets={targets}
                    settings={settings}
                    health={health}
                    onCreated={refreshHome}
                  />
                }
              />
              <Route
                path="/settings"
                element={
                  <SettingsPage
                    settings={settings}
                    bootstrap={bootstrap}
                    health={health}
                    theme={theme}
                    onToggleTheme={() => setTheme((c) => (c === "dark" ? "light" : "dark"))}
                    onRefresh={refreshHome}
                  />
                }
              />
              <Route
                path="/scans/:scanID"
                element={<ScanWorkspacePage onRefreshHome={refreshHome} />}
              />
            </Routes>
          </Suspense>
        </div>
      </div>
    </div>
  );
}
