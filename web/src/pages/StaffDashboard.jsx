import { useCallback, useEffect, useRef, useState } from "react";
import { Link, Navigate, NavLink, useNavigate, useParams } from "react-router-dom";
import {
  BarChart3,
  Camera,
  CheckCircle2,
  Clock,
  ImagePlus,
  LogOut,
  MapPin,
  Package,
  Pencil,
  Plus,
  Trash2,
  Train,
  Upload,
  X,
} from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";
import {
  staffCreateFoundItem,
  staffListFoundItems,
  staffListClaims,
  staffListTransitRoutes,
  staffUpdateFoundItem,
  staffUpdateFoundItemStatus,
  staffDeleteFoundItem,
  mediaInitUploads,
  mediaDeleteUpload,
} from "../api/gateway";
import AnalyticsPanel from "./AnalyticsPanel";
import StaffTransitRoutesPage from "./StaffTransitRoutesPage";

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

function isKnownValue(v) {
  const s = String(v ?? "").trim();
  return s !== "" && s !== "unknown";
}

function mergeDescription(prev, next) {
  const a = String(prev ?? "").trim();
  const b = String(next ?? "").trim();
  if (!a) return b;
  if (!b) return a;
  if (a.includes(b)) return a;
  return `${a}\n\nAdditional photo findings: ${b}`;
}

function mergeExtractedDetails(prev, next) {
  if (!next) return prev;
  if (!prev) return next;
  return {
    item_name: isKnownValue(prev.item_name) ? prev.item_name : next.item_name,
    item_type: isKnownValue(prev.item_type) ? prev.item_type : next.item_type,
    category: isKnownValue(prev.category) ? prev.category : next.category,
    brand: isKnownValue(prev.brand) ? prev.brand : next.brand,
    model: isKnownValue(prev.model) ? prev.model : next.model,
    color: isKnownValue(prev.color) ? prev.color : next.color,
    material: isKnownValue(prev.material) ? prev.material : next.material,
    item_condition: isKnownValue(prev.item_condition)
      ? prev.item_condition
      : next.item_condition,
    item_description: mergeDescription(prev.item_description, next.item_description),
  };
}

function mapFoundItemDTO(dto) {
  if (!dto) return null;
  const dateStr = dto.date_found ? String(dto.date_found).slice(0, 10) : "";
  const images = Array.isArray(dto.images) ? dto.images.filter(Boolean) : [];
  const primary = dto.image || images[0] || null;
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
    image: primary,
    images,
  };
}

const STAFF_SECTIONS = new Set([
  "dashboard",
  "upload",
  "in-progress",
  "claims",
  "analytics",
  "routes",
]);

