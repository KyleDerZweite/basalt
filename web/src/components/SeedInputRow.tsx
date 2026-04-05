import { Trash2 } from "lucide-react";
import type { Seed } from "../types";
import { seedTypes } from "../lib/constants";
import { updateSeed } from "../lib/seeds";

interface SeedInputRowProps {
  seed: Seed;
  index: number;
  seeds: Seed[];
  onChange: (seeds: Seed[]) => void;
  onRemove: () => void;
  canRemove: boolean;
}

export function SeedInputRow({ seed, index, seeds, onChange, onRemove, canRemove }: SeedInputRowProps) {
  return (
    <div className="seed-row">
      <select
        className="seed-type-select"
        value={seed.type}
        onChange={(e) => updateSeed(seeds, onChange, index, { type: e.target.value })}
      >
        {seedTypes.map((t) => (
          <option key={t.value} value={t.value}>{t.label}</option>
        ))}
      </select>
      <input
        type="text"
        className="seed-value-input"
        placeholder={`Enter ${seed.type}…`}
        value={seed.value}
        onChange={(e) => updateSeed(seeds, onChange, index, { value: e.target.value })}
      />
      {canRemove && (
        <button className="seed-remove-btn" onClick={onRemove} type="button" title="Remove">
          <Trash2 size={14} />
        </button>
      )}
    </div>
  );
}
