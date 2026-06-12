"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

// CopyButton copies text to the clipboard and briefly confirms.
export function CopyButton({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(value);
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } catch {
          /* clipboard unavailable */
        }
      }}
    >
      {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
      {label ?? (copied ? "Copied" : "Copy")}
    </Button>
  );
}
