import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { EmptyState } from "../components/EmptyState";
import { SectionHeading } from "../components/SectionHeading";
import { api } from "../lib/api";
import { asMessage } from "../lib/format";
import type { Target } from "../types";

type TargetsPageProps = {
  targets: Target[];
  onRefresh: () => Promise<void>;
};

export function TargetsPage({ targets, onRefresh }: TargetsPageProps) {
  const [displayName, setDisplayName] = useState("");
  const [slug, setSlug] = useState("");
  const [notes, setNotes] = useState("");
  const [selectedTargetID, setSelectedTargetID] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [aliasType, setAliasType] = useState("username");
  const [aliasValue, setAliasValue] = useState("");
  const [aliasLabel, setAliasLabel] = useState("");
  const [aliasPrimary, setAliasPrimary] = useState(false);
  const [localError, setLocalError] = useState("");
  const deferredSearch = useDeferredValue(searchQuery);

  const filteredTargets = useMemo(() => {
    const query = deferredSearch.trim().toLowerCase();
    if (!query) {
      return targets;
    }

    return targets.filter((target) => {
      const haystack = [
        target.display_name,
        target.slug,
        target.notes,
        ...(target.aliases ?? []).flatMap((alias) => [alias.seed_type, alias.seed_value, alias.label]),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return haystack.includes(query);
    });
  }, [deferredSearch, targets]);

  useEffect(() => {
    if (filteredTargets.length === 0) {
      setSelectedTargetID("");
      return;
    }
    if (!filteredTargets.some((target) => target.id === selectedTargetID)) {
      setSelectedTargetID(filteredTargets[0].id);
    }
  }, [filteredTargets, selectedTargetID]);

  const selectedTarget = targets.find((target) => target.id === selectedTargetID) ?? null;

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
    <section className="page stack-xl">
      {localError ? <div className="error-banner">{localError}</div> : null}

      <div className="section-grid section-grid-tight">
        <section className="surface">
          <SectionHeading
            kicker="Directory"
            title="Target dossiers"
            summary="Filter saved people and aliases before you open the deeper editor."
          />
          <label className="field">
            <span>Search dossiers</span>
            <input
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Name, slug, alias, or note"
            />
          </label>

          {filteredTargets.length === 0 ? (
            <EmptyState title="No matching dossiers" detail="Try a broader search term or create a fresh target below." />
          ) : (
            <div className="directory-list selectable">
              {filteredTargets.map((target) => (
                <button
                  className={selectedTarget?.id === target.id ? "directory-row is-selected" : "directory-row"}
                  key={target.id}
                  type="button"
                  onClick={() => setSelectedTargetID(target.id)}
                >
                  <div>
                    <strong>{target.display_name}</strong>
                    <p>{target.notes || "No notes saved for this dossier."}</p>
                  </div>
                  <div className="directory-meta">
                    <span className="chip">{target.slug}</span>
                    <span>{(target.aliases ?? []).length} aliases</span>
                  </div>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="surface">
          <SectionHeading
            kicker="Create"
            title="Add a new dossier"
            summary="Use saved targets when you want repeatable scans and a stable alias ledger."
          />
          <form className="stack" onSubmit={createTarget}>
            <label className="field">
              <span>Display name</span>
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
            <button className="button" type="submit">
              Create target
            </button>
          </form>
        </section>
      </div>

      <section className="surface">
        <SectionHeading
          kicker="Alias ledger"
          title={selectedTarget ? selectedTarget.display_name : "Select a dossier"}
          summary={
            selectedTarget
              ? `Slug ${selectedTarget.slug} · ${(selectedTarget.aliases ?? []).length} saved aliases`
              : "Pick a dossier from the directory to manage usernames, emails, and domains."
          }
          action={
            selectedTarget ? (
              <Link className="button secondary" to={`/new?target=${encodeURIComponent(selectedTarget.slug)}`}>
                Scan this target
              </Link>
            ) : undefined
          }
        />

        {selectedTarget ? (
          <div className="alias-layout">
            <div className="dossier-note">
              <div className="mini-label">Dossier note</div>
              <p>{selectedTarget.notes || "No note stored for this target yet."}</p>
            </div>

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
                <input value={aliasLabel} onChange={(event) => setAliasLabel(event.target.value)} placeholder="main handle" />
              </label>
              <label className="checkbox">
                <input checked={aliasPrimary} onChange={(event) => setAliasPrimary(event.target.checked)} type="checkbox" />
                <span>Primary alias</span>
              </label>
              <button className="button" type="submit">
                Add alias
              </button>
            </form>

            {(selectedTarget.aliases ?? []).length === 0 ? (
              <EmptyState title="No aliases saved" detail="Add at least one handle, email, or domain so this dossier can launch richer scans." />
            ) : (
              <div className="ledger-list">
                {(selectedTarget.aliases ?? []).map((alias) => (
                  <div className="ledger-row static" key={alias.id}>
                    <div className="ledger-copy">
                      <strong>{alias.label || alias.seed_value}</strong>
                      <p>
                        {alias.seed_type}:{alias.seed_value}
                      </p>
                    </div>
                    <div className="ledger-meta">
                      {alias.is_primary ? <span className="chip accent">Primary</span> : null}
                      <button className="button secondary subtle" onClick={() => void removeAlias(alias.id)} type="button">
                        Remove
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <EmptyState title="No dossier selected" detail="Select a saved target to edit aliases and launch a scan from that dossier." />
        )}
      </section>
    </section>
  );
}
