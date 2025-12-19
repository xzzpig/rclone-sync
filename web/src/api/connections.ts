import api from '@/lib/api';
import type {
  Connection,
  ConnectionConfig,
  ImportExecuteRequest,
  ImportParseResult,
  ImportResult,
  RcloneOption,
  RcloneProvider,
  RemoteQuota,
} from '@/lib/types';

// ============= Connection API (new /connections endpoints) =============

/**
 * Get all connections from the database
 */
export const getConnections = async () => {
  const response = await api.get<Connection[]>('/connections');
  return response.data;
};

/**
 * Get a single connection by ID
 */
export const getConnection = async (id: string) => {
  const response = await api.get<Connection>(`/connections/${id}`);
  return response.data;
};

/**
 * Create a new connection
 */
export const createConnection = async (
  name: string,
  type: string,
  config: Record<string, string>
) => {
  const response = await api.post<Connection>('/connections', {
    name,
    type,
    config,
  });
  return response.data;
};

/**
 * Update an existing connection by ID
 */
export const updateConnection = async (
  id: string,
  data: { name?: string; config?: Record<string, string> }
) => {
  const response = await api.put<Connection>(`/connections/${id}`, data);
  return response.data;
};

/**
 * Delete a connection by ID
 * @param force - If true, also delete associated tasks
 */
export const deleteConnection = async (id: string, force = false) => {
  const response = await api.delete(`/connections/${id}`, {
    params: force ? { force: 'true' } : undefined,
  });
  return response.data;
};

/**
 * Get the decrypted config for editing
 */
export const getConnectionConfig = async (id: string) => {
  const response = await api.get<ConnectionConfig>(`/connections/${id}/config`);
  return response.data;
};

/**
 * Test a saved connection
 */
export const testConnection = async (id: string) => {
  const response = await api.post(`/connections/${id}/test`);
  return response.data;
};

/**
 * Test an unsaved connection config
 */
export const testUnsavedConnection = async (type: string, config: Record<string, string>) => {
  const response = await api.post('/connections/test', { type, config });
  return response.data;
};

/**
 * Get quota/usage info for a connection
 */
export const getConnectionQuota = async (id: string) => {
  const response = await api.get<RemoteQuota>(`/connections/${id}/quota`);
  return response.data;
};

// ============= Import API =============

/**
 * Parse rclone.conf content and return preview
 */
export const parseImport = async (content: string): Promise<ImportParseResult> => {
  const response = await api.post<ImportParseResult>('/import/parse', { content });
  return response.data;
};

/**
 * Execute the import with selected connections
 */
export const executeImport = async (request: ImportExecuteRequest): Promise<ImportResult> => {
  const response = await api.post<ImportResult>('/import/execute', request);
  return response.data;
};

// ============= Provider API (unchanged) =============

export const getProviders = async () => {
  const response = await api.get<RcloneProvider[]>('/providers');
  return response.data;
};

export const getProviderOptions = async (provider: string) => {
  const response = await api.get<{ options: RcloneOption[] }>(`/providers/${provider}`);
  return response.data.options;
};
