import { brand } from "@/lib/brand";

export default function Home() {
  return <main>
    <p>One conversation. A new connection.</p>
    <h1>{brand.name}</h1>
    <p>Meet someone for 60 seconds. If you both choose to stay, keep talking.</p>
    <p>The application is under development. Public discovery is not open yet.</p>
  </main>;
}
