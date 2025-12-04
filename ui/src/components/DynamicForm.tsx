import { useState, useEffect } from 'react';
import type { Field } from '../types';

interface DynamicFormProps {
  fields: Field[];
  initialValues?: Record<string, unknown>;
  onSubmit: (values: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

export default function DynamicForm({
  fields,
  initialValues = {},
  onSubmit,
  onCancel,
  isLoading = false,
}: DynamicFormProps) {
  const [values, setValues] = useState<Record<string, unknown>>(initialValues);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    setValues(initialValues);
  }, [initialValues]);

  const handleChange = (code: string, value: unknown) => {
    setValues(prev => ({ ...prev, [code]: value }));
    if (errors[code]) {
      setErrors(prev => {
        const next = { ...prev };
        delete next[code];
        return next;
      });
    }
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    fields.forEach(field => {
      if (field.is_required && field.show_in_form) {
        const value = values[field.code];
        if (value === undefined || value === null || value === '') {
          newErrors[field.code] = `${field.name} is required`;
        }
      }
    });

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;
    await onSubmit(values);
  };

  const renderField = (field: Field) => {
    if (!field.show_in_form) return null;

    const value = values[field.code] ?? '';
    const error = errors[field.code];

    const baseClasses = `w-full px-4 py-2.5 border rounded-lg text-sm transition-all focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent ${
      error ? 'border-red-300' : 'border-gray-300'
    }`;

    switch (field.field_type) {
      case 'text':
      case 'email':
      case 'url':
        return (
          <input
            type={field.field_type}
            value={String(value)}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
            placeholder={`Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'number':
      case 'integer':
      case 'decimal':
        return (
          <input
            type="number"
            value={String(value)}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
            step={field.field_type === 'decimal' ? '0.01' : '1'}
          />
        );

      case 'textarea':
        return (
          <textarea
            value={String(value)}
            onChange={e => handleChange(field.code, e.target.value)}
            className={`${baseClasses} min-h-[100px] resize-y`}
            rows={4}
          />
        );

      case 'boolean':
        return (
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={Boolean(value)}
              onChange={e => handleChange(field.code, e.target.checked)}
              className="w-5 h-5 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm text-gray-700">Yes</span>
          </label>
        );

      case 'select':
        return (
          <select
            value={String(value)}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
          >
            <option value="">Select {field.name}</option>
            {field.options?.map(opt => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        );

      case 'date':
        return (
          <input
            type="date"
            value={String(value).split('T')[0] || ''}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
          />
        );

      case 'datetime':
        return (
          <input
            type="datetime-local"
            value={String(value).slice(0, 16) || ''}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
          />
        );

      default:
        return (
          <input
            type="text"
            value={String(value)}
            onChange={e => handleChange(field.code, e.target.value)}
            className={baseClasses}
          />
        );
    }
  };

  const formFields = fields.filter(f => f.show_in_form).sort((a, b) => a.sort_order - b.sort_order);

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {formFields.map(field => (
        <div key={field.id}>
          <label className="block text-sm font-medium text-gray-700 mb-1.5">
            {field.name}
            {field.is_required && <span className="text-red-500 ml-1">*</span>}
          </label>
          {renderField(field)}
          {errors[field.code] && (
            <p className="mt-1 text-sm text-red-600">{errors[field.code]}</p>
          )}
        </div>
      ))}

      <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {isLoading ? 'Saving...' : 'Save'}
        </button>
      </div>
    </form>
  );
}
