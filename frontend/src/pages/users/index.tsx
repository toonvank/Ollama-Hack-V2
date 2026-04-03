import { useState, useEffect } from "react";
import { Button } from "@nextui-org/button";
import { Input } from "@nextui-org/input";
import { Form } from "@nextui-org/form";
import {
  Modal,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from "@nextui-org/modal";
import { SortDescriptor } from "@react-types/shared";
import { Checkbox } from "@nextui-org/checkbox";
import { Tooltip } from "@nextui-org/tooltip";
import { addToast } from "@/utils/toast";
import { Chip } from "@nextui-org/chip";
import { Select, SelectItem } from "@nextui-org/select";

import {
  useCustomQuery,
  useCustomMutation,
  usePaginationUrlState,
  PaginationValidationConfig,
} from "@/hooks";
import { useAuth } from "@/contexts/AuthContext";
import { authApi, planApi, EnhancedAxiosError } from "@/api";
import {
  UserAuth,
  UserInfo,
  UserUpdate,
  PageResponse,
  SortOrder,
  PlanResponse,
} from "@/types";
import DashboardLayout from "@/layouts/Main";
import { DeleteIcon, EditIcon, PlusIcon } from "@/components/icons";
import { DataTable } from "@/components/DataTable";
import { useDialog } from "@/contexts/DialogContext";

const UsersPage = () => {
  const { user: currentUser, isAdmin } = useAuth();
  const { confirm } = useDialog();

  // Validation config
  const [validationConfig, setValidationConfig] =
    useState<PaginationValidationConfig>({
      page: { min: 1 },
      pageSize: { min: 5, max: 100 },
      totalPages: 1,
      orderBy: {
        allowedFields: ["id", "username", "is_admin", "plan_id"],
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
      orderBy: undefined,
      order: undefined,
    },
    validationConfig,
  );

  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [newUser, setNewUser] = useState({
    username: "",
    password: "",
    is_admin: false,
  });
  const [isCreating, setIsCreating] = useState(false);

  // Edit modal state
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<UserUpdate>({
    username: "",
    is_admin: false,
    plan_id: undefined,
  });
  const [editingUserId, setEditingUserId] = useState<number | undefined>();
  const [isUpdating, setIsUpdating] = useState(false);

  // Sort state
  const [sortDescriptor, setSortDescriptor] = useState<SortDescriptor>({
    column: orderBy || "id",
    direction:
      order === SortOrder.ASC
        ? "ascending"
        : order === SortOrder.DESC
          ? "descending"
          : "ascending",
  });

  // Handle sort
  const handleSort = (descriptor: SortDescriptor) => {
    setSortDescriptor(descriptor);
    setOrderBy(descriptor.column?.toString());
    setOrder(
      descriptor.direction === "ascending" ? SortOrder.ASC : SortOrder.DESC,
    );
  };

  // Fetch user list
  const {
    data: users,
    isLoading,
    error: usersError,
    refetch,
  } = useCustomQuery<PageResponse<UserInfo>>(
    ["users", page, pageSize, searchTerm, orderBy, order],
    () =>
      authApi.getUsers({
        page,
        size: pageSize,
        search: searchTerm,
        order_by: orderBy,
        order,
      }),
    {
      staleTime: 30000,
      enabled: isAdmin,
    },
  );

  // Update validation config when total pages change
  useEffect(() => {
    if (users?.pages) {
      setValidationConfig((prev) => ({
        ...prev,
        totalPages: users.pages,
      }));
    }
  }, [users?.pages]);

  // Fetch all plans (for plan selection when editing users)
  const { data: plans, isLoading: isLoadingPlans } = useCustomQuery<
    PageResponse<PlanResponse>
  >(
    ["plans-for-users", 1, 50],
    () =>
      planApi.getPlans({
        page: 1,
        size: 50,
      }),
    {
      staleTime: 60000,
      enabled: isAdmin,
    },
  );

  // Create User
  const createUserMutation = useCustomMutation<UserInfo, UserAuth>(
    (data) => {
      return authApi.createUser(data);
    },
    {
      onSuccess: () => {
        refetch();
        setIsCreateModalOpen(false);
        setNewUser({ username: "", password: "", is_admin: false });
        setIsCreating(false);
      },
      onError: (err) => {
        addToast({
          title: "Failed to create user",
          description: (err as EnhancedAxiosError).detail || "Failed to create user",
          color: "danger",
        });
        setIsCreating(false);
      },
    },
  );

  // UpdateUser
  const updateUserMutation = useCustomMutation<
    UserInfo,
    { id: number; data: UserUpdate }
  >(({ id, data }) => authApi.updateUser(id, data), {
    onSuccess: () => {
      refetch();
      setIsEditModalOpen(false);
      setEditingUser({
        username: "",
        is_admin: false,
        plan_id: undefined,
      });
      setEditingUserId(undefined);
      setIsUpdating(false);
      addToast({
        title: "User Updated",
        description: "User information has been updated successfully",
        color: "success",
      });
    },
    onError: (err) => {
      addToast({
        title: "Failed to update user",
        description: (err as EnhancedAxiosError).detail || "Failed to update user",
        color: "danger",
      });
      setIsUpdating(false);
    },
  });

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

  // Handle create user
  const handleCreateUser = (e: React.FormEvent) => {
    e.preventDefault();

    if (!newUser.username || !newUser.password) {
      addToast({
        title: "Failed to create user",
        description: "Username and password cannot be empty",
        color: "danger",
      });

      return;
    }

    setIsCreating(true);
    createUserMutation.mutate({
      username: newUser.username,
      password: newUser.password,
    });
  };

  const handleDeleteUser = async (userId: number) => {
    confirm(
      "Are you sure you want to delete this user? This action is irreversible.",
      async () => {
        await authApi.deleteUser(userId);
        refetch();
        addToast({
          title: "User Deleted",
          description: "The user has been deleted successfully",
          color: "success",
        });
      },
    );
  };

  // Open edit modal
  const openEditModal = (user: UserInfo) => {
    setEditingUser({
      username: user.username,
      is_admin: user.is_admin,
      plan_id: user.plan_id,
    });
    setEditingUserId(user.id);
    setIsEditModalOpen(true);
  };

  // Handle form input change
  const handleEditInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;

    setEditingUser((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // Handle checkbox change
  const handleCheckboxChange = (isSelected: boolean) => {
    setEditingUser((prev) => ({
      ...prev,
      is_admin: isSelected,
    }));
  };

  // Handle plan selection change
  const handlePlanChange = (planId: string) => {
    const numericPlanId = planId === "" ? undefined : Number(planId);

    setEditingUser((prev) => ({
      ...prev,
      plan_id: numericPlanId,
    }));
  };

  // Handle edit form submit
  const handleUpdateUser = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingUserId) return;

    // Validate form
    if (!editingUser.username) {
      addToast({
        title: "Failed to update user",
        description: "Username cannot be empty",
        color: "danger",
      });

      return;
    }

    setIsUpdating(true);
    const updateData: UserUpdate = {
      username: editingUser.username,
      is_admin: editingUser.is_admin,
      plan_id: editingUser.plan_id,
    };

    // Only include password if field has a value
    if (editingUser.password) {
      updateData.password = editingUser.password;
    }

    updateUserMutation.mutate({
      id: editingUserId,
      data: updateData,
    });
  };

  // Close edit modal
  const handleCloseEditModal = () => {
    if (!isUpdating) {
      setIsEditModalOpen(false);
      setEditingUser({
        username: "",
        is_admin: false,
        plan_id: undefined,
      });
      setEditingUserId(undefined);
    }
  };

  // Define table columns
  const columns = [
    { key: "id", label: "ID", allowsSorting: true },
    { key: "username", label: "Username", allowsSorting: true },
    { key: "is_admin", label: "Role", allowsSorting: true },
    { key: "plan_id", label: "Plan", allowsSorting: true },
    { key: "actions", label: "Actions" },
  ];

  // Render cell content
  const renderCell = (user: UserInfo, columnKey: string) => {
    switch (columnKey) {
      case "id":
        return user.id;
      case "username":
        return <span>{user.username}</span>;
      case "is_admin":
        return user.is_admin ? (
          <Chip color="primary" variant="flat">
            Admin
          </Chip>
        ) : (
          <Chip color="default" variant="flat">
            User
          </Chip>
        );
      case "plan_id":
        return user.plan_name || "-";
      case "actions":
        return (
          <div className="relative flex items-center gap-2">
            <Tooltip content="Edit User">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => openEditModal(user)}
              >
                <EditIcon />
              </Button>
            </Tooltip>
            {user?.id !== currentUser?.id && isAdmin && (
              <Tooltip content="DeleteUser">
                <Button
                  isIconOnly
                  className="text-default-400 active:opacity-50 text-lg"
                  variant="light"
                  onPress={() => handleDeleteUser(user.id)}
                >
                  <DeleteIcon />
                </Button>
              </Tooltip>
            )}
          </div>
        );
      default:
        return null;
    }
  };

  if (!isAdmin) {
    return (
      <DashboardLayout current_root_href="/users">
        <div className="p-8 text-center">
          <h2 className="text-xl font-semibold mb-4">Insufficient Permissions</h2>
          <p className="text-gray-600 dark:text-gray-400">
            You do not have permission to access this page
          </p>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout current_root_href="/users">
      <DataTable<UserInfo>
        addButtonProps={{
          tooltip: "Create User",
          onClick: () => setIsCreateModalOpen(true),
          isIconOnly: true,
        }}
        autoSearchDelay={1000}
        columns={columns}
        data={users?.items || []}
        emptyContent={
          <>
            <p className="text-xl text-gray-600 dark:text-gray-400">
              No user data
            </p>
            <Tooltip
              color="primary"
              content="Create your first user"
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
        error={usersError}
        isLoading={isLoading}
        page={page}
        pageSize={pageSize}
        pages={Math.ceil((users?.total || 0) / pageSize)}
        renderCell={renderCell}
        searchPlaceholder="Search users..."
        searchTerm={searchTerm}
        selectedSize={pageSize}
        setSearchTerm={setSearchTerm}
        setSize={setPageSize}
        sortDescriptor={sortDescriptor}
        title="User List"
        total={users?.total}
        onPageChange={handlePageChange}
        onSearch={handleSearch}
        onSortChange={handleSort}
      />

      {/* Create UserDialog */}
      <Modal
        isOpen={isCreateModalOpen}
        placement="center"
        onClose={() => !isCreating && setIsCreateModalOpen(false)}
      >
        <ModalContent>
          {(onClose) => (
            <Form className="w-full" onSubmit={handleCreateUser}>
              <ModalHeader>Create New User</ModalHeader>
              <ModalBody className="w-full">
                <Input
                  fullWidth
                  isRequired
                  label="Username"
                  maxLength={128}
                  minLength={3}
                  placeholder="Enter username"
                  value={newUser.username}
                  onChange={(e) =>
                    setNewUser({
                      ...newUser,
                      username: e.target.value,
                    })
                  }
                />
                <Input
                  fullWidth
                  isRequired
                  errorMessage={({ validationErrors }) => validationErrors}
                  label="Password"
                  maxLength={128}
                  minLength={8}
                  placeholder="Enter password"
                  type="password"
                  value={newUser.password}
                  onChange={(e) =>
                    setNewUser({
                      ...newUser,
                      password: e.target.value,
                    })
                  }
                />
                <Checkbox
                  isSelected={newUser.is_admin}
                  onValueChange={(e) =>
                    setNewUser({
                      ...newUser,
                      is_admin: e,
                    })
                  }
                >
                  Set as Admin
                </Checkbox>
              </ModalBody>
              <ModalFooter className="w-full">
                <Button disabled={isCreating} variant="light" onPress={onClose}>
                  Cancel
                </Button>
                <Button color="primary" isLoading={isCreating} type="submit">
                  Create
                </Button>
              </ModalFooter>
            </Form>
          )}
        </ModalContent>
      </Modal>

      {/* Edit UserDialog */}
      <Modal
        isOpen={isEditModalOpen}
        placement="center"
        onClose={handleCloseEditModal}
      >
        <ModalContent>
          <Form className="w-full" onSubmit={handleUpdateUser}>
            <ModalHeader>Edit User</ModalHeader>
            <ModalBody className="w-full">
              <div className="space-y-4">
                <Input
                  fullWidth
                  isRequired
                  label="Username"
                  maxLength={128}
                  minLength={3}
                  name="username"
                  placeholder="Enter username"
                  value={editingUser.username}
                  onChange={handleEditInputChange}
                />
                <Input
                  fullWidth
                  description="Leave empty to keep current password"
                  label="Password"
                  maxLength={128}
                  minLength={8}
                  name="password"
                  placeholder="Enter new password"
                  type="password"
                  value={editingUser.password || ""}
                  onChange={handleEditInputChange}
                />
                <Select
                  label="Associated Plan"
                  placeholder="Select a plan"
                  selectedKeys={
                    editingUser.plan_id ? [editingUser.plan_id.toString()] : []
                  }
                  onChange={(e) => handlePlanChange(e.target.value)}
                >
                  {isLoadingPlans ? (
                    <SelectItem key="loading" value="loading">
                      Loading...
                    </SelectItem>
                  ) : (
                    plans?.items?.map((plan) => (
                      <SelectItem
                        key={plan.id.toString()}
                        value={plan.id.toString()}
                      >
                        {plan.name}
                      </SelectItem>
                    ))
                  )}
                </Select>
                <Checkbox
                  isSelected={editingUser.is_admin}
                  onValueChange={handleCheckboxChange}
                >
                  Set as Admin
                </Checkbox>
              </div>
            </ModalBody>
            <ModalFooter className="w-full">
              <Button
                disabled={isUpdating}
                variant="light"
                onPress={handleCloseEditModal}
              >
                Cancel
              </Button>
              <Button
                color="primary"
                disabled={isUpdating}
                isLoading={isUpdating}
                type="submit"
              >
                {isUpdating ? (
                  <>
                    <span className="ml-2">Saving...</span>
                  </>
                ) : (
                  "Save Changes"
                )}
              </Button>
            </ModalFooter>
          </Form>
        </ModalContent>
      </Modal>
    </DashboardLayout>
  );
};

export default UsersPage;
