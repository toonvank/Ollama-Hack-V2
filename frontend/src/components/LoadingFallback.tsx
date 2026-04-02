import React from "react";

interface LoadingFallbackProps {
  fullScreen?: boolean;
}

/**
 * Generic loading component for code-split loading states
 */
const LoadingFallback: React.FC<LoadingFallbackProps> = ({
  fullScreen = true,
}) => {
  const containerClasses = fullScreen
    ? "flex h-screen w-full items-center justify-center"
    : "flex h-full w-full items-center justify-center p-8";

  return (
    <div className={containerClasses}>
      <div className="h-16 w-16 animate-spin rounded-full border-b-2 border-t-2 border-primary" />
    </div>
  );
};

export default LoadingFallback;
