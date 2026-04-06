import { Link } from "react-router-dom";
import { LogOut, MapPin } from "lucide-react";
import { useAuth } from "../context/useAuth";

export default function PassengerDashboard() {
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <MapPin className="h-5 w-5" aria-hidden />
            <span className="font-semibold">Passenger</span>
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
            Trip planning and live updates will appear here.
          </p>
        </div>
        <div className="rounded-lg border border-border bg-card p-6 text-card-foreground shadow-sm">
          <p className="text-sm">
            You are signed in as a <strong>passenger</strong>. Staff tools use a
            different account type.
          </p>
          <p className="text-sm text-muted-foreground mt-3">
            Transit staff?{" "}
            <Link
              to="/login"
              className="text-primary underline underline-offset-4"
              onClick={() => logout()}
            >
              Sign out and sign in as staff
            </Link>
          </p>
        </div>
      </main>
    </div>
  );
}
