import React from "react";
import { Button } from "@heroui/button";
import { SortDescriptor, Selection, Key } from "@heroui/table";
import { Tooltip } from "@heroui/tooltip";
import { Spinner } from "@heroui/spinner";

import { DataTable } from "@/components/DataTable";
import StatusBadge from "@/components/StatusBadge";
import {
  EndpointWithAIModelCount,
  EndpointStatusEnum,
  SortOrder,
} from "@/types";
import {
  DeleteIcon,
  EditIcon,
  EyeIcon,
  PlusIcon,
  TestIcon,
} from "@/components/icons";

interface EndpointTableProps {
  endpoints: EndpointWithAIModelCount[] | undefined;
  isLoading: boolean;
  error: any;
  page: number;
  pageSize: number;
  searchTerm: string;
  orderBy: string | undefined;
  order: SortOrder | undefined;
  visibleColumns: Selection;
  setVisibleColumns: (selection: Selection) => void;
  setSearchTerm: (term: string) => void;
  setPageSize: (size: number) => void;
  setOrderBy?: (orderBy: string) => void;
  setOrder?: (order: SortOrder) => void;
  onDeleteEndpoint: (id: number) => void;
  onEditEndpoint: (endpoint: EndpointWithAIModelCount) => void;
  onOpenEndpointDetail: (endpointId: number) => void;
  onCreateEndpoint: () => void;
  onPageChange: (page: number) => void;
  onSearch: (e: React.FormEvent) => void;
  onTestEndpoint: (id: number) => void;
  isAdmin: boolean;
  totalPages?: number;
  totalItems?: number;
  testingEndpointIds: number[];
  selectionMode?: "none" | "single" | "multiple";
  selectedKeys?: Selection;
  onSelectionChange?: (keys: Set<Key>) => void;
  selectionToolbarContent?: React.ReactNode;
}

const EndpointTable: React.FC<EndpointTableProps> = ({
  endpoints,
  isLoading,
  error,
  page,
  pageSize,
  searchTerm,
  orderBy,
  order,
  visibleColumns,
  setVisibleColumns,
  setSearchTerm,
  setPageSize,
  setOrderBy,
  setOrder,
  onDeleteEndpoint,
  onEditEndpoint,
  onOpenEndpointDetail,
  onCreateEndpoint,
  onPageChange,
  onSearch,
  onTestEndpoint,
  isAdmin,
  totalPages,
  totalItems,
  testingEndpointIds,
  selectionMode = "none",
  selectedKeys,
  onSelectionChange,
  selectionToolbarContent,
}) => {
  // Get endpoint status
  const getEndpointStatus = (
    endpoint: EndpointWithAIModelCount,
  ): EndpointStatusEnum => {
    if (endpoint.recent_performances.length === 0) {
      return EndpointStatusEnum.UNAVAILABLE;
    }

    return endpoint.recent_performances[0].status;
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
    { key: "url", label: "URL", allowsSorting: true },
    { key: "status", label: "Status", allowsSorting: true },
    { key: "models", label: "AI Models" },
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

  const EndpointActionCell = ({
    endpoint,
    isTesting,
    onTestEndpoint,
    onEditEndpoint,
    onDeleteEndpoint,
    onOpenEndpointDetail,
    isAdmin,
  }) => {
    return (
      <div className="relative flex items-center gap-2">
        <Tooltip content="View Endpoint">
          <Button
            isIconOnly
            className="text-default-400 active:opacity-50 text-lg"
            variant="light"
            onPress={() => {
              if (endpoint.id) {
                onOpenEndpointDetail(endpoint.id);
              }
            }}
          >
            <EyeIcon />
          </Button>
        </Tooltip>
        {isAdmin && (
          <>
            <Tooltip content="Test Endpoint">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                isLoading={isTesting}
                spinner={<Spinner color="warning" size="sm" variant="wave" />}
                variant="light"
                onPress={() => {
                  if (endpoint.id) {
                    onTestEndpoint(endpoint.id);
                  }
                }}
              >
                <TestIcon />
              </Button>
            </Tooltip>
            <Tooltip content="Edit Endpoint">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => onEditEndpoint(endpoint)}
              >
                <EditIcon />
              </Button>
            </Tooltip>
            <Tooltip color="danger" content="Delete Endpoint">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => {
                  if (endpoint.id) {
                    onDeleteEndpoint(endpoint.id);
                  }
                }}
              >
                <DeleteIcon />
              </Button>
            </Tooltip>
          </>
        )}
      </div>
    );
  };

  // Render cell content
  const renderCell = (
    endpoint: EndpointWithAIModelCount,
    columnKey: string,
  ) => {
    switch (columnKey) {
      case "id":
        return endpoint.id;
      case "name":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {endpoint.name}
          </span>
        );
      case "url":
        return (
          <div className="flex flex-col">
            <p className="text-bold text-small">{endpoint.url}</p>
          </div>
        );
      case "status":
        return <StatusBadge status={getEndpointStatus(endpoint)} />;
      case "models":
        return (
          <>
            {endpoint.avaliable_ai_model_count} /{" "}
            {endpoint.total_ai_model_count}
          </>
        );
      case "created_at":
        return endpoint.created_at
          ? new Date(endpoint.created_at + "Z").toLocaleString()
          : "-";
      case "actions":
        return (
          <EndpointActionCell
            endpoint={endpoint}
            isAdmin={isAdmin}
            isTesting={testingEndpointIds.includes(endpoint.id)}
            onDeleteEndpoint={onDeleteEndpoint}
            onEditEndpoint={onEditEndpoint}
            onOpenEndpointDetail={onOpenEndpointDetail}
            onTestEndpoint={() => {
              if (endpoint.id) {
                onTestEndpoint(endpoint.id);
              }
            }}
          />
        );
      default:
        return null;
    }
  };

  return (
    <DataTable<EndpointWithAIModelCount>
      key={testingEndpointIds.join(",")}
      addButtonProps={
        isAdmin
          ? {
              tooltip: "Add Endpoint",
              onClick: onCreateEndpoint,
              isIconOnly: true,
            }
          : undefined
      }
      columns={columns}
      data={endpoints || []}
      emptyContent={
        <>
          <p className="text-xl">No endpoint data</p>
          {isAdmin && (
            <Tooltip
              color="primary"
              content="Add your first endpoint"
              placement="bottom"
            >
              <Button
                isIconOnly
                className="mt-4"
                color="primary"
                onPress={onCreateEndpoint}
              >
                <PlusIcon />
              </Button>
            </Tooltip>
          )}
        </>
      }
      error={error}
      isLoading={isLoading}
      page={page}
      pages={totalPages}
      renderCell={renderCell}
      searchPlaceholder="Search endpoints..."
      searchTerm={searchTerm}
      selectedKeys={selectedKeys}
      selectedSize={pageSize}
      selectionMode={selectionMode}
      selectionToolbarContent={selectionToolbarContent}
      setSearchTerm={setSearchTerm}
      setSize={setPageSize}
      setVisibleColumns={setVisibleColumns}
      sortDescriptor={sortDescriptor}
      title="Endpoint List"
      total={totalItems}
      visibleColumns={visibleColumns}
      onPageChange={onPageChange}
      onSearch={onSearch}
      onSelectionChange={onSelectionChange}
      onSortChange={handleSort}
    />
  );
};

export default EndpointTable;
