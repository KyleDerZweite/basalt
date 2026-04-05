import { useEffect, useState } from "react";
import { Check, Info } from "lucide-react";

import { PretextBlock } from "../components/PretextBlock";
import { api } from "../lib/api";
import { asMessage } from "../lib/format";
import { lineHeights, pretextFonts } from "../lib/typography";
import type { Bootstrap, ModuleStatus, Settings, ThemeMode } from "../types";

interface SettingsPageProps {
  settings: Settings | null;
  bootstrap: Bootstrap | null;
  health: ModuleStatus[];
  theme: ThemeMode;
  onToggleTheme: () => void;
  onRefresh: () => Promise<void>;
}

export function SettingsPage({
  settings,
  bootstrap,
  health,
  theme,
  onToggleTheme,
  onRefresh,
}: SettingsPageProps) {
  const [strictMode, setStrictMode] = useState(false);
  const [disabledModules, setDisabledModules] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  // Sync local state from settings prop
  useEffect(() => {
    if (settings) {
      setStrictMode(settings.strict_mode);
      setDisabledModules(settings.disabled_modules ?? []);
    }
  }, [settings]);

  const toggleModule = (name: string) => {
    setDisabledModules((prev) =>
      prev.includes(name) ? prev.filter((m) => m !== name) : [...prev, name]
    );
    setSaved(false);
  };

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError("");
    setSaved(false);
    try {
      await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify({
          strict_mode: strictMode,
          disabled_modules: disabledModules.length > 0 ? disabledModules : null,
          legal_accepted_at: settings?.legal_accepted_at,
        }),
      });
      await onRefresh();
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setSaving(false);
    }
  }

  async function handleAcceptLegal() {
    setSaving(true);
    setError("");
    try {
      await api("/api/settings", {
        method: "PUT",
        body: JSON.stringify({
          strict_mode: strictMode,
          disabled_modules: disabledModules.length > 0 ? disabledModules : null,
          legal_accepted_at: new Date().toISOString(),
        }),
      });
      await onRefresh();
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-header-kicker">Configuration</div>
        <PretextBlock
          as="h1"
          className="page-header-title"
          text="Settings"
          font={pretextFonts.pageTitle}
          lineHeight={lineHeights.title}
        />
        <PretextBlock
          as="p"
          className="page-header-desc"
          text="Appearance, module defaults, and platform configuration."
          font={pretextFonts.pageDescription}
          lineHeight={lineHeights.body}
        />
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="settings-layout">
        {/* Appearance */}
        <div className="settings-section">
          <div className="settings-section-title">Appearance</div>

          <div className="form-group">
            <label className="form-label">Theme</label>
            <div className="theme-swatches">
              <div
                className={`theme-swatch${theme === "dark" ? " selected" : ""}`}
                onClick={theme === "light" ? onToggleTheme : undefined}
              >
                <div className="swatch-preview dark-preview">
                  <div className="swatch-bar" />
                  <div className="swatch-bar" />
                </div>
                <span className="swatch-label">Dark</span>
              </div>
              <div
                className={`theme-swatch${theme === "light" ? " selected" : ""}`}
                onClick={theme === "dark" ? onToggleTheme : undefined}
              >
                <div className="swatch-preview light-preview">
                  <div className="swatch-bar" />
                  <div className="swatch-bar" />
                </div>
                <span className="swatch-label">Light</span>
              </div>
            </div>
          </div>
        </div>

        {/* Scan Defaults */}
        <div className="settings-section">
          <div className="settings-section-title">Scan Defaults</div>

          <form onSubmit={handleSave}>
            <div className="flex-col gap-4">
              <div className="toggle-row">
                <div className="toggle-info">
                  <div className="toggle-label">Strict Mode</div>
                  <div className="toggle-desc">
                    Only include high-confidence results in all scans by default.
                  </div>
                </div>
                <label className="toggle">
                  <input
                    type="checkbox"
                    checked={strictMode}
                    onChange={(e) => { setStrictMode(e.target.checked); setSaved(false); }}
                  />
                  <span className="toggle-slider" />
                </label>
              </div>

              {health.length > 0 && (
                <div className="form-group">
                  <label className="form-label">Disabled Modules (global default)</label>
                  <div className="modules-checkbox-grid">
                    {[...health]
                      .sort((a, b) => a.name.localeCompare(b.name))
                      .map((mod) => (
                        <label key={mod.name} className="module-checkbox-item">
                          <input
                            type="checkbox"
                            checked={disabledModules.includes(mod.name)}
                            onChange={() => toggleModule(mod.name)}
                          />
                          {mod.name}
                        </label>
                      ))}
                  </div>
                  <span className="form-hint" style={{ marginTop: 6 }}>
                    These can be overridden per-scan on the New Scan page.
                  </span>
                </div>
              )}

              <div className="flex items-center gap-3">
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={saving}
                >
                  {saving ? "Saving…" : "Save Defaults"}
                </button>
                {saved && (
                  <span style={{ fontSize: 13, color: "var(--success)", display: "inline-flex", alignItems: "center", gap: 4 }}><Check size={13} /> Saved</span>
                )}
              </div>
            </div>
          </form>
        </div>

        {/* Module Config / API Keys */}
        <div className="settings-section">
          <div className="settings-section-title">API Keys &amp; Module Config</div>
          <div className="info-banner">
            <Info size={13} style={{ flexShrink: 0 }} /> Module-specific API keys (e.g. Shodan, Hunter.io) are configured in the server config file.
          </div>
          {bootstrap && (
            <div className="flex-col gap-2" style={{ marginTop: 14 }}>
              <div className="form-label">Config File Path</div>
              <div className="code-path">{bootstrap.default_config_path}</div>
              <span className="form-hint">
                Edit this file to add API keys for individual modules.
              </span>
            </div>
          )}
        </div>

        {/* Legal */}
        <div className="settings-section">
          <div className="settings-section-title">Legal</div>

          {settings?.legal_accepted_at ? (
            <div className="flex-col gap-2">
              <div className="info-banner" style={{ background: "var(--success-soft)", borderColor: "rgba(34,197,94,0.2)", color: "var(--success)" }}>
                <Check size={13} /> Legal terms accepted on {new Date(settings.legal_accepted_at).toLocaleDateString()}
              </div>
            </div>
          ) : (
            <div className="flex-col gap-3">
              <PretextBlock
                className="legal-copy"
                text={
                  "Basalt is an OSINT tool intended for privacy self-research and legitimate security use. By accepting, you confirm that you will only use Basalt on subjects you have legal right to investigate, in compliance with applicable laws including GDPR, CCPA, and local regulations."
                }
                font={pretextFonts.pageDescription}
                lineHeight={lineHeights.body}
              />
              <button
                className="btn btn-primary"
                onClick={handleAcceptLegal}
                disabled={saving}
              >
                {saving ? "Saving…" : "I Accept — Use Responsibly"}
              </button>
            </div>
          )}
        </div>

        {/* About */}
        {bootstrap && (
          <div className="settings-section">
            <div className="settings-section-title">About</div>
            <div className="flex-col gap-3">
              <div className="flex items-center gap-3">
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: 4,
                    background: "var(--accent)",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontFamily: "var(--font-display)",
                    fontWeight: 800,
                    fontSize: 18,
                    color: "#000",
                  }}
                >
                  B
                </div>
                <div>
                  <div style={{ fontFamily: "var(--font-display)", fontWeight: 700, fontSize: 16 }}>Basalt</div>
                  <div style={{ fontFamily: "var(--font-mono)", fontSize: 12, color: "var(--text-muted)" }}>
                    v{bootstrap.version}
                  </div>
                </div>
              </div>

              <div className="flex-col gap-2">
                <div className="form-label">Data Directory</div>
                <div className="code-path">{bootstrap.data_dir}</div>
              </div>
              <div className="flex-col gap-2">
                <div className="form-label">API Base URL</div>
                <div className="code-path">{bootstrap.base_url}</div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
