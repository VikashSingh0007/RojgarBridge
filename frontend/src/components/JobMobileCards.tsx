"use client";

import { JobItem } from "@/types";
import { getDeadlineStatus, getPortalBadgeStyle } from "@/lib/formatters";

interface JobMobileCardsProps {
  jobs: JobItem[];
  savedJobIds: number[];
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
  onShareWhatsApp: (job: JobItem) => void;
}

export default function JobMobileCards({
  jobs,
  savedJobIds,
  onToggleBookmark,
  onSelectJob,
  onShareWhatsApp,
}: JobMobileCardsProps) {
  return (
    <div className="space-y-3">
      {jobs.map((job) => {
        const d = job.details;
        const isSaved = savedJobIds.includes(job.id);
        const deadline = getDeadlineStatus(d.application_end);
        const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);

        return (
          <div
            key={job.id}
            onClick={() => onSelectJob(job)}
            className="bg-white rounded-2xl border border-slate-200/90 p-4 shadow-2xs hover:shadow-md transition-all cursor-pointer relative space-y-3"
          >
            {/* Top row: Portal + Badges + Bookmark */}
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5 flex-wrap">
                <span
                  className={`text-[10px] font-black uppercase px-2 py-0.5 rounded border ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
                >
                  {job.source_portal || d.organization}
                </span>
                <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
                  {d.category || "Central Govt"}
                </span>
              </div>

              <div className="flex items-center gap-1">
                {/* WhatsApp Share */}
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onShareWhatsApp(job);
                  }}
                  className="p-1 rounded-md text-emerald-600 bg-emerald-50 border border-emerald-200"
                  title="Share"
                >
                  <svg className="w-3.5 h-3.5 fill-current" viewBox="0 0 24 24">
                    <path d="M12.031 6.172c-3.181 0-5.767 2.586-5.768 5.766-.001 1.298.38 2.27 1.019 3.287l-.582 2.128 2.182-.573c.978.58 1.911.928 3.145.929 3.178 0 5.767-2.587 5.768-5.766.001-3.187-2.575-5.77-5.764-5.771zm3.392 8.244c-.144.405-.837.774-1.17.824-.299.045-.677.063-1.092-.069-.252-.08-.575-.187-.988-.365-1.739-.751-2.874-2.502-2.961-2.617-.087-.116-.708-.94-.708-1.793s.448-1.273.607-1.446c.159-.173.346-.217.462-.217l.332.006c.106.005.249-.04.39.298.144.347.491 1.2.534 1.287.043.087.072.188.014.304-.058.116-.087.188-.173.289l-.26.304c-.087.086-.177.18-.076.354.101.174.449.741.964 1.201.662.591 1.221.774 1.394.86.174.086.275.073.376-.044.101-.116.433-.506.549-.68.116-.173.231-.145.39-.086s1.011.477 1.184.564.289.13.332.202c.045.072.045.419-.1.824z" />
                  </svg>
                </button>

                {/* Bookmark */}
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onToggleBookmark(job.id);
                  }}
                  className={`p-1 rounded-md border ${
                    isSaved
                      ? "text-amber-500 bg-amber-50 border-amber-300"
                      : "text-slate-400 bg-slate-50 border-slate-200"
                  }`}
                  title="Bookmark"
                >
                  <svg className="w-3.5 h-3.5 fill-current" viewBox="0 0 24 24">
                    <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
                  </svg>
                </button>
              </div>
            </div>

            {/* Title & Organization */}
            <div>
              <h4 className="font-extrabold text-slate-900 text-sm tracking-tight line-clamp-2 leading-snug">
                {d.title}
              </h4>
              <p className="text-xs text-slate-500 truncate mt-1">
                🏢 {d.organization} • 📍 {d.state || "All India"}
              </p>
            </div>

            {/* Meta row: Posts & Qualification */}
            <div className="flex items-center gap-3 text-xs text-slate-600 bg-slate-50 p-2.5 rounded-xl border border-slate-100">
              <span className="font-semibold text-slate-800">
                👤 {d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} Posts` : "Refer PDF"}
              </span>
              <span className="text-slate-300">|</span>
              <span className="truncate max-w-[170px] font-medium" title={d.qualification}>
                🎓 {d.qualification || "Refer notice"}
              </span>
            </div>

            {/* Footer row: Deadline + Details CTA */}
            <div className="flex items-center justify-between gap-2 pt-1">
              <span
                className={`inline-flex items-center text-[10px] font-bold px-2 py-0.5 rounded-md border ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
              >
                {deadline.label}
              </span>

              <span className="text-xs font-bold text-blue-600 flex items-center gap-1">
                View Details →
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
