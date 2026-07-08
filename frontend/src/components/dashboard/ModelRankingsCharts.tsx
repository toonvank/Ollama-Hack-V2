import { useEffect, useMemo, useState } from "react";
import Chart from "react-apexcharts";
import { ApexOptions } from "apexcharts";
import { useTheme } from "@heroui/use-theme";

import SectionCard from "@/components/SectionCard";
import LoadingSpinner from "@/components/LoadingSpinner";
import { useCustomQuery } from "@/hooks";
import { aiModelApi } from "@/api";
import { AIModelInfoWithEndpointCount, PageResponse } from "@/types";

const formatModelLabel = (model: AIModelInfoWithEndpointCount) =>
  model.tag ? `${model.name}:${model.tag}` : model.name;

const ModelRankingsCharts = () => {
  const [mounted, setMounted] = useState(false);
  const { theme } = useTheme();

  const { data, isLoading, error } = useCustomQuery<
    PageResponse<AIModelInfoWithEndpointCount>
  >(
    ["models", "dashboard", "rankings"],
    () =>
      aiModelApi.getAIModels({
        page: 1,
        size: 50,
        order_by: "composite_score",
        order: "desc",
      }),
    { staleTime: 30000, refetchInterval: 60000 },
  );

  useEffect(() => {
    setMounted(true);
  }, []);

  const rankedModels = useMemo(
    () =>
      (data?.items ?? []).filter(
        (model) => model.composite_score != null && model.composite_score > 0,
      ),
    [data?.items],
  );

  const topTen = rankedModels.slice(0, 10);

  const frontierModels = useMemo(
    () =>
      rankedModels.filter(
        (model) =>
          model.token_per_second != null && model.token_per_second > 0,
      ),
    [rankedModels],
  );

  const chartTheme: ApexOptions["theme"] = { mode: theme };
  const gridColor = "hsl(var(--heroui-default-300))";
  const labelColor = "hsl(var(--heroui-default-500))";

  const barOptions: ApexOptions = {
    theme: chartTheme,
    chart: {
      type: "bar",
      background: "transparent",
      toolbar: { show: false },
      fontFamily: "Inter, sans-serif",
    },
    plotOptions: {
      bar: {
        horizontal: true,
        borderRadius: 4,
        barHeight: "70%",
      },
    },
    dataLabels: { enabled: false },
    grid: {
      borderColor: gridColor,
      strokeDashArray: 4,
      xaxis: { lines: { show: true } },
      yaxis: { lines: { show: false } },
    },
    xaxis: {
      labels: { style: { colors: labelColor } },
      axisBorder: { show: false },
    },
    yaxis: {
      labels: {
        style: { colors: labelColor },
        maxWidth: 140,
      },
    },
    colors: ["hsl(var(--heroui-primary))"],
    tooltip: {
      y: {
        formatter: (value) => `${Number(value).toFixed(1)} score`,
      },
    },
  };

  const barSeries = [
    {
      name: "Composite Score",
      data: topTen.map((model) => ({
        x: formatModelLabel(model),
        y: model.composite_score ?? 0,
      })),
    },
  ];

  const scatterOptions: ApexOptions = {
    theme: chartTheme,
    chart: {
      type: "scatter",
      background: "transparent",
      toolbar: { show: false },
      fontFamily: "Inter, sans-serif",
      zoom: { enabled: false },
    },
    grid: {
      borderColor: gridColor,
      strokeDashArray: 4,
    },
    xaxis: {
      title: { text: "Composite Score", style: { color: labelColor } },
      tickAmount: 6,
      labels: { style: { colors: labelColor } },
    },
    yaxis: {
      title: { text: "Throughput (tok/s)", style: { color: labelColor } },
      labels: { style: { colors: labelColor } },
      min: 0,
    },
    colors: ["hsl(var(--heroui-success))"],
    markers: {
      size: 6,
      strokeWidth: 0,
      hover: { size: 8 },
    },
    tooltip: {
      custom: ({ seriesIndex, dataPointIndex, w }) => {
        const point = w.config.series[seriesIndex].data[dataPointIndex];
        return `<div class="px-3 py-2 text-sm">
          <div class="font-semibold">${point.label}</div>
          <div>Score: ${point.x.toFixed(1)}</div>
          <div>Throughput: ${point.y.toFixed(1)} tok/s</div>
        </div>`;
      },
    },
  };

  const scatterSeries = [
    {
      name: "Models",
      data: frontierModels.map((model) => ({
        x: model.composite_score ?? 0,
        y: model.token_per_second ?? 0,
        label: formatModelLabel(model),
      })),
    },
  ];

  const renderChartBody = (
    emptyMessage: string,
    options: ApexOptions,
    series: ApexOptions["series"],
    height: number,
  ) => {
    if (isLoading) {
      return (
        <div className="flex justify-center items-center h-72">
          <LoadingSpinner size="medium" />
        </div>
      );
    }

    if (error) {
      return (
        <div className="flex justify-center items-center h-72 text-danger text-sm">
          Failed to load model rankings
        </div>
      );
    }

    if (!mounted || !series?.[0]?.data?.length) {
      return (
        <div className="flex justify-center items-center h-72 text-default-400 text-sm">
          {emptyMessage}
        </div>
      );
    }

    return (
      <div className="h-72">
        <Chart height={height} options={options} series={series} type={options.chart?.type} width="100%" />
      </div>
    );
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <SectionCard
        subtitle="Ranked by composite score across the fleet"
        title="Top Models"
      >
        {renderChartBody(
          "No scored models yet",
          barOptions,
          barSeries,
          288,
        )}
      </SectionCard>

      <SectionCard
        subtitle="Score vs throughput — higher-right is better"
        title="Score vs Throughput"
      >
        {renderChartBody(
          "Need models with both score and throughput data",
          scatterOptions,
          scatterSeries,
          288,
        )}
      </SectionCard>
    </div>
  );
};

export default ModelRankingsCharts;