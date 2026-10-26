"use client";

import { useState } from "react";
import { JobItem } from "@/types";
import { getDeadlineStatus, getPortalBadgeStyle } from "@/lib/formatters";

interface JobDetailsModalProps {
  job: JobItem | null;
  isBookmarked?: boolean;
  onToggleBookmark?: (id: number) => void;
  onClose: () => void;
}

export default function JobDetailsModal({
  job,
  isBookmarked = false,
  onToggleBookmark,
  onClose,
}: JobDetailsModalProps) {
  const [copied, setCopied] = useState(false);

  if (!job) return null;
  const d = job.details;
  const deadline = getDeadlineStatus(d.application_end);
  const portalStyle = getPortalBadgeStyle(job.source_portal || d.organization);

  const handleCopyLink = () => {
    const url = typeof window !== "undefined" ? window.location.href : "";
    navigator.clipboard.writeText(url);
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  const handleShareWhatsApp = () => {
    const text = `📢 *${d.title}*\n🏢 *Org:* ${d.organization}\n👤 *Vacancies:* ${
      d.total_vacancies > 0 ? d.total_vacancies : "Refer PDF"
    }\n⏰ *Last Date:* ${d.application_end || "Refer notice"}\n🔗 *Apply:* ${
      d.official_apply_url || d.official_notice_url || ""
    }\n\nShared via NaukriBridge`;
    window.open(`https://api.whatsapp.com/send?text=${encodeURIComponent(text)}`, "_blank");
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-slate-900/60 backdrop-blur-xs animate-in fade-in duration-200">
      <div className="bg-white rounded-3xl max-w-2xl w-full p-5 sm:p-7 shadow-2xl border border-slate-200 space-y-6 max-h-[92vh] overflow-y-auto">
        {/* Top Header */}
        <div className="flex items-start justify-between gap-4 border-b border-slate-100 pb-4">
          <div className="space-y-1.5 flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span
                className={`text-[11px] font-black uppercase tracking-wider px-2.5 py-0.5 rounded-md border ${portalStyle.bg} ${portalStyle.text} ${portalStyle.border}`}
              >
                {job.source_portal || d.organization}
              </span>
              <span className="text-xs font-semibold px-2.5 py-0.5 rounded-full bg-slate-100 text-slate-700 border border-slate-200">
                {d.category || "Central Govt"}
              </span>
              {d.notice_type && (
                <span className="text-xs font-semibold px-2.5 py-0.5 rounded-full bg-blue-50 text-blue-700 border border-blue-100 capitalize">
                  {d.notice_type.replace("_", " ")}
                </span>
              )}
              {d.confidence_score && (
                <span className="text-[11px] font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                  ✓ Official Gazette Verified
                </span>
              )}
            </div>

            <h2 className="text-lg sm:text-2xl font-black text-slate-900 leading-tight">
              {d.title}
            </h2>

            <p className="text-xs sm:text-sm text-slate-500 flex items-center gap-2">
              <span>🏢 {d.organization}</span>
              <span>•</span>
              <span>📍 {d.state || "All India"}</span>
              {d.advt_number && (
                <>
                  <span>•</span>
                  <span className="font-mono bg-slate-100 px-1.5 py-0.5 rounded text-[11px] text-slate-600">
                    Advt: {d.advt_number}
                  </span>
                </>
              )}
            </p>
          </div>

          <div className="flex items-center gap-1.5 flex-shrink-0">
            {/* Bookmark button */}
            {onToggleBookmark && (
              <button
                onClick={() => onToggleBookmark(job.id)}
                className={`p-2 rounded-xl border transition-colors ${
                  isBookmarked
                    ? "bg-amber-50 text-amber-500 border-amber-300"
                    : "bg-slate-50 text-slate-400 hover:text-amber-500 border-slate-200"
                }`}
                title={isBookmarked ? "Saved" : "Bookmark this job"}
                aria-label="Bookmark"
              >
                <svg
                  className="w-4 h-4 fill-current"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={isBookmarked ? 0 : 2}
                >
                  <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z" />
                </svg>
              </button>
            )}

            {/* Close Button */}
            <button
              onClick={onClose}
              className="w-8 h-8 rounded-full bg-slate-100 hover:bg-slate-200 text-slate-500 flex items-center justify-center font-bold text-sm transition-colors"
              aria-label="Close"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Highlights 4-box Grid */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-gradient-to-br from-slate-50 to-blue-50/30 rounded-2xl p-4 border border-slate-200/90 shadow-2xs">
          <div className="space-y-0.5">
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
              Total Vacancies
            </p>
            <p className="text-base sm:text-lg font-black text-slate-900">
              {d.total_vacancies > 0 ? `${d.total_vacancies.toLocaleString()} Posts` : "Refer PDF"}
            </p>
          </div>

          <div className="space-y-0.5">
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
              Application Deadline
            </p>
            <span
              className={`inline-block text-xs font-bold px-2 py-0.5 rounded-md border ${deadline.bgClass} ${deadline.textClass} ${deadline.borderClass}`}
            >
              {deadline.label}
            </span>
          </div>

          <div className="space-y-0.5">
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
              Age Limit
            </p>
            <p className="text-xs sm:text-sm font-bold text-slate-800">
              {d.min_age && d.max_age ? `${d.min_age} - ${d.max_age} Yrs` : "As per Govt Rules"}
            </p>
          </div>

          <div className="space-y-0.5">
            <p className="text-[10px] font-bold text-slate-400 uppercase tracking-wider">
              Exam Date
            </p>
            <p className="text-xs sm:text-sm font-bold text-slate-800 truncate" title={d.exam_date}>
              {d.exam_date || "To be announced"}
            </p>
          </div>
        </div>

        {/* Detailed Breakdown Sections */}
        <div className="space-y-4 text-xs sm:text-sm text-slate-700">
          {/* Pay Scale & Salary */}
          <div>
            <h4 className="font-bold text-slate-900 mb-1 flex items-center gap-1.5">
              <span>💰 Pay Scale & Salary</span>
            </h4>
            <p className="bg-emerald-50/70 text-emerald-900 px-3.5 py-2.5 rounded-xl border border-emerald-200 font-semibold">
              {d.salary_payscale || "As per 7th Central Pay Commission / State Pay Scale"}
            </p>
          </div>

          {/* Educational Qualification */}
          <div>
            <h4 className="font-bold text-slate-900 mb-1 flex items-center gap-1.5">
              <span>🎓 Required Qualification</span>
            </h4>
            <p className="text-slate-800 bg-slate-50 p-3.5 rounded-xl border border-slate-200 font-medium leading-relaxed">
              {d.qualification || "Please refer to the official notification document."}
            </p>
          </div>

          {/* Application Fee */}
          {(d.fee_general || d.fee_reserved) && (
            <div>
              <h4 className="font-bold text-slate-900 mb-1.5">💳 Application Fee Structure</h4>
              <div className="grid grid-cols-2 gap-3">
                <div className="bg-slate-50 p-3 rounded-xl border border-slate-200">
                  <span className="text-[11px] text-slate-500 font-medium block">
                    General / OBC / EWS
                  </span>
                  <span className="text-sm font-bold text-slate-900">
                    {d.fee_general || "Refer notice"}
                  </span>
                </div>
                <div className="bg-slate-50 p-3 rounded-xl border border-slate-200">
                  <span className="text-[11px] text-slate-500 font-medium block">
                    SC / ST / PwD / Female
                  </span>
                  <span className="text-sm font-bold text-slate-900">
                    {d.fee_reserved || "Refer notice"}
                  </span>
                </div>
              </div>
            </div>
          )}

          {/* Key Summary Highlights */}
          {d.summary_points && d.summary_points.length > 0 && (
            <div>
              <h4 className="font-bold text-slate-900 mb-1.5">📌 Key Highlights & Instructions</h4>
              <ul className="list-disc list-inside space-y-1.5 text-slate-600 bg-blue-50/40 p-4 rounded-xl border border-blue-100">
                {d.summary_points.map((pt, idx) => (
                  <li key={idx} className="leading-relaxed">
                    {pt}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {/* Share & Quick Utility Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-slate-100 text-xs">
          <div className="flex items-center gap-2">
            {/* Share WhatsApp */}
            <button
              onClick={handleShareWhatsApp}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-50 hover:bg-emerald-100 text-emerald-700 border border-emerald-200 font-semibold transition-colors"
            >
              <span>💬 WhatsApp</span>
            </button>

            {/* Copy Link */}
            <button
              onClick={handleCopyLink}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold transition-colors"
            >
              {copied ? "✓ Link Copied!" : "📋 Copy Link"}
            </button>
          </div>

          {d.official_notice_url && (
            <a
              href={d.official_notice_url}
              target="_blank"
              rel="noopener noreferrer"
              className="font-bold text-blue-600 hover:text-blue-800 underline flex items-center gap-1"
            >
              <span>📄 Official Notification PDF ↗</span>
            </a>
          )}
        </div>

        {/* Action Footer */}
        <div className="flex items-center justify-end gap-3 pt-3 border-t border-slate-100">
          <button
            onClick={onClose}
            className="px-5 py-2.5 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-xl transition-colors"
          >
            Close
          </button>

          <a
            href={d.official_apply_url || d.official_notice_url || "#"}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-2.5 text-xs sm:text-sm font-black text-white bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-700 hover:to-indigo-700 rounded-xl shadow-md hover:shadow-lg transition-all"
          >
            <span>Apply on Official Portal</span>
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M14 5l7 7m0 0l-7 7m7-7H3" />
            </svg>
          </a>
        </div>
      </div>
    </div>
  );
}
