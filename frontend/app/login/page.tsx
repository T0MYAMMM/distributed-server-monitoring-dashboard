"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getAuthStatus, initialize } from "@/lib/api/client";
import { useAuth } from "@/lib/hooks/useAuth";

export default function LoginPage() {
  const router = useRouter();
  const { login, isAuthenticated, ready } = useAuth();
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [initializing, setInitializing] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (ready && isAuthenticated) router.replace("/admin");
  }, [ready, isAuthenticated, router]);

  useEffect(() => {
    getAuthStatus()
      .then((s) => setInitializing(!s.initialized))
      .catch(() => setInitializing(true));
  }, []);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      if (initializing) await initialize(password);
      await login(password);
      router.replace("/admin");
    } catch {
      setError(initializing ? "Could not set password" : "Invalid password");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="items-center text-center">
          <span className="mb-2 flex h-11 w-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
            <Activity className="h-6 w-6" />
          </span>
          <CardTitle>{initializing ? "Set Admin Password" : "Admin Login"}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                required
                autoFocus
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
              />
            </div>
            {error && <p className="text-sm text-critical">{error}</p>}
            <Button type="submit" className="w-full" disabled={submitting}>
              {initializing ? "Set Password" : "Sign in"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
