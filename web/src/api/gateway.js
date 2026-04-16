// Prefer same-origin (Vite proxy / ingress) to avoid CORS + cluster-DNS issues
// like `api-gateway:8081` not resolving in the browser.
//
// If you want to hit the gateway directly from the browser, set:
//   VITE_API_GATEWAY_URL=http://localhost:8081
const GATEWAY_BASE_URL = import.meta.env.VITE_API_GATEWAY_URL || "";

async function requestJSON(path, options) {
  const res = await fetch(`${GATEWAY_BASE_URL}${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(options?.headers || {}) },
    ...options,
  });
  const data = await res.json().catch(() => null);
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
