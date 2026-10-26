"use client";

import { useEffect, useState, useMemo } from "react";
import Link from "next/link";
import { JobItem } from "@/types";
import { fetchPublishedJobs, getApiBaseUrl } from "@/lib/api";
import { INITIAL_PUBLISHED_JOBS } from "@/lib/mockData";
import SearchAndCategoryBar from "@/components/SearchAndCategoryBar";
import SidebarFilters from "@/components/SidebarFilters";
import JobTableView from "@/components/JobTableView";
import JobMobileCards from "@/components/JobMobileCards";
import Pagination from "@/components/Pagination";
import JobDetailsModal from "@/components/JobDetailsModal";

export default function CandidateJobsPage() {
  const [jobs, setJobs] = useState<JobItem[]>(INITIAL_PUBLISHED_JOBS);
  const [totalBackendCount, setTotalBackendCount] = useState<number>(
    INITIAL_PUBLISHED_JOBS.length
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategoryPill, setSelectedCategoryPill] = useState("All Jobs");

  // Sidebar Filters State - all start completely empty so no unexpected auto-filters!
  const [selectedCategories, setSelectedCategories] = useState<string[]>([]);
  const [selectedQualifications, setSelectedQualifications] = useState<string[]>([]);
  const [selectedStates, setSelectedStates] = useState<string[]>([]);
  const [selectedJobTypes, setSelectedJobTypes] = useState<string[]>([]);
  const [sortBy, setSortBy] = useState<"latest" | "deadline" | "posts">("latest");

  // Pagination state
  const [currentPage, setCurrentPage] = useState(1);
  const PAGE_SIZE = 10;

  // Selected Job for Detail Modal
  const [selectedJob, setSelectedJob] = useState<JobItem | null>(null);

  // Mobile Filters Modal State
  const [isMobileFilterOpen, setIsMobileFilterOpen] = useState(false);
  const [isConnectedToBackend, setIsConnectedToBackend] = useState<boolean | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Load from Go Backend API: GET /api/jobs/published
  const loadPublishedJobs = async () => {
    setIsLoading(true);
    try {
      const res = await fetchPublishedJobs({ limit: 100 });
      if (res && Array.isArray(res.jobs)) {
        if (res.jobs.length > 0) {
          setJobs(res.jobs);
          setTotalBackendCount(res.count || res.jobs.length);
        }
        setIsConnectedToBackend(true);
      }
    } catch (err) {
      console.warn("Using local fallback data (backend connecting...):", err);
      setIsConnectedToBackend(false);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadPublishedJobs();
  }, []);

  // Compute active filters count
  const activeFilterCount =
    (selectedCategoryPill !== "All Jobs" ? 1 : 0) +
    selectedCategories.length +
    selectedQualifications.length +
    selectedStates.length +
    selectedJobTypes.length +
    (searchQuery.trim() ? 1 : 0);

  // Clear all filters
  const handleClearAll = () => {
    setSelectedCategories([]);
    setSelectedQualifications([]);
    setSelectedStates([]);
    setSelectedJobTypes([]);
    setSelectedCategoryPill("All Jobs");
    setSearchQuery("");
    setCurrentPage(1);
  };

  // Reset page when filters change
  useEffect(() => {
    setCurrentPage(1);
  }, [
    searchQuery,
    selectedCategoryPill,
    selectedCategories,
    selectedQualifications,
    selectedStates,
    selectedJobTypes,
    sortBy,
  ]);

  // Robust Filter & Search & Sort Logic
  const filteredAndSortedJobs = useMemo(() => {
    const list = jobs.filter((job) => {
      const d = job.details;

      // 1. Search Query
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase().trim();
        const matches =
          (d.title && d.title.toLowerCase().includes(q)) ||
          (d.organization && d.organization.toLowerCase().includes(q)) ||
          (d.qualification && d.qualification.toLowerCase().includes(q)) ||
          (d.advt_number && d.advt_number.toLowerCase().includes(q)) ||
          (d.category && d.category.toLowerCase().includes(q)) ||
          (d.state && d.state.toLowerCase().includes(q)) ||
          (job.source_portal && job.source_portal.toLowerCase().includes(q)) ||
          (d.summary_points &&
            d.summary_points.some((p) => p.toLowerCase().includes(q)));
        if (!matches) return false;
      }

      // 2. Category Pill Filter (e.g. UPSC, SSC, Railway, Bank, 10th Pass, etc.)
      if (selectedCategoryPill !== "All Jobs") {
        const pill = selectedCategoryPill.toLowerCase();
        if (pill === "10th pass") {
          if (!d.qualification.toLowerCase().includes("10th")) return false;
        } else if (pill === "12th pass") {
          if (!d.qualification.toLowerCase().includes("12th")) return false;
        } else if (pill === "graduate") {
          if (
            !d.qualification.toLowerCase().includes("graduate") &&
            !d.qualification.toLowerCase().includes("degree")
          ) {
            return false;
          }
        } else if (pill === "engineering") {
          if (
            !d.qualification.toLowerCase().includes("b.e") &&
            !d.qualification.toLowerCase().includes("b.tech") &&
            !d.qualification.toLowerCase().includes("engineering")
          ) {
            return false;
          }
        } else if (pill === "medical") {
          const medText = `${d.title} ${d.qualification} ${d.category}`.toLowerCase();
          if (
            !medText.includes("medic") &&
            !medText.includes("doctor") &&
            !medText.includes("physician") &&
            !medText.includes("surgeon") &&
            !medText.includes("health") &&
            !medText.includes("ayush")
          ) {
            return false;
          }
        } else {
          // General matching for UPSC, SSC, Railway, Bank, State Govt, etc.
          const text = `${d.title} ${d.organization} ${d.category} ${job.source_portal} ${d.notice_type}`.toLowerCase();
          if (!text.includes(pill)) return false;
        }
      }

      // 3. Category / Sector Filter (from Sidebar)
      if (selectedCategories.length > 0) {
        const matchSector = selectedCategories.some((cat) => {
          const c = cat.toLowerCase();
          const target = `${d.category} ${d.organization} ${job.source_portal} ${d.title}`.toLowerCase();
          if (c === "central govt") return target.includes("central") || ["upsc", "ssc", "rrb", "ibps", "isro"].some(p => target.includes(p));
          if (c === "state govt") return target.includes("state") || ["uppsc", "bpsc", "mppsc", "rpsc", "wbpsc"].some(p => target.includes(p));
          if (c === "banking") return target.includes("bank") || target.includes("ibps") || target.includes("sbi");
          if (c === "defence & police") return target.includes("defence") || target.includes("police") || target.includes("army") || target.includes("navy");
          if (c === "teaching") return target.includes("teach") || target.includes("school") || target.includes("lecturer") || target.includes("principal");
          if (c === "engineering & psu") return target.includes("engineer") || target.includes("psu") || target.includes("b.tech") || target.includes("isro");
          return target.includes(c);
        });
        if (!matchSector) return false;
      }

      // 4. Notice Type Filter (from Sidebar)
      if (selectedJobTypes.length > 0) {
        const matchNoticeType = selectedJobTypes.some((type) => {
          const t = type.toLowerCase();
          const jobType = (d.notice_type || "").toLowerCase().replace("_", " ");
          if (t.includes("recruitment")) return jobType.includes("recruitment") || !jobType;
          if (t.includes("admit card")) return jobType.includes("admit");
          if (t.includes("result")) return jobType.includes("result");
          if (t.includes("answer key")) return jobType.includes("answer");
          return jobType.includes(t);
        });
        if (!matchNoticeType) return false;
      }

      // 5. Qualification Filter (from Sidebar)
      if (selectedQualifications.length > 0) {
        const qTarget = (d.qualification || "").toLowerCase();
        const matchesQual = selectedQualifications.some((qual) => {
          const q = qual.toLowerCase();
          if (q.includes("10th")) return qTarget.includes("10th") || qTarget.includes("matric");
          if (q.includes("12th")) return qTarget.includes("12th") || qTarget.includes("intermediate") || qTarget.includes("10+2");
          if (q.includes("iti")) return qTarget.includes("iti");
          if (q.includes("diploma")) return qTarget.includes("diploma");
          if (q.includes("b.tech") || q.includes("b.e")) return qTarget.includes("b.e") || qTarget.includes("b.tech") || qTarget.includes("engineer");
          if (q.includes("post graduate")) return qTarget.includes("post graduate") || qTarget.includes("pg") || qTarget.includes("master");
          if (q.includes("graduate")) return qTarget.includes("graduate") || qTarget.includes("degree") || qTarget.includes("bachelor");
          if (q.includes("ca") || q.includes("finance")) return qTarget.includes("ca") || qTarget.includes("finance");
          return qTarget.includes(q);
        });
        if (!matchesQual) return false;
      }

      // 6. State Filter (from Sidebar)
      if (selectedStates.length > 0) {
        const jobState = (d.state || "All India").toLowerCase();
        const isAllIndia = jobState.includes("all india");
        const matchesState =
          isAllIndia ||
          selectedStates.some(
            (st) => st.toLowerCase() === jobState || jobState.includes(st.toLowerCase())
          );
        if (!matchesState) return false;
      }

      return true;
    });

    // Sorting
    return [...list].sort((a, b) => {
      if (sortBy === "posts") {
        return (b.details.total_vacancies || 0) - (a.details.total_vacancies || 0);
      }
      if (sortBy === "deadline") {
        const dateA = a.details.application_end ? new Date(a.details.application_end).getTime() : Infinity;
        const dateB = b.details.application_end ? new Date(b.details.application_end).getTime() : Infinity;
        return (isNaN(dateA) ? Infinity : dateA) - (isNaN(dateB) ? Infinity : dateB);
      }
      // default: latest
      const timeA = new Date(a.published_at || a.created_at || 0).getTime();
      const timeB = new Date(b.published_at || b.created_at || 0).getTime();
      if (timeA !== timeB) return timeB - timeA;
      return b.id - a.id;
    });
  }, [
    jobs,
    searchQuery,
    selectedCategoryPill,
    selectedCategories,
    selectedJobTypes,
    selectedQualifications,
    selectedStates,
    sortBy,
  ]);

  // Paginated items
  const totalFilteredCount = filteredAndSortedJobs.length;
  const totalPages = Math.ceil(totalFilteredCount / PAGE_SIZE) || 1;
  const paginatedJobs = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredAndSortedJobs.slice(start, start + PAGE_SIZE);
  }, [filteredAndSortedJobs, currentPage]);

  return (
    <div className="space-y-6">
      {/* Top Banner / Backend Status indicator */}
      {isConnectedToBackend === false && (
        <div className="bg-amber-50 border border-amber-200 text-amber-900 px-4 py-2.5 rounded-xl text-xs flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span>ℹ️</span>
            <span>
              <strong>Notice:</strong> Showing preloaded published notifications. Backend API configured at{" "}
              <code className="font-mono bg-amber-100 px-1 py-0.5 rounded">
                {getApiBaseUrl()}/api/jobs/published
              </code>
            </span>
          </div>
          <button
            onClick={loadPublishedJobs}
            className="font-semibold text-blue-700 underline hover:text-blue-900 ml-2"
          >
            Retry Connection ↻
          </button>
        </div>
      )}

      {/* Search & Category Pills Top Section */}
      <SearchAndCategoryBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        selectedCategoryPill={selectedCategoryPill}
        onSelectCategoryPill={setSelectedCategoryPill}
        activeFilterCount={activeFilterCount}
        onToggleFiltersModal={() => setIsMobileFilterOpen(true)}
      />

      {/* Breadcrumb & Quick Actions */}
      <div className="flex items-center justify-between pt-1">
        <nav className="flex items-center gap-2 text-xs font-medium text-slate-500">
          <Link href="/" className="hover:text-blue-600 cursor-pointer">
            Home
          </Link>
          <span>›</span>
          <span className="text-slate-800 font-semibold">Government Jobs</span>
        </nav>

        <div className="flex items-center gap-2">
          {isLoading && (
            <span className="text-xs text-blue-600 animate-pulse font-medium">
              Updating list...
            </span>
          )}
          <Link
            href="/admin"
            className="inline-flex items-center gap-1.5 text-xs font-semibold px-3 py-1 rounded-full bg-slate-100 hover:bg-slate-200 text-slate-700 border border-slate-300 transition-colors"
          >
            <span>🛡️ Admin Portal</span>
            <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
          </Link>
        </div>
      </div>

      {/* Active Filter Chips / Badges Bar */}
      {activeFilterCount > 0 && (
        <div className="flex flex-wrap items-center gap-2 bg-slate-100/70 p-3 rounded-xl border border-slate-200 text-xs">
          <span className="font-bold text-slate-600">Active Filters:</span>

          {searchQuery.trim() && (
            <span className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2.5 py-1 rounded-full font-medium shadow-2xs">
              Keyword: &quot;{searchQuery}&quot;
              <button
                onClick={() => setSearchQuery("")}
                className="hover:text-red-600 font-bold ml-0.5"
              >
                ✕
              </button>
            </span>
          )}

          {selectedCategoryPill !== "All Jobs" && (
            <span className="inline-flex items-center gap-1.5 bg-blue-50 border border-blue-200 text-blue-700 px-2.5 py-1 rounded-full font-semibold shadow-2xs">
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
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2.5 py-1 rounded-full font-medium shadow-2xs"
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
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2.5 py-1 rounded-full font-medium shadow-2xs"
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
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2.5 py-1 rounded-full font-medium shadow-2xs"
            >
              Qual: {qual}
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
              className="inline-flex items-center gap-1.5 bg-white border border-slate-300 text-slate-700 px-2.5 py-1 rounded-full font-medium shadow-2xs"
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
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
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

        {/* Right Main Table & Listings */}
        <div className="lg:col-span-9 space-y-4">
          {/* Section Header & Sort Controls */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight">
                Government Job Notifications
              </h1>
              <p className="text-xs sm:text-sm text-slate-500 mt-0.5">
                Official, verified government recruitment notices, admit cards, and results.
              </p>
            </div>

            <div className="flex items-center gap-3 self-end sm:self-center text-xs">
              <div className="flex items-center gap-1.5 bg-white border border-slate-200 px-3 py-1.5 rounded-lg shadow-2xs">
                <span className="text-slate-500">Sort by</span>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as any)}
                  className="bg-transparent font-bold text-slate-800 focus:outline-none cursor-pointer pr-1"
                >
                  <option value="latest">Latest</option>
                  <option value="deadline">Closing Soon</option>
                  <option value="posts">Highest Posts</option>
                </select>
              </div>

              <span className="text-slate-400 font-medium">
                Showing <strong className="text-slate-800">{totalFilteredCount}</strong>{" "}
                {totalFilteredCount === 1 ? "job" : "jobs"}
              </span>
            </div>
          </div>

          {/* Table View (Desktop & Tablet) */}
          <div className="hidden sm:block">
            {paginatedJobs.length > 0 ? (
              <JobTableView
                jobs={paginatedJobs}
                onSelectJob={(job) => setSelectedJob(job)}
              />
            ) : (
              <div className="bg-white rounded-2xl border border-dashed border-slate-300 p-12 text-center space-y-3">
                <p className="text-slate-600 font-semibold text-base">
                  No notifications match your active filters
                </p>
                <p className="text-xs text-slate-400">
                  Try clearing some filters or searching with different keywords.
                </p>
                <button
                  onClick={handleClearAll}
                  className="px-4 py-2 bg-blue-600 text-white rounded-lg text-xs font-semibold hover:bg-blue-700 transition-colors"
                >
                  Reset All Filters
                </button>
              </div>
            )}
          </div>

          {/* Cards View (Mobile Devices) */}
          <div className="block sm:hidden">
            {paginatedJobs.length > 0 ? (
              <JobMobileCards
                jobs={paginatedJobs}
                onSelectJob={(job) => setSelectedJob(job)}
              />
            ) : (
              <div className="bg-white rounded-xl border border-dashed border-slate-300 p-8 text-center space-y-2">
                <p className="text-slate-600 font-semibold text-sm">
                  No notifications found
                </p>
                <button
                  onClick={handleClearAll}
                  className="px-3.5 py-1.5 bg-blue-600 text-white rounded-lg text-xs font-semibold"
                >
                  Reset Filters
                </button>
              </div>
            )}
          </div>

          {/* Dynamic Pagination */}
          <Pagination
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={setCurrentPage}
            totalItems={totalFilteredCount}
            pageSize={PAGE_SIZE}
          />
        </div>
      </div>

      {/* Details Modal */}
      <JobDetailsModal
        job={selectedJob}
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
