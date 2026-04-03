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
import { Tooltip } from "@heroui/tooltip";
import { Snippet } from "@heroui/snippet";
import { Link } from "@heroui/link";
import { Image } from "@heroui/image";
import {
  Drawer,
  DrawerContent,
  DrawerHeader,
  DrawerBody,
} from "@heroui/drawer";
import { PrismLight as SyntaxHighlighter } from "react-syntax-highlighter";
import bash from "react-syntax-highlighter/dist/esm/languages/prism/bash";
import {
  oneDark,
  oneLight,
} from "react-syntax-highlighter/dist/esm/styles/prism";
import { useTheme } from "@heroui/use-theme";
import { SortDescriptor } from "@react-types/shared";

import { useAuth } from "@/contexts/AuthContext";
import {
  useCustomQuery,
  useCustomMutation,
  usePaginationUrlState,
  PaginationValidationConfig,
} from "@/hooks";
import { apiKeyApi } from "@/api";
import {
  ApiKeyCreate,
  ApiKeyInfo,
  ApiKeyResponse,
  PageResponse,
  SortOrder,
} from "@/types";
import DashboardLayout from "@/layouts/Main";
import {
  DeleteIcon,
  PlusIcon,
  StatisticsIcon,
  QuestionMarkIcon,
  LeftArrowIcon,
  LogoIcon,
  LinkIcon,
  BoolAtlasIcon,
  BookIcon,
} from "@/components/icons";
import { DataTable } from "@/components/DataTable";
import { useDialog } from "@/contexts/DialogContext";
import { StatsDrawer } from "@/components/apikeys";

