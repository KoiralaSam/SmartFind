import { useCallback, useEffect, useState } from "react";
import { Trash2 } from "lucide-react";
import {
  staffCreateTransitRoute,
  staffDeleteTransitRoute,
  staffListTransitRoutes,
} from "../api/gateway";
import { useAuth } from "../context/useAuth";

const field =
  "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

export default function StaffTransitRoutesPage() {
  const { user } = useAuth();
  const [routes, setRoutes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [nameDraft, setNameDraft] = useState("");

  const load = useCallback(async () => {
    setError("");
    setLoading(true);
    try {
      const payload = await staffListTransitRoutes({ limit: 500, offset: 0 });
      setRoutes(Array.isArray(payload?.routes) ? payload.routes : []);
    } catch (e) {
      setError(e?.message || "Failed to load routes.");
      setRoutes([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function handleAdd(e) {
    e.preventDefault();
    const name = nameDraft.trim();
    if (!name || !user?.id) return;
    setSaving(true);
    setError("");
    try {
      await staffCreateTransitRoute({ staffId: user.id, routeName: name });
      setNameDraft("");
      await load();
    } catch (err) {
      setError(err?.message || "Could not create route.");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(route) {
    if (!user?.id || !route?.id) return;
    if (!window.confirm(`Remove transit route "${route.route_name}"?`)) return;
    setError("");
    try {
      await staffDeleteTransitRoute({ staffId: user.id, routeId: route.id });
      await load();
    } catch (err) {
      setError(err?.message || "Could not delete route.");
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Transit routes</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Shared catalog for your agency: every staff account sees the same routes. Anyone signed in as
          staff can add or remove entries.
        </p>
      </div>

      {error ? (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          {error}
        </div>
      ) : null}

      <form onSubmit={handleAdd} className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="min-w-0 flex-1 space-y-2">
          <label htmlFor="new-route-name" className="text-sm font-medium leading-none">
            Route name
          </label>
          <input
            id="new-route-name"
            type="text"
            value={nameDraft}
            onChange={(e) => setNameDraft(e.target.value)}
            className={field}
            placeholder='e.g. "Blue Line", "Route 42", "Central Station"'
            maxLength={200}
            autoComplete="off"
          />
        </div>
        <button
          type="submit"
          disabled={!nameDraft.trim() || saving || !user?.id}
          className="inline-flex h-11 shrink-0 items-center justify-center rounded-xl bg-foreground px-5 text-sm font-medium text-background transition hover:opacity-90 disabled:pointer-events-none disabled:opacity-40"
        >
          {saving ? "Adding…" : "Add route"}
        </button>
      </form>

      {loading ? (
        <div className="rounded-2xl border border-border bg-card p-8 text-center text-sm text-muted-foreground">
          Loading routes…
        </div>
      ) : routes.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center text-sm text-muted-foreground">
          No transit routes yet. Add one above.
        </div>
      ) : (
        <ul className="divide-y divide-border/80 rounded-2xl border border-border/70 bg-card">
          {routes.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between gap-3 px-4 py-3 sm:px-5"
              >
                <div className="min-w-0">
                  <p className="truncate font-medium text-foreground">{r.route_name}</p>
                  {r.created_by_staff_id ? (
                    <p className="truncate text-xs text-muted-foreground">
                      {String(r.created_by_staff_id) === String(user?.id)
                        ? "You added this route"
                        : "Added by a staff account"}
                    </p>
                  ) : null}
                </div>
                <button
                  type="button"
                  onClick={() => handleDelete(r)}
                  className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-lg border border-border px-3 text-xs font-medium text-destructive transition hover:bg-destructive/10"
                >
                  <Trash2 className="h-3.5 w-3.5" aria-hidden />
                  <span className="hidden sm:inline">Delete</span>
                </button>
              </li>
            ))}
        </ul>
      )}
    </div>
  );
}
