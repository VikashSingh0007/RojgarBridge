import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";
import MobileBottomNav from "@/components/MobileBottomNav";

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-inter",
});

export const metadata: Metadata = {
  title: "NaukriBridge | Official Sarkari Naukri & Government Job Updates",
  description: "Explore 100% verified government job notifications, admit cards, results, and syllabus with simplified official updates from UPSC, SSC, RRB, Banking, and State PSCs on NaukriBridge.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={inter.variable}>
      <body className={`${inter.className} min-h-screen flex flex-col bg-[#f8fafc] text-slate-900 antialiased pb-16 lg:pb-0`}>
        <Navbar />
        <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
          {children}
        </main>
        <MobileBottomNav />
      </body>
    </html>
  );
}
