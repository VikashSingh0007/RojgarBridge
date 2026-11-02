"use client";

import { ExternalLink, FileText, Share2 } from "lucide-react";
import { JobItem } from "@/types";
import {
  getDeadlineStatus,
  getPortalBadgeStyle,
  getOfficialApplicationUrl,
  getOfficialNotificationUrl,
  isOfficialGovUrl,
} from "@/lib/formatters";

interface JobCardProps {
  job: JobItem;
  isBookmarked: boolean;
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
  onShareWhatsApp: (job: JobItem) => void;
}

export default function JobCard({
  job,
  isBookmarked,
  onToggleBookmark,
  onSelectJob,
  onShareWhatsApp,
}: JobCardProps) {
  const d = job.details;
  const deadline = getDeadlineStatus(d.application_end);
  const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);
  const applyUrl = getOfficialApplicationUrl(job);
  const noticeUrl = getOfficialNotificationUrl(job);
  const isGovApply = isOfficialGovUrl(applyUrl || undefined);

  return (
    <div
      onClick={() => onSelectJob(job)}
      className="group bg-white rounded-2xl border border-slate-200/90 hover:border-teal-600/70 p-5 shadow-xs hover:shadow-lg transition-all duration-200 flex flex-col justify-between cursor-pointer relative overflow-hidden"
    >
      {/* Top Accent Gradient Bar on Hover */}
      <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-teal-600 via-indigo-600 to-amber-500 opacity-0 group-hover:opacity-100 transition-opacity duration-200" />

      {/* Header Row: Portal + Category + Bookmark */}
      <div className="flex items-center justify-between gap-2 mb-3">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span
            className={`text-[11px] font-black uppercase tracking-wider px-2.5 py-0.5 rounded-md border ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
          >
            {job.source_portal || d.organization}
          </span>
          <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
            {d.category || "Central Govt"}
          </span>
          {d.notice_type && (
            <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-teal-50 text-teal-800 border border-teal-100 capitalize">
              {d.notice_type.replace("_", " ")}
            </span>
          )}
        </div>

        {/* Bookmark Star Button */}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleBookmark(job.id);
          }}
          className={`p-1.5 rounded-lg border transition-colors ${
            isBookmarked
              ? "bg-amber-50 text-amber-500 border-amber-300"
              : "bg-slate-50 text-slate-400 hover:text-amber-500 hover:bg-amber-50 border-slate-200"
          }`}
          title={isBookmarked ? "Remove from bookmarks" : "Save this job"}
          aria-label="Bookmark Job"
        >
          <svg
            className="w-4 h-4 fill-current"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={isBookmarked ? 0 : 2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
            />
          </svg>
        </button>
      </div>

      {/* Main Title & Organization */}
      <div className="space-y-1 mb-4">
        <h3 className="font-extrabold text-slate-900 group-hover:text-teal-800 transition-colors text-sm sm:text-base leading-snug line-clamp-2">
          {d.title}
        </h3>
        <p className="text-xs text-slate-500 flex items-center gap-1.5 truncate">
          <span>🏢 {d.organization}</span>
          {d.state && (
            <>
              <span className="text-slate-300">•</span>
              <span className="text-slate-600">📍 {d.state}</span>
            </>
          )}
        </p>
      </div>

      {/* Highlights Grid */}
      <div className="grid grid-cols-2 gap-2 bg-slate-50/80 rounded-xl p-3 border border-slate-200/70 text-xs mb-4">
        <div>
          <span className="text-[10px] text-slate-400 uppercase font-semibold block">
            Vacancies
          </span>
          <span className="font-bold text-slate-800">
            {d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} Posts` : "Refer PDF"}
          </span>
        </div>

        <div>
          <span className="text-[10px] text-slate-400 uppercase font-semibold block">
            Pay Scale
          </span>
          <span className="font-semibold text-emerald-700 truncate block" title={d.salary_payscale}>
            {d.salary_payscale || "As per Rules"}
          </span>
        </div>

        <div className="col-span-2 pt-1 border-t border-slate-200/50">
          <span className="text-[10px] text-slate-400 uppercase font-semibold block">
            Eligibility
          </span>
          <span className="font-medium text-slate-700 truncate block" title={d.qualification}>
            🎓 {d.qualification || "Refer official notification"}
          </span>
        </div>
      </div>

      {/* Deadline Info */}
      <div className="flex items-center justify-between gap-2 mb-3">
        <span
          className={`inline-flex items-center gap-1 text-[11px] font-semibold px-2.5 py-1 rounded-lg border ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
        >
          {deadline.label}
        </span>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onShareWhatsApp(job);
          }}
          className="p-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-600 border border-emerald-200 transition-colors text-xs flex items-center gap-1"
          title="Share on WhatsApp"
        >
          <Share2 className="w-3.5 h-3.5" />
          <span className="text-[10px] font-bold hidden sm:inline">Share</span>
        </button>
      </div>

      {/* Clearly Separated Action Buttons */}
      <div className="grid grid-cols-2 gap-2 pt-3 border-t border-slate-100">
        {/* View Notification Action */}
        {noticeUrl ? (
          <a
            href={noticeUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-bold text-slate-700 bg-white hover:bg-slate-100 border border-slate-300 rounded-lg transition-colors shadow-2xs"
            title="Open official recruitment notification PDF"
          >
            <FileText className="w-3.5 h-3.5 text-slate-500" />
            <span>View Notice</span>
          </a>
        ) : (
          <button
            disabled
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-semibold text-slate-400 bg-slate-50 border border-slate-200 rounded-lg cursor-not-allowed"
          >
            <FileText className="w-3.5 h-3.5 opacity-50" />
            <span>Notice Pending</span>
          </button>
        )}

        {/* Apply Online Action */}
        {applyUrl ? (
          <a
            href={applyUrl}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-extrabold text-white bg-slate-950 hover:bg-slate-800 rounded-lg transition-all shadow-2xs"
            title={isGovApply ? "Apply directly on official government website" : "Apply online"}
          >
            <span>Apply Online</span>
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
        ) : (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onSelectJob(job);
            }}
            className="inline-flex items-center justify-center gap-1 px-3 py-2 text-xs font-bold text-teal-800 bg-teal-50 hover:bg-teal-100 border border-teal-200 rounded-lg transition-colors"
          >
            <span>View Details</span>
          </button>
        )}
      </div>
    </div>
  );
}
