export interface Task {
    id: string;
    name: string;
    source_path: string;
    remote_name: string;
    remote_path: string;
    direction: 'upload' | 'download' | 'bidirectional';
    schedule?: string;
    realtime: boolean;
    options?: Record<string, any>;
    created_at: string;
    updated_at: string;
}

export type CreateTaskRequest = Omit<Task, 'id' | 'created_at' | 'updated_at'>;

import { API_BASE } from './config';

export const getTasks = async (): Promise<Task[]> => {
    const response = await fetch(`${API_BASE}/tasks`);
    if (!response.ok) {
        throw new Error("Failed to fetch tasks");
    }
    return response.json();
};

export const createTask = async (task: CreateTaskRequest): Promise<Task> => {
    const response = await fetch(`${API_BASE}/tasks`, {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(task),
    });
    if (!response.ok) {
        throw new Error("Failed to create task");
    }
    return response.json();
};

export const getTask = async (id: string): Promise<Task> => {
    const response = await fetch(`${API_BASE}/tasks/${id}`);
    if (!response.ok) {
        throw new Error("Failed to fetch task");
    }
    return response.json();
};

export const updateTask = async (id: string, task: CreateTaskRequest): Promise<Task> => {
    const response = await fetch(`${API_BASE}/tasks/${id}`, {
        method: "PUT",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify(task),
    });
    if (!response.ok) {
        throw new Error("Failed to update task");
    }
    return response.json();
};

export const deleteTask = async (id: string): Promise<void> => {
    const response = await fetch(`${API_BASE}/tasks/${id}`, {
        method: "DELETE",
    });
    if (!response.ok) {
        throw new Error("Failed to delete task");
    }
};

export const runTask = async (id: string): Promise<void> => {
    const response = await fetch(`${API_BASE}/tasks/${id}/run`, {
        method: "POST",
    });
    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || "Failed to run task");
    }
};
