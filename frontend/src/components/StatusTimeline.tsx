import { Card, CardBody } from "@heroui/card";
import { Chip } from "@heroui/chip";
import { Tooltip } from "@heroui/tooltip";

import { EndpointStatusEnum, AIModelStatusEnum } from "@/types";

// Generic status interface
interface PerformanceStatus {
  status: EndpointStatusEnum | AIModelStatusEnum;
  created_at: string;
}

interface StatusTimelineProps<T extends PerformanceStatus> {
  performanceTests: T[];
  type?: "endpoint" | "model";
}

const StatusTimeline = <T extends PerformanceStatus>({
  performanceTests,
  type = "endpoint",
}: StatusTimelineProps<T>) => {
  // Show at most 10 statuses
  const maxStatus = 10;

  // Get chip color
  const getStatusColor = (
    status: EndpointStatusEnum | AIModelStatusEnum | undefined,
  ) => {
    switch (status) {
      case EndpointStatusEnum.AVAILABLE:
      case AIModelStatusEnum.AVAILABLE:
        return "success";
      case EndpointStatusEnum.UNAVAILABLE:
      case AIModelStatusEnum.UNAVAILABLE:
        return "danger";
      case EndpointStatusEnum.FAKE:
      case AIModelStatusEnum.FAKE:
        return "warning";
      case AIModelStatusEnum.MISSING:
        return "secondary";
      default:
        return "default";
    }
  };

  // Format date time
  const formatDateTime = (dateTimeStr: string) => {
    const date = new Date(dateTimeStr + "Z");

    return date.toLocaleString({
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  // Create the displayed status array
  const getStatusList = () => {
    // Copy and limit to at most 10 items
    const statusItems = [...performanceTests]
      .slice(0, maxStatus)
      .sort((a, b) => {
        return (
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
      });

    // Fill with empty statuses if less than 10
    const fillerCount = maxStatus - statusItems.length;
    const fillerItems = Array(fillerCount > 0 ? fillerCount : 0).fill(null);

    return [...statusItems, ...fillerItems];
  };

  return (
    <Card className="w-full" shadow="none">
      <CardBody className="flex flex-row items-center justify-center w-full">
        <div className="flex flex-row-reverse w-full justify-end items-center gap-2">
          {getStatusList().map((test, index) => (
            <Tooltip
              key={index}
              content={
                test ? (
                  <div className="text-sm flex flex-col gap-1 items-center">
                    {type === "model" && (
                      <span>
                        {test.token_per_second
                          ? `${test.token_per_second.toFixed(1)} tps`
                          : "0 tps"}
                      </span>
                    )}
                    <span>{formatDateTime(test.created_at)}</span>
                  </div>
                ) : (
                  "No data"
                )
              }
              placement="top"
            >
              <Chip
                className="w-2 h-6 min-w-0 min-h-0 p-0"
                color={getStatusColor(test?.status)}
                variant="solid"
              />
            </Tooltip>
          ))}
        </div>
      </CardBody>
    </Card>
  );
};

export default StatusTimeline;
