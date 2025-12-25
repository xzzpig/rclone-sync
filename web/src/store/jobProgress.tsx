import { WS_RECONNECT_EVENT } from '@/api/graphql/client';
import { JOB_PROGRESS_SUBSCRIPTION } from '@/api/graphql/queries/subscriptions';
import type { JobProgressEvent } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createSubscription } from '@urql/solid';
import { createContext, createEffect, onCleanup, ParentComponent, useContext } from 'solid-js';

/**
 * Filter for job progress events
 */
export interface JobProgressFilter {
  connectionId?: string;
  taskId?: string;
}

/**
 * Callback function for job progress events
 */
export type JobProgressCallback = (event: JobProgressEvent) => void;

/**
 * Subscription handle returned by subscribe()
 */
interface Subscription {
  unsubscribe: () => void;
}

interface JobProgressActions {
  /**
   * Subscribe to job progress events with optional filter
   * Returns an unsubscribe function
   */
  subscribe: (callback: JobProgressCallback, filter?: JobProgressFilter) => Subscription;

  /**
   * Register a callback to be called on WebSocket reconnection
   * Returns an unsubscribe function
   */
  onReconnect: (callback: () => void) => () => void;
}

const JobProgressContext = createContext<JobProgressActions>();

export const JobProgressProvider: ParentComponent = (props) => {
  // Store all subscribers with their filters
  const subscribers = new Map<JobProgressCallback, JobProgressFilter | undefined>();

  // Store reconnect callbacks
  const reconnectCallbacks = new Set<() => void>();

  // Single global subscription - no filters, receives all events
  const [subscriptionResult] = createSubscription({
    query: JOB_PROGRESS_SUBSCRIPTION,
    variables: {},
  });

  // Handle subscription data updates - dispatch to all matching subscribers
  createEffect(() => {
    const data = subscriptionResult.data?.jobProgress as JobProgressEvent | undefined;
    if (!data) return;

    // Dispatch to all subscribers whose filter matches
    subscribers.forEach((filter, callback) => {
      // Check if event matches filter
      if (filter?.connectionId && data.connectionId !== filter.connectionId) {
        return;
      }
      if (filter?.taskId && data.taskId !== filter.taskId) {
        return;
      }

      // Event matches filter (or no filter) - call the callback
      try {
        callback(data);
      } catch (error) {
        console.error('Error in job progress callback:', error);
      }
    });
  });

  // Listen for WebSocket reconnection events
  const handleReconnect = () => {
    console.info('WebSocket reconnected, notifying all subscribers...');
    reconnectCallbacks.forEach((callback) => {
      try {
        callback();
      } catch (error) {
        console.error('Error in reconnect callback:', error);
      }
    });
  };

  window.addEventListener(WS_RECONNECT_EVENT, handleReconnect);

  // Cleanup on unmount
  onCleanup(() => {
    window.removeEventListener(WS_RECONNECT_EVENT, handleReconnect);
    subscribers.clear();
    reconnectCallbacks.clear();
  });

  const actions: JobProgressActions = {
    subscribe: (callback, filter) => {
      subscribers.set(callback, filter);
      return {
        unsubscribe: () => {
          subscribers.delete(callback);
        },
      };
    },

    onReconnect: (callback) => {
      reconnectCallbacks.add(callback);
      return () => {
        reconnectCallbacks.delete(callback);
      };
    },
  };

  return (
    <JobProgressContext.Provider value={actions}>{props.children}</JobProgressContext.Provider>
  );
};

export const useJobProgress = () => {
  const context = useContext(JobProgressContext);
  if (!context) {
    throw new Error(
      m.error_hookMissingProvider({ hook: 'useJobProgress', provider: 'JobProgressProvider' })
    );
  }
  return context;
};
