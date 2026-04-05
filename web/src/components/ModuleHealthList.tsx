import { useState } from "react";
import { ChevronUp, ChevronDown } from "lucide-react";
import type { ModuleStatus } from "../types";

interface ModuleHealthListProps {
  modules: ModuleStatus[];
  /** How many to show before collapsing. Defaults to 6. */
  limit?: number;
}

export function ModuleHealthList({ modules, limit = 6 }: ModuleHealthListProps) {
  const [expanded, setExpanded] = useState(false);

  if (modules.length === 0) {
    return <div className="health-list" style={{ color: "var(--text-muted)", fontSize: 12 }}>No module data</div>;
  }

  const visible = expanded ? modules : modules.slice(0, limit);
  const hasMore = modules.length > limit;

  return (
    <div className="health-list compact">
      {visible.map((mod) => (
        <div className="health-item" key={mod.name}>
          <span className={`health-dot ${mod.status}`} />
          <span className="health-name">{mod.name}</span>
        </div>
      ))}
      {hasMore && (
        <button
          className="health-expand-btn"
          onClick={() => setExpanded((e) => !e)}
        >
          {expanded ? <><ChevronUp size={12} /> Show less</> : <><ChevronDown size={12} /> +{modules.length - limit} more</>}
        </button>
      )}
    </div>
  );
}
