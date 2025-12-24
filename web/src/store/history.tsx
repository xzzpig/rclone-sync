import { client } from '@/api/graphql/client';
import { JobsListQuery, LogsListQuery } from '@/api/graphql/queries/jobs';
import { JOB_PROGRESS_SUBSCRIPTION } from '@/api/graphql/queries/subscriptions';
import type { JobListItem, JobLogListItem, JobProgressEvent, LogLevel } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createSubscription } from '@urql/solid';
import { createContext, createEffect, ParentComponent, useContext } from 'solid-js';
import { createStore, produce } from 'solid-js/store';

interface HistoryState {
  jobs: JobListItem[];
  jobsTotal: number;
  jobsPage: number;
  jobsPageSize: number;
  logs: JobLogListItem[];
  logsTotal: number;
  logsPage: number;
  logsPageSize: number;
  isLoadingJobs: boolean;
  isLoadingLogs: boolean;
  errorJobs: unknown | null;
  errorLogs: unknown | null;
  // Tracking current filter parameters for subscription
  currentConnectionId: string | null;
  currentTaskId: string | null;
}

const initialState: HistoryState = {
  jobs: [],
  jobsTotal: 0,
  jobsPage: 1,
  jobsPageSize: 10,
  logs: [],
  logsTotal: 0,
  logsPage: 1,
  logsPageSize: 50,
  isLoadingJobs: false,
  isLoadingLogs: false,
  errorJobs: null,
  errorLogs: null,
  currentConnectionId: null,
  currentTaskId: null,
};

interface HistoryActions {
  loadJobs: (params: { connection_id: string; task_id?: string; page?: number }) => Promise<void>;
  loadLogs: (params: {
    connection_id: string;
    task_id?: string;
    job_id?: string;
    level?: LogLevel;
    page?: number;
  }) => Promise<void>;
  clearLogs: () => void;
}

const HistoryContext = createContext<[HistoryState, HistoryActions]>();

export const HistoryProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<HistoryState>(initialState);

  // GraphQL subscription for job progress using @urql/solid
  const [subscriptionResult] = createSubscription({
    query: JOB_PROGRESS_SUBSCRIPTION,
    variables: () => ({
      connectionId: state.currentConnectionId ?? undefined,
      taskId: state.currentTaskId ?? undefined,
    }),
    pause: () => !state.currentConnectionId,
  });

  // Handle subscription data updates
  createEffect(() => {
    const data = subscriptionResult.data?.jobProgress as JobProgressEvent | undefined;
    if (!data) return;

    // Check if the subscription data matches our current filter
    if (state.currentConnectionId && data.connectionId !== state.currentConnectionId) {
      return;
    }
    if (state.currentTaskId && data.taskId !== state.currentTaskId) {
      return;
    }

    setState(
      produce((s) => {
        const jobIndex = s.jobs.findIndex((j) => j.id === data.jobId);
        if (jobIndex !== -1) {
          // Update existing job with subscription data
          const job = s.jobs[jobIndex];
          job.status = data.status;
          job.filesTransferred = data.filesTransferred;
          job.bytesTransferred = data.bytesTransferred;
          if (data.endTime) {
            job.endTime = data.endTime;
          }
          console.info('Updated job from GraphQL subscription:', data.jobId);
        } else if (s.jobsPage === 1) {
          // New job detected on first page - reload to get full job data with task info
          // We use a timeout to debounce rapid updates
          console.info('New job detected, will reload jobs list');
          // Don't reload here directly to avoid infinite loops
          // The UI should handle this via polling or user refresh
        }
      })
    );
  });

  const actions: HistoryActions = {
    loadJobs: async (params) => {
      setState('isLoadingJobs', true);
      // Update current filter state for subscription
      setState('currentConnectionId', params.connection_id);
      setState('currentTaskId', params.task_id ?? null);

      try {
        const page = params.page ?? state.jobsPage;
        const offset = (page - 1) * state.jobsPageSize;

        const result = await client.query(
          JobsListQuery,
          {
            taskId: params.task_id,
            connectionId: params.connection_id,
            pagination: { limit: state.jobsPageSize, offset },
          },
          { requestPolicy: 'network-only' }
        );

        if (result.error) {
          throw new Error(result.error.message);
        }

        const listData = result.data?.job?.list;
        setState('jobs', listData?.items ?? []);
        setState('jobsTotal', listData?.totalCount ?? 0);
        setState('jobsPage', page);
        setState('errorJobs', null);
      } catch (err) {
        setState('errorJobs', err);
        console.error('Failed to fetch jobs:', err);
      } finally {
        setState('isLoadingJobs', false);
      }
    },

    loadLogs: async (params) => {
      setState('isLoadingLogs', true);
      try {
        const page = params.page ?? state.logsPage;
        const offset = (page - 1) * state.logsPageSize;

        const result = await client.query(
          LogsListQuery,
          {
            connectionId: params.connection_id,
            taskId: params.task_id,
            jobId: params.job_id,
            level: params.level,
            pagination: { limit: state.logsPageSize, offset },
          },
          { requestPolicy: 'network-only' }
        );

        if (result.error) {
          throw new Error(result.error.message);
        }

        const listData = result.data?.log?.list;
        setState('logs', listData?.items ?? []);
        setState('logsTotal', listData?.totalCount ?? 0);
        setState('logsPage', page);
        setState('errorLogs', null);
      } catch (err) {
        setState('errorLogs', err);
        console.error('Failed to fetch logs:', err);
      } finally {
        setState('isLoadingLogs', false);
      }
    },

    clearLogs: () => {
      setState('logs', []);
      setState('logsTotal', 0);
      setState('logsPage', 1);
      setState('errorLogs', null);
    },
  };

  return (
    <HistoryContext.Provider value={[state, actions]}>{props.children}</HistoryContext.Provider>
  );
};

export const useHistory = () => {
  const context = useContext(HistoryContext);
  if (!context) {
    throw new Error(
      m.error_hookMissingProvider({ hook: 'useHistory', provider: 'HistoryProvider' })
    );
  }
  return context;
};
