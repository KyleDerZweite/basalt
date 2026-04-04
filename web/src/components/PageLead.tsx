type PageLeadProps = {
  kicker: string;
  title: string;
  summary: string;
  detail?: string;
};

export function PageLead({ kicker, title, summary, detail }: PageLeadProps) {
  return (
    <section className="page-lead">
      <div>
        <div className="section-kicker">{kicker}</div>
        <h1>{title}</h1>
        <p>{summary}</p>
      </div>
      {detail ? <div className="page-lead-detail mono">{detail}</div> : null}
    </section>
  );
}
