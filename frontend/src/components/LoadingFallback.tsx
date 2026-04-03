import React from "react";

interface LoadingFallbackProps {
  fullScreen?: boolean;
}

/**
 * Branded loading component for code-split loading states
 */
const LoadingFallback: React.FC<LoadingFallbackProps> = ({
  fullScreen = true,
}) => {
  const containerClasses = fullScreen
    ? "flex h-screen w-full items-center justify-center bg-background"
    : "flex h-full w-full items-center justify-center p-8";

  return (
    <div className={containerClasses}>
      <div className="flex flex-col items-center gap-4">
        <div className="relative">
          <div className="h-14 w-14 animate-spin rounded-full border-4 border-default-200 border-t-primary" />
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="h-6 w-6 rounded-full bg-primary/20 animate-pulse" />
          </div>
        </div>
        <span className="text-sm text-default-400 animate-pulse">Loading…</span>
      </div>
    </div>
  );
};

export default LoadingFallback;
