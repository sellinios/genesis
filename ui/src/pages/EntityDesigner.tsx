import { useState, useEffect } from 'react';
import { useSchema } from '../contexts/SchemaContext';
import api from '../lib/api';
import Modal from '../components/Modal';
import { Plus, Edit2, Trash2, Database, ChevronDown, ChevronRight, Settings } from 'lucide-react';
import type { Entity, Field, FieldType } from '../types';

export default function EntityDesigner() {
  const { modules, entities, loadSchema } = useSchema();
  const [fieldTypes, setFieldTypes] = useState<FieldType[]>([]);
  const [expandedModules, setExpandedModules] = useState<Set<string>>(new Set());
  const [selectedEntity, setSelectedEntity] = useState<Entity | null>(null);
  const [entityFields, setEntityFields] = useState<Field[]>([]);
  const [isLoadingFields, setIsLoadingFields] = useState(false);

  const [isEntityModalOpen, setIsEntityModalOpen] = useState(false);
  const [editingEntity, setEditingEntity] = useState<Entity | null>(null);
  const [entityFormData, setEntityFormData] = useState({
    name: '',
    code: '',
    description: '',
    table_name: '',
    module_id: '',
    is_active: true,
  });

  const [isFieldModalOpen, setIsFieldModalOpen] = useState(false);
  const [editingField, setEditingField] = useState<Field | null>(null);
  const [fieldFormData, setFieldFormData] = useState({
    name: '',
    code: '',
    description: '',
    field_type_id: '',
    is_required: false,
    is_unique: false,
    show_in_list: true,
    show_in_form: true,
    sort_order: 0,
    default_value: '',
    validation_rules: '',
  });

  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    loadFieldTypes();
    // Expand all modules by default
    setExpandedModules(new Set(modules.map(m => m.id)));
  }, [modules]);

  useEffect(() => {
    if (selectedEntity) {
      loadEntityFields(selectedEntity.id);
    }
  }, [selectedEntity]);

  const loadFieldTypes = async () => {
    try {
      const result = await api.getFieldTypes();
      setFieldTypes(result.field_types || []);
    } catch (err) {
      console.error('Failed to load field types:', err);
    }
  };

  const loadEntityFields = async (entityId: string) => {
    setIsLoadingFields(true);
    try {
      const result = await api.getFields(entityId);
      setEntityFields(result.fields || []);
    } catch (err) {
      console.error('Failed to load fields:', err);
    } finally {
      setIsLoadingFields(false);
    }
  };

  const toggleModule = (moduleId: string) => {
    setExpandedModules(prev => {
      const next = new Set(prev);
      if (next.has(moduleId)) {
        next.delete(moduleId);
      } else {
        next.add(moduleId);
      }
      return next;
    });
  };

  const generateCode = (name: string) => {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_|_$/g, '');
  };

  // Entity handlers
  const handleCreateEntity = (moduleId: string) => {
    setEditingEntity(null);
    setEntityFormData({
      name: '',
      code: '',
      description: '',
      table_name: '',
      module_id: moduleId,
      is_active: true,
    });
    setIsEntityModalOpen(true);
  };

  const handleEditEntity = (entity: Entity) => {
    setEditingEntity(entity);
    setEntityFormData({
      name: entity.name,
      code: entity.code,
      description: entity.description || '',
      table_name: entity.table_name,
      module_id: entity.module_id,
      is_active: entity.is_active,
    });
    setIsEntityModalOpen(true);
  };

  const handleDeleteEntity = async (entity: Entity) => {
    if (!confirm(`Are you sure you want to delete "${entity.name}"? This will also delete all fields.`)) {
      return;
    }

    try {
      await api.deleteEntity(entity.id);
      if (selectedEntity?.id === entity.id) {
        setSelectedEntity(null);
        setEntityFields([]);
      }
      await loadSchema();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const handleSaveEntity = async () => {
    setIsSaving(true);
    setError('');

    try {
      const data = {
        ...entityFormData,
        table_name: entityFormData.table_name || `t_${entityFormData.code}`,
      };

      if (editingEntity) {
        await api.updateEntity(editingEntity.id, data);
      } else {
        await api.createEntity(data);
      }
      setIsEntityModalOpen(false);
      await loadSchema();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  // Field handlers
  const handleCreateField = () => {
    if (!selectedEntity) return;

    setEditingField(null);
    setFieldFormData({
      name: '',
      code: '',
      description: '',
      field_type_id: fieldTypes[0]?.id || '',
      is_required: false,
      is_unique: false,
      show_in_list: true,
      show_in_form: true,
      sort_order: entityFields.length,
      default_value: '',
      validation_rules: '',
    });
    setIsFieldModalOpen(true);
  };

  const handleEditField = (field: Field) => {
    setEditingField(field);
    setFieldFormData({
      name: field.name,
      code: field.code,
      description: field.description || '',
      field_type_id: field.field_type_id,
      is_required: field.is_required,
      is_unique: field.is_unique,
      show_in_list: field.show_in_list,
      show_in_form: field.show_in_form,
      sort_order: field.sort_order,
      default_value: field.default_value || '',
      validation_rules: field.validation_rules || '',
    });
    setIsFieldModalOpen(true);
  };

  const handleDeleteField = async (field: Field) => {
    if (!confirm(`Are you sure you want to delete "${field.name}"?`)) {
      return;
    }

    try {
      await api.deleteField(field.id);
      await loadEntityFields(selectedEntity!.id);
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const handleSaveField = async () => {
    if (!selectedEntity) return;

    setIsSaving(true);
    setError('');

    try {
      const data = {
        ...fieldFormData,
        entity_id: selectedEntity.id,
      };

      if (editingField) {
        await api.updateField(editingField.id, data);
      } else {
        await api.createField(data);
      }
      setIsFieldModalOpen(false);
      await loadEntityFields(selectedEntity.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  const getEntitiesByModule = (moduleId: string) => {
    return entities.filter(e => e.module_id === moduleId);
  };

  return (
    <div className="flex h-[calc(100vh-8rem)] gap-6">
      {/* Left Panel - Entity Tree */}
      <div className="w-80 bg-white rounded-xl border border-gray-200 flex flex-col">
        <div className="p-4 border-b border-gray-200">
          <h2 className="font-semibold text-gray-900">Entities</h2>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {modules.map(module => (
            <div key={module.id} className="mb-2">
              <button
                onClick={() => toggleModule(module.id)}
                className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-50 rounded-lg"
              >
                {expandedModules.has(module.id) ? (
                  <ChevronDown size={16} className="text-gray-400" />
                ) : (
                  <ChevronRight size={16} className="text-gray-400" />
                )}
                <span className="font-medium text-gray-900">{module.name}</span>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCreateEntity(module.id);
                  }}
                  className="ml-auto p-1 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded"
                  title="Add Entity"
                >
                  <Plus size={14} />
                </button>
              </button>

              {expandedModules.has(module.id) && (
                <div className="ml-6 space-y-1">
                  {getEntitiesByModule(module.id).map(entity => (
                    <button
                      key={entity.id}
                      onClick={() => setSelectedEntity(entity)}
                      className={`w-full flex items-center gap-2 px-3 py-2 text-left rounded-lg transition-colors ${
                        selectedEntity?.id === entity.id
                          ? 'bg-indigo-50 text-indigo-700'
                          : 'hover:bg-gray-50 text-gray-700'
                      }`}
                    >
                      <Database size={14} />
                      <span className="text-sm truncate">{entity.name}</span>
                    </button>
                  ))}
                  {getEntitiesByModule(module.id).length === 0 && (
                    <div className="px-3 py-2 text-xs text-gray-400">
                      No entities
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Right Panel - Entity Details & Fields */}
      <div className="flex-1 bg-white rounded-xl border border-gray-200 flex flex-col">
        {selectedEntity ? (
          <>
            {/* Entity Header */}
            <div className="p-4 border-b border-gray-200 flex items-center justify-between">
              <div>
                <h2 className="font-semibold text-gray-900">{selectedEntity.name}</h2>
                <p className="text-sm text-gray-500">
                  Table: {selectedEntity.table_name} | Code: {selectedEntity.code}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleEditEntity(selectedEntity)}
                  className="p-2 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg"
                  title="Edit Entity"
                >
                  <Settings size={18} />
                </button>
                <button
                  onClick={() => handleDeleteEntity(selectedEntity)}
                  className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg"
                  title="Delete Entity"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </div>

            {/* Fields Header */}
            <div className="p-4 border-b border-gray-200 flex items-center justify-between bg-gray-50">
              <h3 className="font-medium text-gray-700">Fields</h3>
              <button
                onClick={handleCreateField}
                className="px-3 py-1.5 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-1"
              >
                <Plus size={14} />
                Add Field
              </button>
            </div>

            {/* Fields List */}
            <div className="flex-1 overflow-y-auto">
              {isLoadingFields ? (
                <div className="flex items-center justify-center h-32">
                  <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-600"></div>
                </div>
              ) : entityFields.length === 0 ? (
                <div className="p-8 text-center text-gray-500">
                  No fields defined. Click "Add Field" to create one.
                </div>
              ) : (
                <table className="w-full">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-4 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Name</th>
                      <th className="px-4 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Type</th>
                      <th className="px-4 py-2 text-center text-xs font-semibold text-gray-600 uppercase">Required</th>
                      <th className="px-4 py-2 text-center text-xs font-semibold text-gray-600 uppercase">List</th>
                      <th className="px-4 py-2 text-center text-xs font-semibold text-gray-600 uppercase">Form</th>
                      <th className="px-4 py-2 text-right text-xs font-semibold text-gray-600 uppercase">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {entityFields.map(field => (
                      <tr key={field.id} className="hover:bg-gray-50">
                        <td className="px-4 py-3">
                          <div className="font-medium text-gray-900">{field.name}</div>
                          <div className="text-xs text-gray-500">{field.code}</div>
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-600">
                          {fieldTypes.find(ft => ft.id === field.field_type_id)?.name || field.field_type_id}
                        </td>
                        <td className="px-4 py-3 text-center">
                          {field.is_required ? (
                            <span className="text-green-600">Yes</span>
                          ) : (
                            <span className="text-gray-400">No</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-center">
                          {field.show_in_list ? (
                            <span className="text-green-600">Yes</span>
                          ) : (
                            <span className="text-gray-400">No</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-center">
                          {field.show_in_form ? (
                            <span className="text-green-600">Yes</span>
                          ) : (
                            <span className="text-gray-400">No</span>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center justify-end gap-1">
                            <button
                              onClick={() => handleEditField(field)}
                              className="p-1.5 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded"
                              title="Edit"
                            >
                              <Edit2 size={14} />
                            </button>
                            <button
                              onClick={() => handleDeleteField(field)}
                              className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded"
                              title="Delete"
                            >
                              <Trash2 size={14} />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-gray-500">
            Select an entity to view and edit its fields
          </div>
        )}
      </div>

      {/* Entity Modal */}
      <Modal
        isOpen={isEntityModalOpen}
        onClose={() => setIsEntityModalOpen(false)}
        title={editingEntity ? 'Edit Entity' : 'New Entity'}
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
              Module
            </label>
            <select
              value={entityFormData.module_id}
              onChange={e => setEntityFormData(prev => ({ ...prev, module_id: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              required
            >
              {modules.map(m => (
                <option key={m.id} value={m.id}>{m.name}</option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={entityFormData.name}
              onChange={e => {
                setEntityFormData(prev => ({
                  ...prev,
                  name: e.target.value,
                  code: prev.code || generateCode(e.target.value),
                  table_name: prev.table_name || `t_${generateCode(e.target.value)}`,
                }));
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="e.g., Product"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Code <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={entityFormData.code}
                onChange={e => setEntityFormData(prev => ({ ...prev, code: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono"
                placeholder="e.g., product"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Table Name <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={entityFormData.table_name}
                onChange={e => setEntityFormData(prev => ({ ...prev, table_name: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono"
                placeholder="e.g., t_product"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={entityFormData.description}
              onChange={e => setEntityFormData(prev => ({ ...prev, description: e.target.value }))}
              rows={2}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="entity_is_active"
              checked={entityFormData.is_active}
              onChange={e => setEntityFormData(prev => ({ ...prev, is_active: e.target.checked }))}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <label htmlFor="entity_is_active" className="text-sm text-gray-700">
              Active
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-200">
          <button
            onClick={() => setIsEntityModalOpen(false)}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSaveEntity}
            disabled={isSaving}
            className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors"
          >
            {isSaving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </Modal>

      {/* Field Modal */}
      <Modal
        isOpen={isFieldModalOpen}
        onClose={() => setIsFieldModalOpen(false)}
        title={editingField ? 'Edit Field' : 'New Field'}
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
              value={fieldFormData.name}
              onChange={e => {
                setFieldFormData(prev => ({
                  ...prev,
                  name: e.target.value,
                  code: prev.code || generateCode(e.target.value),
                }));
              }}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              placeholder="e.g., Product Name"
              required
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Code <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={fieldFormData.code}
                onChange={e => setFieldFormData(prev => ({ ...prev, code: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono"
                placeholder="e.g., product_name"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Field Type <span className="text-red-500">*</span>
              </label>
              <select
                value={fieldFormData.field_type_id}
                onChange={e => setFieldFormData(prev => ({ ...prev, field_type_id: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                required
              >
                {fieldTypes.map(ft => (
                  <option key={ft.id} value={ft.id}>{ft.name}</option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              value={fieldFormData.description}
              onChange={e => setFieldFormData(prev => ({ ...prev, description: e.target.value }))}
              rows={2}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Default Value
              </label>
              <input
                type="text"
                value={fieldFormData.default_value}
                onChange={e => setFieldFormData(prev => ({ ...prev, default_value: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Sort Order
              </label>
              <input
                type="number"
                value={fieldFormData.sort_order}
                onChange={e => setFieldFormData(prev => ({ ...prev, sort_order: parseInt(e.target.value) || 0 }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="field_is_required"
                  checked={fieldFormData.is_required}
                  onChange={e => setFieldFormData(prev => ({ ...prev, is_required: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <label htmlFor="field_is_required" className="text-sm text-gray-700">Required</label>
              </div>
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="field_is_unique"
                  checked={fieldFormData.is_unique}
                  onChange={e => setFieldFormData(prev => ({ ...prev, is_unique: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <label htmlFor="field_is_unique" className="text-sm text-gray-700">Unique</label>
              </div>
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="field_show_in_list"
                  checked={fieldFormData.show_in_list}
                  onChange={e => setFieldFormData(prev => ({ ...prev, show_in_list: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <label htmlFor="field_show_in_list" className="text-sm text-gray-700">Show in List</label>
              </div>
              <div className="flex items-center gap-3">
                <input
                  type="checkbox"
                  id="field_show_in_form"
                  checked={fieldFormData.show_in_form}
                  onChange={e => setFieldFormData(prev => ({ ...prev, show_in_form: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <label htmlFor="field_show_in_form" className="text-sm text-gray-700">Show in Form</label>
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-200">
          <button
            onClick={() => setIsFieldModalOpen(false)}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSaveField}
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
