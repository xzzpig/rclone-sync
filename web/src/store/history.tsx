import * as m from '@/paraglide/messages.js';
import { createStore } from 'solid-js/store';
import { createContext, useContext, ParentComponent } from 'solid-js';
import { getJobs } from '@/api/history';
import { getLogs } from '@/api/logs';
import { Job, JobLog } from '@/lib/types';

interface HistoryState {
  jobs: Job[];
  jobsTotal: number;
  jobsPage: number;
  jobsPageSize: number;
  logs: JobLog[];
  logsTotal: number;
  logsPage: number;
  logsPageSize: number;
  isLoadingJobs: boolean;
  isLoadingLogs: boolean;
  errorJobs: unknown | null;
  errorLogs: unknown | null;
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
};

interface HistoryActions {
  loadJobs: (params: { remote_name: string; task_id?: string; page?: number }) => Promise<void>;
  loadLogs: (params: {
    remote_name: string;
    task_id?: string;
    job_id?: string;
    level?: string;
    page?: number;
  }) => Promise<void>;
  clearLogs: () => void;
}

const HistoryContext = createContext<[HistoryState, HistoryActions]>();

export const HistoryProvider: ParentComponent = (props) => {
  const [state, setState] = createStore<HistoryState>(initialState);

  const actions: HistoryActions = {
    loadJobs: async (params) => {
      setState('isLoadingJobs', true);
      try {
        const page = params.page ?? state.jobsPage;
        const offset = (page - 1) * state.jobsPageSize;
        const response = await getJobs({
          ...params,
          limit: state.jobsPageSize,
          offset,
        });
        setState('jobs', response.data);
        setState('jobsTotal', response.total);
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
        const response = await getLogs({
          ...params,
          limit: state.logsPageSize,
          offset,
        });
        setState('logs', response.data);
        setState('logsTotal', response.total);
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
