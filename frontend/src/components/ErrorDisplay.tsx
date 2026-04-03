import { Button } from "@nextui-org/button";

interface ErrorDisplayProps {
  error: Error | null;
  className?: string;
  onRetry?: () => void;
}

const ErrorDisplay = ({
  error,
  className = "",
  onRetry,
}: ErrorDisplayProps) => {
  if (!error) return null;

  return (
    <div
      className={`p-4 border border-danger-200 bg-danger-50 dark:bg-danger-900/20 dark:border-danger-800 rounded-lg my-4 ${className}`}
    >
      <div className="flex items-start gap-3">
        <span className="text-danger text-lg mt-0.5">⚠</span>
        <div className="flex-1 min-w-0">
          <p className="font-medium text-danger-700 dark:text-danger-400">
            Something went wrong
          </p>
          <p className="text-sm text-danger-600 dark:text-danger-300 mt-1">
            {error.message || "An unexpected error occurred"}
          </p>
        </div>
        {onRetry && (
          <Button
            color="danger"
            size="sm"
            variant="flat"
            onPress={onRetry}
          >
            Retry
          </Button>
        )}
      </div>
    </div>
  );
};

export default ErrorDisplay;
