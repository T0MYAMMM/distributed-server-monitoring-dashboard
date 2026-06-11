import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

// cn merges conditional class names and resolves Tailwind conflicts so the last
// utility wins. Used by every shared/ui component.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
