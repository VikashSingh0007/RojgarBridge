"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { BriefcaseBusiness, FileCheck2, Megaphone, Trophy } from "lucide-react";

export default function MobileBottomNav() {
  const [currentHash, setCurrentHash] = useState("#jobs");

  useEffect(() => {
    const handleHash = () => {
      const h = window.location.hash || "#jobs";
      setCurrentHash(h);
    };
    handleHash();
    window.addEventListener("hashchange", handleHash);
    return () => window.removeEventListener("hashchange", handleHash);
  }, []);

  const tabs = [
    { name: "Recruitments", href: "/#jobs", hash: "#jobs", icon: BriefcaseBusiness },
    { name: "Announcements", href: "/#announcements", hash: "#announcements", icon: Megaphone },
    { name: "Admit Card", href: "/#admit-cards", hash: "#admit-cards", icon: FileCheck2 },
    { name: "Results", href: "/#results", hash: "#results", icon: Trophy },
  ];

  return (
    <nav
      className="lg:hidden fixed bottom-0 left-0 right-0 bg-white/95 backdrop-blur-md border-t border-slate-200 z-50 px-2 py-1 shadow-lg"
      aria-label="Mobile Bottom Navigation"
    >
      <div className="flex items-center justify-around">
        {tabs.map((tab) => {
          const isActive = currentHash === tab.hash;
          const Icon = tab.icon;
          return (
            <Link
              key={tab.name}
              href={tab.href}
              className={`flex flex-col items-center justify-center py-1 px-3 rounded-lg text-[10px] font-semibold transition-colors ${
                isActive ? "text-teal-800 font-extrabold" : "text-slate-500 hover:text-slate-900"
              }`}
            >
              <Icon className={`h-4 w-4 mb-0.5 ${isActive ? "text-teal-700" : "text-slate-400"}`} />
              <span>{tab.name}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
