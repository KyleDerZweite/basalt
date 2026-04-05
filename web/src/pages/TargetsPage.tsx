import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight, Crosshair, X } from "lucide-react";

import { EmptyState } from "../components/EmptyState";
import { PretextBlock } from "../components/PretextBlock";
import { TargetCard } from "../components/TargetCard";
import { api } from "../lib/api";
import { asMessage } from "../lib/format";
import { seedTypes } from "../lib/constants";
import { lineHeights, pretextFonts } from "../lib/typography";
import type { Target } from "../types";

interface TargetsPageProps {
  targets: Target[];
  onRefresh: () => Promise<void>;
}

type PanelMode = "view" | "edit" | "create";

export function TargetsPage({ targets, onRefresh }: TargetsPageProps) {
  const navigate = useNavigate();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [panelMode, setPanelMode] = useState<PanelMode | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Form state
  const [displayName, setDisplayName] = useState("");
  const [notes, setNotes] = useState("");
  const [aliasType, setAliasType] = useState("username");
  const [aliasValue, setAliasValue] = useState("");

  const selectedTarget = targets.find((t) => t.id === selectedId) ?? null;

  function openCreate() {
    setSelectedId(null);
    setDisplayName("");
    setNotes("");
    setPanelMode("create");
    setError("");
  }

  function openView(target: Target) {
    setSelectedId(target.id);
    setPanelMode("view");
    setError("");
  }

  function openEdit(target: Target) {
    setSelectedId(target.id);
    setDisplayName(target.display_name);
    setNotes(target.notes ?? "");
    setPanelMode("edit");
    setError("");
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!displayName.trim()) return;
    setLoading(true);
    setError("");
    try {
      await api("/api/targets", {
        method: "POST",
        body: JSON.stringify({ display_name: displayName.trim(), notes: notes.trim() || undefined }),
      });
      await onRefresh();
      setDisplayName("");
      setNotes("");
      setPanelMode(null);
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setLoading(false);
    }
  }

  async function handleUpdate(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedId || !displayName.trim()) return;
    setLoading(true);
    setError("");
    try {
      await api(`/api/targets/${selectedId}`, {
        method: "PATCH",
        body: JSON.stringify({ display_name: displayName.trim(), notes: notes.trim() || undefined }),
      });
      await onRefresh();
      setPanelMode("view");
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setLoading(false);
    }
  }

  async function handleDelete(targetId: string) {
    if (!confirm("Delete this target? This cannot be undone.")) return;
    setLoading(true);
    try {
      await api(`/api/targets/${targetId}`, { method: "DELETE" });
      await onRefresh();
      if (selectedId === targetId) { setSelectedId(null); setPanelMode(null); }
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setLoading(false);
    }
  }

  async function handleAddAlias(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedId || !aliasValue.trim()) return;
    setLoading(true);
    setError("");
    try {
      await api(`/api/targets/${selectedId}/aliases`, {
        method: "POST",
        body: JSON.stringify({ seed_type: aliasType, seed_value: aliasValue.trim() }),
      });
      await onRefresh();
      setAliasValue("");
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setLoading(false);
    }
  }

  async function handleDeleteAlias(aliasId: string) {
    if (!selectedId) return;
    setLoading(true);
    try {
      await api(`/api/targets/${selectedId}/aliases/${aliasId}`, { method: "DELETE" });
      await onRefresh();
    } catch (reason) {
      setError(asMessage(reason));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <div className="page-header">
        <div className="page-header-kicker">Subject Management</div>
        <PretextBlock
          as="h1"
          className="page-header-title"
          text="Targets"
          font={pretextFonts.pageTitle}
          lineHeight={lineHeights.title}
        />
        <PretextBlock
          as="p"
          className="page-header-desc"
          text="Map identities across multiple seeds. Targets link scans to a named subject."
          font={pretextFonts.pageDescription}
          lineHeight={lineHeights.body}
        />
      </div>

      {error && <div className="error-banner">{error}</div>}

      <div className="targets-layout">
        {/* Left: target list */}
        <div>
          <div className="section-head">
            <span className="section-title">{targets.length} Target{targets.length !== 1 ? "s" : ""}</span>
            <button className="btn btn-primary btn-sm" onClick={openCreate}>
              + New Target
            </button>
          </div>

          {targets.length === 0 ? (
            <div className="card">
              <EmptyState
                icon={<Crosshair size={24} />}
                title="No targets yet"
                desc="Create a target to group multiple identities for a single person or entity."
                action={
                  <button className="btn btn-primary btn-sm" onClick={openCreate}>
                    Create Target
                  </button>
                }
              />
            </div>
          ) : (
            <div className="targets-list">
              {targets.map((target) => (
                <TargetCard
                  key={target.id}
                  target={target}
                  selected={selectedId === target.id}
                  onClick={() => openView(target)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Right: detail / create panel */}
        {panelMode && (
          <div className="target-detail-panel">
            <div className="detail-panel-head">
              <PretextBlock
                className="detail-panel-title"
                text={
                  panelMode === "create" ? "New Target" :
                  panelMode === "edit" ? "Edit Target" :
                  selectedTarget?.display_name ?? "Target"
                }
                font={pretextFonts.detailTitle}
                lineHeight={lineHeights.ui}
              />
              <div className="flex gap-2">
                {panelMode === "view" && selectedTarget && (
                  <>
                    <button className="btn btn-ghost btn-sm" onClick={() => openEdit(selectedTarget)}>
                      Edit
                    </button>
                    <button
                      className="btn btn-danger btn-sm"
                      onClick={() => handleDelete(selectedTarget.id)}
                      disabled={loading}
                    >
                      Delete
                    </button>
                  </>
                )}
                <button
                  className="btn btn-ghost btn-sm btn-icon"
                  onClick={() => setPanelMode(null)}
                  title="Close"
                >
                  <X size={14} />
                </button>
              </div>
            </div>

            <div className="detail-panel-body">
              {/* Create / Edit form */}
              {(panelMode === "create" || panelMode === "edit") && (
                <form onSubmit={panelMode === "create" ? handleCreate : handleUpdate}>
                  <div className="flex-col gap-4">
                    <div className="form-group">
                      <label className="form-label">Display Name</label>
                      <input
                        type="text"
                        placeholder="John Doe"
                        value={displayName}
                        onChange={(e) => setDisplayName(e.target.value)}
                        required
                      />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Notes (optional)</label>
                      <textarea
                        placeholder="Context, notes, or links…"
                        value={notes}
                        onChange={(e) => setNotes(e.target.value)}
                      />
                    </div>
                    <div className="flex gap-2">
                      <button type="submit" className="btn btn-primary flex-1" disabled={loading}>
                        {loading ? "Saving…" : panelMode === "create" ? "Create Target" : "Save Changes"}
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={() => setPanelMode(panelMode === "edit" ? "view" : null)}
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                </form>
              )}

              {/* View mode */}
              {panelMode === "view" && selectedTarget && (
                <>
                  {selectedTarget.notes && (
                    <div>
                      <div className="section-title" style={{ marginBottom: 8 }}>Notes</div>
                      <PretextBlock
                        className="target-detail-notes"
                        text={selectedTarget.notes}
                        font={pretextFonts.targetNotes}
                        lineHeight={lineHeights.body}
                      />
                    </div>
                  )}

                  {/* Aliases */}
                  <div>
                    <div className="section-head" style={{ marginBottom: 10 }}>
                      <span className="section-title">Aliases</span>
                    </div>

                    {(selectedTarget.aliases ?? []).length === 0 ? (
                      <PretextBlock
                        className="target-detail-hint"
                        text="No aliases yet. Add one below."
                        font={pretextFonts.findingSummary}
                        lineHeight={lineHeights.ui}
                      />
                    ) : (
                      <div className="alias-chips" style={{ marginBottom: 14 }}>
                        {selectedTarget.aliases!.map((alias) => (
                          <span className="alias-chip" key={alias.id}>
                            <span className="alias-chip-type">{alias.seed_type}</span>
                            {alias.seed_value}
                            <button
                              className="alias-chip-remove"
                              onClick={() => handleDeleteAlias(alias.id)}
                              disabled={loading}
                              title="Remove alias"
                            >
                              <X size={12} />
                            </button>
                          </span>
                        ))}
                      </div>
                    )}

                    {/* Add alias form */}
                    <form onSubmit={handleAddAlias}>
                      <div className="flex-col gap-2">
                        <div className="seed-row">
                          <select
                            className="seed-type-select"
                            value={aliasType}
                            onChange={(e) => setAliasType(e.target.value)}
                          >
                            {seedTypes.map((t) => (
                              <option key={t.value} value={t.value}>{t.label}</option>
                            ))}
                          </select>
                          <input
                            type="text"
                            className="seed-value-input"
                            placeholder="Value…"
                            value={aliasValue}
                            onChange={(e) => setAliasValue(e.target.value)}
                          />
                        </div>
                        <button type="submit" className="btn btn-ghost btn-full" disabled={loading || !aliasValue.trim()}>
                          {loading ? "Adding…" : "+ Add Alias"}
                        </button>
                      </div>
                    </form>
                  </div>

                  {/* Scan this target */}
                  <div className="divider" />
                  <button
                    className="btn btn-primary btn-full"
                    onClick={() => navigate(`/new?target=${selectedTarget.slug}`)}
                  >
                    Scan This Target <ArrowRight size={14} />
                  </button>
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
