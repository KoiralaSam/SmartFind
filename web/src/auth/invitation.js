/**
 * Valid invitation codes for transit staff signup (replace with API check later).
 */
export const INVITATION_CODES = [
  "SMARTFIND-TRANSIT-2026",
  "DEMO-INVITE",
];

export function isValidInvitationCode(code) {
  const trimmed = (code || "").trim();
  return INVITATION_CODES.some((c) => c === trimmed);
}
