"use client";

import { useState, useMemo } from "react";
import {
  Calendar,
  CheckCircle2,
  ExternalLink,
  FileCheck2,
  FileText,
  KeyRound,
  Megaphone,
  Search,
  Sparkles,
  Trophy,
} from "lucide-react";
import { JobItem } from "@/types";
import {
  getAnnouncementTypeInfo,
  getOfficialNotificationUrl,
  getOfficialApplicationUrl,
  getPortalBadgeStyle,
  getRelatedRecruitmentName,
  isOfficialGovUrl,
} from "@/lib/formatters";

interface AnnouncementsViewProps {
  announcements: JobItem[];
  activeCategory: string; // "all" | "admit_card" | "result" | "interview_schedule" | "answer_key"
  onChangeCategory: (category: string) => void;
  onSelectAnnouncement: (job: JobItem) => void;
  isLoading?: boolean;
}

const CATEGORY_TABS = [
  { id: "all", label: "All Updates", icon: Megaphone },
  { id: "admit_card", label: "Admit Cards", icon: FileCheck2 },
  { id: "result", label: "Results", icon: Trophy },
  { id: "interview_schedule", label: "Exam Dates", icon: Calendar },
  { id: "answer_key", label: "Answer Keys", icon: KeyRound },
] as const;

export default function AnnouncementsView({
  announcements,
  activeCategory,
  onChangeCategory,
  onSelectAnnouncement,
  isLoading = false,
}: AnnouncementsViewProps) {
  const [searchQuery, setSearchQuery] = useState("");

  const filtered = useMemo(() => {
    let list = announcements;

    // Filter by category tab
    if (activeCategory && activeCategory !== "all") {
      list = list.filter((item) => {
        const type = (item.details.notice_type || "").toLowerCase();
        if (activeCategory === "admit_card") return type.includes("admit");
        if (activeCategory === "result") return type.includes("result");
        if (activeCategory === "interview_schedule")
          return type.includes("interview") || type.includes("schedule") || type.includes("exam");
        if (activeCategory === "answer_key") return type.includes("answer");
        return true;
      });
    }

    // Filter by search query
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase().trim();
      list = list.filter((item) => {
        const title = item.details.title.toLowerCase();
        const org = (item.details.organization || item.source_portal || "").toLowerCase();
        const advt = (item.details.advt_number || "").toLowerCase();
        return title.includes(q) || org.includes(q) || advt.includes(q);
      });
    }

    return list;
  }, [announcements, activeCategory, searchQuery]);

  return (
    <div className="space-y-6">
      {/* Top Section Header */}
      <div className="rounded-2xl border border-slate-200/90 bg-white p-5 sm:p-6 shadow-xs space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-teal-50 text-teal-700">
                <Megaphone className="h-4 w-4" />
              </span>
              <h2 className="text-base sm:text-lg font-extrabold text-slate-900">
                Government Recruitment Announcements
              </h2>
            </div>
            <p className="mt-1 text-xs text-slate-500">
              Verified admit cards, exam dates, answer keys, results, and official corrigendums
            </p>
          </div>

          {/* Quick Search */}
          <div className="relative w-full sm:w-72">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Filter announcements…"
              className="w-full rounded-xl border border-slate-200 bg-slate-50/80 py-2 pl-9 pr-3 text-xs font-semibold text-slate-800 placeholder-slate-400 focus:border-teal-600 focus:bg-white focus:outline-hidden"
            />
          </div>
        </div>

        {/* Category Filter Pills */}
        <div className="flex flex-wrap items-center gap-2 border-t border-slate-100 pt-3">
          {CATEGORY_TABS.map(({ id, label, icon: Icon }) => {
            const isActive = activeCategory === id;
            return (
              <button
                key={id}
                onClick={() => onChangeCategory(id)}
                className={`inline-flex items-center gap-1.5 rounded-lg px-3.5 py-1.5 text-xs font-bold transition-all ${
                  isActive
                    ? "bg-slate-950 text-white shadow-xs"
                    : "border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50"
                }`}
              >
                <Icon className={`h-3.5 w-3.5 ${isActive ? "text-teal-400" : "text-slate-400"}`} />
                <span>{label}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Announcements Cards Grid */}
      {isLoading && announcements.length === 0 ? (
        <div className="rounded-2xl border border-slate-200 bg-white p-12 text-center space-y-3">
          <div className="h-8 w-8 animate-spin rounded-full border-3 border-teal-700 border-t-transparent mx-auto" />
          <p className="text-sm font-bold text-slate-700">Loading verified announcements…</p>
        </div>
      ) : filtered.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((item) => {
            const d = item.details;
            const typeInfo = getAnnouncementTypeInfo(d.notice_type);
            const portalStyle = getPortalBadgeStyle(item.source_portal || d.organization);
            const relatedExam = getRelatedRecruitmentName(d.title);
            const noticeUrl = getOfficialNotificationUrl(item);
            const applyUrl = getOfficialApplicationUrl(item);
            const primaryDocUrl = noticeUrl || applyUrl;
            const isGovDoc = isOfficialGovUrl(primaryDocUrl || undefined);

            const displayDate =
              d.exam_date ||
              d.application_end ||
              (item.raw_notice?.published_at
                ? item.raw_notice.published_at
                : new Date(item.created_at).toLocaleDateString("en-IN", {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                  }));

            return (
              <article
                key={item.id}
                onClick={() => onSelectAnnouncement(item)}
                className="group relative flex flex-col justify-between rounded-2xl border border-slate-200/90 bg-white p-5 shadow-xs transition-all duration-200 hover:border-teal-600/70 hover:shadow-md cursor-pointer"
              >
                <div className="space-y-3">
                  {/* Top Badges Row */}
                  <div className="flex items-center justify-between gap-2 flex-wrap">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      {/* Announcement Type Badge */}
                      <span
                        className={`inline-flex items-center gap-1 rounded-md border px-2.5 py-0.5 text-[10px] font-black uppercase tracking-wider ${typeInfo.bg} ${typeInfo.text} ${typeInfo.border}`}
                      >
                        {typeInfo.label}
                      </span>

                      {/* Recruiting Authority Badge */}
                      <span
                        className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-extrabold uppercase tracking-wide ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
                      >
                        {item.source_portal || d.organization}
                      </span>
                    </div>

                    {/* Status Pill */}
                    <span className="inline-flex items-center gap-1 text-[10px] font-bold text-emerald-700 bg-emerald-50 border border-emerald-100 rounded-full px-2 py-0.5">
                      <Sparkles className="h-3 w-3" />
                      <span>{typeInfo.tag.split(" ")[0]} Active</span>
                    </span>
                  </div>

                  {/* Related Recruitment Link */}
                  <div className="rounded-lg bg-slate-50/90 border border-slate-200/70 px-2.5 py-1.5 text-[11px] text-slate-600">
                    <span className="font-semibold text-slate-400 block text-[9px] uppercase tracking-wider">
                      Related Recruitment
                    </span>
                    <span className="font-bold text-slate-800 line-clamp-1" title={relatedExam}>
                      🏛️ {relatedExam}
                    </span>
                  </div>

                  {/* Announcement Title */}
                  <h3 className="text-xs sm:text-sm font-extrabold leading-[1.4] text-slate-900 group-hover:text-teal-800 transition-colors line-clamp-2">
                    {d.title}
                  </h3>

                  {/* Metadata Row */}
                  <div className="flex items-center gap-3 text-xs text-slate-500 pt-1">
                    <span className="inline-flex items-center gap-1">
                      <Calendar className="h-3.5 w-3.5 text-slate-400" />
                      <span>{displayDate}</span>
                    </span>
                    {d.advt_number && (
                      <span className="truncate text-slate-400 text-[11px]" title={d.advt_number}>
                        • Ref: {d.advt_number}
                      </span>
                    )}
                  </div>
                </div>

                {/* Differentiated Bottom Actions */}
                <div className="mt-4 grid grid-cols-2 gap-2 pt-3 border-t border-slate-100">
                  {/* View Details Button */}
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelectAnnouncement(item);
                    }}
                    className="inline-flex items-center justify-center gap-1 rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-xs font-bold text-slate-700 hover:bg-slate-50 shadow-2xs transition-colors"
                  >
                    <span>View Details</span>
                  </button>

                  {/* Official PDF / Download Link */}
                  {primaryDocUrl ? (
                    <a
                      href={primaryDocUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      onClick={(e) => e.stopPropagation()}
                      className="inline-flex items-center justify-center gap-1.5 rounded-lg bg-slate-950 px-2.5 py-2 text-xs font-extrabold text-white hover:bg-slate-800 shadow-2xs transition-colors"
                      title={
                        isGovDoc
                          ? "Open official document on official government portal"
                          : "Open announcement link"
                      }
                    >
                      <FileText className="h-3.5 w-3.5 text-teal-400" />
                      <span>Official Doc</span>
                      <ExternalLink className="h-3 w-3 opacity-70" />
                    </a>
                  ) : (
                    <span
                      onClick={(e) => e.stopPropagation()}
                      className="inline-flex items-center justify-center rounded-lg border border-slate-200 bg-slate-50 px-2.5 py-2 text-xs font-medium text-slate-400 cursor-default"
                    >
                      Doc Pending
                    </span>
                  )}
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        /* Empty State */
        <div className="rounded-2xl border border-dashed border-slate-300 bg-white p-12 text-center space-y-3">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-teal-50 text-teal-700">
            <Megaphone className="h-6 w-6" />
          </div>
          <h3 className="text-base font-extrabold text-slate-800">
            No Announcements Found in This Category
          </h3>
          <p className="mx-auto max-w-sm text-xs text-slate-500">
            Try switching categories or clearing search keywords to view all official government announcements.
          </p>
          <button
            onClick={() => {
              onChangeCategory("all");
              setSearchQuery("");
            }}
            className="mt-2 rounded-xl bg-teal-700 px-4 py-2 text-xs font-bold text-white hover:bg-teal-800"
          >
            Show All Updates
          </button>
        </div>
      )}
    </div>
  );
}

