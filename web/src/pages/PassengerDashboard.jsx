import { useState } from "react";
import { Link } from "react-router-dom";
import { LogOut, MapPin, Send } from "lucide-react";
import { useAuth } from "../context/useAuth";

const field =
  "flex h-11 w-full rounded-xl border border-input bg-background px-3.5 text-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring";

export default function PassengerDashboard() {
  const { user, logout } = useAuth();
  const [itemDescription, setItemDescription] = useState("");
  const [routeOrLine, setRouteOrLine] = useState("");
  const [lostDate, setLostDate] = useState("");
  const [details, setDetails] = useState("");
  const [submitted, setSubmitted] = useState(false);

  function handleSubmit(e) {
    e.preventDefault();
    setSubmitted(true);
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-muted/40 to-background">
      <header className="border-b border-border/80 bg-background/90 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-2xl items-center justify-between px-4">
          <Link
            to="/"
            className="flex items-center gap-2 font-semibold tracking-tight hover:opacity-90"
          >
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-foreground text-background">
              <MapPin className="h-4 w-4" aria-hidden />
            </span>
            <span className="hidden sm:inline">Lost item</span>
          </Link>
          <div className="flex items-center gap-3 text-sm">
            {user?.picture ? (
              <img
                src={user.picture}
                alt=""
                className="h-8 w-8 rounded-full border border-border object-cover"
              />
            ) : null}
            <span className="hidden max-w-[120px] truncate text-muted-foreground sm:inline sm:max-w-[180px]">
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
            Report
          </p>
          <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
            Tell us what you lost
          </h1>
          <p className="max-w-lg text-sm leading-relaxed text-muted-foreground">
            Signed in as{" "}
            <span className="font-medium text-foreground">{user?.email}</span>.
            We’ll use this to contact you if we find a match.
          </p>
        </div>

        {submitted ? (
          <div
            className="rounded-2xl border border-border bg-card p-8 shadow-sm"
            role="status"
          >
            <p className="text-lg font-semibold">We got your report</p>
            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
              Thanks — this demo doesn’t send data to a server yet. In
              production, staff would see your report here.
            </p>
            <button
              type="button"
              onClick={() => {
                setSubmitted(false);
                setItemDescription("");
                setRouteOrLine("");
                setLostDate("");
                setDetails("");
              }}
              className="mt-6 text-sm font-medium text-foreground underline underline-offset-4"
            >
              Submit another report
            </button>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit}
            className="space-y-6 rounded-2xl border border-border bg-card p-6 shadow-sm sm:p-8"
          >
            <div className="space-y-2">
              <label
                htmlFor="lost-item"
                className="text-sm font-medium leading-none"
              >
                What did you lose?
              </label>
              <input
                id="lost-item"
                name="item"
                type="text"
                required
                value={itemDescription}
                onChange={(e) => setItemDescription(e.target.value)}
                className={field}
                placeholder="e.g. Black backpack, keys on a red lanyard"
              />
            </div>

            <div className="grid gap-5 sm:grid-cols-2">
              <div className="space-y-2 sm:col-span-1">
                <label
                  htmlFor="route"
                  className="text-sm font-medium leading-none"
                >
                  Route or line
                </label>
                <input
                  id="route"
                  name="route"
                  type="text"
                  value={routeOrLine}
                  onChange={(e) => setRouteOrLine(e.target.value)}
                  className={field}
                  placeholder="Route 42, Blue Line…"
                />
              </div>
              <div className="space-y-2 sm:col-span-1">
                <label
                  htmlFor="lost-date"
                  className="text-sm font-medium leading-none"
                >
                  Last had it around
                </label>
                <input
                  id="lost-date"
                  name="lostDate"
                  type="date"
                  value={lostDate}
                  onChange={(e) => setLostDate(e.target.value)}
                  className={field}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label
                htmlFor="details"
                className="text-sm font-medium leading-none"
              >
                Anything else?
              </label>
              <textarea
                id="details"
                name="details"
                rows={4}
                value={details}
                onChange={(e) => setDetails(e.target.value)}
                className={`${field} min-h-[120px] resize-y py-3`}
                placeholder="Where you sat, approximate time, color, brand…"
              />
            </div>

            <button
              type="submit"
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-foreground px-4 text-sm font-medium text-background transition hover:opacity-90 sm:w-auto sm:min-w-[180px]"
            >
              <Send className="h-4 w-4" />
              Submit report
            </button>
          </form>
        )}

        <p className="mt-10 text-center text-sm text-muted-foreground">
          Staff member?{" "}
          <Link
            to="/staff/auth"
            className="font-medium text-foreground underline underline-offset-4"
            onClick={() => logout()}
          >
            Sign out and use staff login
          </Link>
        </p>
      </main>
    </div>
  );
}
