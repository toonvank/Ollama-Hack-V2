import { useState, useEffect } from "react";
import { Button } from "@heroui/button";
import { Input } from "@heroui/input";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@heroui/modal";
import { SortDescriptor } from "@react-types/shared";
import { Form } from "@heroui/form";
import { Tooltip } from "@heroui/tooltip";
import { addToast } from "@heroui/toast";
import { Checkbox } from "@heroui/checkbox";

import {
  useCustomQuery,
  useCustomMutation,
  usePaginationUrlState,
  PaginationValidationConfig,
} from "@/hooks";
import { EnhancedAxiosError, planApi } from "@/api";
import {
  PlanCreate,
  PlanResponse,
  PlanUpdate,
  PageResponse,
  SortOrder,
} from "@/types";
import DashboardLayout from "@/layouts/Main";
import { useDialog } from "@/contexts/DialogContext";
import { DeleteIcon, EditIcon, PlusIcon } from "@/components/icons";
import { DataTable } from "@/components/DataTable";

const PlansPage = () => {
  const { confirm } = useDialog();

  // Validation config
  const [validationConfig, setValidationConfig] =
    useState<PaginationValidationConfig>({
      page: { min: 1 },
      pageSize: { min: 5, max: 100 },
      totalPages: 1,
      orderBy: {
        allowedFields: ["id", "name", "rpm", "rpd", "is_default"],
        defaultField: "id",
      },
    });

  // Pagination and search state
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
      orderBy: "id",
      order: SortOrder.ASC,
    },
    validationConfig,
  );

  // Modal state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingPlan, setEditingPlan] = useState<PlanResponse | null>(null);

  // Form state
  const [newPlan, setNewPlan] = useState<PlanCreate>({
    name: "",
    description: "",
    rpm: 60,
    rpd: 1000,
    is_default: false,
  });
  const [isLoading, setIsLoading] = useState(false);

  // Sort state
  const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>({
    column: orderBy,
    direction: order === SortOrder.ASC ? "ascending" : "descending",
  });

  // Handle sort
  const handleSort = (descriptor: SortDescriptor) => {
    setSortDescriptor(descriptor);
    setOrderBy(descriptor.column?.toString() || "id");
    setOrder(
      descriptor.direction === "ascending" ? SortOrder.ASC : SortOrder.DESC,
    );
  };

  // Fetch plan list
  const {
    data: plans,
    isLoading: isLoadingPlans,
    error: plansError,
    refetch,
  } = useCustomQuery<PageResponse<PlanResponse>>(
    ["plans", page, pageSize, searchTerm, orderBy, order],
    () =>
      planApi.getPlans({
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
    if (plans?.pages) {
      setValidationConfig((prev) => ({
        ...prev,
        totalPages: plans.pages,
      }));
    }
  }, [plans?.pages]);

  // Create Plan
  const createPlanMutation = useCustomMutation<PlanResponse, PlanCreate>(
    (data) => planApi.createPlan(data),
    {
      onSuccess: () => {
        refetch();
        setIsCreateModalOpen(false);
        setNewPlan({
          name: "",
          description: "",
          rpm: 60,
          rpd: 1000,
          is_default: false,
        });
        setIsLoading(false);
      },
      onError: (err) => {
        addToast({
          title: "Failed to create plan",
          description: (err as EnhancedAxiosError).detail || "Failed to create plan",
          color: "danger",
        });
        setIsLoading(false);
      },
    },
  );

  // Update plan
  const updatePlanMutation = useCustomMutation<
    PlanResponse,
    { id: number; data: PlanUpdate }
  >(({ id, data }) => planApi.updatePlan(id, data), {
    onSuccess: () => {
      refetch();
      setIsEditModalOpen(false);
      setEditingPlan(null);
      setIsLoading(false);
    },
    onError: (err) => {
      addToast({
        title: "Failed to update plan",
        description: (err as EnhancedAxiosError).detail || "Failed to update plan",
        color: "danger",
      });
      setIsLoading(false);
    },
  });

  // Delete plan
  const deletePlanMutation = useCustomMutation<void, number>(
    (id) => planApi.deletePlan(id),
    {
      onSuccess: () => {
        refetch();
      },
    },
  );

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

  // Handle create plan
  const handleCreatePlan = (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    createPlanMutation.mutate(newPlan);
  };

  // Handle update plan
  const handleUpdatePlan = (e: React.FormEvent) => {
    e.preventDefault();

    if (!editingPlan) return;

    setIsLoading(true);
    updatePlanMutation.mutate({
      id: editingPlan.id,
      data: {
        name: editingPlan.name,
        description: editingPlan.description,
        rpm: editingPlan.rpm,
        rpd: editingPlan.rpd,
        is_default: editingPlan.is_default,
      },
    });
  };

  // Handle delete plan
  const handleDeletePlan = (id: number) => {
    confirm(
      "Are you sure you want to delete this plan? This action is irreversible and may affect users on this plan.",
      () => {
        deletePlanMutation.mutate(id);
      },
    );
  };

  // Open edit modal
  const openEditModal = (plan: PlanResponse) => {
    setEditingPlan(plan);
    setIsEditModalOpen(true);
  };

  // Define table columns
  const columns = [
    { key: "id", label: "ID", allowsSorting: true },
    { key: "name", label: "Name", allowsSorting: true },
    { key: "rpm", label: "RPM", allowsSorting: true },
    { key: "rpd", label: "RPD", allowsSorting: true },
    { key: "is_default", label: "Default Plan", allowsSorting: true },
    { key: "actions", label: "Actions" },
  ];

  // Render cell content
  const renderCell = (plan: PlanResponse, columnKey: string) => {
    switch (columnKey) {
      case "id":
        return plan.id;
      case "name":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {plan.name}
          </span>
        );
      case "rpm":
        return plan.rpm;
      case "rpd":
        return plan.rpd;
      case "is_default":
        return plan.is_default ? (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100">
            Yes
          </span>
        ) : (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-100">
            No
          </span>
        );
      case "actions":
        return (
          <div className="relative flex items-center gap-2">
            <Tooltip content="Edit">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => openEditModal(plan)}
              >
                <EditIcon />
              </Button>
            </Tooltip>
            <Tooltip color="danger" content="Delete">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => handleDeletePlan(plan.id)}
              >
                <DeleteIcon />
              </Button>
            </Tooltip>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <DashboardLayout current_root_href="/plans">
      <DataTable<PlanResponse>
        addButtonProps={{
          tooltip: "Create Plan",
          onClick: () => setIsCreateModalOpen(true),
          isIconOnly: true,
        }}
        autoSearchDelay={1000}
        columns={columns}
        data={plans?.items || []}
        emptyContent={
          <>
            <p className="text-xl text-gray-600 dark:text-gray-400">
              No plan data
            </p>
            <Tooltip
              color="primary"
              content="Create your first plan"
              placement="bottom"
            >
              <Button
                isIconOnly
                className="mt-4"
                color="primary"
                onPress={() => setIsCreateModalOpen(true)}
              >
                <PlusIcon />
              </Button>
            </Tooltip>
          </>
        }
        error={plansError}
        isLoading={isLoadingPlans}
        page={page}
        pageSize={pageSize}
        pages={Math.ceil((plans?.total || 0) / pageSize)}
        renderCell={renderCell}
        searchPlaceholder="Search plans..."
        searchTerm={searchTerm}
        selectedSize={pageSize}
        setSearchTerm={setSearchTerm}
        setSize={setPageSize}
        sortDescriptor={sortDescriptor}
        title="Plan List"
        total={plans?.total}
        onPageChange={handlePageChange}
        onSearch={handleSearch}
        onSortChange={handleSort}
      />

      {/* Create PlanDialog */}
      <Modal
        isOpen={isCreateModalOpen}
        placement="center"
        onClose={() => !isLoading && setIsCreateModalOpen(false)}
      >
        <ModalContent>
          {(onClose) => (
            <Form className="w-full" onSubmit={handleCreatePlan}>
              <ModalHeader>Create New Plan</ModalHeader>
              <ModalBody className="w-full">
                <Input
                  fullWidth
                  isRequired
                  label="Plan Name"
                  placeholder="Enter plan name"
                  value={newPlan.name}
                  onChange={(e) =>
                    setNewPlan({
                      ...newPlan,
                      name: e.target.value,
                    })
                  }
                />
                <Input
                  fullWidth
                  isRequired
                  label="Description"
                  placeholder="Enter plan description"
                  value={newPlan.description}
                  onChange={(e) =>
                    setNewPlan({
                      ...newPlan,
                      description: e.target.value,
                    })
                  }
                />
                <Input
                  fullWidth
                  isRequired
                  errorMessage={({ validationErrors }) => {
                    return validationErrors;
                  }}
                  label="Requests Per Minute (RPM)"
                  max={1000000}
                  min={0}
                  placeholder="Enter requests per minute limit"
                  type="number"
                  value={newPlan.rpm.toString()}
                  onChange={(e) =>
                    setNewPlan({
                      ...newPlan,
                      rpm: parseInt(e.target.value),
                    })
                  }
                />
                <Input
                  fullWidth
                  isRequired
                  errorMessage={({ validationErrors }) => {
                    return validationErrors;
                  }}
                  label="Requests Per Day (RPD)"
                  max={1000000}
                  min={0}
                  placeholder="Enter requests per day limit"
                  type="number"
                  value={newPlan.rpd.toString()}
                  onChange={(e) =>
                    setNewPlan({
                      ...newPlan,
                      rpd: parseInt(e.target.value),
                    })
                  }
                />
                <Checkbox
                  isSelected={newPlan.is_default}
                  onValueChange={(isSelected) =>
                    setNewPlan({
                      ...newPlan,
                      is_default: isSelected,
                    })
                  }
                >
                  Set as Default Plan
                </Checkbox>
              </ModalBody>
              <ModalFooter className="w-full">
                <Button disabled={isLoading} variant="light" onPress={onClose}>
                  Cancel
                </Button>
                <Button color="primary" isLoading={isLoading} type="submit">
                  Create
                </Button>
              </ModalFooter>
            </Form>
          )}
        </ModalContent>
      </Modal>

      {/* Edit PlanDialog */}
      <Modal
        isOpen={isEditModalOpen}
        placement="center"
        onClose={() => !isLoading && setIsEditModalOpen(false)}
      >
        <ModalContent>
          {(onClose) => (
            <Form className="w-full" onSubmit={handleUpdatePlan}>
              <ModalHeader>Edit Plan</ModalHeader>
              <ModalBody className="w-full">
                {editingPlan && (
                  <div className="space-y-4">
                    <Input
                      fullWidth
                      isRequired
                      label="Plan Name"
                      placeholder="Enter plan name"
                      value={editingPlan.name}
                      onChange={(e) =>
                        setEditingPlan({
                          ...editingPlan,
                          name: e.target.value,
                        })
                      }
                    />
                    <Input
                      fullWidth
                      isRequired
                      label="Description"
                      placeholder="Enter plan description"
                      value={editingPlan.description || ""}
                      onChange={(e) =>
                        setEditingPlan({
                          ...editingPlan,
                          description: e.target.value,
                        })
                      }
                    />
                    <Input
                      fullWidth
                      isRequired
                      errorMessage={({ validationErrors }) => {
                        return validationErrors;
                      }}
                      label="Requests Per Minute (RPM)"
                      max={1000000}
                      min={0}
                      placeholder="Enter requests per minute limit"
                      type="number"
                      value={editingPlan.rpm.toString()}
                      onChange={(e) =>
                        setEditingPlan({
                          ...editingPlan,
                          rpm: parseInt(e.target.value),
                        })
                      }
                    />
                    <Input
                      fullWidth
                      isRequired
                      errorMessage={({ validationErrors }) => {
                        return validationErrors;
                      }}
                      label="Requests Per Day (RPD)"
                      max={1000000}
                      min={0}
                      placeholder="Enter requests per day limit"
                      type="number"
                      value={editingPlan.rpd.toString()}
                      onChange={(e) =>
                        setEditingPlan({
                          ...editingPlan,
                          rpd: parseInt(e.target.value),
                        })
                      }
                    />
                    <Checkbox
                      isSelected={editingPlan.is_default}
                      onValueChange={(isSelected) =>
                        setEditingPlan({
                          ...editingPlan,
                          is_default: isSelected,
                        })
                      }
                    >
                      Set as Default Plan
                    </Checkbox>
                  </div>
                )}
              </ModalBody>
              <ModalFooter className="w-full">
                <Button disabled={isLoading} variant="light" onPress={onClose}>
                  Cancel
                </Button>
                <Button color="primary" isLoading={isLoading} type="submit">
                  Save
                </Button>
              </ModalFooter>
            </Form>
          )}
        </ModalContent>
      </Modal>
    </DashboardLayout>
  );
};

export default PlansPage;
