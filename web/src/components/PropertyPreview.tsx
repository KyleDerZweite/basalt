import { formatNodeType } from "../lib/format";

type PropertyPreviewProps = {
  properties?: Record<string, unknown>;
};

export function PropertyPreview({ properties }: PropertyPreviewProps) {
  if (!properties) {
    return null;
  }

  const entries = Object.entries(properties)
    .filter(([, value]) => value !== "" && value !== null && value !== undefined)
    .filter(([key]) => ["site_name", "profile_url", "full_name", "location", "website", "bio"].includes(key))
    .slice(0, 4);

  if (entries.length === 0) {
    return null;
  }

  return (
    <div className="detail-list compact">
      {entries.map(([key, value]) => (
        <div className="detail-row" key={key}>
          <strong>{formatNodeType(key)}</strong>
          <span>{String(value)}</span>
        </div>
      ))}
    </div>
  );
}
