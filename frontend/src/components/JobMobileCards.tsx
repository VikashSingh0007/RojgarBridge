"use client";

import { Bookmark, GraduationCap, MapPin, Users } from "lucide-react";
import { JobItem } from "@/types";
import { getCompactDeadlineLabel, getDeadlineStatus, getShortQualification } from "@/lib/formatters";

interface JobMobileCardsProps {
  jobs: JobItem[];
  savedJobIds: number[];
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
}

export default function JobMobileCards({ jobs, savedJobIds, onToggleBookmark, onSelectJob }: JobMobileCardsProps) {
  return (
    <div className="space-y-3">
      {jobs.map((job) => {
        const d = job.details;
        const isSaved = savedJobIds.includes(job.id);
        const deadline = getDeadlineStatus(d.application_end);
        const location = d.state && d.state.toLowerCase() !== "all india" ? d.state : null;

        return (
          <article key={job.id} onClick={() => onSelectJob(job)} className="cursor-pointer rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-center justify-between gap-3">
              <span className="rounded-md bg-teal-50 px-2 py-1 text-xs font-bold text-teal-800 ring-1 ring-inset ring-teal-200">
                {job.source_portal || d.organization}
              </span>
              <span className={`text-xs font-bold ${deadline.textClass}`}>{getCompactDeadlineLabel(deadline.label)}</span>
            </div>

            <h3 className="mt-2.5 text-base font-bold leading-snug text-slate-950">{d.title}</h3>

            <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-slate-600">
              <span className="inline-flex items-center gap-1.5"><GraduationCap className="h-4 w-4 text-slate-400" />{getShortQualification(d.qualification)}</span>
              <span className="inline-flex items-center gap-1.5"><Users className="h-4 w-4 text-slate-400" />{d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} vacancies` : "Vacancies in notice"}</span>
              {location && <span className="inline-flex items-center gap-1.5"><MapPin className="h-4 w-4 text-slate-400" />{location}</span>}
            </div>

            <div className="mt-3 flex justify-end border-t border-slate-100 pt-2">
              <button
                onClick={(event) => { event.stopPropagation(); onToggleBookmark(job.id); }}
                className={`inline-flex min-h-9 items-center gap-1.5 rounded-lg px-2.5 text-xs font-semibold ${isSaved ? "bg-amber-50 text-amber-700" : "text-slate-500 hover:bg-slate-50"}`}
              >
                <Bookmark className={`h-3.5 w-3.5 ${isSaved ? "fill-current" : ""}`} />{isSaved ? "Saved" : "Save"}
              </button>
            </div>
          </article>
        );
      })}
    </div>
  );
}
