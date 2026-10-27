"use client";

import { useEffect, useState, Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import Link from "next/link";
import { verifyGoogleAuth } from "@/lib/api";

function AuthCallbackContent() {
  const searchParams = useSearchParams();
  const router = useRouter();

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [userName, setUserName] = useState<string>("");
  const [isDemoLoading, setIsDemoLoading] = useState<boolean>(false);

  useEffect(() => {
    const errorParam = searchParams.get("error");
    const tokenParam = searchParams.get("token");
    const userParam = searchParams.get("user");

    if (errorParam) {
      setStatus("error");
      if (errorParam === "google_client_id_not_configured") {
        setErrorMessage(
          "Google Client ID & Secret are not yet configured in backend/.env. Add GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET from console.cloud.google.com."
        );
      } else {
        setErrorMessage(`Google authentication returned: ${errorParam}`);
      }
      return;
    }

    if (tokenParam && userParam) {
      try {
        const decodedUser = JSON.parse(decodeURIComponent(userParam));
        setUserName(decodedUser.name || decodedUser.email || "Candidate");

        // Save session
        localStorage.setItem("rozgarhub_user_token", tokenParam);
        localStorage.setItem("rozgarhub_user", JSON.stringify(decodedUser));

        if (decodedUser.role === "admin") {
          localStorage.setItem("rozgarhub_admin_token", tokenParam);
          localStorage.setItem("rozgarhub_admin_user", JSON.stringify(decodedUser));
        }

        setStatus("success");
        setTimeout(() => {
          if (decodedUser.role === "admin") {
            router.push("/admin");
          } else {
            router.push("/");
          }
        }, 1500);
      } catch (e) {
        setStatus("error");
        setErrorMessage("Failed to decode user authentication payload.");
      }
    }
  }, [searchParams, router]);

  // Demo Google Login to test DB user storage immediately before setting up Google Cloud Console
  const handleTestDemoGoogleLogin = async () => {
    try {
      setIsDemoLoading(true);
      const res = await verifyGoogleAuth({
        mock_user: {
          id: "google_demo_" + Date.now(),
          email: "aspirant.demo@gmail.com",
          name: "Verified Google Aspirant",
          picture: "https://lh3.googleusercontent.com/a/default-user=s96-c",
        },
      });

      if (res && res.success) {
        localStorage.setItem("rozgarhub_user_token", res.token);
        localStorage.setItem("rozgarhub_user", JSON.stringify(res.user));
        setUserName(res.user.name);
        setStatus("success");
        setTimeout(() => {
          router.push("/");
        }, 1200);
      }
    } catch (err: any) {
      setErrorMessage(err?.message || "Failed to complete demo Google sign in");
    } finally {
      setIsDemoLoading(false);
    }
  };

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white border border-slate-200 rounded-3xl shadow-xl p-8 text-center space-y-6">
        {status === "loading" && (
          <div className="space-y-4 py-8">
            <div className="w-12 h-12 border-3 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto" />
            <h2 className="text-base font-bold text-slate-800">Verifying Google Account...</h2>
            <p className="text-xs text-slate-500">Connecting with Google OAuth 2.0 and syncing database profile.</p>
          </div>
        )}

        {status === "success" && (
          <div className="space-y-4 py-6">
            <div className="w-14 h-14 bg-emerald-50 text-emerald-600 border border-emerald-200 rounded-2xl flex items-center justify-center text-2xl mx-auto shadow-xs">
              ✓
            </div>
            <div>
              <h2 className="text-lg font-black text-slate-900">Sign-in Successful!</h2>
              <p className="text-xs text-slate-500 mt-1">Welcome back, <strong className="text-slate-800">{userName}</strong>.</p>
            </div>
            <p className="text-xs text-blue-600 font-medium animate-pulse">Redirecting to your dashboard...</p>
          </div>
        )}

        {status === "error" && (
          <div className="space-y-5 py-4">
            <div className="w-14 h-14 bg-amber-50 text-amber-600 border border-amber-200 rounded-2xl flex items-center justify-center text-2xl mx-auto shadow-xs">
              ⚙️
            </div>
            <div>
              <h2 className="text-lg font-black text-slate-900">Google OAuth Setup</h2>
              <p className="text-xs text-slate-600 mt-2 text-left leading-relaxed bg-slate-50 p-3.5 rounded-xl border border-slate-200">
                {errorMessage}
              </p>
            </div>

            <div className="space-y-2 pt-2">
              <button
                onClick={handleTestDemoGoogleLogin}
                disabled={isDemoLoading}
                className="w-full py-2.5 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-bold transition-all shadow-xs flex items-center justify-center gap-2 cursor-pointer disabled:opacity-50"
              >
                {isDemoLoading ? "Saving to DB..." : "⚡ Test Simulated Google Sign-In"}
              </button>

              <Link
                href="/admin"
                className="block w-full py-2.5 px-4 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-xl text-xs font-bold transition-colors"
              >
                Go to Admin Portal
              </Link>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default function AuthCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="min-h-[70vh] flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
        </div>
      }
    >
      <AuthCallbackContent />
    </Suspense>
  );
}
