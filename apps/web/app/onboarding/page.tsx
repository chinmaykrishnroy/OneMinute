import type { Metadata } from "next";
import { Onboarding } from "@/components/social/onboarding";
export const metadata: Metadata = { title: "Welcome" };
export default function Page() { return <Onboarding api={process.env.API_PUBLIC_URL ?? "http://localhost:8080"} />; }
