import type { Metadata } from "next";
import { brand } from "@/lib/brand";
import "./globals.css";
import "./product-shell.css";
import { CommunicationProvider } from "@/components/communication/runtime";

export const metadata: Metadata = {
  title: { default: brand.name, template: `%s · ${brand.name}` },
  description: "Meet someone new, one conversation at a time.",
  manifest: "/manifest.webmanifest",
  icons: {
    icon: [{ url: "/icon.svg", type: "image/svg+xml" }, { url: "/icons/favicon.ico", sizes: "any" }],
    apple: "/apple-icon.png",
  },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en" suppressHydrationWarning><body><CommunicationProvider api={process.env.API_PUBLIC_URL ?? "http://localhost:8080"}>{children}</CommunicationProvider></body></html>;
}
