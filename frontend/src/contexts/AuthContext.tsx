import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { DialogProvider } from "./DialogContext";

import { authApi } from "@/api";
import { UserInfo } from "@/types";

interface AuthContextType {
  user: UserInfo | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  isAdmin: boolean;
  token: string | null;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  getTokenFromStorage: () => string | null;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [user, setUser] = useState<UserInfo | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [token, setToken] = useState<string | null>(null);

  const getTokenFromStorage = useCallback(() => {
    return localStorage.getItem("auth_token");
  }, []);

  const loadUser = useCallback(async () => {
    const storedToken = getTokenFromStorage();

    if (!storedToken) {
      setIsLoading(false);

      return;
    }

    setToken(storedToken);
    try {
      const userData = await authApi.getCurrentUser();

      setUser(userData);
    } catch (error: unknown) {
      const status = (error as { response?: { status?: number } })?.response
        ?.status;

      // Only clear session on explicit auth failure — not on 502/timeouts during VPN flaps
      if (status === 401 || status === 403) {
        localStorage.removeItem("auth_token");
        setToken(null);
        setUser(null);
      }
    } finally {
      setIsLoading(false);
    }
  }, [getTokenFromStorage]);

  useEffect(() => {
    loadUser();
  }, [loadUser]);

  const login = async (username: string, password: string) => {
    try {
      const response = await authApi.login(username, password);

      localStorage.setItem("auth_token", response.access_token);
      setToken(response.access_token);
      await loadUser();
    } catch (error) {
      throw error;
    }
  };

  const logout = () => {
    localStorage.removeItem("auth_token");
    setToken(null);
    setUser(null);
  };

  const value = {
    user,
    isLoading,
    isAuthenticated: !!token,
    isAdmin: user?.is_admin || false,
    token,
    login,
    logout,
    getTokenFromStorage,
  };

  return (
    <AuthContext.Provider value={value}>
      <DialogProvider>{children}</DialogProvider>
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);

  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }

  return context;
};

export default AuthContext;
