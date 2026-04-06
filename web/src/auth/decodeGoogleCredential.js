/**
 * Decode Google ID token payload (client-side preview only; verify on your API in production).
 */
export function decodeGoogleCredential(credential) {
  if (!credential || typeof credential !== "string") {
    return null;
  }
  const parts = credential.split(".");
  if (parts.length < 2) return null;
  try {
    const base64Url = parts[1];
    const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
    const json = decodeURIComponent(
      atob(base64)
        .split("")
        .map((c) => "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2))
        .join(""),
    );
    return JSON.parse(json);
  } catch {
    return null;
  }
}
