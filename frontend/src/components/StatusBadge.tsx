import { Chip } from "@heroui/chip";

import { AIModelStatusEnum, EndpointStatusEnum } from "@/types";

interface StatusBadgeProps {
  status: AIModelStatusEnum | EndpointStatusEnum;
  className?: string;
}

const StatusBadge = ({ status }: StatusBadgeProps) => {
  // // Set color based on status type
  // const getBadgeStyle = () => {
  //     switch (status) {
  //         case "available":
  //             return "bg-success-100 text-success-700 border-success-200";
  //         case "unavailable":
  //             return "bg-danger-100 text-danger-700 border-danger-200";
  //         case "fake":
  //             return "bg-warning-100 text-warning-700 border-warning-200";
  //         case "missing":
  //             return "bg-gray-100 text-gray-700 border-gray-200";
  //         default:
  //             return "bg-gray-100 text-gray-700 border-gray-200";
  //     }
  // };

  // Get status text
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

  // return (
  //     <span
  //         className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${getBadgeStyle()} ${className}`}
  //     >
  //         {status === "available" && (
  //             <span className="w-1.5 h-1.5 mr-1.5 rounded-full bg-success-500"></span>
  //         )}
  //         {status === "unavailable" && (
  //             <span className="w-1.5 h-1.5 mr-1.5 rounded-full bg-danger-500"></span>
  //         )}
  //         {status === "fake" && (
  //             <span className="w-1.5 h-1.5 mr-1.5 rounded-full bg-warning-500"></span>
  //         )}
  //         {status === "missing" && (
  //             <span className="w-1.5 h-1.5 mr-1.5 rounded-full bg-gray-500"></span>
  //         )}
  //         {getStatusText()}
  //     </span>
  // );
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
};

export default StatusBadge;
