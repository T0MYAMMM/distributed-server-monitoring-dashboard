"use client";

import { useMutation } from "@tanstack/react-query";
import { submitFeedback } from "@/lib/api/client";

// useSubmitFeedback posts a feedback item. Submission is public (any viewer can
// send), so it needs no auth.
export function useSubmitFeedback() {
  return useMutation({
    mutationFn: (body: { category: string; message: string; page: string }) => submitFeedback(body),
  });
}
