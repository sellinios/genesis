import { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { useSchema } from '../contexts/SchemaContext';
import api from '../lib/api';
import DataTable from '../components/DataTable';
import DynamicForm from '../components/DynamicForm';
import Modal from '../components/Modal';
import { Plus, RefreshCw } from 'lucide-react';
import type { Field } from '../types';

export default function EntityPage() {
  const { entityCode } = useParams<{ entityCode: string }>();
  const { getEntityByCode } = useSchema();
  const entity = entityCode ? getEntityByCode(entityCode) : null;

  const [data, setData] = useState<Record<string, unknown>[]>([]);
  const [fields, setFields] = useState<Field[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingRecord, setEditingRecord] = useState<Record<string, unknown> | null>(null);
  const [formData, setFormData] = useState<Record<string, unknown>>({});
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (entityCode) {
      loadData();
    }
  }, [entityCode]);

  const loadData = async () => {
    if (!entityCode) return;

    setIsLoading(true);
    setError('');

    try {
      const [schemaResult, dataResult] = await Promise.all([
        api.getEntitySchema(entityCode),
        api.getData(entityCode),
      ]);

      setFields(schemaResult.fields || []);
      setData(dataResult.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingRecord(null);
    setFormData({});
    setIsModalOpen(true);
  };

  const handleEdit = (record: Record<string, unknown>) => {
    setEditingRecord(record);
    setFormData({ ...record });
    setIsModalOpen(true);
  };

  const handleDelete = async (record: Record<string, unknown>) => {
    if (!entityCode || !record.id) return;

    if (!confirm('Are you sure you want to delete this record?')) return;

    try {
      await api.deleteRecord(entityCode, String(record.id));
      await loadData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to delete');
    }
  };

  const handleSave = async () => {
    if (!entityCode) return;

    setIsSaving(true);
    try {
      if (editingRecord) {
        await api.updateRecord(entityCode, String(editingRecord.id), formData);
      } else {
        await api.createRecord(entityCode, formData);
      }
      setIsModalOpen(false);
      await loadData();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  if (!entityCode) {
    return (
      <div className="flex items-center justify-center h-64">
        <p className="text-gray-500">No entity selected</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            {entity?.name || entityCode}
          </h1>
          {entity?.description && (
            <p className="text-gray-500 mt-1">{entity.description}</p>
          )}
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={loadData}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors flex items-center gap-2"
          >
            <RefreshCw size={18} />
            Refresh
          </button>
          <button
            onClick={handleCreate}
            className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
          >
            <Plus size={18} />
            Create New
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-red-600">
          {error}
        </div>
      )}

      {/* Loading */}
      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
        </div>
      ) : (
        /* Data Table */
        <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
          <DataTable
            data={data}
            fields={fields}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
          {data.length === 0 && (
            <div className="p-12 text-center text-gray-500">
              No records found. Click "Create New" to add one.
            </div>
          )}
        </div>
      )}

      {/* Create/Edit Modal */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={editingRecord ? 'Edit Record' : 'Create New Record'}
        size="lg"
      >
        <DynamicForm
          fields={fields}
          values={formData}
          onChange={setFormData}
        />
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
