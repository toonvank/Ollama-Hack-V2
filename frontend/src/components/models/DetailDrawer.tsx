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
import { addToast } from "@heroui/toast";

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
  const [isRescanning, setIsRescanning] = useState(false);

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

  const handleRescan = async (
    scope: "linked" | "available" | "recent" | "all",
  ) => {
    if (!id) return;
    setIsRescanning(true);
    try {
      const res = await aiModelApi.rescanModel(Number(id), {
        scope,
        limit: scope === "linked" ? 500 : 200,
        clear_health: true,
      });
      addToast({
        title: "Rescan queued",
        description:
          res.message ||
          `Queued ${res.queued} endpoint retests (${res.scope})`,
        color: "success",
      });
      // Refresh drawer after a short delay so early results can show
      setTimeout(() => refetch(), 3000);
    } catch (e) {
      console.error("Rescan failed:", e);
      addToast({
        title: "Rescan failed",
        description: (e as Error)?.message || "Could not queue rescan",
        color: "danger",
      });
    } finally {
      setIsRescanning(false);
    }
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
          <div className="flex items-center gap-2">
            <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
              {endpoint.url}
            </span>
            {(endpoint.endpoint_type || "ollama") === "openai" && (
              <span className="inline-flex items-center rounded-full bg-purple-100 px-1.5 py-0.5 text-xs font-medium text-purple-800 dark:bg-purple-900 dark:text-purple-200">
                SGLang
              </span>
            )}
          </div>
        );
      case "status":
        return (
          <div className="flex flex-col gap-0.5 items-start">
            <Popover showArrow placement="top">
              <PopoverTrigger>
                <Button isIconOnly className="p-0 h-auto w-auto" variant="light">
                  <StatusBadge status={endpoint.status} />
                </Button>
              </PopoverTrigger>
              <PopoverContent>
                <StatusTimeline
                  performanceTests={endpoint.model_performances || []}
                  type="model"
                />
              </PopoverContent>
            </Popover>
            {(endpoint.host_status || endpoint.model_on_host) && (
              <span className="text-[10px] text-gray-500 dark:text-gray-400 whitespace-nowrap">
                host:{endpoint.host_status || "?"} · model:
                {endpoint.model_on_host || "?"}
              </span>
            )}
          </div>
        );
      case "performance":
        // Show last measured TPS even if host is currently down (stale metrics)
        return endpoint.token_per_second != null ? (
          <div className="flex flex-col gap-0.5">
            <Tooltip content="Last measured generation speed (may be stale if host is down)">
              <span>
                {endpoint.token_per_second.toFixed(1)} tps
                {endpoint.status !== AIModelStatusEnum.AVAILABLE ? (
                  <span className="text-xs text-warning ml-1">(stale)</span>
                ) : null}
              </span>
            </Tooltip>
            {endpoint.max_connection_time != null && (
              <Tooltip content="Last measured time to first stream chunk">
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {endpoint.max_connection_time.toFixed(2)}s reply
                </span>
              </Tooltip>
            )}
          </div>
        ) : (
          "Not tested"
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
              <div className="flex flex-row gap-2 items-center justify-end w-auto flex-wrap">
                <StatusBadge
                  status={
                    (model.avaliable_endpoint_count ?? 0) > 0
                      ? AIModelStatusEnum.AVAILABLE
                      : AIModelStatusEnum.UNAVAILABLE
                  }
                />
                <Tooltip content="Retest hosts already linked to this model (fixes stale unavailable)">
                  <Button
                    size="sm"
                    color="primary"
                    variant="flat"
                    isLoading={isRescanning}
                    onPress={() => handleRescan("linked")}
                  >
                    Rescan linked
                  </Button>
                </Tooltip>
                <Tooltip content="Priority-test recent/new endpoints to discover this model">
                  <Button
                    size="sm"
                    color="secondary"
                    variant="flat"
                    isLoading={isRescanning}
                    onPress={() => handleRescan("recent")}
                  >
                    Scan new pool
                  </Button>
                </Tooltip>
                <Tooltip content="Priority-test healthy general pool (full tags probe)">
                  <Button
                    size="sm"
                    variant="bordered"
                    isLoading={isRescanning}
                    onPress={() => handleRescan("available")}
                  >
                    Scan general pool
                  </Button>
                </Tooltip>
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
                    Routable / linked endpoints
                  </h4>
                  <p>
                    <span className="font-semibold">
                      {model.avaliable_endpoint_count ?? 0}
                    </span>
                    {" routable · "}
                    <span>{model.total_endpoint_count ?? 0} total links</span>
                  </p>
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    Routable = host available + model available + health OK.
                    List “Endpoints” column uses routable only — stale links can
                    still show old tps until Rescan.
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
              {(model.endpoints?.items?.length ?? 0) === 0 ? (
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
                  pages={Math.max(
                    1,
                    Math.ceil((model.endpoints?.total || 0) / size),
                  )}
                  removeWrapper={true}
                  renderCell={renderCell}
                  selectedSize={size}
                  setSize={setSize}
                  showCustomPageSize={false}
                  title="Available Endpoints"
                  total={model.endpoints?.total}
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
