import React, { ReactNode, useMemo, useState } from "react";
import {
  Table,
  TableHeader,
  TableBody,
  TableColumn,
  TableRow,
  TableCell,
  SortDescriptor,
  Selection,
  Key,
} from "@heroui/table";
import { Button } from "@heroui/button";
import {
  Dropdown,
  DropdownTrigger,
  DropdownMenu,
  DropdownItem,
  DropdownSection,
} from "@heroui/dropdown";
import { Tooltip } from "@heroui/tooltip";
import { Input } from "@heroui/input";

import LoadingSpinner from "./LoadingSpinner";
import ErrorDisplay from "./ErrorDisplay";
import SearchForm from "./SearchForm";
import { PlusIcon } from "./icons";
import Pagination from "./Pagination";

// ChevronDownIcon component
const ChevronDownIcon = ({
  strokeWidth = 1.5,
  ...otherProps
}: { strokeWidth?: number } & React.SVGProps<SVGSVGElement>) => {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      focusable="false"
      height="1em"
      role="presentation"
      viewBox="0 0 24 24"
      width="1em"
      {...otherProps}
    >
      <path
        d="m19.92 8.95-6.52 6.52c-.77.77-2.03.77-2.8 0L4.08 8.95"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeMiterlimit={10}
        strokeWidth={strokeWidth}
      />
    </svg>
  );
};

// Column definition types
export interface Column {
  key: string;
  label: string;
  allowsSorting?: boolean;
}

// Add button property types
export interface AddButtonProps {
  tooltip?: string;
  onClick: () => void;
  label?: string;
  isIconOnly?: boolean;
}

// DataTable component property types
export interface DataTableProps<T> {
  title: string;
  columns: Column[];
  data: T[];
  total?: number;
  page: number;
  pages?: number;
  pageSize?: number;
  onPageChange: (page: number) => void;
  sortDescriptor: SortDescriptor;
  onSortChange: (descriptor: SortDescriptor) => void;
  isLoading?: boolean;
  error?: Error;
  searchTerm?: string;
  searchPlaceholder?: string;
  onSearch?: (e: React.FormEvent) => void;
  setSearchTerm?: (value: string) => void;
  renderCell: (item: T, columnKey: string) => ReactNode;
  emptyContent?: ReactNode;
  addButtonProps?: AddButtonProps;
  visibleColumns?: Selection;
  setVisibleColumns?: (selection: Selection) => void;
  selectedSize?: number;
  setSize?: (size: number) => void;
  autoSearchDelay?: number;
  removeWrapper?: boolean;
  topActionContent?: ReactNode;
  minPageSize?: number; // Minimum page size
  maxPageSize?: number; // Maximum page size
  // Multi-select related props
  selectionMode?: "none" | "single" | "multiple";
  selectedKeys?: Selection;
  onSelectionChange?: (keys: Set<Key>) => void;
  selectionToolbarContent?: ReactNode;
  showJumper?: boolean;
  showCustomPageSize?: boolean;
}

