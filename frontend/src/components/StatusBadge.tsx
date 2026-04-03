import React from "react";
import { Chip } from "@nextui-org/chip";

import { AIModelStatusEnum, EndpointStatusEnum } from "@/types";

interface StatusBadgeProps {
  status: AIModelStatusEnum | EndpointStatusEnum;
  className?: string;
}

const StatusBadge = React.memo(({ status }: StatusBadgeProps) => {
  const getStatusText = () => {
    switch (status) {
      case "available":
        return "Available";
      case "unavailable":
        return "Unavailable";
      case "fake":
        return "Honeypot";
      case "missing":
        return "Deleted";
      default:
        return status;
    }
  };

  const getBadgeColor = () => {
    switch (status) {
      case "available":
        return "success";
      case "unavailable":
        return "danger";
      case "fake":
        return "warning";
      case "missing":
        return "default";
      default:
        return "default";
    }
  };

  return (
    <Chip color={getBadgeColor()} size="sm" variant="flat">
      {getStatusText()}
    </Chip>
  );
});

StatusBadge.displayName = "StatusBadge";

export default StatusBadge;
