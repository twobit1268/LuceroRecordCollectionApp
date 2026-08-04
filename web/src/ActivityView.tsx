import { useEffect, useState } from "react";
import { activityApi, type ActivityEntry, type GenreCount } from "./api";

export default function ActivityView() {
  const [feed, setFeed] = useState<ActivityEntry[]>([]);
  const [genres, setGenres] = useState<GenreCount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const [feedData, genreData] = await Promise.all([activityApi.feed(), activityApi.genres()]);
      setFeed(feedData);
      setGenres(genreData);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const maxCount = Math.max(1, ...genres.map((g) => g.count));

  return (
    <section>
      <h2>Activity</h2>
      <p className="hint">
        This is fed asynchronously by GCP Pub/Sub — after adding a record to a collection, it can
        take a moment to show up here.
      </p>
      <button onClick={load} disabled={loading}>
        {loading ? "Refreshing…" : "Refresh"}
      </button>

      {error && <p className="error">{error}</p>}

      <h3>Genre counts</h3>
      {genres.length === 0 ? (
        <p className="empty">No activity yet.</p>
      ) : (
        <div className="bar-chart">
          {genres.map((g) => (
            <div className="bar-row" key={g.genre}>
              <span className="bar-label">{g.genre}</span>
              <div className="bar-track">
                <div className="bar-fill" style={{ width: `${(g.count / maxCount) * 100}%` }} />
              </div>
              <span className="bar-count">{g.count}</span>
            </div>
          ))}
        </div>
      )}

      <h3>Recent activity</h3>
      {feed.length === 0 ? (
        <p className="empty">No activity yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Customer</th>
              <th>Record ID</th>
              <th>Genre</th>
              <th>Added</th>
            </tr>
          </thead>
          <tbody>
            {feed.map((e) => (
              <tr key={e.id}>
                <td>{e.customerId}</td>
                <td className="mono">{e.recordId}</td>
                <td>{e.genre}</td>
                <td>{new Date(e.addedAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
