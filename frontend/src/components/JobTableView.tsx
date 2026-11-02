"use client";

import {
  Bookmark,
  CalendarClock,
  ExternalLink,
  FileText,
  GraduationCap,
  MapPin,
  Users,
} from "lucide-react";
import { JobItem } from "@/types";
import {
  getCompactDeadlineLabel,
  getDeadlineStatus,
  getOfficialApplicationUrl,
  getOfficialNotificationUrl,
  getPortalBadgeStyle,
  getShortQualification,
  isOfficialGovUrl,
} from "@/lib/formatters";

interface JobTableViewProps {
  jobs: JobItem[];
  savedJobIds: number[];
  selectedJobId?: number | null;
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
}

export default function JobTableView({
  jobs,
  savedJobIds,
  selectedJobId,
  onToggleBookmark,
  onSelectJob,
}: JobTableViewProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="divide-y divide-slate-100">
        {jobs.map((job) => {
          const d = job.details;
          const deadline = getDeadlineStatus(d.application_end);
          const isSaved = savedJobIds.includes(job.id);
          const isSelected = selectedJobId === job.id;
          const location = d.state && d.state.toLowerCase() !== "all india" ? d.state : null;
          const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);
          const applyUrl = getOfficialApplicationUrl(job);
          const noticeUrl = getOfficialNotificationUrl(job);
          const isGovApply = isOfficialGovUrl(applyUrl || undefined);

          return (
            <article
              key={job.id}
              onClick={() => onSelectJob(job)}
              className={`group grid cursor-pointer gap-4 px-5 py-4 transition-colors lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center ${
                isSelected
                  ? "bg-slate-50/80 shadow-[inset_3px_0_0_#0f766e]"
                  : "hover:bg-slate-50/70"
              }`}
              aria-current={isSelected ? "true" : undefined}
            >
              {/* Left Column: Title, Metadata, Badges */}
              <div className="min-w-0 space-y-1.5">
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-extrabold uppercase tracking-wide ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
                  >
                    {job.source_portal || d.organization}
                  </span>
                  {d.category && (
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-semibold text-slate-600">
                      {d.category}
                    </span>
                  )}
                  {d.salary_payscale && (
                    <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-bold text-emerald-700 border border-emerald-100">
                      💰 {d.salary_payscale}
                    </span>
                  )}
                </div>

                <h3 className="text-xs font-extrabold leading-[1.5] text-slate-950 transition-colors group-hover:text-teal-800 sm:text-sm">
                  {d.title}
                </h3>

                <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-600">
                  <span className="inline-flex items-center gap-1.5 font-medium">
                    <GraduationCap className="h-3.5 w-3.5 text-slate-400" />
                    {getShortQualification(d.qualification)}
                  </span>
                  <span className="inline-flex items-center gap-1.5 font-semibold text-slate-700">
                    <Users className="h-3.5 w-3.5 text-slate-400" />
                    {d.total_vacancies > 0
                      ? `${d.total_vacancies.toLocaleString()} vacancies`
                      : "Vacancies in notice"}
                  </span>
                  {location && (
                    <span className="inline-flex items-center gap-1.5">
                      <MapPin className="h-3.5 w-3.5 text-slate-400" />
                      {location}
                    </span>
                  )}
                </div>
              </div>

              {/* Right Column: Deadline + Differentiated Action Buttons */}
              <div className="flex flex-wrap items-center justify-between gap-2.5 sm:justify-end">
                {/* Deadline Badge */}
                <span
                  className={`inline-flex min-h-8 items-center gap-1.5 whitespace-nowrap rounded-lg border px-2.5 text-xs font-extrabold ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
                >
                  <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
                  {getCompactDeadlineLabel(deadline.label)}
                </span>

                {/* View Official Notification (PDF) */}
                {noticeUrl ? (
                  <a
                    href={noticeUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={(e) => e.stopPropagation()}
                    className="inline-flex min-h-8 items-center gap-1.5 whitespace-nowrap rounded-lg border border-slate-300 bg-white px-3 text-xs font-bold text-slate-700 shadow-2xs transition-colors hover:bg-slate-100 hover:text-slate-900"
                    title="Open official recruitment notification document (PDF)"
                  >
                    <FileText className="h-3.5 w-3.5 text-slate-500" />
                    <span>Notification</span>
                  </a>
                ) : (
                  <span
                    onClick={(e) => e.stopPropagation()}
                    className="inline-flex min-h-8 items-center gap-1 rounded-lg border border-slate-200 bg-slate-50 px-2.5 text-xs font-medium text-slate-400 cursor-default"
                  >
                    <FileText className="h-3.5 w-3.5 opacity-50" />
                    <span>Pending</span>
                  </span>
                )}

                {/* Apply Online (Official Portal) */}
                {applyUrl ? (
                  <a
                    href={applyUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={(e) => e.stopPropagation()}
                    className="inline-flex min-h-8 items-center gap-1.5 whitespace-nowrap rounded-lg bg-slate-950 px-3.5 text-xs font-extrabold text-white shadow-2xs transition-all hover:bg-slate-800"
                    title={isGovApply ? "Apply directly on official government website" : "Apply online"}
                  >
                    <span>Apply Online</span>
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                ) : (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelectJob(job);
                    }}
                    className="inline-flex min-h-8 items-center gap-1 rounded-lg border border-teal-200 bg-teal-50 px-3 text-xs font-bold text-teal-800 hover:bg-teal-100"
                  >
                    <span>Details</span>
                  </button>
                )}

                {/* Bookmark Toggle */}
                <button
                  onClick={(event) => {
                    event.stopPropagation();
                    onToggleBookmark(job.id);
                  }}
                  className={`flex h-8 w-8 items-center justify-center rounded-lg border transition-colors ${
                    isSaved
                      ? "border-amber-200 bg-amber-50 text-amber-600"
                      : "border-slate-200 bg-white text-slate-400 hover:border-teal-300 hover:text-teal-800"
                  }`}
                  aria-label={isSaved ? "Remove saved job" : "Save job"}
                  title={isSaved ? "Remove from bookmarks" : "Save this job"}
                >
                  <Bookmark className={`h-3.5 w-3.5 ${isSaved ? "fill-current" : ""}`} />
                </button>
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
