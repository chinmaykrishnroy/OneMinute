import { Discovery } from "@/components/discovery/discovery";

export default function DiscoverPage() {
  return <Discovery api={process.env.API_PUBLIC_URL ?? "http://localhost:8080"} />;
}
