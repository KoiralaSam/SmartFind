import { STAFF_SEED } from "./demoAccounts";

const STORAGE_KEY = "smartfind-staff-registrations";

function loadRegistrations() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

/** All staff accounts: seed + locally registered. */
export function getAllStaffAccounts() {
  return [...STAFF_SEED, ...loadRegistrations()];
}

export function findStaffByEmail(email) {
  const normalized = email.trim().toLowerCase();
  return getAllStaffAccounts().find(
    (a) => a.email.toLowerCase() === normalized,
  );
}

/**
 * @param {{ email: string; password: string; name: string }} account
 */
export function registerStaffAccount(account) {
  const registrations = loadRegistrations();
  const next = [
    ...registrations,
    {
      email: account.email.trim().toLowerCase(),
      password: account.password,
      name: account.name.trim(),
    },
  ];
  localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
}
