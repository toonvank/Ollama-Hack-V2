import React, { useMemo } from "react";
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

const MAX_STATUS = 10;

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

const formatDateTime = (dateTimeStr: string) => {
  try {
    const date = new Date(dateTimeStr);

    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    }).format(date);
  } catch {
    return dateTimeStr;
  }
};

const StatusTimeline = React.memo(<T extends PerformanceStatus>({
  performanceTests,
  type = "endpoint",
}: StatusTimelineProps<T>) => {
  const statusList = useMemo(() => {
    const statusItems = [...performanceTests]
      .slice(0, MAX_STATUS)
      .sort((a, b) => {
        return (
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
      });

    const fillerCount = MAX_STATUS - statusItems.length;
    const fillerItems = Array(fillerCount > 0 ? fillerCount : 0).fill(null);

    return [...statusItems, ...fillerItems];
  }, [performanceTests]);

  return (
    <Card className="w-full" shadow="none">
      <CardBody className="flex flex-row items-center justify-center w-full">
        <div className="flex flex-row-reverse w-full justify-end items-center gap-2">
          {statusList.map((test, index) => (
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
}) as <T extends PerformanceStatus>(props: StatusTimelineProps<T>) => React.ReactElement;

export default StatusTimeline;
