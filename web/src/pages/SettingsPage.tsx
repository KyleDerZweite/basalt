import { useEffect, useState } from "react";

import { SectionHeading } from "../components/SectionHeading";
import { api } from "../lib/api";
import { asMessage, splitCommaList } from "../lib/format";
import type { Bootstrap, Settings, ThemeMode } from "../types";

type SettingsPageProps = {
  settings: Settings | null;
  bootstrap: Bootstrap | null;
  onRefresh: () => Promise<void>;
  theme: ThemeMode;
};

export function SettingsPage({ settings, bootstrap, onRefresh, theme }: SettingsPageProps) {
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
    <section className="page stack-xl">
      {localError ? <div className="error-banner">{localError}</div> : null}

      <div className="section-grid section-grid-tight">
        <form className="surface stack" onSubmit={save}>
          <SectionHeading
            kicker="Defaults"
            title="Scan behavior"
            summary="These saved defaults are shared by the CLI and the browser workspace."
          />
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
          <button className="button" type="submit">
            Save settings
          </button>
        </form>

        <section className="surface">
          <SectionHeading
            kicker="Runtime"
            title="Local environment"
            summary="Everything here is local metadata from the running Basalt process."
          />
          <div className="detail-list">
            <div className="detail-row">
              <strong>Theme</strong>
              <span>{theme}</span>
            </div>
            <div className="detail-row">
              <strong>Version</strong>
              <span className="mono">{bootstrap?.version ?? "unknown"}</span>
            </div>
            <div className="detail-row">
              <strong>Base URL</strong>
              <span className="mono">{bootstrap?.base_url ?? "unknown"}</span>
            </div>
            <div className="detail-row">
              <strong>Data directory</strong>
              <span className="mono">{bootstrap?.data_dir ?? "unknown"}</span>
            </div>
            <div className="detail-row">
              <strong>Config path</strong>
              <span className="mono">{bootstrap?.default_config_path ?? "unknown"}</span>
            </div>
          </div>
        </section>
      </div>
    </section>
  );
}
