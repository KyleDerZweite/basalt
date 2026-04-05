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
  { to: "/",        label: "Home",     icon: "home" },
  { to: "/new",     label: "New Scan", icon: "scan-search" },
  { to: "/targets", label: "Targets",  icon: "crosshair" },
  { to: "/settings",label: "Settings", icon: "sliders-horizontal" },
] as const;

export function getRouteLabel(pathname: string): string {
  if (pathname.startsWith("/scans/")) {
    return "Workspace";
  }

  const match = navItems.find((item) => item.to === "/" ? pathname === "/" : pathname.startsWith(item.to));
  return match?.label ?? "Basalt";
}

export const seedTypes = [
  { value: "username",  label: "Username"  },
  { value: "email",     label: "Email"     },
  { value: "domain",    label: "Domain"    },
  { value: "phone",     label: "Phone"     },
  { value: "full_name", label: "Full Name" },
] as const;

export const ACTIVE_STATUSES = ["queued", "verifying", "running"] as const;
