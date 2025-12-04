import { useState } from 'react';
import { useSchema } from '../contexts/SchemaContext';
import api from '../lib/api';
import Modal from '../components/Modal';
import { Plus, Edit2, Trash2, Layers, ToggleLeft, ToggleRight } from 'lucide-react';
import type { Module } from '../types';

export default function ModuleDesigner() {
  const { modules, loadSchema } = useSchema();
  const [error, setError] = useState('');

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingModule, setEditingModule] = useState<Module | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    code: '',
    description: '',
    icon: '',
    sort_order: 0,
    is_active: true,
  });
  const [isSaving, setIsSaving] = useState(false);

  const handleCreate = () => {
    setEditingModule(null);
    setFormData({
      name: '',
      code: '',
      description: '',
      icon: 'Layers',
      sort_order: modules.length,
      is_active: true,
    });
    setIsModalOpen(true);
  };

  const handleEdit = (module: Module) => {
    setEditingModule(module);
    setFormData({
      name: module.name,
      code: module.code,
      description: module.description || '',
      icon: module.icon || 'Layers',
      sort_order: module.sort_order || 0,
      is_active: module.is_active,
    });
    setIsModalOpen(true);
  };

  const handleDelete = async (module: Module) => {
    if (!confirm(`Are you sure you want to delete "${module.name}"? This will also delete all entities in this module.`)) {
      return;
    }

    try {
      await api.deleteModule(module.id);
      await loadSchema();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const handleToggleActive = async (module: Module) => {
    try {
      await api.updateModule(module.id, { is_active: !module.is_active });
      await loadSchema();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to update');
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setError('');

    try {
      if (editingModule) {
        await api.updateModule(editingModule.id, formData);
      } else {
        await api.createModule(formData);
      }
      setIsModalOpen(false);
      await loadSchema();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  const generateCode = (name: string) => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_|_$/g, '');
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Module Designer</h1>
          <p className="text-gray-500 mt-1">Create and manage modules to organize your entities</p>
        </div>
        <button
          onClick={handleCreate}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
        >
          <Plus size={18} />
          New Module
        </button>
      </div>

      {/* Modules Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {modules.map(module => (
          <div
            key={module.id}
            className={`bg-white rounded-xl border ${
              module.is_active ? 'border-gray-200' : 'border-gray-300 opacity-60'
            } p-6 hover:shadow-lg transition-all`}
          >
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className={`w-10 h-10 ${module.is_active ? 'bg-indigo-100' : 'bg-gray-100'} rounded-lg flex items-center justify-center`}>
                  <Layers size={20} className={module.is_active ? 'text-indigo-600' : 'text-gray-400'} />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900">{module.name}</h3>
                  <p className="text-sm text-gray-500">{module.code}</p>
                </div>
              </div>
              <button
                onClick={() => handleToggleActive(module)}
                className={`p-1 ${module.is_active ? 'text-green-600' : 'text-gray-400'}`}
                title={module.is_active ? 'Active - Click to deactivate' : 'Inactive - Click to activate'}
              >
                {module.is_active ? <ToggleRight size={24} /> : <ToggleLeft size={24} />}
              </button>
            </div>

            {module.description && (
              <p className="text-sm text-gray-600 mb-4">{module.description}</p>
            )}

            <div className="flex items-center justify-between pt-4 border-t border-gray-100">
              <span className="text-xs text-gray-500">
                Sort: {module.sort_order}
              </span>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleEdit(module)}
                  className="p-1.5 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors"
                  title="Edit"
                >
                  <Edit2 size={16} />
                </button>
                <button
                  onClick={() => handleDelete(module)}
                  className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  title="Delete"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          </div>
        ))}

        {modules.length === 0 && (
          <div className="col-span-full py-12 text-center text-gray-500">
            No modules yet. Click "New Module" to create your first module.
          </div>
        )}
      </div>

      {/* Create/Edit Modal */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={editingModule ? 'Edit Module' : 'New Module'}
        size="md"
      >
        {error && (
          <div className="p-4 mb-4 bg-red-50 border border-red-200 rounded-lg text-red-600 text-sm">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={e => {
                setFormData(prev => ({
                  ...prev,
                  name: e.target.value,
                  code: prev.code || generateCode(e.target.value),
                }));
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="e.g., Inventory"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Code <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={formData.code}
              onChange={e => setFormData(prev => ({ ...prev, code: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono"
              placeholder="e.g., inventory"
              required
            />
            <p className="text-xs text-gray-500 mt-1">
              Unique identifier for this module (lowercase, no spaces)
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={formData.description}
              onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
              rows={2}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="What this module is for..."
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Icon
              </label>
              <select
                value={formData.icon}
                onChange={e => setFormData(prev => ({ ...prev, icon: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="Layers">Layers</option>
                <option value="Database">Database</option>
                <option value="Users">Users</option>
                <option value="Settings">Settings</option>
                <option value="FileText">FileText</option>
                <option value="ShoppingCart">ShoppingCart</option>
                <option value="Package">Package</option>
                <option value="Briefcase">Briefcase</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Sort Order
              </label>
              <input
                type="number"
                value={formData.sort_order}
                onChange={e => setFormData(prev => ({ ...prev, sort_order: parseInt(e.target.value) || 0 }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="is_active"
              checked={formData.is_active}
              onChange={e => setFormData(prev => ({ ...prev, is_active: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <label htmlFor="is_active" className="text-sm text-gray-700">
              Active (visible in sidebar)
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-200">
          <button
            onClick={() => setIsModalOpen(false)}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {isSaving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </Modal>
    </div>
  );
}
