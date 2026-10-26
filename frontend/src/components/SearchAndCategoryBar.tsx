"use client";

import { useState, useRef, useEffect } from "react";

interface SearchAndCategoryBarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  selectedCategoryPill: string;
  onSelectCategoryPill: (pill: string) => void;
  activeFilterCount: number;
  onToggleFiltersModal: () => void;
  onSelectTrendingSearch?: (query: string) => void;
  isFiltersOpen?: boolean;
}

const ALL_SUGGESTIONS = [
  { text: "SSC CGL", tag: "Staff Selection Commission", icon: "🏢" },
  { text: "SSC CHSL", tag: "12th Pass", icon: "🏢" },
  { text: "UPSC Civil Services", tag: "IAS / IPS / IFS", icon: "🏛️" },
  { text: "UPSC NDA", tag: "Defence Officer", icon: "🛡️" },
  { text: "ISRO Scientist / Engineer", tag: "Space Research", icon: "🚀" },
  { text: "IBPS PO", tag: "Banking Probationary Officer", icon: "🏦" },
  { text: "SBI Clerk", tag: "Junior Associate", icon: "🏦" },
  { text: "Railway RRB NTPC", tag: "Indian Railways", icon: "🚂" },
  { text: "Railway Group D", tag: "10th Pass", icon: "🚂" },
  { text: "UPPSC Teacher", tag: "Teaching & Faculty", icon: "📚" },
  { text: "DSSSB Delhi", tag: "Delhi Subordinate", icon: "🏛️" },
  { text: "BPSC Bihar", tag: "State Commission", icon: "🏛️" },
  { text: "Police Constable", tag: "Police Department", icon: "👮" },
  { text: "10th Pass Govt Jobs", tag: "Matriculation", icon: "📄" },
  { text: "12th Pass Govt Jobs", tag: "Intermediate", icon: "📜" },
  { text: "Graduate Govt Jobs", tag: "Degree Required", icon: "🎓" },
  { text: "Engineering Govt Jobs", tag: "B.Tech / Diploma", icon: "🔬" },
  { text: "Admit Card", tag: "Hall Tickets", icon: "🎫" },
  { text: "Result", tag: "Merit Lists & Cutoffs", icon: "🏆" },
];

const CATEGORY_PILLS = [
  { label: "All Jobs", icon: "⚡" },
  { label: "UPSC", icon: "🏛️" },
  { label: "SSC", icon: "🏢" },
  { label: "Railway", icon: "🚂" },
  { label: "Bank", icon: "🏦" },
  { label: "ISRO", icon: "🚀" },
  { label: "State Govt", icon: "🇮🇳" },
  { label: "Defence", icon: "🛡️" },
  { label: "Police", icon: "👮" },
  { label: "Teaching", icon: "📚" },
  { label: "Graduate", icon: "🎓" },
  { label: "10th Pass", icon: "📄" },
  { label: "12th Pass", icon: "📜" },
  { label: "Engineering", icon: "🔬" },
  { label: "Medical", icon: "🩺" },
  { label: "Admit Card", icon: "🎫" },
  { label: "Result", icon: "🏆" },
];

const TRENDING_SEARCHES = [
  "SSC CGL",
  "ISRO Scientist",
  "UPSC NDA",
  "UPPSC",
  "Teacher",
  "Bank PO",
  "10th Pass",
];

