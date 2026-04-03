import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tsconfigPaths from "vite-tsconfig-paths";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tsconfigPaths()],
  build: {
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        manualChunks: {
          // React core
          "react-core": ["react", "react-dom", "react-router-dom"],

          // UI component library
          "ui-components": [
            "@nextui-org/number-input",
            "@nextui-org/button",
            "@nextui-org/card",
            "@nextui-org/input",
            "@nextui-org/modal",
            "@nextui-org/navbar",
            "@nextui-org/table",
            "@nextui-org/tabs",
            "@nextui-org/toast",
            "@nextui-org/switch",
            "@nextui-org/system",
            "@nextui-org/link",
            "@nextui-org/dropdown",
            "@nextui-org/code",
            "@nextui-org/avatar",
            "@nextui-org/autocomplete",
            "@nextui-org/snippet",
            "@nextui-org/pagination",
            "@nextui-org/use-theme",
            "@nextui-org/drawer",
            "@nextui-org/form",
            "@nextui-org/checkbox",
            "@nextui-org/tooltip",
            "@nextui-org/progress",
            "@nextui-org/select",
            "@nextui-org/chip",
          ],

          "ui-support": ["@nextui-org/theme", "framer-motion", "tailwind-variants"],

          // Chart libraries
          charts: ["apexcharts", "react-apexcharts"],

          // Utility libraries
          utils: [
            "axios",
            "date-fns",
            "clsx",
            "jwt-decode",
            "@tanstack/react-query",
            "react-hook-form",
            "react-syntax-highlighter",
          ],

          // Icon libraries
          icons: [
            "@fortawesome/fontawesome-svg-core",
            "@fortawesome/free-regular-svg-icons",
            "@fortawesome/free-solid-svg-icons",
            "@fortawesome/react-fontawesome",
          ],
        },
      },
    },
  },
});
