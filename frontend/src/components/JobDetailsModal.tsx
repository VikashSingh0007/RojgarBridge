"use client";

import { useEffect, useMemo, useState } from "react";
import {
  Banknote,
  Bookmark,
  CalendarClock,
  Check,
  CheckCircle2,
  Clipboard,
  ExternalLink,
  GraduationCap,
  MapPin,
  MessageCircle,
  Users,
  WalletCards,
  X,
} from "lucide-react";
import { JobItem } from "@/types";
import { getDeadlineStatus, getPortalBadgeStyle } from "@/lib/formatters";

interface JobDetailsModalProps {
  job: JobItem | null;
  isBookmarked?: boolean;
  onToggleBookmark?: (id: number) => void;
  onClose: () => void;
}

function DetailSection({
  icon,
  title,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="grid gap-2.5 py-4 sm:grid-cols-[180px_1fr] sm:gap-6">
      <h3 className="flex items-center gap-2 text-xs font-bold text-slate-500">
        <span className="text-teal-700">{icon}</span>
        {title}
      </h3>
      <div className="min-w-0 text-sm font-semibold leading-6 text-slate-800">{children}</div>
    </section>
  );
}

export default function JobDetailsModal({
  job,
  isBookmarked = false,
  onToggleBookmark,
  onClose,
}: JobDetailsModalProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!job) return;

    const previousOverflow = document.body.style.overflow;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };

    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [job, onClose]);

  const conciseSummary = useMemo(() => {
    if (!job?.details.summary_points) return [];
    return job.details.summary_points
      .filter((point) => !/https?:\/\//i.test(point))
      .slice(0, 4);
  }, [job]);

  if (!job) return null;

  const d = job.details;
  const deadline = getDeadlineStatus(d.application_end);
  const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);
  const noticeUrl = d.official_notice_url;
  const primaryUrl = d.official_apply_url || noticeUrl;
  const hasSeparateNotice = Boolean(
    noticeUrl && d.official_apply_url && noticeUrl !== d.official_apply_url
  );
  const deadlineLabel = deadline.type === "unknown" ? "Not specified" : deadline.label;

  const handleCopyLink = () => {
    const url = typeof window !== "undefined" ? window.location.href : "";
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  const handleShareWhatsApp = () => {
    const text = `📢 *${d.title}*\n🏢 *Org:* ${d.organization}\n👤 *Vacancies:* ${
      d.total_vacancies > 0 ? d.total_vacancies : "Not specified"
    }\n⏰ *Last Date:* ${d.application_end || "Not specified"}\n🔗 *Apply:* ${primaryUrl || ""}\n\nShared via RojgarBridge`;
    window.open(`https://api.whatsapp.com/send?text=${encodeURIComponent(text)}`, "_blank");
  };

  return (
    <div
      className="job-drawer-backdrop fixed inset-0 z-50 !mt-0 bg-slate-950/45 backdrop-blur-[1px]"
      onMouseDown={onClose}
    >
      <aside
        className="job-drawer-enter absolute inset-y-0 right-0 flex w-full flex-col border-l border-slate-200 bg-white shadow-2xl sm:max-w-2xl lg:max-w-3xl"
        onMouseDown={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="job-drawer-title"
      >
        <header className="flex-none border-b border-slate-200 bg-white px-5 py-4 sm:px-6">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1">
              <h2
                id="job-drawer-title"
                className="max-w-2xl text-lg font-extrabold leading-[1.35] tracking-[-0.02em] text-slate-950 sm:text-xl"
              >
                {d.title}
              </h2>

              <div className="mt-2.5 flex flex-wrap items-center gap-1.5 text-xs text-slate-500">
                <span className="mr-1 inline-flex items-center gap-1.5 font-medium sm:text-sm">
                  <MapPin className="h-4 w-4" />
                  {d.state || "All India"}
                </span>
                {d.category ? (
                  <span className="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-semibold text-slate-600">
                    {d.category}
                  </span>
                ) : null}
                {d.notice_type ? (
                  <span className="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-semibold capitalize text-slate-600">
                    {d.notice_type.replaceAll("_", " ")}
                  </span>
                ) : null}
              </div>
            </div>

            <div className="flex flex-none items-center gap-1.5">
              <span
                className={`rounded-md border px-2 py-1 text-[10px] font-extrabold uppercase tracking-[0.08em] ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
              >
                {job.source_portal || d.organization}
              </span>
              {onToggleBookmark ? (
                <button
                  onClick={() => onToggleBookmark(job.id)}
                  className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                    isBookmarked
                      ? "bg-amber-50 text-amber-600"
                      : "text-slate-400 hover:bg-slate-100 hover:text-slate-700"
                  }`}
                  title={isBookmarked ? "Remove saved job" : "Save job"}
                  aria-label={isBookmarked ? "Remove saved job" : "Save job"}
                >
                  <Bookmark className={`h-4 w-4 ${isBookmarked ? "fill-current" : ""}`} />
                </button>
              ) : null}
              <button
                onClick={onClose}
                className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900"
                aria-label="Close job details"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto px-5 sm:px-6">
          <section className="grid grid-cols-2 gap-x-4 gap-y-4 border-b border-slate-200 py-5 sm:grid-cols-4">
            <div>
              <p className="mb-1.5 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Closing date
              </p>
              <span
                className={`inline-flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs font-extrabold ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
              >
                <CalendarClock className="h-3.5 w-3.5" />
                {deadlineLabel}
              </span>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Vacancies
              </p>
              <p className="text-sm font-extrabold text-slate-900">
                {d.total_vacancies > 0 ? d.total_vacancies.toLocaleString() : "Not specified"}
              </p>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Age limit
              </p>
              <p className="text-sm font-extrabold text-slate-900">
                {d.min_age && d.max_age ? `${d.min_age}–${d.max_age} years` : "Not specified"}
              </p>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Exam date
              </p>
              <p className="text-sm font-extrabold text-slate-900">
                {d.exam_date || "Not announced"}
              </p>
            </div>
          </section>

          <div className="divide-y divide-slate-100">
            <DetailSection icon={<GraduationCap className="h-4 w-4" />} title="Required qualification">
              {d.qualification || "Not specified in the notice"}
            </DetailSection>

            <DetailSection icon={<Banknote className="h-4 w-4" />} title="Pay scale and salary">
              {d.salary_payscale || "Not specified in the notice"}
            </DetailSection>

            <DetailSection icon={<WalletCards className="h-4 w-4" />} title="Application fee">
              <dl className="grid gap-3 sm:grid-cols-2">
                <div>
                  <dt className="text-[11px] font-medium text-slate-500">General / OBC / EWS</dt>
                  <dd className="mt-0.5 text-sm font-bold text-slate-900">
                    {d.fee_general || "Not specified"}
                  </dd>
                </div>
                <div>
                  <dt className="text-[11px] font-medium text-slate-500">SC / ST / PwD / Female</dt>
                  <dd className="mt-0.5 text-sm font-bold text-slate-900">
                    {d.fee_reserved || "Not specified"}
                  </dd>
                </div>
              </dl>
            </DetailSection>

            {conciseSummary.length > 0 ? (
              <DetailSection icon={<Users className="h-4 w-4" />} title="Key information">
                <ul className="space-y-2 font-medium text-slate-600">
                  {conciseSummary.map((point, index) => (
                    <li key={`${point}-${index}`} className="flex gap-2.5">
                      <CheckCircle2 className="mt-1 h-3.5 w-3.5 flex-none text-teal-600" />
                      <span>{point}</span>
                    </li>
                  ))}
                </ul>
              </DetailSection>
            ) : null}
          </div>

          <div className="flex flex-wrap items-center gap-2 border-t border-slate-100 py-4 text-xs">
            <button
              onClick={handleShareWhatsApp}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 font-semibold text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
            >
              <MessageCircle className="h-4 w-4" />
              Share
            </button>
            <button
              onClick={handleCopyLink}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 font-semibold text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
            >
              {copied ? <Check className="h-4 w-4" /> : <Clipboard className="h-4 w-4" />}
              {copied ? "Link copied" : "Copy link"}
            </button>
          </div>
        </div>

        <footer className="flex flex-none items-center justify-end gap-2 border-t border-slate-200 bg-white px-5 py-3 sm:px-6">
          {hasSeparateNotice ? (
            <a
              href={noticeUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 rounded-lg px-3.5 py-2.5 text-xs font-bold text-slate-700 transition-colors hover:bg-slate-100"
            >
              View official notice
              <ExternalLink className="h-3.5 w-3.5" />
            </a>
          ) : null}

          {primaryUrl ? (
            <a
              href={primaryUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 rounded-lg bg-slate-950 px-5 py-2.5 text-xs font-extrabold text-white shadow-sm transition-colors hover:bg-slate-800 sm:text-sm"
            >
              {d.official_apply_url ? "Apply on official website" : "Open official notification"}
              <ExternalLink className="h-4 w-4" />
            </a>
          ) : (
            <span className="text-xs font-semibold text-slate-500">Official link unavailable</span>
          )}
        </footer>
      </aside>
    </div>
  );
}
