"use client";

import { JobItem } from "@/types";
import { getDeadlineStatus, getPortalBadgeStyle } from "@/lib/formatters";

interface JobTableViewProps {
  jobs: JobItem[];
  savedJobIds: number[];
  onToggleBookmark: (id: number) => void;
  onSelectJob: (job: JobItem) => void;
  onShareWhatsApp: (job: JobItem) => void;
}

export default function JobTableView({
  jobs,
  savedJobIds,
  onToggleBookmark,
  onSelectJob,
  onShareWhatsApp,
}: JobTableViewProps) {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/90 shadow-sm overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50/80 text-[11px] font-black text-slate-500 uppercase tracking-wider">
              <th className="py-3 px-3 text-center w-10">★</th>
              <th className="py-3 px-4">Notification & Department</th>
              <th className="py-3 px-4">Organization</th>
              <th className="py-3 px-4 text-center">Vacancies</th>
              <th className="py-3 px-4">Qualification</th>
              <th className="py-3 px-4">Deadline Status</th>
              <th className="py-3 px-5 text-right">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 text-sm">
            {jobs.map((job) => {
              const d = job.details;
              const isSaved = savedJobIds.includes(job.id);
              const deadline = getDeadlineStatus(d.application_end);
              const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);

              return (
                <tr
                  key={job.id}
                  className="hover:bg-blue-50/30 transition-colors group cursor-pointer"
                  onClick={() => onSelectJob(job)}
                >
                  {/* Star Bookmark Column */}
                  <td className="py-4 px-3 align-top text-center">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onToggleBookmark(job.id);
                      }}
                      className={`p-1 rounded-md transition-colors ${
                        isSaved
                          ? "text-amber-500 hover:text-amber-600"
                          : "text-slate-300 hover:text-amber-400 group-hover:text-slate-400"
                      }`}
                      title={isSaved ? "Remove from saved" : "Bookmark this notice"}
                      aria-label="Bookmark"
                    >
                      <svg
                        className="w-4 h-4 fill-current"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={isSaved ? 0 : 2}
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
                        />
                      </svg>
                    </button>
                  </td>

                  {/* Title + Advt + Category Tag */}
                  <td className="py-4 px-4 align-top max-w-sm">
                    <div className="space-y-1">
                      <div className="flex items-center gap-1.5 flex-wrap">
                        <span
                          className={`text-[10px] font-black uppercase tracking-wider px-2 py-0.5 rounded border ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
                        >
                          {job.source_portal || d.organization}
                        </span>
                        <span className="inline-block text-[10px] font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
                          {d.category || "Central Govt"}
                        </span>
                        {d.notice_type && (
                          <span className="inline-block text-[10px] font-semibold px-2 py-0.5 rounded-full bg-blue-50 text-blue-700 border border-blue-100 capitalize">
                            {d.notice_type.replace("_", " ")}
                          </span>
                        )}
                      </div>

                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectJob(job);
                        }}
                        className="font-bold text-slate-900 group-hover:text-blue-600 hover:underline text-left block leading-snug text-[13px] sm:text-sm pt-0.5"
                      >
                        {d.title}
                      </button>

                      {d.advt_number && (
                        <p className="text-[11px] text-slate-400 font-mono">
                          Advt: {d.advt_number}
                        </p>
                      )}
                    </div>
                  </td>

                  {/* Organization & Location */}
                  <td className="py-4 px-4 align-top text-xs font-medium text-slate-700">
                    <div className="space-y-0.5">
                      <p className="font-semibold text-slate-800">{d.organization}</p>
                      <p className="text-[11px] text-slate-500">📍 {d.state || "All India"}</p>
                    </div>
                  </td>

                  {/* Vacancies / Posts */}
                  <td className="py-4 px-4 align-top text-center text-xs">
                    <span className="font-bold text-slate-900 bg-slate-100 px-2.5 py-1 rounded-lg border border-slate-200 block text-center">
                      {d.total_vacancies > 0 ? d.total_vacancies.toLocaleString() : "Refer PDF"}
                    </span>
                  </td>

                  {/* Qualification */}
                  <td
                    className="py-4 px-4 align-top text-xs text-slate-600 max-w-[180px] truncate"
                    title={d.qualification}
                  >
                    <span className="truncate block font-medium">
                      🎓 {d.qualification || "Refer notification"}
                    </span>
                  </td>

                  {/* Deadline Status Badge */}
                  <td className="py-4 px-4 align-top whitespace-nowrap">
                    <span
                      className={`inline-flex items-center gap-1 text-[11px] font-bold px-2.5 py-1 rounded-lg border ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
                    >
                      {deadline.label}
                    </span>
                  </td>

                  {/* Action Buttons */}
                  <td className="py-4 px-5 align-top text-right whitespace-nowrap">
                    <div className="flex items-center justify-end gap-1.5">
                      {/* WhatsApp Share Button */}
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onShareWhatsApp(job);
                        }}
                        className="p-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-600 border border-emerald-200 transition-colors"
                        title="Share on WhatsApp"
                        aria-label="Share WhatsApp"
                      >
                        <svg className="w-3.5 h-3.5 fill-current" viewBox="0 0 24 24">
                          <path d="M12.031 6.172c-3.181 0-5.767 2.586-5.768 5.766-.001 1.298.38 2.27 1.019 3.287l-.582 2.128 2.182-.573c.978.58 1.911.928 3.145.929 3.178 0 5.767-2.587 5.768-5.766.001-3.187-2.575-5.77-5.764-5.771zm3.392 8.244c-.144.405-.837.774-1.17.824-.299.045-.677.063-1.092-.069-.252-.08-.575-.187-.988-.365-1.739-.751-2.874-2.502-2.961-2.617-.087-.116-.708-.94-.708-1.793s.448-1.273.607-1.446c.159-.173.346-.217.462-.217l.332.006c.106.005.249-.04.39.298.144.347.491 1.2.534 1.287.043.087.072.188.014.304-.058.116-.087.188-.173.289l-.26.304c-.087.086-.177.18-.076.354.101.174.449.741.964 1.201.662.591 1.221.774 1.394.86.174.086.275.073.376-.044.101-.116.433-.506.549-.68.116-.173.231-.145.39-.086s1.011.477 1.184.564.289.13.332.202c.045.072.045.419-.1.824z" />
                        </svg>
                      </button>

                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectJob(job);
                        }}
                        className="inline-flex items-center text-xs font-bold px-3 py-1.5 rounded-lg bg-blue-50 text-blue-600 hover:bg-blue-600 hover:text-white border border-blue-200 transition-all shadow-2xs"
                      >
                        View Details
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
