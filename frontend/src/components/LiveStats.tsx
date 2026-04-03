import { useEffect, useState } from "react";
import ReactApexChart from "react-apexcharts";
import { Card, CardHeader } from "@heroui/card";

export const LiveStats = () => {
  const [stats, setStats] = useState({
    total_requests: 0,
    active_requests: 0,
    cache_hits: 0,
    failed_requests: 0,
  });

  const [history, setHistory] = useState<{ x: number; y: number }[]>([]);

  useEffect(() => {
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8000";
    const evtSource = new EventSource(`${apiUrl}/api/v2/stats/live`);

    evtSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setStats(data);

      setHistory((prev) => {
        const newHistory = [...prev, { x: new Date().getTime(), y: data.active_requests }];
        return newHistory.slice(-20); // Keep last 20 seconds
      });
    };

    return () => {
      evtSource.close();
    };
  }, []);

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
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <Card className="col-span-1 p-6 shadow-sm border border-gray-100 dark:border-gray-800 flex flex-col justify-between">
        <div>
          <h4 className="text-gray-500 text-sm font-semibold uppercase">Total Requests</h4>
          <p className="text-3xl font-bold text-primary">{stats.total_requests}</p>
        </div>
        <div className="mt-4">
          <h4 className="text-gray-500 text-sm font-semibold uppercase flex justify-between">
            Cache Hits <span className="text-success">{stats.total_requests > 0 ? Math.round((stats.cache_hits / stats.total_requests) * 100) : 0}%</span>
          </h4>
          <p className="text-3xl font-bold text-success">{stats.cache_hits}</p>
        </div>
        <div className="mt-4 border-t border-gray-100 dark:border-gray-800 pt-4">
          <h4 className="text-gray-500 text-sm font-semibold uppercase">Failed Routes</h4>
          <p className="text-3xl font-bold text-danger">{stats.failed_requests}</p>
        </div>
      </Card>
      
      <Card className="col-span-3 p-4 shadow-sm border border-gray-100 dark:border-gray-800">
        <CardHeader className="p-0 pb-2">
          <h3 className="font-semibold flex items-center text-lg">
            <span className="relative flex h-3 w-3 mr-3 mt-1">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-success"></span>
            </span>
            Live Proxy Traffic 
          </h3>
        </CardHeader>
        <div className="h-64 mt-2">
          {typeof window !== "undefined" && (
            <ReactApexChart
              options={chartOptions}
              series={[{ name: "Active Prompts", data: history }]}
              type="area"
              height="100%"
            />
          )}
        </div>
      </Card>
    </div>
  );
};
