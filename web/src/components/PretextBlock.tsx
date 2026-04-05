import { createElement, type CSSProperties, useEffect, useRef, useState } from "react";
import { usePretextLayout } from "../hooks/usePretextLayout";

type PretextTag = "div" | "p" | "h1" | "h2" | "span";

interface PretextBlockProps {
  as?: PretextTag;
  className?: string;
  text: string;
  font: string;
  lineHeight: number;
  whiteSpace?: "normal" | "pre-wrap";
  style?: CSSProperties;
}

export function PretextBlock({
  as: Tag = "div",
  className,
  text,
  font,
  lineHeight,
  whiteSpace = "normal",
  style,
}: PretextBlockProps) {
  const ref = useRef<HTMLElement | null>(null);
  const [width, setWidth] = useState(0);
  const { height, lines } = usePretextLayout({
    text,
    font,
    lineHeight,
    width,
    whiteSpace,
  });

  useEffect(() => {
    const node = ref.current;
    if (!node) {
      return undefined;
    }

    const updateWidth = () => setWidth(node.clientWidth);
    updateWidth();

    const observer = new ResizeObserver(() => updateWidth());
    observer.observe(node);

    return () => observer.disconnect();
  }, []);

  return createElement(
    Tag,
    {
      ref: ref as never,
      className,
      style: {
        ...style,
        minHeight: height,
        ["--pretext-line-height" as string]: `${lineHeight}px`,
      },
    },
    width > 0
      ? lines.map((line, index) => (
          <span className="pretext-line" key={`${index}-${line}`}>
            {line}
          </span>
        ))
      : text,
  );
}
