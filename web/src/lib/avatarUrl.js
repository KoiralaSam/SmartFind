const LEGO_BASE = "https://randomuser.me/api/portraits/lego";

/** Deterministic Lego portrait index 0–9 from a string (e.g. email). */
export function legoAvatarFromSeed(seed) {
  const s = String(seed || "user");
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (h * 31 + s.charCodeAt(i)) | 0;
  }
  const idx = Math.abs(h) % 10;
  return `${LEGO_BASE}/${idx}.jpg`;
}

/** Prefer explicit picture URL (e.g. Google or backend), else Lego from email. */
export function accountPictureUrl(user) {
  const url = user?.picture && String(user.picture).trim();
  if (url) return url;
  if (user?.email) return legoAvatarFromSeed(user.email);
  return `${LEGO_BASE}/0.jpg`;
}
