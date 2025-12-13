import api from '@/lib/api';
import { RcloneOption, type RcloneProvider, type Remote, type RemoteQuota } from '@/lib/types';

export const getConnections = async () => {
  const response = await api.get<Remote[]>('/remotes');
  return response.data;
};

export const createConnection = async (name: string, params: Record<string, string>) => {
  const response = await api.post(`/remotes/${name}`, params);
  return response.data;
};

export const getProviders = async () => {
  const response = await api.get<RcloneProvider[]>('/providers');
  return response.data;
};

export const getProviderOptions = async (provider: string) => {
  const response = await api.get<{ options: RcloneOption[] }>(`/providers/${provider}`);
  return response.data.options;
};

export const testConnection = async (provider: string, params: Record<string, string>) => {
  const response = await api.post('/remotes/test', { provider, params });
  return response.data;
};

export const getRemoteQuota = async (name: string) => {
  const response = await api.get<RemoteQuota>(`/remotes/${name}/quota`);
  return response.data;
};

export const getRemoteConfig = async (name: string) => {
  const response = await api.get<Record<string, string>>(`/remotes/${name}`);
  return response.data;
};

export const deleteConnection = async (name: string) => {
  const response = await api.delete(`/remotes/${name}`);
  return response.data;
};
