/**
 * Centralized type definitions derived from GraphQL queries
 */
import type { graphql, ResultOf, VariablesOf } from '@/api/graphql/graphql';

// Import queries for type derivation
import type {
  ConnectionGetBasicQuery,
  ConnectionGetConfigQuery,
  ConnectionGetQuotaQuery,
  ConnectionsListQuery,
} from '@/api/graphql/queries/connections';
import type { FilesLocalQuery } from '@/api/graphql/queries/files';
import type { IMPORT_PARSE } from '@/api/graphql/queries/import';
import type { JobProgressQuery, JobsListQuery, LogsListQuery } from '@/api/graphql/queries/jobs';
import type { ProviderGetQuery, ProvidersListQuery } from '@/api/graphql/queries/providers';
import type { JOB_PROGRESS_SUBSCRIPTION } from '@/api/graphql/queries/subscriptions';
import type {
  TaskCreateMutation,
  TasksListQuery,
  TaskUpdateMutation,
} from '@/api/graphql/queries/tasks';

// ============================================================================
// GraphQL Derived Types from Queries
// ============================================================================

// Connection types (from ConnectionsListQuery and Connection Get queries)
export type ConnectionListItem = NonNullable<
  NonNullable<ResultOf<typeof ConnectionsListQuery>['connection']>['list']
>['items'][number];

// Connection basic type (id, name, type) - for display purposes
export type ConnectionBasic = NonNullable<
  NonNullable<ResultOf<typeof ConnectionGetBasicQuery>['connection']>['get']
>;

// Connection with config type - for settings/editing
export type ConnectionWithConfig = NonNullable<
  NonNullable<ResultOf<typeof ConnectionGetConfigQuery>['connection']>['get']
>;

// Connection with quota type - for overview/storage display
export type ConnectionWithQuota = NonNullable<
  NonNullable<ResultOf<typeof ConnectionGetQuotaQuery>['connection']>['get']
>;

// Backward compatibility alias - use ConnectionWithConfig for full config access
export type ConnectionDetail = ConnectionWithConfig;

export type LoadStatus = ReturnType<typeof graphql.scalar<'ConnectionLoadStatus'>>;

// Task types (from TasksListQuery)
export type TaskListItem = NonNullable<
  NonNullable<ResultOf<typeof TasksListQuery>['task']>['list']
>['items'][number];

export type SyncDirection = TaskListItem['direction'];

// Task mutation input types (from TaskCreateMutation and TaskUpdateMutation)
export type CreateTaskInput = VariablesOf<typeof TaskCreateMutation>['input'];
export type UpdateTaskInput = VariablesOf<typeof TaskUpdateMutation>['input'];

// ConflictResolution enum (from CreateTaskInput.options.conflictResolution)
export type ConflictResolution = NonNullable<
  NonNullable<CreateTaskInput['options']>['conflictResolution']
>;

// Job types (from JobsListQuery)
export type JobListItem = NonNullable<
  NonNullable<ResultOf<typeof JobsListQuery>['job']>['list']
>['items'][number];

export type JobStatus = ReturnType<typeof graphql.scalar<'JobStatus'>>;

// Job Log types (from LogsListQuery)
export type JobLogListItem = NonNullable<
  NonNullable<ResultOf<typeof LogsListQuery>['log']>['list']
>['items'][number];

export type LogLevel = JobLogListItem['level'];

// Status type for UI components (JobStatus + IDLE for idle state)
export type StatusType = JobStatus | 'IDLE';

// Provider types (from ProvidersListQuery and ProviderGetQuery)
export type ProviderListItem = NonNullable<
  NonNullable<ResultOf<typeof ProvidersListQuery>['provider']>['list']
>[number];

export type ProviderDetail = NonNullable<
  NonNullable<ResultOf<typeof ProviderGetQuery>['provider']>['get']
>;

export type ProviderOption = NonNullable<ProviderDetail['options']>[number];

// File types (from FilesLocalQuery)
export type FileEntry = NonNullable<
  NonNullable<ResultOf<typeof FilesLocalQuery>['file']>['local']
>[number];

// Import types (from IMPORT_PARSE mutation)
// Note: IMPORT_PARSE returns a union type, we extract from ImportParseSuccess
export type ParsedConnection = Extract<
  NonNullable<ResultOf<typeof IMPORT_PARSE>>['import']['parse'],
  { __typename: 'ImportParseSuccess' }
>['connections'][number];

// Job Progress types (from JobProgressQuery and JOB_PROGRESS_SUBSCRIPTION)
export type JobProgress = NonNullable<
  NonNullable<ResultOf<typeof JobProgressQuery>['job']>['progress']
>;

export type JobProgressEvent = NonNullable<
  ResultOf<typeof JOB_PROGRESS_SUBSCRIPTION>['jobProgress']
>;

// Quota types (from ConnectionGetQuotaQuery)
export type ConnectionQuota = NonNullable<ConnectionWithQuota['quota']>;

// ============================================================================
// Type Aliases for Backward Compatibility
// ============================================================================

// Shorter aliases commonly used in components
export type Connection = ConnectionListItem;
export type ConnectionFull = ConnectionDetail;
export type Task = TaskListItem;
export type TaskWithConnection = TaskListItem;
export type Job = JobListItem;
export type JobWithTask = JobListItem;
export type JobLog = JobLogListItem;
export type RcloneProvider = ProviderListItem;
export type RcloneOption = ProviderOption;
export type RemoteQuota = ConnectionQuota;

// ============================================================================
// Frontend Local Types (not from GraphQL)
// ============================================================================

/**
 * Extended ParsedConnection for import wizard UI state
 */
export interface ImportPreviewItem {
  // From ParsedConnection
  name: string;
  type: string;
  config: Record<string, string>;
  // Frontend local state
  selected: boolean;
  isConflict: boolean;
  isDuplicate: boolean;
  editedName?: string;
  editedConfig?: Record<string, string>;
}

/**
 * Import execution result
 */
export interface ImportResult {
  imported: number;
  skipped: number;
  failed: number;
  errors?: string[];
}

/**
 * Paginated response wrapper for REST API
 */
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
}

// ============================================================================
// Legacy Types (for backward compatibility with REST API)
// ============================================================================

/**
 * Connection config for REST API (legacy)
 */
export interface ConnectionConfig {
  type: string;
  [key: string]: string;
}

/**
 * Import parse result for REST API (legacy)
 */
export interface ImportParseResult {
  connections: ParsedConnection[];
  validation?: {
    valid: ParsedConnection[];
    conflicts: string[];
    internal_duplicates: string[];
  };
}

/**
 * Import execute request for REST API (legacy)
 */
export interface ImportExecuteRequest {
  connections: {
    name: string;
    type: string;
    config: Record<string, string>;
  }[];
  overwrite: boolean;
}

// ============================================================================
// Log Level Filter Constants
// ============================================================================

/**
 * Log level filter options for UI (lowercase for URL-friendly values)
 * Includes 'all' for showing all log levels
 */
export const LOG_LEVEL_FILTERS = ['all', 'info', 'warning', 'error'] as const;

/**
 * Log level filter type derived from LOG_LEVEL_FILTERS constant
 * Type: 'all' | 'info' | 'warning' | 'error'
 */
export type LogLevelFilter = (typeof LOG_LEVEL_FILTERS)[number];
