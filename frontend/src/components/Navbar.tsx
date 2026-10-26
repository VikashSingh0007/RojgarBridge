"use client";

import { useState } from "react";
import Link from "next/link";

interface NavbarProps {
  onSearchClick?: () => void;
}

export default function Navbar({ onSearchClick }: NavbarProps) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [activeTab, setActiveTab] = useState("Jobs");

  const navLinks = [
    { name: "Home", href: "#" },
    { name: "Jobs", href: "#" },
    { name: "Admit Card", href: "#" },
    { name: "Results", href: "#" },
    { name: "Syllabus", href: "#" },
    { name: "Exam Calendar", href: "#" },
    { name: "Contact", href: "#" },
  ];

  return (
    <header className="bg-white/95 backdrop-blur-md border-b border-slate-200/80 sticky top-0 z-40 shadow-2xs">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16 sm:h-18">
          {/* Logo Brand */}
          <Link href="/" className="flex items-center gap-2.5 sm:gap-3 group">
            {/* Modern Geometric Bridge & Career Star Icon */}
            <div className="w-10 h-10 sm:w-11 sm:h-11 rounded-xl bg-gradient-to-tr from-blue-600 via-indigo-600 to-sky-500 flex items-center justify-center shadow-md shadow-blue-500/25 group-hover:scale-104 transition-all duration-200 border border-white/25 flex-shrink-0">
              <svg
                viewBox="0 0 32 32"
                className="w-6 h-6 text-white"
                fill="none"
                stroke="currentColor"
              >
                {/* Central Star of Opportunity */}
                <path
                  d="M16 4.5L17.2 9.2L22 10.5L17.2 11.8L16 16.5L14.8 11.8L10 10.5L14.8 9.2L16 4.5Z"
                  fill="currentColor"
                  stroke="none"
                />
                {/* Modern Suspension Bridge Arch */}
                <path
                  d="M4 25.5C8.5 17.5 23.5 17.5 28 25.5"
                  strokeWidth="2.2"
                  strokeLinecap="round"
                />
                {/* Bridge Foundation Deck */}
                <path
                  d="M3 26H29"
                  strokeWidth="2.2"
                  strokeLinecap="round"
                />
                {/* Suspension Pillars */}
                <path
                  d="M10.5 21V26M21.5 21V26"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                  strokeOpacity="0.7"
                />
              </svg>
            </div>

            <div className="flex flex-col">
              <div className="flex items-center gap-1.5">
                <span className="text-xl sm:text-2xl font-black tracking-tight text-slate-900 leading-none">
                  Naukri<span className="bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">Bridge</span>
                </span>
                <span className="hidden sm:inline-block text-[9px] font-black uppercase tracking-wider text-blue-700 bg-blue-50 px-1.5 py-0.5 rounded-md border border-blue-200/60 shadow-2xs">
                  Govt Jobs
                </span>
              </div>
              <span className="text-[11px] font-medium text-slate-500 tracking-wide mt-0.5 flex items-center gap-1.5">
                <span>Official Sarkari Updates</span>
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 inline-block"></span>
              </span>
            </div>
          </Link>

          {/* Desktop Navigation Links */}
          <nav className="hidden lg:flex items-center gap-7 h-full">
            {navLinks.map((link) => {
              const isActive = activeTab === link.name;
              return (
                <button
                  key={link.name}
                  onClick={() => setActiveTab(link.name)}
                  className={`relative h-full flex items-center text-sm font-semibold transition-colors px-1 ${
                    isActive
                      ? "text-blue-600"
                      : "text-slate-600 hover:text-slate-900"
                  }`}
                >
                  {link.name}
                  {isActive && (
                    <span className="absolute bottom-0 left-0 right-0 h-[3px] bg-blue-600 rounded-t-md" />
                  )}
                </button>
              );
            })}
          </nav>

          {/* Right Action Buttons */}
          <div className="hidden sm:flex items-center gap-3">
            <Link
              href="/admin"
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg border border-slate-300 transition-colors"
            >
              <span>🛡️ Admin Review</span>
              <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse"></span>
            </Link>

            <button
              onClick={onSearchClick}
              className="p-2 text-slate-600 hover:text-blue-600 hover:bg-slate-50 rounded-full transition-colors"
              title="Search"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </button>

            <button className="px-4 py-2 text-sm font-semibold text-slate-700 hover:text-blue-600 rounded-lg transition-colors">
              Login
            </button>

            <button className="px-5 py-2 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-all hover:shadow">
              Sign Up
            </button>
          </div>

          {/* Mobile hamburger & search */}
          <div className="flex lg:hidden items-center gap-2">
            <button
              onClick={onSearchClick}
              className="p-2 text-slate-700 hover:bg-slate-100 rounded-lg"
              title="Search"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </button>

            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="p-2 text-slate-700 hover:bg-slate-100 rounded-lg"
              aria-label="Toggle menu"
            >
              {mobileMenuOpen ? (
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              ) : (
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
                </svg>
              )}
            </button>
          </div>
        </div>

        {/* Mobile menu dropdown */}
        {mobileMenuOpen && (
          <div className="lg:hidden py-4 border-t border-slate-100 space-y-2">
            {navLinks.map((link) => (
              <button
                key={link.name}
                onClick={() => {
                  setActiveTab(link.name);
                  setMobileMenuOpen(false);
                }}
                className={`block w-full text-left px-3 py-2 rounded-lg text-sm font-semibold ${
                  activeTab === link.name
                    ? "bg-blue-50 text-blue-600"
                    : "text-slate-700 hover:bg-slate-50"
                }`}
              >
                {link.name}
              </button>
            ))}
            <div className="flex gap-2 pt-3 border-t border-slate-100 px-3">
              <button className="flex-1 py-2 text-sm font-semibold border border-slate-200 rounded-lg text-slate-700">
                Login
              </button>
              <button className="flex-1 py-2 text-sm font-semibold bg-blue-600 text-white rounded-lg">
                Sign Up
              </button>
            </div>
          </div>
        )}
      </div>
    </header>
  );
}
