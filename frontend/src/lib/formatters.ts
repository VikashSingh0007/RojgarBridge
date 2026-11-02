export interface DeadlineStatus {
  type: "urgent" | "warning" | "normal" | "expired" | "unknown";
  label: string;
  bgClass: string;
  textClass: string;
  borderClass: string;
  daysRemaining?: number;
}

export function getDeadlineStatus(endDateStr?: string): DeadlineStatus {
  if (!endDateStr || endDateStr.toLowerCase().includes("refer") || endDateStr.toLowerCase().includes("announced")) {
    return {
      type: "unknown",
      label: "Refer Notice",
      bgClass: "bg-slate-50",
      textClass: "text-slate-600",
      borderClass: "border-slate-200",
    };
  }

  const end = new Date(endDateStr);
  if (isNaN(end.getTime())) {
    return {
      type: "unknown",
      label: endDateStr,
      bgClass: "bg-slate-50",
      textClass: "text-slate-600",
      borderClass: "border-slate-200",
    };
  }

  const now = new Date();
  // Difference in calendar days
  const diffTime = end.getTime() - now.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays < 0) {
    return {
      type: "expired",
      label: "Closed",
      bgClass: "bg-zinc-100",
      textClass: "text-zinc-500 line-through",
      borderClass: "border-zinc-200",
      daysRemaining: diffDays,
    };
  }

  if (diffDays === 0) {
    return {
      type: "urgent",
      label: "🚨 Closes Today!",
      bgClass: "bg-rose-50",
      textClass: "text-rose-700 font-bold animate-pulse",
      borderClass: "border-rose-200",
      daysRemaining: 0,
    };
  }

  if (diffDays === 1) {
    return {
      type: "urgent",
      label: "⚠️ Closes Tomorrow",
      bgClass: "bg-rose-50",
      textClass: "text-rose-700 font-bold",
      borderClass: "border-rose-200",
      daysRemaining: 1,
    };
  }

  if (diffDays <= 7) {
    return {
      type: "warning",
      label: `⏰ ${diffDays} Days Left`,
      bgClass: "bg-amber-50",
      textClass: "text-amber-700 font-semibold",
      borderClass: "border-amber-200",
      daysRemaining: diffDays,
    };
  }

  // Format date e.g. "24 Nov 2026"
  const formatted = end.toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
  });

  return {
    type: "normal",
    label: `Ends ${formatted}`,
    bgClass: "bg-blue-50/70",
    textClass: "text-blue-700 font-medium",
    borderClass: "border-blue-100",
    daysRemaining: diffDays,
  };
}

export function getPortalBadgeStyle(portal: string): { bg: string; text: string; border: string } {
  const p = (portal || "").toUpperCase();
  switch (p) {
    case "UPSC":
      return { bg: "bg-indigo-50", text: "text-indigo-700", border: "border-indigo-200" };
    case "SSC":
      return { bg: "bg-blue-50", text: "text-blue-700", border: "border-blue-200" };
    case "ISRO":
      return { bg: "bg-cyan-50", text: "text-cyan-700", border: "border-cyan-200" };
    case "IBPS":
      return { bg: "bg-emerald-50", text: "text-emerald-700", border: "border-emerald-200" };
    case "DSSSB":
      return { bg: "bg-amber-50", text: "text-amber-700", border: "border-amber-200" };
    case "UPPSC":
      return { bg: "bg-orange-50", text: "text-orange-700", border: "border-orange-200" };
    case "BPSC":
      return { bg: "bg-teal-50", text: "text-teal-700", border: "border-teal-200" };
    default:
      return { bg: "bg-slate-100", text: "text-slate-700", border: "border-slate-200" };
  }
}

