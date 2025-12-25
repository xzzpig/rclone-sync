import { client } from '@/api/graphql/client';
import {
  TaskCreateMutation,
  TaskDeleteMutation,
  TaskRunMutation,
  TaskUpdateMutation,
  TasksListQuery,
} from '@/api/graphql/queries/tasks';
import type {
  CreateTaskInput,
  JobProgressEvent,
  StatusType,
  TaskListItem,
  UpdateTaskInput,
} from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { ParentComponent, createContext, onCleanup, onMount, useContext } from 'solid-js';
import { createStore, produce } from 'solid-js/store';
import { useJobProgress } from './jobProgress';

interface TaskState {
  tasks: TaskListItem[];
  isLoading: boolean;
  error: unknown | null;
}

const initialState: TaskState = {
  tasks: [],
  isLoading: false,
  error: null,
};

interface TaskActions {
  loadTasks: (connectionId?: string) => Promise<void>;
  getTaskStatus: (connectionId?: string) => StatusType;
  runTask: (id: string) => Promise<void>;
  createTask: (input: CreateTaskInput) => Promise<void>;
  updateTask: (id: string, input: UpdateTaskInput) => Promise<void>;
  deleteTask: (id: string) => Promise<void>;
}

const TaskContext = createContext<[TaskState, TaskActions]>();

export const TaskProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<TaskState>(initialState);
  const jobProgress = useJobProgress();

  // Handle job progress events from centralized subscription
  const handleJobProgress = (data: JobProgressEvent) => {
    setState(
      produce((s) => {
        const taskIndex = s.tasks.findIndex((t) => t.id === data.taskId);
        if (taskIndex !== -1) {
          const task = s.tasks[taskIndex];
          // Update the task's latest job with subscription data
          task.latestJob = {
            id: data.jobId,
            status: data.status,
            startTime: data.startTime,
            endTime: data.endTime,
            filesTransferred: data.filesTransferred,
            bytesTransferred: data.bytesTransferred,
          };
          console.info('Updated task from GraphQL subscription:', task.id);
        }
      })
    );
  };

  const actions: TaskActions = {
    loadTasks: async (_connectionId?: string) => {
      setState('isLoading', true);
      try {
        const result = await client.query(
          TasksListQuery,
          { pagination: { limit: 1000, offset: 0 } },
          { requestPolicy: 'network-only' }
        );

        if (result.error) {
          throw new Error(result.error.message);
        }

        const items = result.data?.task?.list?.items ?? [];

        // Update tasks with new data, preserving subscription updates if more recent
        setState('tasks', (oldTasks) => {
          const taskMap = new Map(oldTasks.map((t) => [t.id, t]));

          items.forEach((newTask) => {
            const oldTask = taskMap.get(newTask.id);
            if (oldTask?.latestJob?.id === newTask.latestJob?.id && oldTask?.latestJob) {
              // Same job - preserve subscription data (more real-time)
              taskMap.set(newTask.id, {
                ...newTask,
                latestJob: oldTask.latestJob,
              });
            } else {
              // New job or no existing data - use query data
              taskMap.set(newTask.id, newTask);
            }
          });

          return Array.from(taskMap.values());
        });

        setState('error', null);
      } catch (err) {
        setState('error', err);
        console.error('Failed to fetch tasks:', err);
      } finally {
        setState('isLoading', false);
      }
    },

    getTaskStatus: (connectionId?: string): StatusType => {
      const relevantTasks = connectionId
        ? state.tasks.filter((t) => t.connection?.id === connectionId)
        : state.tasks;

      if (relevantTasks.length === 0) return 'IDLE';

      const getStatus = (task: TaskListItem) => task.latestJob?.status;

      // GraphQL returns uppercase enum values: PENDING, RUNNING, SUCCESS, FAILED, CANCELLED
      const isRunning = (s?: string) => s && ['RUNNING', 'PENDING'].includes(s);
      const isFailed = (s?: string) => s && ['FAILED'].includes(s);
      const isSuccess = (s?: string) => s && ['SUCCESS'].includes(s);

      if (relevantTasks.some((t) => isRunning(getStatus(t)))) return 'RUNNING';
      if (relevantTasks.some((t) => isFailed(getStatus(t)))) return 'FAILED';
      if (relevantTasks.every((t) => isSuccess(getStatus(t)))) return 'SUCCESS';

      return 'IDLE';
    },

    runTask: async (id: string) => {
      try {
        const result = await client.mutation(TaskRunMutation, { taskId: id });
        if (result.error) {
          throw new Error(result.error.message);
        }

        // Update local state with the new job
        const job = result.data?.task?.run;
        if (job) {
          setState(
            produce((s) => {
              const taskIndex = s.tasks.findIndex((t) => t.id === id);
              if (taskIndex !== -1) {
                s.tasks[taskIndex].latestJob = {
                  id: job.id,
                  status: job.status,
                  startTime: job.startTime,
                  endTime: null,
                  filesTransferred: 0,
                  bytesTransferred: 0,
                };
              }
            })
          );
        }
      } catch (err) {
        console.error('Failed to run task:', err);
        throw err;
      }
    },

    createTask: async (input) => {
      try {
        const result = await client.mutation(TaskCreateMutation, { input });

        if (result.error) {
          throw new Error(result.error.message);
        }

        // Reload tasks to get the full task data with connection
        await actions.loadTasks();
      } catch (err: unknown) {
        console.error('Failed to create task:', err);
        const errorMessage = err instanceof Error ? err.message : m.error_unknownError();
        throw new Error(errorMessage);
      }
    },

    updateTask: async (id: string, input) => {
      try {
        const result = await client.mutation(TaskUpdateMutation, { id, input });

        if (result.error) {
          throw new Error(result.error.message);
        }

        // Reload tasks to get the updated data with connection
        await actions.loadTasks();
      } catch (err: unknown) {
        console.error('Failed to update task:', err);
        const errorMessage = err instanceof Error ? err.message : m.error_unknownError();
        throw new Error(errorMessage);
      }
    },

    deleteTask: async (id: string) => {
      try {
        const result = await client.mutation(TaskDeleteMutation, { id });
        if (result.error) {
          throw new Error(result.error.message);
        }
        setState('tasks', (t) => t.filter((task) => task.id !== id));
      } catch (err) {
        console.error('Failed to delete task:', err);
      }
    },
  };

  // Subscribe to job progress events (no filter - all events)
  onMount(() => {
    const subscription = jobProgress.subscribe(handleJobProgress);
    const unsubscribeReconnect = jobProgress.onReconnect(() => {
      console.info('WebSocket reconnected, reloading tasks...');
      // Clear old tasks first to avoid stale subscription data being preserved
      setState('tasks', []);
      actions.loadTasks();
    });

    onCleanup(() => {
      subscription.unsubscribe();
      unsubscribeReconnect();
    });
  });

  return <TaskContext.Provider value={[state, actions]}>{props.children}</TaskContext.Provider>;
};

export const useTasks = () => {
  const context = useContext(TaskContext);
  if (!context) {
    throw new Error(m.error_hookMissingProvider({ hook: 'useTasks', provider: 'TaskProvider' }));
  }
  return context;
};
