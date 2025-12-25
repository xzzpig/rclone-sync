import { TRANSFER_PROGRESS_SUBSCRIPTION } from '@/api/graphql/queries/subscriptions';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import type { TransferItem } from '@/lib/types';
import { formatBytes } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { createSubscription } from '@urql/solid';
import { Accessor, Component, createEffect, createMemo, createSignal, For, Show } from 'solid-js';
import IconLoader from '~icons/lucide/loader';

interface ActiveTransfersCardProps {
  connectionId: Accessor<string | undefined>;
}

const ActiveTransfersCard: Component<ActiveTransfersCardProps> = (props) => {
  // Subscribe to transfer progress events for this connection
  const [transferResult] = createSubscription({
    query: TRANSFER_PROGRESS_SUBSCRIPTION,
    variables: () => ({ connectionId: props.connectionId() }),
    pause: () => !props.connectionId(),
  });

  // Maintain local state for active transfers (incremental updates)
  const [transfersMap, setTransfersMap] = createSignal<Map<string, TransferItem>>(new Map());

  // Process incremental updates from subscription
  createEffect(() => {
    const event = transferResult.data?.transferProgress;
    if (!event?.transfers) return;

    setTransfersMap((prev) => {
      const newMap = new Map(prev);
      for (const transfer of event.transfers) {
        const size = transfer.size;
        const bytes = transfer.bytes;
        if (bytes >= size && size > 0) {
          // Transfer completed (bytes == size), remove it
          newMap.delete(transfer.name);
        } else {
          // Update or add transfer
          newMap.set(transfer.name, transfer);
        }
      }
      return newMap;
    });
  });

  // Reset transfers when connection changes
  createEffect(() => {
    props.connectionId();
    setTransfersMap(new Map());
  });

  // Convert map to array for rendering
  const activeTransfers = createMemo(() => Array.from(transfersMap().values()));

  return (
    <Card>
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">{m.overview_activeTransfers()}</CardTitle>
        <IconLoader class="size-4 animate-spin text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <Show
          when={activeTransfers().length > 0}
          fallback={
            <div class="flex flex-col items-center justify-center py-4 text-muted-foreground">
              <p class="text-sm">{m.overview_noActiveTransfers()}</p>
            </div>
          }
        >
          <div class="space-y-3">
            <For each={activeTransfers()}>
              {(transfer) => {
                // Compute percent dynamically to ensure reactivity
                const percent = () => {
                  return transfer.size > 0
                    ? Math.min(100, (transfer.bytes / transfer.size) * 100)
                    : 0;
                };
                return (
                  <div class="space-y-1">
                    <div class="flex items-center justify-between text-xs">
                      <span class="truncate font-medium" title={transfer.name}>
                        {transfer.name}
                      </span>
                      <span class="ml-2 shrink-0 text-muted-foreground">
                        {formatBytes(transfer.bytes)} / {formatBytes(transfer.size)}
                      </span>
                    </div>
                    <Progress value={percent()} minValue={0} maxValue={100} class="h-1.5" />
                  </div>
                );
              }}
            </For>
          </div>
        </Show>
      </CardContent>
    </Card>
  );
};

export default ActiveTransfersCard;
