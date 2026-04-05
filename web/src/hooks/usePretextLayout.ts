import { useMemo } from "react";
import { layout, layoutWithLines, prepare, prepareWithSegments } from "@chenglou/pretext";

type WhiteSpaceMode = "normal" | "pre-wrap";

interface BaseOptions {
  text: string;
  font: string;
  lineHeight: number;
  width: number;
  whiteSpace?: WhiteSpaceMode;
}

interface UsePretextHeightOptions extends BaseOptions {
  mode: "height";
}

interface UsePretextLinesOptions extends BaseOptions {
  mode?: "lines";
}

type UsePretextLayoutOptions = UsePretextHeightOptions | UsePretextLinesOptions;

const preparedCache = new Map<string, ReturnType<typeof prepare>>();
const preparedSegmentCache = new Map<string, ReturnType<typeof prepareWithSegments>>();

function cacheKey(text: string, font: string, whiteSpace: WhiteSpaceMode): string {
  return `${font}::${whiteSpace}::${text}`;
}

function getPreparedText(text: string, font: string, whiteSpace: WhiteSpaceMode) {
  const key = cacheKey(text, font, whiteSpace);
  const cached = preparedCache.get(key);
  if (cached) {
    return cached;
  }

  const prepared = prepare(text, font, { whiteSpace });
  preparedCache.set(key, prepared);
  return prepared;
}

function getPreparedSegments(text: string, font: string, whiteSpace: WhiteSpaceMode) {
  const key = cacheKey(text, font, whiteSpace);
  const cached = preparedSegmentCache.get(key);
  if (cached) {
    return cached;
  }

  const prepared = prepareWithSegments(text, font, { whiteSpace });
  preparedSegmentCache.set(key, prepared);
  return prepared;
}

export function usePretextLayout(options: UsePretextLayoutOptions) {
  const {
    text,
    font,
    lineHeight,
    width,
    whiteSpace = "normal",
  } = options;

  return useMemo(() => {
    const safeWidth = Math.max(1, Math.floor(width));
    const value = text.trim();

    if (!value) {
      return {
        height: lineHeight,
        lineCount: 1,
        lines: [text],
      };
    }

    if (options.mode === "height") {
      const prepared = getPreparedText(text, font, whiteSpace);
      const result = layout(prepared, safeWidth, lineHeight);
      return {
        height: result.height,
        lineCount: result.lineCount,
        lines: [text],
      };
    }

    const prepared = getPreparedSegments(text, font, whiteSpace);
    const result = layoutWithLines(prepared, safeWidth, lineHeight);

    return {
      height: result.height,
      lineCount: result.lineCount,
      lines: result.lines.map((line) => line.text || "\u00A0"),
    };
  }, [font, lineHeight, options.mode, text, whiteSpace, width]);
}
