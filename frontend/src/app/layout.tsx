import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import Navbar from "@/components/Navbar";
import MobileBottomNav from "@/components/MobileBottomNav";

const themeScript = `(function(){try{var saved=localStorage.getItem('rojgarbridge-theme');var dark=saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.classList.toggle('dark',dark)}catch(e){}})();`;

const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-inter",
});

export const metadata: Metadata = {
  title: "RojgarBridge | Verified Government Jobs and Exam Updates",
  description: "Find verified government job notifications, admit cards and results from UPSC, SSC, Railways, Banking and State PSCs on RojgarBridge.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={inter.variable} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className={`${inter.className} min-h-screen flex flex-col bg-[#f8fafc] text-slate-900 antialiased pb-16 lg:pb-0`}>
        <Navbar />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-5 sm:px-6 sm:py-7 lg:px-8">
          {children}
        </main>
        <footer className="hidden border-t border-slate-200 bg-white lg:block">
          <div className="mx-auto flex max-w-7xl items-center justify-between px-8 py-6 text-sm text-slate-500">
            <p>© 2026 RojgarBridge. Information simplified from official sources.</p>
            <p>Always verify details on the official portal before applying.</p>
          </div>
        </footer>
        <MobileBottomNav />
      </body>
    </html>
  );
}
