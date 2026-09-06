import type { Metadata } from "next";
import { Moments } from "@/components/social/moments";
export const metadata: Metadata = { title: "Moments" };
export default function Page() { return <Moments />; }
