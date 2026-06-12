"use client";

import { ThemeProvider } from "next-themes";
import { Toaster } from "react-hot-toast";
import { QueryProvider } from "@/lib/query";

// Providers wraps the app in the theme, server-state, and toast contexts.
export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider attribute="class" defaultTheme="dark" enableSystem={false} disableTransitionOnChange>
      <QueryProvider>
        {children}
        <Toaster
          position="top-right"
          toastOptions={{
            duration: 3000,
            style: {
              background: "hsl(var(--card))",
              color: "hsl(var(--card-foreground))",
              border: "1px solid hsl(var(--border))",
            },
          }}
        />
      </QueryProvider>
    </ThemeProvider>
  );
}
