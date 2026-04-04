import type { ReactNode } from "react";
import { Link, NavLink } from "react-router-dom";

import { navItems } from "../lib/constants";
import type { ThemeMode } from "../types";

type AppShellProps = {
  children: ReactNode;
  liveScanCount: number;
  targetCount: number;
  theme: ThemeMode;
  onToggleTheme: () => void;
};

export function AppShell({ children, liveScanCount, targetCount, theme, onToggleTheme }: AppShellProps) {
  return (
    <div className="app-shell">
      <div className="app-atmosphere" aria-hidden="true">
        <div className="atmosphere-orb atmosphere-orb-a" />
        <div className="atmosphere-orb atmosphere-orb-b" />
        <div className="atmosphere-grid" />
      </div>

      <header className="site-header">
        <Link className="brand-lockup" to="/">
          <span className="brand-kicker">Graph-first OSINT</span>
          <span className="brand-wordmark">BASALT</span>
          <span className="brand-subtitle">Local investigation workspace</span>
        </Link>

        <nav className="site-nav" aria-label="Primary">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              className={({ isActive }) => (isActive ? "nav-link is-active" : "nav-link")}
              to={item.to}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="site-tools">
          <div className="tool-note">
            <span>{liveScanCount} live scans</span>
            <span>{targetCount} saved targets</span>
          </div>
          <button className="theme-toggle" type="button" onClick={onToggleTheme}>
            {theme === "dark" ? "Day mode" : "Night mode"}
          </button>
        </div>
      </header>

      <main className="page-shell">{children}</main>
    </div>
  );
}
