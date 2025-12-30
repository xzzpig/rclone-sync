import { ConnectionGetQuotaQuery } from '@/api/graphql/queries/connections';
import ActiveTransfersCard from '@/modules/connections/components/ActiveTransfersCard';
import RunningJobsCard from '@/modules/connections/components/RunningJobsCard';
import StatusIcon from '@/components/common/StatusIcon';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Skeleton } from '@/components/ui/skeleton';
import { formatBytes } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useTasks } from '@/store/tasks';
import { useParams } from '@solidjs/router';
import { createQuery } from '@urql/solid';
import { Component, createMemo, Show } from 'solid-js';
import IconFile from '~icons/lucide/file';
import IconHardDrive from '~icons/lucide/hard-drive';
import IconPieChart from '~icons/lucide/pie-chart';
import IconTrash2 from '~icons/lucide/trash-2';

const Overview: Component = () => {
  const params = useParams();
  const [, actions] = useTasks();

  const connectionId = () => params.connectionId;

  // Use GraphQL query to fetch connection with quota
  const [connectionResult] = createQuery({
    query: ConnectionGetQuotaQuery,
    variables: () => ({ id: connectionId()! }),
    pause: () => !connectionId(),
  });

  // Extract quota from GraphQL response
  const quota = () => connectionResult.data?.connection?.get?.quota;

  // Use createMemo to ensure proper reactive tracking when connection changes
  const status = createMemo(() => actions.getTaskStatus(connectionId()));

  const statusLabel = () => {
    const s = status();
    if (s === 'RUNNING') return m.status_running();
    if (s === 'SUCCESS') return m.overview_healthy();
    if (s === 'FAILED') return m.status_failed();
    return m.status_idle();
  };

  // Calculate usage percentage
  const usagePercent = () => {
    const q = quota();
    if (!q || !q.total || !q.used) return 0;
    return Math.min(100, (Number(q.used) / Number(q.total)) * 100);
  };

  // Check if quota is available
  const hasQuota = () => {
    const q = quota();
    return q && (q.total != null || q.used != null);
  };

  // Format nullable number to bytes or return placeholder
  const formatQuotaValue = (value: string | number | null | undefined): string => {
    if (value == null) return '-';
    return formatBytes(Number(value));
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
              {status() === 'RUNNING'
                ? m.overview_syncInProgress()
                : m.overview_lastCheckCompleted()}
            </p>
          </CardContent>
        </Card>

        {/* Quota Card - Compact Scheme A */}
        <Card class="col-span-1 lg:col-span-3">
          <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle class="text-sm font-medium">{m.overview_storageUsage()}</CardTitle>
          </CardHeader>
          <CardContent>
            <Show
              when={!connectionResult.fetching}
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
              <Show
                when={hasQuota()}
                fallback={
                  <div class="flex flex-col items-center justify-center py-4 text-muted-foreground">
                    <p class="text-sm">{m.overview_quotaUnavailable()}</p>
                  </div>
                }
              >
                <div class="space-y-3">
                  {/* Main Usage & Percentage */}
                  <div class="flex items-end justify-between">
                    <div class="flex items-baseline gap-1">
                      <span class="text-3xl font-bold tracking-tight">
                        {formatQuotaValue(quota()?.used)}
                      </span>
                      <span class="text-sm font-medium text-muted-foreground">
                        {m.overview_used()}
                      </span>
                    </div>
                    <span class="text-sm font-medium text-muted-foreground">
                      {usagePercent().toFixed(1)}%
                    </span>
                  </div>

                  <Progress
                    value={Number(quota()?.used) || 0}
                    minValue={0}
                    maxValue={Number(quota()?.total) || 100}
                    class="h-2.5"
                  />

                  {/* Footer Stats Row - Compact */}
                  <div class="flex flex-wrap items-center gap-x-6 gap-y-2 pt-1 text-xs text-muted-foreground">
                    {/* Total */}
                    <Show when={quota()?.total != null}>
                      <div class="flex items-center gap-1.5">
                        <IconHardDrive class="size-3.5" />
                        <span>{m.overview_total()}:</span>
                        <span class="font-medium text-foreground">
                          {formatQuotaValue(quota()?.total)}
                        </span>
                      </div>
                    </Show>

                    {/* Trashed */}
                    <Show when={quota()?.trashed != null}>
                      <div class="flex items-center gap-1.5">
                        <IconTrash2 class="size-3.5" />
                        <span>{m.overview_trashed()}:</span>
                        <span class="font-medium text-foreground">
                          {formatQuotaValue(quota()?.trashed)}
                        </span>
                      </div>
                    </Show>

                    {/* Other */}
                    <Show when={quota()?.other != null}>
                      <div class="flex items-center gap-1.5">
                        <IconPieChart class="size-3.5" />
                        <span>{m.overview_other()}:</span>
                        <span class="font-medium text-foreground">
                          {formatQuotaValue(quota()?.other)}
                        </span>
                      </div>
                    </Show>

                    {/* Objects */}
                    <Show when={quota()?.objects != null}>
                      <div class="flex items-center gap-1.5">
                        <IconFile class="size-3.5" />
                        <span>{m.overview_objects()}:</span>
                        <span class="font-medium text-foreground">
                          {Number(quota()?.objects).toLocaleString()}
                        </span>
                      </div>
                    </Show>
                  </div>
                </div>
              </Show>
            </Show>
          </CardContent>
        </Card>
      </div>

      {/* Running Jobs Card - auto-hides when no running jobs */}
      <RunningJobsCard connectionId={connectionId} />

      {/* Active Transfers Card - Only show when running */}
      <Show when={status() === 'RUNNING'}>
        <ActiveTransfersCard connectionId={connectionId} />
      </Show>
    </div>
  );
};

export default Overview;
