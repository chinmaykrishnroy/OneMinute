"use client";
import { useEffect, useState, FormEvent } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import {
  Profile,
  profileReady,
  intents,
  languages,
  interests,
  interestLabel,
} from "@/lib/preferences";

export function Onboarding({ api }: { api: string }) {
  const router = useRouter();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [step, setStep] = useState(0);
  const [status, setStatus] = useState("Loading your account...");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let stopped = false;
    async function load() {
      try {
        const response = await fetch(new URL("/v1/profile", api), {
          credentials: "include",
        });
        if (response.status === 401) {
          router.replace("/");
          return;
        }
        if (!response.ok) throw new Error();
        const value: Profile = await response.json();
        if (stopped) return;
        if (profileReady(value)) {
          router.replace("/app/discover");
          return;
        }
        setProfile({
          ...value,
          discoveryIntent: value.discoveryIntent || "surprise_me",
        });
        setStatus("");
      } catch {
        if (!stopped) setStatus("Could not load your account. Please refresh.");
      }
    }
    void load();
    return () => {
      stopped = true;
    };
  }, [api, router]);
  function toggle(kind: "languages" | "interests", value: string) {
    setProfile((p) =>
      p
        ? {
            ...p,
            [kind]: p[kind].includes(value)
              ? p[kind].filter((item) => item !== value)
              : [...p[kind], value].slice(0, kind === "languages" ? 5 : 12),
          }
        : p,
    );
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!profile || busy) return;
    if (step < 2) {
      setStep(step + 1);
      return;
    }
    setBusy(true);
    try {
      const response = await fetch(new URL("/v1/profile", api), {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(profile),
      });
      if (!response.ok) throw new Error();
      router.replace("/app/discover");
    } catch {
      setStatus("Could not save. Please try again.");
      setBusy(false);
    }
  }
  return (
    <main className="onboarding-shell">
      <Link className="wordmark" href="/">
        OneMinute
      </Link>
      <section className="onboarding-card">
        <p className="eyebrow">Welcome / Step {step + 1} of 3</p>
        <progress max={3} value={step + 1} aria-label="Setup progress" />
        <h1>
          {
            [
              "Let's start with you.",
              "Your kind of conversation.",
              "Find some common ground.",
            ][step]
          }
        </h1>
        <p>
          Just a few details before your first hello. You can always change
          these in the You tab.
        </p>
        {profile && (
          <form className="profile-form" onSubmit={submit}>
            {step === 0 && (
              <label>
                What should we call you?
                <input
                  autoComplete="nickname"
                  required
                  maxLength={80}
                  value={profile.displayName}
                  onChange={(e) =>
                    setProfile({ ...profile, displayName: e.target.value })
                  }
                />
              </label>
            )}
            {step === 1 && (
              <>
                <label>
                  I am here for
                  <select
                    value={profile.discoveryIntent}
                    onChange={(e) =>
                      setProfile({
                        ...profile,
                        discoveryIntent: e.target.value,
                      })
                    }
                  >
                    {intents.map(([value, label]) => (
                      <option key={value} value={value}>
                        {label}
                      </option>
                    ))}
                  </select>
                </label>
                <p className="field-hint">
                  Dating only matches with another person who also chooses
                  Dating.
                </p>
                <fieldset>
                  <legend>Languages you speak (choose 1 to 5)</legend>
                  <div className="interest-grid">
                    {languages.map(([value, label]) => (
                      <label className="interest" key={value}>
                        <input
                          type="checkbox"
                          checked={profile.languages.includes(value)}
                          onChange={() => toggle("languages", value)}
                        />
                        {label}
                      </label>
                    ))}
                  </div>
                </fieldset>
              </>
            )}
            {step === 2 && (
              <fieldset>
                <legend>A few things you enjoy (optional)</legend>
                <div className="interest-grid">
                  {interests.map((value) => (
                    <label className="interest" key={value}>
                      <input
                        type="checkbox"
                        checked={profile.interests.includes(value)}
                        onChange={() => toggle("interests", value)}
                      />
                      {interestLabel(value)}
                    </label>
                  ))}
                </div>
              </fieldset>
            )}
            <div className="setup-actions">
              {step > 0 && (
                <button
                  type="button"
                  className="quiet-button"
                  onClick={() => setStep(step - 1)}
                >
                  Back
                </button>
              )}
              <button
                disabled={
                  busy ||
                  !profile.displayName.trim() ||
                  (step > 0 && !profile.languages.length)
                }
              >
                {busy
                  ? "Saving..."
                  : step === 2
                    ? "Start discovering"
                    : "Continue"}
              </button>
            </div>
          </form>
        )}
        <p role="status">{status}</p>
      </section>
    </main>
  );
}
