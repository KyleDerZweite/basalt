import type { ReactNode } from "react";

type SectionHeadingProps = {
  kicker: string;
  title: string;
  summary?: string;
  action?: ReactNode;
};

export function SectionHeading({ kicker, title, summary, action }: SectionHeadingProps) {
  return (
    <div className="section-heading">
      <div>
        <div className="section-kicker">{kicker}</div>
        <h3>{title}</h3>
        {summary ? <p>{summary}</p> : null}
      </div>
      {action ? <div className="section-action">{action}</div> : null}
    </div>
  );
}
