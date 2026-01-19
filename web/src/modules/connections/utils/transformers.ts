import {
  type HookDetail,
  type HookInput,
  type UpdateTaskInput,
  type TaskDetail,
  type UpdateHookInput,
} from '@/lib/types';

export function hookToHookInput(hook: HookDetail | HookInput): HookInput {
  return {
    enabled: hook.enabled,
    priority: hook.priority,
    event: hook.event,
    type: hook.type,
    onError: hook.onError,
    config: {
      url: hook.config.url,
      method: hook.config.method,
      headers: hook.config.headers,
      body: hook.config.body,
      command: hook.config.command,
      workDir: hook.config.workDir,
      timeout: hook.config.timeout,
    },
  };
}

export function hookToUpdateHookInput(hook: HookDetail | UpdateHookInput): UpdateHookInput {
  return {
    enabled: hook.enabled,
    priority: hook.priority,
    event: hook.event,
    type: hook.type,
    onError: hook.onError,
    config: hook.config
      ? {
          url: hook.config.url,
          method: hook.config.method,
          headers: hook.config.headers,
          body: hook.config.body,
          command: hook.config.command,
          workDir: hook.config.workDir,
          timeout: hook.config.timeout,
        }
      : undefined,
  };
}

export function taskToUpdateTaskInput(task: TaskDetail): UpdateTaskInput {
  return {
    name: task.name,
    direction: task.direction,
    schedule: task.schedule,
    realtime: task.realtime,
    enabled: task.enabled ?? true,
    options: task.options
      ? {
          conflictResolution: task.options.conflictResolution,
          filters: (task.options.filters ?? []).filter((f): f is string => !!f),
          noDelete: task.options.noDelete,
          transfers: task.options.transfers,
        }
      : undefined,
    hooks: (task.hooks ?? []).map((h) => ({
      id: h.id,
      hook: hookToHookInput(h),
    })),
  };
}

export function toUpdateTaskInput(input: UpdateTaskInput): UpdateTaskInput {
  return {
    name: input.name,
    direction: input.direction,
    schedule: input.schedule,
    realtime: input.realtime,
    enabled: input.enabled,
    sourcePath: input.sourcePath,
    remotePath: input.remotePath,
    connectionID: input.connectionID,
    options: input.options
      ? {
          conflictResolution: input.options.conflictResolution,
          filters: (input.options.filters ?? []).filter((f): f is string => !!f),
          noDelete: input.options.noDelete,
          transfers: input.options.transfers,
        }
      : undefined,
    hooks: input.hooks?.map((h) => ({
      id: h.id,
      delete: h.delete,
      hook: h.hook ? hookToHookInput(h.hook) : undefined,
    })),
  };
}
