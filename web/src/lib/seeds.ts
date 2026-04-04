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