const ApiKeysPage = () => {
  SyntaxHighlighter.registerLanguage("bash", bash);
  const { theme } = useTheme();
  const { isAdmin } = useAuth();
  const { confirm } = useDialog();

  // Validation config
  const [validationConfig, setValidationConfig] =
    useState<PaginationValidationConfig>({
      page: { min: 1 },
      pageSize: { min: 5, max: 100 },
      totalPages: 1,
      orderBy: {
        allowedFields: ["id", "name", "created_at", "last_used_at"],
        defaultField: "id",
      },
    });

  // Use URL params for state management instead of multiple useState
  const {
    page,
    pageSize: size,
    search: searchTerm,
    orderBy,
    order,
    setPage,
    setPageSize: setSize,
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
  const [newApiKeyName, setNewApiKeyName] = useState("");
  const [createdApiKey, setCreatedApiKey] = useState<ApiKeyResponse | null>(
    null,
  );
  const [isCreating, setIsCreating] = useState(false);
  const [isHelpDrawerOpen, setIsHelpDrawerOpen] = useState(false);

  // Stats drawer state
  const [isStatsDrawerOpen, setIsStatsDrawerOpen] = useState(false);
  const [selectedApiKeyId, setSelectedApiKeyId] = useState<number | null>(null);
  const [selectedApiKeyName, setSelectedApiKeyName] = useState<string>("");

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

  // Fetch API key list
  const {
    data: apiKeys,
    isLoading,
    error,
    refetch,
  } = useCustomQuery<PageResponse<ApiKeyInfo>>(
    ["apikeys", page, size, searchTerm, orderBy, order],
    () =>
      apiKeyApi.getApiKeys({
        page,
        size,
        search: searchTerm,
        order_by: orderBy,
        order,
      }),
    { staleTime: 30000 },
  );

  // Update validation config when total pages change
  useEffect(() => {
    if (apiKeys?.pages) {
      setValidationConfig((prev) => ({
        ...prev,
        totalPages: apiKeys.pages,
      }));
    }
  }, [apiKeys?.pages]);

  // Create API Key
  const createApiKeyMutation = useCustomMutation<ApiKeyResponse, ApiKeyCreate>(
    (data) => apiKeyApi.createApiKey(data),
    {
      onSuccess: (data) => {
        setCreatedApiKey(data);
        refetch();
        setIsCreating(false);
      },
      onError: () => {
        setIsCreating(false);
      },
    },
  );

  // Handle search
  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // Reset page on search
    setPage(1);
    refetch();
  };

  // Handle page change
  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  // Handle create API key
  const handleCreateApiKey = () => {
    setIsCreating(true);
    createApiKeyMutation.mutate({ name: newApiKeyName });
  };

  // Handle close success dialog
  const handleCloseSuccessModal = () => {
    setCreatedApiKey(null);
    setNewApiKeyName("");
    setIsCreateModalOpen(false);
  };

  // Handle delete API key
  const handleDeleteApiKey = (id: number) => {
    confirm("Are you sure you want to delete this API key? This action is irreversible.", async () => {
      try {
        await apiKeyApi.deleteApiKey(id);
        refetch();
      } catch (err) {
        addToast({
          title: "Failed to delete API key",
          description: (err as Error).message || "Please try again",
          color: "danger",
        });
      }
    });
  };

  // Open API key stats drawer
  const openApiKeyStats = (apiKey: ApiKeyInfo) => {
    setSelectedApiKeyId(apiKey.id);
    setSelectedApiKeyName(apiKey.name);
    setIsStatsDrawerOpen(true);
  };

  // Close API key stats drawer
  const closeApiKeyStats = () => {
    setIsStatsDrawerOpen(false);
  };

  // Define table columns
  const columns = [
    { key: "id", label: "ID", allowsSorting: true },
    { key: "name", label: "Name", allowsSorting: true },
    { key: "created_at", label: "Created At", allowsSorting: true },
    { key: "last_used_at", label: "Last Used", allowsSorting: true },
    ...(isAdmin
      ? [{ key: "user_id", label: "User", allowsSorting: true }]
      : []),
    { key: "actions", label: "Actions" },
  ];

  // Render cell content
  const renderCell = (apiKey: ApiKeyInfo, columnKey: string) => {
    switch (columnKey) {
      case "id":
        return apiKey.id;
      case "name":
        return (
          <span className="whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">
            {apiKey.name}
          </span>
        );
      case "created_at":
        return new Date(apiKey.created_at).toLocaleString();
      case "last_used_at":
        return apiKey.last_used_at
          ? new Date(apiKey.last_used_at).toLocaleString()
          : "Never used";
      case "user_id":
        return apiKey.user_name || "-";
      case "actions":
        return (
          <div className="relative flex items-center gap-2">
            <Tooltip content="View Key Usage">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => openApiKeyStats(apiKey)}
              >
                <StatisticsIcon />
              </Button>
            </Tooltip>
            <Tooltip color="danger" content="Delete Key">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => {
                  if (apiKey.id) {
                    handleDeleteApiKey(apiKey.id);
                  }
                }}
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
    <DashboardLayout current_root_href="/apikeys">
      <DataTable<ApiKeyInfo>
        addButtonProps={{
          tooltip: "Create API Key",
          onClick: () => setIsCreateModalOpen(true),
          isIconOnly: true,
        }}
        autoSearchDelay={1000}
        columns={columns}
        data={apiKeys?.items || []}
        emptyContent={
          <>
            <p className="text-xl text-gray-600 dark:text-gray-400">
              No API key data
            </p>
            <Tooltip
              color="primary"
              content="Create your first API key"
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
        error={error}
        isLoading={isLoading}
        page={page}
        pageSize={size}
        pages={apiKeys?.pages}
        renderCell={renderCell}
        searchPlaceholder="Search API keys..."
        searchTerm={searchTerm}
        selectedSize={size}
        setSearchTerm={setSearchTerm}
        setSize={setSize}
        sortDescriptor={sortDescriptor}
        title="API Key List"
        topActionContent={
          <Tooltip color="default" content="Usage Guide">
            <Button
              isIconOnly
              color="default"
              variant="bordered"
              onPress={() => setIsHelpDrawerOpen(true)}
            >
              <QuestionMarkIcon />
            </Button>
          </Tooltip>
        }
        total={apiKeys?.total}
        onPageChange={handlePageChange}
        onSearch={handleSearch}
        onSortChange={handleSort}
      />

      {/* Create API KeyDialog */}
      <Modal
        isOpen={isCreateModalOpen}
        placement="center"
        onClose={() => setIsCreateModalOpen(false)}
      >
        <ModalContent>
          {(onClose) => (
            <>
              <ModalHeader>
                {createdApiKey ? "Key Created Successfully" : "Create API Key"}
              </ModalHeader>
              <ModalBody className="w-full">
                {createdApiKey ? (
                  <div>
                    <p className="mb-2">
                      Please save your API key now. This is the only time you will be able to see it:
                    </p>
                    <Snippet color="primary" symbol="">
                      {createdApiKey.key}
                    </Snippet>
                  </div>
                ) : (
                  <Input
                    fullWidth
                    isRequired
                    errorMessage={({ validationErrors }) => {
                      return validationErrors;
                    }}
                    label="API Key Name"
                    maxLength={128}
                    minLength={3}
                    placeholder="Enter API key name"
                    value={newApiKeyName}
                    onChange={(e) => setNewApiKeyName(e.target.value)}
                  />
                )}
              </ModalBody>
              <ModalFooter className="w-full">
                {createdApiKey ? (
                  <Button color="primary" onPress={handleCloseSuccessModal}>
                    OK
                  </Button>
                ) : (
                  <>
                    <Button
                      disabled={isCreating}
                      variant="light"
                      onPress={onClose}
                    >
                      Cancel
                    </Button>
                    <Button
                      color="primary"
                      disabled={isCreating}
                      isLoading={isCreating}
                      onPress={handleCreateApiKey}
                    >
                      Create
                    </Button>
                  </>
                )}
              </ModalFooter>
            </>
          )}
        </ModalContent>
      </Modal>

      {/* APIKey statistics drawer */}
      {selectedApiKeyId && (
        <StatsDrawer
          apiKeyName={selectedApiKeyName}
          id={selectedApiKeyId}
          isOpen={isStatsDrawerOpen}
          onClose={closeApiKeyStats}
        />
      )}

      {/* Usage GuideDrawer */}
      <Drawer
        backdrop="blur"
        classNames={{
          base: "data-[placement=right]:sm:m-2 data-[placement=left]:sm:m-2  rounded-medium",
        }}
        isOpen={isHelpDrawerOpen}
        placement="right"
        size="2xl"
        onClose={() => setIsHelpDrawerOpen(false)}
      >
        <DrawerContent>
          <DrawerHeader className="absolute top-0 inset-x-0 z-50 flex flex-row gap-2 px-2 py-2 border-b border-default-200/50 justify-between bg-content1/50 backdrop-saturate-150 backdrop-blur-lg">
            <Tooltip content="Close">
              <Button
                isIconOnly
                className="text-default-400 active:opacity-50 text-lg"
                variant="light"
                onPress={() => setIsHelpDrawerOpen(false)}
              >
                <LeftArrowIcon />
              </Button>
            </Tooltip>
          </DrawerHeader>
          <DrawerBody className="pt-16 w-full">
            <div className="flex w-full justify-center items-center pt-4">
              <LogoIcon className="w-32 h-32" />
            </div>
            <div className="flex flex-col gap-2 py-4">
              <h1 className="text-2xl font-bold leading-7">
                Aggregated API Usage Guide
              </h1>
              <div className="flex flex-col mt-4 gap-3 items-start">
                <h2 className="text-medium font-medium">What is the Aggregated API?</h2>
                <div className="text-medium text-default-500 flex flex-col gap-2">
                  <p>
                    The Aggregated API is the core feature of <i>Ollama Hack</i>.
                    It provides smart access to discovered high-availability models through an Ollama-compatible OpenAI API.
                  </p>
                </div>
              </div>
              <div className="flex flex-col mt-4 gap-3 items-start w-full">
                <h2 className="text-medium font-medium">How to use the Aggregated API?</h2>
                <div className="text-medium text-default-500 flex flex-col gap-2">
                  <p>
                    First, you need to generate an API key. Note that you will only see
                    the API key once upon creation — it cannot be viewed again, so keep it safe.
                  </p>
                  <div className="flex flex-col gap-2 w-full justify-center items-center">
                    <Image
                      alt="API Key Creation"
                      className="h-full"
                      src="/images/apikeys/apikey-create.png"
                    />
                    <Image
                      alt="API Key Creation"
                      className="h-full"
                      src="/images/apikeys/apikey-created.png"
                    />
                  </div>
                  <p>
                    You can now use this API key to access the Aggregated API.
                    The API key is passed via request headers.
                  </p>
                  <p>For example, you can access the Aggregated API like this:</p>
                  <SyntaxHighlighter
                    language="bash"
                    style={theme === "light" ? oneLight : oneDark}
                    wrapLines={true}
                    wrapLongLines={true}
                  >
                    {`curl -X POST http://localhost:3000/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <your-api-key>" \\
  -d '{
    "model": "qwq:latest",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant"},
      {"role": "user", "content": "Hello, please introduce yourself"}
    ]
  }'`}
                  </SyntaxHighlighter>
                  <p>
                    Aggregated API
                    Intelligently detects the models you use and forwards your requests to optimal endpoints based on generation speed. All
                    APIs are compatible with Ollama APIs
                    , including streaming generation. Simply follow the Ollama API
                    documentation to make calls. Official Ollama API documentation is provided below for reference.
                  </p>
                  <div className="flex justify-around w-full mt-4">
                    <div className="flex gap-3 items-center">
                      <div className="flex items-center justify-center border-1 border-default-200/50 rounded-small w-11 h-11">
                        <BookIcon className="w-6 h-6" />
                      </div>
                      <div className="flex flex-col gap-0.5">
                        <Link
                          isExternal
                          showAnchorIcon
                          anchorIcon={<LinkIcon />}
                          className="group gap-x-1 text-medium text-foreground font-medium"
                          href="https://github.com/ollama/ollama/blob/main/docs/api.md"
                          rel="noreferrer noopener"
                        >
                          Ollama API Docs
                        </Link>
                        <p className="text-small text-default-500">
                          Ollama Native API Docs
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-3 items-center">
                      <div className="flex items-center justify-center border-1 border-default-200/50 rounded-small w-11 h-11">
                        <BoolAtlasIcon className="w-6 h-6" />
                      </div>
                      <div className="flex flex-col gap-0.5">
                        <Link
                          isExternal
                          showAnchorIcon
                          anchorIcon={<LinkIcon />}
                          className="group gap-x-1 text-medium text-foreground font-medium"
                          href="https://github.com/ollama/ollama/blob/main/docs/openai.md"
                          rel="noreferrer noopener"
                        >
                          OpenAI Compatible API Docs
                        </Link>
                        <p className="text-small text-default-500">
                          OpenAI Compatible API Docs
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </DashboardLayout>
  );
};

export default ApiKeysPage;
