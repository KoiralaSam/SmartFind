import { useCallback, useEffect, useMemo, useState } from "react";
import { ACCOUNTS } from "../auth/demoAccounts";
import { AuthContext } from "./auth-context";

const STORAGE_KEY = "smartfind-auth";

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

  const login = useCallback(async (email, password, role) => {
    const account = ACCOUNTS.find(
      (a) =>
        a.email === email &&
        a.password === password &&
        a.role === role,
    );
    if (!account) {
      return {
        ok: false,
        error: "Invalid email or password for this account type.",
      };
    }
    const next = {
      email: account.email,
      role: account.role,
      name: account.name,
    };
    setUser(next);
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    return { ok: true };
  }, []);

  const logout = useCallback(() => {
    setUser(null);
    sessionStorage.removeItem(STORAGE_KEY);
  }, []);

  const value = useMemo(
    () => ({ user, login, logout }),
    [user, login, logout],
  );

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}
