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
