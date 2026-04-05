import type { ReactNode } from "react";

interface TopBarProps {
  crumb?: string;
  title: string;
  meta?: ReactNode;
  actions?: ReactNode;
}

export function TopBar({ crumb, title, meta, actions }: TopBarProps) {
  return (
    <div className="topbar">
      <div className="topbar-breadcrumb">
        {crumb && (
          <>
            <span className="topbar-crumb">{crumb}</span>
            <span className="topbar-crumb-sep">›</span>
          </>
        )}
        <span className="topbar-title">{title}</span>
      </div>
      {meta && <div className="topbar-meta">{meta}</div>}
      {actions && <div className="topbar-actions">{actions}</div>}
    </div>
  );
}
