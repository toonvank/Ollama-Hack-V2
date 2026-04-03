import React, { useState, KeyboardEvent } from "react";
import { Pagination as HeroUIPagination } from "@nextui-org/pagination";
import { NumberInput } from "@nextui-org/number-input";
import { Button } from "@nextui-org/button";

interface PaginationProps {
  currentPage: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  className?: string;
  props?: PaginationProps;
  showJumper?: boolean; // Whether to show the page jump input
}

const Pagination = ({
  currentPage,
  totalPages,
  onPageChange,
  props,
  showJumper = true, // Show jump input by default
}: PaginationProps) => {
  // Page input state
  const [jumpValue, setJumpValue] = useState<number | null>(null);

  // Don't show pagination if only one page
  if (totalPages <= 1) return null;

  // Handle jump
  const handleJump = () => {
    if (jumpValue !== null) {
      // NumberInput already ensures value is in range
      if (
        jumpValue >= 1 &&
        jumpValue <= totalPages &&
        jumpValue !== currentPage
      ) {
        onPageChange(jumpValue);
      }
      // Clear input after jump
      setJumpValue(null);
    }
  };

  // Handle enter key to jump
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      handleJump();
    }
  };

  return (
    <>
      <div className="flex flex-wrap items-center gap-2">
        <HeroUIPagination
          showControls
          color="primary"
          page={currentPage}
          radius="md"
          size="md"
          total={totalPages}
          onChange={onPageChange}
          {...props}
          classNames={
            {
              // wrapper: "gap-1",
              // prev: "text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900",
              // next: "text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900",
              // item: "text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-900",
              // cursor: "bg-primary-500 text-white",
            }
          }
        />

        {/* Page jump input */}
        {showJumper && totalPages > 5 && (
          <div className="items-center ml-2 gap-1 hidden sm:flex">
            <span className="text-sm text-gray-500 dark:text-gray-400">
              Go to
            </span>
            <NumberInput
              hideStepper
              className="w-16"
              classNames={{
                mainWrapper: "h-8",
                input: "h-8",
                inputWrapper: "h-8 min-h-8",
              }}
              maxValue={totalPages}
              minValue={1}
              placeholder={currentPage.toString()}
              radius="sm"
              size="sm"
              value={jumpValue}
              onKeyDown={handleKeyDown}
              onValueChange={(value) => setJumpValue(value)}
            />
            <span className="text-sm text-gray-500 dark:text-gray-400">page</span>
            <Button
              color="primary"
              size="sm"
              variant="light"
              onClick={handleJump}
            >
              Go
            </Button>
          </div>
        )}
      </div>
    </>
  );
};

export default Pagination;
