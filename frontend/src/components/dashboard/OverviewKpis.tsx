import { Card } from "@heroui/card";

import { LiveStatsData } from "@/hooks/useLiveStats";
import { AIModelInfoWithEndpointCount } from "@/types";

interface OverviewKpisProps {
  endpointsTotal: number;
  endpointsAvailable: number;
  modelsTotal: number;
  modelsAvailable: number;
  topModel?: AIModelInfoWithEndpointCount;
  liveStats: LiveStatsData;
  loading?: boolean;
}

interface KpiCardProps {
  label: string;
  value: string;
  context: string;
  accent?: "primary" | "success" | "warning" | "secondary";
}

const accentClasses = {
  primary: "text-primary",
  success: "text-success",
  warning: "text-warning",
  secondary: "text-secondary",
};

const KpiCard = ({
  label,
  value,
  context,
  accent = "primary",
  loading = false,
}: KpiCardProps & { loading?: boolean }) => (
  <Card className="p-4 shadow-sm border border-default-200">
    <p className="text-xs uppercase tracking-wide text-default-500 font-semibold">
      {label}
    </p>
    <p className={`text-3xl font-bold mt-2 ${loading ? "text-default-300 animate-pulse" : accentClasses[accent]}`}>
      {loading ? "…" : value}
    </p>
    <p className="text-sm text-default-400 mt-1 truncate" title={context}>
      {context}
    </p>
  </Card>
);

const formatPercent = (part: number, total: number) => {
  if (total <= 0) return 0;
  return Math.round((part / total) * 100);
};

const formatModelName = (model?: AIModelInfoWithEndpointCount) => {
  if (!model) return "—";
  return model.tag ? `${model.name}:${model.tag}` : model.name;
};

const OverviewKpis = ({
  endpointsTotal,
  endpointsAvailable,
  modelsTotal,
  modelsAvailable,
  topModel,
  liveStats,
  loading = false,
}: OverviewKpisProps) => {
  const successfulRequests = Math.max(
    liveStats.total_requests - liveStats.failed_requests,
    0,
  );
  const successRate = formatPercent(
    successfulRequests,
    liveStats.total_requests,
  );
  const cacheRate = formatPercent(
    liveStats.cache_hits,
    liveStats.total_requests,
  );

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-5 gap-4 mb-6">
      <KpiCard
        accent="primary"
        context={
          loading
            ? "Loading endpoint stats…"
            : `${endpointsAvailable} of ${endpointsTotal} available (${formatPercent(endpointsAvailable, endpointsTotal)}%)`
        }
        label="Endpoints"
        loading={loading}
        value={`${endpointsAvailable}/${endpointsTotal}`}
      />
      <KpiCard
        accent="success"
        context={
          loading
            ? "Loading model stats…"
            : `${modelsAvailable} of ${modelsTotal} available (${formatPercent(modelsAvailable, modelsTotal)}%)`
        }
        label="AI Models"
        loading={loading}
        value={`${modelsAvailable}/${modelsTotal}`}
      />
      <KpiCard
        accent="success"
        context={
          liveStats.total_requests > 0
            ? `${liveStats.failed_requests} failed · ${cacheRate}% cache hits`
            : "Waiting for proxy traffic"
        }
        label="Proxy Success"
        value={
          liveStats.total_requests > 0 ? `${successRate}%` : "—"
        }
      />
      <KpiCard
        accent="secondary"
        context={
          topModel?.composite_score != null
            ? `Score ${topModel.composite_score.toFixed(1)} · ${topModel.endpoints} endpoints`
            : "No scored models yet"
        }
        label="Top Model"
        loading={loading}
        value={formatModelName(topModel)}
      />
      <KpiCard
        accent="warning"
        context={`${liveStats.tester_pending} pending in queue`}
        label="Tester Throughput"
        value={`${liveStats.tester_speed}/min`}
      />
    </div>
  );
};

export default OverviewKpis;