import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ArrowRight, ChevronUp, ChevronDown } from "lucide-react";

import { PretextBlock } from "../components/PretextBlock";
import { SeedInputRow } from "../components/SeedInputRow";
import { api } from "../lib/api";
import { asMessage } from "../lib/format";
import { lineHeights, pretextFonts } from "../lib/typography";
import type { ModuleStatus, ScanRecord, Seed, Settings, Target } from "../types";

interface NewScanPageProps {
  targets: Target[];
  settings: Settings | null;
  health: ModuleStatus[];
  onCreated: () => Promise<void>;
}

const DEFAULT_SEEDS: Seed[] = [{ type: "username", value: "" }];

export function NewScanPage({ targets, settings, health, onCreated }: NewScanPageProps) {
  const navigate = useNavigate();
  const [params] = useSearchParams();

  // Form state
  const [targetRef, setTargetRef] = useState<string>(params.get("target") ?? "");
  const [seeds, setSeeds] = useState<Seed[]>(DEFAULT_SEEDS);
  const [depth, setDepth] = useState(2);
  const [concurrency, setConcurrency] = useState(5);
  const [timeout, setTimeout_] = useState(10);
  const [strictMode, setStrictMode] = useState(settings?.strict_mode ?? false);
  const [disabledModules, setDisabledModules] = useState<string[]>(settings?.disabled_modules ?? []);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Pre-fill seed from quick-launch
  useEffect(() => {
    const quickSeed = params.get("seed");
    if (quickSeed) {
      setSeeds([{ type: "username", value: quickSeed }]);
    }
  }, [params]);

  // Pre-fill seeds from selected target
  useEffect(() => {
    if (!targetRef) return;
    const target = targets.find((t) => t.slug === targetRef || t.id === targetRef);
    if (target?.aliases?.length) {
      setSeeds(target.aliases.map((a) => ({ type: a.seed_type, value: a.seed_value })));
    }
  }, [targetRef, targets]);

  const selectedTarget = useMemo(
    () => targets.find((t) => t.slug === targetRef || t.id === targetRef) ?? null,
    [targets, targetRef]
  );
  const selectedTargetAliasCount = selectedTarget?.aliases?.length ?? 0;

  const addSeed = useCallback(() => {
    setSeeds((prev) => [...prev, { type: "username", value: "" }]);
  }, []);

  const removeSeed = useCallback((index: number) => {
    setSeeds((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const toggleModule = (name: string) => {
    setDisabledModules((prev) =>
      prev.includes(name) ? prev.filter((m) => m !== name) : [...prev, name]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const validSeeds = seeds.filter((s) => s.value.trim() !== "");
    if (validSeeds.length === 0) {
      if (!selectedTarget) {
        setError("Add at least one seed value or select a target.");
        return;
      }
      if (selectedTargetAliasCount === 0) {
        setError("Selected target has no aliases. Add a seed or configure target aliases first.");
        return;
      }
    }

    setLoading(true);
    setError("");

    try {
      const scan = await api<ScanRecord>("/api/scans", {
        method: "POST",
        body: JSON.stringify({
          seeds: validSeeds.length > 0 ? validSeeds : undefined,
          depth,
          concurrency,
          timeout_seconds: timeout,
          strict_mode: strictMode,
          disabled_modules: disabledModules.length > 0 ? disabledModules : undefined,
          target_ref: targetRef || undefined,
        }),
      });
      await onCreated();
      navigate(`/scans/${scan.id}`);
    } catch (reason) {
      setError(asMessage(reason));
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="page-header">
        <div className="page-header-kicker">Investigation</div>
        <PretextBlock
          as="h1"
          className="page-header-title"
          text="New Scan"
          font={pretextFonts.pageTitle}
          lineHeight={lineHeights.title}
        />
        <PretextBlock
          as="p"
          className="page-header-desc"
          text="Configure seeds and settings, then launch an investigation."
          font={pretextFonts.pageDescription}
          lineHeight={lineHeights.body}
        />
      </div>

      {error && <div className="error-banner">{error}</div>}

      <form onSubmit={handleSubmit}>
        <div className="scan-form-layout">
          {/* Left: form */}
          <div className="scan-form">
            {/* Target picker */}
            <div className="form-group">
              <label className="form-label">Target (optional)</label>
              <select
                value={targetRef}
                onChange={(e) => setTargetRef(e.target.value)}
              >
                <option value="">No target - anonymous scan</option>
                {targets.map((t) => (
                  <option key={t.id} value={t.slug}>{t.display_name}</option>
                ))}
              </select>
              <span className="form-hint">
                Selecting a target links results and can launch a scan from its stored aliases even if manual seeds are empty.
              </span>
            </div>

            {/* Seeds */}
            <div>
              <div className="section-head">
                <span className="section-title">Seeds</span>
                <button
                  type="button"
                  className="btn btn-ghost btn-sm"
                  onClick={addSeed}
                >
                  + Add Seed
                </button>
              </div>
              <span className="form-hint">
                Manual seeds are optional when the selected target already has aliases configured.
              </span>
              <div className="seeds-list">
                {seeds.map((seed, i) => (
                  <SeedInputRow
                    key={i}
                    seed={seed}
                    index={i}
                    seeds={seeds}
                    onChange={setSeeds}
                    onRemove={() => removeSeed(i)}
                    canRemove={seeds.length > 1}
                  />
                ))}
              </div>
            </div>

            {/* Advanced settings */}
            <div className="scan-settings-section">
              <div
                className="scan-settings-toggle"
                onClick={() => setAdvancedOpen((o) => !o)}
              >
                <span className="scan-settings-toggle-title">Advanced Settings</span>
                <span style={{ color: "var(--text-muted)", fontSize: 13 }}>
                  {advancedOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
                </span>
              </div>

              {advancedOpen && (
                <div className="scan-settings-body">
                  {/* Depth */}
                  <div className="form-group">
                    <label className="form-label">Pivot Depth - {depth}</label>
                    <input
                      type="range"
                      min={1}
                      max={5}
                      value={depth}
                      onChange={(e) => setDepth(Number(e.target.value))}
                    />
                    <span className="form-hint">How many hops to follow from the initial seeds.</span>
                  </div>

                  {/* Concurrency */}
                  <div className="form-group">
                    <label className="form-label">Concurrency</label>
                    <input
                      type="number"
                      min={1}
                      max={20}
                      value={concurrency}
                      onChange={(e) => setConcurrency(Number(e.target.value))}
                    />
                    <span className="form-hint">Number of modules running in parallel.</span>
                  </div>

                  {/* Timeout */}
                  <div className="form-group">
                    <label className="form-label">Timeout (seconds)</label>
                    <input
                      type="number"
                      min={5}
                      max={120}
                      value={timeout}
                      onChange={(e) => setTimeout_(Number(e.target.value))}
                    />
                  </div>

                  {/* Strict mode */}
                  <div className="toggle-row">
                    <div className="toggle-info">
                      <div className="toggle-label">Strict Mode</div>
                      <div className="toggle-desc">Only include high-confidence results.</div>
                    </div>
                    <label className="toggle">
                      <input
                        type="checkbox"
                        checked={strictMode}
                        onChange={(e) => setStrictMode(e.target.checked)}
                      />
                      <span className="toggle-slider" />
                    </label>
                  </div>

                  {/* Disabled modules */}
                  {health.length > 0 && (
                    <div className="form-group">
                      <label className="form-label">Disable Modules</label>
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
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Submit */}
            <button
              type="submit"
              className="btn btn-primary btn-full btn-lg"
              disabled={loading}
            >
              {loading ? "Launching…" : <>Launch Scan <ArrowRight size={14} /></>}
            </button>
          </div>

          {/* Right: preview */}
          <div className="scan-preview-card">
            <div className="scan-preview-label">Scan Preview</div>
            <div className="scan-preview-body">
              <div className="scan-preview-row">
                <div className="scan-preview-key">Target</div>
                <div className="scan-preview-val">
                  {selectedTarget ? selectedTarget.display_name : "Anonymous"}
                </div>
              </div>

              <div className="scan-preview-row">
                <div className="scan-preview-key">Seeds</div>
                <PretextBlock
                  className="scan-preview-val"
                  text={
                    seeds.filter((s) => s.value.trim()).length > 0
                      ? seeds
                          .filter((s) => s.value.trim())
                          .map((s) => `${s.type}: ${s.value}`)
                          .join("\n")
                      : selectedTargetAliasCount > 0
                        ? `Using ${selectedTargetAliasCount} stored alias${selectedTargetAliasCount === 1 ? "" : "es"} from target`
                        : "None configured"
                  }
                  font={pretextFonts.previewValue}
                  lineHeight={lineHeights.body}
                  whiteSpace="pre-wrap"
                />
              </div>

              <div className="scan-preview-row">
                <div className="scan-preview-key">Depth</div>
                <div className="scan-preview-val">{depth}</div>
              </div>

              <div className="scan-preview-row">
                <div className="scan-preview-key">Concurrency</div>
                <div className="scan-preview-val">{concurrency}</div>
              </div>

              <div className="scan-preview-row">
                <div className="scan-preview-key">Timeout</div>
                <div className="scan-preview-val">{timeout}s</div>
              </div>

              <div className="scan-preview-row">
                <div className="scan-preview-key">Strict Mode</div>
                <div className="scan-preview-val">{strictMode ? "Yes" : "No"}</div>
              </div>

              {disabledModules.length > 0 && (
                <div className="scan-preview-row">
                  <div className="scan-preview-key">Disabled</div>
                  <PretextBlock
                    className="scan-preview-val"
                    text={disabledModules.join(", ")}
                    font={pretextFonts.previewValue}
                    lineHeight={lineHeights.body}
                  />
                </div>
              )}

              {/* Module health summary */}
              {health.length > 0 && (
                <div className="scan-preview-row">
                  <div className="scan-preview-key">Modules</div>
                  <div className="scan-preview-val" style={{ color: "var(--success)" }}>
                    {health.filter((m) => m.status === "healthy" && !disabledModules.includes(m.name)).length} ready
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </form>
    </div>
  );
}
