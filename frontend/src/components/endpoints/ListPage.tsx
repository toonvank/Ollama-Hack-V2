import React, { useState, useRef, useEffect, useMemo } from "react";
import { addToast } from "@heroui/toast";
import { Key, Selection } from "@heroui/table";
import { Button } from "@heroui/button";

import EndpointDetailDrawer from "@/components/endpoints/DetailDrawer";
import CreateEndpointModal from "@/components/endpoints/CreateModal";
import EndpointTable from "@/components/endpoints/Table";
import EndpointEditModal from "@/components/endpoints/EditModal";
import { useAuth } from "@/contexts/AuthContext";
import {
  useCustomQuery,
  usePaginationUrlState,
  PaginationValidationConfig,
} from "@/hooks";
import { endpointApi } from "@/api";
import {
  EndpointWithAIModelCount,
  PageResponse,
  SortOrder,
  TaskStatusEnum,
} from "@/types";
import DashboardLayout from "@/layouts/Main";
import ErrorDisplay from "@/components/ErrorDisplay";
import { useDialog } from "@/contexts/DialogContext";
import { DeleteIcon, TestIcon } from "@/components/icons";

// Endpoint list page
const EndpointListPage = () => {
  const { isAdmin } = useAuth();
  const { confirm } = useDialog();

  // Validation config
  const [validationConfig, setValidationConfig] =
    useState<PaginationValidationConfig>({
      page: { min: 1 },
      pageSize: { min: 5, max: 100 },
      totalPages: 1,
      orderBy: {
        allowedFields: ["id", "name", "status", "created_at"],
        defaultField: "id",
      },
    });

  // Use URL params for state management instead of multiple useState
  const {
    page,
    pageSize,
    search: searchTerm,
    orderBy,
    order,
    setPage,
    setPageSize,
    setSearch: setSearchTerm,
    setOrderBy,
    setOrder,
  } = usePaginationUrlState(
    {
      page: 1,
      pageSize: 10,
      search: "",
      orderBy: "status",
      order: SortOrder.ASC,
    },
    validationConfig,
  );

  // Detail drawer state
  const [isDetailDrawerOpen, setIsDetailDrawerOpen] = useState(false);
  const [selectedEndpointId, setSelectedEndpointId] = useState<number | null>(
    null,
  );

  // Multi-select state
  const [selectedEndpointIds, setSelectedEndpointIds] = useState<Selection>(
    new Set([]),
  );

  // New state variables
  const INITIAL_VISIBLE_COLUMNS = [
    "id",
    "name",
    // "url",
    "status",
    "models",
    // "created_at",
    "actions",
  ];
  const [visibleColumns, setVisibleColumns] = useState<Selection>(
    new Set(INITIAL_VISIBLE_COLUMNS),
  );

  // Modal state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingEndpoint, setEditingEndpoint] =
    useState<EndpointWithAIModelCount | null>(null);

  // Test state management
  const [testingEndpointIds, setTestingEndpointIds] = useState<number[]>([]);
  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Fetch endpoint list
  const {
    data: endpoints,
    isLoading,
    error: endpointsError,
    refetch,
  } = useCustomQuery<PageResponse<EndpointWithAIModelCount>>(
    ["endpoints", page, searchTerm, orderBy, order, pageSize],
    () =>
      endpointApi.getEndpoints({
        page,
        size: pageSize,
        search: searchTerm,
        order_by: orderBy,
        order,
      }),
    { staleTime: 30000 },
  );

  // Update validation config when total pages change
  useEffect(() => {
    if (endpoints?.pages) {
      setValidationConfig((prev) => ({
        ...prev,
        totalPages: endpoints.pages,
      }));
    }
  }, [endpoints?.pages]);

  useEffect(() => {
    const running_endpoint_ids = endpoints?.items
      .filter((endpoint) => endpoint.task_status === TaskStatusEnum.RUNNING)
      .map((endpoint) => endpoint.id);

    if (running_endpoint_ids && running_endpoint_ids.length > 0) {
      let new_testing_endpoint_ids = new Set([
        ...testingEndpointIds,
        ...running_endpoint_ids,
      ]);

      setTestingEndpointIds(Array.from(new_testing_endpoint_ids));
    }
  }, [endpoints, setTestingEndpointIds]);

  // Handle search
  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    refetch();
  };

  // Handle page change
  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  // Handle selection change
  const handleSelectionChange = (keys: Set<Key>) => {
    setSelectedEndpointIds(keys);
  };

  // Handle batch test endpoints
  const handleBatchTestEndpoints = () => {
    if (!selectedEndpointIds || selectedEndpointIds.size === 0) return;

    const endpointIds = Array.from(selectedEndpointIds).map(Number);

    confirm(
      `Are you sure you want to test the selected ${endpointIds.length} endpoints?`,
      async () => {
        try {
          const result = await endpointApi.batchTestEndpoints({
            endpoint_ids: endpointIds,
          });

          // Add tested endpoints to the testing list
          setTestingEndpointIds((prev) => {
            const newTestingIds = new Set([...prev]);

            endpointIds.forEach((id) => {
              if (!prev.includes(id)) {
                newTestingIds.add(id);
              }
            });

            return Array.from(newTestingIds);
          });

          addToast({
            title: "Batch test triggered",
            description: `Success: ${result.success_count}, Failed: ${result.failed_count}`,
            color: "primary",
          });

          // Clear selection
          setSelectedEndpointIds(new Set([]));
        } catch (err) {
          addToast({
            title: "Batch test failed",
            description: (err as Error).message || "Please try again",
            color: "danger",
          });
        }
      },
      "Confirm Batch Test",
    );
  };

  // Handle batch delete endpoints
  const handleBatchDeleteEndpoints = () => {
    if (!selectedEndpointIds || selectedEndpointIds.size === 0) return;

    const endpointIds = Array.from(selectedEndpointIds).map(Number);

    confirm(
      `Are you sure you want to delete the selected ${endpointIds.length} endpoints? This action cannot be undone.`,
      async () => {
        try {
          const result = await endpointApi.batchDeleteEndpoints({
            endpoint_ids: endpointIds,
          });

          addToast({
            title: "Batch delete successful",
            description: `Success: ${result.success_count}, Failed: ${result.failed_count}`,
            color: "success",
          });

          // Clear selection
          setSelectedEndpointIds(new Set([]));

          // Refresh list
          refetch();
        } catch (err) {
          addToast({
            title: "Batch delete failed",
            description: (err as Error).message || "Please try again",
            color: "danger",
          });
        }
      },
      "Confirm Batch Delete",
    );
  };

  // Handle delete endpoint
  const handleDeleteEndpoint = async (id: number) => {
    confirm("Are you sure you want to delete this endpoint? This action cannot be undone.", async () => {
      try {
        await endpointApi.deleteEndpoint(id);
        refetch();
        setIsEditModalOpen(false);
      } catch (err) {
        addToast({
          title: "Failed to delete endpoint",
          description: (err as Error).message || "Please try again",
          color: "danger",
        });
      }
    });
  };

  // Handle test endpoint
  const handleTestEndpoint = (id: number) => {
    confirm(
      `Are you sure you want to test endpoint ${id}?`,
      async () => {
        try {
          // Use functional update to avoid stale closure
          setTestingEndpointIds((prev) => {
            if (prev.includes(id)) {
              addToast({
                title: "Already Testing",
                description: `Endpoint ${id} is already being tested. Please wait for results.`,
                color: "warning",
              });

              return prev;
            }
            endpointApi.triggerEndpointTest(id);
            addToast({
              title: "Test Triggered",
              description: `Endpoint ${id} test started. Please wait for results.`,
              color: "primary",
            });

            return [...prev, id];
          });
        } catch (err) {
          addToast({
            title: "Failed to trigger test",
            description: (err as Error).message || "Please try again",
            color: "danger",
          });
        }
      },
      "Confirm Test Endpoint",
    );
  };

  // Create selection toolbar content
  const selectionToolbarContent = useMemo(() => {
    if (!selectedEndpointIds || selectedEndpointIds.size === 0) return null;

    return (
      <div className="flex gap-2">
        <Button
          color="primary"
          size="sm"
          startContent={<TestIcon />}
          variant="flat"
          onPress={handleBatchTestEndpoints}
        >
          Batch Test
        </Button>
        <Button
          color="danger"
          size="sm"
          startContent={<DeleteIcon />}
          variant="flat"
          onPress={handleBatchDeleteEndpoints}
        >
          Batch Delete
        </Button>
      </div>
    );
  }, [selectedEndpointIds]);

  // Add polling logic
  useEffect(() => {
    // Start polling if there are endpoints being tested
    if (testingEndpointIds.length > 0 && !pollingIntervalRef.current) {
      // Create a copy of current ID list for closure
      const currentTestingIds = [...testingEndpointIds];

      pollingIntervalRef.current = setInterval(async () => {
        // Get current latest test ID list
        let stillTestingIds = [...currentTestingIds];

        // Check status of each endpoint being tested
        for (const endpointId of currentTestingIds) {
          try {
            const task = await endpointApi.getEndpointTask(endpointId);

            const endpoint = endpoints?.items.find(
              (endpoint) => endpoint.id === endpointId,
            );

            if (endpoint) {
              endpoint.task_status = task.status;
            }

            if (
              task.status !== TaskStatusEnum.RUNNING &&
              task.status !== TaskStatusEnum.PENDING
            ) {
              if (task.status === TaskStatusEnum.FAILED) {
                addToast({
                  title: "Test Failed",
                  description: `Endpoint ${endpointId} test failed. Please try again.`,
                  color: "danger",
                });
              } else if (task.status === TaskStatusEnum.DONE) {
                addToast({
                  title: "Test Successful",
                  description: `Endpoint ${endpointId} test completed successfully.`,
                  color: "success",
                });
              }
              // Remove from test list if no running tasks
              stillTestingIds = stillTestingIds.filter(
                (id) => Number(id) !== Number(endpointId),
              );
              // Refresh endpoint list for latest status
              refetch();
            }
          } catch {
            // Remove from list on error to avoid infinite retries
            stillTestingIds = stillTestingIds.filter(
              (id) => Number(id) !== Number(endpointId),
            );
          }
        }

        // Update testing endpoint list - use functional update for latest state
        setTestingEndpointIds((prev) => {
          if (JSON.stringify(stillTestingIds) !== JSON.stringify(prev)) {
            return stillTestingIds;
          }

          return prev;
        });

        // Clear polling if no endpoints are being tested
        if (stillTestingIds.length === 0 && pollingIntervalRef.current) {
          clearInterval(pollingIntervalRef.current);
          pollingIntervalRef.current = null;
        }
      }, 5000); // Poll every 5 seconds
    }

    return () => {
      // Clear polling on component unmount
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
    };
  }, [testingEndpointIds, refetch]);

  // Open endpoint detail drawer
  const openEndpointDetail = (endpointId: number) => {
    setSelectedEndpointId(endpointId);
    setIsDetailDrawerOpen(true);
  };

  // Close endpoint detail drawer
  const closeEndpointDetail = () => {
    setIsDetailDrawerOpen(false);
  };

  return (
    <DashboardLayout current_root_href="/endpoints">
      {endpointsError ? (
        <ErrorDisplay
          error={
            new Error((endpointsError as Error)?.message || "Failed to load endpoint list")
          }
        />
      ) : (
        <EndpointTable
          endpoints={endpoints?.items}
          error={endpointsError}
          isAdmin={isAdmin}
          isLoading={isLoading}
          order={order}
          orderBy={orderBy}
          page={page}
          pageSize={pageSize}
          searchTerm={searchTerm}
          selectedKeys={selectedEndpointIds}
          selectionMode="multiple"
          selectionToolbarContent={selectionToolbarContent}
          setOrder={setOrder}
          setOrderBy={setOrderBy}
          setPageSize={setPageSize}
          setSearchTerm={setSearchTerm}
          setVisibleColumns={setVisibleColumns}
          testingEndpointIds={testingEndpointIds}
          totalItems={endpoints?.total}
          totalPages={endpoints?.pages}
          visibleColumns={visibleColumns}
          onCreateEndpoint={() => setIsCreateModalOpen(true)}
          onDeleteEndpoint={handleDeleteEndpoint}
          onEditEndpoint={(endpoint) => {
            setEditingEndpoint(endpoint);
            setIsEditModalOpen(true);
          }}
          onOpenEndpointDetail={openEndpointDetail}
          onPageChange={handlePageChange}
          onSearch={handleSearch}
          onSelectionChange={handleSelectionChange}
          onTestEndpoint={handleTestEndpoint}
        />
      )}

      {/* Create endpoint dialog */}
      <CreateEndpointModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={refetch}
      />

      {/* Edit endpoint dialog */}
      {editingEndpoint && (
        <EndpointEditModal
          endpointId={editingEndpoint.id}
          endpointName={editingEndpoint.name}
          endpointUrl={editingEndpoint.url}
          isOpen={isEditModalOpen}
          onClose={() => setIsEditModalOpen(false)}
          onDelete={handleDeleteEndpoint}
          onSuccess={refetch}
        />
      )}

      {/* Endpoint detail drawer */}
      {selectedEndpointId && (
        <EndpointDetailDrawer
          id={selectedEndpointId}
          isAdmin={isAdmin}
          isOpen={isDetailDrawerOpen}
          onClose={closeEndpointDetail}
          onDelete={handleDeleteEndpoint}
          onEdit={(endpoint) => {
            setEditingEndpoint(endpoint);
            setIsEditModalOpen(true);
          }}
        />
      )}
    </DashboardLayout>
  );
};

export default EndpointListPage;
