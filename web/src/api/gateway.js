// Prefer same-origin (Vite proxy / ingress) to avoid CORS + cluster-DNS issues
// like `api-gateway:8081` not resolving in the browser.
//
// If you want to hit the gateway directly from the browser, set:
//   VITE_API_BASE_URL=http://localhost:8081
//
// `VITE_API_GATEWAY_URL` remains supported as a fallback for older setups.
const GATEWAY_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ||
  import.meta.env.VITE_API_GATEWAY_URL ||
  "";
const GATEWAY_PATH_PREFIX = GATEWAY_BASE_URL ? "" : "/gateway";
const STORAGE_KEY = "smartfind-auth";

function getSessionToken() {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return "";
    const parsed = JSON.parse(raw);
    return String(parsed?.sessionToken || "").trim();
  } catch {
    return "";
  }
}

async function requestJSON(path, options) {
  const sessionToken = getSessionToken();
  const hasAuthHeader = Boolean(options?.headers?.Authorization);
  const isPassengerPath = String(path || "").startsWith("/passenger/");
  const autoAuthHeader =
    sessionToken && !hasAuthHeader && !isPassengerPath
      ? { Authorization: `Bearer ${sessionToken}` }
      : {};
  const url = `${GATEWAY_BASE_URL}${GATEWAY_PATH_PREFIX}${path}`;
  const requestOptions = {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...autoAuthHeader,
      ...(options?.headers || {}),
    },
    ...options,
  };
  let res = await fetch(url, requestOptions);
  let data = await res.json().catch(() => null);

  // If a stale token from session storage is sent, the gateway may reject it
  // before considering the valid httpOnly session cookie. Retry once cookie-only.
  if (
    !res.ok &&
    res.status === 401 &&
    Object.keys(autoAuthHeader).length > 0 &&
    String(data?.error || "")
      .toLowerCase()
      .includes("invalid forwarded token")
  ) {
    const retryHeaders = {
      ...(options?.headers || {}),
      "Content-Type": "application/json",
    };
    delete retryHeaders.Authorization;
    res = await fetch(url, { ...requestOptions, headers: retryHeaders });
    data = await res.json().catch(() => null);
  }
  if (!res.ok) {
    const msg = data?.error || `Request failed (${res.status})`;
    throw new Error(msg);
  }
  return data;
}

export async function staffLogin(email, password) {
  return requestJSON("/staff/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function staffSignup({ transitCode, fullName, email, password }) {
  return requestJSON("/staff", {
    method: "POST",
    body: JSON.stringify({
      transit_code: transitCode,
      full_name: fullName,
      email,
      password,
    }),
  });
}

export async function staffLogout() {
  return requestJSON("/staff/logout", { method: "POST", body: "{}" });
}

export async function passengerLogout() {
  return requestJSON("/passenger/logout", { method: "POST", body: "{}" });
}

export async function passengerListLostReports(params) {
  const q = new URLSearchParams();
  if (params?.status) q.set("status", params.status);
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/passenger/lost-reports${suffix}`, { method: "GET" });
}

export async function passengerListClaims(params) {
  const q = new URLSearchParams();
  if (params?.status) q.set("status", params.status);
  if (params?.limit != null) q.set("limit", String(params.limit));
  if (params?.offset != null) q.set("offset", String(params.offset));
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/passenger/claims${suffix}`, { method: "GET" });
}

export async function passengerFileClaim({ foundItemId, lostReportId, message }) {
  return requestJSON(`/passenger/claims`, {
    method: "POST",
    body: JSON.stringify({
      found_item_id: foundItemId,
      lost_report_id: lostReportId,
      message: message || "",
    }),
  });
}

export async function passengerListNotifications({ limit, unreadOnly, createdBefore } = {}) {
  const q = new URLSearchParams();
  if (limit != null) q.set("limit", String(limit));
  if (unreadOnly) q.set("unread_only", "1");
  if (createdBefore) q.set("created_before", createdBefore);
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/passenger/notifications${suffix}`, { method: "GET" });
}

export async function passengerMarkNotificationsRead(notificationIds) {
  const ids = Array.isArray(notificationIds)
    ? notificationIds.filter(Boolean)
    : [];
  if (ids.length === 0) return { status: "ok" };
  return requestJSON(`/passenger/notifications/read`, {
    method: "POST",
    body: JSON.stringify({ notification_ids: ids }),
  });
}

export async function staffListFoundItems(params) {
  const q = new URLSearchParams();
  if (params?.status) q.set("status", params.status);
  if (params?.routeId) q.set("route_id", params.routeId);
  if (params?.postedByStaffId) q.set("posted_by_staff_id", params.postedByStaffId);
  if (params?.limit != null) q.set("limit", String(params.limit));
  if (params?.offset != null) q.set("offset", String(params.offset));
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/staff/found-items${suffix}`, { method: "GET" });
}

export async function staffCreateFoundItem(body) {
  return requestJSON("/staff/found-items", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function staffUpdateFoundItemStatus(body) {
  return requestJSON("/staff/found-items/status", {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function staffUpdateFoundItem(foundItemId, body) {
  return requestJSON(`/staff/found-items/${encodeURIComponent(foundItemId)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function staffDeleteFoundItem(foundItemId, staffId) {
  return requestJSON(`/staff/found-items/${encodeURIComponent(foundItemId)}`, {
    method: "DELETE",
    body: JSON.stringify({ staff_id: staffId }),
  });
}

export async function staffListClaims(params) {
  const q = new URLSearchParams();
  if (params?.status) q.set("status", params.status);
  if (params?.itemId) q.set("item_id", params.itemId);
  if (params?.passengerId) q.set("passenger_id", params.passengerId);
  if (params?.limit != null) q.set("limit", String(params.limit));
  if (params?.offset != null) q.set("offset", String(params.offset));
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/staff/claims${suffix}`, { method: "GET" });
}

/** Transit lines / routes (DB `routes` table). */
export async function staffListTransitRoutes(params) {
  const q = new URLSearchParams();
  if (params?.createdByStaffId) {
    q.set("created_by_staff_id", params.createdByStaffId);
  }
  if (params?.limit != null) q.set("limit", String(params.limit));
  if (params?.offset != null) q.set("offset", String(params.offset));
  const suffix = q.toString() ? `?${q.toString()}` : "";
  return requestJSON(`/staff/routes${suffix}`, { method: "GET" });
}

export async function staffCreateTransitRoute({ staffId, routeName }) {
  return requestJSON("/staff/routes", {
    method: "POST",
    body: JSON.stringify({ staff_id: staffId, route_name: routeName }),
  });
}

export async function staffDeleteTransitRoute({ staffId, routeId }) {
  const q = new URLSearchParams({
    staff_id: String(staffId || "").trim(),
    route_id: String(routeId || "").trim(),
  });
  return requestJSON(`/staff/routes?${q.toString()}`, { method: "DELETE" });
}

export async function mediaInitUploads(files) {
  return requestJSON("/media/uploads/init", {
    method: "POST",
    body: JSON.stringify({ files }),
  });
}

export async function mediaDeleteUpload(s3Key) {
  return requestJSON("/media/uploads/delete", {
    method: "POST",
    body: JSON.stringify({ s3_key: s3Key }),
  });
}

export async function extractImageDetails(imageBase64) {
  return requestJSON("/extract", {
    method: "POST",
    body: JSON.stringify({ image_base64: imageBase64 }),
  });
}
