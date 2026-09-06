import type { Metadata } from "next";
import { Messages } from "@/components/communication/messages";
import { Suspense } from "react";
export const metadata: Metadata = { title: "Messages" };
export default function Page() { return <Suspense fallback={<p>Loading messages...</p>}><Messages /></Suspense>; }