export function getShortQualification(qualification?: string): string {
  if (!qualification) return "See official notice";
  const value = qualification.toLowerCase();
  if (value.includes("post graduate") || value.includes("master")) return "Post Graduate";
  if (value.includes("b.e") || value.includes("b.tech") || value.includes("engineering")) return "Engineering Graduate";
  if (value.includes("12th")) return "12th Pass";
  if (value.includes("10th")) return "10th Pass";
  if (value.includes("diploma")) return "Diploma";
  if (value.includes("iti")) return "ITI";
  if (value.includes("bachelor") || value.includes("graduate")) return "Graduate";
  return qualification.length > 32 ? `${qualification.slice(0, 29).trim()}…` : qualification;
}

export function getCompactDeadlineLabel(label: string): string {
  return label
    .replace(/[🚨⚠️⏰]/gu, "")
    .replace(/^Ends\s+/i, "Closes ")
    .trim();
}

// LocalStorage Bookmarks Helpers
const BOOKMARKS_KEY = "naukribridge_bookmarked_job_ids";
const LEGACY_BOOKMARKS_KEY = "naukrisetu_bookmarked_job_ids";

export function getSavedJobIds(): number[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(BOOKMARKS_KEY) || localStorage.getItem(LEGACY_BOOKMARKS_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

export function toggleSavedJobId(id: number): boolean {
  if (typeof window === "undefined") return false;
  try {
    const ids = getSavedJobIds();
    const exists = ids.includes(id);
    let updated: number[];
    if (exists) {
      updated = ids.filter((x) => x !== id);
    } else {
      updated = [...ids, id];
    }
    localStorage.setItem(BOOKMARKS_KEY, JSON.stringify(updated));
    return !exists;
  } catch {
    return false;
  }
}

/**
 * Checks if a URL points to an official government or constitutional agency domain
 */
export function isOfficialGovUrl(url?: string): boolean {
  if (!url) return false;
  try {
    const parsed = new URL(url);
    const host = parsed.hostname.toLowerCase();
    return (
      host.endsWith(".gov.in") ||
      host.endsWith(".nic.in") ||
      host.endsWith(".ac.in") ||
      host.endsWith(".edu.in") ||
      host.endsWith(".mil.in") ||
      host.endsWith(".res.in") ||
      host.endsWith("ibps.in") ||
      host.includes("upsc.gov.in") ||
      host.includes("ssc.gov.in") ||
      host.includes("isro.gov.in")
    );
  } catch {
    return false;
  }
}

/**
 * Derives the verified official application website for a job
 */
export function getOfficialApplicationUrl(job: {
  source_portal?: string;
  details: {
    official_apply_url?: string;
    organization?: string;
  };
}): string | null {
  const d = job.details;
  const applyUrl = d.official_apply_url?.trim();

  // If application URL is explicitly set and is a webpage (not a direct PDF document)
  if (
    applyUrl &&
    !applyUrl.toLowerCase().endsWith(".pdf") &&
    !applyUrl.toLowerCase().includes("/attachment/") &&
    !applyUrl.toLowerCase().includes("/uploads/masterdata/noticeboards/")
  ) {
    return applyUrl;
  }

  // Official verified application portals by authority
  const portal = (job.source_portal || d.organization || "").toUpperCase();
  if (portal.includes("SSC") || portal.includes("STAFF SELECTION")) {
    return "https://ssc.gov.in";
  }
  if (portal.includes("UPSC") || portal.includes("UNION PUBLIC")) {
    return "https://upsconline.nic.in";
  }
  if (portal.includes("IBPS")) {
    return "https://ibps.in";
  }
  if (portal.includes("ISRO")) {
    return "https://www.isro.gov.in/Careers.html";
  }
  if (portal.includes("UPPSC")) {
    return "https://uppsc.up.nic.in";
  }
  if (portal.includes("DSSSB")) {
    return "https://dsssbonline.nic.in";
  }
  if (portal.includes("BPSC")) {
    return "https://onlinebpsc.bihar.gov.in";
  }
  if (portal.includes("RRB") || portal.includes("RAILWAY")) {
    return "https://rrbapply.gov.in";
  }
  if (portal.includes("NTA")) {
    return "https://nta.ac.in";
  }

  // If applyUrl was provided even as fallback
  if (applyUrl) return applyUrl;
  return null;
}

/**
 * Returns the verified official notification PDF / document URL
 */
export function getOfficialNotificationUrl(job: {
  raw_notice?: { notice_url?: string };
  details: {
    official_notice_url?: string;
    official_apply_url?: string;
  };
}): string | null {
  const d = job.details;
  if (d.official_notice_url && d.official_notice_url.trim()) {
    return d.official_notice_url.trim();
  }
  if (job.raw_notice?.notice_url && job.raw_notice.notice_url.trim()) {
    return job.raw_notice.notice_url.trim();
  }
  if (d.official_apply_url && d.official_apply_url.toLowerCase().endsWith(".pdf")) {
    return d.official_apply_url.trim();
  }
  return null;
}

/**
 * Cleans announcement titles to deduce the core related recruitment or examination name
 */
export function getRelatedRecruitmentName(title: string): string {
  if (!title) return "Government Recruitment";
  let clean = title;

  // Remove common announcement prefixes
  const prefixes = [
    /^notice\s+of\s+/i,
    /^notice\s+for\s+/i,
    /^notice\s*:\s*/i,
    /^admit\s+cards?\s+(?:for|of|regarding)\s+/i,
    /^e-admit\s+cards?\s+(?:for|of)?\s*/i,
    /^declaration\s+of\s+(?:results?|marks)\s+(?:for|of)\s+/i,
    /^results?\s+(?:for|of)\s+/i,
    /^schedule\s+of\s+/i,
    /^examination\s+schedule\s+(?:for|of)\s+/i,
    /^interview\s+schedule\s+(?:for|of)\s+/i,
    /^answer\s+keys?\s+(?:for|of|regarding)\s+/i,
    /^tentative\s+vacanc(?:y|ies)\s+(?:for|of)\s+/i,
    /^final\s+vacancies\s+(?:for|of)\s+/i,
    /^corrigendum\s+(?:to|regarding|for)\s+/i,
    /^marks\s+of\s+/i,
  ];

  for (const p of prefixes) {
    clean = clean.replace(p, "");
  }

  // Remove dates or trailing noise like "as on 24.09.2026"
  clean = clean.replace(/\s+as\s+on\s+[\d./-]+/gi, "");
  clean = clean.replace(/\s*\[.*\]\s*/g, " ");
  clean = clean.trim();

  return clean || title;
}

/**
 * Returns visual styling and iconography for announcement types
 */
export function getAnnouncementTypeInfo(type?: string): {
  label: string;
  bg: string;
  text: string;
  border: string;
  tag: string;
} {
  const t = (type || "").toLowerCase();

  if (t.includes("admit") || t.includes("hall_ticket")) {
    return {
      label: "Admit Card",
      bg: "bg-amber-50",
      text: "text-amber-800",
      border: "border-amber-200",
      tag: "🎫 Hall Ticket Released",
    };
  }
  if (t.includes("result") || t.includes("merit")) {
    return {
      label: "Result Declared",
      bg: "bg-emerald-50",
      text: "text-emerald-800",
      border: "border-emerald-200",
      tag: "🏆 Result Live",
    };
  }
  if (t.includes("answer") || t.includes("key")) {
    return {
      label: "Answer Key",
      bg: "bg-purple-50",
      text: "text-purple-800",
      border: "border-purple-200",
      tag: "🔑 Response Sheet Available",
    };
  }
  if (t.includes("interview") || t.includes("schedule") || t.includes("exam")) {
    return {
      label: "Exam Schedule",
      bg: "bg-blue-50",
      text: "text-blue-800",
      border: "border-blue-200",
      tag: "📅 Dates Announced",
    };
  }
  if (t.includes("corrigendum") || t.includes("extension")) {
    return {
      label: "Corrigendum",
      bg: "bg-rose-50",
      text: "text-rose-800",
      border: "border-rose-200",
      tag: "📢 Date / Notice Extension",
    };
  }

  return {
    label: "Official Announcement",
    bg: "bg-slate-100",
    text: "text-slate-800",
    border: "border-slate-200",
    tag: "🏛️ Official Update",
  };
}
