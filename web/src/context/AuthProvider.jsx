import { useCallback, useEffect, useMemo, useState } from "react";
import { passengerLogout, staffLogin, staffLogout, staffSignup } from "../api/gateway";
import { legoAvatarFromSeed } from "../lib/avatarUrl";
import { AuthContext } from "./auth-context";

const STORAGE_KEY = "smartfind-auth";
const GATEWAY_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ||
  import.meta.env.VITE_API_GATEWAY_URL ||
  "";
const GATEWAY_PATH_PREFIX = GATEWAY_BASE_URL ? "" : "/gateway";

function persistUser(next) {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next));
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);

  useEffect(() => {
    try {
      const raw = sessionStorage.getItem(STORAGE_KEY);
      if (raw) setUser(JSON.parse(raw));
    } catch {
      sessionStorage.removeItem(STORAGE_KEY);
    }
  }, []);

  const loginStaff = useCallback(async (email, password) => {
    try {
      const payload = await staffLogin(email.trim(), password);
      const staff = payload?.staff;
      if (!staff?.id || !staff?.email) {
        return { ok: false, error: "Staff profile was missing in response." };
      }
      const next = {
        id: staff.id,
        email: staff.email,
        name: staff.full_name || staff.email.split("@")[0] || "Staff",
        role: "staff",
        authProvider: "password",
        picture: legoAvatarFromSeed(staff.email),
        sessionToken: payload.session_token || undefined,
      };
      setUser(next);
      persistUser(next);
      return { ok: true };
    } catch (err) {
      return { ok: false, error: err?.message || "Staff login failed." };
    }
  }, []);

  const signupStaff = useCallback(
    async ({ email, password, invitationCode, name }) => {
      const normalized = email.trim().toLowerCase();
      const displayName =
        (name && name.trim()) || normalized.split("@")[0] || "Staff";
      try {
        await staffSignup({
          transitCode: invitationCode,
          fullName: displayName,
          email: normalized,
          password,
        });
      } catch (err) {
        return { ok: false, error: err?.message || "Signup failed." };
      }
      return loginStaff(normalized, password);
    },
    [loginStaff],
  );

  const loginPassengerGoogle = useCallback(async (credential) => {
    if (!credential || typeof credential !== "string") {
      return {
        ok: false,
        error: "Google credential is missing. Please try again.",
      };
    }
    let payload;
    try {
      const res = await fetch(
        `${GATEWAY_BASE_URL}${GATEWAY_PATH_PREFIX}/passenger/login`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ id_token: credential }),
        },
      );
      payload = await res.json();
      if (!res.ok) {
        return {
          ok: false,
          error: payload?.error || "Passenger login failed. Please try again.",
        };
      }
    } catch {
      return {
        ok: false,
        error: "Could not reach login service. Please try again.",
      };
    }

    const passenger = payload?.passenger;
    if (!passenger?.email) {
      return { ok: false, error: "Passenger profile was missing in response." };
    }
    const avatar =
      (passenger.avatar_url && String(passenger.avatar_url).trim()) || "";
    const next = {
      id: passenger.id,
      email: passenger.email,
      name: passenger.full_name || passenger.email.split("@")[0] || "Passenger",
      role: "passenger",
      authProvider: "google",
      picture: avatar || legoAvatarFromSeed(passenger.email),
      sessionToken: payload.session_token || undefined,
    };
    setUser(next);
    persistUser(next);
    return { ok: true };
  }, []);

  const logout = useCallback(async () => {
    try {
      if (user?.role === "staff") {
        await staffLogout();
      } else if (user?.role === "passenger") {
        await passengerLogout();
      }
    } finally {
      setUser(null);
      sessionStorage.removeItem(STORAGE_KEY);
    }
  }, [user]);

  const value = useMemo(
    () => ({
      user,
      loginStaff,
      signupStaff,
      loginPassengerGoogle,
      logout,
    }),
    [user, loginStaff, signupStaff, loginPassengerGoogle, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
