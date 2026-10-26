"use client";

import { useEffect, useRef, useState } from "react";
import { ArrowUpRight, BadgeCheck, Building2, Check, ChevronDown, Filter, Landmark, Search, ShieldCheck, TrainFront, University, X } from "lucide-react";

interface SearchAndCategoryBarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  selectedCategoryPills: string[];
  onToggleCategoryPill: (pill: string) => void;
  activeFilterCount: number;
  onToggleFiltersModal: () => void;
  sortBy: "latest" | "deadline" | "posts";
  onSortChange: (sort: "latest" | "deadline" | "posts") => void;
  onSelectTrendingSearch?: (query: string) => void;
  isFiltersOpen?: boolean;
  isMobileFiltersOpen?: boolean;
}

const SUGGESTIONS = [
  { text: "SSC CGL", tag: "Staff Selection Commission" },
  { text: "UPSC Civil Services", tag: "IAS, IPS and IFS" },
  { text: "Railway RRB NTPC", tag: "Indian Railways" },
  { text: "IBPS PO", tag: "Banking" },
  { text: "10th Pass Govt Jobs", tag: "Matriculation" },
  { text: "Graduate Govt Jobs", tag: "Degree required" },
];

const CATEGORIES = [
  { label: "All Jobs", icon: BadgeCheck },
  { label: "SSC", icon: Building2 },
  { label: "UPSC", icon: Landmark },
  { label: "Railway", icon: TrainFront },
  { label: "Bank", icon: University },
  { label: "State Govt", icon: ShieldCheck },
];

const SORT_OPTIONS = [
  { value: "latest", label: "Latest First" },
  { value: "deadline", label: "Closing Soon" },
  { value: "posts", label: "Highest Posts" },
] as const;

