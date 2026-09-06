import type { Metadata } from "next";
import { Inbox } from "@/components/social/inbox";
export const metadata: Metadata = { title: "Messages" };
export default function Page() { const api = process.env.API_PUBLIC_URL ?? "http://localhost:8080"; return <Inbox api={api} />; }
