// User authentication request
export interface UserAuth {
  username: string;
  password: string;
}

// Token response
export interface Token {
  access_token: string;
  token_type: string;
}

// User info
export interface UserInfo {
  id: number;
  username: string;
  is_admin: boolean;
  plan_id?: number;
  plan_name?: string;
}

// User update request
export interface UserUpdate {
  username?: string;
  is_admin?: boolean;
  password?: string;
  plan_id?: number;
}

// Password change request
export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}
