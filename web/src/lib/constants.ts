export const themeStorageKey = "basalt-theme";

export const scanEventTypes = [
  "scan_queued",
  "scan_status",
  "module_verified",
  "verify_complete",
  "module_started",
  "module_finished",
  "node_discovered",
  "edge_discovered",
  "module_error",
  "scan_finished",
  "scan_failed",
] as const;

export const navItems = [
  { to: "/", label: "Home" },
  { to: "/targets", label: "Targets" },
  { to: "/new", label: "New Scan" },
  { to: "/settings", label: "Settings" },
] as const;
