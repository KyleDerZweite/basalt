import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
import { Route, Routes, useLocation } from "react-router-dom";
import { Menu, X } from "lucide-react";

import { Sidebar } from "./components/Sidebar";
import { useMediaQuery } from "./hooks/useMediaQuery";
import { api } from "./lib/api";
import { asMessage } from "./lib/format";
import { ACTIVE_STATUSES, getRouteLabel, themeStorageKey } from "./lib/constants";
import { HomePage } from "./pages/HomePage";
import { NewScanPage } from "./pages/NewScanPage";
import { ScanWorkspacePage } from "./pages/ScanWorkspacePage";
import { SettingsPage } from "./pages/SettingsPage";
import { TargetsPage } from "./pages/TargetsPage";
import type { Bootstrap, ModuleStatus, ScanRecord, Settings, Target, ThemeMode } from "./types";

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

  useEffect(() => {
    void refreshHome();
    
    // Global poll for status updates every 10 seconds
    const timer = setInterval(() => {
      void refreshHome();
    }, 10000);
    
    return () => clearInterval(timer);
  }, [refreshHome]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  useEffect(() => {
    if (isMobileShell) {
      setMobileNavOpen(false);
    }
  }, [isMobileShell, location.pathname]);

  // Workspace is full-bleed — no page padding
  const isWorkspace = location.pathname.startsWith("/scans/");
  const routeLabel = getRouteLabel(location.pathname);

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
        </div>
      </div>
    </div>
  );
}
