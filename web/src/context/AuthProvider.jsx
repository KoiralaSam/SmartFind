import { useCallback, useEffect, useMemo, useState } from "react";
import { isValidInvitationCode } from "../auth/invitation";
import { findStaffByEmail, registerStaffAccount } from "../auth/staffAccounts";
import { AuthContext } from "./auth-context";

const STORAGE_KEY = "smartfind-auth";

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
    const account = findStaffByEmail(email);
    if (!account || account.password !== password) {
      return {
        ok: false,
        error: "Invalid email or password.",
      };
    }
    const next = {
      email: account.email,
      name: account.name,
      role: "staff",
      authProvider: "password",
    };
    setUser(next);
    persistUser(next);
    return { ok: true };
  }, []);

  const signupStaff = useCallback(
    async ({ email, password, invitationCode, name }) => {
      if (!isValidInvitationCode(invitationCode)) {
        return { ok: false, error: "Invalid or expired transit code." };
      }
      const normalized = email.trim().toLowerCase();
      if (findStaffByEmail(normalized)) {
        return {
          ok: false,
          error: "An account with this email already exists.",
        };
      }
      const displayName =
        (name && name.trim()) || normalized.split("@")[0] || "Staff";
      registerStaffAccount({
        email: normalized,
        password,
        name: displayName,
      });
      const next = {
        email: normalized,
        name: displayName,
        role: "staff",
        authProvider: "password",
      };
      setUser(next);
      persistUser(next);
      return { ok: true };
    },
    [],
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
      const res = await fetch("http://localhost:8081/passenger/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ id_token: credential }),
      });
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
    const next = {
      id: passenger.id,
      email: passenger.email,
      name: passenger.full_name || passenger.email.split("@")[0] || "Passenger",
      role: "passenger",
      authProvider: "google",
      picture: passenger.avatar_url || undefined,
      sessionToken: payload.session_token || undefined,
    };
    setUser(next);
    persistUser(next);
    return { ok: true };
  }, []);

  const logout = useCallback(() => {
    setUser(null);
    sessionStorage.removeItem(STORAGE_KEY);
  }, []);

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
