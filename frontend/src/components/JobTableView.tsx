"use client";

import { Bookmark, CalendarClock, GraduationCap, MapPin, Users } from "lucide-react";
import { JobItem } from "@/types";
import { getCompactDeadlineLabel, getDeadlineStatus, getShortQualification } from "@/lib/formatters";

interface JobTableViewProps {
  jobs: JobItem[];
  savedJobIds: number[];
  selectedJobId?: number | null;
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
}

export default function JobTableView({ jobs, savedJobIds, selectedJobId, onToggleBookmark, onSelectJob }: JobTableViewProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="divide-y divide-slate-100">
        {jobs.map((job) => {
          const d = job.details;
          const deadline = getDeadlineStatus(d.application_end);
          const isSaved = savedJobIds.includes(job.id);
          const isSelected = selectedJobId === job.id;
          const location = d.state && d.state.toLowerCase() !== "all india" ? d.state : null;

          return (
            <article
              key={job.id}
              onClick={() => onSelectJob(job)}
              className={`group grid cursor-pointer gap-4 px-5 py-4 transition-colors sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center ${isSelected ? "shadow-[inset_3px_0_0_#0f766e] hover:bg-slate-50" : "hover:bg-slate-50"}`}
              aria-current={isSelected ? "true" : undefined}
            >
              <div className="min-w-0">
                <h3 className="text-base font-bold leading-snug text-slate-950 transition-colors group-hover:text-teal-800 sm:text-lg">
                  {d.title}
                </h3>

                <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-sm text-slate-600">
                  <span className="inline-flex items-center gap-1.5">
                    <GraduationCap className="h-4 w-4 text-slate-400" />
                    {getShortQualification(d.qualification)}
                  </span>
                  <span className="inline-flex items-center gap-1.5">
                    <Users className="h-4 w-4 text-slate-400" />
                    {d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} vacancies` : "Vacancies in notice"}
                  </span>
                  {location && (
                    <span className="inline-flex items-center gap-1.5">
                      <MapPin className="h-4 w-4 text-slate-400" />{location}
                    </span>
                  )}
                </div>
              </div>

              <div className="flex items-center justify-between gap-3 sm:justify-end">
                <span className="whitespace-nowrap text-[11px] font-bold uppercase tracking-wide text-slate-400">
                  {job.source_portal || d.organization}
                </span>
                <span className={`inline-flex min-h-9 items-center gap-1.5 whitespace-nowrap rounded-lg border px-3 text-sm font-extrabold ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}>
                  <CalendarClock className="h-4 w-4" aria-hidden="true" />
                  {getCompactDeadlineLabel(deadline.label)}
                </span>
                <button
                  onClick={(event) => { event.stopPropagation(); onToggleBookmark(job.id); }}
                  className={`flex h-10 w-10 items-center justify-center rounded-lg border transition-colors ${isSaved ? "border-amber-200 bg-amber-50 text-amber-600" : "border-slate-200 bg-white text-slate-500 hover:border-teal-300 hover:text-teal-800"}`}
                  aria-label={isSaved ? "Remove saved job" : "Save job"}
                >
                  <Bookmark className={`h-4 w-4 ${isSaved ? "fill-current" : ""}`} />
                </button>
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
