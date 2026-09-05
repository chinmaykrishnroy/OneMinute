import { brand } from "@/lib/brand";
import { SignIn } from "@/components/auth/sign-in";

export default function Home() {
  const api = process.env.API_PUBLIC_URL ?? "http://localhost:8080";
  return <main>
    <div className="hero">
      <div><p className="eyebrow">One conversation. A new connection.</p><h1>{brand.name}</h1><p className="hero-copy">Meet someone for 60 seconds. If you both choose to stay, keep talking.</p></div>
      <SignIn api={api} />
    </div>
    <p className="development-note">Discovery is being built now. The networking lab is already live.</p>
  </main>;
}
