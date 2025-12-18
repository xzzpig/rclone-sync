import api from '@/lib/api';
import { JobLog, PaginatedResponse } from '@/lib/types';

export interface LogsParams {
  connection_id: string;
  task_id?: string;
  job_id?: string;
  level?: string;
  limit?: number;
  offset?: number;
}

export const getLogs = async (params: LogsParams) => {
  const response = await api.get<PaginatedResponse<JobLog>>('/logs', { params });
  return response.data;
};
