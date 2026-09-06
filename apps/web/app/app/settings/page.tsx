import type { Metadata } from "next";
import { AppSettings } from "@/components/communication/settings";
export const metadata: Metadata = { title: "Settings" };
export default function Page() { return <AppSettings />; }
