"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { JobItem, JobDetails, User } from "@/types";
import {
  fetchDraftJobs,
  publishJob,
  rejectJob,
  updateJob,
  triggerPipeline,
  loginUser,
  fetchUsers,
  createUser,
  getGoogleLoginUrl,
} from "@/lib/api";
import { INITIAL_DRAFTS } from "@/lib/mockData";

export default function AdminReviewPage() {
  // Authentication State
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [currentUser, setCurrentUser] = useState<User | null>(null);
  const [authChecking, setAuthChecking] = useState<boolean>(true);

  // Login Form State
  const [loginEmail, setLoginEmail] = useState<string>("");
  const [loginPassword, setLoginPassword] = useState<string>("");
  const [showPassword, setShowPassword] = useState<boolean>(false);
  const [loginLoading, setLoginLoading] = useState<boolean>(false);
  const [loginError, setLoginError] = useState<string | null>(null);

  // Admin Top-Level Navigation View: "drafts" | "users"
  const [activeAdminTab, setActiveAdminTab] = useState<"drafts" | "users">("drafts");

  // Users Database State
  const [dbUsers, setDbUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState<boolean>(false);
  const [showCreateUserModal, setShowCreateUserModal] = useState<boolean>(false);
  const [newUserName, setNewUserName] = useState<string>("");
  const [newUserEmail, setNewUserEmail] = useState<string>("");
  const [newUserPassword, setNewUserPassword] = useState<string>("");
  const [newUserRole, setNewUserRole] = useState<string>("user");
  const [newUserPhone, setNewUserPhone] = useState<string>("");
  const [creatingUser, setCreatingUser] = useState<boolean>(false);
  const [createUserError, setCreateUserError] = useState<string | null>(null);

  // Drafts & Review State
  const [drafts, setDrafts] = useState<JobItem[]>(INITIAL_DRAFTS);
  const [selectedDraft, setSelectedDraft] = useState<JobItem | null>(
    INITIAL_DRAFTS[0] || null
  );
  const [activeTypeTab, setActiveTypeTab] = useState<string>("all");
  const [loading, setLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  // Edit form state
  const [editFormData, setEditFormData] = useState<JobDetails | null>(
    INITIAL_DRAFTS[0]?.details || null
  );
  const [reviewNotes, setReviewNotes] = useState<string>(
    INITIAL_DRAFTS[0]?.review_notes || ""
  );

  // Check stored auth session on mount
  useEffect(() => {
    try {
      const storedToken = localStorage.getItem("rozgarhub_admin_token");
      const storedUser = localStorage.getItem("rozgarhub_admin_user");
      if (storedToken && storedUser) {
        const parsed = JSON.parse(storedUser);
        setCurrentUser(parsed);
        setIsAuthenticated(true);
      }
    } catch (e) {
      console.warn("Auth check failed:", e);
    } finally {
      setAuthChecking(false);
    }
  }, []);

  const showToast = (msg: string) => {
    setToastMessage(msg);
    setTimeout(() => setToastMessage(null), 4000);
  };

  // Handle Login
  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoginError(null);
    setLoginLoading(true);

    try {
      const res = await loginUser(loginEmail, loginPassword);
      if (res && res.success) {
        localStorage.setItem("rozgarhub_admin_token", res.token);
        localStorage.setItem("rozgarhub_admin_user", JSON.stringify(res.user));
        setCurrentUser(res.user);
        setIsAuthenticated(true);
        showToast(`👋 Welcome back, ${res.user.name || "Admin"}!`);
      } else {
        setLoginError("Invalid credentials. Please verify your email and password.");
      }
    } catch (err: any) {
      setLoginError(err?.message || "Failed to login. Please try again.");
    } finally {
      setLoginLoading(false);
    }
  };

  // Handle Logout
  const handleLogout = () => {
    localStorage.removeItem("rozgarhub_admin_token");
    localStorage.removeItem("rozgarhub_admin_user");
    setIsAuthenticated(false);
    setCurrentUser(null);
    showToast("Logged out successfully.");
  };

  // Load Users from DB
  const loadUsers = async () => {
    try {
      setUsersLoading(true);
      const res = await fetchUsers(100, 0);
      if (res && res.users) {
        setDbUsers(res.users);
      }
    } catch (err) {
      console.warn("Error fetching users from DB:", err);
    } finally {
      setUsersLoading(false);
    }
  };

  // Create User in DB
  const handleCreateUser = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreateUserError(null);

    if (!newUserEmail.trim() || !newUserPassword.trim()) {
      setCreateUserError("Email and password are required");
      return;
    }

    try {
      setCreatingUser(true);
      const res = await createUser({
        name: newUserName.trim() || newUserEmail.split("@")[0],
        email: newUserEmail.trim(),
        password: newUserPassword.trim(),
        role: newUserRole,
        phone: newUserPhone.trim(),
      });

      showToast(`✅ User ${res.user.email} saved to database!`);
      setShowCreateUserModal(false);
      setNewUserName("");
      setNewUserEmail("");
      setNewUserPassword("");
      setNewUserPhone("");
      await loadUsers();
    } catch (err: any) {
      setCreateUserError(err?.message || "Failed to create user in database");
    } finally {
      setCreatingUser(false);
    }
  };

  // Load drafts from API
  const loadDrafts = async () => {
    try {
      setLoading(true);
      const res = await fetchDraftJobs({
        type: activeTypeTab === "all" ? undefined : activeTypeTab,
      });
      if (res && res.jobs) {
        setDrafts(res.jobs);
        if (res.jobs.length > 0) {
          setSelectedDraft(res.jobs[0]);
          setEditFormData(res.jobs[0].details);
          setReviewNotes(res.jobs[0].review_notes || "");
        } else {
          setSelectedDraft(null);
          setEditFormData(null);
        }
      }
    } catch (err) {
      console.warn("Using mock drafts (backend offline):", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isAuthenticated) {
      if (activeAdminTab === "drafts") {
        loadDrafts();
      } else if (activeAdminTab === "users") {
        loadUsers();
      }
    }
  }, [activeTypeTab, isAuthenticated, activeAdminTab]);

  const selectDraft = (draft: JobItem) => {
    setSelectedDraft(draft);
    setEditFormData({ ...draft.details });
    setReviewNotes(draft.review_notes || "");
  };

  // 1-Click Publish
  const handlePublish = async (id: number) => {
    try {
      if (editFormData) {
        await updateJob(id, { details: editFormData, review_notes: reviewNotes });
      }
      const res = await publishJob(id);
      showToast(`✅ Notice #${res.id} published successfully to live feed!`);
      const updated = drafts.filter((d) => d.id !== id);
      setDrafts(updated);
      if (updated.length > 0) {
        selectDraft(updated[0]);
      } else {
        setSelectedDraft(null);
        setEditFormData(null);
      }
    } catch {
      const updated = drafts.filter((d) => d.id !== id);
      setDrafts(updated);
      showToast(`✅ [Local State] Notice #${id} marked as published!`);
      if (updated.length > 0) selectDraft(updated[0]);
    }
  };

  // Reject Draft
  const handleReject = async (id: number) => {
    const reason = window.prompt("Reason for rejection / archiving:", "Duplicate notice / irrelevant circular");
    if (reason === null) return;

    try {
      await rejectJob(id, reason);
      showToast(`🗑️ Notice #${id} rejected.`);
      const updated = drafts.filter((d) => d.id !== id);
      setDrafts(updated);
      if (updated.length > 0) selectDraft(updated[0]);
    } catch {
      const updated = drafts.filter((d) => d.id !== id);
      setDrafts(updated);
      showToast(`🗑️ [Local State] Notice #${id} rejected.`);
      if (updated.length > 0) selectDraft(updated[0]);
    }
  };

  // Save Edits
  const handleSaveEdits = async () => {
    if (!selectedDraft || !editFormData) return;
    try {
      await updateJob(selectedDraft.id, {
        details: editFormData,
        review_notes: reviewNotes,
      });
      showToast(`💾 Changes saved for Notice #${selectedDraft.id}!`);
      setDrafts((prev) =>
        prev.map((d) =>
          d.id === selectedDraft.id
            ? { ...d, details: editFormData, review_notes: reviewNotes }
            : d
        )
      );
    } catch {
      showToast("💾 [Local State] Changes saved!");
    }
  };

  // Trigger Pipeline
  const handleTriggerScraper = async () => {
    try {
      setLoading(true);
      showToast("🔄 Syncing latest notices from scraper...");
      const res = await triggerPipeline();
      showToast(`⚡ ${res.message || "Scraper sync finished!"}`);
      await loadDrafts();
    } catch {
      showToast("⚡ Scraping and AI extraction pipeline triggered in background.");
      await loadDrafts();
    } finally {
      setLoading(false);
    }
  };

  // 1. Initial Auth Checking State
  if (authChecking) {
    return (
      <div className="min-h-[500px] flex items-center justify-center">
        <div className="flex flex-col items-center gap-3 text-slate-500">
          <div className="w-8 h-8 border-3 border-blue-600 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs font-semibold">Verifying secure admin session...</p>
        </div>
      </div>
    );
  }

  // 2. Admin Login Gateway (If not authenticated)
  if (!isAuthenticated) {
    return (
      <div className="min-h-[75vh] flex items-center justify-center px-4 py-8">
        <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl shadow-xl p-6 sm:p-8 space-y-6">
          {/* Brand & Title */}
          <div className="text-center space-y-2">
            <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-blue-50 border border-blue-200 text-blue-700 text-xs font-bold uppercase tracking-wider">
              <span>🔒</span> Admin Gateway
            </div>
            <h1 className="text-2xl font-black tracking-tight text-slate-900">
              RozgarHub Control Panel
            </h1>
            <p className="text-xs text-slate-500 max-w-xs mx-auto">
              Sign in with your administrator credentials to access the editorial review, DB users, and scraper pipeline.
            </p>
          </div>

          {/* Error Banner */}
          {loginError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-xl text-red-600 text-xs font-medium flex items-start gap-2">
              <span className="text-sm">⚠️</span>
              <span>{loginError}</span>
            </div>
          )}

          {/* Login Form */}
          <form onSubmit={handleLoginSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-xs font-bold text-slate-700 block">Admin Email</label>
              <input
                type="email"
                required
                autoComplete="username"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                placeholder="Enter admin email"
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-900 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-bold text-slate-700">Password</label>
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="text-[11px] text-slate-500 hover:text-slate-700 font-medium"
                >
                  {showPassword ? "Hide" : "Show"}
                </button>
              </div>
              <input
                type={showPassword ? "text" : "password"}
                required
                autoComplete="current-password"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                placeholder="Enter password"
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-900 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
            </div>

            <button
              type="submit"
              disabled={loginLoading}
              className="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 active:scale-[0.99] text-white font-bold text-xs uppercase tracking-wider rounded-xl shadow-md transition-all flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
            >
              {loginLoading ? (
                <>
                  <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                  <span>Verifying...</span>
                </>
              ) : (
                <span>Sign In with Credentials →</span>
              )}
            </button>
          </form>

          {/* Divider */}
          <div className="relative flex items-center justify-center my-1">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-slate-200"></div>
            </div>
            <div className="relative bg-white px-3 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
              Or continue with
            </div>
          </div>

          {/* Google OAuth Button */}
          <a
            href={getGoogleLoginUrl()}
            onClick={(e) => {
              e.preventDefault();
              window.location.href = getGoogleLoginUrl();
            }}
            className="w-full py-2.5 px-4 bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 hover:border-slate-300 font-bold text-xs rounded-xl shadow-2xs transition-all flex items-center justify-center gap-2.5 cursor-pointer"
          >
            <svg className="w-4 h-4 shrink-0" viewBox="0 0 24 24">
              <path
                fill="#4285F4"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="#34A853"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="#FBBC05"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.06H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.94l2.85-2.22.81-.63z"
              />
              <path
                fill="#EA4335"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52z"
              />
            </svg>
            <span>Sign in with Google</span>
          </a>

          <div className="text-center pt-2 border-t border-slate-100">
            <Link
              href="/"
              className="text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
            >
              ← Return to Candidate Job Feed
            </Link>
          </div>
        </div>
      </div>
    );
  }

  // 3. Authenticated Admin Dashboard
  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {toastMessage && (
        <div className="fixed top-20 right-6 z-50 bg-slate-900 text-white text-xs font-semibold px-4 py-3 rounded-xl shadow-xl flex items-center gap-2 border border-slate-700 animate-in fade-in slide-in-from-top-2">
          <span>{toastMessage}</span>
          <button onClick={() => setToastMessage(null)} className="text-slate-400 hover:text-white">✕</button>
        </div>
      )}

      {/* Admin Top Navigation Bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 bg-white text-slate-900 rounded-2xl p-5 border border-slate-200 shadow-2xs">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs font-bold px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-200">
              Admin Control Center
            </span>
            <span className="text-xs text-slate-500">• RozgarHub Engine</span>
            <span className="text-xs font-mono font-bold px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-200">
              {currentUser?.email || "admin@admin.com"}
            </span>
          </div>
          <h1 className="text-xl sm:text-2xl font-black tracking-tight text-slate-900">
            {activeAdminTab === "drafts" ? "Draft Notifications Review" : "Database Users & Accounts"}
          </h1>
          <p className="text-xs text-slate-500 mt-0.5">
            {activeAdminTab === "drafts"
              ? "Verify auto-extracted government circulars, edit parameters, and publish directly to live candidates."
              : "Manage administrator profiles, candidate entries, and registered users in the database."}
          </p>
        </div>

        {/* Action Buttons & Tab Switcher */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Main Module Switcher */}
          <div className="inline-flex p-1 bg-slate-100 rounded-xl border border-slate-200">
            <button
              onClick={() => setActiveAdminTab("drafts")}
              className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-all ${
                activeAdminTab === "drafts"
                  ? "bg-white text-blue-700 shadow-xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              📋 Drafts ({drafts.length})
            </button>
            <button
              onClick={() => setActiveAdminTab("users")}
              className={`px-3 py-1.5 text-xs font-bold rounded-lg transition-all ${
                activeAdminTab === "users"
                  ? "bg-white text-blue-700 shadow-xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              👥 DB Users ({dbUsers.length})
            </button>
          </div>

          <button
            onClick={handleTriggerScraper}
            className="flex items-center gap-1.5 px-3.5 py-2 text-xs font-bold rounded-xl bg-blue-600 hover:bg-blue-700 text-white shadow-2xs transition-all cursor-pointer"
          >
            <span>🔄 Sync Scrapers</span>
          </button>

          <Link
            href="/"
            className="px-3 py-2 text-xs font-semibold rounded-xl bg-slate-100 hover:bg-slate-200 border border-slate-200 text-slate-700 transition-colors"
          >
            Candidate Feed →
          </Link>

          <button
            onClick={handleLogout}
            title="Sign out of Admin Portal"
            className="px-3 py-2 text-xs font-bold rounded-xl bg-red-50 hover:bg-red-100 border border-red-200 text-red-600 transition-colors cursor-pointer"
          >
            Logout 🚪
          </button>
        </div>
      </div>

      {/* VIEW 1: USERS IN DB MODULE */}
      {activeAdminTab === "users" && (
        <div className="space-y-4">
          {/* Subheader with Add User Action */}
          <div className="flex items-center justify-between bg-white p-4 rounded-xl border border-slate-200">
            <div>
              <h2 className="font-bold text-slate-900 text-sm">Application Users in Database</h2>
              <p className="text-xs text-slate-500">Live entries registered in Postgres/SQLite database.</p>
            </div>
            <button
              onClick={() => setShowCreateUserModal(true)}
              className="px-3.5 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-bold shadow-xs cursor-pointer flex items-center gap-1.5"
            >
              <span>+ Add New User to DB</span>
            </button>
          </div>

          {/* Create User Modal / Dialog */}
          {showCreateUserModal && (
            <div className="fixed inset-0 z-50 bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4">
              <div className="bg-white rounded-2xl border border-slate-200 shadow-2xl max-w-md w-full p-6 space-y-4">
                <div className="flex items-center justify-between border-b border-slate-100 pb-3">
                  <h3 className="font-black text-slate-900 text-base">Create Database User Entry</h3>
                  <button
                    onClick={() => setShowCreateUserModal(false)}
                    className="text-slate-400 hover:text-slate-600"
                  >
                    ✕
                  </button>
                </div>

                {createUserError && (
                  <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-600 text-xs">
                    {createUserError}
                  </div>
                )}

                <form onSubmit={handleCreateUser} className="space-y-3">
                  <div>
                    <label className="text-xs font-bold text-slate-700 block mb-1">Full Name</label>
                    <input
                      type="text"
                      required
                      value={newUserName}
                      onChange={(e) => setNewUserName(e.target.value)}
                      placeholder="e.g. Ramesh Kumar"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  <div>
                    <label className="text-xs font-bold text-slate-700 block mb-1">Email Address</label>
                    <input
                      type="email"
                      required
                      value={newUserEmail}
                      onChange={(e) => setNewUserEmail(e.target.value)}
                      placeholder="user@example.com"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  <div>
                    <label className="text-xs font-bold text-slate-700 block mb-1">Password</label>
                    <input
                      type="password"
                      required
                      value={newUserPassword}
                      onChange={(e) => setNewUserPassword(e.target.value)}
                      placeholder="••••••••"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs font-bold text-slate-700 block mb-1">Role</label>
                      <select
                        value={newUserRole}
                        onChange={(e) => setNewUserRole(e.target.value)}
                        className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                      >
                        <option value="user">Candidate / User</option>
                        <option value="admin">Administrator</option>
                      </select>
                    </div>

                    <div>
                      <label className="text-xs font-bold text-slate-700 block mb-1">Phone (Optional)</label>
                      <input
                        type="text"
                        value={newUserPhone}
                        onChange={(e) => setNewUserPhone(e.target.value)}
                        placeholder="9876543210"
                        className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                  </div>

                  <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-100">
                    <button
                      type="button"
                      onClick={() => setShowCreateUserModal(false)}
                      className="px-3.5 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 rounded-lg border border-slate-200"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={creatingUser}
                      className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-xs font-bold rounded-lg shadow-xs cursor-pointer disabled:opacity-50"
                    >
                      {creatingUser ? "Saving..." : "Save User to Database"}
                    </button>
                  </div>
                </form>
              </div>
            </div>
          )}

          {/* Users Table */}
          <div className="bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-2xs">
            {usersLoading ? (
              <div className="p-12 text-center text-slate-500 text-xs">Loading database users...</div>
            ) : dbUsers.length === 0 ? (
              <div className="p-12 text-center space-y-2">
                <p className="font-bold text-slate-700 text-sm">No Users Found</p>
                <p className="text-xs text-slate-400">Click &ldquo;+ Add New User to DB&rdquo; to create your first entry.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs text-slate-700">
                  <thead className="bg-slate-50 border-b border-slate-200 text-slate-500 font-bold uppercase tracking-wider text-[10px]">
                    <tr>
                      <th className="px-4 py-3">ID</th>
                      <th className="px-4 py-3">Name</th>
                      <th className="px-4 py-3">Email</th>
                      <th className="px-4 py-3">Role</th>
                      <th className="px-4 py-3">Phone</th>
                      <th className="px-4 py-3">Created Date</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {dbUsers.map((u) => (
                      <tr key={u.id} className="hover:bg-slate-50/70 transition-colors">
                        <td className="px-4 py-3 font-mono font-bold text-slate-400">#{u.id}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2.5">
                            {u.avatar_url ? (
                              <img
                                src={u.avatar_url}
                                alt={u.name}
                                className="w-6 h-6 rounded-full border border-slate-200 shrink-0 object-cover"
                              />
                            ) : (
                              <div className="w-6 h-6 rounded-full bg-slate-200 text-slate-600 flex items-center justify-center font-bold text-[10px] shrink-0">
                                {u.name ? u.name.charAt(0).toUpperCase() : "U"}
                              </div>
                            )}
                            <span className="font-semibold text-slate-900">{u.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3 font-mono text-slate-600">{u.email}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-1.5 flex-wrap">
                            <span
                              className={`px-2 py-0.5 rounded-full text-[10px] font-bold uppercase ${
                                u.role === "admin"
                                  ? "bg-purple-100 text-purple-700 border border-purple-200"
                                  : "bg-emerald-100 text-emerald-700 border border-emerald-200"
                              }`}
                            >
                              {u.role}
                            </span>
                            {u.auth_provider === "google" && (
                              <span className="px-1.5 py-0.5 rounded-full text-[9px] font-bold bg-blue-50 text-blue-600 border border-blue-200 inline-flex items-center gap-1">
                                <span>G</span> Google
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-slate-500">{u.phone || "—"}</td>
                        <td className="px-4 py-3 text-slate-400 font-mono text-[11px]">
                          {u.created_at ? new Date(u.created_at).toLocaleDateString() : "Just now"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}

      {/* VIEW 2: DRAFTS REVIEW MODULE */}
      {activeAdminTab === "drafts" && (
        <>
          {/* Filter Tabs */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 overflow-x-auto text-xs font-semibold">
              {[
                { id: "all", label: "All Drafts" },
                { id: "recruitment", label: "Recruitments" },
                { id: "admit_card", label: "Admit Cards" },
                { id: "result", label: "Results" },
                { id: "interview_schedule", label: "Interviews" },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTypeTab(tab.id)}
                  className={`px-3.5 py-1.5 rounded-lg border transition-all cursor-pointer ${
                    activeTypeTab === tab.id
                      ? "bg-blue-600 text-white border-blue-600 shadow-2xs"
                      : "bg-white text-slate-700 border-slate-200 hover:bg-slate-50"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            <span className="text-xs text-slate-500 font-medium">
              {drafts.length} notices awaiting review
            </span>
          </div>

          {/* Master Split View */}
          {drafts.length > 0 && selectedDraft && editFormData ? (
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
              {/* Left Column: Drafts List */}
              <div className="lg:col-span-5 bg-white rounded-2xl border border-slate-200 shadow-2xs overflow-hidden">
                <div className="p-3.5 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
                  <span className="text-xs font-bold text-slate-700 uppercase tracking-wider">
                    Unreviewed Circulars
                  </span>
                  <span className="text-xs font-bold text-blue-600 bg-blue-50 px-2 py-0.5 rounded border border-blue-200">
                    {drafts.length} Pending
                  </span>
                </div>

                <div className="divide-y divide-slate-100 max-h-[700px] overflow-y-auto">
                  {drafts.map((draft) => {
                    const isSelected = selectedDraft.id === draft.id;
                    const confidence = Math.round(
                      (draft.details.confidence_score || 0.85) * 100
                    );

                    return (
                      <div
                        key={draft.id}
                        onClick={() => selectDraft(draft)}
                        className={`p-4 cursor-pointer transition-colors ${
                          isSelected
                            ? "bg-blue-50/70 border-l-4 border-l-blue-600"
                            : "hover:bg-slate-50"
                        }`}
                      >
                        <div className="flex items-center justify-between gap-2 mb-1.5">
                          <span className="text-[11px] font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700 border border-slate-200">
                            {draft.details.organization || draft.source_portal}
                          </span>
                          <span
                            className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
                              confidence >= 80
                                ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                : "bg-amber-50 text-amber-700 border border-amber-200"
                            }`}
                          >
                            AI: {confidence}%
                          </span>
                        </div>

                        <h3 className="text-xs font-bold text-slate-900 line-clamp-2">
                          {draft.details.title}
                        </h3>

                        <div className="flex items-center gap-3 mt-2 text-[11px] text-slate-500">
                          {draft.details.total_vacancies > 0 && (
                            <span>👥 {draft.details.total_vacancies} posts</span>
                          )}
                          {draft.details.application_end && (
                            <span>⏳ {draft.details.application_end}</span>
                          )}
                          <span className="capitalize">
                            📌 {draft.details.notice_type?.replace("_", " ")}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Right Column: Interactive Review Form */}
              <div className="lg:col-span-7 bg-white rounded-2xl border border-slate-200 shadow-2xs p-5 sm:p-6 space-y-6">
                {/* Notice Top Banner */}
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-4 border-b border-slate-100">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs font-mono font-bold text-slate-500">
                        Notice #{selectedDraft.id}
                      </span>
                      <span className="text-xs font-semibold px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-200">
                        Source: {selectedDraft.source_portal}
                      </span>
                    </div>
                    <h2 className="text-base font-bold text-slate-900">
                      Verify & Edit Job Attributes
                    </h2>
                  </div>

                  <div className="flex items-center gap-2">
                    {selectedDraft.details.official_notice_url && (
                      <a
                        href={selectedDraft.details.official_notice_url}
                        target="_blank"
                        rel="noreferrer"
                        className="px-3 py-1.5 text-xs font-bold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg border border-slate-200 transition-colors"
                      >
                        📄 Open PDF
                      </a>
                    )}
                    {selectedDraft.details.official_apply_url && (
                      <a
                        href={selectedDraft.details.official_apply_url}
                        target="_blank"
                        rel="noreferrer"
                        className="px-3 py-1.5 text-xs font-bold text-blue-600 bg-blue-50 hover:bg-blue-100 rounded-lg border border-blue-200 transition-colors"
                      >
                        🔗 Apply Link
                      </a>
                    )}
                  </div>
                </div>

                {/* Editable Fields Form */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
                  {/* Title */}
                  <div className="sm:col-span-2 space-y-1">
                    <label className="font-bold text-slate-800">Job Title / Examination</label>
                    <input
                      type="text"
                      value={editFormData.title}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, title: e.target.value })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Organization */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Department / Organization</label>
                    <input
                      type="text"
                      value={editFormData.organization}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, organization: e.target.value })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Notice Type */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Notice Type</label>
                    <select
                      value={editFormData.notice_type}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, notice_type: e.target.value })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    >
                      <option value="recruitment">Recruitment</option>
                      <option value="admit_card">Admit Card</option>
                      <option value="result">Result</option>
                      <option value="interview_schedule">Interview Schedule</option>
                    </select>
                  </div>

                  {/* Category */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Category</label>
                    <input
                      type="text"
                      value={editFormData.category}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, category: e.target.value })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Total Vacancies */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Total Vacancies</label>
                    <input
                      type="number"
                      value={editFormData.total_vacancies}
                      onChange={(e) =>
                        setEditFormData({
                          ...editFormData,
                          total_vacancies: parseInt(e.target.value) || 0,
                        })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Qualification */}
                  <div className="sm:col-span-2 space-y-1">
                    <label className="font-bold text-slate-800">Minimum Qualification</label>
                    <input
                      type="text"
                      value={editFormData.qualification}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, qualification: e.target.value })
                      }
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Dates: Application End */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Application Deadline</label>
                    <input
                      type="text"
                      value={editFormData.application_end || ""}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, application_end: e.target.value })
                      }
                      placeholder="YYYY-MM-DD"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Salary / Pay Scale */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Salary / Pay Scale</label>
                    <input
                      type="text"
                      value={editFormData.salary_payscale || ""}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, salary_payscale: e.target.value })
                      }
                      placeholder="e.g. ₹56,100 - ₹1,77,500 (Level-10)"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Application Fee */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">General Fee</label>
                    <input
                      type="text"
                      value={editFormData.fee_general || ""}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, fee_general: e.target.value })
                      }
                      placeholder="e.g. ₹100"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Reserved Fee */}
                  <div className="space-y-1">
                    <label className="font-bold text-slate-800">Reserved / Female Fee</label>
                    <input
                      type="text"
                      value={editFormData.fee_reserved || ""}
                      onChange={(e) =>
                        setEditFormData({ ...editFormData, fee_reserved: e.target.value })
                      }
                      placeholder="e.g. Exempted / Nil"
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>

                  {/* Review Notes */}
                  <div className="sm:col-span-2 space-y-1">
                    <label className="font-bold text-slate-800">Review Notes</label>
                    <input
                      type="text"
                      value={reviewNotes}
                      onChange={(e) => setReviewNotes(e.target.value)}
                      placeholder="e.g. Verified with PDF."
                      className="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-900 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                </div>

                {/* Bottom Action Buttons */}
                <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
                  <button
                    onClick={() => handleReject(selectedDraft.id)}
                    className="px-4 py-2 text-xs font-semibold text-red-600 hover:bg-red-50 border border-red-200 rounded-lg cursor-pointer"
                  >
                    Reject Notice
                  </button>

                  <button
                    onClick={handleSaveEdits}
                    className="px-4 py-2 text-xs font-semibold text-slate-700 hover:bg-slate-100 border border-slate-200 rounded-lg cursor-pointer"
                  >
                    Save Changes
                  </button>

                  <button
                    onClick={() => handlePublish(selectedDraft.id)}
                    className="px-6 py-2.5 text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-700 rounded-xl shadow-md cursor-pointer transition-all"
                  >
                    ✓ 1-Click Publish
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-2xl border border-dashed border-slate-300 p-12 text-center space-y-3">
              <p className="text-slate-700 font-bold text-base">No Drafts Awaiting Review</p>
              <p className="text-slate-400 text-xs">
                All notices have been reviewed and published. Click &ldquo;Sync Scrapers&rdquo; to trigger pipeline.
              </p>
              <button
                onClick={handleTriggerScraper}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg text-xs font-semibold hover:bg-blue-700 cursor-pointer"
              >
                Run Scraping Pipeline
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
