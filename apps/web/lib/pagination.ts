export const TABLE_PAGE_SIZE = 15;

export function paginate<T>(items: T[], value: string | string[] | undefined, pageSize = TABLE_PAGE_SIZE) {
  const raw = Array.isArray(value) ? value[0] : value;
  const parsed = Number.parseInt(raw || "1", 10);
  const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
  const page = Math.min(Number.isFinite(parsed) && parsed > 0 ? parsed : 1, totalPages);
  const start = (page - 1) * pageSize;

  return {
    items: items.slice(start, start + pageSize),
    page,
    pageSize,
    totalItems: items.length,
    totalPages,
  };
}
