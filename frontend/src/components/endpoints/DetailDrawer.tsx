import {
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerHeader,
} from "@heroui/drawer";
import { Card, CardHeader, CardBody } from "@heroui/card";
import { Divider } from "@heroui/divider";
import { Chip } from "@heroui/chip";
import { Tooltip } from "@heroui/tooltip";
import React, { useState, useEffect } from "react";
import { Button } from "@heroui/button";

import { DeleteIcon, EditIcon, LeftArrowIcon } from "../icons";
import StatusTimeline from "../StatusTimeline";

import { useCustomQuery } from "@/hooks";
import { endpointApi } from "@/api";
import {
  EndpointWithAIModels,
  AIModelStatusEnum,
  EndpointStatusEnum,
} from "@/types";
import StatusBadge from "@/components/StatusBadge";
import { DataTable } from "@/components/DataTable";
import LoadingSpinner from "@/components/LoadingSpinner";
import ErrorDisplay from "@/components/ErrorDisplay";

interface EndpointDetailProps {
  id: string | number;
  isOpen: boolean;
  onClose: () => void;
  isAdmin: boolean;
  onDelete: (endpoint: EndpointWithAIModels) => void;
  onEdit: (endpoint: EndpointWithAIModels) => void;
}

const EndpointDetailDrawer = ({
  id,
  isOpen,
  onClose,
  isAdmin,
  onEdit,
  onDelete,
}: EndpointDetailProps) => {
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(5);

  // Fetch endpoint details
  const {
    data: endpoint,
    isLoading,
    error,
    refetch,
  } = useCustomQuery<EndpointWithAIModels>(
    ["endpoint-drawer", id, page, size],
    () => endpointApi.getEndpointById(Number(id), page, size),
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

  // Get endpoint status
  const getEndpointStatus = (): EndpointStatusEnum => {
    if (!endpoint || endpoint.recent_performances.length === 0) {
      return EndpointStatusEnum.UNAVAILABLE;
    }

    return endpoint.recent_performances[0].status;
  };

  // Format date
  const formatDate = (dateString: string) => {
    return new Date(dateString + "Z").toLocaleString();
  };

  // Define table columns
  const columns = [
    { key: "name", label: "Model" },
    { key: "tag", label: "Tag" },
    { key: "status", label: "Status" },
    { key: "performance", label: "Performance" },
  ];

  // Render cell content
  const renderCell = (model: any, columnKey: string) => {
    switch (columnKey) {
      case "name":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {model.name}
          </span>
        );
      case "status":
        return <StatusBadge status={model.status} />;
      case "tag":
        return (
          <Chip color="primary" size="sm" variant="flat">
            {model.tag}
          </Chip>
        );
      case "performance":
        return model.status === AIModelStatusEnum.AVAILABLE ? (
          <Tooltip content="Generation speed (tokens per second)">
            <div>
              {model.token_per_second
                ? `${model.token_per_second.toFixed(2)} tokens/s`
                : "Not tested"}
            </div>
          </Tooltip>
        ) : (
          "Unavailable"
        );
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
          error={new Error((error as Error)?.message || "Failed to load endpoint details")}
        />
      );
    }

    if (!endpoint) {
      return (
        <div className="text-center py-8">
          <p>Endpoint not found</p>
        </div>
      );
    }

    return (
      <>
        <div className="grid grid-cols-1 gap-6 mb-6">
          {/* Endpoint info card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between gap-2 p-4">
              <h3 className="text-xl font-bold">Endpoint Info</h3>
              <div className="flex flex-row gap-2 items-center justify-end w-auto">
                <StatusTimeline
                  performanceTests={endpoint.recent_performances}
                />
                <StatusBadge status={getEndpointStatus()} />
              </div>
            </CardHeader>
            <Divider />
            <CardBody className="p-4">
              <div className="space-y-4">
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    ID
                  </h4>
                  <p>{endpoint.id}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Name
                  </h4>
                  <p>{endpoint.name}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    URL
                  </h4>
                  <p className="break-all">{endpoint.url}</p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Ollama Version
                  </h4>
                  <p>
                    {endpoint.recent_performances.length > 0 &&
                    endpoint.recent_performances[0].ollama_version
                      ? endpoint.recent_performances[0].ollama_version
                      : "Unknown"}
                  </p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Created At
                  </h4>
                  <p>
                    {endpoint.created_at
                      ? formatDate(endpoint.created_at)
                      : "Unknown"}
                  </p>
                </div>
                <div>
                  <h4 className="text-sm text-gray-500 dark:text-gray-400">
                    Last Checked
                  </h4>
                  <p>
                    {endpoint.recent_performances.length > 0
                      ? formatDate(endpoint.recent_performances[0].created_at)
                      : "Unknown"}
                  </p>
                </div>
              </div>
            </CardBody>
          </Card>

          {/* AI model list card */}
          <Card>
            <CardHeader className="p-4">
              <h3 className="text-xl font-bold">Available Models</h3>
            </CardHeader>
            <Divider />
            <CardBody className="p-4">
              {endpoint.ai_models.items.length === 0 ? (
                <div className="py-8 text-center">
                  <p className="text-gray-500 dark:text-gray-400">
                    No AI models available on this endpoint
                  </p>
                </div>
              ) : (
                <DataTable
                  columns={columns}
                  data={endpoint.ai_models.items}
                  emptyContent={
                    <p className="text-gray-500 dark:text-gray-400">
                      No AI models available on this endpoint
                    </p>
                  }
                  isLoading={isLoading}
                  page={page}
                  pages={Math.ceil((endpoint.ai_models.total || 0) / size)}
                  removeWrapper={true}
                  renderCell={renderCell}
                  selectedSize={size}
                  setSize={setSize}
                  showCustomPageSize={false}
                  title="Available Models"
                  total={endpoint.ai_models.total}
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
        base: "data-[placement=right]:sm:m-2 data-[placement=left]:sm:m-2  rounded-medium",
      }}
      isOpen={isOpen}
      placement="right"
      size="lg"
      onOpenChange={onClose}
    >
      <DrawerContent>
        <>
          {/* <DrawerHeader className="flex flex-col gap-1">
              {`EndpointsDetails ${endpoint ? `- ${endpoint.name}` : ""}`}
            </DrawerHeader> */}
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
            <div>
              {isAdmin && (
                <>
                  <Tooltip content="Edit Endpoint">
                    <Button
                      isIconOnly
                      className="text-default-400 active:opacity-50 text-lg"
                      variant="light"
                      onPress={() => {
                        onEdit(endpoint);
                      }}
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
                          onDelete(endpoint);
                        }
                      }}
                    >
                      <DeleteIcon />
                    </Button>
                  </Tooltip>
                </>
              )}
            </div>
          </DrawerHeader>
          <DrawerBody className="pt-16">{renderContent()}</DrawerBody>
          {/* <DrawerFooter>
              <Button color="primary" variant="light" onPress={onCloseDrawer}>
                Close
              </Button>
            </DrawerFooter> */}
        </>
      </DrawerContent>
    </Drawer>
  );
};

export default EndpointDetailDrawer;
