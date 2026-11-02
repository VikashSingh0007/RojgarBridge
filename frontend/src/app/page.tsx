"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import {
  BriefcaseBusiness,
  LoaderCircle,
  Megaphone,
  Search,
} from "lucide-react";
import { JobItem } from "@/types";
import { fetchPublishedJobs } from "@/lib/api";
import { getSavedJobIds, toggleSavedJobId } from "@/lib/formatters";
import SearchAndCategoryBar from "@/components/SearchAndCategoryBar";
import SidebarFilters from "@/components/SidebarFilters";
import JobTableView from "@/components/JobTableView";
import JobMobileCards from "@/components/JobMobileCards";
import JobDetailsModal from "@/components/JobDetailsModal";
import AnnouncementsView from "@/components/AnnouncementsView";

export default function CandidateJobsPage() {
  // Main Navigation Tab State: "jobs" (Recruitments) | "announcements" (Admit Cards, Results, etc.)
  const [activeMainTab, setActiveMainTab] = useState<"jobs" | "announcements">("jobs");
  const [announcementCategory, setAnnouncementCategory] = useState<string>("all");

  // Jobs Stream State
  const [jobs, setJobs] = useState<JobItem[]>([]);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [selectedCategoryPills, setSelectedCategoryPills] = useState<string[]>([]);

  // Announcements Stream State
  const [announcements, setAnnouncements] = useState<JobItem[]>([]);
  const [announcementsTotalCount, setAnnouncementsTotalCount] = useState<number>(0);
  const [isAnnouncementsLoading, setIsAnnouncementsLoading] = useState(false);

  // Sidebar Filters State (for Jobs)
  const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
  const [selectedQualifications, setSelectedQualifications] = useState<string[]>([]);
  const [selectedStates, setSelectedStates] = useState<string[]>([]);
  const [selectedJobTypes, setSelectedJobTypes] = useState<string[]>([]);
  const [sortBy, setSortBy] = useState<"latest" | "deadline" | "posts">("latest");

  // Bookmarks state
  const [savedJobIds, setSavedJobIds] = useState<number[]>([]);
  const [showOnlySaved, setShowOnlySaved] = useState(false);

  // Toast feedback state
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Incremental loading state
  const [currentPage, setCurrentPage] = useState(1);
  const PAGE_SIZE = 12;
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const loadCooldownRef = useRef(false);
  const loadCooldownTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const loadMoreStateRef = useRef({ hasMoreJobs: false, isLoading: true });
  const checkForMoreRef = useRef<() => void>(() => undefined);

  // Selected Job for Detail Modal
  const [selectedJob, setSelectedJob] = useState<JobItem | null>(null);
  const [lastSelectedJobId, setLastSelectedJobId] = useState<number | null>(null);

  const handleSelectJob = (job: JobItem) => {
    setSelectedJob(job);
    setLastSelectedJobId(job.id);
  };

  // Mobile & Desktop Filters State
  const [isMobileFilterOpen, setIsMobileFilterOpen] = useState(false);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isLoading, setIsLoading] = useState(true);

  const handleToggleFilters = () => {
    if (typeof window !== "undefined" && window.innerWidth < 1024) {
      setIsMobileFilterOpen((prev) => !prev);
    } else {
      setIsSidebarOpen((prev) => !prev);
    }
  };

  // Initialize saved bookmarks from localStorage
  useEffect(() => {
    setSavedJobIds(getSavedJobIds());
  }, []);

  // Listen to URL hash changes for direct tab linking
  useEffect(() => {
    const syncFromHash = () => {
      const hash = window.location.hash.toLowerCase();
      if (hash === "#announcements") {
        setActiveMainTab("announcements");
        setAnnouncementCategory("all");
      } else if (hash === "#admit-cards" || hash === "#admit-card") {
        setActiveMainTab("announcements");
        setAnnouncementCategory("admit_card");
      } else if (hash === "#results" || hash === "#result") {
        setActiveMainTab("announcements");
        setAnnouncementCategory("result");
      } else if (hash === "#saved-jobs" || hash === "#saved") {
        setActiveMainTab("jobs");
        setShowOnlySaved(true);
      } else if (hash === "#jobs") {
        setActiveMainTab("jobs");
      }
    };

    syncFromHash();
    window.addEventListener("hashchange", syncFromHash);
    return () => window.removeEventListener("hashchange", syncFromHash);
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
      showToast(isSaved ? "Job saved to your shortlist" : "Job removed from your shortlist");
    },
    [showToast]
  );

  // Debounce search query to prevent excessive queries while typing
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Reset page whenever filters change
  useEffect(() => {
    loadCooldownRef.current = false;
    if (loadCooldownTimerRef.current) clearTimeout(loadCooldownTimerRef.current);
    setCurrentPage(1);
  }, [
    debouncedQuery,
    selectedCategoryPills,
    selectedCategories,
    selectedQualifications,
    selectedStates,
    selectedJobTypes,
    sortBy,
    showOnlySaved,
    activeMainTab,
  ]);

  // Fetch Recruitment Jobs (activeMainTab === "jobs")
  useEffect(() => {
    if (activeMainTab !== "jobs") return;
    let cancelled = false;

    async function loadFilteredData() {
      setIsLoading(true);
      try {
        const cats = [...selectedCategories];
        const quals = [...selectedQualifications];
        const types = [...selectedJobTypes];

        selectedCategoryPills.forEach((pill) => {
          if (
            pill === "10th Pass" ||
            pill === "12th Pass" ||
            pill === "Graduate" ||
            pill === "Engineering"
          ) {
            if (!quals.includes(pill)) quals.push(pill);
          } else {
            if (!cats.includes(pill)) cats.push(pill);
          }
        });

        // Ensure recruitments are strictly queried for jobs tab
        const searchTypes = types.length > 0 ? types : ["recruitment"];

        const res = await fetchPublishedJobs({
          q: debouncedQuery.trim() || undefined,
          category: cats.length > 0 ? cats : undefined,
          qualification: quals.length > 0 ? quals : undefined,
          type: searchTypes,
          state: selectedStates.length > 0 ? selectedStates : undefined,
          sort: sortBy,
          limit: showOnlySaved ? 100 : PAGE_SIZE,
          offset: showOnlySaved ? 0 : (currentPage - 1) * PAGE_SIZE,
        });

        if (!cancelled && res) {
          let list = res.jobs || [];
          let count = res.total !== undefined ? res.total : res.count || 0;

          if (showOnlySaved) {
            const currentSaved = getSavedJobIds();
            list = list.filter((j) => currentSaved.includes(j.id));
            count = list.length;
          }

          setJobs((current) => {
            if (currentPage === 1) return list;
            const existingIds = new Set(current.map((item) => item.id));
            return [...current, ...list.filter((item) => !existingIds.has(item.id))];
          });
          setTotalCount(count);
        }
      } catch (err) {
        console.warn("Backend jobs query error:", err);
        if (!cancelled) {
          if (currentPage === 1) {
            setJobs([]);
            setTotalCount(0);
          }
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
          if (currentPage > 1) {
            if (loadCooldownTimerRef.current) clearTimeout(loadCooldownTimerRef.current);
            loadCooldownTimerRef.current = setTimeout(() => {
              loadCooldownRef.current = false;
              window.requestAnimationFrame(() => checkForMoreRef.current());
            }, 1000);
          }
        }
      }
    }

    loadFilteredData();

    return () => {
      cancelled = true;
    };
  }, [
    activeMainTab,
    debouncedQuery,
    selectedCategoryPills,
    selectedCategories,
    selectedQualifications,
    selectedStates,
    selectedJobTypes,
    sortBy,
    currentPage,
    showOnlySaved,
  ]);

  // Fetch Announcements (activeMainTab === "announcements")
  useEffect(() => {
    let cancelled = false;

    async function loadAnnouncements() {
      setIsAnnouncementsLoading(true);
      try {
        const types =
          announcementCategory === "all"
            ? ["admit_card", "result", "interview_schedule", "answer_key"]
            : [announcementCategory];

        const res = await fetchPublishedJobs({
          type: types,
          limit: 50,
          offset: 0,
        });

        if (!cancelled && res) {
          const list = res.jobs || [];
          const count = res.total !== undefined ? res.total : res.count || 0;
          setAnnouncements(list);
          setAnnouncementsTotalCount(count);
        }
      } catch (err) {
        console.warn("Backend announcements query error:", err);
        if (!cancelled) {
          setAnnouncements([]);
          setAnnouncementsTotalCount(0);
        }
      } finally {
        if (!cancelled) {
          setIsAnnouncementsLoading(false);
        }
      }
    }

    loadAnnouncements();

    return () => {
      cancelled = true;
    };
  }, [announcementCategory]);

  const hasMoreJobs = !showOnlySaved && jobs.length < totalCount;

  useEffect(() => {
    loadMoreStateRef.current = { hasMoreJobs, isLoading };
  }, [hasMoreJobs, isLoading]);

  useEffect(() => {
    let frameId: number | null = null;

    const checkForMore = () => {
      frameId = null;
      const sentinel = loadMoreRef.current;
      const { hasMoreJobs: canLoadMore, isLoading: loading } = loadMoreStateRef.current;
      if (!sentinel || !canLoadMore || loading || loadCooldownRef.current) return;

      const viewportHeight = window.innerHeight || document.documentElement.clientHeight;
      const distanceFromViewport = sentinel.getBoundingClientRect().top - viewportHeight;
      if (distanceFromViewport > 180) return;

      loadCooldownRef.current = true;
      setCurrentPage((page) => page + 1);
    };

    checkForMoreRef.current = checkForMore;

    const scheduleCheck = () => {
      if (frameId !== null) return;
      frameId = window.requestAnimationFrame(checkForMore);
    };

    document.addEventListener("scroll", scheduleCheck, true);
    window.addEventListener("scroll", scheduleCheck, { passive: true });
    window.addEventListener("resize", scheduleCheck);
    scheduleCheck();

    return () => {
      checkForMoreRef.current = () => undefined;
      document.removeEventListener("scroll", scheduleCheck, true);
      window.removeEventListener("scroll", scheduleCheck);
      window.removeEventListener("resize", scheduleCheck);
      if (frameId !== null) window.cancelAnimationFrame(frameId);
    };
  }, []);

  useEffect(() => {
    return () => {
      if (loadCooldownTimerRef.current) clearTimeout(loadCooldownTimerRef.current);
    };
  }, []);

  // Compute active filters count
  const activeFilterCount =
    selectedCategoryPills.length +
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
    setSelectedCategoryPills([]);
    setSearchQuery("");
    setShowOnlySaved(false);
    setCurrentPage(1);
  };

  return (
    <div className="space-y-5">
      {/* Toast Notification Popup */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 bg-slate-900 text-white text-xs font-semibold px-4 py-2.5 rounded-xl shadow-2xl border border-slate-700 animate-in fade-in slide-in-from-bottom duration-200 flex items-center gap-2">
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Top Main Section Switcher: Recruitments & Jobs vs Announcements */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 pb-3">
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setActiveMainTab("jobs");
              if (typeof window !== "undefined") window.location.hash = "jobs";
            }}
            className={`inline-flex items-center gap-2 rounded-xl px-4 py-2 text-xs sm:text-sm font-extrabold transition-all ${
              activeMainTab === "jobs"
                ? "bg-slate-950 text-white shadow-xs"
                : "border border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
            }`}
          >
            <BriefcaseBusiness className="h-4 w-4" />
            <span>Recruitments & Openings</span>
            {totalCount > 0 && (
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                  activeMainTab === "jobs"
                    ? "bg-teal-800 text-teal-200"
                    : "bg-slate-100 text-slate-600"
                }`}
              >
                {totalCount}
              </span>
            )}
          </button>

          <button
            onClick={() => {
              setActiveMainTab("announcements");
              if (typeof window !== "undefined") window.location.hash = "announcements";
            }}
            className={`inline-flex items-center gap-2 rounded-xl px-4 py-2 text-xs sm:text-sm font-extrabold transition-all ${
              activeMainTab === "announcements"
                ? "bg-slate-950 text-white shadow-xs"
                : "border border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
            }`}
          >
            <Megaphone className="h-4 w-4 text-teal-400" />
            <span>Official Announcements</span>
            {announcementsTotalCount > 0 && (
              <span
                className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                  activeMainTab === "announcements"
                    ? "bg-teal-800 text-teal-200"
                    : "bg-slate-100 text-slate-600"
                }`}
              >
                {announcementsTotalCount}
              </span>
            )}
          </button>
        </div>

        {/* Shortlist Quick Toggle */}
        {activeMainTab === "jobs" && (
          <button
            onClick={() => setShowOnlySaved((prev) => !prev)}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-bold transition-colors ${
              showOnlySaved
                ? "border border-amber-300 bg-amber-50 text-amber-800"
                : "border border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
            }`}
          >
            <span>★ Shortlisted</span>
            {savedJobIds.length > 0 && (
              <span className="rounded-full bg-amber-200/80 px-1.5 py-0.2 text-[10px] text-amber-900 font-extrabold">
                {savedJobIds.length}
              </span>
            )}
          </button>
        )}
      </div>

      {/* ============================================================== */}
      {/* 1. RECRUITMENT & JOBS SECTION                                  */}
      {/* ============================================================== */}
      {activeMainTab === "jobs" && (
        <div className="space-y-4">
          {/* Search & Category Pills */}
          <SearchAndCategoryBar
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            selectedCategoryPills={selectedCategoryPills}
            onToggleCategoryPill={(pill) => {
              if (pill === "All Jobs") {
                setSelectedCategoryPills([]);
                return;
              }
              setSelectedCategoryPills((current) =>
                current.includes(pill)
                  ? current.filter((item) => item !== pill)
                  : [...current, pill]
              );
            }}
            activeFilterCount={activeFilterCount}
            onToggleFiltersModal={handleToggleFilters}
            sortBy={sortBy}
            onSortChange={setSortBy}
            isFiltersOpen={isSidebarOpen}
            isMobileFiltersOpen={isMobileFilterOpen}
            onSelectTrendingSearch={(t) => setSearchQuery(t)}
          />

          {/* Main 2-Column Content Layout */}
          <div id="jobs" className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
            {/* Left Sidebar Filter (Desktop) */}
            {isSidebarOpen && (
              <aside className="hidden lg:block lg:col-span-3 sticky top-28 transition-all">
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
                  onClose={() => setIsSidebarOpen(false)}
                  activeCount={activeFilterCount}
                />
              </aside>
            )}

            {/* Right Main Content */}
            <div
              className={`${
                isSidebarOpen ? "lg:col-span-9" : "lg:col-span-12"
              } space-y-3.5 transition-all`}
            >
              {isLoading && jobs.length === 0 ? (
                <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center space-y-3">
                  <div className="animate-spin h-8 w-8 border-3 border-teal-700 border-t-transparent rounded-full mx-auto" />
                  <p className="text-slate-700 font-bold text-sm">Loading verified openings…</p>
                </div>
              ) : jobs.length > 0 ? (
                <>
                  {/* Desktop Table View */}
                  <div className="hidden sm:block">
                    <JobTableView
                      jobs={jobs}
                      savedJobIds={savedJobIds}
                      selectedJobId={lastSelectedJobId}
                      onToggleBookmark={handleToggleBookmark}
                      onSelectJob={handleSelectJob}
                    />
                  </div>

                  {/* Mobile Streamlined Cards View */}
                  <div className="block sm:hidden">
                    <JobMobileCards
                      jobs={jobs}
                      savedJobIds={savedJobIds}
                      selectedJobId={lastSelectedJobId}
                      onToggleBookmark={handleToggleBookmark}
                      onSelectJob={handleSelectJob}
                    />
                  </div>
                </>
              ) : (
                /* Empty State */
                <div className="bg-white rounded-2xl border border-dashed border-slate-300 p-10 text-center space-y-3 shadow-2xs">
                  <div className="w-12 h-12 rounded-full bg-teal-50 text-teal-700 flex items-center justify-center text-xl mx-auto">
                    <Search className="h-5 w-5" />
                  </div>
                  <div className="space-y-0.5">
                    <h3 className="text-base font-extrabold text-slate-800">
                      {showOnlySaved
                        ? "No Shortlisted Jobs Found"
                        : "No recruitments found matching your selection"}
                    </h3>
                    <p className="text-xs text-slate-500 max-w-md mx-auto">
                      {showOnlySaved
                        ? "Click the star icon (★) on any opening to save it for quick review."
                        : "Try broadening your keywords or resetting filters to view all available openings."}
                    </p>
                  </div>

                  <div className="pt-1 flex flex-wrap items-center justify-center gap-1.5">
                    <button
                      onClick={() => setSearchQuery("UPSC")}
                      className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-teal-50 text-xs font-semibold text-slate-700"
                    >
                      UPSC
                    </button>
                    <button
                      onClick={() => setSearchQuery("SSC")}
                      className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-teal-50 text-xs font-semibold text-slate-700"
                    >
                      SSC
                    </button>
                    <button
                      onClick={() => setSearchQuery("ISRO")}
                      className="px-2.5 py-1 rounded-full bg-slate-100 hover:bg-teal-50 text-xs font-semibold text-slate-700"
                    >
                      ISRO
                    </button>
                  </div>

                  <button
                    onClick={handleClearAll}
                    className="mt-2 px-4 py-2 bg-teal-700 text-white rounded-xl text-xs font-bold hover:bg-teal-800 shadow-2xs transition-all"
                  >
                    Reset All Filters
                  </button>
                </div>
              )}

              {!showOnlySaved && jobs.length > 0 && (
                <div
                  ref={loadMoreRef}
                  className="flex min-h-16 items-center justify-center border-t border-slate-200 py-4 text-xs text-slate-500"
                  aria-live="polite"
                >
                  {hasMoreJobs ? (
                    isLoading ? (
                      <span className="inline-flex items-center gap-2 font-semibold text-slate-600">
                        <LoaderCircle className="h-4 w-4 animate-spin text-teal-700" />
                        Loading more opportunities…
                      </span>
                    ) : (
                      <span>Scroll to load more</span>
                    )
                  ) : (
                    <span>All {totalCount} opportunities loaded</span>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ============================================================== */}
      {/* 2. DEDICATED ANNOUNCEMENTS SECTION                             */}
      {/* ============================================================== */}
      {activeMainTab === "announcements" && (
        <div id="announcements">
          <AnnouncementsView
            announcements={announcements}
            activeCategory={announcementCategory}
            onChangeCategory={setAnnouncementCategory}
            onSelectAnnouncement={handleSelectJob}
            isLoading={isAnnouncementsLoading}
          />
        </div>
      )}

      {/* Shared Details Modal (For both Jobs and Announcements) */}
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
