import { CACHE_STATUS_SUBSCRIPTION } from '@/api/graphql/queries/subscriptions';
import type { CacheStatus } from '@/lib/types';
import { createSubscription } from '@urql/solid';
import { createContext, ParentComponent, useContext } from 'solid-js';

interface CacheStatusActions {
  useConnectionCacheStatus: (
    connectionId: () => string | undefined
  ) => () => CacheStatus | undefined;
}

const CacheStatusContext = createContext<CacheStatusActions>();

export const CacheStatusProvider: ParentComponent = (props) => {
  const useConnectionCacheStatus = (connectionId: () => string | undefined) => {
    const [subscriptionResult] = createSubscription({
      query: CACHE_STATUS_SUBSCRIPTION,
      get variables() {
        return { connectionId: connectionId() ?? '' };
      },
      get pause() {
        return !connectionId();
      },
    });

    return () => subscriptionResult.data?.cacheStatus as CacheStatus | undefined;
  };

  const actions: CacheStatusActions = {
    useConnectionCacheStatus,
  };

  return (
    <CacheStatusContext.Provider value={actions}>{props.children}</CacheStatusContext.Provider>
  );
};

export const useCacheStatus = () => {
  const context = useContext(CacheStatusContext);
  if (!context) {
    throw new Error('useCacheStatus must be used within CacheStatusProvider');
  }
  return context;
};
