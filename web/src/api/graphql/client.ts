import { Client, fetchExchange, subscriptionExchange } from '@urql/core';
import { cacheExchange, type Cache } from '@urql/exchange-graphcache';
import { persistedExchange } from '@urql/exchange-persisted';
import { createClient as createWSClient } from 'graphql-ws';
import { onCleanup } from 'solid-js';
import { getLocale } from '@/paraglide/runtime';

// Event name for WebSocket reconnection
export const WS_RECONNECT_EVENT = 'ws-reconnect';

// WebSocket client for subscriptions
const wsClient = createWSClient({
  url: `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/graphql`,
  on: {
    connected: (_socket, _payload, wasRetry) => {
      // wasRetry is true when this is a reconnection
      if (wasRetry) {
        console.info('WebSocket reconnected, dispatching event');
        window.dispatchEvent(new CustomEvent(WS_RECONNECT_EVENT));
      }
    },
  },
});

// Helper to invalidate list queries after create/delete mutations
function invalidateListQuery(cache: Cache, typename: 'Task' | 'Connection' | 'Job') {
  const queryMap = {
    Task: 'TaskQuery',
    Connection: 'ConnectionQuery',
    Job: 'JobQuery',
  };
  const queryType = queryMap[typename];

  // Invalidate the namespace query fields on Query root
  cache.inspectFields('Query').forEach((field) => {
    if (
      field.fieldName === typename.toLowerCase() ||
      field.fieldName === 'task' ||
      field.fieldName === 'connection' ||
      field.fieldName === 'job'
    ) {
      cache.invalidate('Query', field.fieldName, field.arguments ?? undefined);
    }
  });

  // Also invalidate namespace query list field
  cache.inspectFields(queryType).forEach((field) => {
    if (field.fieldName === 'list') {
      cache.invalidate(queryType, field.fieldName, field.arguments ?? undefined);
    }
  });
}

// Create the urql client
export const client = new Client({
  url: '/api/graphql',
  // Add Accept-Language header for i18n support (FR-013)
  fetchOptions: () => ({
    headers: {
      'Accept-Language': getLocale(),
    },
  }),
  exchanges: [
    cacheExchange({
      keys: {
        // Pagination & Connections - no cache key needed
        TaskConnection: () => null,
        ConnectionConnection: () => null,
        JobConnection: () => null,
        JobLogConnection: () => null,
        OffsetPageInfo: () => null,

        // Namespaces (Singletons) - no cache key needed
        TaskQuery: () => null,
        ConnectionQuery: () => null,
        JobQuery: () => null,
        LogQuery: () => null,
        ProviderQuery: () => null,
        FileQuery: () => null,
        TaskMutation: () => null,
        ConnectionMutation: () => null,
        ImportMutation: () => null,

        // Value Objects / Results - no cache key needed
        TaskSyncOptions: () => null,
        ConnectionQuota: () => null,
        ConnectionTestSuccess: () => null,
        ConnectionTestFailure: () => null,
        ImportParseSuccess: () => null,
        ImportParseError: () => null,
        ImportExecuteResult: () => null,
        ParsedConnection: () => null,
        FileEntry: () => null,
        ProviderOption: () => null,
        OptionExample: () => null,
        JobProgressEvent: () => null,

        // Custom Keys - entities with specific key fields
        Provider: (data) => (data as unknown as { name: string }).name,
      },
      resolvers: {
        TaskQuery: {
          get: (_, args) => ({ __typename: 'Task', id: args.id }),
        },
        ConnectionQuery: {
          get: (_, args) => ({ __typename: 'Connection', id: args.id }),
        },
        JobQuery: {
          get: (_, args) => ({ __typename: 'Job', id: args.id }),
        },
        ProviderQuery: {
          get: (_, args) => ({ __typename: 'Provider', name: args.name }),
        },
      },
      // Cache updates for mutations - invalidate list queries after create/delete
      // Note: Since we use namespace pattern (mutation { task { create } }), the updates need
      // to be configured on the namespace types, not the root Mutation type
      updates: {
        Mutation: {
          // Root-level Mutation updates if any non-namespaced mutations exist
        },
        TaskMutation: {
          create: (_result, _args, cache) => {
            invalidateListQuery(cache, 'Task');
          },
          delete: (_result, _args, cache) => {
            invalidateListQuery(cache, 'Task');
          },
        },
        ConnectionMutation: {
          create: (_result, _args, cache) => {
            invalidateListQuery(cache, 'Connection');
          },
          delete: (_result, _args, cache) => {
            invalidateListQuery(cache, 'Connection');
          },
        },
      },
      // Note: Optimistic updates are not easily supported for namespaced mutations
      // (mutation { task { delete(id) } }) in graphcache. The cache invalidation
      // in updates above handles refetching affected queries after mutations complete.
    }),
    ...(window.isSecureContext
      ? [
          persistedExchange({
            preferGetForPersistedQueries: true,
            enableForMutation: true,
          }),
        ]
      : []),
    fetchExchange,
    subscriptionExchange({
      forwardSubscription: (request) => ({
        subscribe: (sink) => ({
          unsubscribe: wsClient.subscribe(
            {
              query: request.query as string,
              variables: request.variables as Record<string, unknown>,
            },
            sink
          ),
        }),
      }),
    }),
  ],
});

/**
 * Register a callback to be invoked when WebSocket reconnects.
 * Automatically cleans up when the owner scope is disposed.
 * Must be called within a reactive context (component, createRoot, etc.)
 *
 * @example
 * ```tsx
 * // In a component or provider
 * onWsReconnect(() => {
 *   console.info('WebSocket reconnected, reloading data...');
 *   refetchData();
 * });
 * ```
 */
export function onWsReconnect(callback: () => void): void {
  window.addEventListener(WS_RECONNECT_EVENT, callback);
  onCleanup(() => {
    window.removeEventListener(WS_RECONNECT_EVENT, callback);
  });
}

export default client;
