import * as m from '@/paraglide/messages.js';
import { createTask, deleteTask, getTasks, runTask, updateTask } from '@/api/tasks';
import { extractErrorMessage } from '@/lib/api';
import { SSEClient } from '@/lib/sse';
import { JobProgressEvent, Task, TaskStatus } from '@/lib/types';
import { ParentComponent, createContext, onCleanup, useContext } from 'solid-js';
import { createStore, produce } from 'solid-js/store';

interface TaskState {
  tasks: Task[];
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
  startGlobalSseSubscription: () => void;
  getTaskStatus: (connectionId?: string) => TaskStatus;
  runTask: (id: string) => Promise<void>;
  createTask: (task: Omit<Task, 'id' | 'edges'>) => Promise<void>;
  updateTask: (id: string, updates: Partial<Task>) => Promise<void>;
  deleteTask: (id: string) => Promise<void>;
}

const TaskContext = createContext<[TaskState, TaskActions]>();

export const TaskProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<TaskState>(initialState);
  let sseClient: ReturnType<SSEClient['connect']> | null = null;

  const actions: TaskActions = {
    loadTasks: async (connectionId?: string) => {
      setState('isLoading', true);
      try {
        const newTasks = await getTasks(connectionId ? { connection_id: connectionId } : undefined);

        // Merge strategy: preserve SSE real-time updated jobs data
        setState('tasks', (oldTasks) => {
          // Create a map of id -> task to preserve old data
          const taskMap = new Map(oldTasks.map((t) => [t.id, t]));

          // Merge new tasks
          newTasks.forEach((newTask) => {
            const oldTask = taskMap.get(newTask.id);
            if (oldTask && oldTask.edges?.jobs && oldTask.edges.jobs.length > 0) {
              // If task exists with jobs data, merge and preserve old jobs
              taskMap.set(newTask.id, {
                ...newTask,
                edges: {
                  ...newTask.edges,
                  jobs: oldTask.edges.jobs,
                },
              });
            } else {
              // New task or no jobs data, use new data
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

    startGlobalSseSubscription: () => {
      if (sseClient) {
        sseClient.close();
      }

      // Use the global SSE endpoint
      const client = new SSEClient('/api/events?event=job_progress');
      sseClient = client.connect();

      sseClient.on('job_progress', (data: JobProgressEvent) => {
        if (!data || !data.task_id) return;

        setState(
          produce((s) => {
            const taskIndex = s.tasks.findIndex((t) => t.id === data.task_id);
            if (taskIndex !== -1) {
              const task = s.tasks[taskIndex];
              if (!task.edges) {
                task.edges = {};
              }
              task.edges.jobs ??= [];

              const jobIndex = task.edges.jobs.findIndex((j) => j.id === data.id);
              if (jobIndex !== -1) {
                // Update existing job
                task.edges.jobs[jobIndex] = { ...task.edges.jobs[jobIndex], ...data };
              } else {
                // Add new job
                task.edges.jobs.unshift(data);
              }
              s.tasks = [...s.tasks];
              console.info('Updated task from global SSE:', task);
            }
          })
        );
      });
    },

    getTaskStatus: (connectionId?: string): TaskStatus => {
      const relevantTasks = connectionId
        ? state.tasks.filter((t) => t.connection_id === connectionId)
        : state.tasks;

      if (relevantTasks.length === 0) return 'idle';

      const getStatus = (task: Task) => task.edges?.jobs?.[0]?.status;

      const isRunning = (s?: string) => s && ['running', 'processing', 'queued'].includes(s);
      const isFailed = (s?: string) => s && ['failed', 'error'].includes(s);
      const isSuccess = (s?: string) => s && ['success', 'finished', 'done'].includes(s);

      if (relevantTasks.some((t) => isRunning(getStatus(t)))) return 'running';
      if (relevantTasks.some((t) => isFailed(getStatus(t)))) return 'failed';
      if (relevantTasks.every((t) => isSuccess(getStatus(t)))) return 'success';

      return 'idle';
    },

    runTask: async (id: string) => {
      try {
        await runTask(id);
        // The SSE event will update the task status
      } catch (err) {
        console.error('Failed to run task:', err);
        throw err;
      }
    },

    createTask: async (task: Omit<Task, 'id' | 'edges'>) => {
      try {
        const newTask = await createTask(task);
        setState('tasks', (tasks) => [...tasks, newTask]);
      } catch (err: unknown) {
        console.error('Failed to create task:', err);
        // Use ApiError if available, otherwise extract from error
        const errorMessage = extractErrorMessage(err) ?? m.error_unknownError();
        throw new Error(errorMessage);
      }
    },

    updateTask: async (id: string, updates: Partial<Task>) => {
      try {
        const updatedTask = await updateTask(id, updates);
        setState('tasks', (tasks) => tasks.map((t) => (t.id === id ? updatedTask : t)));
      } catch (err: unknown) {
        console.error('Failed to update task:', err);
        // Use ApiError if available, otherwise extract from error
        const errorMessage = extractErrorMessage(err) ?? m.error_unknownError();
        throw new Error(errorMessage);
      }
    },

    deleteTask: async (id: string) => {
      try {
        await deleteTask(id);
        setState('tasks', (t) => t.filter((task) => task.id !== id));
      } catch (err) {
        console.error('Failed to delete task:', err);
      }
    },
  };

  onCleanup(() => {
    if (sseClient) {
      sseClient.close();
      sseClient = null;
    }
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
