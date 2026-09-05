import type { Metadata } from "next";
import { brand } from "@/lib/brand";
import "./globals.css";

export const metadata: Metadata = {
  title: brand.name,
  description: "Meet someone new, one conversation at a time.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
