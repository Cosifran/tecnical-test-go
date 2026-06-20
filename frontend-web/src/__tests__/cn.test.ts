import { describe, it, expect } from "vitest";
import { cn } from "../utils/cn";

describe("cn utility", () => {
  it("merges multiple class names", () => {
    const result = cn("px-4", "py-2", "bg-slate-800");
    expect(result).toBe("px-4 py-2 bg-slate-800");
  });

  it("handles conditional classes with falsy values", () => {
    const isActive = true;
    const isDisabled = false;
    const result = cn("base-class", isActive && "active-class", isDisabled && "disabled-class");
    expect(result).toBe("base-class active-class");
  });

  it("resolves conflicting Tailwind classes using tailwind-merge", () => {
    const result = cn("px-4", "px-6");
    expect(result).toBe("px-6");
  });

  it("handles empty inputs", () => {
    const result = cn();
    expect(result).toBe("");
  });

  it("handles undefined and null values gracefully", () => {
    const result = cn("text-white", undefined, null, "bg-slate-900");
    expect(result).toBe("text-white bg-slate-900");
  });

  it("merges clsx conditionals with tailwind-merge dedup", () => {
    const result = cn("rounded-lg border", true && "border-slate-700", false && "hidden");
    expect(result).toBe("rounded-lg border border-slate-700");
  });
});