export default function SearchAndCategoryBar({
  searchQuery,
  onSearchChange,
  selectedCategoryPill,
  onSelectCategoryPill,
  activeFilterCount,
  onToggleFiltersModal,
  onSelectTrendingSearch,
  isFiltersOpen = true,
}: SearchAndCategoryBarProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [showSuggestions, setShowSuggestions] = useState(false);

  // Filter suggestions based on searchQuery
  const filteredSuggestions = searchQuery.trim()
    ? ALL_SUGGESTIONS.filter(
        (item) =>
          item.text.toLowerCase().includes(searchQuery.toLowerCase()) ||
          item.tag.toLowerCase().includes(searchQuery.toLowerCase())
      ).slice(0, 7)
    : ALL_SUGGESTIONS.slice(0, 6);

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setShowSuggestions(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Keyboard shortcut: Pressing "/" or "Ctrl/Cmd + K" focuses search bar, Escape closes suggestions
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setShowSuggestions(false);
        return;
      }
      if (
        (e.key === "/" || (e.key === "k" && (e.metaKey || e.ctrlKey))) &&
        document.activeElement !== inputRef.current &&
        !(document.activeElement instanceof HTMLInputElement) &&
        !(document.activeElement instanceof HTMLTextAreaElement)
      ) {
        e.preventDefault();
        inputRef.current?.focus();
        setShowSuggestions(true);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const handleSelectSuggestion = (text: string) => {
    setShowSuggestions(false);
    if (onSelectTrendingSearch) {
      onSelectTrendingSearch(text);
    } else {
      onSearchChange(text);
    }
  };

  return (
    <div className="space-y-2">
      {/* Sleek, Compact Search Bar with Suggestions Dropdown */}
      <div ref={containerRef} className="relative">
        <div className="bg-white rounded-xl border border-slate-200/90 hover:border-slate-300 focus-within:border-blue-500 focus-within:ring-2 focus-within:ring-blue-100 shadow-2xs transition-all flex items-center gap-2 px-3 py-1.5 h-11">
          {/* Search Icon */}
          <svg
            className="w-4 h-4 text-slate-400 flex-shrink-0"
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

          {/* Text input */}
          <input
            ref={inputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => {
              onSearchChange(e.target.value);
              setShowSuggestions(true);
            }}
            onFocus={() => setShowSuggestions(true)}
            placeholder="Search notifications by title, department, exam or post..."
            className="w-full text-xs sm:text-sm text-slate-800 placeholder:text-slate-400 focus:outline-none bg-transparent"
          />

          {/* Clear or Keyboard Shortcut Indicator */}
          {searchQuery ? (
            <button
              onClick={() => {
                onSearchChange("");
                setShowSuggestions(false);
              }}
              className="text-slate-400 hover:text-slate-700 font-bold text-xs p-1 rounded-md hover:bg-slate-100 transition-colors flex-shrink-0"
              title="Clear search"
            >
              ✕
            </button>
          ) : (
            <kbd className="hidden sm:inline-flex items-center text-[10px] font-mono text-slate-400 bg-slate-100 border border-slate-200 px-1.5 py-0.5 rounded select-none flex-shrink-0">
              /
            </kbd>
          )}

          {/* Searchbar Filter Toggle Button - Visible on Desktop & Mobile */}
          <button
            type="button"
            onClick={onToggleFiltersModal}
            className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 text-xs font-bold rounded-lg transition-all border flex-shrink-0 ${
              isFiltersOpen || activeFilterCount > 0
                ? "bg-blue-50 text-blue-700 border-blue-300 shadow-2xs hover:bg-blue-100"
                : "bg-slate-50 text-slate-700 border-slate-200 hover:bg-slate-100"
            }`}
            title={isFiltersOpen ? "Hide filters sidebar" : "Show filters sidebar"}
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
            </svg>
            <span className="hidden xs:inline">{isFiltersOpen ? "Filters ▾" : "Filters ▸"}</span>
            <span className="xs:hidden">Filters</span>
            {activeFilterCount > 0 && (
              <span className="w-4 h-4 rounded-full bg-blue-600 text-white text-[10px] flex items-center justify-center font-bold">
                {activeFilterCount}
              </span>
            )}
          </button>
        </div>

        {/* Search Suggestions Dropdown Popover */}
        {showSuggestions && (
          <div className="absolute left-0 right-0 top-full mt-1.5 bg-white rounded-xl border border-slate-200 shadow-xl z-50 overflow-hidden text-xs divide-y divide-slate-100 animate-in fade-in slide-in-from-top-1 duration-150">
            <div className="px-3 py-1.5 bg-slate-50/90 flex items-center justify-between text-[11px] text-slate-500 font-semibold">
              <span className="flex items-center gap-1">
                <span>🔍</span> {searchQuery.trim() ? "Search Suggestions" : "Popular Searches"}
              </span>
              <span className="text-[10px] text-slate-400">Esc to close</span>
            </div>

            <div className="max-h-60 overflow-y-auto divide-y divide-slate-50">
              {filteredSuggestions.length > 0 ? (
                filteredSuggestions.map((item) => (
                  <button
                    key={item.text}
                    type="button"
                    onClick={() => handleSelectSuggestion(item.text)}
                    className="w-full px-3.5 py-2 text-left flex items-center justify-between hover:bg-blue-50/80 transition-colors group cursor-pointer"
                  >
                    <div className="flex items-center gap-2.5">
                      <span className="text-sm">{item.icon}</span>
                      <span className="font-semibold text-slate-800 group-hover:text-blue-700">
                        {item.text}
                      </span>
                    </div>
                    <span className="text-[10px] text-slate-400 group-hover:text-blue-600 font-medium">
                      {item.tag} ↗
                    </span>
                  </button>
                ))
              ) : (
                <button
                  type="button"
                  onClick={() => handleSelectSuggestion(searchQuery)}
                  className="w-full px-3.5 py-2 text-left flex items-center justify-between hover:bg-blue-50/80 transition-colors group cursor-pointer"
                >
                  <div className="flex items-center gap-2">
                    <span>🔍</span>
                    <span className="font-semibold text-blue-700">
                      Search for &quot;{searchQuery}&quot;
                    </span>
                  </div>
                  <span className="text-[10px] text-blue-500">Press Enter ↵</span>
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Compact Trending Quick Searches */}
      <div className="flex items-center gap-1.5 text-xs overflow-x-auto scrollbar-none no-scrollbar py-0.5">
        <span className="text-[10px] font-black uppercase text-slate-400 tracking-wider flex items-center gap-1 flex-shrink-0">
          <span>🔥</span> Popular:
        </span>
        {TRENDING_SEARCHES.map((query) => (
          <button
            key={query}
            onClick={() => {
              if (onSelectTrendingSearch) {
                onSelectTrendingSearch(query);
              } else {
                onSearchChange(query);
              }
            }}
            className="px-2 py-0.5 rounded-md bg-slate-100 hover:bg-blue-50 hover:text-blue-700 text-slate-600 border border-slate-200/70 font-semibold whitespace-nowrap transition-colors text-[10px]"
          >
            {query}
          </button>
        ))}
      </div>

      {/* Compact Category Pills */}
      <div className="flex items-center gap-1.5 overflow-x-auto pb-0.5 scrollbar-none no-scrollbar">
        {CATEGORY_PILLS.map((pill) => {
          const isSelected = selectedCategoryPill === pill.label;
          return (
            <button
              key={pill.label}
              onClick={() => onSelectCategoryPill(pill.label)}
              className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-bold whitespace-nowrap transition-all border ${
                isSelected
                  ? "bg-blue-600 border-blue-600 text-white shadow-2xs"
                  : "bg-white border-slate-200/90 text-slate-700 hover:border-slate-300 hover:bg-slate-50"
              }`}
            >
              <span className="text-xs">{pill.icon}</span>
              <span>{pill.label}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
