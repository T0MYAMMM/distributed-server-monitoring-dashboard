// Auth token storage. The token lives in localStorage (persisted across
// reloads) and is read in-process for each request. Changes broadcast an event
// so the WebSocket manager can re-handshake and keep IP masking correct live.

const TOKEN_KEY = "adminToken";
export const AUTH_CHANGED_EVENT = "auth-changed";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  window.localStorage.setItem(TOKEN_KEY, token);
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}

export function clearToken(): void {
  window.localStorage.removeItem(TOKEN_KEY);
  window.dispatchEvent(new Event(AUTH_CHANGED_EVENT));
}

export function isAuthenticated(): boolean {
  return getToken() !== null;
}

// onAuthChange subscribes to login/logout; returns an unsubscribe function.
export function onAuthChange(fn: () => void): () => void {
  if (typeof window === "undefined") return () => {};
  window.addEventListener(AUTH_CHANGED_EVENT, fn);
  window.addEventListener("storage", fn); // cross-tab
  return () => {
    window.removeEventListener(AUTH_CHANGED_EVENT, fn);
    window.removeEventListener("storage", fn);
  };
}
