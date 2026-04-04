import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import { SectionHeading } from "../components/SectionHeading";
import { api } from "../lib/api";
import { asMessage, splitCommaList } from "../lib/format";
import { updateSeed } from "../lib/seeds";
import type { ScanRecord, Seed, Settings, Target } from "../types";

type NewScanPageProps = {
  targets: Target[];
  settings: Settings | null;
  onCreated: () => Promise<void>;
};

export function NewScanPage({ targets, settings, onCreated }: NewScanPageProps) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const requestedTarget = searchParams.get("target") ?? "";
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

  useEffect(() => {
    if (!requestedTarget) {
      return;
    }
    if (targets.some((target) => target.slug === requestedTarget)) {
      setTargetRef((current) => current || requestedTarget);
    }
  }, [requestedTarget, targets]);

  const selectedTarget = useMemo(
    () => targets.find((target) => target.slug === targetRef) ?? null,
    [targetRef, targets],
  );
  const explicitSeeds = seeds.filter((seed) => seed.value.trim() !== "");
  const disabledCount = splitCommaList(disabledModules).length;

  async function createScan(event: React.FormEvent) {
    event.preventDefault();
    try {
      const payload = {
        target_ref: targetRef || undefined,
        seeds: explicitSeeds,
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
    <section className="page stack-xl">
      {localError ? <div className="error-banner">{localError}</div> : null}

      <form className="stack-xl" onSubmit={createScan}>
        <section className="surface">
          <div className="launch-layout">
            <div className="stack">
              <SectionHeading
                kicker="Scope"
                title="Mission profile"
                summary="Combine a saved dossier with extra seeds only when the case actually needs more explicit pivots."
              />

              <label className="field">
                <span>Target dossier</span>
                <select value={targetRef} onChange={(event) => setTargetRef(event.target.value)}>
                  <option value="">No saved target</option>
                  {targets.map((target) => (
                    <option key={target.id} value={target.slug}>
                      {target.display_name}
                    </option>
                  ))}
                </select>
              </label>

              <div className="target-preview">
                <strong>{selectedTarget ? selectedTarget.display_name : "Ad hoc scan"}</strong>
                <p>
                  {selectedTarget
                    ? (selectedTarget.aliases ?? []).map((alias) => `${alias.seed_type}:${alias.seed_value}`).join(" · ") || "No aliases stored on this dossier."
                    : "This scan will rely only on the explicit seeds you add below."}
                </p>
              </div>

              <div className="stack">
                <div className="section-heading">
                  <div>
                    <div className="section-kicker">Extra seeds</div>
                    <h3>Explicit pivots</h3>
                  </div>
                  <button
                    className="button secondary subtle"
                    type="button"
                    onClick={() => setSeeds((current) => [...current, { type: "username", value: "" }])}
                  >
                    Add seed
                  </button>
                </div>

                <div className="seed-list">
                  {seeds.map((seed, index) => (
                    <div className="seed-row" key={`${seed.type}-${index}`}>
                      <select value={seed.type} onChange={(event) => updateSeed(seeds, setSeeds, index, { type: event.target.value })}>
                        <option value="username">username</option>
                        <option value="email">email</option>
                        <option value="domain">domain</option>
                      </select>
                      <input
                        value={seed.value}
                        onChange={(event) => updateSeed(seeds, setSeeds, index, { value: event.target.value })}
                        placeholder="identifier"
                      />
                      <button
                        className="button secondary subtle"
                        type="button"
                        onClick={() => setSeeds((current) => current.filter((_, itemIndex) => itemIndex !== index))}
                      >
                        Remove
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <aside className="launch-brief">
              <div className="mini-label">Launch brief</div>
              <strong>{selectedTarget?.display_name ?? (explicitSeeds[0] ? explicitSeeds[0].value : "Awaiting inputs")}</strong>
              <p>
                {selectedTarget
                  ? "Saved aliases will stay attached to this scan and make later reruns cleaner."
                  : "Ad hoc scans are useful when you need to move quickly without creating a persistent dossier first."}
              </p>
              <dl className="brief-list">
                <div>
                  <dt>Saved aliases</dt>
                  <dd>{selectedTarget?.aliases?.length ?? 0}</dd>
                </div>
                <div>
                  <dt>Explicit seeds</dt>
                  <dd>{explicitSeeds.length}</dd>
                </div>
                <div>
                  <dt>Mode</dt>
                  <dd>{strictMode ? "strict" : "standard"}</dd>
                </div>
                <div>
                  <dt>Disabled modules</dt>
                  <dd>{disabledCount}</dd>
                </div>
              </dl>
            </aside>
          </div>
        </section>

        <section className="surface">
          <SectionHeading
            kicker="Execution"
            title="Runtime controls"
            summary="Tune the breadth and speed of collection before you open the workspace."
          />

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
              <span>Timeout seconds</span>
              <input
                type="number"
                min={1}
                value={timeoutSeconds}
                onChange={(event) => setTimeoutSeconds(Number(event.target.value) || 1)}
              />
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

          <div className="action-row">
            <button className="button" type="submit">
              Start scan
            </button>
            <Link className="button secondary" to="/targets">
              Manage targets
            </Link>
          </div>
        </section>
      </form>
    </section>
  );
}
