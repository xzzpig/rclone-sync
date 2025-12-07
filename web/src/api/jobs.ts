export interface Job {
    id: string;
    task_id: string;
    status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled';
    start_time: string;
    end_time?: string;
    files_transferred: number;
    bytes_transferred: number;
    errors?: string;
    trigger: string;
    edges?: {
        task?: {
            name: string;
        };
    };
}

export interface JobLog {
    id: number;
    job_id: string;
    level: string;
    message: string;
    path?: string;
    time: string;
}

import { API_BASE } from './config';

export const getJobs = async (limit = 10, offset = 0): Promise<Job[]> => {
    const response = await fetch(`${API_BASE}/jobs?limit=${limit}&offset=${offset}`);
    if (!response.ok) {
        throw new Error("Failed to fetch jobs");
    }
    return response.json();
};

export const getJob = async (id: string): Promise<Job & { edges: { logs: JobLog[] } }> => {
    const response = await fetch(`${API_BASE}/jobs/${id}`);
    if (!response.ok) {
        throw new Error("Failed to fetch job details");
    }
    return response.json();
};
