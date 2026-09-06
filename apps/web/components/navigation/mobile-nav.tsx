import Link from "next/link";
import { Icon, IconName } from "./icon";
export type Tab = "discover" | "messages" | "posts" | "profile" | "connections";
const tabs: { key: IconName; href: string; label: string }[] = [
 { key: "discover", href: "/app/discover", label: "Discover" },
 { key: "messages", href: "/app/messages", label: "Messages" },
 { key: "posts", href: "/app/posts", label: "Posts" },
 { key: "profile", href: "/app/profile", label: "You" },
];
export function MobileNav({ current }: { current: Tab }) {
 return <nav className="product-nav" aria-label="Main navigation">{tabs.map(tab => <Link key={tab.key} href={tab.href} aria-label={tab.label} title={tab.label} aria-current={current === tab.key ? "page" : undefined}><Icon name={tab.key} /><span>{tab.label}</span><i aria-hidden="true" /></Link>)}</nav>;
}
export function AppHeader({ title }: { title: string }) {
 return <header className="app-header product-header"><Link className="wordmark" href="/app/discover"><svg width="28" height="28" viewBox="0 0 32 32" aria-hidden="true"><rect x="2" y="2" width="28" height="28" rx="9" fill="#d7edb7" stroke="currentColor" strokeWidth="2"/><path d="M16 8v8l6-3" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"/></svg>OneMinute</Link><span className="page-label">{title}</span><Link className="header-account" href="/app/profile" aria-label="Your account"><Icon name="profile" /></Link></header>;
}
