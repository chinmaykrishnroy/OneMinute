import type { Metadata } from "next";
import { Activity } from "@/components/communication/activity";
export const metadata: Metadata = { title: "Activity" };
export default function Page() { return <Activity />; }
