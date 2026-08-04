import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = { title: "Codex Tempo", description: "Agent work accounting" };

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="es"><body>{children}</body></html>;
}
