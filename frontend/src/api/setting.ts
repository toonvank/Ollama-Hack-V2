import apiClient from "./client";

import { SystemSettingKey, SystemSettings } from "@/types";

export const settingApi = {
  // Get all settings
  getSettings: () => {
    return apiClient.get<Record<SystemSettingKey, string>>(`/api/v2/setting/`);
  },

  // Get setting by key
  getSettingByKey: (key: SystemSettingKey) => {
    return apiClient.get<SystemSettings>(`/api/v2/setting/${key}`);
  },

  // UpdateSettings
  updateSetting: (key: SystemSettingKey, value: string) => {
    return apiClient.patch<SystemSettings>(
      `/api/v2/setting/${key}?value=${value}`,
    );
  },
};

export default settingApi;
