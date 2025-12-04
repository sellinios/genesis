import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SchemaProvider } from './contexts/SchemaContext';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import EntityPage from './pages/EntityPage';
import Articles from './pages/Articles';
import ModuleDesigner from './pages/ModuleDesigner';
import EntityDesigner from './pages/EntityDesigner';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function AppRoutes() {
  const { isAuthenticated } = useAuth();

  return (
    <Routes>
      {/* Public routes */}
      <Route
        path="/login"
        element={isAuthenticated ? <Navigate to="/app" replace /> : <Login />}
      />

      {/* Protected routes */}
      <Route
        path="/app"
        element={
          <ProtectedRoute>
            <SchemaProvider>
              <Layout />
            </SchemaProvider>
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="articles" element={<Articles />} />
        <Route path=":moduleCode/:entityCode" element={<EntityPage />} />
      </Route>

      {/* Admin routes */}
      <Route
        path="/admin"
        element={
          <ProtectedRoute>
            <SchemaProvider>
              <Layout />
            </SchemaProvider>
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/admin/modules" replace />} />
        <Route path="modules" element={<ModuleDesigner />} />
        <Route path="entities" element={<EntityDesigner />} />
      </Route>

      {/* Redirects */}
      <Route path="/" element={<Navigate to="/app" replace />} />
      <Route path="*" element={<Navigate to="/app" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  );
}
