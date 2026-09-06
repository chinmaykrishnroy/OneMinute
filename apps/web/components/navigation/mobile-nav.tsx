import Link from "next/link";

export function MobileNav({ current }: { current: "discover" | "connections" | "profile" }) {
  return <nav className="mobile-tabs" aria-label="App">
    <Link href="/">Home</Link>
    <Link href="/app/discover" aria-current={current === "discover" ? "page" : undefined}>Discover</Link>
    <Link href="/app/connections" aria-current={current === "connections" ? "page" : undefined}>Connections</Link>
    <Link href="/app/profile" aria-current={current === "profile" ? "page" : undefined}>Profile</Link>
  </nav>;
}
