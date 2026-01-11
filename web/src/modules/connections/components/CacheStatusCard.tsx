import { Component, Show, createSignal } from 'solid-js';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showToast } from '@/components/ui/toast';
import { createMutation } from '@urql/solid';
import { ConnectionClearCacheMutation } from '@/api/graphql/queries/connections';
import * as m from '@/paraglide/messages.js';
import { cn, formatBytes } from '@/lib/utils';
import type { CacheStatus } from '@/lib/types';
import IconLoader2 from '~icons/lucide/loader-2';
import IconTrash2 from '~icons/lucide/trash-2';

interface CacheStatusCardProps {
  status?: CacheStatus | null;
  connectionId?: string;
  class?: string;
}

export const CacheStatusCard: Component<CacheStatusCardProps> = (props) => {
  const [isDialogOpen, setIsDialogOpen] = createSignal(false);
  const [isClearing, setIsClearing] = createSignal(false);
  const [, executeClearCache] = createMutation(ConnectionClearCacheMutation);

  const formatTime = (timeStr?: string | null) => {
    if (!timeStr) return m.time_never();
    return new Date(timeStr).toLocaleString();
  };

  const handleClearCache = async () => {
    if (!props.connectionId) return;

    setIsClearing(true);
    try {
      const result = await executeClearCache({ id: props.connectionId });

      if (result.error) {
        showToast({
          title: m.connection_cache_clearFailed(),
          description: result.error.message,
          variant: 'error',
        });
        return;
      }

      const clearResult = result.data?.connection?.clearCache;
      if (clearResult?.success) {
        showToast({
          title: m.connection_cache_clearSuccess(),
          description: m.connection_cache_clearSuccessDesc({
            count: String(clearResult.clearedCount ?? 0),
          }),
        });
      } else {
        showToast({
          title: m.connection_cache_clearFailed(),
          description: clearResult?.message ?? '',
          variant: 'error',
        });
      }
    } finally {
      setIsClearing(false);
      setIsDialogOpen(false);
    }
  };

  return (
    <Card class={cn('overflow-hidden', props.class)}>
      <CardHeader class="pb-2">
        <div class="flex items-center justify-between">
          <CardTitle class="text-xs font-medium text-muted-foreground">
            {m.connection_cache_status()}
          </CardTitle>
          <div class="flex items-center gap-2">
            <Show when={props.connectionId && props.status}>
              <Button
                variant="ghost"
                size="sm"
                class="h-5 px-1.5 text-[10px]"
                onClick={() => setIsDialogOpen(true)}
              >
                <IconTrash2 class="mr-1 size-3" />
                {m.connection_cache_clear()}
              </Button>
            </Show>
            <Show when={props.status}>
              <Badge
                variant={props.status?.running ? 'success' : 'secondary'}
                class="h-5 px-1.5 text-[10px]"
              >
                {props.status?.running ? m.status_running() : m.status_idle()}
              </Badge>
            </Show>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <Show when={props.status}>
          <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs sm:grid-cols-4">
            <div class="space-y-0.5">
              <p class="text-muted-foreground">{m.connection_cache_entries()}</p>
              <p class="font-bold">{props.status?.entriesCount ?? 0}</p>
            </div>
            <div class="space-y-0.5">
              <p class="text-muted-foreground">{m.connection_cache_dbSize()}</p>
              <p class="font-bold">{formatBytes(Number(props.status?.dbSizeBytes ?? 0))}</p>
            </div>
            <div class="space-y-0.5">
              <p class="text-muted-foreground">{m.connection_cache_notifySupported()}</p>
              <p class="font-bold">
                {props.status?.changeNotifySupported
                  ? m.connection_cache_supported()
                  : m.connection_cache_notSupported()}
              </p>
            </div>
            <div class="space-y-0.5">
              <p class="text-muted-foreground">{m.connection_cache_lastNotify()}</p>
              <p class="truncate font-bold">{formatTime(props.status?.lastNotifyTime)}</p>
            </div>
          </div>
        </Show>
      </CardContent>

      <Dialog open={isDialogOpen()} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{m.connection_cache_clearConfirmTitle()}</DialogTitle>
            <DialogDescription>{m.connection_cache_clearConfirmDesc()}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDialogOpen(false)}>
              {m.common_cancel()}
            </Button>
            <Button variant="destructive" onClick={handleClearCache} disabled={isClearing()}>
              {isClearing() ? <IconLoader2 class="mr-2 size-4 animate-spin" /> : null}
              {isClearing() ? m.connection_cache_clearing() : m.connection_cache_clear()}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
};
