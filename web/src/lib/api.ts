export async function api<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, {
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });
  const isJSON = (response.headers.get("content-type") ?? "").includes("application/json");
  const payload = isJSON ? (await response.json()) : await response.text();

  if (!response.ok) {
    if (isJSON && typeof payload === "object" && payload && "error" in payload) {
      throw new Error(String((payload as { error: string }).error));
    }
    throw new Error(typeof payload === "string" ? payload : response.statusText);
  }

  return payload as T;
}
