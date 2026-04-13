import { useState } from "react";
import { Link, Navigate } from "react-router-dom";
import { ArrowLeft, UserRound } from "lucide-react";
import { GoogleLogin } from "@react-oauth/google";
import { useAuth } from "../context/useAuth";

const PASSENGER_HERO_IMAGE =
  "https://images.unsplash.com/photo-1544620347-c4fd4a3d5957?auto=format&fit=crop&w=2000&q=80";

export default function PassengerSignInPage() {
  const { user, loginPassengerGoogle } = useAuth();
  const [error, setError] = useState("");

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
              Continue to chat with the transit assistant using your Google account.
            </p>
          </div>

          <div className="flex flex-col items-center gap-5 rounded-xl border border-border/50 bg-white/60 px-4 py-8 sm:px-6">
            <GoogleLogin
              onSuccess={async (credentialResponse) => {
                setError("");
                const result = await loginPassengerGoogle(
                  credentialResponse?.credential,
                );
                if (!result.ok) {
                  setError(result.error || "Google sign-in failed.");
                }
              }}
              onError={() => {
                setError("Google sign-in was cancelled or failed.");
              }}
              shape="pill"
              theme="outline"
              text="signin_with"
              size="large"
              width="280"
            />
            {error ? (
              <p className="text-center text-sm text-destructive" role="alert">
                {error}
              </p>
            ) : (
              <p className="text-center text-xs text-muted-foreground">
                Secure sign-in via Google.
              </p>
            )}
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
