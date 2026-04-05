import { useState } from "react";
import { NavLink } from "react-router-dom";
import {
  Home,
  ScanSearch,
  Crosshair,
  SlidersHorizontal,
  X,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";
import { navItems } from "../lib/constants";
import type { Bootstrap } from "../types";

const navIconMap: Record<string, React.ReactNode> = {
  home: <Home size={16} />,
  "scan-search": <ScanSearch size={16} />,
  crosshair: <Crosshair size={16} />,
  "sliders-horizontal": <SlidersHorizontal size={16} />,
};

interface SidebarProps {
  liveScanCount: number;
  bootstrap: Bootstrap | null;
  mobileOpen: boolean;
  isMobile: boolean;
  onRequestClose: () => void;
}

export function Sidebar({
  liveScanCount,
  bootstrap,
  mobileOpen,
  isMobile,
  onRequestClose,
}: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);
  const isCollapsed = !isMobile && collapsed;

  return (
    <>
      {isMobile && mobileOpen && (
        <button
          type="button"
          className="sidebar-backdrop"
          aria-label="Close navigation"
          onClick={onRequestClose}
        />
      )}

      <aside
        className={`sidebar${isCollapsed ? " collapsed" : ""}${isMobile ? " mobile" : ""}${mobileOpen ? " open" : ""}`}
      >
        <div className="sidebar-logo">
          <div className="sidebar-logo-mark">B</div>
          <span className="sidebar-logo-name">Basalt</span>
          {isMobile && (
            <button
              className="sidebar-mobile-close"
              type="button"
              onClick={onRequestClose}
              aria-label="Close navigation"
            >
              <X size={16} />
            </button>
          )}
        </div>

        <nav className="sidebar-nav">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === "/"}
              className={({ isActive }) => `sidebar-item${isActive ? " active" : ""}`}
              onClick={isMobile ? onRequestClose : undefined}
            >
              <span className="sidebar-item-icon">{navIconMap[item.icon]}</span>
              <span className="sidebar-item-label">{item.label}</span>
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-bottom">
          {liveScanCount > 0 && (
            <div className="sidebar-live-badge">
              <span className="live-dot" />
              <span className="sidebar-badge-label">
                {liveScanCount} scan{liveScanCount !== 1 ? "s" : ""} running
              </span>
            </div>
          )}
          {bootstrap && (
            <div className="sidebar-version mono" title={`Basalt version ${bootstrap.version}`}>
              {bootstrap.version}
            </div>
          )}
          {!isMobile && (
            <button
              className="sidebar-collapse-btn"
              onClick={() => setCollapsed((current) => !current)}
              title={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
            >
              {isCollapsed ? <ChevronsRight size={14} /> : <ChevronsLeft size={14} />}
            </button>
          )}
        </div>
      </aside>
    </>
  );
}
