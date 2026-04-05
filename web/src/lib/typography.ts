export const fontFamilies = {
  display: '"IBM Plex Mono", "SFMono-Regular", Consolas, "Liberation Mono", monospace',
  body: '"JetBrains Mono", "SFMono-Regular", Consolas, "Liberation Mono", monospace',
} as const;

export const lineHeights = {
  compact: 18,
  ui: 20,
  body: 22,
  card: 24,
  title: 34,
} as const;

export function buildCanvasFont(weight: number, size: number, family: keyof typeof fontFamilies): string {
  return `${weight} ${size}px ${fontFamilies[family]}`;
}

export const pretextFonts = {
  kicker: buildCanvasFont(600, 12, "display"),
  section: buildCanvasFont(600, 12, "display"),
  pageTitle: buildCanvasFont(600, 28, "display"),
  pageDescription: buildCanvasFont(400, 14, "body"),
  cardTitle: buildCanvasFont(600, 13, "display"),
  emptyTitle: buildCanvasFont(600, 16, "display"),
  emptyDescription: buildCanvasFont(400, 13, "body"),
  findingTitle: buildCanvasFont(600, 13, "display"),
  findingSummary: buildCanvasFont(400, 12, "body"),
  eventMessage: buildCanvasFont(400, 12, "body"),
  eventMeta: buildCanvasFont(400, 11, "body"),
  targetName: buildCanvasFont(600, 15, "display"),
  targetNotes: buildCanvasFont(400, 13, "body"),
  inspectorLabel: buildCanvasFont(600, 17, "display"),
  inspectorValue: buildCanvasFont(400, 12, "body"),
  insightHeadline: buildCanvasFont(400, 13, "body"),
  detailTitle: buildCanvasFont(700, 14, "display"),
  previewValue: buildCanvasFont(400, 12, "body"),
} as const;
