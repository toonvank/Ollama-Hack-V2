import { SVGProps } from "react";
import { FontAwesomeIconProps as FontAwesomeIconPropsType } from "@fortawesome/react-fontawesome";
export type IconSvgProps = SVGProps<SVGSVGElement> & {
  size?: number;
};

// remove icon prop
export type FontAwesomeIconProps = Omit<FontAwesomeIconPropsType, "icon">;

// Export all types
export * from "./common";
export * from "./auth";
export * from "./endpoint";
export * from "./model";
export * from "./apikey";
export * from "./plan";
export * from "./setting";
