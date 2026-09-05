import { notFound } from "next/navigation";
import { NetworkingLab } from "@/components/video/lab";

export const dynamic = "force-dynamic";
export default function LabPage() {
  if (process.env.RTC_LAB_ENABLED !== "true" || process.env.APP_ENV === "production") notFound();
  return <NetworkingLab api={process.env.API_PUBLIC_URL || "http://localhost:8080"} />;
}
