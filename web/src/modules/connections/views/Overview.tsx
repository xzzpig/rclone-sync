import { getConnectionQuota } from '@/api/connections';
import StatusIcon from '@/components/common/StatusIcon';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import { formatBytes } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useTasks } from '@/store/tasks';
import { useParams } from '@solidjs/router';
import { useQuery } from '@tanstack/solid-query';
import { Component, createMemo, Show } from 'solid-js';

const Overview: Component = () => {
  const params = useParams();
  const [, actions] = useTasks();

  const connectionId = () => params.connectionId;
  const quotaQuery = useQuery(() => ({
    queryKey: ['connectionQuota', connectionId()],
    queryFn: () => getConnectionQuota(connectionId()!),
    enabled: !!connectionId(),
    refetchInterval: 60000, // Refetch every 60 seconds
  }));

  // Use createMemo to ensure proper reactive tracking when connection changes
  const status = createMemo(() => actions.getTaskStatus(connectionId()));

  const statusLabel = () => {
    const s = status();
    if (s === 'running') return m.status_running();
    if (s === 'success') return m.overview_healthy();
    if (s === 'failed') return m.status_failed();
    return m.status_idle();
  };

  // Calculate usage percentage
  const usagePercent = () => {
    const q = quotaQuery.data;
    if (!q || !q.total || !q.used) return 0;
    return Math.min(100, (q.used / q.total) * 100);
  };

  return (
    <div class="h-full space-y-4 overflow-auto p-1">
      <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Status Card */}
        <Card>
          <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle class="text-sm font-medium">{m.overview_currentStatus()}</CardTitle>
            <StatusIcon status={status()} class="size-6" />
          </CardHeader>
          <CardContent>
            <div class="text-2xl font-bold tracking-tight">{statusLabel()}</div>
            <p class="mt-1 text-xs text-muted-foreground">
              {status() === 'running'
                ? m.overview_syncInProgress()
                : m.overview_lastCheckCompleted()}
            </p>
          </CardContent>
        </Card>

        {/* Quota Card */}
        {/* TODO: Display more detailed quota information */}
        <Card class="col-span-1 lg:col-span-3">
          <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle class="text-sm font-medium">{m.overview_storageUsage()}</CardTitle>
          </CardHeader>
          <CardContent>
            <Show
              when={!quotaQuery.isLoading}
              fallback={
                <div class="space-y-4">
                  <Skeleton class="h-8 w-[180px]" />
                  <div class="space-y-2">
                    <Skeleton class="h-2 w-full rounded-full" />
                    <Skeleton class="h-3 w-[100px]" />
                  </div>
                </div>
              }
            >
              <div class="flex items-end gap-2">
                <div class="text-2xl font-bold tracking-tight">
                  {formatBytes(quotaQuery.data?.used)}
                </div>
                <div class="mb-1 text-sm font-medium text-muted-foreground">
                  {m.overview_of()} {formatBytes(quotaQuery.data?.total)} {m.overview_used()}
                </div>
              </div>
              <Progress
                value={quotaQuery.data?.used ?? 0}
                minValue={0}
                maxValue={quotaQuery.data?.total ?? 100}
                class="mt-4"
              />
              <div class="mt-2 flex justify-between text-xs text-muted-foreground">
                <span>{usagePercent().toFixed(1)}%</span>
                <span>
                  {formatBytes(quotaQuery.data?.total)} {m.overview_total()}
                </span>
              </div>
            </Show>
          </CardContent>
        </Card>
      </div>
    </div>
  );
};

export default Overview;
