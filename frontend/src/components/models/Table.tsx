import React from "react";
import { Button } from "@heroui/button";
import { SortDescriptor } from "@heroui/table";
import { Tooltip } from "@heroui/tooltip";

import { DataTable } from "@/components/DataTable";
import StatusBadge from "@/components/StatusBadge";
import {
  AIModelInfoWithEndpointCount,
  AIModelStatusEnum,
  SortOrder,
} from "@/types";
import { EyeIcon } from "@/components/icons";
import { Switch } from "@heroui/switch";

interface ModelTableProps {
  models: AIModelInfoWithEndpointCount[] | undefined;
  isLoading: boolean;
  error: any;
  page: number;
  pageSize: number;
  searchTerm: string;
  orderBy: string | undefined;
  order: SortOrder | undefined;
  setSearchTerm: (term: string) => void;
  setPageSize: (size: number) => void;
  setOrderBy?: (orderBy: string) => void;
  setOrder?: (order: SortOrder) => void;
  onOpenModelDetail: (modelId: number) => void;
  onPageChange: (page: number) => void;
  onSearch: (e: React.FormEvent) => void;
  onToggleModel?: (modelId: number, enabled: boolean) => void;
  totalPages?: number;
  totalItems?: number;
}

const ModelTable: React.FC<ModelTableProps> = ({
  models,
  isLoading,
  error,
  page,
  pageSize,
  searchTerm,
  orderBy,
  order,
  setSearchTerm,
  setPageSize,
  setOrderBy,
  setOrder,
  onOpenModelDetail,
  onPageChange,
  onSearch,
  onToggleModel,
  totalPages,
  totalItems,
}) => {
  // Get model status
  const getModelStatus = (
    model: AIModelInfoWithEndpointCount,
  ): AIModelStatusEnum => {
    if (!model.enabled) {
      return AIModelStatusEnum.UNAVAILABLE;
    }
    if (model.endpoints === 0) {
      return AIModelStatusEnum.UNAVAILABLE;
    }

    return AIModelStatusEnum.AVAILABLE;
  };

  // Sort state
  const [sortDescriptor, setSortDescriptor] = React.useState<SortDescriptor>({
    column: orderBy || "id",
    direction:
      order === SortOrder.ASC
        ? "ascending"
        : order === SortOrder.DESC
          ? "descending"
          : "ascending",
  });

  // Define table columns
  const columns = [
    { key: "id", label: "ID", allowsSorting: true },
    { key: "name", label: "Name", allowsSorting: true },
    { key: "tag", label: "Tag", allowsSorting: true },
    { key: "endpoints", label: "Endpoints" },
    { key: "enabled", label: "Enabled" },
    { key: "status", label: "Status" },
    { key: "created_at", label: "Created At", allowsSorting: true },
    { key: "actions", label: "Actions" },
  ];

  // Handle sort
  const handleSort = (descriptor: SortDescriptor) => {
    setSortDescriptor(descriptor);
    if (descriptor.column) {
      const newOrderBy = descriptor.column.toString();
      const newOrder =
        descriptor.direction === "ascending" ? SortOrder.ASC : SortOrder.DESC;

      // Update parent sort state
      if (orderBy !== newOrderBy || order !== newOrder) {
        setOrderBy &&
          typeof setOrderBy === "function" &&
          setOrderBy(newOrderBy);
        setOrder && typeof setOrder === "function" && setOrder(newOrder);
      }
    }
  };

  // Render cell content
  const renderCell = (
    model: AIModelInfoWithEndpointCount,
    columnKey: string,
  ) => {
    switch (columnKey) {
      case "id":
        return model.id;
      case "name":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {model.name}
          </span>
        );
      case "tag":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {model.tag}
          </span>
        );
      case "endpoints":
        return model.endpoints || 0;
      case "enabled":
        return (
          <Switch
            isSelected={model.enabled}
            size="sm"
            color="success"
            onChange={(e) => {
              if (model.id && onToggleModel) {
                onToggleModel(model.id, e.target.checked);
              }
            }}
          />
        );
      case "status":
        return <StatusBadge status={getModelStatus(model)} />;
      case "created_at":
        return model.created_at
          ? new Date(model.created_at + "Z").toLocaleString()
          : "-";
      case "actions":
        return (
          <div className="relative flex items-center gap-2">
            <Tooltip content="View Model">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => {
                  if (model.id) {
                    onOpenModelDetail(model.id);
                  }
                }}
              >
                <EyeIcon />
              </Button>
            </Tooltip>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <DataTable<AIModelInfoWithEndpointCount>
      autoSearchDelay={1000}
      columns={columns}
      data={models || []}
      emptyContent={
        <p className="text-xl text-gray-600 dark:text-gray-400">No model data</p>
      }
      error={error}
      isLoading={isLoading}
      page={page}
      pages={totalPages}
      renderCell={renderCell}
      searchPlaceholder="Search models..."
      searchTerm={searchTerm}
      selectedSize={pageSize}
      setSearchTerm={setSearchTerm}
      setSize={setPageSize}
      sortDescriptor={sortDescriptor}
      title="Model List"
      total={totalItems}
      onPageChange={onPageChange}
      onSearch={onSearch}
      onSortChange={handleSort}
    />
  );
};

export default ModelTable;
