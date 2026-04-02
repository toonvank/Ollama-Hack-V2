import React from "react";
interface ErrorDisplayProps {
  error: Error | null;
  className?: string;
}

const ErrorDisplay = ({ error, className = "" }: ErrorDisplayProps) => {
  if (!error) return null;

  return (
    <div
      className={`p-4 text-white bg-danger-500 rounded-md my-4 ${className}`}
    >
      <p className="font-medium">Error</p>
      <p>{error.message || "An error occurred"}</p>
    </div>
  );
};

export default ErrorDisplay;
