"use client";

import { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { JobItem } from "@/types";
import { fetchPublishedJobs } from "@/lib/api";
import { INITIAL_PUBLISHED_JOBS } from "@/lib/mockData";
import { getSavedJobIds, toggleSavedJobId } from "@/lib/formatters";
import SearchAndCategoryBar from "@/components/SearchAndCategoryBar";
import SidebarFilters from "@/components/SidebarFilters";
import JobTableView from "@/components/JobTableView";
import JobCard from "@/components/JobCard";
import JobMobileCards from "@/components/JobMobileCards";
import Pagination from "@/components/Pagination";
import JobDetailsModal from "@/components/JobDetailsModal";

export default function CandidateJobsPage() {
  const [jobs, setJobs] = useState<JobItem[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [selectedCategoryPill, setSelectedCategoryPill] = useState("All Jobs");

  // Sidebar Filters State
  const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
  const [selectedQualifications, setSelectedQualifications] = useState<string[]>([]);
  const [selectedStates, setSelectedStates] = useState<string[]>([]);
  const [selectedJobTypes, setSelectedJobTypes] = useState<string[]>([]);
  const [sortBy, setSortBy] = useState<"latest" | "deadline" | "posts">("latest");

  // View Mode: Grid (Cards) vs Table
  const [viewMode, setViewMode] = useState<"table" | "grid">("table");

  // Bookmarks state
  const [savedJobIds, setSavedJobIds] = useState<number[]>([]);
  const [showOnlySaved, setShowOnlySaved] = useState(false);

  // Toast feedback state
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const PAGE_SIZE = 12;

  // Selected Job for Detail Modal
  const [selectedJob, setSelectedJob] = useState<JobItem | null>(null);

  // Mobile Filters Modal State
  const [isMobileFilterOpen, setIsMobileFilterOpen] = useState(false);
  const [isConnectedToBackend, setIsConnectedToBackend] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Initialize saved bookmarks from localStorage
  useEffect(() => {
    setSavedJobIds(getSavedJobIds());
  }, []);

  const showToast = useCallback((msg: string) => {
    setToastMessage(msg);
    setTimeout(() => {
      setToastMessage((prev) => (prev === msg ? null : prev));
    }, 2800);
  }, []);

  const handleToggleBookmark = useCallback(
    (id: number) => {
      const isSaved = toggleSavedJobId(id);
      setSavedJobIds(getSavedJobIds());
      showToast(isSaved ? "⭐ Saved to your shortlisted jobs!" : "Removed from shortlisted jobs");
    },
    [showToast]
  );

  const handleShareWhatsApp = useCallback((job: JobItem) => {
    const d = job.details;
    const text = `📢 *${d.title}*\n🏢 *Organization:* ${d.organization}\n👤 *Vacancies:* ${
      d.total_vacancies > 0 ? d.total_vacancies.toLocaleString() : "Refer PDF"
    }\n⏰ *Deadline:* ${d.application_end || "Refer notification"}\n🔗 *Apply:* ${
      d.official_apply_url || d.official_notice_url || "https://naukrisetu.gov.in"
    }\n\nShared via NaukriSetu`;
    window.open(`https://api.whatsapp.com/send?text=${encodeURIComponent(text)}`, "_blank");
  }, []);

  // Debounce search query to prevent excessive queries while typing
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Reset page to 1 whenever any filter or search changes
  useEffect(() => {
    setCurrentPage(1);
  }, [
    debouncedQuery,
    selectedCategoryPill,
    selectedCategories,
    selectedQualifications,
    selectedStates,
    selectedJobTypes,
    sortBy,
    showOnlySaved,
  ]);

  // Fetch data directly from backend
  useEffect(() => {
    let cancelled = false;

    async function loadFilteredData() {
      setIsLoading(true);
      try {
        const cats = [...selectedCategories];
        const quals = [...selectedQualifications];
        const types = [...selectedJobTypes];

        if (selectedCategoryPill !== "All Jobs") {
          const pill = selectedCategoryPill;
          if (
            pill === "10th Pass" ||
            pill === "12th Pass" ||
            pill === "Graduate" ||
            pill === "Engineering"
          ) {
            quals.push(pill);
          } else if (
            pill === "Recruitment" ||
            pill === "Admit Card" ||
            pill === "Result" ||
            pill === "Answer Key"
          ) {
            types.push(pill);
          } else {
            cats.push(pill);
          }
        }

        const res = await fetchPublishedJobs({
          q: debouncedQuery.trim() || undefined,
          category: cats.length > 0 ? cats : undefined,
          qualification: quals.length > 0 ? quals : undefined,
          type: types.length > 0 ? types : undefined,
          state: selectedStates.length > 0 ? selectedStates : undefined,
          sort: sortBy,
          limit: showOnlySaved ? 100 : PAGE_SIZE,
          offset: showOnlySaved ? 0 : (currentPage - 1) * PAGE_SIZE,
        });

        if (!cancelled && res) {
          let list = res.jobs || [];
          let count = res.total !== undefined ? res.total : (res.count || 0);

          if (showOnlySaved) {
            const currentSaved = getSavedJobIds();
            list = list.filter((j) => currentSaved.includes(j.id));
            count = list.length;
          }

          setJobs(list);
          setTotalCount(count);
          setIsConnectedToBackend(true);
        }
      } catch (err) {
        console.warn("Error fetching notifications, showing local backup:", err);
        if (!cancelled) {
          setIsConnectedToBackend(false);
          let list = INITIAL_PUBLISHED_JOBS;
          if (showOnlySaved) {
            const currentSaved = getSavedJobIds();
            list = list.filter((j) => currentSaved.includes(j.id));
          }
          setJobs(list.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE));
          setTotalCount(list.length);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    loadFilteredData();

    return () => {
      cancelled = true;
    };
  }, [
    debouncedQuery,
    selectedCategoryPill,
    selectedCategories,
    selectedQualifications,
    selectedStates,
    selectedJobTypes,
    sortBy,
    currentPage,
    showOnlySaved,
  ]);

  // Compute active filters count
  const activeFilterCount =
    (selectedCategoryPill !== "All Jobs" ? 1 : 0) +
    selectedCategories.length +
    selectedQualifications.length +
    selectedStates.length +
    selectedJobTypes.length +
    (searchQuery.trim() ? 1 : 0) +
    (showOnlySaved ? 1 : 0);

  // Clear all filters
  const handleClearAll = () => {
    setSelectedCategories([]);
    setSelectedQualifications([]);
    setSelectedStates([]);
    setSelectedJobTypes([]);
    setSelectedCategoryPill("All Jobs");
    setSearchQuery("");
    setShowOnlySaved(false);
    setCurrentPage(1);
  };

  const totalPages = Math.ceil(totalCount / PAGE_SIZE) || 1;

  return (
    <div className="space-y-4">
      {/* Toast Notification Popup */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 bg-slate-900 text-white text-xs font-semibold px-4 py-2.5 rounded-xl shadow-2xl border border-slate-700 animate-in fade-in slide-in-from-bottom duration-200 flex items-center gap-2">
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Slim & Compact Header Band (Takes minimal screen space) */}
      <div className="bg-gradient-to-r from-slate-900 via-slate-800 to-blue-900 text-white px-4 py-2.5 sm:px-5 sm:py-3 rounded-2xl shadow-sm border border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
        <div className="flex items-center gap-2.5">
          <span className="flex-shrink-0 w-7 h-7 rounded-lg bg-blue-600/30 border border-blue-400/40 flex items-center justify-center text-sm">
            🇮🇳
          </span>
          <div>
            <h1 className="text-sm sm:text-base font-extrabold tracking-tight leading-tight">
              Official Government Job Notifications & Updates
            </h1>
            <p className="text-[11px] text-slate-300">
              Verified notices from UPSC, SSC, Railway, Banking, Defence & State Commissions
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2 text-xs self-start sm:self-center">
          <span className="px-2.5 py-0.5 rounded-full bg-white/10 text-slate-200 border border-white/15 text-[11px] font-semibold whitespace-nowrap">
            ✓ 100+ Live Notices
          </span>
          <span className="px-2.5 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 text-[11px] font-bold whitespace-nowrap">
            Updated Today
          </span>
        </div>
      </div>

      {/* Connection Notice (Only shown if server is offline, clean non-technical text) */}
      {isConnectedToBackend === false && (
        <div className="bg-amber-50 border border-amber-200 text-amber-900 px-3.5 py-2 rounded-xl text-xs flex items-center justify-between shadow-2xs">
          <div className="flex items-center gap-2">
            <span>ℹ️</span>
            <span>
              Showing preloaded official notifications. Connecting to live updates server...
            </span>
          </div>
          <button
            onClick={() => setCurrentPage(1)}
            className="font-bold text-blue-700 hover:underline ml-2"
          >
            Refresh ↻
          </button>
        </div>
      )}

      {/* Search & Category Pills */}
      <SearchAndCategoryBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        selectedCategoryPill={selectedCategoryPill}
        onSelectCategoryPill={setSelectedCategoryPill}
        activeFilterCount={activeFilterCount}
        onToggleFiltersModal={() => setIsMobileFilterOpen(true)}
        onSelectTrendingSearch={(t) => setSearchQuery(t)}
      />

      {/* Breadcrumb & Saved Shortlist Button */}
      <div className="flex flex-wrap items-center justify-between gap-2 pt-0.5">
        <nav className="flex items-center gap-1.5 text-xs font-medium text-slate-500">
          <Link href="/" className="hover:text-blue-600">
            Home
          </Link>
          <span>›</span>
          <span className="text-slate-800 font-semibold">Government Jobs</span>
        </nav>

        <div className="flex items-center gap-2">
          {/* Saved Bookmarks Toggle Button */}
          <button
            onClick={() => setShowOnlySaved(!showOnlySaved)}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-bold transition-all border ${
              showOnlySaved
                ? "bg-amber-50 text-amber-700 border-amber-300 shadow-2xs scale-102"
                : "bg-white hover:bg-slate-50 text-slate-600 border-slate-200"
            }`}
          >
            <span>{showOnlySaved ? "⭐" : "☆"}</span>
            <span>Saved Jobs ({savedJobIds.length})</span>
          </button>

          {/* Admin Review Portal */}
          <Link
            href="/admin"
            className="inline-flex items-center gap-1 text-xs font-bold px-3 py-1.5 rounded-xl bg-slate-100 hover:bg-slate-200 text-slate-700 border border-slate-300 transition-colors"
          >
            <span>🛡️ Admin Portal</span>
          </Link>
        </div>
      </div>

      {/* Active Filter Chips Bar (Student-friendly non-technical wording) */}
      {activeFilterCount > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 bg-slate-100/80 p-2.5 rounded-xl border border-slate-200 text-xs">
          <span className="font-bold text-slate-600">Applied Filters:</span>

          {showOnlySaved && (
            <span className="inline-flex items-center gap-1.5 bg-amber-50 border border-amber-300 text-amber-800 px-2 py-0.5 rounded-full font-bold shadow-2xs">
              ⭐ Shortlisted Jobs Only
              <button
                onClick={() => setShowOnlySaved(false)}
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          )}

          {searchQuery.trim() && (
            <span className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2 py-0.5 rounded-full font-medium shadow-2xs">
              Search: &quot;{searchQuery}&quot;
              <button
                onClick={() => setSearchQuery("")}
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          )}

          {selectedCategoryPill !== "All Jobs" && (
            <span className="inline-flex items-center gap-1.5 bg-blue-50 border border-blue-200 text-blue-700 px-2 py-0.5 rounded-full font-bold shadow-2xs">
              {selectedCategoryPill}
              <button
                onClick={() => setSelectedCategoryPill("All Jobs")}
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          )}

          {selectedCategories.map((cat) => (
            <span
              key={cat}
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2 py-0.5 rounded-full font-medium shadow-2xs"
            >
              Sector: {cat}
              <button
                onClick={() =>
                  setSelectedCategories((prev) => prev.filter((x) => x !== cat))
                }
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          ))}

          {selectedJobTypes.map((type) => (
            <span
              key={type}
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2 py-0.5 rounded-full font-medium shadow-2xs"
            >
              Type: {type}
              <button
                onClick={() =>
                  setSelectedJobTypes((prev) => prev.filter((x) => x !== type))
                }
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          ))}

          {selectedQualifications.map((qual) => (
            <span
              key={qual}
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2 py-0.5 rounded-full font-medium shadow-2xs"
            >
              Qualification: {qual}
              <button
                onClick={() =>
                  setSelectedQualifications((prev) => prev.filter((x) => x !== qual))
                }
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          ))}

          {selectedStates.map((st) => (
            <span
              key={st}
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2 py-0.5 rounded-full font-medium shadow-2xs"
            >
              State: {st}
              <button
                onClick={() =>
                  setSelectedStates((prev) => prev.filter((x) => x !== st))
                }
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          ))}

          <button
            onClick={handleClearAll}
            className="text-xs font-bold text-red-600 hover:text-red-700 hover:underline ml-auto"
          >
            Clear All ({activeFilterCount})
          </button>
        </div>
      )}

      {/* Main 2-Column Content Layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Sidebar Filter (Desktop) */}
        <aside className="hidden lg:block lg:col-span-3 sticky top-28">
          <SidebarFilters
            selectedCategories={selectedCategories}
            onChangeCategories={setSelectedCategories}
            selectedQualifications={selectedQualifications}
            onChangeQualifications={setSelectedQualifications}
            selectedStates={selectedStates}
            onChangeStates={setSelectedStates}
            selectedJobTypes={selectedJobTypes}
            onChangeJobTypes={setSelectedJobTypes}
            onClearAll={handleClearAll}
            activeCount={activeFilterCount}
          />
        </aside>

        {/* Right Main Content */}
        <div className="lg:col-span-9 space-y-3.5">
          {/* Section Header, View Mode Switcher, & Sort Controls (Compact padding) */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 bg-white px-4 py-2.5 rounded-2xl border border-slate-200 shadow-2xs">
            <div>
              <h2 className="text-sm sm:text-base font-extrabold text-slate-900 tracking-tight flex items-center gap-2">
                <span>Government Job Notifications</span>
                {isLoading && (
                  <span className="w-2 h-2 rounded-full bg-blue-600 animate-ping" />
                )}
              </h2>
              <p className="text-xs text-slate-500">
                Showing {jobs.length} of <strong className="text-slate-800">{totalCount}</strong> verified notices
              </p>
            </div>

            <div className="flex items-center gap-2 self-end sm:self-center text-xs">
              {/* View Mode Toggle: Table vs Grid */}
              <div className="hidden sm:flex items-center bg-slate-100 p-0.5 rounded-lg border border-slate-200">
                <button
                  onClick={() => setViewMode("table")}
                  className={`px-2.5 py-1 rounded-md font-bold flex items-center gap-1 transition-all ${
                    viewMode === "table"
                      ? "bg-white text-slate-900 shadow-2xs"
                      : "text-slate-500 hover:text-slate-900"
                  }`}
                  title="Table View"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
                  </svg>
                  <span>Table</span>
                </button>
                <button
                  onClick={() => setViewMode("grid")}
                  className={`px-2.5 py-1 rounded-md font-bold flex items-center gap-1 transition-all ${
                    viewMode === "grid"
                      ? "bg-white text-slate-900 shadow-2xs"
                      : "text-slate-500 hover:text-slate-900"
                  }`}
                  title="Grid Cards View"
                >
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
                  </svg>
                  <span>Cards</span>
                </button>
              </div>

              {/* Sort By Dropdown */}
              <div className="flex items-center gap-1.5 bg-slate-50 border border-slate-200 px-2.5 py-1 rounded-lg shadow-2xs">
                <span className="text-slate-500 font-medium">Sort:</span>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as any)}
                  className="bg-transparent font-bold text-slate-800 focus:outline-none cursor-pointer"
                >
                  <option value="latest">Latest First</option>
                  <option value="deadline">Closing Soon</option>
                  <option value="posts">Highest Posts</option>
                </select>
              </div>
            </div>
          </div>

          {/* Main Listings Display */}
          {isLoading && jobs.length === 0 ? (
            <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center space-y-3">
              <div className="animate-spin h-8 w-8 border-3 border-blue-600 border-t-transparent rounded-full mx-auto" />
              <p className="text-slate-700 font-bold text-sm">
                Loading notifications...
              </p>
            </div>
          ) : jobs.length > 0 ? (
            <>
              {/* Desktop View (Table vs Grid Cards) */}
              <div className="hidden sm:block">
                {viewMode === "table" ? (
                  <JobTableView
                    jobs={jobs}
                    savedJobIds={savedJobIds}
                    onToggleBookmark={handleToggleBookmark}
                    onSelectJob={(job) => setSelectedJob(job)}
                    onShareWhatsApp={handleShareWhatsApp}
                  />
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3.5">
                    {jobs.map((job) => (
                      <JobCard
                        key={job.id}
                        job={job}
                        isBookmarked={savedJobIds.includes(job.id)}
                        onToggleBookmark={handleToggleBookmark}
                        onSelectJob={(j) => setSelectedJob(j)}
                        onShareWhatsApp={handleShareWhatsApp}
                      />
                    ))}
                  </div>
                )}
              </div>

              {/* Mobile View (Streamlined Cards) */}
              <div className="block sm:hidden">
                <JobMobileCards
                  jobs={jobs}
                  savedJobIds={savedJobIds}
                  onToggleBookmark={handleToggleBookmark}
                  onSelectJob={(job) => setSelectedJob(job)}
                  onShareWhatsApp={handleShareWhatsApp}
                />
              </div>
            </>
          ) : (
            /* Empty State */
            <div className="bg-white rounded-2xl border border-dashed border-slate-300 p-10 text-center space-y-3 shadow-2xs">
              <div className="w-12 h-12 rounded-full bg-blue-50 text-blue-600 flex items-center justify-center text-xl mx-auto">
                🔍
              </div>
              <div className="space-y-0.5">
                <h3 className="text-base font-extrabold text-slate-800">
                  {showOnlySaved
                    ? "No Shortlisted Jobs Found"
                    : "No notifications found matching your selection"}
                </h3>
                <p className="text-xs text-slate-500 max-w-md mx-auto">
                  {showOnlySaved
                    ? "Click the star icon (★) on any notice to save it for quick review."
                    : "Try broadening your search keywords or resetting filters to view all available notices."}
                </p>
              </div>

              {/* Recommended Quick Searches */}
              <div className="pt-1 flex flex-wrap items-center justify-center gap-1.5">
                <button
                  onClick={() => setSearchQuery("UPSC")}
                  className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-blue-50 text-xs font-semibold text-slate-700"
                >
                  UPSC
                </button>
                <button
                  onClick={() => setSearchQuery("SSC")}
                  className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-blue-50 text-xs font-semibold text-slate-700"
                >
                  SSC
                </button>
                <button
                  onClick={() => setSearchQuery("ISRO")}
                  className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-blue-50 text-xs font-semibold text-slate-700"
                >
                  ISRO
                </button>
                <button
                  onClick={() => setSelectedCategoryPill("Graduate")}
                  className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-blue-50 text-xs font-semibold text-slate-700"
                >
                  Graduate Jobs
                </button>
              </div>

              <button
                onClick={handleClearAll}
                className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-xl text-xs font-bold hover:bg-blue-700 shadow-2xs transition-all"
              >
                Reset All Filters
              </button>
            </div>
          )}

          {/* Pagination */}
          {!showOnlySaved && (
            <Pagination
              currentPage={currentPage}
              totalPages={totalPages}
              onPageChange={setCurrentPage}
              totalItems={totalCount}
              pageSize={PAGE_SIZE}
            />
          )}
        </div>
      </div>

      {/* Details Modal */}
      <JobDetailsModal
        job={selectedJob}
        isBookmarked={selectedJob ? savedJobIds.includes(selectedJob.id) : false}
        onToggleBookmark={handleToggleBookmark}
        onClose={() => setSelectedJob(null)}
      />

      {/* Mobile Drawer Filter Modal */}
      {isMobileFilterOpen && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center p-0 sm:p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="bg-white rounded-t-3xl sm:rounded-2xl max-w-lg w-full max-h-[85vh] overflow-y-auto p-5 shadow-2xl space-y-4 animate-in slide-in-from-bottom duration-200">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <h3 className="font-bold text-slate-900 text-base">Filter Jobs & Vacancies</h3>
              <button
                onClick={() => setIsMobileFilterOpen(false)}
                className="w-8 h-8 rounded-full bg-slate-100 flex items-center justify-center text-slate-500 font-bold"
              >
                ✕
              </button>
            </div>

            <SidebarFilters
              selectedCategories={selectedCategories}
              onChangeCategories={setSelectedCategories}
              selectedQualifications={selectedQualifications}
              onChangeQualifications={setSelectedQualifications}
              selectedStates={selectedStates}
              onChangeStates={setSelectedStates}
              selectedJobTypes={selectedJobTypes}
              onChangeJobTypes={setSelectedJobTypes}
              onClearAll={handleClearAll}
              onApplyFilters={() => setIsMobileFilterOpen(false)}
              activeCount={activeFilterCount}
            />
          </div>
        </div>
      )}
    </div>
  );
}
