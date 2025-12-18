// Connection types for the new /connections API
export type LoadStatus = 'loaded' | 'loading' | 'error';

export interface Connection {
  id: string;
  name: string;
  type: string;
  load_status: LoadStatus;
  load_error?: string;
  created_at: string;
  updated_at: string;
}

export interface ConnectionConfig {
  type: string;
  [key: string]: string;
}

// Import types for rclone.conf import wizard
// 匹配后端 rclone.ParsedConnection
export interface ParsedConnection {
  name: string;
  type: string;
  config: Record<string, string>;
}

// 匹配后端 rclone.ValidationResult
export interface ValidationResult {
  valid: ParsedConnection[];
  conflicts: string[];
  internal_duplicates: string[];
}

// 匹配后端 handlers.ParseResponse
export interface ImportParseResult {
  connections: ParsedConnection[];
  validation?: ValidationResult;
}

// 匹配后端 handlers.ConnectionToImport
export interface ConnectionToImport {
  name: string;
  type: string;
  config: Record<string, string>;
}

// 匹配后端 handlers.ExecuteRequest
export interface ImportExecuteRequest {
  connections: ConnectionToImport[];
  overwrite: boolean;
}

// 匹配后端 handlers.ExecuteResponse
export interface ImportResult {
  imported: number;
  skipped: number;
  failed: number;
  errors?: string[];
}

// 前端本地状态：扩展 ParsedConnection 用于 UI 交互
export interface ImportPreviewItem extends ParsedConnection {
  // 前端本地状态
  selected: boolean;
  isConflict: boolean;
  isDuplicate: boolean;
  editedName?: string;
  editedConfig?: Record<string, string>;
}

// Legacy Remote type (for backward compatibility with existing code)
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
  connection_id: string;
  remote_path: string;
  direction: 'upload' | 'download' | 'bidirectional';
  schedule?: string;
  realtime: boolean;
  options: Record<string, unknown>;
  edges: {
    jobs?: Job[];
    connection?: Connection;
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
  connection_id: string;
  status: JobStatus;
  start_time: string;
  end_time?: string;
}
