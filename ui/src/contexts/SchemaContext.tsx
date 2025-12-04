import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import type { Module, Entity, Schema } from '../types';
import api from '../lib/api';
import { useAuth } from './AuthContext';

interface SchemaContextType {
  modules: Module[];
  entities: Entity[];
  isLoading: boolean;
  reload: () => Promise<void>;
  getEntitiesByModule: (moduleId: string) => Entity[];
  getModuleByCode: (code: string) => Module | undefined;
  getEntityByCode: (code: string) => Entity | undefined;
}

const SchemaContext = createContext<SchemaContextType | null>(null);

export function SchemaProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth();
  const [modules, setModules] = useState<Module[]>([]);
  const [entities, setEntities] = useState<Entity[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const loadSchema = async () => {
    if (!isAuthenticated) return;

    setIsLoading(true);
    try {
      const schema = await api.getSchema();
      setModules(schema.modules || []);
      setEntities(schema.entities || []);
    } catch (err) {
      console.error('Failed to load schema:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadSchema();
  }, [isAuthenticated]);

  const getEntitiesByModule = (moduleId: string) => {
    return entities.filter(e => e.module_id === moduleId);
  };

  const getModuleByCode = (code: string) => {
    return modules.find(m => m.code === code);
  };

  const getEntityByCode = (code: string) => {
    return entities.find(e => e.code === code);
  };

  return (
    <SchemaContext.Provider
      value={{
        modules,
        entities,
        isLoading,
        reload: loadSchema,
        getEntitiesByModule,
        getModuleByCode,
        getEntityByCode,
      }}
    >
      {children}
    </SchemaContext.Provider>
  );
}

export function useSchema() {
  const context = useContext(SchemaContext);
  if (!context) {
    throw new Error('useSchema must be used within SchemaProvider');
  }
  return context;
}
