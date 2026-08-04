// Thin fetch wrappers around the three services. Base URLs are configurable
// via Vite env vars so the same build works against docker-compose's
// service names or plain localhost, without a code change.
const CATALOG_URL = import.meta.env.VITE_CATALOG_URL ?? "http://localhost:8081";
const COLLECTION_URL = import.meta.env.VITE_COLLECTION_URL ?? "http://localhost:8082";
const ACTIVITY_URL = import.meta.env.VITE_ACTIVITY_URL ?? "http://localhost:8083";

export interface AlbumRecord {
  id: string;
  title: string;
  artist: string;
  genre: string;
  year: number;
  createdAt: string;
}

export interface CollectionEntry {
  id: string;
  customerId: string;
  recordId: string;
  addedAt: string;
}

export interface ActivityEntry {
  id: string;
  customerId: string;
  recordId: string;
  genre: string;
  addedAt: string;
  receivedAt: string;
}

export interface GenreCount {
  genre: string;
  count: number;
}

async function asJson<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed with status ${res.status}`);
  }
  return res.json() as Promise<T>;
}

const jsonHeaders = { "Content-Type": "application/json" };

export const catalogApi = {
  list: (): Promise<AlbumRecord[]> =>
    fetch(`${CATALOG_URL}/records`).then((r) => asJson<AlbumRecord[]>(r)),

  create: (input: { title: string; artist: string; genre: string; year: number }): Promise<AlbumRecord> =>
    fetch(`${CATALOG_URL}/records`, {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify(input),
    }).then((r) => asJson<AlbumRecord>(r)),
};

export const collectionApi = {
  list: (customerId: string): Promise<CollectionEntry[]> =>
    fetch(`${COLLECTION_URL}/collections/${encodeURIComponent(customerId)}/records`).then((r) =>
      asJson<CollectionEntry[]>(r)
    ),

  add: (customerId: string, recordId: string): Promise<CollectionEntry> =>
    fetch(`${COLLECTION_URL}/collections/${encodeURIComponent(customerId)}/records`, {
      method: "POST",
      headers: jsonHeaders,
      body: JSON.stringify({ recordId }),
    }).then((r) => asJson<CollectionEntry>(r)),

  remove: async (customerId: string, entryId: string): Promise<void> => {
    const res = await fetch(
      `${COLLECTION_URL}/collections/${encodeURIComponent(customerId)}/records/${entryId}`,
      { method: "DELETE" }
    );
    if (!res.ok && res.status !== 204) {
      throw new Error(`Failed to remove entry (status ${res.status})`);
    }
  },
};

export const activityApi = {
  feed: (limit = 20): Promise<ActivityEntry[]> =>
    fetch(`${ACTIVITY_URL}/activity/feed?limit=${limit}`).then((r) => asJson<ActivityEntry[]>(r)),

  genres: (): Promise<GenreCount[]> =>
    fetch(`${ACTIVITY_URL}/activity/genres`).then((r) => asJson<GenreCount[]>(r)),
};
