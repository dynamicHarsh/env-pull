import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import Navbar from "@/components/Navbar";
import Footer from "@/components/Footer";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetbrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL("https://envpull.tech"),
  title: {
    default: "env-pull | The Universal Secrets Adapter",
    template: "%s | env-pull",
  },
  description:
    "Zero-disk, zero-config secrets injection for local development. Stop shuffling .env files. Secure your local workflow with AWS and 1Password integration.",
  keywords: [
    "secrets management",
    "cli",
    "developer tools",
    "env variables",
    "zero-trust",
    "open source",
    "devsecops",
    "local development",
  ],
  openGraph: {
    type: "website",
    url: "https://envpull.tech",
    title: "env-pull | The Universal Secrets Adapter",
    description:
      "Zero-disk, zero-config secrets injection for local development. Stop shuffling .env files.",
    siteName: "env-pull",
  },
  twitter: {
    card: "summary_large_image",
    title: "env-pull | The Universal Secrets Adapter",
    description:
      "Zero-disk, zero-config secrets injection for local development. Stop shuffling .env files.",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${jetbrainsMono.variable} h-full`}
    >
      <body className="bg-zinc-950 text-zinc-100 antialiased font-sans min-h-full flex flex-col">
        <Navbar />
        <main className="flex-grow flex flex-col">
          {children}
        </main>
        <Footer />
      </body>
    </html>
  );
}
