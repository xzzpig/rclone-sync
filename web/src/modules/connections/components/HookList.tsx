import TableSkeleton from '@/components/common/TableSkeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch, SwitchControl, SwitchThumb } from '@/components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import * as m from '@/paraglide/messages.js';
import { createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import IconEdit from '~icons/lucide/edit';
import IconTrash2 from '~icons/lucide/trash-2';

import { type HookListItem } from '@/lib/types';
import { getEventLabel, getOnErrorLabel, getTypeLabel } from '../utils/hooks';

interface HookListProps {
  hooks: HookListItem[];
  fetching?: boolean;
  onEdit?: (hook: HookListItem) => void;
  onEnabledChange?: (id: string, enabled: boolean) => void;
  onDelete?: (id: string) => void;
}

function HookListSkeleton(props: { isCompact?: boolean }) {
  return (
    <div class="space-y-4">
      <Show when={props.isCompact}>
        <div class="grid grid-cols-1 gap-4">
          <For each={Array(3)}>
            {() => (
              <Card>
                <CardContent class="p-4">
                  <div class="mb-3 flex items-center justify-between">
                    <div class="flex flex-col gap-2">
                      <Skeleton class="h-3 w-16" />
                      <Skeleton class="h-5 w-24" />
                    </div>
                    <Skeleton class="h-6 w-10 rounded-full" />
                  </div>
                  <div class="space-y-2">
                    <Skeleton class="h-10 w-full rounded" />
                    <div class="flex items-center justify-between">
                      <Skeleton class="h-3 w-32" />
                      <div class="flex gap-2">
                        <Skeleton class="size-8 rounded" />
                        <Skeleton class="size-8 rounded" />
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}
          </For>
        </div>
      </Show>

      <Show when={!props.isCompact}>
        <div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead class="whitespace-nowrap">{m.hook_event()}</TableHead>
                <TableHead class="w-full">{m.hook_type()}</TableHead>
                <TableHead class="whitespace-nowrap text-center">{m.hook_priority()}</TableHead>
                <TableHead class="whitespace-nowrap">{m.hook_onError()}</TableHead>
                <TableHead class="whitespace-nowrap text-center">{m.common_status()}</TableHead>
                <TableHead class="whitespace-nowrap text-right">{m.common_actions()}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableSkeleton columns={6} rows={3} />
            </TableBody>
          </Table>
        </div>
      </Show>
    </div>
  );
}

export function HookList(props: HookListProps) {
  let containerRef: HTMLDivElement | undefined;
  const [isCompact, setIsCompact] = createSignal(false);
  const [confirmDeleteId, setConfirmDeleteId] = createSignal<string | null>(null);

  onMount(() => {
    if (!containerRef) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setIsCompact(entry.contentRect.width < 640);
      }
    });
    observer.observe(containerRef);
    onCleanup(() => observer.disconnect());
  });

  const handleToggleEnabled = (id: string, enabled: boolean) => {
    props.onEnabledChange?.(id, enabled);
  };

  const handleDelete = (id: string) => {
    props.onDelete?.(id);
  };

  const hooks = () => props.hooks;

  return (
    <div ref={containerRef} class="space-y-4">
      <Show when={!props.fetching} fallback={<HookListSkeleton isCompact={isCompact()} />}>
        <Show
          when={hooks().length > 0}
          fallback={
            <div class="rounded-md border border-dashed p-8 text-center">
              <p class="text-sm text-muted-foreground">{m.hook_noHooks()}</p>
            </div>
          }
        >
          <Show when={isCompact()}>
            <div class="grid grid-cols-1 gap-4">
              <For each={hooks()}>
                {(hook) => (
                  <Card>
                    <CardContent class="p-4">
                      <div class="mb-3 flex items-center justify-between">
                        <div class="flex flex-col gap-1">
                          <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                            {getEventLabel(hook.event)}
                          </span>
                          <div class="flex items-center gap-2">
                            <span class="font-bold">{getTypeLabel(hook.type)}</span>
                            <Show when={hook.priority !== null}>
                              <Badge variant="secondary" class="h-4 px-1 text-[10px]">
                                P{hook.priority}
                              </Badge>
                            </Show>
                          </div>
                        </div>
                        <Switch
                          class="relative"
                          checked={hook.enabled}
                          onChange={(checked) => handleToggleEnabled(hook.id, checked)}
                        >
                          <SwitchControl>
                            <SwitchThumb />
                          </SwitchControl>
                        </Switch>
                      </div>

                      <div class="space-y-2 text-sm">
                        <div class="break-all rounded bg-muted/50 p-2 font-mono text-xs">
                          {hook.type === 'HTTP'
                            ? (hook.config.url ?? '-')
                            : (hook.config.command ?? '-')}
                        </div>

                        <div class="flex items-center justify-between">
                          <span class="text-xs text-muted-foreground">
                            {m.hook_onError()}: {getOnErrorLabel(hook.onError)}
                          </span>
                          <div class="flex gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              class="size-8"
                              onClick={() => props.onEdit?.(hook)}
                            >
                              <IconEdit class="size-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              class="size-8 text-destructive hover:bg-destructive/10 hover:text-destructive"
                              onClick={() => setConfirmDeleteId(hook.id)}
                            >
                              <IconTrash2 class="size-4" />
                            </Button>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}
              </For>
            </div>
          </Show>

          <Show when={!isCompact()}>
            <div>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead class="whitespace-nowrap">{m.hook_event()}</TableHead>
                    <TableHead class="w-full">{m.hook_type()}</TableHead>
                    <TableHead class="whitespace-nowrap text-center">{m.hook_priority()}</TableHead>
                    <TableHead class="whitespace-nowrap">{m.hook_onError()}</TableHead>
                    <TableHead class="whitespace-nowrap text-center">{m.common_status()}</TableHead>
                    <TableHead class="whitespace-nowrap text-right">{m.common_actions()}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <For each={hooks()}>
                    {(hook) => (
                      <TableRow>
                        <TableCell class="whitespace-nowrap">{getEventLabel(hook.event)}</TableCell>
                        <TableCell class="w-full">
                          <div class="flex flex-col gap-1">
                            <span class="font-medium">{getTypeLabel(hook.type)}</span>
                            <span class="break-all text-xs text-muted-foreground">
                              <Show when={hook.type === 'HTTP'}>{hook.config.url ?? '-'}</Show>
                              <Show when={hook.type === 'COMMAND'}>
                                {hook.config.command ?? '-'}
                              </Show>
                            </span>
                          </div>
                        </TableCell>
                        <TableCell class="text-center">{hook.priority ?? '-'}</TableCell>
                        <TableCell class="whitespace-nowrap">
                          <span class="text-xs">{getOnErrorLabel(hook.onError)}</span>
                        </TableCell>
                        <TableCell class="text-center">
                          <Switch
                            class="relative"
                            checked={hook.enabled}
                            onChange={(checked) => handleToggleEnabled(hook.id, checked)}
                          >
                            <SwitchControl>
                              <SwitchThumb />
                            </SwitchControl>
                          </Switch>
                        </TableCell>
                        <TableCell class="whitespace-nowrap text-right">
                          <div class="flex justify-end gap-2">
                            <Button variant="ghost" size="sm" onClick={() => props.onEdit?.(hook)}>
                              {m.common_edit()}
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              class="text-destructive hover:bg-destructive/10 hover:text-destructive"
                              onClick={() => setConfirmDeleteId(hook.id)}
                            >
                              {m.common_delete()}
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                  </For>
                </TableBody>
              </Table>
            </div>
          </Show>
        </Show>
      </Show>

      <Dialog
        open={confirmDeleteId() !== null}
        onOpenChange={(open) => !open && setConfirmDeleteId(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{m.hook_confirmDelete()}</DialogTitle>
            <DialogDescription>{m.hook_confirmDeleteDesc()}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmDeleteId(null)}>
              {m.common_cancel()}
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                const id = confirmDeleteId();
                if (id) {
                  handleDelete(id);
                  setConfirmDeleteId(null);
                }
              }}
            >
              {m.common_delete()}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
