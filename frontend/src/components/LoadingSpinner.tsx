import React from "react";

interface LoadingSpinnerProps {
  size?: "small" | "medium" | "large";
  className?: string;
}

const LoadingSpinner = React.memo(
  ({ size = "medium", className = "" }: LoadingSpinnerProps) => {
    const sizeClasses = {
      small: "w-4 h-4",
      medium: "w-8 h-8",
      large: "w-12 h-12",
    };

    return (
      <div className={`flex justify-center items-center ${className}`}>
        <div
          className={`${sizeClasses[size]} border-4 border-t-4 border-gray-200 border-t-primary-500 rounded-full animate-spin`}
        />
      </div>
    );
  },
);

LoadingSpinner.displayName = "LoadingSpinner";

export default LoadingSpinner;
