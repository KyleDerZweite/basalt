type SummaryListItemProps = {
  label: string;
  items: string[];
};

export function SummaryListItem({ label, items }: SummaryListItemProps) {
  return (
    <div className="summary-group">
      <strong>{label}</strong>
      {items.length === 0 ? (
        <p>None</p>
      ) : (
        <ul>
          {items.map((item, index) => (
            <li key={`${label}-${index}`}>{item}</li>
          ))}
        </ul>
      )}
    </div>
  );
}
