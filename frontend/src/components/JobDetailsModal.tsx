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
  FileText,
  GraduationCap,
  MapPin,
  MessageCircle,
  ShieldCheck,
  Users,
  WalletCards,
  X,
} from "lucide-react";
import { JobItem } from "@/types";
import {
  getDeadlineStatus,
  getOfficialApplicationUrl,
  getOfficialNotificationUrl,
  getPortalBadgeStyle,
  isOfficialGovUrl,
} from "@/lib/formatters";

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
    <section className="grid gap-2 py-3.5 sm:grid-cols-[160px_1fr] sm:gap-4">
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
      .slice(0, 5);
  }, [job]);

  if (!job) return null;

  const d = job.details;
  const deadline = getDeadlineStatus(d.application_end);
  const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);
  const applyUrl = getOfficialApplicationUrl(job);
  const noticeUrl = getOfficialNotificationUrl(job);
  const isGovApply = isOfficialGovUrl(applyUrl || undefined);
  const deadlineLabel = deadline.type === "unknown" ? "Refer notification" : deadline.label;

  const handleCopyLink = () => {
    const url = typeof window !== "undefined" ? window.location.href : "";
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  const handleShareWhatsApp = () => {
    const text = `📢 *${d.title}*\n🏢 *Org:* ${d.organization}\n👤 *Vacancies:* ${
      d.total_vacancies > 0 ? d.total_vacancies.toLocaleString() : "Refer notice"
    }\n⏰ *Last Date:* ${d.application_end || "Refer notice"}\n🔗 *Apply:* ${applyUrl || ""}\n📄 *Notification:* ${noticeUrl || ""}\n\nShared via RojgarBridge`;
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
                className="max-w-2xl text-xs font-extrabold leading-[1.5] text-slate-950 sm:text-sm"
              >
                {d.title}
              </h2>

              <div className="mt-2.5 flex flex-wrap items-center gap-1.5 text-xs text-slate-500">
                <span className="mr-1 inline-flex items-center gap-1.5 font-medium sm:text-sm">
                  <MapPin className="h-4 w-4 text-slate-400" />
                  {d.state || "All India"}
                </span>
                {d.category ? (
                  <span className="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-semibold text-slate-600">
                    {d.category}
                  </span>
                ) : null}
                {d.notice_type ? (
                  <span className="rounded-full bg-teal-50 px-2.5 py-1 text-[11px] font-semibold capitalize text-teal-800 border border-teal-100">
                    {d.notice_type.replaceAll("_", " ")}
                  </span>
                ) : null}
                {d.advt_number && (
                  <span className="rounded-full bg-slate-50 px-2.5 py-1 text-[11px] font-medium text-slate-500 border border-slate-200">
                    Advt: {d.advt_number}
                  </span>
                )}
              </div>
            </div>

            <div className="flex flex-none items-center gap-1.5">
              <span
                className={`inline-flex h-8 items-center rounded-md border px-2 text-[10px] font-extrabold uppercase tracking-[0.08em] ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
              >
                {job.source_portal || d.organization}
              </span>
              {onToggleBookmark ? (
                <button
                  onClick={() => onToggleBookmark(job.id)}
                  className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                    isBookmarked
                      ? "bg-amber-50 text-amber-600 border border-amber-200"
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
          {/* Top Quick Status Metric Bar */}
          <section className="grid grid-cols-2 gap-x-4 gap-y-3 border-b border-slate-200 py-4 sm:grid-cols-4">
            <div>
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Closing date
              </p>
              <span
                className={`inline-flex min-h-8 items-center gap-1.5 rounded-lg border px-2.5 text-xs font-extrabold ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
              >
                <CalendarClock className="h-3.5 w-3.5" />
                {deadlineLabel}
              </span>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Vacancies
              </p>
              <p className="flex min-h-8 items-center text-sm font-extrabold text-slate-900">
                {d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} Posts` : "Refer PDF"}
              </p>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Age limit
              </p>
              <p className="flex min-h-8 items-center text-sm font-extrabold text-slate-900">
                {d.min_age && d.max_age
                  ? `${d.min_age}–${d.max_age} years`
                  : d.min_age
                  ? `Min ${d.min_age} years`
                  : d.max_age
                  ? `Max ${d.max_age} years`
                  : "As per rules"}
              </p>
            </div>
            <div className="sm:border-l sm:border-slate-200 sm:pl-4">
              <p className="mb-1 text-[10px] font-bold uppercase tracking-[0.08em] text-slate-400">
                Exam date
              </p>
              <p className="flex min-h-8 items-center text-sm font-extrabold text-slate-900">
                {d.exam_date || "To be announced"}
              </p>
            </div>
          </section>

          {/* Detailed Specifications */}
          <div className="divide-y divide-slate-100">
            <DetailSection icon={<GraduationCap className="h-4 w-4" />} title="Eligibility criteria">
              {d.qualification || "Refer to the official recruitment notification"}
            </DetailSection>

            <DetailSection icon={<Banknote className="h-4 w-4" />} title="Pay scale & salary">
              {d.salary_payscale || "As per 7th CPC / State Government norms"}
            </DetailSection>

            <DetailSection icon={<WalletCards className="h-4 w-4" />} title="Application fee">
              <dl className="grid gap-3 sm:grid-cols-2">
                <div>
                  <dt className="text-[11px] font-medium text-slate-500">General / OBC / EWS</dt>
                  <dd className="mt-0.5 text-sm font-bold text-slate-900">
                    {d.fee_general || "Refer notification"}
                  </dd>
                </div>
                <div>
                  <dt className="text-[11px] font-medium text-slate-500">SC / ST / PwD / Female</dt>
                  <dd className="mt-0.5 text-sm font-bold text-slate-900">
                    {d.fee_reserved || "Exempted / As per rules"}
                  </dd>
                </div>
              </dl>
            </DetailSection>

            {conciseSummary.length > 0 ? (
              <DetailSection icon={<Users className="h-4 w-4" />} title="Notice summary">
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

            {/* Official Verification Notice Callout */}
            <div className="py-4 my-2 rounded-xl bg-slate-50 border border-slate-200 p-4 text-xs text-slate-600 space-y-1">
              <div className="flex items-center gap-1.5 font-bold text-slate-800">
                <ShieldCheck className="h-4 w-4 text-teal-700" />
                <span>Official Government Recruitment Guarantee</span>
              </div>
              <p>
                RojgarBridge extracts and verifies openings directly from official government portals.
                Candidates are strongly advised to read the official notification PDF before submitting applications.
              </p>
            </div>
          </div>
        </div>

        {/* Action Bar Footer */}
        <footer className="flex flex-none flex-wrap items-center justify-between gap-3 border-t border-slate-200 bg-white px-5 py-3.5 sm:px-6">
          <div className="flex items-center gap-1 text-xs">
            <button
              onClick={handleShareWhatsApp}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 font-semibold text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
            >
              <MessageCircle className="h-4 w-4 text-emerald-600" />
              <span>Share</span>
            </button>
            <button
              onClick={handleCopyLink}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-2 font-semibold text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
            >
              {copied ? <Check className="h-4 w-4 text-teal-700" /> : <Clipboard className="h-4 w-4" />}
              <span>{copied ? "Copied" : "Copy"}</span>
            </button>
          </div>

          <div className="ml-auto flex flex-wrap items-center gap-2">
            {/* View Official Notification PDF (Independent Action) */}
            {noticeUrl ? (
              <a
                href={noticeUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 rounded-lg border border-slate-300 bg-white px-3.5 py-2 text-xs font-bold text-slate-700 shadow-2xs transition-colors hover:bg-slate-100 hover:text-slate-950 sm:text-sm"
                title="Open official recruitment notification document (PDF)"
              >
                <FileText className="h-4 w-4 text-slate-500" />
                <span>View Notification</span>
                <ExternalLink className="h-3.5 w-3.5 text-slate-400" />
              </a>
            ) : (
              <span className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-medium text-slate-400">
                <FileText className="h-4 w-4 opacity-50" />
                Notification Pending
              </span>
            )}

            {/* Apply Online (Redirects to official portal in new tab) */}
            {applyUrl ? (
              <a
                href={applyUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-2 rounded-lg bg-slate-950 px-4 py-2 text-xs font-extrabold text-white shadow-2xs transition-all hover:bg-slate-800 sm:text-sm"
                title={isGovApply ? "Apply directly on official government website" : "Apply online"}
              >
                <span>Apply Online</span>
                {isGovApply && (
                  <span className="rounded bg-teal-800/80 px-1.5 py-0.5 text-[9px] font-extrabold uppercase tracking-wider text-teal-200">
                    Official
                  </span>
                )}
                <ExternalLink className="h-4 w-4" />
              </a>
            ) : (
              <span className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-medium text-slate-400">
                Apply link in notice
              </span>
            )}
          </div>
        </footer>
      </aside>
    </div>
  );
}
