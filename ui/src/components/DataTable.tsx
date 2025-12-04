import { Edit2, Trash2, Eye } from 'lucide-react';
import type { Field } from '../types';

interface DataTableProps {
  fields: Field[];
  data: Record<string, unknown>[];
  isLoading?: boolean;
  onView?: (record: Record<string, unknown>) => void;
  onEdit?: (record: Record<string, unknown>) => void;
  onDelete?: (record: Record<string, unknown>) => void;
}

export default function DataTable({
  fields,
  data,
  isLoading = false,
  onView,
  onEdit,
  onDelete,
}: DataTableProps) {
  const visibleFields = fields
    .filter(f => f.show_in_list)
    .sort((a, b) => a.sort_order - b.sort_order);

  const formatValue = (value: unknown, field: Field): string => {
    if (value === null || value === undefined) return '-';

    switch (field.field_type) {
      case 'boolean':
        return value ? 'Yes' : 'No';
      case 'date':
        return new Date(String(value)).toLocaleDateString();
      case 'datetime':
        return new Date(String(value)).toLocaleString();
      case 'decimal':
        return Number(value).toFixed(2);
      default:
        return String(value);
    }
  };

  if (isLoading) {
    return (
      <div className="bg-white rounded-xl border border-gray-200 p-12">
        <div className="flex items-center justify-center">
          <div className="w-8 h-8 border-2 border-indigo-600 border-t-transparent rounded-full animate-spin" />
        </div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="bg-white rounded-xl border border-gray-200 p-12 text-center">
        <div className="text-gray-400 mb-2">
          <Eye size={48} className="mx-auto" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">No data found</h3>
        <p className="text-gray-500">Create your first record to get started</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="bg-gray-50 border-b border-gray-200">
              {visibleFields.map(field => (
                <th
                  key={field.id}
                  className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider"
                >
                  {field.name}
                </th>
              ))}
              {(onView || onEdit || onDelete) && (
                <th className="px-6 py-3 text-right text-xs font-semibold text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              )}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {data.map((record, index) => (
              <tr
                key={String(record.id || index)}
                className="hover:bg-gray-50 transition-colors"
              >
                {visibleFields.map(field => (
                  <td
                    key={field.id}
                    className="px-6 py-4 text-sm text-gray-900 whitespace-nowrap"
                  >
                    {formatValue(record[field.code], field)}
                  </td>
                ))}
                {(onView || onEdit || onDelete) && (
                  <td className="px-6 py-4 text-right">
                    <div className="flex items-center justify-end gap-2">
                      {onView && (
                        <button
                          onClick={() => onView(record)}
                          className="p-1.5 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors"
                          title="View"
                        >
                          <Eye size={16} />
                        </button>
                      )}
                      {onEdit && (
                        <button
                          onClick={() => onEdit(record)}
                          className="p-1.5 text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors"
                          title="Edit"
                        >
                          <Edit2 size={16} />
                        </button>
                      )}
                      {onDelete && (
                        <button
                          onClick={() => onDelete(record)}
                          className="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                          title="Delete"
                        >
                          <Trash2 size={16} />
                        </button>
                      )}
                    </div>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
