import api from '@/lib/api';
import { Task } from '@/lib/types';

export const getTasks = async (params?: { connection_id?: string }) => {
  const response = await api.get<Task[]>('/tasks', { params });
  return response.data;
};

export const runTask = async (id: string) => {
  const response = await api.post(`/tasks/${id}/run`);
  return response.data;
};

export const createTask = async (data: Omit<Task, 'id' | 'edges'>) => {
  const response = await api.post<Task>('/tasks', data);
  return response.data;
};

export const updateTask = async (id: string, data: Partial<Task>) => {
  const response = await api.put<Task>(`/tasks/${id}`, data);
  return response.data;
};

export const deleteTask = async (id: string) => {
  await api.delete(`/tasks/${id}`);
};
