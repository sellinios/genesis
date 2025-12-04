import { Outlet, Link, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useSchema } from '../contexts/SchemaContext';
import {
  Home,
  FileText,
  Settings,
  Users,
  Database,
  LogOut,
  ChevronDown,
  Menu,
  X,
  Layers,
  Box,
} from 'lucide-react';
import { useState } from 'react';

const iconMap: Record<string, typeof Home> = {
  home: Home,
  file: FileText,
  settings: Settings,
  users: Users,
  database: Database,
  layers: Layers,
  box: Box,
};

export default function Layout() {
  const { user, logout } = useAuth();
  const { modules, getEntitiesByModule } = useSchema();
  const location = useLocation();
  const navigate = useNavigate();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [expandedModules, setExpandedModules] = useState<string[]>([]);

  const toggleModule = (moduleId: string) => {
    setExpandedModules(prev =>
      prev.includes(moduleId)
        ? prev.filter(id => id !== moduleId)
        : [...prev, moduleId]
    );
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const isActive = (path: string) => location.pathname === path;
  const isModuleActive = (moduleCode: string) => location.pathname.startsWith(`/app/${moduleCode}`);

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside
        className={`${
          sidebarOpen ? 'w-64' : 'w-0'
        } bg-white border-r border-gray-200 flex flex-col transition-all duration-200 overflow-hidden`}
      >
        {/* Logo */}
        <div className="h-16 flex items-center px-6 border-b border-gray-200 bg-gradient-to-b from-indigo-50 to-white">
          <span className="text-xl font-bold text-indigo-600">Genesis</span>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-4">
          {/* Dashboard */}
          <div className="px-3 mb-2">
            <Link
              to="/app"
              className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive('/app')
                  ? 'bg-indigo-50 text-indigo-600'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
              }`}
            >
              <Home size={18} />
              Dashboard
            </Link>
          </div>

          {/* Articles (Aethra) */}
          <div className="px-3 mb-2">
            <Link
              to="/app/articles"
              className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                location.pathname.startsWith('/app/articles')
                  ? 'bg-indigo-50 text-indigo-600'
                  : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
              }`}
            >
              <FileText size={18} />
              Articles
            </Link>
          </div>

          {/* Dynamic Modules */}
          {modules.filter(m => m.is_active).map(module => {
            const moduleEntities = getEntitiesByModule(module.id).filter(e => e.is_active);
            const isExpanded = expandedModules.includes(module.id);
            const IconComponent = iconMap[module.icon] || Box;

            return (
              <div key={module.id} className="px-3 mb-1">
                <button
                  onClick={() => toggleModule(module.id)}
                  className={`w-full flex items-center justify-between gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                    isModuleActive(module.code)
                      ? 'bg-indigo-50 text-indigo-600'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
                  }`}
                >
                  <div className="flex items-center gap-3">
                    <IconComponent size={18} />
                    {module.name}
                  </div>
                  {moduleEntities.length > 0 && (
                    <ChevronDown
                      size={16}
                      className={`transition-transform ${isExpanded ? 'rotate-180' : ''}`}
                    />
                  )}
                </button>

                {isExpanded && moduleEntities.length > 0 && (
                  <div className="ml-6 mt-1 space-y-1">
                    {moduleEntities.map(entity => (
                      <Link
                        key={entity.id}
                        to={`/app/${module.code}/${entity.code}`}
                        className={`block px-3 py-2 rounded-lg text-sm transition-colors ${
                          isActive(`/app/${module.code}/${entity.code}`)
                            ? 'bg-indigo-50 text-indigo-600 font-medium'
                            : 'text-gray-500 hover:bg-gray-50 hover:text-gray-700'
                        }`}
                      >
                        {entity.name}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            );
          })}

          {/* Admin Section */}
          <div className="mt-6 pt-6 border-t border-gray-200">
            <div className="px-3 mb-2">
              <span className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                Admin
              </span>
            </div>
            <div className="px-3 space-y-1">
              <Link
                to="/admin/modules"
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  location.pathname.startsWith('/admin/modules')
                    ? 'bg-indigo-50 text-indigo-600'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
                }`}
              >
                <Layers size={18} />
                Module Designer
              </Link>
              <Link
                to="/admin/entities"
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  location.pathname.startsWith('/admin/entities')
                    ? 'bg-indigo-50 text-indigo-600'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
                }`}
              >
                <Database size={18} />
                Entity Designer
              </Link>
              <Link
                to="/admin/users"
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  location.pathname.startsWith('/admin/users')
                    ? 'bg-indigo-50 text-indigo-600'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-indigo-600'
                }`}
              >
                <Users size={18} />
                Users
              </Link>
            </div>
          </div>
        </nav>

        {/* User */}
        <div className="border-t border-gray-200 p-4">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg bg-indigo-600 text-white flex items-center justify-center font-semibold text-sm">
              {user?.first_name?.[0] || 'U'}
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-gray-900 truncate">
                {user?.first_name} {user?.last_name}
              </div>
              <div className="text-xs text-gray-500 truncate">{user?.email}</div>
            </div>
            <button
              onClick={handleLogout}
              className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
              title="Logout"
            >
              <LogOut size={18} />
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Header */}
        <header className="h-16 bg-white border-b border-gray-200 flex items-center px-6 gap-4">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </header>

        {/* Page Content */}
        <main className="flex-1 overflow-y-auto bg-gray-50 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
