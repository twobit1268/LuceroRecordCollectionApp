import { useEffect, useState } from "react";
import { catalogApi, type AlbumRecord } from "./api";

export default function CatalogView() {
  const [records, setRecords] = useState<AlbumRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState({ title: "", artist: "", genre: "", year: "" });
  const [submitting, setSubmitting] = useState(false);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const data = await catalogApi.list();
      setRecords(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await catalogApi.create({
        title: form.title,
        artist: form.artist,
        genre: form.genre,
        year: Number(form.year),
      });
      setForm({ title: "", artist: "", genre: "", year: "" });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section>
      <h2>Catalog</h2>

      <form onSubmit={handleSubmit} className="row-form">
        <input
          placeholder="Title"
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
          required
        />
        <input
          placeholder="Artist"
          value={form.artist}
          onChange={(e) => setForm({ ...form, artist: e.target.value })}
          required
        />
        <input
          placeholder="Genre"
          value={form.genre}
          onChange={(e) => setForm({ ...form, genre: e.target.value })}
          required
        />
        <input
          placeholder="Year"
          type="number"
          value={form.year}
          onChange={(e) => setForm({ ...form, year: e.target.value })}
          required
        />
        <button type="submit" disabled={submitting}>
          {submitting ? "Adding…" : "Add record"}
        </button>
      </form>

      {error && <p className="error">{error}</p>}
      {loading ? (
        <p>Loading…</p>
      ) : records.length === 0 ? (
        <p className="empty">No records yet — add one above.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Title</th>
              <th>Artist</th>
              <th>Genre</th>
              <th>Year</th>
              <th>ID</th>
            </tr>
          </thead>
          <tbody>
            {records.map((r) => (
              <tr key={r.id}>
                <td>{r.title}</td>
                <td>{r.artist}</td>
                <td>{r.genre}</td>
                <td>{r.year}</td>
                <td className="mono">{r.id}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
