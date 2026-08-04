import "server-only";

const apiURL = process.env.INTERNAL_API_URL || "http://localhost:8080";

export async function tempoFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiURL}${path}`, {
    ...init,
    cache: "no-store",
    headers: {
      Authorization: `Bearer ${process.env.INTERNAL_API_TOKEN || ""}`,
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
  if (!response.ok) throw new Error(`Tempo API returned ${response.status}`);
  return response.json() as Promise<T>;
}

export function todayRange() {
  const now = new Date();
  const from = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const to = new Date(from);
  to.setDate(to.getDate() + 1);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function recentRange(days: number) {
  const to = new Date();
  const from = new Date(to.getTime() - days * 24 * 60 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
}
