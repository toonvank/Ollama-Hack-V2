import {
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerHeader,
} from "@heroui/drawer";
import { Popover, PopoverContent, PopoverTrigger } from "@heroui/popover";
import { Card, CardHeader, CardBody } from "@heroui/card";
import { Divider } from "@heroui/divider";
import { Tooltip } from "@heroui/tooltip";
import React, { useState, useEffect } from "react";
import { Button } from "@heroui/button";

import { LeftArrowIcon } from "../icons";
import StatusTimeline from "../StatusTimeline";

import { useCustomQuery } from "@/hooks";
import { aiModelApi } from "@/api";
import {
  AIModelInfoWithEndpoint,
  ModelFromEndpointInfo,
  AIModelStatusEnum,
} from "@/types";
import StatusBadge from "@/components/StatusBadge";
import { DataTable } from "@/components/DataTable";
import LoadingSpinner from "@/components/LoadingSpinner";
import ErrorDisplay from "@/components/ErrorDisplay";

interface ModelDetailProps {
  id: string | number;
  isOpen: boolean;
  onClose: () => void;
}

const ModelDetailDrawer = ({ id, isOpen, onClose }: ModelDetailProps) => {
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(5);

  // Get model details
  const {
    data: model,
    isLoading,
    error,
    refetch,
  } = useCustomQuery<AIModelInfoWithEndpoint>(
    ["model-drawer", id, page, size],
    () => aiModelApi.getAIModelById(Number(id), page, size),
    { staleTime: 30000, enabled: !!id && isOpen },
  );

  // Refetch data when drawer opens
  useEffect(() => {
    if (isOpen && id) {
      refetch();
    }
  }, [isOpen, id, refetch]);

  // Handle page change
  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  // Format date
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  // Define table columns
  const columns = [
    // { key: "endpoint", label: "Endpoints" },
    { key: "url", label: "URL" },
    { key: "status", label: "Status" },
    { key: "performance", label: "Performance" },
    // { key: "actions", label: "Actions" },
  ];

  // Render cell content
  const renderCell = (endpoint: ModelFromEndpointInfo, columnKey: string) => {
    switch (columnKey) {
      case "url":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {endpoint.url}
          </span>
        );
      case "status":
        return (
          <Popover showArrow placement="top">
            <PopoverTrigger>
              <Button isIconOnly className="p-0 h-auto w-auto" variant="light">
                <StatusBadge status={endpoint.status} />
              </Button>
            </PopoverTrigger>
            <PopoverContent>
              <StatusTimeline
                performanceTests={endpoint.model_performances}
                type="model"
              />
            </PopoverContent>
          </Popover>
        );
      case "performance":
        return endpoint.status === AIModelStatusEnum.AVAILABLE ? (
          <Tooltip content="Generation speed (tokens per second)">
            <div>
              {endpoint.token_per_second
                ? `${endpoint.token_per_second.toFixed(1)} tps`
                : "Not tested"}
            </div>
          </Tooltip>
        ) : (
          "Unavailable"
        );
      //   case "actions":
      //     return (
      //       <Button
      //         isIconOnly
      //         className="text-lg text-default-400 active:opacity-50"
      //         size="sm"
      //         variant="light"
      //       >
      //         <EyeIcon />
      //       </Button>
      //     );
      default:
        return null;
    }
  };

  // Render drawer content
  const renderContent = () => {
    if (isLoading) {
      return (
        <div className="flex justify-center py-8">
          <LoadingSpinner size="large" />
        </div>
      );
    }

    if (error) {
      return (
        <ErrorDisplay
          error={new Error((error as Error)?.message || "Failed to load model details")}
        />
      );
    }

    if (!model) {
      return (
        <div className="text-center py-8">
          <p>Model not found</p>
        </div>
      );
    }

    return (
      <>
        <div className="grid grid-cols-1 gap-6 mb-6">
          {/* Model info card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between gap-2 p-4">
              <h3 className="text-xl font-bold">Model Info</h3>
              <div className="flex flex-row gap-2 items-center justify-end w-auto">
                <StatusBadge
                  status={
                    model.avaliable_endpoint_count > 0
                      ? AIModelStatusEnum.AVAILABLE
                      : AIModelStatusEnum.UNAVAILABLE
                  }
                />
              </div>
            </CardHeader>
            <Divider />
            <CardBody className="p-4">
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    ID
                  </h4>
                  <p>{model.id}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Name
                  </h4>
                  <p>{model.name}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Tag
                  </h4>
                  <p>{model.tag}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Created At
                  </h4>
                  <p>
                    {model.created_at ? formatDate(model.created_at) : "Unknown"}
                  </p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Available Endpoints
                  </h4>
                  <p>
                    {model.avaliable_endpoint_count} /{" "}
                    {model.total_endpoint_count}
                  </p>
                </div>
              </div>
            </CardBody>
          </Card>

          {/* Available endpoints card */}
          <Card>
            <CardHeader className="p-4">
              <h3 className="text-xl font-bold">Available Endpoints</h3>
            </CardHeader>
            <Divider />
            <CardBody className="p-4">
              {model.endpoints.items.length === 0 ? (
                <div className="py-8 text-center">
                  <p className="text-gray-500 dark:text-gray-400">
                    No available endpoints
                  </p>
                </div>
              ) : (
                <DataTable
                  columns={columns}
                  data={model.endpoints.items}
                  emptyContent={
                    <p className="text-gray-500 dark:text-gray-400">
                      No available endpoints
                    </p>
                  }
                  isLoading={isLoading}
                  page={page}
                  pages={Math.ceil((model.endpoints.total || 0) / size)}
                  removeWrapper={true}
                  renderCell={renderCell}
                  selectedSize={size}
                  setSize={setSize}
                  showCustomPageSize={false}
                  title="Available Endpoints"
                  total={model.endpoints.total}
                  onPageChange={handlePageChange}
                />
              )}
            </CardBody>
          </Card>
        </div>
      </>
    );
  };

  return (
    <Drawer
      backdrop="blur"
      classNames={{
        base: "data-[placement=right]:sm:m-2 data-[placement=left]:sm:m-2 rounded-medium",
      }}
      isOpen={isOpen}
      placement="right"
      size="lg"
      onOpenChange={onClose}
    >
      <DrawerContent>
        {() => (
          <>
            <DrawerHeader className="absolute top-0 inset-x-0 z-50 flex flex-row gap-2 px-2 py-2 border-b border-default-200/50 justify-between bg-content1/50 backdrop-saturate-150 backdrop-blur-lg">
              <Tooltip content="Close">
                <Button
                  isIconOnly
                  className="text-default-400 active:opacity-50 text-lg"
                  variant="light"
                  onPress={onClose}
                >
                  <LeftArrowIcon />
                </Button>
              </Tooltip>
            </DrawerHeader>
            <DrawerBody className="pt-16">{renderContent()}</DrawerBody>
          </>
        )}
      </DrawerContent>
    </Drawer>
  );
};

export default ModelDetailDrawer;