export default function SearchAndCategoryBar({ searchQuery, onSearchChange, selectedCategoryPills, onToggleCategoryPill, activeFilterCount, onToggleFiltersModal, sortBy, onSortChange, onSelectTrendingSearch, isFiltersOpen = false, isMobileFiltersOpen = false }: SearchAndCategoryBarProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const sortRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [showSortMenu, setShowSortMenu] = useState(false);
  const [isDesktopViewport, setIsDesktopViewport] = useState(true);
  const filtersAreOpen = isDesktopViewport ? isFiltersOpen : isMobileFiltersOpen;
  const filteredSuggestions = searchQuery.trim()
    ? SUGGESTIONS.filter((item) => `${item.text} ${item.tag}`.toLowerCase().includes(searchQuery.toLowerCase()))
    : SUGGESTIONS;

  useEffect(() => {
    const media = window.matchMedia("(min-width: 1024px)");
    const updateViewport = () => setIsDesktopViewport(media.matches);
    updateViewport();
    media.addEventListener("change", updateViewport);
    return () => media.removeEventListener("change", updateViewport);
  }, []);

  useEffect(() => {
    const close = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) setShowSuggestions(false);
      if (sortRef.current && !sortRef.current.contains(event.target as Node)) setShowSortMenu(false);
    };
    const closeOnScroll = () => {
      setShowSuggestions(false);
      setShowSortMenu(false);
    };
    document.addEventListener("mousedown", close);
    document.addEventListener("scroll", closeOnScroll, true);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("scroll", closeOnScroll, true);
    };
  }, []);

  useEffect(() => {
    const shortcut = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowSuggestions(false);
        setShowSortMenu(false);
      }
      if (event.key === "/" && document.activeElement !== inputRef.current) {
        event.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, []);

  const choose = (text: string) => {
    (onSelectTrendingSearch || onSearchChange)(text);
    setShowSuggestions(false);
  };

  return (
    <section className="rounded-xl border border-slate-200 bg-white p-3 shadow-sm sm:p-4">
      <div ref={containerRef} className="relative">
        <div className="flex h-14 items-center rounded-xl border border-slate-300 bg-slate-50/70 p-1 transition-colors focus-within:border-teal-600 focus-within:bg-white">
          <Search className="ml-3 h-5 w-5 flex-none text-slate-400" />
          <input
            ref={inputRef}
            value={searchQuery}
            onChange={(event) => { onSearchChange(event.target.value); setShowSuggestions(true); }}
            onFocus={() => setShowSuggestions(true)}
            placeholder="Search job title, exam, department or qualification"
            className="min-w-0 flex-1 bg-transparent px-3 text-sm font-medium text-slate-950 outline-none focus-visible:outline-none placeholder:font-normal placeholder:text-slate-400 sm:text-base"
            aria-label="Search government opportunities"
          />
          {searchQuery && <button onClick={() => onSearchChange("")} className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-700" aria-label="Clear search"><X className="h-4 w-4" /></button>}
        </div>

        {showSuggestions && (
          <div className="absolute left-0 right-0 top-full z-50 mt-2 overflow-hidden rounded-xl border border-slate-200 bg-white py-2 text-slate-900 shadow-2xl shadow-slate-900/10">
            <div className="px-4 pb-2 pt-1 text-xs font-bold uppercase tracking-wider text-slate-400">{searchQuery ? "Suggested searches" : "Popular searches"}</div>
            {(filteredSuggestions.length ? filteredSuggestions : [{ text: searchQuery, tag: "Search all opportunities" }]).map((item) => (
              <button key={item.text} onClick={() => choose(item.text)} className="flex w-full items-center justify-between gap-4 px-4 py-2.5 text-left hover:bg-slate-50">
                <span><span className="block text-sm font-semibold text-slate-900">{item.text}</span><span className="block text-xs text-slate-500">{item.tag}</span></span>
                <ArrowUpRight className="h-4 w-4 flex-none text-slate-400" />
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="mt-2.5 flex min-w-0 flex-wrap items-center gap-1.5">
        {CATEGORIES.map(({ label, icon: Icon }) => {
          const selected = label === "All Jobs" ? selectedCategoryPills.length === 0 : selectedCategoryPills.includes(label);
          return (
            <button
              key={label}
              onClick={() => onToggleCategoryPill(label)}
              aria-pressed={selected}
              className={`inline-flex min-h-9 flex-none items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-semibold transition-all ${selected ? "border-teal-300 bg-teal-50 text-teal-800" : "border-transparent bg-slate-50 text-slate-600 hover:border-slate-200 hover:bg-white hover:text-slate-950"}`}
            >
              <Icon className="h-3.5 w-3.5" />{label}
              {selected && label !== "All Jobs" && <X className="ml-0.5 h-3.5 w-3.5" aria-hidden="true" />}
            </button>
          );
        })}
        <button
          onClick={onToggleFiltersModal}
          aria-pressed={filtersAreOpen}
          className={`inline-flex min-h-9 flex-none items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-semibold transition-colors ${filtersAreOpen ? "border-teal-300 bg-teal-50 text-teal-800" : "border-slate-200 bg-white text-slate-700 hover:border-teal-300 hover:text-teal-800"}`}
        >
          <Filter className="h-3.5 w-3.5" />More filters{activeFilterCount > 0 && <span className="rounded-full bg-teal-700 px-1.5 text-[10px] font-extrabold text-white">{activeFilterCount}</span>}
        </button>
        <div ref={sortRef} className="relative ml-auto flex-none">
          <button
            type="button"
            onClick={() => {
              setShowSuggestions(false);
              setShowSortMenu((open) => !open);
            }}
            className={`inline-flex min-h-9 items-center gap-2 rounded-lg border bg-white px-3 text-xs transition-colors ${showSortMenu ? "border-slate-400 text-slate-950" : "border-slate-200 text-slate-700 hover:border-slate-300"}`}
            aria-haspopup="listbox"
            aria-expanded={showSortMenu}
            aria-label="Sort jobs"
          >
            <span className="hidden font-medium text-slate-500 sm:inline">Sort by</span>
            <span className="font-bold">{SORT_OPTIONS.find((option) => option.value === sortBy)?.label}</span>
            <ChevronDown className={`h-3.5 w-3.5 text-slate-400 transition-transform ${showSortMenu ? "rotate-180" : ""}`} />
          </button>

          {showSortMenu && (
            <div className="absolute right-0 top-full z-50 mt-1.5 w-44 overflow-hidden rounded-lg border border-slate-200 bg-white p-1 shadow-lg shadow-slate-900/10" role="listbox" aria-label="Sort jobs">
              {SORT_OPTIONS.map((option) => {
                const selected = option.value === sortBy;
                return (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => {
                      onSortChange(option.value);
                      setShowSortMenu(false);
                    }}
                    className={`flex min-h-9 w-full items-center justify-between gap-3 rounded-md px-2.5 py-1.5 text-left text-xs font-semibold transition-colors ${selected ? "bg-slate-100 text-slate-950" : "text-slate-600 hover:bg-slate-50 hover:text-slate-950"}`}
                    role="option"
                    aria-selected={selected}
                  >
                    <span>{option.label}</span>
                    {selected && <Check className="h-3.5 w-3.5 text-teal-700" aria-hidden="true" />}
                  </button>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
