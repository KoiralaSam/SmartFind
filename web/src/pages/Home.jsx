import { Link, useNavigate } from "react-router-dom";
import { Bus, Shield, UserRound } from "lucide-react";
import { AccountAvatar } from "../components/AccountAvatar";
import { useAuth } from "../context/useAuth";

/** Transit / bus (Unsplash — free to use). */
const TRANSIT_HERO_IMAGE =
  "https://images.unsplash.com/photo-1544620347-c4fd4a3d5957?auto=format&fit=crop&w=2400&q=80";

export default function Home() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  function switchRole(targetPath) {
    logout();
    navigate(targetPath, { replace: true });
  }

  return (
    <div className="relative min-h-screen">
      <img
        src={TRANSIT_HERO_IMAGE}
        alt=""
        className="pointer-events-none absolute inset-0 h-full w-full object-cover"
        loading="eager"
        decoding="async"
      />
      <div
        className="absolute inset-0 bg-gradient-to-b from-black/55 via-black/45 to-black/65 md:bg-gradient-to-r md:from-black/70 md:via-black/45 md:to-black/70"
        aria-hidden
      />

      <header className="relative z-20 border-b border-white/10 bg-black/20 backdrop-blur-md">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between gap-3 px-4">
          <div className="flex min-w-0 items-center gap-2.5 font-semibold tracking-tight text-white">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-white text-black">
              <Bus className="h-5 w-5" aria-hidden />
            </span>
            <span className="truncate">SmartFind</span>
          </div>

          {user ? (
            <div className="flex shrink-0 items-center gap-2 sm:gap-3 text-sm">
              <AccountAvatar
                user={user}
                className="border-white/25 bg-black/30"
              />
              <span className="hidden max-w-[120px] truncate text-white/80 sm:inline sm:max-w-[180px]">
                {user.name}
              </span>
              <button
                type="button"
                onClick={() => logout()}
                className="text-xs text-white/70 hover:text-white sm:text-sm"
              >
                Sign out
              </button>
            </div>
          ) : null}
        </div>
      </header>

      {!user ? (
        <div className="relative z-10 grid min-h-[calc(100vh-3.5rem)] grid-cols-1 md:grid-cols-2">
          <Link
            to="/staff/auth"
            className="group flex min-h-[45vh] flex-col justify-center border-b border-white/15 bg-black/25 p-8 transition hover:bg-black/40 md:min-h-[calc(100vh-3.5rem)] md:border-b-0 md:border-r md:p-12 lg:p-16"
          >
            <div className="mx-auto w-full max-w-md">
              <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                <Shield className="h-8 w-8" strokeWidth={1.5} aria-hidden />
              </div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">
                Transit staff
              </p>
              <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                Agency sign in &amp; register
              </h2>
              <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                Use your work email and password. New accounts need a transit
                code from your administrator.
              </p>
              <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl bg-white px-8 text-sm font-semibold text-black transition group-hover:bg-white/95">
                Continue as staff
              </span>
            </div>
          </Link>

          <Link
            to="/passenger/sign-in"
            className="group flex min-h-[45vh] flex-col justify-center bg-black/15 p-8 transition hover:bg-black/30 md:min-h-[calc(100vh-3.5rem)] md:p-12 lg:p-16"
          >
            <div className="mx-auto w-full max-w-md">
              <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                <UserRound className="h-8 w-8" strokeWidth={1.5} aria-hidden />
              </div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">
                Passenger
              </p>
              <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                Report a lost item
              </h2>
              <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                Sign in with your Google account so we can contact you if your
                property is found.
              </p>
              <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl border-2 border-white bg-transparent px-8 text-sm font-semibold text-white transition group-hover:bg-white group-hover:text-black">
                Continue as passenger
              </span>
            </div>
          </Link>
        </div>
      ) : (
        <div className="relative z-10 grid min-h-[calc(100vh-3.5rem)] grid-cols-1 md:grid-cols-2">
          {/* Staff panel */}
          {user?.role === "staff" ? (
            <Link
              to="/staff"
              className="group flex min-h-[45vh] flex-col justify-center border-b border-white/15 bg-black/25 p-8 transition hover:bg-black/40 md:min-h-[calc(100vh-3.5rem)] md:border-b-0 md:border-r md:p-12 lg:p-16"
            >
              <div className="mx-auto w-full max-w-md">
                <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                  <Shield className="h-8 w-8" strokeWidth={1.5} aria-hidden />
                </div>
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">Transit staff</p>
                <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                  Agency sign in &amp; register
                </h2>
                <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                  Use your work email and password. New accounts need a transit code from your administrator.
                </p>
                <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl bg-white px-8 text-sm font-semibold text-black transition group-hover:bg-white/95">
                  Continue as staff
                </span>
              </div>
            </Link>
          ) : (
            <button
              type="button"
              onClick={() => switchRole("/staff/auth")}
              className="group flex min-h-[45vh] flex-col justify-center border-b border-white/15 bg-black/25 p-8 text-left transition hover:bg-black/40 md:min-h-[calc(100vh-3.5rem)] md:border-b-0 md:border-r md:p-12 lg:p-16"
            >
              <div className="mx-auto w-full max-w-md">
                <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                  <Shield className="h-8 w-8" strokeWidth={1.5} aria-hidden />
                </div>
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">Transit staff</p>
                <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                  Agency sign in &amp; register
                </h2>
                <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                  Use your work email and password. New accounts need a transit code from your administrator.
                </p>
                <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl bg-white px-8 text-sm font-semibold text-black transition group-hover:bg-white/95">
                  Continue as staff
                </span>
              </div>
            </button>
          )}

          {/* Passenger panel */}
          {user?.role === "passenger" ? (
            <Link
              to="/passenger/chat"
              className="group flex min-h-[45vh] flex-col justify-center bg-black/15 p-8 transition hover:bg-black/30 md:min-h-[calc(100vh-3.5rem)] md:p-12 lg:p-16"
            >
              <div className="mx-auto w-full max-w-md">
                <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                  <UserRound className="h-8 w-8" strokeWidth={1.5} aria-hidden />
                </div>
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">Passenger</p>
                <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                  Report a lost item
                </h2>
                <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                  Sign in with your Google account so we can contact you if your property is found.
                </p>
                <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl border-2 border-white bg-transparent px-8 text-sm font-semibold text-white transition group-hover:bg-white group-hover:text-black">
                  Continue as passenger
                </span>
              </div>
            </Link>
          ) : (
            <button
              type="button"
              onClick={() => switchRole("/passenger/sign-in")}
              className="group flex min-h-[45vh] flex-col justify-center bg-black/15 p-8 text-left transition hover:bg-black/30 md:min-h-[calc(100vh-3.5rem)] md:p-12 lg:p-16"
            >
              <div className="mx-auto w-full max-w-md">
                <div className="mb-4 inline-flex rounded-full border border-white/20 bg-white/10 p-3 text-white backdrop-blur-sm">
                  <UserRound className="h-8 w-8" strokeWidth={1.5} aria-hidden />
                </div>
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/70">Passenger</p>
                <h2 className="mt-2 text-balance text-3xl font-semibold tracking-tight text-white md:text-4xl">
                  Report a lost item
                </h2>
                <p className="mt-3 text-pretty text-sm leading-relaxed text-white/80 md:text-base">
                  Sign in with your Google account so we can contact you if your property is found.
                </p>
                <span className="mt-8 inline-flex h-12 items-center justify-center rounded-xl border-2 border-white bg-transparent px-8 text-sm font-semibold text-white transition group-hover:bg-white group-hover:text-black">
                  Continue as passenger
                </span>
              </div>
            </button>
          )}
        </div>
      )}
    </div>
  );
}
