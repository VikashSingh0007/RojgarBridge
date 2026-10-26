"use client";

import { Bookmark, GraduationCap, MapPin, Users } from "lucide-react";
import { JobItem } from "@/types";
import { getCompactDeadlineLabel, getDeadlineStatus, getShortQualification } from "@/lib/formatters";

interface JobTableViewProps {
  jobs: JobItem[];
  savedJobIds: number[];
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
}

export default function JobTableView({ jobs, savedJobIds, onToggleBookmark, onSelectJob }: JobTableViewProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="divide-y divide-slate-100">
        {jobs.map((job) => {
          const d = job.details;
          const deadline = getDeadlineStatus(d.application_end);
          const isSaved = savedJobIds.includes(job.id);
          const location = d.state && d.state.toLowerCase() !== "all india" ? d.state : null;

          return (
            <article
              key={job.id}
              onClick={() => onSelectJob(job)}
              className="group grid cursor-pointer gap-4 px-5 py-4 transition-colors hover:bg-slate-50 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
            >
              <div className="min-w-0">
                <div className="mb-1.5 flex items-center gap-2">
                  <span className="rounded-md bg-teal-50 px-2 py-1 text-xs font-bold text-teal-800 ring-1 ring-inset ring-teal-200">
                    {job.source_portal || d.organization}
                  </span>
                </div>

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
                <span className={`whitespace-nowrap text-sm font-bold ${deadline.textClass}`}>
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
