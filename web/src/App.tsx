import { startTransition, useCallback, useEffect, useMemo, useState } from "react";
import { Route, Routes, useLocation } from "react-router-dom";

import { AppShell } from "./components/AppShell";
import { PageLead } from "./components/PageLead";
import { api } from "./lib/api";
import { asMessage, pageTitle } from "./lib/format";
import { themeStorageKey } from "./lib/constants";
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
    if (typeof window === "undefined") {
      return "light";
    }
    const stored = window.localStorage.getItem(themeStorageKey);
    return stored === "dark" ? "dark" : "light";
  });
  const location = useLocation();
  const pageMeta = useMemo(() => pageTitle(location.pathname), [location.pathname]);
  const liveScanCount = scans.filter((scan) => ["queued", "verifying", "running"].includes(scan.status)).length;

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
  }, [refreshHome]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  return (
    <AppShell
      liveScanCount={liveScanCount}
      targetCount={targets.length}
      theme={theme}
      onToggleTheme={() => setTheme((current) => (current === "dark" ? "light" : "dark"))}
    >
      {location.pathname !== "/" && !location.pathname.startsWith("/scans/") ? (
        <PageLead
          kicker={pageMeta.kicker}
          title={pageMeta.title}
          summary={pageMeta.summary}
          detail={bootstrap ? `${bootstrap.base_url} · v${bootstrap.version}` : undefined}
        />
      ) : null}

      {error ? <div className="error-banner">{error}</div> : null}

      <Routes>
        <Route path="/" element={<HomePage scans={scans} targets={targets} health={health} settings={settings} />} />
        <Route path="/targets" element={<TargetsPage targets={targets} onRefresh={refreshHome} />} />
        <Route path="/new" element={<NewScanPage targets={targets} settings={settings} onCreated={refreshHome} />} />
        <Route path="/settings" element={<SettingsPage settings={settings} bootstrap={bootstrap} onRefresh={refreshHome} theme={theme} />} />
        <Route path="/scans/:scanID" element={<ScanWorkspacePage onRefreshHome={refreshHome} />} />
      </Routes>
    </AppShell>
  );
}
