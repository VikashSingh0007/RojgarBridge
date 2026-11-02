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

interface JobMobileCardsProps {
  jobs: JobItem[];
  savedJobIds: number[];
  selectedJobId?: number | null;
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
}

export default function JobMobileCards({
  jobs,
  savedJobIds,
  selectedJobId,
  onToggleBookmark,
  onSelectJob,
}: JobMobileCardsProps) {
  return (
    <div className="space-y-3">
      {jobs.map((job) => {
        const d = job.details;
        const isSaved = savedJobIds.includes(job.id);
        const isSelected = selectedJobId === job.id;
        const deadline = getDeadlineStatus(d.application_end);
        const location = d.state && d.state.toLowerCase() !== "all india" ? d.state : null;
        const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);
        const applyUrl = getOfficialApplicationUrl(job);
        const noticeUrl = getOfficialNotificationUrl(job);
        const isGovApply = isOfficialGovUrl(applyUrl || undefined);

        return (
          <article
            key={job.id}
            onClick={() => onSelectJob(job)}
            className={`cursor-pointer rounded-xl border border-slate-200 bg-white p-4 transition-all ${
              isSelected ? "shadow-[inset_3px_0_0_#0f766e] ring-1 ring-teal-600/30" : "shadow-sm"
            }`}
            aria-current={isSelected ? "true" : undefined}
          >
            {/* Top row: Portal + Title */}
            <div className="flex items-start justify-between gap-2.5">
              <div className="min-w-0 flex-1 space-y-1">
                <div className="flex flex-wrap items-center gap-1.5">
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
                </div>
                <h3 className="text-xs font-extrabold leading-[1.4] text-slate-950 line-clamp-2">
                  {d.title}
                </h3>
              </div>

              {/* Bookmark */}
              <button
                onClick={(event) => {
                  event.stopPropagation();
                  onToggleBookmark(job.id);
                }}
                className={`flex h-8 w-8 flex-none items-center justify-center rounded-lg border transition-colors ${
                  isSaved
                    ? "border-amber-200 bg-amber-50 text-amber-600"
                    : "border-slate-200 bg-white text-slate-400"
                }`}
                aria-label={isSaved ? "Remove saved job" : "Save job"}
              >
                <Bookmark className={`h-4 w-4 ${isSaved ? "fill-current" : ""}`} />
              </button>
            </div>

            {/* Middle metadata */}
            <div className="mt-2.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-600">
              <span className="inline-flex items-center gap-1">
                <GraduationCap className="h-3.5 w-3.5 text-slate-400" />
                {getShortQualification(d.qualification)}
              </span>
              <span className="inline-flex items-center gap-1 font-semibold text-slate-700">
                <Users className="h-3.5 w-3.5 text-slate-400" />
                {d.total_vacancies > 0
                  ? `${d.total_vacancies.toLocaleString()} vacancies`
                  : "Refer notice"}
              </span>
              {location && (
                <span className="inline-flex items-center gap-1">
                  <MapPin className="h-3.5 w-3.5 text-slate-400" />
                  {location}
                </span>
              )}
            </div>

            {/* Deadline + Salary */}
            <div className="mt-3 flex items-center justify-between border-t border-slate-100 pt-2.5">
              <span
                className={`inline-flex min-h-7 items-center gap-1.5 rounded-lg border px-2 text-[11px] font-extrabold ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
              >
                <CalendarClock className="h-3 w-3" aria-hidden="true" />
                {getCompactDeadlineLabel(deadline.label)}
              </span>

              {d.salary_payscale && (
                <span className="text-[11px] font-bold text-emerald-700 truncate max-w-[150px]">
                  {d.salary_payscale}
                </span>
              )}
            </div>

            {/* Differentiated Bottom Actions */}
            <div className="mt-3 grid grid-cols-2 gap-2 pt-2 border-t border-slate-100">
              {noticeUrl ? (
                <a
                  href={noticeUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  className="inline-flex min-h-8 items-center justify-center gap-1.5 rounded-lg border border-slate-300 bg-white px-2 text-xs font-bold text-slate-700 shadow-2xs transition-colors hover:bg-slate-50"
                >
                  <FileText className="h-3.5 w-3.5 text-slate-500" />
                  <span>View Notice</span>
                </a>
              ) : (
                <span
                  onClick={(e) => e.stopPropagation()}
                  className="inline-flex min-h-8 items-center justify-center gap-1 rounded-lg border border-slate-200 bg-slate-50 px-2 text-xs font-medium text-slate-400"
                >
                  <span>Notice Pending</span>
                </span>
              )}

              {applyUrl ? (
                <a
                  href={applyUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  className="inline-flex min-h-8 items-center justify-center gap-1.5 rounded-lg bg-slate-950 px-2 text-xs font-extrabold text-white shadow-2xs transition-colors hover:bg-slate-800"
                  title={isGovApply ? "Apply directly on official portal" : "Apply online"}
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
                  className="inline-flex min-h-8 items-center justify-center rounded-lg border border-teal-200 bg-teal-50 px-2 text-xs font-bold text-teal-800"
                >
                  <span>Details</span>
                </button>
              )}
            </div>
          </article>
        );
      })}
    </div>
  );
}
