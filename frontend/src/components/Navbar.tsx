"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Bookmark,
  BriefcaseBusiness,
  FileCheck2,
  Globe2,
  LayoutDashboard,
  Megaphone,
  Menu,
  Trophy,
  X,
  LogOut,
} from "lucide-react";
import ThemeToggle from "@/components/ThemeToggle";
import { getGoogleLoginUrl } from "@/lib/api";
import { User } from "@/types";

export default function Navbar() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [currentHash, setCurrentHash] = useState<string>("#jobs");
  const pathname = usePathname();
  const isAdmin = pathname.startsWith("/admin");

  useEffect(() => {
    try {
      const stored = localStorage.getItem("rozgarhub_user");
      if (stored) {
        setUser(JSON.parse(stored));
      } else {
        const adminStored = localStorage.getItem("rozgarhub_admin_user");
        if (adminStored) setUser(JSON.parse(adminStored));
      }
    } catch {}
  }, [pathname]);

  // Track hash changes for active nav highlighting
  useEffect(() => {
    const syncHash = () => {
      setCurrentHash(window.location.hash || "#jobs");
    };
    syncHash();
    window.addEventListener("hashchange", syncHash);
    return () => window.removeEventListener("hashchange", syncHash);
  }, [pathname]);

  const handleLogout = () => {
    localStorage.removeItem("rozgarhub_user");
    localStorage.removeItem("rozgarhub_user_token");
    localStorage.removeItem("rozgarhub_admin_user");
    localStorage.removeItem("rozgarhub_admin_token");
    setUser(null);
    window.location.reload();
  };

  const navLinks = [
    { name: "Jobs & Recruitments", href: "/#jobs", hash: "#jobs", icon: BriefcaseBusiness },
    { name: "Announcements", href: "/#announcements", hash: "#announcements", icon: Megaphone },
    { name: "Admit Cards", href: "/#admit-cards", hash: "#admit-cards", icon: FileCheck2 },
    { name: "Results", href: "/#results", hash: "#results", icon: Trophy },
  ];

  const isLinkActive = (href: string) => {
    if (isAdmin) return false;
    if (href === "/#jobs") {
      return !currentHash || currentHash === "#jobs" || currentHash === "";
    }
    if (href === "/#announcements") {
      return currentHash === "#announcements";
    }
    if (href === "/#admit-cards") {
      return currentHash.startsWith("#admit");
    }
    if (href === "/#results") {
      return currentHash.startsWith("#result");
    }
    return false;
  };

  return (
    <header className="sticky top-0 z-40 border-b border-slate-200/80 bg-white/95 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-900/95">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-3 sm:px-6 lg:px-8 gap-2 sm:gap-4">
        {/* Brand / Logo */}
        <Link
          href="/"
          className="group flex items-center gap-2.5 sm:gap-3 shrink-0"
          aria-label="RojgarBridge home"
        >
          <div className="flex h-9 w-9 sm:h-10 sm:w-10 shrink-0 items-center justify-center rounded-xl bg-slate-950 text-white shadow-sm transition-transform group-hover:-translate-y-0.5 dark:bg-teal-600">
            <svg viewBox="0 0 32 32" className="h-5 w-5 sm:h-6 sm:w-6" fill="none" stroke="currentColor">
              <path
                d="M4 24.5C8.5 16.5 23.5 16.5 28 24.5M3 25H29M10.5 20V25M21.5 20V25"
                strokeWidth="2"
                strokeLinecap="round"
              />
              <path
                d="M16 4.5L17.1 8.9L21.5 10L17.1 11.1L16 15.5L14.9 11.1L10.5 10L14.9 8.9L16 4.5Z"
                fill="currentColor"
                stroke="none"
              />
            </svg>
          </div>
          <div className="flex flex-col">
            <div className="text-lg sm:text-xl font-extrabold tracking-tight text-slate-950 dark:text-white leading-tight whitespace-nowrap">
              Rojgar<span className="text-teal-700 dark:text-teal-400">Bridge</span>
            </div>
            <div className="hidden lg:block text-[11px] font-medium text-slate-500 dark:text-slate-400 leading-tight whitespace-nowrap">
              Verified government opportunities
            </div>
          </div>
        </Link>

        {/* Primary Desktop Navigation Links (Large Screens) */}
        <nav className="hidden xl:flex items-center gap-1 shrink-0" aria-label="Primary navigation">
          {navLinks.map(({ name, href, icon: Icon }) => {
            const active = isLinkActive(href);
            return (
              <Link
                key={name}
                href={href}
                className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs lg:text-sm font-semibold whitespace-nowrap shrink-0 transition-colors ${
                  active
                    ? "bg-slate-100 text-slate-950 font-bold dark:bg-slate-800 dark:text-white"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-950 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-white"
                }`}
              >
                <Icon className={`h-4 w-4 shrink-0 ${active ? "text-teal-700 dark:text-teal-400" : ""}`} aria-hidden="true" />
                <span>{name}</span>
              </Link>
            );
          })}
        </nav>

        {/* Right Section Controls */}
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
          <ThemeToggle compact />

          {/* Saved Bookmarks Button (Visible on md and up) */}
          <a
            href="/#saved-jobs"
            className="hidden md:inline-flex h-9 sm:h-10 items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 text-xs sm:text-sm font-semibold text-slate-700 hover:border-blue-200 hover:bg-blue-50 hover:text-blue-700 whitespace-nowrap shrink-0 transition-colors dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
          >
            <Bookmark className="h-3.5 w-3.5 sm:h-4 sm:w-4 shrink-0" />
            <span>Saved</span>
          </a>

          {/* User Profile / Google Sign in */}
          {user ? (
            <div className="flex items-center gap-1 sm:gap-1.5 pl-1 border-l border-slate-200 dark:border-slate-700 shrink-0">
              <div className="flex items-center gap-1.5 px-2 py-1 sm:py-1.5 rounded-lg bg-slate-50 border border-slate-200 dark:bg-slate-800 dark:border-slate-700 shrink-0">
                {user.avatar_url ? (
                  <img
                    src={user.avatar_url}
                    alt={user.name}
                    className="w-5 h-5 rounded-full object-cover shrink-0"
                  />
                ) : (
                  <div className="w-5 h-5 rounded-full bg-blue-600 text-white text-[10px] font-bold flex items-center justify-center shrink-0">
                    {user.name ? user.name.charAt(0).toUpperCase() : "U"}
                  </div>
                )}
                <span className="text-xs font-bold text-slate-800 dark:text-slate-200 max-w-[80px] sm:max-w-[110px] truncate whitespace-nowrap">
                  {user.name.split(" ")[0]}
                </span>
              </div>
              <button
                onClick={handleLogout}
                title="Sign out"
                className="p-1.5 sm:p-2 text-slate-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-slate-800 rounded-lg transition-colors cursor-pointer shrink-0"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <a
              href={getGoogleLoginUrl()}
              onClick={(e) => {
                e.preventDefault();
                window.location.href = getGoogleLoginUrl();
              }}
              className="inline-flex h-9 sm:h-10 items-center gap-1.5 sm:gap-2 rounded-lg border border-slate-200 bg-white px-2.5 sm:px-3 text-xs font-bold text-slate-700 hover:border-slate-300 hover:bg-slate-50 transition-colors shadow-2xs whitespace-nowrap shrink-0 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 cursor-pointer"
            >
              <svg className="w-3.5 h-3.5 shrink-0" viewBox="0 0 24 24">
                <path
                  fill="#4285F4"
                  d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                />
                <path
                  fill="#34A853"
                  d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                />
                <path
                  fill="#FBBC05"
                  d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
                />
                <path
                  fill="#EA4335"
                  d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
                />
              </svg>
              <span className="whitespace-nowrap">Sign in</span>
            </a>
          )}

          {/* Public / Admin Toggle (Visible on sm and up) */}
          <div
            className="hidden sm:inline-flex h-9 sm:h-10 items-center rounded-lg border border-slate-200 bg-slate-100 p-0.5 sm:p-1 shrink-0 dark:border-slate-700 dark:bg-slate-800"
            role="group"
            aria-label="Switch workspace"
          >
            <Link
              href="/"
              aria-current={!isAdmin ? "page" : undefined}
              className={`inline-flex h-7 sm:h-8 items-center gap-1 rounded-md px-2 sm:px-2.5 text-xs font-bold whitespace-nowrap transition-all ${
                !isAdmin
                  ? "bg-white text-slate-950 shadow-sm dark:bg-slate-900 dark:text-white"
                  : "text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
              }`}
            >
              <Globe2 className="h-3 w-3 sm:h-3.5 sm:w-3.5 shrink-0" />
              <span>Public</span>
            </Link>
            <Link
              href="/admin"
              aria-current={isAdmin ? "page" : undefined}
              className={`inline-flex h-7 sm:h-8 items-center gap-1 rounded-md px-2 sm:px-2.5 text-xs font-bold whitespace-nowrap transition-all ${
                isAdmin
                  ? "bg-slate-950 text-white shadow-sm dark:bg-teal-600 dark:text-white"
                  : "text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white"
              }`}
            >
              <LayoutDashboard className="h-3 w-3 sm:h-3.5 sm:w-3.5 shrink-0" />
              <span>Admin</span>
            </Link>
          </div>

          {/* Mobile & Tablet Hamburger Button (Visible on screens < xl) */}
          <button
            onClick={() => setMobileMenuOpen((open) => !open)}
            className="flex h-9 w-9 sm:h-10 sm:w-10 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 hover:bg-slate-100 transition-colors xl:hidden shrink-0 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
            aria-label="Toggle navigation"
            aria-expanded={mobileMenuOpen}
          >
            {mobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>
      </div>

      {/* Mobile & Tablet Full Navigation Drawer */}
      {mobileMenuOpen && (
        <div className="border-t border-slate-200/80 bg-white/98 px-4 py-3 shadow-lg xl:hidden dark:border-slate-800 dark:bg-slate-900/98 animate-in slide-in-from-top-2 duration-150">
          <nav className="mx-auto grid max-w-7xl gap-2" aria-label="Mobile navigation">
            {/* Switch workspace toggle inside mobile menu */}
            <div
              className="grid grid-cols-2 rounded-xl bg-slate-100 p-1 dark:bg-slate-800"
              role="group"
              aria-label="Switch workspace"
            >
              <Link
                href="/"
                onClick={() => setMobileMenuOpen(false)}
                className={`flex min-h-9 items-center justify-center gap-1.5 rounded-lg text-xs font-bold transition-all ${
                  !isAdmin
                    ? "bg-white text-slate-950 shadow-sm dark:bg-slate-900 dark:text-white"
                    : "text-slate-500 hover:text-slate-900 dark:text-slate-400"
                }`}
              >
                <Globe2 className="h-3.5 w-3.5" />
                <span>Public Portal</span>
              </Link>
              <Link
                href="/admin"
                onClick={() => setMobileMenuOpen(false)}
                className={`flex min-h-9 items-center justify-center gap-1.5 rounded-lg text-xs font-bold transition-all ${
                  isAdmin
                    ? "bg-slate-950 text-white shadow-sm dark:bg-teal-600 dark:text-white"
                    : "text-slate-500 hover:text-slate-900 dark:text-slate-400"
                }`}
              >
                <LayoutDashboard className="h-3.5 w-3.5" />
                <span>Admin Panel</span>
              </Link>
            </div>

            {/* Primary Nav Links */}
            <div className="grid gap-1 pt-1 border-t border-slate-100 dark:border-slate-800">
              {navLinks.map(({ name, href, icon: Icon }) => {
                const active = isLinkActive(href);
                return (
                  <Link
                    key={name}
                    href={href}
                    onClick={() => setMobileMenuOpen(false)}
                    className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-xs sm:text-sm font-semibold transition-colors ${
                      active
                        ? "bg-slate-100 text-slate-950 font-bold dark:bg-slate-800 dark:text-white"
                        : "text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800"
                    }`}
                  >
                    <Icon className={`h-4 w-4 ${active ? "text-teal-700 dark:text-teal-400" : "text-slate-500"}`} />
                    <span>{name}</span>
                  </Link>
                );
              })}

              {/* Saved items shortcut in mobile menu */}
              <a
                href="/#saved-jobs"
                onClick={() => setMobileMenuOpen(false)}
                className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-xs sm:text-sm font-semibold text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800 transition-colors"
              >
                <Bookmark className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                <span>Saved Jobs</span>
              </a>
            </div>
          </nav>
        </div>
      )}
    </header>
  );
}
