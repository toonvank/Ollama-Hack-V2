import React from "react";
import { Card, CardHeader, CardBody } from "@heroui/card";
import { Chip } from "@heroui/chip";
import { Spinner } from "@heroui/spinner";
import { useCustomQuery } from "@/hooks";
import { aiModelApi } from "@/api";

const SmartModelsCard: React.FC = () => {
  const { data, isLoading, error } = useCustomQuery(
    ["smart-models"],
    () => aiModelApi.getSmartModels(),
    { staleTime: 10000, refetchInterval: 30000 } // Refresh every 30s
  );

  return (
    <Card className="w-full mb-6">
      <CardHeader className="flex gap-3">
        <div className="flex flex-col">
          <p className="text-lg font-semibold">Smart Model Profiles</p>
          <p className="text-small text-default-500">
            Current model resolutions for smart routing
          </p>
        </div>
      </CardHeader>
      <CardBody>
        {isLoading && (
          <div className="flex justify-center items-center p-4">
            <Spinner size="lg" />
          </div>
        )}

        {error && (
          <div className="text-danger p-4">
            Failed to load smart model information
          </div>
        )}

        {data && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {data.smart_models.map((sm) => (
              <div
                key={sm.smart_model}
                className="border border-default-200 rounded-lg p-4 dark:border-default-100"
              >
                <div className="flex items-center justify-between mb-2">
                  <h3 className="text-base font-semibold text-primary">
                    {sm.smart_model}
                  </h3>
                  {sm.resolved ? (
                    <Chip color="success" size="sm" variant="flat">
                      Active
                    </Chip>
                  ) : (
                    <Chip color="warning" size="sm" variant="flat">
                      No Match
                    </Chip>
                  )}
                </div>
                
                <p className="text-sm text-default-500 mb-3">
                  {sm.description}
                </p>

                {sm.resolved ? (
                  <div className="space-y-1.5">
                    <div className="flex items-baseline gap-2">
                      <span className="text-xs text-default-400 min-w-[80px]">
                        Model:
                      </span>
                      <span className="text-sm font-medium text-default-700 dark:text-default-300">
                        {sm.model_full}
                      </span>
                    </div>
                    <div className="flex items-baseline gap-2">
                      <span className="text-xs text-default-400 min-w-[80px]">
                        Endpoint:
                      </span>
                      <span className="text-sm text-default-600 dark:text-default-400">
                        {sm.endpoint}
                      </span>
                    </div>
                    {sm.token_per_second && (
                      <div className="flex items-baseline gap-2">
                        <span className="text-xs text-default-400 min-w-[80px]">
                          Speed:
                        </span>
                        <span className="text-sm text-success font-medium">
                          {sm.token_per_second.toFixed(1)} tok/s
                        </span>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-sm text-warning">
                    {sm.error || "No models available"}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </CardBody>
    </Card>
  );
};

export default SmartModelsCard;
