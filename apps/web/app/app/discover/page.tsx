import type { Metadata } from "next";
import { Discovery } from "@/components/discovery/discovery";

export const metadata: Metadata = { title: "Discover" };

export default function DiscoverPage() {
  return <Discovery api={process.env.API_PUBLIC_URL ?? "http://localhost:8080"} />;
}
