import type { Seed } from "../types";

export function updateSeed(seeds: Seed[], setSeeds: (value: Seed[]) => void, index: number, next: Partial<Seed>) {
  setSeeds(
    seeds.map((seed, itemIndex) =>
      itemIndex === index
        ? { ...seed, ...next }
        : seed,
    ),
  );
}

/** Guess the seed type from a raw value: email, domain, or username. */
export function detectSeedType(raw: string): Seed["type"] {
  const value = raw.trim();
  if (/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) return "email";
  if (!/\s/.test(value) && /^(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$/.test(value)) return "domain";
  return "username";
}
