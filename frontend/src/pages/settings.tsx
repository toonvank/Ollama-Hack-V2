import { useEffect, useState } from "react";
import { Button } from "@nextui-org/button";
import { Input } from "@nextui-org/input";
import { Card } from "@nextui-org/card";
import { addToast } from "@/utils/toast";

import { authApi, settingApi } from "@/api";
import { useAuth } from "@/contexts/AuthContext";
import DashboardLayout from "@/layouts/Main";
import ErrorDisplay from "@/components/ErrorDisplay";
import { useCustomQuery } from "@/hooks";
import { SystemSettingKey, SystemSettings } from "@/types";

const Settings = () => {
  const { isAdmin, user } = useAuth();
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [
    isUpdateEndpointTaskIntervalLoading,
    setIsUpdateEndpointTaskIntervalLoading,
  ] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [success, setSuccess] = useState(false);
  const [updateEndpointTaskInterval, setUpdateEndpointTaskInterval] =
    useState(24);

  const { data: updateEndpointTaskIntervalData } =
    useCustomQuery<SystemSettings>(
      ["updateEndpointTaskInterval"],
      () =>
        settingApi.getSettingByKey(
          SystemSettingKey.UPDATE_ENDPOINT_TASK_INTERVAL_HOURS,
        ),
      { enabled: !!user && isAdmin, staleTime: 30000 },
    );

  useEffect(() => {
    setUpdateEndpointTaskInterval(
      Number(updateEndpointTaskIntervalData?.value),
    );
  }, [updateEndpointTaskIntervalData]);

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setSuccess(false);

    if (!oldPassword || !newPassword || !confirmPassword) {
      setError(new Error("Please fill in all password fields"));

      return;
    }

    if (newPassword !== confirmPassword) {
      setError(new Error("New password and confirmation do not match"));

      return;
    }

    try {
      setIsLoading(true);
      await authApi.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      });
      // Clear form
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      addToast({
        title: "Password Changed",
        description: "Please sign in with your new password",
        color: "success",
      });
    } catch {
      addToast({
        title: "Password Change Failed",
        description: "Please check if your current password is correct",
        color: "danger",
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleUpdateEndpointTaskInterval = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsUpdateEndpointTaskIntervalLoading(true);

    try {
      await settingApi.updateSetting(
        SystemSettingKey.UPDATE_ENDPOINT_TASK_INTERVAL_HOURS,
        updateEndpointTaskInterval.toString(),
      );
      addToast({
        title: "Interval Updated",
        description: "Please wait for the endpoint tasks to update",
        color: "success",
      });
    } catch {
      addToast({
        title: "Interval Update Failed",
        description: "Please check if the interval value is valid",
        color: "danger",
      });
    } finally {
      setIsUpdateEndpointTaskIntervalLoading(false);
    }

    setUpdateEndpointTaskInterval(Number(updateEndpointTaskIntervalData?.value) || 24);
  };

  return (
    <DashboardLayout current_root_href="/settings">
      <div className="max-w-3xl mx-auto">
        <Card className="p-6">
          <h2 className="text-xl font-semibold mb-6">Change Password</h2>

          {error && <ErrorDisplay className="mb-4" error={error} />}

          {success && (
            <div className="p-4 mb-4 text-white bg-success-500 rounded-md">
              <p>Password changed successfully!</p>
            </div>
          )}

          <form onSubmit={handleChangePassword}>
            <div className="space-y-4">
              <div>
                {/* <label
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                  htmlFor="oldPassword"
                >
                  Current Password
                </label> */}
                <Input
                  fullWidth
                  id="oldPassword"
                  label="Current Password"
                  placeholder="Enter current password"
                  type="password"
                  value={oldPassword}
                  onChange={(e) => setOldPassword(e.target.value)}
                />
              </div>

              <div>
                <Input
                  fullWidth
                  id="newPassword"
                  label="New Password"
                  placeholder="Enter new password"
                  type="password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                />
              </div>

              <div>
                <Input
                  fullWidth
                  id="confirmPassword"
                  label="Confirm New Password"
                  placeholder="Re-enter new password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                />
              </div>
            </div>

            <div className="mt-6">
              <Button color="primary" isLoading={isLoading} type="submit">
                Change Password
              </Button>
            </div>
          </form>
        </Card>

        {isAdmin && (
          <Card className="p-6 mt-6">
            <h2 className="text-xl font-semibold mb-4">System Settings</h2>
            <form onSubmit={handleUpdateEndpointTaskInterval}>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <Input
                    description="Set to 0 to disable auto-update"
                    label="Endpoint Test Interval (hours)"
                    min={0}
                    type="number"
                    value={updateEndpointTaskInterval}
                    onChange={(e) =>
                      setUpdateEndpointTaskInterval(Number(e.target.value))
                    }
                  />
                </div>
                <div className="flex justify-between">
                  <Button
                    color="primary"
                    isLoading={isUpdateEndpointTaskIntervalLoading}
                    type="submit"
                  >
                    Update
                  </Button>
                </div>
              </div>
            </form>
          </Card>
        )}

        <Card className="p-6 mt-6">
          <h2 className="text-xl font-semibold mb-4">Account Info</h2>
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Username:</span>
              <span className="font-medium">{user?.username}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">
                User ID:
              </span>
              <span className="font-medium">{user?.id}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">
                User Type:
              </span>
              <span className="font-medium">
                {user?.is_admin ? "Admin" : "User"}
              </span>
            </div>
          </div>
        </Card>
      </div>
    </DashboardLayout>
  );
};

export default Settings;
