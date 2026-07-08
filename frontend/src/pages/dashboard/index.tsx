import { Card } from "@heroui/card";

import { useAuth } from "@/contexts/AuthContext";
import { useCustomQuery, useLiveStats } from "@/hooks";
import { endpointApi, aiModelApi, planApi } from "@/api";
import {
  PageResponse,
  EndpointWithAIModelCount,
  AIModelInfoWithEndpointCount,
  PlanResponse,
  ApiError,
} from "@/types";
import DashboardLayout from "@/layouts/Main";
import LoadingSpinner from "@/components/LoadingSpinner";
import ErrorDisplay from "@/components/ErrorDisplay";
import { LiveStats } from "@/components/LiveStats";
import OverviewKpis from "@/components/dashboard/OverviewKpis";
import ModelRankingsCharts from "@/components/dashboard/ModelRankingsCharts";

const DashboardPage = () => {
  const { user } = useAuth();
  const { stats: liveStats, history, connected } = useLiveStats();

  const {
    data: userPlan,
    isLoading: isLoadingPlan,
    error: planError,
  } = useCustomQuery<PlanResponse>(
    ["plan", "current"],
    () => planApi.getCurrentUserPlan(),
    { enabled: !!user },
  );

  const {
    data: endpoints,
    isLoading: isLoadingEndpoints,
    error: endpointsError,
  } = useCustomQuery<PageResponse<EndpointWithAIModelCount>>(
    ["endpoints", "stats"],
    () =>
      endpointApi.getEndpoints({
        page: 1,
        size: 1,
      }),
    { enabled: true },
  );

  const {
    data: availableEndpoints,
    isLoading: isLoadingAvailableEndpoints,
    error: availableEndpointsError,
  } = useCustomQuery<PageResponse<EndpointWithAIModelCount>>(
    ["endpoints", "stats", "available"],
    () =>
      endpointApi.getEndpoints({
        page: 1,
        size: 1,
        status: "available",
      }),
    { enabled: true },
  );

  const {
    data: models,
    isLoading: isLoadingModels,
    error: modelsError,
  } = useCustomQuery<PageResponse<AIModelInfoWithEndpointCount>>(
    ["models", "stats"],
    () =>
      aiModelApi.getAIModels({
        page: 1,
        size: 1,
      }),
    { enabled: true },
  );

  const {
    data: availableModels,
    isLoading: isLoadingAvailableModels,
    error: availableModelsError,
  } = useCustomQuery<PageResponse<AIModelInfoWithEndpointCount>>(
    ["models", "stats", "available"],
    () =>
      aiModelApi.getAIModels({
        page: 1,
        size: 1,
        status: "available",
      }),
    { enabled: true },
  );

  const {
    data: topModels,
    isLoading: isLoadingTopModel,
    error: topModelError,
  } = useCustomQuery<PageResponse<AIModelInfoWithEndpointCount>>(
    ["models", "stats", "top"],
    () =>
      aiModelApi.getAIModels({
        page: 1,
        size: 1,
        order_by: "composite_score",
        order: "desc",
      }),
    { enabled: true },
  );

  const isLoading =
    isLoadingPlan ||
    isLoadingEndpoints ||
    isLoadingModels ||
    isLoadingAvailableEndpoints ||
    isLoadingAvailableModels ||
    isLoadingTopModel;
  const error =
    planError ||
    endpointsError ||
    modelsError ||
    availableEndpointsError ||
    availableModelsError ||
    topModelError;

  const getErrorForDisplay = () => {
    if (!error) return null;

    return new Error((error as ApiError)?.message || "An error occurred");
  };

  if (isLoading) {
    return (
      <DashboardLayout current_root_href="/">
        <div className="flex justify-center items-center h-64">
          <LoadingSpinner size="large" />
        </div>
      </DashboardLayout>
    );
  }

  const topModel = topModels?.items?.[0];

  return (
    <DashboardLayout current_root_href="/">
      {error && <ErrorDisplay error={getErrorForDisplay()} />}

      <div className="mb-6">
        <h1 className="text-2xl font-bold">Overview</h1>
        <p className="text-default-500 mt-1">
          Fleet health, model performance, and live proxy activity
        </p>
      </div>

      <OverviewKpis
        endpointsAvailable={availableEndpoints?.total || 0}
        endpointsTotal={endpoints?.total || 0}
        liveStats={liveStats}
        modelsAvailable={availableModels?.total || 0}
        modelsTotal={models?.total || 0}
        topModel={topModel}
      />

      <ModelRankingsCharts />

      <LiveStats connected={connected} history={history} stats={liveStats} />

      {userPlan && (
        <Card className="p-6 shadow-sm border border-default-200">
          <h3 className="font-semibold text-lg mb-4">Current Plan</h3>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div>
              <p className="text-xs uppercase tracking-wide text-default-500 font-semibold">
                Plan
              </p>
              <p className="font-medium mt-1">{userPlan.name}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-default-500 font-semibold">
                Requests / Minute
              </p>
              <p className="font-medium mt-1">{userPlan.rpm}</p>
            </div>
            <div>
              <p className="text-xs uppercase tracking-wide text-default-500 font-semibold">
                Requests / Day
              </p>
              <p className="font-medium mt-1">{userPlan.rpd}</p>
            </div>
          </div>
          {userPlan.description && (
            <p className="text-default-500 text-sm mt-4">{userPlan.description}</p>
          )}
        </Card>
      )}
    </DashboardLayout>
  );
};

export default DashboardPage;