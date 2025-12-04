import { useSchema } from '../contexts/SchemaContext';
import { Layers, Database, FileText, Users } from 'lucide-react';
import { Link } from 'react-router-dom';

export default function Dashboard() {
  const { modules, entities } = useSchema();

  const stats = [
    {
      label: 'Modules',
      value: modules.length,
      icon: Layers,
      color: 'bg-indigo-500',
      link: '/admin/modules',
    },
    {
      label: 'Entities',
      value: entities.length,
      icon: Database,
      color: 'bg-emerald-500',
      link: '/admin/entities',
    },
    {
      label: 'Articles',
      value: '-',
      icon: FileText,
      color: 'bg-amber-500',
      link: '/app/articles',
    },
    {
      label: 'Users',
      value: '-',
      icon: Users,
      color: 'bg-rose-500',
      link: '/admin/users',
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-500 mt-1">Welcome to Genesis - Your No-Code Business Platform</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map(stat => (
          <Link
            key={stat.label}
            to={stat.link}
            className="bg-white rounded-xl border border-gray-200 p-6 hover:shadow-lg hover:-translate-y-1 transition-all"
          >
            <div className="flex items-center gap-4">
              <div className={`w-12 h-12 ${stat.color} rounded-xl flex items-center justify-center text-white`}>
                <stat.icon size={24} />
              </div>
              <div>
                <div className="text-2xl font-bold text-gray-900">{stat.value}</div>
                <div className="text-sm text-gray-500">{stat.label}</div>
              </div>
            </div>
          </Link>
        ))}
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Active Modules</h2>
          {modules.length === 0 ? (
            <p className="text-gray-500">No modules configured yet.</p>
          ) : (
            <div className="space-y-2">
              {modules.filter(m => m.is_active).map(module => (
                <div
                  key={module.id}
                  className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-indigo-100 rounded-lg flex items-center justify-center">
                      <Layers size={16} className="text-indigo-600" />
                    </div>
                    <span className="font-medium text-gray-900">{module.name}</span>
                  </div>
                  <span className="text-xs text-gray-500">{module.code}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-white rounded-xl border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h2>
          <div className="space-y-3">
            <Link
              to="/admin/modules"
              className="block p-4 bg-indigo-50 hover:bg-indigo-100 rounded-lg transition-colors"
            >
              <div className="font-medium text-indigo-700">Create New Module</div>
              <div className="text-sm text-indigo-600">Add a new module to organize your entities</div>
            </Link>
            <Link
              to="/admin/entities"
              className="block p-4 bg-emerald-50 hover:bg-emerald-100 rounded-lg transition-colors"
            >
              <div className="font-medium text-emerald-700">Design Entity</div>
              <div className="text-sm text-emerald-600">Create new entity with custom fields</div>
            </Link>
            <Link
              to="/app/articles"
              className="block p-4 bg-amber-50 hover:bg-amber-100 rounded-lg transition-colors"
            >
              <div className="font-medium text-amber-700">Manage Articles</div>
              <div className="text-sm text-amber-600">Create and publish articles</div>
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
