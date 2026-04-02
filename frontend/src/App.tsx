import { Route, Routes } from "react-router-dom";
import { Suspense, lazy } from "react";

import ProtectedRoute from "@/components/ProtectedRoute";
import LoadingFallback from "@/components/LoadingFallback";

// Lazy-load page components with React.lazy
const LoginPage = lazy(() => import("@/pages/login"));
const InitPage = lazy(() => import("@/pages/init"));
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const ProfilePage = lazy(() => import("@/pages/profile"));
const SettingsPage = lazy(() => import("@/pages/settings"));
const EndpointsPage = lazy(() => import("@/pages/endpoints"));
const ModelsPage = lazy(() => import("@/pages/models"));
const ApiKeysPage = lazy(() => import("@/pages/apikeys"));
const UsersPage = lazy(() => import("@/pages/users"));
const PlansPage = lazy(() => import("@/pages/plans"));
const UnauthorizedPage = lazy(() => import("@/pages/unauthorized"));
const NotFoundPage = lazy(() => import("@/pages/notfound"));

function App() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <Routes>
        {/* Public routes */}
        <Route element={<LoginPage />} path="/login" />
        <Route element={<InitPage />} path="/init" />
        <Route element={<UnauthorizedPage />} path="/unauthorized" />

        {/* Protected routes */}
        <Route element={<ProtectedRoute />}>
          <Route element={<DashboardPage />} path="/" />
          {/* Endpoint routes */}
          <Route element={<EndpointsPage />} path="/endpoints/*" />
          {/* Model routes */}
          <Route element={<ModelsPage />} path="/models/*" />
          {/* API Key routes */}
          <Route element={<ApiKeysPage />} path="/apikeys/*" />
          {/* Profile and Settings routes */}
          <Route element={<ProfilePage />} path="/profile" />
          <Route element={<SettingsPage />} path="/settings" />
        </Route>

        {/* Admin routes */}
        <Route element={<ProtectedRoute requireAdmin={true} />}>
          <Route element={<UsersPage />} path="/users/*" />
          <Route element={<PlansPage />} path="/plans/*" />
        </Route>

        {/* 404 page */}
        <Route element={<NotFoundPage />} path="*" />
      </Routes>
    </Suspense>
  );
}

export default App;
