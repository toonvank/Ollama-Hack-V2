import { Card, CardBody, CardHeader } from "@heroui/card";
import { ReactNode } from "react";
import clsx from "clsx";

interface SectionCardProps {
  title: string;
  subtitle?: string;
  live?: boolean;
  className?: string;
  bodyClassName?: string;
  headerAction?: ReactNode;
  children: ReactNode;
}

const SectionCard = ({
  title,
  subtitle,
  live = false,
  className,
  bodyClassName,
  headerAction,
  children,
}: SectionCardProps) => {
  return (
    <Card className={clsx("shadow-sm border border-default-200", className)}>
      <CardHeader className="flex items-start justify-between gap-3 px-6 pt-5 pb-0">
        <div className="flex flex-col gap-1 min-w-0">
          <h3 className="font-semibold flex items-center text-lg">
            <span className="relative flex h-2.5 w-2.5 mr-2.5 shrink-0">
              {live ? (
                <>
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75" />
                  <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-success" />
                </>
              ) : (
                <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-primary" />
              )}
            </span>
            <span className="truncate">{title}</span>
          </h3>
          {subtitle && (
            <p className="text-small text-default-500 pl-5">{subtitle}</p>
          )}
        </div>
        {headerAction}
      </CardHeader>
      <CardBody className={clsx("px-6 pb-6 pt-4", bodyClassName)}>
        {children}
      </CardBody>
    </Card>
  );
};

export default SectionCard;