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
            "@heroui/number-input",
            "@heroui/button",
            "@heroui/card",
            "@heroui/input",
            "@heroui/modal",
            "@heroui/navbar",
            "@heroui/table",
            "@heroui/tabs",
            "@heroui/toast",
            "@heroui/switch",
            "@heroui/system",
            "@heroui/link",
            "@heroui/dropdown",
            "@heroui/code",
            "@heroui/avatar",
            "@heroui/autocomplete",
            "@heroui/snippet",
            "@heroui/pagination",
            "@heroui/use-theme",
            "@heroui/drawer",
            "@heroui/form",
            "@heroui/checkbox",
            "@heroui/tooltip",
            "@heroui/progress",
            "@heroui/select",
            "@heroui/chip",
          ],

          "ui-support": ["@heroui/theme", "framer-motion", "tailwind-variants"],

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