// ─── Tab link (URL-backed) ───────────────────────────────────
function StaffTabLink({ to, icon: Icon, label, count }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-medium transition ${
          isActive
            ? "bg-foreground text-background shadow-sm"
            : "text-muted-foreground hover:bg-muted hover:text-foreground"
        }`
      }
    >
      {({ isActive }) => (
        <>
          <Icon className="h-4 w-4" />
          <span className="hidden sm:inline">{label}</span>
          {count > 0 && (
            <span
              className={`ml-1 inline-flex h-5 min-w-[20px] items-center justify-center rounded-full px-1.5 text-xs font-semibold ${
                isActive
                  ? "bg-background/20 text-background"
                  : "bg-muted-foreground/15 text-muted-foreground"
              }`}
            >
              {count}
            </span>
          )}
        </>
      )}
    </NavLink>
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
function ItemCard({ item, claimants = [], onClaim, onEdit, onDelete }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5 shadow-sm">
      <div className="flex items-start gap-4">
        {item.image && (
          <div className="shrink-0">
            <img
              src={item.image}
              alt={item.itemName}
              className="h-16 w-16 rounded-xl border border-border object-cover"
            />
            {Array.isArray(item.images) && item.images.length > 1 && (
              <div className="mt-2 flex items-center gap-1">
                {item.images.slice(1).map((src) => (
                  <img
                    key={src}
                    src={src}
                    alt=""
                    className="h-6 w-6 rounded-md border border-border object-cover"
                    loading="lazy"
                  />
                ))}
              </div>
            )}
          </div>
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
            {Array.isArray(claimants) && claimants.length > 0 && (
              <div className="pt-1">
                <p className="text-xs font-semibold text-foreground">Claimed by</p>
                <div className="mt-1 space-y-0.5">
                  {claimants.map((c, idx) => (
                    <p
                      key={`${c.name}-${idx}`}
                      className="text-sm font-medium text-foreground/90"
                    >
                      {c.name || "Unknown passenger"}
                    </p>
                  ))}
                </div>
              </div>
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

          <div className="flex shrink-0 items-center gap-2">
            {onClaim && item.status === "unclaimed" && (
              <button
                type="button"
                onClick={() => onClaim(item.id)}
                className="rounded-xl border border-border bg-foreground px-3 py-1.5 text-xs font-medium text-background transition hover:opacity-90"
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
            {onEdit && (
              <button
                type="button"
                onClick={() => onEdit(item)}
                title="Edit item"
                className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border text-muted-foreground transition hover:bg-muted hover:text-foreground"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
            )}
            {onDelete && (
              <button
                type="button"
                onClick={() => onDelete(item)}
                title="Delete item"
                className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border text-muted-foreground transition hover:bg-red-50 hover:text-red-600"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ─── Edit Found Item Modal ────────────────────────────────────
function EditFoundItemModal({ item, routes, onClose, onSave }) {
  const fieldCls =
    "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

  const [form, setForm] = useState({
    itemName: item.itemName || "",
    itemDescription: item.description || "",
    itemType: item.itemType || "",
    brand: item.brand || "",
    model: item.model || "",
    color: item.color || "",
    material: item.material || "",
    itemCondition: item.itemCondition || "",
    category: item.category || "",
    locationFound: item.locationFound || "",
    routeOrStation: item.routeOrStation || "",
    routeId: item.routeId || "",
    dateFound: item.dateFound || "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function set(key) {
    return (e) => setForm((prev) => ({ ...prev, [key]: e.target.value }));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!form.itemName.trim()) {
      setError("Item name is required.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      await onSave(item.id, form);
      onClose();
    } catch (err) {
      setError(err?.message || "Failed to save changes.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center">
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={onClose}
      />
      <div className="relative z-10 w-full max-h-[90vh] overflow-y-auto rounded-t-2xl sm:max-w-lg sm:rounded-2xl border border-border bg-background shadow-xl">
        <div className="sticky top-0 flex items-center justify-between border-b border-border bg-background px-5 py-4">
          <h2 className="text-base font-semibold">Edit Found Item</h2>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 p-5">
          {error && (
            <div className="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium leading-none">
              Item Name <span className="text-destructive">*</span>
            </label>
            <input
              type="text"
              required
              value={form.itemName}
              onChange={set("itemName")}
              className={fieldCls}
              placeholder="e.g. Black backpack"
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Category</label>
              <select value={form.category} onChange={set("category")} className={fieldCls}>
                <option value="">— Select —</option>
                {CATEGORIES.map((c) => (
                  <option key={c} value={c}>{c}</option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Color</label>
              <input
                type="text"
                value={form.color}
                onChange={set("color")}
                className={fieldCls}
                placeholder="e.g. Black"
              />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Brand</label>
              <input type="text" value={form.brand} onChange={set("brand")} className={fieldCls} placeholder="e.g. Nike" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Model</label>
              <input type="text" value={form.model} onChange={set("model")} className={fieldCls} placeholder="e.g. Air Max" />
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Material</label>
              <input type="text" value={form.material} onChange={set("material")} className={fieldCls} placeholder="e.g. Leather" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Condition</label>
              <input type="text" value={form.itemCondition} onChange={set("itemCondition")} className={fieldCls} placeholder="e.g. Good" />
            </div>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium leading-none">Description</label>
            <textarea
              rows={3}
              value={form.itemDescription}
              onChange={set("itemDescription")}
              className="flex w-full rounded-xl border border-input bg-background px-3.5 py-2.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="Describe the item…"
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Location Found</label>
              <input type="text" value={form.locationFound} onChange={set("locationFound")} className={fieldCls} placeholder="e.g. Bus seat 14A" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Date Found</label>
              <input type="date" value={form.dateFound} onChange={set("dateFound")} className={fieldCls} />
            </div>
          </div>

          {routes.length > 0 && (
            <div className="space-y-2">
              <label className="text-sm font-medium leading-none">Route / Station</label>
              <select value={form.routeId} onChange={(e) => {
                const sel = routes.find((r) => r.id === e.target.value);
                setForm((prev) => ({
                  ...prev,
                  routeId: e.target.value,
                  routeOrStation: sel?.route_name || prev.routeOrStation,
                }));
              }} className={fieldCls}>
                <option value="">— Unchanged —</option>
                {routes.map((r) => (
                  <option key={r.id} value={r.id}>{r.route_name}</option>
                ))}
              </select>
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 rounded-xl border border-border px-4 py-2.5 text-sm font-medium transition hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="flex-1 rounded-xl bg-foreground px-4 py-2.5 text-sm font-medium text-background transition hover:opacity-90 disabled:opacity-50"
            >
              {loading ? "Saving…" : "Save Changes"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ─── Delete Confirm Modal ─────────────────────────────────────
function DeleteConfirmModal({ item, onClose, onConfirm }) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleConfirm() {
    setLoading(true);
    setError("");
    try {
      await onConfirm(item.id);
      onClose();
    } catch (err) {
      setError(err?.message || "Failed to delete item.");
      setLoading(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center px-4">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 w-full max-w-sm rounded-2xl border border-border bg-background p-6 shadow-xl">
        <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-red-100">
          <Trash2 className="h-5 w-5 text-red-600" />
        </div>
        <h2 className="mb-1 text-base font-semibold">Delete Found Item</h2>
        <p className="mb-1 text-sm text-muted-foreground">
          Are you sure you want to delete{" "}
          <span className="font-medium text-foreground">{item.itemName}</span>?
          This also removes its embedding and cannot be undone.
        </p>
        {error && (
          <p className="mt-3 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}
        <div className="mt-5 flex gap-3">
          <button
            type="button"
            onClick={onClose}
            className="flex-1 rounded-xl border border-border px-4 py-2.5 text-sm font-medium transition hover:bg-muted"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={loading}
            className="flex-1 rounded-xl bg-red-600 px-4 py-2.5 text-sm font-medium text-white transition hover:opacity-90 disabled:opacity-50"
          >
            {loading ? "Deleting…" : "Delete"}
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Main Dashboard ──────────────────────────────────────────
export default function StaffDashboard() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const { section } = useParams();
  const normalizedSection = section === "claimed" ? "claims" : section;
  const tab =
    normalizedSection && STAFF_SECTIONS.has(normalizedSection) ? normalizedSection : null;
  const [items, setItems] = useState([]);
  const [claims, setClaims] = useState([]);
  const [itemsLoading, setItemsLoading] = useState(false);
  const [itemsError, setItemsError] = useState("");

  // Upload form state
  const [itemName, setItemName] = useState("");
  const [locationFound, setLocationFound] = useState("");
  const [selectedRouteId, setSelectedRouteId] = useState("");
  const [transitRoutes, setTransitRoutes] = useState([]);
  const [dateFound, setDateFound] = useState("");
  // photos: array of { id, url (data-URI for preview), data (base64 for API) }
  const [photos, setPhotos] = useState([]);
  const [uploadSuccess, setUploadSuccess] = useState(false);

  // AI-extracted detail fields
  const [extractedDetails, setExtractedDetails] = useState(null);
  const [extractingCount, setExtractingCount] = useState(0);
  const extracting = extractingCount > 0;
  const [extractError, setExtractError] = useState(null);
  const [extractionFindings, setExtractionFindings] = useState([]);
  // Editable fields pre-filled by AI
  const [editableDescription, setEditableDescription] = useState("");
  const [editableCategory, setEditableCategory] = useState("");
  const [uploadError, setUploadError] = useState("");
  const lastAutoDescriptionRef = useRef("");
  const lastAutoCategoryRef = useRef("");

  const [cameraOpen, setCameraOpen] = useState(false);
  const [cameraError, setCameraError] = useState(null);
  const videoRef = useRef(null);
  const streamRef = useRef(null);

  const uploadPhotoToS3 = useCallback(async (entry) => {
    if (!entry?.id) return;
    if (entry.s3Key) return;

    setPhotos((prev) =>
      prev.map((p) =>
        p.id === entry.id ? { ...p, uploading: true, uploadError: null } : p,
      ),
    );

    try {
      const blob =
        entry.file ||
        entry.blob ||
        (await (await fetch(entry.data || entry.url)).blob());
      const contentType = blob.type || "image/jpeg";

      const init = await mediaInitUploads([
        { content_type: contentType, size_bytes: blob.size },
      ]);
      const upload = init?.uploads?.[0];
      if (!upload?.upload_url || !upload?.s3_key) {
        throw new Error("failed to init upload");
      }

      const res = await fetch(upload.upload_url, {
        method: "PUT",
        headers: upload.headers || { "Content-Type": contentType },
        body: blob,
      });
      if (!res.ok) throw new Error(`upload failed (HTTP ${res.status})`);

      setPhotos((prev) =>
        prev.map((p) =>
          p.id === entry.id
            ? { ...p, uploading: false, s3Key: upload.s3_key }
            : p,
        ),
      );
    } catch (err) {
      setPhotos((prev) =>
        prev.map((p) =>
          p.id === entry.id
            ? {
              ...p,
              uploading: false,
              uploadError: err?.message || "upload failed",
            }
            : p,
        ),
      );
    }
  }, [mediaInitUploads]);

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
      const claimsPayload = await staffListClaims({ limit: 500, offset: 0 });
      setClaims(Array.isArray(claimsPayload?.claims) ? claimsPayload.claims : []);
    } catch (err) {
      setItemsError(err?.message || "Failed to load found items.");
    } finally {
      setItemsLoading(false);
    }
  }, [user?.id]);

  useEffect(() => {
    refreshItems();
  }, [refreshItems]);

  useEffect(() => {
    if (!user?.id) return undefined;
    let cancelled = false;
    (async () => {
      try {
        const payload = await staffListTransitRoutes({ limit: 500, offset: 0 });
        if (!cancelled) {
          setTransitRoutes(Array.isArray(payload?.routes) ? payload.routes : []);
        }
      } catch {
        if (!cancelled) setTransitRoutes([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [user?.id]);

  const claimed = items.filter((i) => i.status === "claimed");
  const claimedByUserItemIDs = new Set(
    claims
      .filter((c) => {
        const st = String(c?.status || "").toLowerCase();
        return st === "pending" || st === "approved";
      })
      .map((c) => String(c?.item_id || "").trim())
      .filter(Boolean),
  );
  const claimedByUsers = items.filter((i) => claimedByUserItemIDs.has(i.id));
  const inProgress = items.filter(
    (i) => i.status === "unclaimed" && !claimedByUserItemIDs.has(i.id),
  );
  const claimantsByItemID = claims.reduce((acc, c) => {
    const st = String(c?.status || "").toLowerCase();
    if (st !== "pending" && st !== "approved") return acc;
    const itemID = String(c?.item_id || "").trim();
    if (!itemID) return acc;
    const name = String(c?.claimant_name || "").trim();
    const next = { name };
    if (!acc[itemID]) {
      acc[itemID] = [];
    }
    if (!acc[itemID].some((x) => x.name === next.name)) {
      acc[itemID].push(next);
    }
    return acc;
  }, {});

  const runExtractionOnPhoto = useCallback(
    async (photo) => {
      setExtractingCount((n) => n + 1);
      setExtractError(null);
      try {
        const res = await fetch("/api/extract", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ image_base64: photo.data }),
        });
        if (!res.ok) {
          const payload = await res.json().catch(() => null);
          const msg =
            payload?.error ||
            payload?.detail ||
            `Failed to analyze image (HTTP ${res.status})`;
          throw new Error(msg);
        }
        const data = await res.json();
        setExtractionFindings((prev) => [
          ...prev,
          { photoId: photo.id, details: data },
        ]);
        setExtractError(null);
        if (!itemName && isKnownValue(data.item_name)) {
          setItemName(data.item_name);
        }
      } catch (err) {
        setExtractError(
          err?.message ||
            "Could not extract details from image. You can still fill in the details manually.",
        );
      } finally {
        setExtractingCount((n) => Math.max(0, n - 1));
      }
    },
    [itemName],
  );

  useEffect(() => {
    if (!extractionFindings.length) {
      setExtractedDetails(null);
      if (!editableDescription) lastAutoDescriptionRef.current = "";
      if (!editableCategory) lastAutoCategoryRef.current = "";
      return;
    }

    const combined = extractionFindings.reduce(
      (acc, cur) => mergeExtractedDetails(acc, cur.details),
      null,
    );
    setExtractedDetails(combined);

    const nextAutoDescription = combined?.item_description || "";
    if (
      nextAutoDescription !== editableDescription &&
      (!editableDescription || editableDescription === lastAutoDescriptionRef.current)
    ) {
      setEditableDescription(nextAutoDescription);
      lastAutoDescriptionRef.current = nextAutoDescription;
    }

    const nextAutoCategory = isKnownValue(combined?.category)
      ? combined.category
      : "";
    if (
      nextAutoCategory !== editableCategory &&
      (!editableCategory || editableCategory === lastAutoCategoryRef.current)
    ) {
      setEditableCategory(nextAutoCategory);
      lastAutoCategoryRef.current = nextAutoCategory;
    }
  }, [extractionFindings, editableCategory, editableDescription]);

  const appendPhotos = useCallback(
    async (newPhotoEntries) => {
      if (!newPhotoEntries.length) return;
      let toAdd = [];
      setPhotos((prev) => {
        const slots = MAX_PHOTOS - prev.length;
        toAdd = newPhotoEntries.slice(0, slots);
        if (!toAdd.length) return prev;
        return [...prev, ...toAdd];
      });
      if (!toAdd.length) return;
      for (const entry of toAdd) {
        // Run extraction + S3 upload immediately for every added photo.
        // eslint-disable-next-line no-await-in-loop
        await Promise.all([runExtractionOnPhoto(entry), uploadPhotoToS3(entry)]);
      }
    },
    [runExtractionOnPhoto, uploadPhotoToS3],
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
                file,
                s3Key: "",
                uploading: false,
                uploadError: null,
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
      const isLocalhost =
        window.location.hostname === "localhost" ||
        window.location.hostname === "127.0.0.1" ||
        window.location.hostname === "[::1]";
      if (!navigator.mediaDevices?.getUserMedia) {
        setCameraError("Camera is not supported in this browser.");
        return;
      }
      if (!window.isSecureContext && !isLocalhost) {
        setCameraError(
          `Camera requires HTTPS (or localhost). Current origin: ${window.location.origin}.`,
        );
        return;
      }
      try {
        const preferred = {
          video: { facingMode: { ideal: "environment" } },
          audio: false,
        };
        try {
          stream = await navigator.mediaDevices.getUserMedia(preferred);
        } catch (err) {
          if (err?.name === "OverconstrainedError") {
            stream = await navigator.mediaDevices.getUserMedia({
              video: true,
              audio: false,
            });
          } else {
            throw err;
          }
        }
        if (cancelled) {
          stream.getTracks().forEach((t) => t.stop());
          return;
        }
        streamRef.current = stream;
        const video = videoRef.current;
        if (video) {
          video.srcObject = stream;
          try {
            await video.play();
          } catch (err) {
            // Keep the stream alive even if playback is blocked by autoplay policy.
            // eslint-disable-next-line no-console
            console.warn("video.play() failed", err);
          }
        }
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error("getUserMedia failed", err);
        if (!cancelled) {
          const name = err?.name || "";
          if (name === "NotAllowedError" || name === "PermissionDeniedError") {
            setCameraError(
              "Camera permission denied. Allow camera access in your browser site settings, then try again.",
            );
          } else if (name === "NotFoundError" || name === "DevicesNotFoundError") {
            setCameraError("No camera was found on this device. Use file upload.");
          } else if (name === "NotReadableError") {
            setCameraError(
              "Camera is already in use by another app/tab. Close it and try again.",
            );
          } else {
            setCameraError(
              "Could not access the camera. Allow permission or use file upload.",
            );
          }
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
    const blob = await (await fetch(dataUrl)).blob();
    const entry = {
      id: crypto.randomUUID(),
      url: dataUrl,
      data: dataUrl,
      blob,
      s3Key: "",
      uploading: false,
      uploadError: null,
    };
    await appendPhotos([entry]);
    closeCamera();
  }

  function removePhoto(id) {
    const removed = photos.find((p) => p.id === id);
    setPhotos((prev) => prev.filter((p) => p.id !== id));
    setExtractionFindings((prev) => prev.filter((f) => f.photoId !== id));

    const key = removed?.s3Key;
    if (key && !removed?.uploading) {
      // Best-effort cleanup in S3 so abandoned uploads don't linger.
      mediaDeleteUpload(key).catch(() => null);
    }
  }

  const handleUpload = useCallback(
    async (e) => {
      e.preventDefault();
      if (photos.length === 0) return;
      if (!user?.id) return;
      setUploadError("");

      const pending = photos.some((p) => p.uploading);
      if (pending) {
        setUploadError("Please wait for photo uploads to finish.");
        return;
      }
      const keys = photos.map((p) => p.s3Key).filter(Boolean);
      if (keys.length === 0) {
        setUploadError("No uploaded image keys found. Please re-upload photos.");
        return;
      }
      if (transitRoutes.length === 0) {
        setUploadError("Add at least one transit route under the Routes tab before uploading.");
        return;
      }
      if (!String(selectedRouteId || "").trim()) {
        setUploadError("Select a transit route.");
        return;
      }

      const dateISO = dateFound
        ? new Date(`${dateFound}T00:00:00Z`).toISOString()
        : undefined;

      const routeMeta = transitRoutes.find((r) => r.id === selectedRouteId);
      const routeOrStationVal = String(routeMeta?.route_name || "").trim();
      const routeIdVal = String(selectedRouteId || "").trim();

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
          route_or_station: routeOrStationVal,
          route_id: routeIdVal,
          date_found: dateISO,
          image_keys: keys,
          primary_image_key: keys[0] || "",
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
          routeOrStation: routeOrStationVal,
          routeId: routeIdVal,
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
      setSelectedRouteId("");
      setDateFound("");
      setPhotos([]);
      setExtractedDetails(null);
      setExtractError(null);
      setExtractionFindings([]);
      setEditableDescription("");
      setEditableCategory("");
      lastAutoDescriptionRef.current = "";
      lastAutoCategoryRef.current = "";
      setUploadSuccess(true);
      setTimeout(() => setUploadSuccess(false), 3000);
    },
    [
      itemName,
      locationFound,
      selectedRouteId,
      transitRoutes,
      dateFound,
      photos,
      user,
      extractedDetails,
      editableDescription,
      editableCategory,
    ],
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

  // ── Edit / Delete state ───────────────────────────────────
  const [editItem, setEditItem] = useState(null);
  const [deleteItem, setDeleteItem] = useState(null);

  const handleSaveEdit = useCallback(
    async (foundItemId, form) => {
      if (!user?.id) throw new Error("not authenticated");
      const body = {
        staff_id: user.id,
        item_name: form.itemName || undefined,
        item_description: form.itemDescription || undefined,
        item_type: form.itemType || undefined,
        brand: form.brand || undefined,
        model: form.model || undefined,
        color: form.color || undefined,
        material: form.material || undefined,
        item_condition: form.itemCondition || undefined,
        category: form.category || undefined,
        location_found: form.locationFound || undefined,
        route_or_station: form.routeOrStation || undefined,
        route_id: form.routeId || undefined,
        date_found: form.dateFound
          ? new Date(form.dateFound).toISOString()
          : undefined,
      };
      // strip undefined keys so the backend only patches what changed
      const cleaned = Object.fromEntries(
        Object.entries(body).filter(([, v]) => v !== undefined),
      );
      const updated = await staffUpdateFoundItem(foundItemId, cleaned);
      const mapped = mapFoundItemDTO(updated);
      setItems((prev) =>
        prev.map((item) => (item.id === foundItemId ? { ...item, ...(mapped || {}) } : item)),
      );
    },
    [user?.id],
  );

  const handleConfirmDelete = useCallback(
    async (foundItemId) => {
      if (!user?.id) throw new Error("not authenticated");
      await staffDeleteFoundItem(foundItemId, user.id);
      setItems((prev) => prev.filter((item) => item.id !== foundItemId));
    },
    [user?.id],
  );

  if (!tab) {
    return <Navigate to="/staff/dashboard" replace />;
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-muted/40 to-background">
      {editItem && (
        <EditFoundItemModal
          item={editItem}
          routes={transitRoutes}
          onClose={() => setEditItem(null)}
          onSave={handleSaveEdit}
        />
      )}
      {deleteItem && (
        <DeleteConfirmModal
          item={deleteItem}
          onClose={() => setDeleteItem(null)}
          onConfirm={handleConfirmDelete}
        />
      )}

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
          <StaffTabLink to="/staff/dashboard" icon={Package} label="Dashboard" count={0} />
          <StaffTabLink to="/staff/upload" icon={Upload} label="Upload Item" count={0} />
          <StaffTabLink
            to="/staff/in-progress"
            icon={Clock}
            label="In Progress"
            count={inProgress.length}
          />
          <StaffTabLink
            to="/staff/claims"
            icon={CheckCircle2}
            label="Claims"
            count={claimedByUsers.length}
          />
          <StaffTabLink to="/staff/analytics" icon={BarChart3} label="Analytics" count={0} />
          <StaffTabLink to="/staff/routes" icon={MapPin} label="Routes" count={0} />
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
                value={inProgress.length}
                accent="bg-amber-100 text-amber-700"
              />
              <StatCard
                icon={CheckCircle2}
                label="Claims"
                value={claimedByUsers.length}
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
                    onClick={() => navigate("/staff/upload")}
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
                      onEdit={setEditItem}
                      onDelete={setDeleteItem}
                    />
                  ))}
                  {items.length > 5 && (
                    <p className="text-center text-xs text-muted-foreground">
                      Showing 5 of {items.length} items.{" "}
                      <button
                        type="button"
                        onClick={() => navigate("/staff/in-progress")}
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
                    Route / Station <span className="text-destructive">*</span>
                  </label>
                  <select
                    id="item-route"
                    required
                    value={selectedRouteId}
                    onChange={(e) => setSelectedRouteId(e.target.value)}
                    className={field}
                  >
                    <option value="" disabled>
                      Select a transit route
                    </option>
                    {transitRoutes.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.route_name}
                      </option>
                    ))}
                  </select>
                  {transitRoutes.length === 0 ? (
                    <p className="text-xs text-amber-800 dark:text-amber-200/90">
                      At least one route is required to upload. Add routes under the{" "}
                      <button
                        type="button"
                        onClick={() => navigate("/staff/routes")}
                        className="font-medium text-foreground underline underline-offset-2"
                      >
                        Routes
                      </button>{" "}
                      tab.
                    </p>
                  ) : null}
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
                    {photos.length}/{MAX_PHOTOS} · each photo analyzed & uploaded
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
                        {photo.uploading && (
                          <div className="absolute inset-0 flex items-center justify-center rounded-xl bg-black/50">
                            <div className="h-5 w-5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                          </div>
                        )}
                        {!photo.uploading && photo.uploadError && (
                          <div className="absolute inset-0 flex items-center justify-center rounded-xl bg-black/60 p-1 text-center text-[10px] font-medium text-white">
                            Upload failed
                          </div>
                        )}
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
                          autoPlay
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
                    AI is analysing {extractingCount} photo{extractingCount === 1 ? "" : "s"}…
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
                      AI Extracted Details ({extractionFindings.length} photo{extractionFindings.length === 1 ? "" : "s"})
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

                  {extractionFindings.length > 1 && (
                    <div className="space-y-2 rounded-xl border border-border bg-background/60 p-3">
                      <p className="text-[11px] font-medium text-muted-foreground">
                        Findings per photo
                      </p>
                      <div className="space-y-2">
                        {extractionFindings.map((f, idx) => (
                          <div key={f.photoId} className="text-xs text-muted-foreground">
                            <span className="font-medium text-foreground">
                              Photo {idx + 1}:
                            </span>{" "}
                            {f?.details?.item_description || "—"}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {photos.length === 0 && (
                <p className="text-sm text-destructive">
                  At least one photo is required.
                </p>
              )}

              <button
                type="submit"
                disabled={
                  photos.length === 0 ||
                  extracting ||
                  photos.some((p) => p.uploading || !p.s3Key || p.uploadError) ||
                  transitRoutes.length === 0 ||
                  !String(selectedRouteId || "").trim()
                }
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

            {inProgress.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center">
                <Clock className="mx-auto mb-3 h-8 w-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">
                  No unclaimed items at the moment.
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {inProgress.map((item) => (
                  <ItemCard
                    key={item.id}
                    item={item}
                    onEdit={setEditItem}
                    onDelete={setDeleteItem}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Claims */}
        {tab === "claims" && (
          <div className="space-y-6">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">
                Claims
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">
                Items that users have successfully claimed.
              </p>
            </div>

            {claimedByUsers.length === 0 ? (
              <div className="rounded-2xl border border-dashed border-border bg-card p-8 text-center">
                <CheckCircle2 className="mx-auto mb-3 h-8 w-8 text-muted-foreground/50" />
                <p className="text-sm text-muted-foreground">
                  No claims yet.
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {claimedByUsers.map((item) => (
                  <ItemCard
                    key={item.id}
                    item={item}
                    claimants={claimantsByItemID[item.id] || []}
                    onClaim={item.status === "unclaimed" ? handleClaim : undefined}
                    onEdit={setEditItem}
                    onDelete={setDeleteItem}
                  />
                ))}
              </div>
            )}
          </div>
        )}

        {/* Analytics */}
        {tab === "analytics" && <AnalyticsPanel />}

        {/* Transit routes (lines / stations catalog) */}
        {tab === "routes" && <StaffTransitRoutesPage />}
      </main>
    </div>
  );
}
