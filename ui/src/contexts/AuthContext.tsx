import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { User, Tenant } from '../types';
import api from '../lib/api';

interface AuthContextType {
  user: User | null;
  tenant: Tenant | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  login: (email: string, password: string, tenantId: string) => Promise<void>;
  logout: () => Promise<void>;
  setTenant: (tenant: Tenant) => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const token = api.getToken();
    if (token) {
      api.getMe()
        .then(user => setUser(user))
        .catch(() => {
          api.setToken(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = async (email: string, password: string, tenantId: string) => {
    const result = await api.login(email, password, tenantId);
    setUser(result.user);
  };

  const logout = async () => {
    await api.logout();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        tenant,
        isLoading,
        isAuthenticated: !!user,
        login,
        logout,
        setTenant,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
