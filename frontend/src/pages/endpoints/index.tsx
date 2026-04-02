import React, { lazy, Suspense } from "react";
import { Routes, Route } from "react-router-dom";

import LoadingFallback from "@/components/LoadingFallback";

// Lazy load EndpointListPage component
const EndpointListPage = lazy(() => import("@/components/endpoints/ListPage"));

// Router component
const EndpointsPage = () => {
  return (
    <Routes>
      <Route
        element={
          <Suspense fallback={<LoadingFallback fullScreen={false} />}>
            <EndpointListPage />
          </Suspense>
        }
        path="/"
      />
    </Routes>
  );
};

export default EndpointsPage;
