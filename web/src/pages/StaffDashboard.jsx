import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  BarChart3,
  Camera,
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
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";
import {
  staffCreateFoundItem,
  staffListFoundItems,
  staffUpdateFoundItemStatus,
} from "../api/gateway";
import AnalyticsPanel from "./AnalyticsPanel";

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

const field =
  "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

function mapFoundItemDTO(dto) {
  if (!dto) return null;
  const dateStr = dto.date_found ? String(dto.date_found).slice(0, 10) : "";
  return {
    id: dto.id,
    itemName: dto.item_name,
    description: dto.item_description || "",
    category: dto.category || "",
    itemType: dto.item_type || "",
    brand: dto.brand || "",
    model: dto.model || "",
    color: dto.color || "",
    material: dto.material || "",
    itemCondition: dto.item_condition || "",
    locationFound: dto.location_found || "",
    routeOrStation: dto.route_or_station || "",
    routeId: dto.route_id || "",
    dateFound: dateStr,
    status: dto.status || "unclaimed",
    createdAt: dto.created_at || null,
    updatedAt: dto.updated_at || null,
    claimedAt: dto.status === "claimed" ? dto.updated_at || null : null,
    image: null,
    images: [],
  };
}

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
  const [items, setItems] = useState([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  const [itemsError, setItemsError] = useState("");

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
  const [uploadError, setUploadError] = useState("");

  const [cameraOpen, setCameraOpen] = useState(false);
  const [cameraError, setCameraError] = useState(null);
  const videoRef = useRef(null);
  const streamRef = useRef(null);

  const refreshItems = useCallback(async () => {
    if (!user?.id) return;
    setItemsLoading(true);
    setItemsError("");
    try {
      const payload = await staffListFoundItems({
        postedByStaffId: user.id,
        limit: 200,
        offset: 0,
      });
      const next = (payload?.items || [])
        .map(mapFoundItemDTO)
        .filter(Boolean);
      setItems(next);
    } catch (err) {
      setItemsError(err?.message || "Failed to load found items.");
    } finally {
      setItemsLoading(false);
    }
  }, [user?.id]);

  useEffect(() => {
    refreshItems();
  }, [refreshItems]);

  const unclaimed = items.filter((i) => i.status === "unclaimed");
  const claimed = items.filter((i) => i.status === "claimed");

  const runExtractionOnPrimary = useCallback(
    async (primary) => {
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
        setEditableCategory(
          data.category && data.category !== "unknown" ? data.category : "",
        );
      } catch {
        setExtractError(
          "Could not extract details from image. You can still fill in the details manually.",
        );
      } finally {
        setExtracting(false);
      }
    },
    [itemName],
  );

  const appendPhotos = useCallback(
    async (newPhotoEntries) => {
      if (!newPhotoEntries.length) return;
      let toAdd = [];
      let wasEmpty = false;
      setPhotos((prev) => {
        const slots = MAX_PHOTOS - prev.length;
        toAdd = newPhotoEntries.slice(0, slots);
        wasEmpty = prev.length === 0;
        if (!toAdd.length) return prev;
        return [...prev, ...toAdd];
      });
      if (!toAdd.length) return;
      if (wasEmpty && toAdd.length > 0) {
        await runExtractionOnPrimary(toAdd[0]);
      }
    },
    [runExtractionOnPrimary],
  );

  async function handleImageChange(e) {
    const files = Array.from(e.target.files || []);
    if (!files.length) return;

    const slots = MAX_PHOTOS - photos.length;
    const toRead = files.slice(0, slots);

    const newPhotos = await Promise.all(
      toRead.map(
        (file) =>
          new Promise((resolve) => {
            const reader = new FileReader();
            reader.onload = (ev) =>
              resolve({
                id: crypto.randomUUID(),
                url: ev.target.result,
                data: ev.target.result,
              });
            reader.readAsDataURL(file);
          }),
      ),
    );

    await appendPhotos(newPhotos);
    e.target.value = "";
  }

  useEffect(() => {
    if (!cameraOpen) return undefined;

    let cancelled = false;
    let stream = null;

    (async () => {
      setCameraError(null);
      try {
        stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: "environment" } },
          audio: false,
        });
        if (cancelled) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }
        streamRef.current = stream;
        const video = videoRef.current;
        if (video) {
          video.srcObject = stream;
          await video.play();
        }
      } catch {
        if (!cancelled) {
          setCameraError(
            "Could not access the camera. Allow permission or use file upload.",
          );
        }
      }
    })();

    return () => {
      cancelled = true;
      if (stream) stream.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
      const video = videoRef.current;
      if (video) video.srcObject = null;
    };
  }, [cameraOpen]);

  function closeCamera() {
    setCameraOpen(false);
    setCameraError(null);
  }

  async function captureFromCamera() {
    const video = videoRef.current;
    if (!video || video.readyState < 2) return;
    const w = video.videoWidth;
    const h = video.videoHeight;
    if (!w || !h) return;

    const canvas = document.createElement("canvas");
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.drawImage(video, 0, 0);
    const dataUrl = canvas.toDataURL("image/jpeg", 0.92);
    const entry = {
      id: crypto.randomUUID(),
      url: dataUrl,
      data: dataUrl,
    };
    await appendPhotos([entry]);
    closeCamera();
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
    async (e) => {
      e.preventDefault();
      if (photos.length === 0) return;
      if (!user?.id) return;
      setUploadError("");

      const dateISO = dateFound
        ? new Date(`${dateFound}T00:00:00Z`).toISOString()
        : undefined;

      try {
        const created = await staffCreateFoundItem({
          staff_id: user.id,
          item_name: itemName.trim(),
          item_description:
            editableDescription ||
            extractedDetails?.item_description ||
            "",
          item_type: extractedDetails?.item_type || "",
          brand: extractedDetails?.brand || "",
          model: extractedDetails?.model || "",
          color: extractedDetails?.color || "",
          material: extractedDetails?.material || "",
          item_condition: extractedDetails?.item_condition || "",
          category: editableCategory || extractedDetails?.category || "",
          location_found: locationFound.trim(),
          route_or_station: routeOrStation.trim(),
          route_id: "",
          date_found: dateISO,
        });

        const mapped = mapFoundItemDTO(created) || {
          id: crypto.randomUUID(),
          itemName: itemName.trim(),
          description:
            editableDescription ||
            extractedDetails?.item_description ||
            "",
          category: editableCategory || extractedDetails?.category || "",
          locationFound: locationFound.trim(),
          routeOrStation: routeOrStation.trim(),
          dateFound,
          status: "unclaimed",
          image: photos[0]?.url || null,
          images: photos.map((p) => p.url),
        };
        mapped.image = photos[0]?.url || mapped.image || null;
        mapped.images = photos.map((p) => p.url);

        setItems((prev) => [mapped, ...prev]);
      } catch (err) {
        setUploadError(err?.message || "Failed to upload item.");
        return;
      }

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

  const handleClaim = useCallback(
    async (id) => {
      if (!user?.id) return;
      try {
        const updated = await staffUpdateFoundItemStatus({
          staff_id: user.id,
          found_item_id: id,
          status: "claimed",
        });
        const mapped = mapFoundItemDTO(updated);
        setItems((prev) =>
          prev.map((item) => (item.id === id ? { ...item, ...(mapped || {}) } : item)),
        );
      } catch (err) {
        setItemsError(err?.message || "Failed to update item status.");
      }
    },
    [user?.id],
  );

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
            <AccountAvatar user={user} />
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
        {itemsError ? (
          <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">
            {itemsError}{" "}
            <button
              type="button"
              onClick={refreshItems}
              className="ml-2 font-medium text-foreground underline underline-offset-2"
            >
              Retry
            </button>
          </div>
        ) : null}

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
              {itemsLoading ? (
                <div className="rounded-2xl border border-border bg-card p-8 text-center">
                  <p className="text-sm text-muted-foreground">Loading items…</p>
                </div>
              ) : items.length === 0 ? (
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
            {uploadError ? (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-700">
                {uploadError}
              </div>
            ) : null}

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

                <div className="space-y-3">
                  <div className="flex flex-wrap gap-3">
                    {photos.map((photo, idx) => (
                      <div key={photo.id} className="relative">
                        <img
                          src={photo.url}
                          alt={`Photo ${idx + 1}`}
                          className="h-20 w-20 rounded-xl border border-border object-cover"
                        />
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
                  </div>

                  {photos.length < MAX_PHOTOS && (
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-stretch">
                      <label
                        htmlFor="item-image"
                        className="flex min-h-[5rem] flex-1 cursor-pointer flex-col items-center justify-center gap-1.5 rounded-xl border-2 border-dashed border-border bg-muted/30 px-3 transition hover:border-muted-foreground/40 hover:bg-muted/50"
                      >
                        <ImagePlus className="h-5 w-5 text-muted-foreground/50" />
                        <span className="text-center text-[10px] text-muted-foreground">
                          {photos.length === 0
                            ? "Upload from gallery"
                            : "Add from gallery"}
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
                      <button
                        type="button"
                        onClick={() => setCameraOpen(true)}
                        disabled={extracting}
                        className="flex min-h-[5rem] shrink-0 flex-col items-center justify-center gap-1.5 rounded-xl border-2 border-dashed border-border bg-muted/30 px-4 transition hover:border-muted-foreground/40 hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50 sm:w-36"
                      >
                        <Camera className="h-5 w-5 text-muted-foreground/50" />
                        <span className="text-[10px] text-muted-foreground">
                          Take photo
                        </span>
                      </button>
                    </div>
                  )}
                </div>
              </div>

              {cameraOpen && (
                <div
                  className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
                  role="dialog"
                  aria-modal="true"
                  aria-labelledby="camera-dialog-title"
                >
                  <div className="w-full max-w-lg overflow-hidden rounded-2xl border border-border bg-background shadow-lg">
                    <div className="flex items-center justify-between border-b border-border px-4 py-3">
                      <h2
                        id="camera-dialog-title"
                        className="text-sm font-semibold"
                      >
                        Take a photo
                      </h2>
                      <button
                        type="button"
                        onClick={closeCamera}
                        className="rounded-lg p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground"
                        aria-label="Close camera"
                      >
                        <X className="h-5 w-5" />
                      </button>
                    </div>
                    <div className="space-y-3 p-4">
                      <div className="relative aspect-[4/3] w-full overflow-hidden rounded-xl bg-black">
                        <video
                          ref={videoRef}
                          playsInline
                          muted
                          className="h-full w-full object-cover"
                        />
                        {cameraError && (
                          <div className="absolute inset-0 flex items-center justify-center bg-black/70 p-4 text-center text-sm text-white">
                            {cameraError}
                          </div>
                        )}
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          onClick={captureFromCamera}
                          disabled={!!cameraError}
                          className="inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-foreground px-4 py-2.5 text-sm font-medium text-background transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                        >
                          <Camera className="h-4 w-4" />
                          Use photo
                        </button>
                        <button
                          type="button"
                          onClick={closeCamera}
                          className="rounded-xl border border-border px-4 py-2.5 text-sm font-medium transition hover:bg-muted"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}

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
