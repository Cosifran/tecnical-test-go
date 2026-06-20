import { cn } from "@/utils/cn";

interface LoadingSpinnerProps {
  size?: "sm" | "md" | "lg";
  label?: string;
  className?: string;
}

const sizeClasses = {
  sm: "h-4 w-4 border-2",
  md: "h-8 w-8 border-3",
  lg: "h-12 w-12 border-4",
} as const;

export function LoadingSpinner({
  size = "md",
  label = "Loading...",
  className,
}: LoadingSpinnerProps) {
  return (
    <div
      data-testid="loading-spinner-wrapper"
      className={cn("flex flex-col items-center justify-center gap-2", className)}
    >
      <div
        data-testid="loading-spinner"
        className={cn(
          "animate-spin rounded-full border-slate-600 border-t-blue-500",
          sizeClasses[size]
        )}
      />
      {label && (
        <p className="text-sm text-slate-400">{label}</p>
      )}
    </div>
  );
}