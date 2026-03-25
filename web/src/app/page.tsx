"use client";

import { FormEvent, useState } from "react";
import { APP_NAME } from "../constants";
import type { ItemCategory } from "../types";

interface SubmittedLostReport {
  id: string;
  fullName: string;
  category: ItemCategory;
  description: string;
  lostAt: string;
  status: "submitted";
}

const categories: ItemCategory[] = [
  "electronics",
  "wallet",
  "bag",
  "documents",
  "clothing",
  "other",
];

export default function Home() {
  const [fullName, setFullName] = useState("");
  const [category, setCategory] = useState<ItemCategory>("electronics");
  const [description, setDescription] = useState("");
  const [submittedReport, setSubmittedReport] = useState<SubmittedLostReport | null>(null);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const now = new Date();
    const report: SubmittedLostReport = {
      id: `LST-${now.getTime()}`,
      fullName,
      category,
      description,
      lostAt: now.toISOString(),
      status: "submitted",
    };

    setSubmittedReport(report);
  }

  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 md:px-8">
      <div className="mx-auto w-full max-w-3xl rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
        <h1 className="mb-4 text-2xl font-bold">{APP_NAME}</h1>

        <form className="grid gap-4" onSubmit={handleSubmit}>
          <label className="grid gap-1 text-sm">
            Full Name
            <input
              value={fullName}
              onChange={(event) => setFullName(event.target.value)}
              className="rounded-md border border-slate-300 px-3 py-2 outline-none ring-emerald-500 transition focus:ring-2"
              placeholder="Enter full name"
              required
            />
          </label>

          <label className="grid gap-1 text-sm">
            Category
            <select
              value={category}
              onChange={(event) => setCategory(event.target.value as ItemCategory)}
              className="rounded-md border border-slate-300 px-3 py-2 outline-none ring-emerald-500 transition focus:ring-2"
            >
              {categories.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>

          <label className="grid gap-1 text-sm">
            Description
            <textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="min-h-24 rounded-md border border-slate-300 px-3 py-2 outline-none ring-emerald-500 transition focus:ring-2"
              placeholder="Black backpack with charger"
              required
            />
          </label>

          <button
            type="submit"
            className="rounded-md bg-emerald-600 px-4 py-2 font-semibold text-white transition hover:bg-emerald-700"
          >
            Submit Mock Report
          </button>
        </form>

        <h2 className="mt-6 text-lg font-semibold">Latest Payload</h2>
        <pre className="mt-2 overflow-x-auto rounded-md bg-slate-950 p-3 text-xs text-slate-100">
          {JSON.stringify(submittedReport, null, 2) || "Submit a report to see data"}
        </pre>
      </div>
    </main>
  );
}
