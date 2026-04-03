import React from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { Card, CardBody } from "@nextui-org/card";

import LoadingSpinner from "./LoadingSpinner";

import { useAuth } from "@/contexts/AuthContext";
import DashboardLayout from "@/layouts/Main";

interface ProtectedRouteProps {
  requireAdmin?: boolean;
}

const ProtectedRoute = ({ requireAdmin = false }: ProtectedRouteProps) => {
  const { isAuthenticated, isLoading, isAdmin } = useAuth();
  const location = useLocation();

  if (isLoading) {
    // Loading, return a loading component
    return (
      <DashboardLayout>
        <Card>
          <CardBody>
            <LoadingSpinner />
          </CardBody>
        </Card>
      </DashboardLayout>
    );
  }

  // If user is not authenticated, redirect to login
  if (!isAuthenticated) {
    return <Navigate replace state={{ from: location }} to="/login" />;
  }

  // If admin required but user is not admin, redirect to unauthorized page
  if (requireAdmin && !isAdmin) {
    return <Navigate replace to="/unauthorized" />;
  }

  // User is authenticated and meets permission requirements, render child routes
  return <Outlet />;
};

export default ProtectedRoute;
