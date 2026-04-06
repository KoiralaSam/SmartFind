import { Link, Navigate, useNavigate } from "react-router-dom";
import { ArrowLeft, UserRound } from "lucide-react";
import { useAuth } from "../context/useAuth";

const PASSENGER_HERO_IMAGE =
  "https://images.unsplash.com/photo-1544620347-c4fd4a3d5957?auto=format&fit=crop&w=2000&q=80";

/** Official multicolor Google “G” mark (matches Sign in with Google branding). */
function GoogleMark() {
  return (
    <svg className="h-5 w-5 shrink-0" viewBox="0 0 24 24" aria-hidden>
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
      />
    </svg>
  );
}

export default function PassengerSignInPage() {
  const navigate = useNavigate();
  const { user } = useAuth();

  if (user?.role === "passenger") {
    return <Navigate to="/passenger/chat" replace />;
  }
  if (user?.role === "staff") {
    return <Navigate to="/staff" replace />;
  }

  return (
    <div className="relative min-h-screen">
      <img
        src={PASSENGER_HERO_IMAGE}
        alt=""
        className="pointer-events-none absolute inset-0 h-full w-full object-cover"
        loading="eager"
        decoding="async"
      />
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-[2px]"
        aria-hidden
      />

      <Link
        to="/"
        className="absolute left-4 top-4 z-20 inline-flex items-center gap-2 text-sm font-medium text-white/90 transition hover:text-white sm:left-6 sm:top-6"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to home
      </Link>

      <div className="relative z-10 flex min-h-screen items-center justify-center px-4 py-16 sm:px-6 sm:py-12">
        <div className="w-full max-w-md rounded-2xl border border-border/80 bg-background/95 p-8 shadow-2xl backdrop-blur-sm sm:p-10">
          <div className="mb-8 flex flex-col items-center text-center">
            <div className="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-foreground text-background">
              <UserRound className="h-6 w-6" aria-hidden strokeWidth={1.75} />
            </div>
            <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">
              Passenger sign-in
            </h1>
            <p className="mt-2 text-pretty text-sm text-muted-foreground sm:text-base">
              Continue to chat with the transit assistant. Google sign-in will be
              added later.
            </p>
          </div>

          <div className="flex flex-col items-center gap-5 rounded-xl border border-border/50 bg-white/60 px-4 py-8 sm:px-6">
            <button
              type="button"
              onClick={() => navigate("/passenger/chat")}
              className="inline-flex h-11 min-w-[240px] max-w-full cursor-pointer items-center justify-center gap-3 rounded-full border border-[#dadce0] bg-white px-6 text-sm font-medium text-[#3c4043] shadow-[0_1px_2px_rgba(0,0,0,0.05)] transition hover:bg-[#f8f9fa]"
            >
              <GoogleMark />
              Sign in with Google
            </button>
            <p className="text-center text-xs text-muted-foreground">
              Opens the assistant chat — no account required for this preview.
            </p>
          </div>

          <p className="mt-8 text-center text-sm text-muted-foreground">
            Transit staff?{" "}
            <Link
              to="/staff/auth"
              className="font-medium text-foreground underline underline-offset-4"
            >
              Staff sign-in
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
