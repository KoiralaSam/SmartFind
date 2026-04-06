import { Link } from "react-router-dom";
import { LogOut, Train } from "lucide-react";
import { useAuth } from "../context/useAuth";

export default function StaffDashboard() {
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Train className="h-5 w-5" aria-hidden />
            <span className="font-semibold">Staff console</span>
          </div>
          <div className="flex items-center gap-4 text-sm">
            <span className="text-muted-foreground hidden sm:inline">
              {user?.name}
            </span>
            <button
              type="button"
              onClick={() => logout()}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm hover:bg-muted"
            >
              <LogOut className="h-4 w-4" />
              Sign out
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8 space-y-6">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Welcome, {user?.name}
          </h1>
          <p className="text-muted-foreground mt-1">
            Transit staff tools will appear here (routes, vehicles, incidents).
          </p>
        </div>
        <div className="rounded-lg border border-border bg-card p-6 text-card-foreground shadow-sm">
          <p className="text-sm">
            You are signed in as <strong>transit staff</strong>. Passenger
            features are available on a separate sign-in.
          </p>
          <p className="text-sm text-muted-foreground mt-3">
            Need the passenger app?{" "}
            <Link
              to="/login"
              className="text-primary underline underline-offset-4"
              onClick={() => logout()}
            >
              Sign out and use passenger login
            </Link>
          </p>
        </div>
      </main>
    </div>
  );
}
