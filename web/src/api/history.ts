import api from '@/lib/api';
import { Job, PaginatedResponse } from '@/lib/types';

export interface JobsParams {
  remote_name?: string;
  task_id?: string;
  limit?: number;
  offset?: number;
}

export const getJobs = async (params?: JobsParams) => {
  const response = await api.get<PaginatedResponse<Job>>('/jobs', { params });
  return response.data;
};

export const getJob = async (id: string) => {
  const response = await api.get<Job>(`/jobs/${id}`);
  return response.data;
};
