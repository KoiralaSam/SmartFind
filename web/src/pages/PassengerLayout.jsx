import { NavLink, Outlet } from "react-router-dom";
import { ClipboardList, LogOut, MessageSquareText, ShieldCheck } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";

function NavItem({ to, icon: Icon, label }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium transition ${
          isActive
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
        }`
      }
    >
      <Icon className="h-4 w-4" aria-hidden />
      <span>{label}</span>
    </NavLink>
  );
}

export default function PassengerLayout() {
  const { user, logout } = useAuth();

  return (
    <div className="flex h-screen overflow-hidden bg-gradient-to-b from-muted/40 via-background to-background">
      <aside className="hidden w-64 shrink-0 border-r border-border/80 bg-background/90 p-4 backdrop-blur-md sm:flex sm:flex-col">
        <div className="flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-violet-600 to-indigo-600 text-white shadow-sm">
            <ShieldCheck className="h-4 w-4" aria-hidden />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold">SmartFind</p>
            <p className="truncate text-xs text-muted-foreground">Passenger</p>
          </div>
        </div>

        <nav className="mt-6 space-y-1">
          <NavItem to="/passenger/chat" icon={MessageSquareText} label="Chat" />
          <NavItem to="/passenger/reports" icon={ClipboardList} label="Lost reports" />
          <NavItem to="/passenger/claims" icon={ShieldCheck} label="Claims" />
        </nav>

        <div className="mt-auto flex items-center justify-between gap-2 rounded-xl border border-border/80 bg-card px-3 py-2">
          <div className="flex min-w-0 items-center gap-2">
            <AccountAvatar user={user} sizeClass="h-8 w-8" />
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{user?.name || "Passenger"}</p>
              <p className="truncate text-xs text-muted-foreground">{user?.email || ""}</p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => logout()}
            className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2 py-1 text-xs font-medium transition hover:bg-muted"
            title="Logout"
          >
            <LogOut className="h-3.5 w-3.5" aria-hidden />
          </button>
        </div>
      </aside>

      <main className="min-w-0 flex-1">
        <Outlet />
      </main>
    </div>
  );
}

