import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@nextui-org/button";
import { Input } from "@nextui-org/input";
import { Link } from "@nextui-org/link";
import { Form } from "@nextui-org/form";
import { addToast } from "@/utils/toast";
import { Card, CardBody, CardFooter, CardHeader } from "@nextui-org/card";

import { authApi, EnhancedAxiosError } from "@/api";
import LoadingSpinner from "@/components/LoadingSpinner";
import DashboardLayout from "@/layouts/Main";
import { LogoIcon } from "@/components/icons";

const InitPage = () => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isChecking, setIsChecking] = useState(true);
  const [initialized, setInitialized] = useState(false);

  const navigate = useNavigate();

  // Check if system is initialized
  useEffect(() => {
    const checkInitialization = async () => {
      try {
        setIsChecking(true);
        await authApi.getCurrentUser();
        // If user info is retrieved, system is initialized
        setInitialized(true);
        // Auto redirect to login page
        setTimeout(() => navigate("/login"), 2000);
      } catch {
        // Error means system is not initialized, proceed
        setInitialized(false);
      } finally {
        setIsChecking(false);
      }
    };

    checkInitialization();
  }, [navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!username || !password) {
      addToast({
        title: "Initialization Failed",
        description: "Please enter a username and password",
        color: "danger",
      });

      return;
    }

    try {
      setIsLoading(true);
      await authApi.initUser({ username, password });
      setInitialized(true);
      // Redirect to login after initialization
      setTimeout(() => navigate("/login"), 2000);
    } catch (err) {
      addToast({
        title: "Initialization Failed",
        description:
          (err as EnhancedAxiosError).detail || "Initialization failed. Please try again later.",
        color: "danger",
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (isChecking) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-full pt-10">
          <LoadingSpinner className="p-8" size="large" />
        </div>
      </DashboardLayout>
    );
  }

  if (initialized) {
    return (
      <DashboardLayout>
        {/* <div className="flex items-center justify-center min-h-full pt-10">
                    <div className="w-full max-w-md p-8 space-y-8 text-center">
                        <h1 className="text-2xl font-bold">
                            System Initialized
                        </h1>
                        <p>
                            System has already been initialized. Redirecting to login...
                        </p>
                        <Link href="/login" className="inline-block">
                            <Button color="primary">Sign In Now</Button>
                        </Link>
                        </div>
                    </div> */}
        <div className="flex items-center justify-center min-h-full pt-10">
          <Card className="w-full max-w-md p-8">
            <CardHeader>
              <h1 className="text-2xl font-bold">System Initialized</h1>
            </CardHeader>
            <CardBody>
              <p>System has already been initialized. Redirecting to login...</p>
            </CardBody>
            <CardFooter>
              <Link className="inline-block" href="/login">
                <Button color="primary">Sign In Now</Button>
              </Link>
            </CardFooter>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="flex items-center justify-center min-h-full pt-10">
        <Card className="w-full max-w-md p-8">
          <CardHeader>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <LogoIcon className="w-8 h-8" /> Admin Account Setup
            </h1>
          </CardHeader>
          <Form className="space-y-4" onSubmit={handleSubmit}>
            <CardBody className="space-y-4">
              <Input
                isRequired
                label="Username"
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
              <Input
                isRequired
                errorMessage={({ validationDetails, validationErrors }) => {
                  if (validationDetails.tooShort) {
                    return "Password must be at least 8 characters";
                  }
                  if (validationDetails.tooLong) {
                    return "Password must be at most 128 characters";
                  }

                  return validationErrors;
                }}
                label="Password"
                maxLength={128}
                minLength={8}
                placeholder="Enter your password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <Input
                isRequired
                label="Confirm Password"
                placeholder="Please re-enter your password"
                type="password"
                validate={(value) => {
                  if (value !== password) {
                    return "Passwords do not match";
                  }

                  return null;
                }}
              />
            </CardBody>
            <CardFooter>
              <Button
                fullWidth
                color="primary"
                isLoading={isLoading}
                type="submit"
              >
                Initialize System
              </Button>
            </CardFooter>
          </Form>
        </Card>
      </div>
    </DashboardLayout>
  );
};

export default InitPage;
