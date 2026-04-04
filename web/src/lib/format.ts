import type { ModuleStatus, Seed } from "../types";

export function splitCommaList(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function formatDate(value?: string) {
  if (!value) {
    return "n/a";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

export function formatSeed(seed: Seed) {
  return `${seed.type}:${seed.value}`;
}

export function asMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : String(reason);
}

export function formatNodeType(value: string) {
  return value.replaceAll("_", " ");
}

export function formatNodeCategory(value: string) {
  if (value === "web") {
    return "web";
  }
  return value.replaceAll("_", " ");
}

export function summarizeHealth(health: ModuleStatus[]) {
  return health.reduce(
    (summary, item) => {
      if (item.status === "healthy") {
        summary.healthy += 1;
      } else if (item.status === "degraded") {
        summary.degraded += 1;
      } else {
        summary.offline += 1;
      }
      return summary;
    },
    { healthy: 0, degraded: 0, offline: 0 },
  );
}

export function pageTitle(pathname: string) {
  if (pathname === "/targets") {
    return {
      kicker: "Targets",
      title: "Dossiers and alias sets",
      summary: "Curate repeatable scan subjects instead of rebuilding usernames, emails, and domains every time.",
    };
  }
  if (pathname === "/new") {
    return {
      kicker: "Launch",
      title: "Start a scan",
      summary: "Compose a mission profile, tune runtime behavior, and open the graph workspace directly on completion.",
    };
  }
  if (pathname === "/settings") {
    return {
      kicker: "Settings",
      title: "Local defaults",
      summary: "Saved settings apply across the CLI and the browser workspace so scan behavior stays predictable.",
    };
  }
  return {
    kicker: "Overview",
    title: "Basalt home",
    summary: "Recent scans, module readiness, and the saved dossiers that shape repeatable investigations.",
  };
}
