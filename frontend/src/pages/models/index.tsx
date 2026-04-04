import { useState, useEffect } from "react";
import { Routes, Route } from "react-router-dom";

import {
  useCustomQuery,
  usePaginationUrlState,
  PaginationValidationConfig,
} from "@/hooks";
import { aiModelApi } from "@/api";
import { AIModelInfoWithEndpointCount, PageResponse } from "@/types";
import DashboardLayout from "@/layouts/Main";
import ModelTable from "@/components/models/Table";
import ModelDetailDrawer from "@/components/models/DetailDrawer";
import SmartModelsCard from "@/components/models/SmartModelsCard";
import { addToast } from "@heroui/toast";

// Model list page
export const ModelListPage = () => {
  // Validation config
  const [validationConfig, setValidationConfig] =
    useState<PaginationValidationConfig>({
      page: { min: 1 },
      pageSize: { min: 5, max: 100 },
      totalPages: 1,
      orderBy: {
        allowedFields: ["id", "name", "created_at", "token_per_second"],
        defaultField: "token_per_second",
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
      orderBy: "token_per_second",
      order: "desc",
    },
    validationConfig,
  );

  // Detail drawer state
  const [isDetailDrawerOpen, setIsDetailDrawerOpen] = useState(false);
  const [selectedModelId, setSelectedModelId] = useState<number | null>(null);

  // Fetch model list
  const {
    data: models,
    isLoading,
    error: modelsError,
    refetch,
  } = useCustomQuery<PageResponse<AIModelInfoWithEndpointCount>>(
    ["models", page, pageSize, searchTerm, orderBy, order],
    () =>
      aiModelApi.getAIModels({
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
    if (models?.pages) {
      setValidationConfig((prev) => ({
        ...prev,
        totalPages: models.pages,
      }));
    }
  }, [models?.pages]);

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

  // Open model detail drawer
  const openModelDetail = (modelId: number) => {
    setSelectedModelId(modelId);
    setIsDetailDrawerOpen(true);
  };

  // Close model detail drawer
  const closeModelDetail = () => {
    setIsDetailDrawerOpen(false);
  };

  // Handle toggle model enabled
  const handleToggleModel = async (modelId: number, enabled: boolean) => {
    try {
      await aiModelApi.toggleModel(modelId, enabled);
      addToast({
        title: "Success",
        description: `Model ${enabled ? "enabled" : "disabled"} successfully`,
        color: "success",
      });
      refetch();
    } catch (e) {
      console.error("Failed to toggle model:", e);
      addToast({
        title: "Error",
        description: "Failed to update model status",
        color: "danger",
      });
    }
  };

  return (
    <DashboardLayout current_root_href="/models">
      <SmartModelsCard />
      <ModelTable
        error={modelsError}
        isLoading={isLoading}
        models={models?.items}
        order={order}
        orderBy={orderBy}
        page={page}
        pageSize={pageSize}
        searchTerm={searchTerm}
        setOrder={setOrder}
        setOrderBy={setOrderBy}
        setPageSize={setPageSize}
        setSearchTerm={setSearchTerm}
        totalItems={models?.total}
        totalPages={models?.pages}
        onOpenModelDetail={openModelDetail}
        onPageChange={handlePageChange}
        onSearch={handleSearch}
        onToggleModel={handleToggleModel}
      />

      {/* ModelDetails drawer */}
      {selectedModelId && (
        <ModelDetailDrawer
          id={selectedModelId}
          isOpen={isDetailDrawerOpen}
          onClose={closeModelDetail}
        />
      )}
    </DashboardLayout>
  );
};

// Model router entry
const ModelsPage = () => {
  return (
    <Routes>
      <Route index element={<ModelListPage />} />
    </Routes>
  );
};

export default ModelsPage;
