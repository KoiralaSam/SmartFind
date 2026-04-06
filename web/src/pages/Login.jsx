import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Train, User } from "lucide-react";
import { useAuth } from "../context/useAuth";

const ROLES = [
  {
    id: "staff",
    label: "Transit staff",
    description: "Operations, dispatch, and fleet tools",
    icon: Train,
  },
  {
    id: "passenger",
    label: "Passenger",
    description: "Plan trips and track your ride",
    icon: User,
  },
];

export default function Login() {
  const { user, login } = useAuth();
  const navigate = useNavigate();
  const [role, setRole] = useState("passenger");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  if (user) {
    return (
      <Navigate
        to={user.role === "staff" ? "/staff" : "/passenger"}
        replace
      />
    );
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    const result = await login(email.trim(), password, role);
    setSubmitting(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    navigate(role === "staff" ? "/staff" : "/passenger", { replace: true });
  }

  return (
    <div className="min-h-screen bg-background flex flex-col items-center justify-center p-6">
      <div className="w-full max-w-md space-y-8">
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-semibold tracking-tight">SmartFind</h1>
          <p className="text-sm text-muted-foreground">
            Sign in with the account type that matches your role.
          </p>
        </div>

        <div className="grid grid-cols-2 gap-2 rounded-lg border border-border p-1 bg-muted/40">
          {ROLES.map((r) => {
            const Icon = r.icon;
            const active = role === r.id;
            return (
              <button
                key={r.id}
                type="button"
                onClick={() => {
                  setRole(r.id);
                  setError("");
                }}
                className={`flex flex-col items-center gap-1 rounded-md px-3 py-3 text-sm transition-colors ${
                  active
                    ? "bg-background shadow-sm text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                <Icon className="h-5 w-5" aria-hidden />
                <span className="font-medium">{r.label}</span>
              </button>
            );
          })}
        </div>

        <p className="text-xs text-center text-muted-foreground px-1">
          {ROLES.find((r) => r.id === role)?.description}
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label
              htmlFor="email"
              className="text-sm font-medium leading-none"
            >
              Email
            </label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder={
                role === "staff"
                  ? "staff@transit.local"
                  : "passenger@example.com"
              }
            />
          </div>
          <div className="space-y-2">
            <label
              htmlFor="password"
              className="text-sm font-medium leading-none"
            >
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="••••••••"
            />
          </div>

          {error ? (
            <p
              className="text-sm text-destructive"
              role="alert"
            >
              {error}
            </p>
          ) : null}

          <button
            type="submit"
            disabled={submitting}
            className="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50"
          >
            {submitting ? "Signing in…" : "Sign in"}
          </button>
        </form>

        <p className="text-xs text-center text-muted-foreground border-t border-border pt-6">
          Demo: staff{" "}
          <code className="rounded bg-muted px-1 py-0.5">staff@transit.local</code>{" "}
          / passenger{" "}
          <code className="rounded bg-muted px-1 py-0.5">
            passenger@example.com
          </code>{" "}
          — password{" "}
          <code className="rounded bg-muted px-1 py-0.5">demo123</code>
        </p>
      </div>
    </div>
  );
}
