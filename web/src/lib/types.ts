export type Remote = {
  name: string;
  type: string;
  remote?: string;
};

export type RcloneProvider = {
  name: string;
  description: string;
  prefix: string;
};

export type RcloneOption = {
  Name: string;
  Help: string;
  Default: unknown;
  DefaultStr: string;
  Value: unknown;
  ValueStr: string;
  Type: string;
  Required: boolean;
  IsPassword?: boolean;
  Advanced?: boolean;
  Exclusive?: boolean;
  Examples: { Value: string; Help: string }[];
  Groups?: string;
};

export type RemoteQuota = {
  total?: number;
  used?: number;
  trashed?: number;
  other?: number;
  free?: number;
  objects?: number;
};

// Paginated response wrapper
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
}

export type JobStatus = 'pending' | 'running' | 'failed' | 'success' | 'canceled';

export interface Job {
  id: string;
  status: JobStatus;
  trigger?: string;
  start_time: string;
  end_time?: string;
  files_transferred: number;
  bytes_transferred: number;
  edges?: {
    task?: Task;
  };
}

export interface JobLog {
  id: string;
  level: 'info' | 'warning' | 'error';
  what: 'upload' | 'download' | 'delete' | 'move' | 'error' | 'unknown';
  path?: string;
  size?: number;
  time: string;
}

export type TaskStatus = 'running' | 'failed' | 'success' | 'idle';

export interface Task {
  id: string;
  name: string;
  source_path: string;
  remote_name: string;
  remote_path: string;
  direction: 'upload' | 'download' | 'bidirectional';
  schedule?: string;
  realtime: boolean;
  options: Record<string, unknown>;
  edges: {
    jobs?: Job[];
  };
}

export type SyncDirection = 'upload' | 'download' | 'bidirectional';

export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
}

export interface JobProgressEvent {
  id: string;
  job_id: string;
  task_id: string;
  files_transferred: number;
  bytes_transferred: number;
  remote_name: string;
  status: JobStatus;
  start_time: string;
  end_time?: string;
}
