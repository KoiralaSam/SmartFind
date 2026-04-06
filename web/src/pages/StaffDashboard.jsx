import { Link } from "react-router-dom";
import { LogOut, Train } from "lucide-react";
import { useAuth } from "../context/useAuth";

export default function StaffDashboard() {
  const { user, logout } = useAuth();

  return (
    <div className="min-h-screen bg-gradient-to-b from-muted/40 to-background">
      <header className="border-b border-border/80 bg-background/90 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-2xl items-center justify-between px-4">
          <Link
            to="/"
            className="flex items-center gap-2 font-semibold tracking-tight hover:opacity-90"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-foreground text-background">
              <Train className="h-4 w-4" aria-hidden />
            </span>
            <span className="hidden sm:inline">Staff</span>
          </Link>
          <div className="flex items-center gap-3 text-sm">
            <span className="hidden max-w-[140px] truncate text-muted-foreground sm:inline sm:max-w-[200px]">
              {user?.name}
            </span>
            <button
              type="button"
              onClick={() => logout()}
              className="inline-flex items-center gap-1.5 rounded-full border border-border px-3 py-1.5 text-xs font-medium hover:bg-muted sm:text-sm"
            >
              <LogOut className="h-3.5 w-3.5" />
              Sign out
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-2xl px-4 py-10">
        <div className="mb-10 space-y-2">
          <p className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            Console
          </p>
          <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
            Hi, {user?.name}
          </h1>
          <p className="max-w-lg text-sm leading-relaxed text-muted-foreground">
            Dispatch and operations tools will plug in here. For now this is a
            placeholder screen.
          </p>
        </div>

        <div className="rounded-2xl border border-border bg-card p-8 shadow-sm">
          <p className="text-sm leading-relaxed">
            You’re signed in as <strong>transit staff</strong>. Passengers use
            Google from the home page to report lost items.
          </p>
          <p className="mt-4 text-sm text-muted-foreground">
            Need to report something as a rider?{" "}
            <Link
              to="/passenger/sign-in"
              className="font-medium text-foreground underline underline-offset-4"
              onClick={() => logout()}
            >
              Sign out and continue as a passenger
            </Link>
            .
          </p>
        </div>
      </main>
    </div>
  );
}
