import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  BarChart3,
  CheckCircle2,
  Clock,
  ImagePlus,
  LogOut,
  Package,
  Plus,
  Train,
  Upload,
  X,
} from "lucide-react";
import { useAuth } from "../context/useAuth";
import AnalyticsPanel from "./AnalyticsPanel";

const STORAGE_KEY = "smartfind-found-items";

const CATEGORIES = [
  "Bags & Luggage",
  "Electronics",
  "Clothing & Accessories",
  "Documents & Cards",
  "Keys",
  "Bottles & Containers",
  "Books & Stationery",
  "Toys & Games",
  "Other",
];

const MAX_PHOTOS = 5;

function loadItems() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

function saveItems(items) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
}

const field =
  "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";


// ─── Tab Button ──────────────────────────────────────────────
function TabButton({ active, icon: Icon, label, count, onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition ${
        active
          ? "bg-foreground text-background shadow-sm"
          : "text-muted-foreground hover:bg-muted hover:text-foreground"
      }`}
    >
      <Icon className="h-4 w-4" />
      <span className="hidden sm:inline">{label}</span>
      {count > 0 && (
        <span
          className={`ml-1 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full px-1.5 text-xs font-semibold ${
            active
              ? "bg-background/20 text-background"
              : "bg-muted-foreground/15 text-muted-foreground"
          }`}
        >
          {count}
        </span>
      )}
    </button>
  );
}

// ─── Stat Card ───────────────────────────────────────────────
function StatCard({ icon: Icon, label, value, accent }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
      <div className="flex items-center gap-3">
        <div
          className={`flex h-10 w-10 items-center justify-center rounded-xl ${accent}`}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <p className="text-2xl font-semibold tracking-tight">{value}</p>
          <p className="text-xs text-muted-foreground">{label}</p>
        </div>
      </div>
    </div>
  );
}

// ─── Item Card ───────────────────────────────────────────────
function ItemCard({ item, onClaim }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
      <div className="flex items-start gap-4">
        {item.image && (
          <img
            src={item.image}
            alt={item.itemName}
            className="h-16 w-16 shrink-0 rounded-xl border border-border object-cover"
          />
        )}
        <div className="flex min-w-0 flex-1 items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-1">
          <h3 className="truncate text-sm font-semibold">{item.itemName}</h3>
          <p className="text-xs text-muted-foreground">
            {item.category || "Uncategorized"}
            {item.color && item.color !== "unknown" ? ` · ${item.color}` : ""}
            {item.brand && item.brand !== "unknown" ? ` · ${item.brand}` : ""}
            {item.locationFound ? ` · ${item.locationFound}` : ""}
          </p>
          {item.description && (
            <p className="line-clamp-2 text-xs leading-relaxed text-muted-foreground">
              {item.description}
            </p>
          )}
          <p className="text-[11px] text-muted-foreground/70">
            Found {item.dateFound || "N/A"}
            {item.routeOrStation ? ` — ${item.routeOrStation}` : ""}
          </p>
          {item.status === "claimed" && item.claimedAt && (
            <p className="text-[11px] font-medium text-green-600">
              Claimed on {new Date(item.claimedAt).toLocaleDateString()}
            </p>
          )}
        </div>
        {onClaim && item.status === "unclaimed" && (
          <button
            type="button"
            onClick={() => onClaim(item.id)}
            className="shrink-0 rounded-xl border border-border bg-foreground px-3 py-1.5 text-xs font-medium text-background transition hover:opacity-90"
          >
            Mark Claimed
          </button>
        )}
        {item.status === "claimed" && (
          <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2.5 py-1 text-xs font-medium text-green-700">
            <CheckCircle2 className="h-3 w-3" />
            Claimed
          </span>
        )}
        </div>
      </div>
    </div>
  );
}

// ─── Main Dashboard ──────────────────────────────────────────
export default function StaffDashboard() {
  const { user, logout } = useAuth();
  const [tab, setTab] = useState("dashboard");
  const [items, setItems] = useState(loadItems);

  // Upload form state
  const [itemName, setItemName] = useState("");
  const [locationFound, setLocationFound] = useState("");
  const [routeOrStation, setRouteOrStation] = useState("");
  const [dateFound, setDateFound] = useState("");
  // photos: array of { id, url (data-URI for preview), data (base64 for API) }
  const [photos, setPhotos] = useState([]);
  const [uploadSuccess, setUploadSuccess] = useState(false);

  // AI-extracted detail fields
  const [extractedDetails, setExtractedDetails] = useState(null);
  const [extracting, setExtracting] = useState(false);
  const [extractError, setExtractError] = useState(null);
  // Editable fields pre-filled by AI
  const [editableDescription, setEditableDescription] = useState("");
  const [editableCategory, setEditableCategory] = useState("");

  useEffect(() => {
    saveItems(items);
  }, [items]);

  const unclaimed = items.filter((i) => i.status === "unclaimed");
  const claimed = items.filter((i) => i.status === "claimed");

  async function handleImageChange(e) {
    const files = Array.from(e.target.files || []);
    if (!files.length) return;

    // Enforce max photos limit
    const slots = MAX_PHOTOS - photos.length;
    const toAdd = files.slice(0, slots);
    const isFirst = photos.length === 0;

    // Read all selected files in parallel
    const newPhotos = await Promise.all(
      toAdd.map(
        (file) =>
          new Promise((resolve) => {
            const reader = new FileReader();
            reader.onload = (ev) =>
              resolve({ id: crypto.randomUUID(), url: ev.target.result, data: ev.target.result });
            reader.readAsDataURL(file);
          }),
      ),
    );

    setPhotos((prev) => [...prev, ...newPhotos]);

    // Run AI extraction on the primary (first) photo only
    if (isFirst && newPhotos.length > 0) {
      const primary = newPhotos[0];
      setExtracting(true);
      setExtractError(null);
      setExtractedDetails(null);
      setEditableDescription("");
      setEditableCategory("");
      try {
        const res = await fetch("/api/extract", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ image_base64: primary.data }),
        });
        if (!res.ok) throw new Error("Failed to analyze image");
        const data = await res.json();
        setExtractedDetails(data);
        if (!itemName && data.item_name && data.item_name !== "unknown") {
          setItemName(data.item_name);
        }
        setEditableDescription(data.item_description || "");
        setEditableCategory(data.category && data.category !== "unknown" ? data.category : "");
      } catch {
        setExtractError("Could not extract details from image. You can still fill in the details manually.");
      } finally {
        setExtracting(false);
      }
    }

    // Reset the file input so the same file can be re-selected
    e.target.value = "";
  }

  function removePhoto(id) {
    setPhotos((prev) => {
      const next = prev.filter((p) => p.id !== id);
      // If the primary photo was removed, clear AI results
      if (prev[0]?.id === id) {
        setExtractedDetails(null);
        setExtractError(null);
        setEditableDescription("");
        setEditableCategory("");
      }
      return next;
    });
  }

  const handleUpload = useCallback(
    (e) => {
      e.preventDefault();
      if (photos.length === 0) return;
      const newItem = {
        id: crypto.randomUUID(),
        itemName: itemName.trim(),
        description: editableDescription || extractedDetails?.item_description || "",
        category: editableCategory || extractedDetails?.category || "",
        itemType: extractedDetails?.item_type || "",
        brand: extractedDetails?.brand || "",
        model: extractedDetails?.model || "",
        color: extractedDetails?.color || "",
        material: extractedDetails?.material || "",
        itemCondition: extractedDetails?.item_condition || "",
        locationFound: locationFound.trim(),
        routeOrStation: routeOrStation.trim(),
        dateFound: dateFound || new Date().toISOString().split("T")[0],
        image: photos[0]?.url || null,      // primary photo for the card thumbnail
        images: photos.map((p) => p.url),   // all photos
        status: "unclaimed",
        postedBy: user?.email || "staff",
        createdAt: new Date().toISOString(),
      };
      setItems((prev) => [newItem, ...prev]);
      setItemName("");
      setLocationFound("");
      setRouteOrStation("");
      setDateFound("");
      setPhotos([]);
      setExtractedDetails(null);
      setExtractError(null);
      setEditableDescription("");
      setEditableCategory("");
      setUploadSuccess(true);
      setTimeout(() => setUploadSuccess(false), 3000);
    },
    [itemName, locationFound, routeOrStation, dateFound, photos, user, extractedDetails, editableDescription, editableCategory],
  );

  const handleClaim = useCallback((id) => {
    setItems((prev) =>
      prev.map((item) =>
        item.id === id
          ? { ...item, status: "claimed", claimedAt: new Date().toISOString() }
          : item,
      ),
    );
  }, []);

  return (
    <div className="min-h-screen bg-gradient-to-b from-muted/40 to-background">
      {/* ─── Header ─────────────────────────────────────────── */}
      <header className="sticky top-0 z-20 border-b border-border/80 bg-background/90 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <Link
            to="/"
            className="flex items-center gap-2 font-semibold tracking-tight hover:opacity-90"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-foreground text-background">
              <Train className="h-4 w-4" aria-hidden />
            </span>
            <span className="hidden sm:inline">SmartFind Staff</span>
          </Link>

          <div className="flex items-center gap-3 text-sm">
            <span className="hidden max-w-[140px] truncate text-muted-foreground sm:inline sm:max-w-[200px]">
              {user?.name}
            </span>
            <button
              type="button"
              onClick={() => logout()}
              className="inline-flex items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-xs font-medium hover:bg-muted sm:text-sm"
            >
              <LogOut className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Sign out</span>
            </button>
          </div>
        </div>
      </header>

      {/* ─── Tab Navigation ─────────────────────────────────── */}
      <nav className="border-b border-border/60 bg-background/60 backdrop-blur-sm">
        <div className="mx-auto flex max-w-5xl gap-1 overflow-x-auto px-4 py-2">
          <TabButton
            active={tab === "dashboard"}
            icon={Package}
            label="Dashboard"
            count={0}
            onClick={() => setTab("dashboard")}
          />
          <TabButton
            active={tab === "upload"}
            icon={Upload}
            label="Upload Item"
            count={0}
            onClick={() => setTab("upload")}
          />
          <TabButton
            active={tab === "in-progress"}
            icon={Clock}
            label="In Progress"
            count={unclaimed.length}
            onClick={() => setTab("in-progress")}
          />
          <TabButton
            active={tab === "claimed"}
            icon={CheckCircle2}
            label="Claimed"
            count={claimed.length}
            onClick={() => setTab("claimed")}
          />
          <TabButton
            active={tab === "analytics"}
            icon={BarChart3}
            label="Analytics"
            count={0}
            onClick={() => setTab("analytics")}
          />
        </div>
      </nav>

      {/* ─── Content ────────────────────────────────────────── */}
      <main className="mx-auto max-w-5xl px-4 py-8">
        {/* Dashboard */}
        {tab === "dashboard" && (
          <div className="space-y-8">
            <div>
              <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                Console
              </p>
              <h1 className="mt-1 text-2xl font-semibold tracking-tight md:text-3xl">
                Hi, {user?.name}
              </h1>
              <p className="mt-1 max-w-lg text-sm leading-relaxed text-muted-foreground">
                Manage found items, track unclaimed items, and process claims.
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
              <StatCard
                icon={Package}
                label="Total Items"
                value={items.length}
                accent="bg-foreground/10 text-foreground"
              />
              <StatCard
                icon={Clock}
                label="Unclaimed"
                value={unclaimed.length}
                accent="bg-amber-100 text-amber-700"
              />
              <StatCard
                icon={CheckCircle2}
                label="Claimed"
                value={claimed.length}
                accent="bg-green-100 text-green-700"
              />
            </div>

            {/* Recent items preview */}
            <div>
              <h2 className="mb-3 text-sm font-semibold">Recent Items</h2>
              {items.length === 0 ? (
                <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center">
                  <Package className="mx-auto mb-3 h-8 w-8 text-muted-foreground/50" />
                  <p className="text-sm text-muted-foreground">
                    No items yet. Upload a found item to get started.
                  </p>
                  <button
                    type="button"
                    onClick={() => setTab("upload")}
                    className="mt-4 inline-flex items-center gap-1.5 rounded-xl bg-foreground px-4 py-2 text-sm font-medium text-background transition hover:opacity-90"
                  >
                    <Plus className="h-4 w-4" />
                    Upload Item
                  </button>
                </div>
              ) : (
                <div className="space-y-3">
                  {items.slice(0, 5).map((item) => (
                    <ItemCard
                      key={item.id}
                      item={item}
                      onClaim={item.status === "unclaimed" ? handleClaim : undefined}
                    />
                  ))}
                  {items.length > 5 && (
                    <p className="text-center text-xs text-muted-foreground">
                      Showing 5 of {items.length} items.{" "}
                      <button
                        type="button"
                        onClick={() => setTab("in-progress")}
                        className="font-medium text-foreground underline underline-offset-2"
                      >
                        View all
                      </button>
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Upload Found Item */}
        {tab === "upload" && (
          <div className="mx-auto max-w-lg space-y-6">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">
                Upload Found Item
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Log a new item that was found on transit.
              </p>
            </div>

            {uploadSuccess && (
              <div className="rounded-xl border border-green-200 bg-green-50 p-4 text-sm text-green-700">
                Item uploaded successfully! It is now listed as unclaimed.
              </div>
            )}

            <form onSubmit={handleUpload} className="space-y-5">
              <div className="space-y-2">
                <label htmlFor="item-name" className="text-sm font-medium leading-none">
                  Item Name <span className="text-destructive">*</span>
                </label>
                <input
                  id="item-name"
                  type="text"
                  required
                  value={itemName}
                  onChange={(e) => setItemName(e.target.value)}
                  className={field}
                  placeholder="e.g. Black backpack"
                />
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <label htmlFor="item-location" className="text-sm font-medium leading-none">
                    Location Found
                  </label>
                  <input
                    id="item-location"
                    type="text"
                    value={locationFound}
                    onChange={(e) => setLocationFound(e.target.value)}
                    className={field}
                    placeholder="e.g. Bus seat 14A"
                  />
                </div>
                <div className="space-y-2">
                  <label htmlFor="item-route" className="text-sm font-medium leading-none">
                    Route / Station
                  </label>
                  <input
                    id="item-route"
                    type="text"
                    value={routeOrStation}
                    onChange={(e) => setRouteOrStation(e.target.value)}
                    className={field}
                    placeholder="e.g. Route 42"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label htmlFor="item-date" className="text-sm font-medium leading-none">
                  Date Found <span className="text-destructive">*</span>
                </label>
                <input
                  id="item-date"
                  type="date"
                  required
                  value={dateFound}
                  onChange={(e) => setDateFound(e.target.value)}
                  className={field}
                />
              </div>

              {/* ── Photo Upload ─────────────────────────────── */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium leading-none">
                    Photos <span className="text-destructive">*</span>
                  </label>
                  <span className="text-xs text-muted-foreground">
                    {photos.length}/{MAX_PHOTOS} · first photo used for AI
                  </span>
                </div>

                <div className="flex flex-wrap gap-3">
                  {/* Thumbnail grid */}
                  {photos.map((photo, idx) => (
                    <div key={photo.id} className="relative">
                      <img
                        src={photo.url}
                        alt={`Photo ${idx + 1}`}
                        className="h-20 w-20 rounded-xl border border-border object-cover"
                      />
                      {/* Primary badge */}
                      {idx === 0 && (
                        <span className="absolute bottom-1 left-1 rounded-md bg-foreground/80 px-1.5 py-0.5 text-[9px] font-semibold text-background">
                          Primary
                        </span>
                      )}
                      <button
                        type="button"
                        onClick={() => removePhoto(photo.id)}
                        className="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-destructive-foreground shadow-sm transition hover:opacity-90"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </div>
                  ))}

                  {/* Add more / initial upload button */}
                  {photos.length < MAX_PHOTOS && (
                    <label
                      htmlFor="item-image"
                      className={`flex cursor-pointer flex-col items-center justify-center gap-1.5 rounded-xl border-2 border-dashed border-border bg-muted/30 transition hover:border-muted-foreground/40 hover:bg-muted/50 ${
                        photos.length === 0 ? "h-20 w-full" : "h-20 w-20"
                      }`}
                    >
                      <ImagePlus className="h-5 w-5 text-muted-foreground/50" />
                      <span className="text-[10px] text-muted-foreground">
                        {photos.length === 0 ? "Click to upload photos" : "Add more"}
                      </span>
                      <input
                        id="item-image"
                        type="file"
                        accept="image/*"
                        multiple
                        className="hidden"
                        onChange={handleImageChange}
                      />
                    </label>
                  )}
                </div>
              </div>

              {/* ── AI Extraction Status ──────────────────────── */}
              {extracting && (
                <div className="flex items-center gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
                  <p className="text-sm text-blue-700">
                    AI is analysing the primary photo…
                  </p>
                </div>
              )}

              {extractError && (
                <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">
                  {extractError}
                </div>
              )}

              {/* ── AI Results — editable ─────────────────────── */}
              {extractedDetails && !extracting && (
                <div className="space-y-4 rounded-xl border border-border bg-muted/30 p-4">
                  <div className="flex items-center justify-between">
                    <p className="text-xs font-semibold uppercase tracking-[0.15em] text-muted-foreground">
                      AI Extracted Details
                    </p>
                    <span className="rounded-full bg-green-100 px-2 py-0.5 text-[10px] font-medium text-green-700">
                      Review &amp; edit if needed
                    </span>
                  </div>

                  {/* Editable category */}
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-muted-foreground">
                      Category
                    </label>
                    <select
                      value={editableCategory}
                      onChange={(e) => setEditableCategory(e.target.value)}
                      className={field}
                    >
                      <option value="">— Select category —</option>
                      {CATEGORIES.map((c) => (
                        <option key={c} value={c}>{c}</option>
                      ))}
                    </select>
                  </div>

                  {/* Editable description */}
                  <div className="space-y-1.5">
                    <label className="text-xs font-medium text-muted-foreground">
                      Description
                    </label>
                    <textarea
                      rows={3}
                      value={editableDescription}
                      onChange={(e) => setEditableDescription(e.target.value)}
                      className="flex w-full rounded-xl border border-input bg-background px-3.5 py-2.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      placeholder="Describe the item…"
                    />
                  </div>

                  {/* Read-only detail chips */}
                  <div className="flex flex-wrap gap-2">
                    {[
                      { label: "Color", value: extractedDetails.color },
                      { label: "Brand", value: extractedDetails.brand },
                      { label: "Material", value: extractedDetails.material },
                      { label: "Condition", value: extractedDetails.item_condition },
                    ]
                      .filter(({ value }) => value && value !== "unknown")
                      .map(({ label, value }) => (
                        <span
                          key={label}
                          className="rounded-full border border-border bg-background px-2.5 py-1 text-[11px] text-muted-foreground"
                        >
                          <span className="font-medium text-foreground">{label}:</span> {value}
                        </span>
                      ))}
                  </div>
                </div>
              )}

              {photos.length === 0 && (
                <p className="text-sm text-destructive">
                  At least one photo is required.
                </p>
              )}

              <button
                type="submit"
                disabled={photos.length === 0 || extracting}
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <Upload className="h-4 w-4" />
                Upload Item
              </button>
            </form>
          </div>
        )}

        {/* In Progress (Unclaimed) */}
        {tab === "in-progress" && (
          <div className="space-y-6">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">
                In Progress
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Items that have been found but not yet claimed by their owner.
              </p>
            </div>

            {unclaimed.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center">
                <Clock className="mx-auto mb-3 h-8 w-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">
                  No unclaimed items at the moment.
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {unclaimed.map((item) => (
                  <ItemCard key={item.id} item={item} onClaim={handleClaim} />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Claimed */}
        {tab === "claimed" && (
          <div className="space-y-6">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">
                Claimed Items
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Items that have been successfully returned to their owner.
              </p>
            </div>

            {claimed.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center">
                <CheckCircle2 className="mx-auto mb-3 h-8 w-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">
                  No claimed items yet.
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {claimed.map((item) => (
                  <ItemCard key={item.id} item={item} />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Analytics */}
        {tab === "analytics" && <AnalyticsPanel />}
      </main>
    </div>
  );
}