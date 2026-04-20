import { NavLink, Outlet } from "react-router-dom";
import { ClipboardList, LogOut, MessageSquareText, ShieldCheck } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { NotificationsPanel } from "../components/NotificationsPanel";
import { useAuth } from "../context/useAuth";

function NavItem({ to, icon: Icon, label, compact = false }) {
  return (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `group flex items-center text-sm font-medium transition ${
          isActive
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted/60 hover:text-foreground"
        } ${
          compact
            ? "flex-1 flex-col justify-center gap-1 rounded-lg px-2 py-2 text-[11px]"
            : "gap-2.5 rounded-lg px-2.5 py-2"
        }`
      }
    >
      <Icon className="h-4 w-4 shrink-0" aria-hidden />
      <span>{label}</span>
    </NavLink>
  );
}
export default function PassengerLayout() {
  const { user, logout } = useAuth();

  return (
    <div className="h-screen overflow-hidden bg-background">
      <div className="flex h-full w-full p-2 sm:p-3">
        <aside className="hidden w-64 shrink-0 border-r border-border/70 pr-4 md:flex md:flex-col">
          <div className="flex min-w-0 items-center gap-3 py-2">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-foreground text-background">
              <span className="text-sm font-bold tracking-tight">SF</span>
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold tracking-tight">SmartFind</p>
              <p className="truncate text-xs text-muted-foreground">Lost & found</p>
            </div>
          </div>
          <nav className="mt-4 space-y-1.5">
            <NavItem to="/passenger/chat" icon={MessageSquareText} label="Chat" />
            <NavItem to="/passenger/reports" icon={ClipboardList} label="Lost reports" />
            <NavItem to="/passenger/claims" icon={ShieldCheck} label="Claims" />
          </nav>
          <div className="mt-auto">
            <div className="flex items-center justify-between gap-2 pb-2">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Notifications
              </p>
              <NotificationsPanel />
            </div>
            <button
              type="button"
              onClick={() => logout()}
              className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
              title="Logout"
            >
              <LogOut className="h-4 w-4 shrink-0" aria-hidden />
              Logout
            </button>
            <div className="mt-3 flex min-w-0 items-center gap-2 border-t border-border/70 pt-3">
              <AccountAvatar user={user} sizeClass="h-8 w-8" />
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{user?.name || "Passenger"}</p>
                <p className="truncate text-xs text-muted-foreground">{user?.email || ""}</p>
              </div>
            </div>
          </div>
        </aside>

        <div className="min-w-0 flex-1 px-2 py-2 sm:px-3 sm:py-3 md:pl-5">
          <header className="border-b border-border/70 px-1 py-2.5 md:hidden">
            <div className="flex items-center justify-between gap-2">
              <div className="flex min-w-0 items-center gap-3">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-foreground text-background">
                  <span className="text-sm font-bold tracking-tight">SF</span>
                </div>
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold tracking-tight">SmartFind</p>
                  <p className="truncate text-xs text-muted-foreground">Lost & found</p>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <NotificationsPanel />
                <AccountAvatar user={user} sizeClass="h-8 w-8" />
                <button
                  type="button"
                  onClick={() => logout()}
                  className="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border/70 text-muted-foreground transition hover:bg-muted/60 hover:text-foreground"
                  title="Logout"
                >
                  <LogOut className="h-4 w-4 shrink-0" aria-hidden />
                </button>
              </div>
            </div>
            <nav className="mt-2 grid grid-cols-3 gap-1 border-t border-border/70 pt-2">
              <NavItem compact to="/passenger/chat" icon={MessageSquareText} label="Chat" />
              <NavItem compact to="/passenger/reports" icon={ClipboardList} label="Reports" />
              <NavItem compact to="/passenger/claims" icon={ShieldCheck} label="Claims" />
            </nav>
          </header>

          <main className="min-h-0 h-[calc(100%-6.75rem)] md:h-full overflow-hidden">
            <div className="h-full">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </div>
  );
}
