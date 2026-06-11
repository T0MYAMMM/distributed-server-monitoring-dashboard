"use client";

import { useCallback, useEffect, useState } from "react";
import { clearToken, getToken, onAuthChange, setToken } from "@/lib/auth";
import { login as apiLogin } from "@/lib/api/client";

// useAuth is the App Router auth hook: token state synced with localStorage and
// the auth-changed event, plus login/logout actions.
export function useAuth() {
  const [token, setTok] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    setTok(getToken());
    setReady(true);
    return onAuthChange(() => setTok(getToken()));
  }, []);

  const login = useCallback(async (password: string) => {
    const { token } = await apiLogin(password);
    setToken(token);
  }, []);

  const logout = useCallback(() => clearToken(), []);

  return { token, isAuthenticated: token !== null, ready, login, logout };
}
