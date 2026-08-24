import { useState } from "react";
import CatalogView from "./CatalogView";
import CollectionView from "./CollectionView";
import ActivityView from "./ActivityView";
import { Flags } from "./flags";
import { useFlag } from "./useFlag";
import "./App.css";

type Tab = "catalog" | "collection" | "activity";

const TABS: { id: Tab; label: string }[] = [
  { id: "catalog", label: "Catalog" },
  { id: "collection", label: "Collection" },
  { id: "activity", label: "Activity" },
];

export default function App() {
  const [tab, setTab] = useState<Tab>("catalog");
  const darkMode = useFlag(Flags.dark_mode);

  return (
    <div className={`app${darkMode ? " dark" : ""}`}>
      <header>
        <h1>VinylVault</h1>
        <p className="subtitle">Distributed Go/GCP record-collection microservices</p>
        {darkMode && <span className="flag-badge">🌙 Dark Mode (CloudBees flag)</span>}
      </header>

      <nav>
        {TABS.map((t) => (
          <button
            key={t.id}
            className={t.id === tab ? "tab active" : "tab"}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
      </nav>

      <main>
        {tab === "catalog" && <CatalogView />}
        {tab === "collection" && <CollectionView />}
        {tab === "activity" && <ActivityView />}
      </main>
    </div>
  );
}
