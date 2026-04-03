import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import { Button } from "@heroui/button";
import { Input } from "@heroui/input";
import { addToast } from "@heroui/toast";
import { Card, CardBody, CardFooter, CardHeader } from "@heroui/card";
import { Form } from "@heroui/form";

import { EnhancedAxiosError } from "@/api";
import { useAuth } from "@/contexts/AuthContext";
import { LogoIcon } from "@/components/icons";
import DashboardLayout from "@/layouts/Main";

const LoginPage = () => {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const navigate = useNavigate();
  const location = useLocation();
  const { login } = useAuth();

  // Get redirect source
  const from =
    (location.state as { from?: { pathname: string } })?.from?.pathname || "/";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!username || !password) {
      addToast({
        title: "Login Failed",
        description: "Please enter a username and password",
        color: "danger",
      });

      return;
    }

    try {
      setIsLoading(true);
      await login(username, password);
      navigate(from, { replace: true });
    } catch (err) {
      addToast({
        title: "Login Failed",
        description:
          (err as EnhancedAxiosError).detail || "An unknown error occurred. Please try again later.",
        color: "danger",
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <DashboardLayout>
      <div className="flex items-center justify-center min-h-full pt-10">
        <Card className="w-full max-w-md p-8">
          <CardHeader>
            <h1 className="text-2xl font-bold flex items-center gap-2">
              <LogoIcon className="w-8 h-8" /> Ollama Hack
            </h1>
          </CardHeader>
          <Form onSubmit={handleSubmit}>
            <CardBody className="space-y-4">
              <Input
                required
                label="Username"
                placeholder="Enter your username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
              <Input
                required
                label="Password"
                placeholder="Enter your password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </CardBody>
            <CardFooter>
              <Button
                fullWidth
                color="primary"
                isLoading={isLoading}
                type="submit"
              >
                Sign In
              </Button>
            </CardFooter>
          </Form>
        </Card>
      </div>
    </DashboardLayout>
  );
};

export default LoginPage;
