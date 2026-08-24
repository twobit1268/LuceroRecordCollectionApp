import { useEffect, useState } from "react";
import { catalogApi, collectionApi, type AlbumRecord, type CollectionEntry } from "./api";
import { Flags } from "./flags";
import { useFlag } from "./useFlag";

export default function CollectionView() {
  const [customerId, setCustomerId] = useState("demo-customer");
  const [entries, setEntries] = useState<CollectionEntry[]>([]);
  const [catalog, setCatalog] = useState<AlbumRecord[]>([]);
  const [selectedRecordId, setSelectedRecordId] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const exportCsv = useFlag(Flags.export_csv);

  async function loadCatalog() {
    try {
      const records = await catalogApi.list();
      setCatalog(records);
      if (records.length > 0) setSelectedRecordId(records[0].id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function loadCollection() {
    if (!customerId) return;
    setLoading(true);
    setError(null);
    try {
      const data = await collectionApi.list(customerId);
      setEntries(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { loadCatalog(); }, []);
  useEffect(() => { loadCollection(); }, [customerId]); // eslint-disable-line react-hooks/exhaustive-deps

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedRecordId) return;
    setError(null);
    try {
      await collectionApi.add(customerId, selectedRecordId);
      await loadCollection();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleRemove(entryId: string) {
    setError(null);
    try {
      await collectionApi.remove(customerId, entryId);
      await loadCollection();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function recordLabel(recordId: string): string {
    const record = catalog.find((r) => r.id === recordId);
    return record ? `${record.title} — ${record.artist}` : recordId;
  }

  function handleExportCsv() {
    const header = 'Record,Added\n';
    const rows = entries.map((e) => `"${recordLabel(e.recordId)}","${new Date(e.addedAt).toLocaleString()}"`).join('\n');
    const blob = new Blob([header + rows], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${customerId}-collection.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <section>
      <h2>
        Collection
        {exportCsv && <span className="flag-badge">⬇ Export CSV (CloudBees flag)</span>}
      </h2>

      <div className="row-form">
        <label>
          Customer ID{" "}
          <input value={customerId} onChange={(e) => setCustomerId(e.target.value)} />
        </label>
        {exportCsv && entries.length > 0 && (
          <button onClick={handleExportCsv} className="export-btn">Export CSV</button>
        )}
      </div>

      <form onSubmit={handleAdd} className="row-form">
        <select value={selectedRecordId} onChange={(e) => setSelectedRecordId(e.target.value)}>
          {catalog.length === 0 && <option value="">No records in catalog yet</option>}
          {catalog.map((r) => (
            <option key={r.id} value={r.id}>{r.title} — {r.artist}</option>
          ))}
        </select>
        <button type="submit" disabled={!selectedRecordId}>Add to collection</button>
      </form>

      {error && <p className="error">{error}</p>}
      {loading ? (
        <p>Loading…</p>
      ) : entries.length === 0 ? (
        <p className="empty">No records in this customer's collection yet.</p>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Record</th>
              <th>Added</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.id}>
                <td>{recordLabel(e.recordId)}</td>
                <td>{new Date(e.addedAt).toLocaleString()}</td>
                <td>
                  <button onClick={() => handleRemove(e.id)} className="link-button">Remove</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
