import { useEffect, useState } from "react";
import {
  Link,
  Navigate,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { ArrowLeft, Bus } from "lucide-react";
import { useAuth } from "../context/useAuth";

const STAFF_AUTH_IMAGE =
  "https://images.unsplash.com/photo-1544620347-c4fd4a3d5957?auto=format&fit=crop&w=2000&q=80";

const field =
  "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

export default function StaffAuthPage() {
  const { user, loginStaff, signupStaff } = useAuth();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const tabFromUrl = searchParams.get("mode") === "signup" ? "signup" : "login";
  const [tab, setTab] = useState(tabFromUrl);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [transitCode, setTransitCode] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    setTab(tabFromUrl);
  }, [tabFromUrl]);

  function switchTab(next) {
    setTab(next);
    setError("");
    setSearchParams(next === "signup" ? { mode: "signup" } : {});
  }

  if (user?.role === "staff") {
    return <Navigate to="/staff" replace />;
  }
  if (user?.role === "passenger") {
    return <Navigate to="/passenger" replace />;
  }

  async function handleLogin(e) {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    const result = await loginStaff(email.trim(), password);
    setSubmitting(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    navigate("/staff", { replace: true });
  }

  async function handleSignup(e) {
    e.preventDefault();
    setError("");
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    setSubmitting(true);
    const result = await signupStaff({
      email: email.trim(),
      password,
      invitationCode: transitCode.trim(),
    });
    setSubmitting(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    navigate("/staff", { replace: true });
  }

  return (
    <div className="relative min-h-screen">
      <img
        src={STAFF_AUTH_IMAGE}
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
                <Bus className="h-6 w-6" aria-hidden />
              </div>
              <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">
                Staff access
              </h1>
            </div>

            <div className="flex justify-center">
              <div
                className="inline-flex rounded-xl border border-border bg-muted/40 p-1"
                role="tablist"
                aria-label="Sign in or sign up"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === "login"}
                  onClick={() => switchTab("login")}
                  className={`w-[132px] rounded-lg py-2.5 text-sm font-medium transition sm:w-[152px] ${
                    tab === "login"
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  Sign in
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={tab === "signup"}
                  onClick={() => switchTab("signup")}
                  className={`w-[132px] rounded-lg py-2.5 text-sm font-medium transition sm:w-[152px] ${
                    tab === "signup"
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  Sign up
                </button>
              </div>
            </div>

            {tab === "login" ? (
              <form onSubmit={handleLogin} className="mt-8 space-y-5">
                <div className="space-y-2">
                  <label
                    htmlFor="staff-email"
                    className="text-sm font-medium leading-none"
                  >
                    Email
                  </label>
                  <input
                    id="staff-email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className={field}
                    placeholder="you@transit.agency"
                  />
                </div>
                <div className="space-y-2">
                  <label
                    htmlFor="staff-password"
                    className="text-sm font-medium leading-none"
                  >
                    Password
                  </label>
                  <input
                    id="staff-password"
                    name="password"
                    type="password"
                    autoComplete="current-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={field}
                  />
                </div>
                {error ? (
                  <p className="text-center text-sm text-destructive" role="alert">
                    {error}
                  </p>
                ) : null}
                <button
                  type="submit"
                  disabled={submitting}
                  className="inline-flex h-11 w-full items-center justify-center rounded-xl bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:pointer-events-none disabled:opacity-50"
                >
                  {submitting ? "Signing in…" : "Sign in"}
                </button>
              </form>
            ) : (
              <form onSubmit={handleSignup} className="mt-8 space-y-5">
                <div className="space-y-2">
                  <label
                    htmlFor="signup-email"
                    className="text-sm font-medium leading-none"
                  >
                    Email
                  </label>
                  <input
                    id="signup-email"
                    name="email"
                    type="email"
                    autoComplete="email"
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className={field}
                    placeholder="you@transit.agency"
                  />
                </div>
                <div className="space-y-2">
                  <label
                    htmlFor="transit-code"
                    className="text-sm font-medium leading-none"
                  >
                    Transit code
                  </label>
                  <input
                    id="transit-code"
                    name="transitCode"
                    type="text"
                    autoComplete="off"
                    required
                    value={transitCode}
                    onChange={(e) => setTransitCode(e.target.value)}
                    className={field}
                    placeholder="From your administrator"
                  />
                </div>
                <div className="space-y-2">
                  <label
                    htmlFor="signup-password"
                    className="text-sm font-medium leading-none"
                  >
                    Password
                  </label>
                  <input
                    id="signup-password"
                    name="password"
                    type="password"
                    autoComplete="new-password"
                    required
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className={field}
                  />
                </div>
                <div className="space-y-2">
                  <label
                    htmlFor="signup-confirm"
                    className="text-sm font-medium leading-none"
                  >
                    Confirm password
                  </label>
                  <input
                    id="signup-confirm"
                    name="confirm"
                    type="password"
                    autoComplete="new-password"
                    required
                    value={confirm}
                    onChange={(e) => setConfirm(e.target.value)}
                    className={field}
                  />
                </div>
                {error ? (
                  <p className="text-center text-sm text-destructive" role="alert">
                    {error}
                  </p>
                ) : null}
                <button
                  type="submit"
                  disabled={submitting}
                  className="inline-flex h-11 w-full items-center justify-center rounded-xl bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 disabled:pointer-events-none disabled:opacity-50"
                >
                  {submitting ? "Creating account…" : "Create account"}
                </button>
              </form>
            )}
        </div>
      </div>
    </div>
  );
}
