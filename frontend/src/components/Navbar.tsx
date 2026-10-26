"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Bookmark, BriefcaseBusiness, FileCheck2, Globe2, Languages, LayoutDashboard, Menu, Trophy, X } from "lucide-react";
import ThemeToggle from "@/components/ThemeToggle";

export default function Navbar() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const pathname = usePathname();
  const isAdmin = pathname.startsWith("/admin");
  const navLinks = [
    { name: "Jobs", href: "/#jobs", icon: BriefcaseBusiness },
    { name: "Admit Cards", href: "/#admit-cards", icon: FileCheck2 },
    { name: "Results", href: "/#results", icon: Trophy },
  ];

  return (
    <header className="sticky top-0 z-40 border-b border-slate-200/80 bg-white/95 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <Link href="/" className="group flex items-center gap-3" aria-label="RojgarBridge home">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-950 text-white shadow-sm transition-transform group-hover:-translate-y-0.5">
            <svg viewBox="0 0 32 32" className="h-6 w-6" fill="none" stroke="currentColor">
              <path d="M4 24.5C8.5 16.5 23.5 16.5 28 24.5M3 25H29M10.5 20V25M21.5 20V25" strokeWidth="2" strokeLinecap="round" />
              <path d="M16 4.5L17.1 8.9L21.5 10L17.1 11.1L16 15.5L14.9 11.1L10.5 10L14.9 8.9L16 4.5Z" fill="currentColor" stroke="none" />
            </svg>
          </div>
          <div>
            <div className="text-xl font-extrabold tracking-tight text-slate-950">Rojgar<span className="text-teal-700">Bridge</span></div>
            <div className="text-xs font-medium text-slate-500">Verified government opportunities</div>
          </div>
        </Link>

        <nav className="hidden items-center gap-1 lg:flex" aria-label="Primary navigation">
          {navLinks.map(({ name, href, icon: Icon }, index) => (
            <Link key={name} href={href} className={`inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-colors ${!isAdmin && index === 0 ? "bg-slate-100 text-slate-950" : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"}`}>
              <Icon className="h-4 w-4" aria-hidden="true" />{name}
            </Link>
          ))}
        </nav>

        <div className="hidden items-center gap-2 sm:flex">
          <ThemeToggle compact />
          <button className="inline-flex h-10 items-center gap-2 rounded-lg px-3 text-sm font-semibold text-slate-600 hover:bg-slate-50 hover:text-slate-950" aria-label="Change language">
            <Languages className="h-4 w-4" /><span className="hidden md:inline">English</span>
          </button>
          <a href="/#saved-jobs" className="inline-flex h-10 items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 hover:border-blue-200 hover:bg-blue-50 hover:text-blue-700">
            <Bookmark className="h-4 w-4" />Saved jobs
          </a>
          <div className="flex h-10 items-center rounded-lg border border-slate-200 bg-slate-100 p-1" role="group" aria-label="Switch workspace">
            <Link href="/" aria-current={!isAdmin ? "page" : undefined} className={`inline-flex h-8 items-center gap-1.5 rounded-md px-3 text-xs font-bold transition-all ${!isAdmin ? "bg-white text-slate-950 shadow-sm" : "text-slate-500 hover:text-slate-900"}`}>
              <Globe2 className="h-3.5 w-3.5" />Public
            </Link>
            <Link href="/admin" aria-current={isAdmin ? "page" : undefined} className={`inline-flex h-8 items-center gap-1.5 rounded-md px-3 text-xs font-bold transition-all ${isAdmin ? "bg-slate-950 text-white shadow-sm" : "text-slate-500 hover:text-slate-900"}`}>
              <LayoutDashboard className="h-3.5 w-3.5" />Admin
            </Link>
          </div>
        </div>

        <button onClick={() => setMobileMenuOpen((open) => !open)} className="flex h-10 w-10 items-center justify-center rounded-lg text-slate-700 hover:bg-slate-100 sm:hidden" aria-label="Toggle navigation" aria-expanded={mobileMenuOpen}>
          {mobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {mobileMenuOpen && (
        <div className="border-t border-slate-100 bg-white px-4 py-3 sm:hidden">
          <nav className="mx-auto grid max-w-7xl gap-1" aria-label="Mobile navigation">
            <div className="mb-2 flex items-center justify-between rounded-xl border border-slate-200 bg-slate-50 p-2">
              <span className="px-2 text-sm font-semibold text-slate-600">Appearance</span>
              <ThemeToggle />
            </div>
            <div className="mb-2 grid grid-cols-2 rounded-xl bg-slate-100 p-1" role="group" aria-label="Switch workspace">
              <Link href="/" onClick={() => setMobileMenuOpen(false)} className={`flex min-h-10 items-center justify-center gap-2 rounded-lg text-sm font-bold ${!isAdmin ? "bg-white text-slate-950 shadow-sm" : "text-slate-500"}`}>
                <Globe2 className="h-4 w-4" />Public
              </Link>
              <Link href="/admin" onClick={() => setMobileMenuOpen(false)} className={`flex min-h-10 items-center justify-center gap-2 rounded-lg text-sm font-bold ${isAdmin ? "bg-slate-950 text-white shadow-sm" : "text-slate-500"}`}>
                <LayoutDashboard className="h-4 w-4" />Admin
              </Link>
            </div>
            {navLinks.map(({ name, href, icon: Icon }) => (
              <Link key={name} href={href} onClick={() => setMobileMenuOpen(false)} className="flex items-center gap-3 rounded-lg px-3 py-3 text-sm font-semibold text-slate-700 hover:bg-slate-50">
                <Icon className="h-4 w-4 text-blue-700" />{name}
              </Link>
            ))}
          </nav>
        </div>
      )}
    </header>
  );
}
