import ReactApexChart from "react-apexcharts";

import SectionCard from "@/components/SectionCard";
import { LiveStatsData } from "@/hooks/useLiveStats";

interface LiveStatsProps {
  stats: LiveStatsData;
  history: { x: number; y: number }[];
  connected: boolean;
}

export const LiveStats = ({ stats, history, connected }: LiveStatsProps) => {

  const chartOptions = {
    chart: {
      type: "area" as const,
      animations: {
        enabled: true,
        easing: "linear" as const,
        dynamicAnimation: { speed: 1000 },
      },
      toolbar: { show: false },
      zoom: { enabled: false },
    },
    dataLabels: { enabled: false },
    stroke: { curve: "smooth" as const, width: 2 },
    xaxis: {
      type: "datetime" as const,
      range: 20000,
      labels: { show: false },
    },
    yaxis: {
      min: 0,
      forceNiceScale: true,
    },
    colors: ["#3b82f6"],
    fill: {
      type: "gradient",
    },
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-4 gap-4 mb-6">
      <SectionCard
        bodyClassName="pt-2"
        className="lg:col-span-1"
        title="Live Proxy"
        live={connected}
      >
        <div className="space-y-4">
          <div>
            <p className="text-xs uppercase tracking-wide text-default-500 font-semibold">
              Total Requests
            </p>
            <p className="text-3xl font-bold text-primary">
              {stats.total_requests}
            </p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-wide text-default-500 font-semibold flex justify-between">
              Cache Hits
              <span className="text-success">
                {stats.total_requests > 0
                  ? Math.round((stats.cache_hits / stats.total_requests) * 100)
                  : 0}
                %
              </span>
            </p>
            <p className="text-2xl font-bold text-success">{stats.cache_hits}</p>
          </div>
          <div className="border-t border-default-200 pt-4">
            <p className="text-xs uppercase tracking-wide text-default-500 font-semibold mb-2">
              Tester Queue
            </p>
            <div className="flex justify-between items-end">
              <div>
                <p className="text-2xl font-bold text-warning">
                  {stats.tester_pending}
                </p>
                <span className="text-xs text-default-500 uppercase">
                  Pending
                </span>
              </div>
              <div className="text-right">
                <p className="text-2xl font-bold text-primary">
                  {stats.tester_speed}
                </p>
                <span className="text-xs text-default-500 uppercase">
                  Tests / Min
                </span>
              </div>
            </div>
          </div>
        </div>
      </SectionCard>

      <SectionCard
        bodyClassName="pt-2"
        className="lg:col-span-3"
        live={connected}
        subtitle="Active prompts over the last 20 seconds"
        title="Live Proxy Traffic"
      >
        <div className="h-64">
          {typeof window !== "undefined" && (
            <ReactApexChart
              height="100%"
              options={chartOptions}
              series={[{ name: "Active Prompts", data: history }]}
              type="area"
            />
          )}
        </div>
      </SectionCard>
    </div>
  );
};