// Generic DataTable component
export const DataTable = <T extends { id?: number | string }>({
  title,
  columns,
  data,
  total,
  page,
  pages = 1,
  pageSize = 10,
  onPageChange,
  sortDescriptor,
  onSortChange,
  isLoading = false,
  error,
  searchTerm = "",
  searchPlaceholder = "Search...",
  onSearch,
  setSearchTerm,
  renderCell,
  emptyContent,
  addButtonProps,
  visibleColumns = new Set(columns.map((col) => col.key)),
  setVisibleColumns,
  selectedSize = pageSize,
  setSize,
  autoSearchDelay = 0,
  removeWrapper = false,
  topActionContent,
  minPageSize = 5,
  maxPageSize = 100,
  // Multi-select related props
  selectionMode = "none",
  selectedKeys,
  onSelectionChange,
  selectionToolbarContent,
  showJumper = true,
  showCustomPageSize = true,
}: DataTableProps<T>) => {
  // Get header columns
  const headerColumns = React.useMemo(() => {
    if (visibleColumns === "all") return columns;

    return columns.filter((column) =>
      Array.from(visibleColumns).includes(column.key),
    );
  }, [visibleColumns, columns]);

  // Rows per page
  const pageSizeOptions = [5, 10, 15, 30, 50];
  const [pageSizeSelectedKeys, setPageSizeSelectedKeys] = useState(
    pageSizeOptions.includes(selectedSize)
      ? new Set([selectedSize.toString()])
      : new Set(["custom"]),
  );

  // Custom page size
  const [customPageSize, setCustomPageSize] = useState<string>(
    selectedSize.toString(),
  );

  // Validate custom page size
  const validateCustomPageSize = (value: string): boolean => {
    return (
      value.match(/^\d+$/) &&
      Number(value) >= minPageSize &&
      Number(value) <= maxPageSize
    );
  };

  // Check if custom page size is invalid
  const isInvalidCustomPageSize = useMemo(() => {
    return !validateCustomPageSize(customPageSize);
  }, [customPageSize, minPageSize, maxPageSize]);

  // Handle apply custom page size
  const applyCustomPageSize = () => {
    if (isInvalidCustomPageSize) {
      return;
    }

    const customPageSizeNumber = Number(customPageSize);

    setSize?.(customPageSizeNumber);
    if (pageSizeOptions.includes(customPageSizeNumber)) {
      setPageSizeSelectedKeys(new Set([customPageSize]));
    } else {
      setPageSizeSelectedKeys(new Set(["custom"]));
    }

    setCustomPageSize(customPageSize);
    onPageChange(1);
  };

  // Handle enter key for custom page size
  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      applyCustomPageSize();
    }
  };

  // Bottom content area
  const bottomContent = React.useMemo(() => {
    return (
      <div className="py-2 px-2 flex justify-between items-center flex-col gap-3">
        {pages > 1 && (
          <Pagination
            currentPage={page}
            showJumper={showJumper}
            totalPages={pages}
            onPageChange={onPageChange}
          />
        )}
      </div>
    );
  }, [
    pages,
    page,
    onPageChange,
    selectedSize,
    setSize,
    total,
    data.length,
    pageSize,
    customPageSize,
    minPageSize,
    maxPageSize,
  ]);

  // Top content area
  const topContent = React.useMemo(() => {
    return (
      <div className="flex justify-between flex-col gap-3 w-full">
        <div className="flex justify-between gap-3 items-end w-full">
          {setSearchTerm && onSearch && (
            <SearchForm
              autoSearchDelay={autoSearchDelay}
              handleSearch={onSearch}
              placeholder={searchPlaceholder}
              searchTerm={searchTerm}
              setSearchTerm={setSearchTerm}
            />
          )}
          <div className="flex gap-3">
            {setVisibleColumns && (
              <Dropdown>
                <DropdownTrigger className="hidden sm:flex">
                  <Button
                    endContent={<ChevronDownIcon className="text-small" />}
                    variant="flat"
                  >
                    Columns
                  </Button>
                </DropdownTrigger>
                <DropdownMenu
                  disallowEmptySelection
                  aria-label="Visible Columns"
                  closeOnSelect={false}
                  selectedKeys={visibleColumns}
                  selectionMode="multiple"
                  onSelectionChange={setVisibleColumns}
                >
                  {columns.map((column) => (
                    <DropdownItem key={column.key} className="capitalize">
                      {column.label}
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
            )}
            {topActionContent}
            {addButtonProps && (
              <Tooltip
                color="primary"
                content={addButtonProps.tooltip || "Add"}
              >
                <Button
                  color="primary"
                  isIconOnly={addButtonProps.isIconOnly || false}
                  onPress={addButtonProps.onClick}
                >
                  {addButtonProps.label || <PlusIcon />}
                </Button>
              </Tooltip>
            )}
          </div>
        </div>
        {pages > 1 && (
          <div className="flex justify-between items-center flex-wrap gap-2 w-full">
            {setSize && (
              <Dropdown>
                <DropdownTrigger>
                  {/* <Button
                    className="text-default-400 text-small"
                    endContent={<ChevronDownIcon className="text-small" />}
                    variant="light"
                  >
                    Rows per page: {selectedSize}
                  </Button> */}
                  <div className="flex items-center gap-1 text-default-400 text-small ml-2 cursor-pointer">
                    <span>Rows per page: {selectedSize}</span>
                    <ChevronDownIcon />
                  </div>
                </DropdownTrigger>
                <DropdownMenu
                  disallowEmptySelection
                  aria-label="Rows per page"
                  closeOnSelect={true}
                  selectedKeys={pageSizeSelectedKeys}
                  selectionMode="single"
                  onSelectionChange={(e) => {
                    const key = e.currentKey;

                    // No need to handle "custom" option, integrated into dropdown
                    if (key && key !== "custom") {
                      setSize(Number(key));
                      setPageSizeSelectedKeys(new Set([key]));
                      setCustomPageSize(key);
                      onPageChange(1);
                    }
                  }}
                >
                  <DropdownSection showDivider title="Preset Sizes">
                    {pageSizeOptions.map((size) => (
                      <DropdownItem key={size}>{size}</DropdownItem>
                    ))}
                  </DropdownSection>
                  {showCustomPageSize && (
                    <DropdownSection title="Custom">
                      <DropdownItem
                        isReadOnly
                        endContent={
                          <Button
                            color="primary"
                            size="sm"
                            variant="flat"
                            onClick={(e) => {
                              e.preventDefault();
                              e.stopPropagation();
                              applyCustomPageSize();
                            }}
                          >
                            Apply
                          </Button>
                        }
                        startContent={
                          <Input
                            aria-label="Custom page size"
                            className="w-20"
                            color={
                              isInvalidCustomPageSize ? "danger" : "default"
                            }
                            errorMessage={`Page size must be between ${minPageSize} and ${maxPageSize} `}
                            isInvalid={isInvalidCustomPageSize}
                            placeholder={`${minPageSize}-${maxPageSize}`}
                            radius="sm"
                            size="sm"
                            value={customPageSize}
                            onKeyDown={handleKeyDown}
                            onValueChange={setCustomPageSize}
                          />
                        }
                        textValue="Custom page size"
                      />
                    </DropdownSection>
                  )}
                </DropdownMenu>
              </Dropdown>
            )}

            <span className="text-default-400 text-small mr-2">
              Total: {total || data.length} records
            </span>
          </div>
        )}
        {selectionMode === "multiple" &&
          selectedKeys &&
          selectedKeys.size > 0 && (
            <div className="w-full flex justify-between items-center">
              <span className="text-default-400 text-small ml-2">
                Selected {selectedKeys.size} items
              </span>
              {selectionToolbarContent}
            </div>
          )}
      </div>
    );
  }, [
    searchTerm,
    searchPlaceholder,
    setSearchTerm,
    onSearch,
    columns,
    visibleColumns,
    setVisibleColumns,
    addButtonProps,
    autoSearchDelay,
    selectionMode,
    selectedKeys,
    selectionToolbarContent,
    pages,
    setSize,
    selectedSize,
    pageSizeSelectedKeys,
    customPageSize,
    isInvalidCustomPageSize,
    minPageSize,
    maxPageSize,
    applyCustomPageSize,
    handleKeyDown,
  ]);

  // Render table
  const renderTable = () => (
    <Table
      isHeaderSticky
      aria-label={title}
      bottomContent={bottomContent}
      bottomContentPlacement="outside"
      // classNames={{
      //   wrapper: "max-h-[600px]",
      // }}
      removeWrapper={removeWrapper}
      selectedKeys={selectedKeys}
      selectionMode={selectionMode}
      sortDescriptor={sortDescriptor}
      topContent={topContent}
      topContentPlacement="outside"
      onSelectionChange={(selection) => {
        if (selection === "all") {
          onSelectionChange?.(new Set(data.map((item) => item.id?.toString())));
        } else {
          onSelectionChange?.(selection);
        }
      }}
      onSortChange={onSortChange}
    >
      <TableHeader columns={headerColumns}>
        {(column) => (
          <TableColumn key={column.key} allowsSorting={column.allowsSorting}>
            {column.label}
          </TableColumn>
        )}
      </TableHeader>
      <TableBody
        emptyContent={emptyContent || <p className="text-xl">No data</p>}
        isLoading={isLoading}
        items={data}
        loadingContent={
          <div className="flex justify-center py-8">
            <LoadingSpinner size="large" />
          </div>
        }
      >
        {(item) => (
          <TableRow key={item.id?.toString()}>
            {(columnKey) => (
              <TableCell>{renderCell(item, columnKey.toString())}</TableCell>
            )}
          </TableRow>
        )}
      </TableBody>
    </Table>
  );

  // Main render logic
  return (
    <div className="w-full">
      {error ? (
        <ErrorDisplay error={new Error(error?.message || `Loading${title}failed`)} />
      ) : (
        renderTable()
      )}
    </div>
  );
};
