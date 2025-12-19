import { getConnections } from '@/api/connections';
import LanguageSwitcher from '@/components/common/LanguageSwitcher';
import ModeToggle from '@/components/common/ModeToggle';
import { Button } from '@/components/ui/button';
import { ListSkeleton } from '@/components/ui/skeletons';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { AddConnectionDialog } from '@/modules/connections/components/AddConnectionDialog';
import { ConnectionSidebarItem } from '@/modules/connections/components/ConnectionSidebarItem';
import { ImportWizard } from '@/modules/connections/components/ImportWizard/ImportWizard';
import * as m from '@/paraglide/messages.js';
import { useTasks } from '@/store/tasks';
import { A } from '@solidjs/router';
import { useQuery } from '@tanstack/solid-query';
import { Component, For, createSignal } from 'solid-js';
import IconCloud from '~icons/lucide/cloud';
import IconImport from '~icons/lucide/import';
import IconLayoutGrid from '~icons/lucide/layout-grid';
import IconPlus from '~icons/lucide/plus';
import IconSettings from '~icons/lucide/settings';

const Sidebar: Component = () => {
  const connectionsQuery = useQuery(() => ({
    queryKey: ['connections'],
    queryFn: getConnections,
    staleTime: 5 * 60 * 1000, // 5 minutes
  }));
  const [isDialogOpen, setIsDialogOpen] = createSignal(false);
  const [isImportOpen, setIsImportOpen] = createSignal(false);
  const [, actions] = useTasks();

  return (
    <nav class="flex size-full flex-col bg-card" role="navigation" aria-label="Main navigation">
      <div class="p-6 pb-2">
        <div class="mb-6 flex items-center gap-2 px-2">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary text-lg font-bold text-primary-foreground">
            C
          </div>
          <h1 class="text-lg font-bold tracking-tight">{m.app_title()}</h1>
        </div>

        <div class="space-y-1">
          <A
            href="/overview"
            activeClass="bg-secondary text-foreground shadow-sm"
            class="group flex w-full items-center rounded-md px-3 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
          >
            <IconLayoutGrid class="mr-3 size-4" />
            {m.common_overview()}
          </A>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto px-4 py-2">
        <div class="mb-2 flex items-center justify-between p-2">
          <h2 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {m.common_connections()}
          </h2>
          <div class="flex items-center gap-1">
            <Tooltip>
              <TooltipTrigger
                as={Button}
                variant="ghost"
                size="sm"
                class="size-6 p-0 hover:bg-muted"
                onClick={() => setIsImportOpen(true)}
              >
                <IconImport class="size-4" />
              </TooltipTrigger>
              <TooltipContent>{m.connection_importConfig()}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                as={Button}
                variant="ghost"
                size="sm"
                class="size-6 p-0 hover:bg-muted"
                onClick={() => setIsDialogOpen(true)}
              >
                <IconPlus class="size-4" />
              </TooltipTrigger>
              <TooltipContent>{m.connection_addNew()}</TooltipContent>
            </Tooltip>
          </div>
        </div>

        <div class="space-y-1">
          {connectionsQuery.isLoading ? (
            <div class="space-y-2 px-2">
              <ListSkeleton count={3} />
            </div>
          ) : (
            <For
              each={connectionsQuery.data}
              fallback={
                <div class="rounded-lg border-2 border-dashed border-muted-foreground/20 px-2 py-8 text-center">
                  <IconCloud class="mx-auto mb-2 size-8 text-muted-foreground/40" />
                  <p class="text-xs text-muted-foreground">{m.sidebar_noConnections()}</p>
                  <Button
                    variant="link"
                    size="sm"
                    onClick={() => setIsDialogOpen(true)}
                    class="mt-1 h-auto py-0 text-xs"
                  >
                    {m.sidebar_addOneNow()}
                  </Button>
                </div>
              }
            >
              {(conn) => (
                <ConnectionSidebarItem connection={conn} status={actions.getTaskStatus(conn.id)} />
              )}
            </For>
          )}
        </div>
      </div>

      <div class="mt-auto border-t border-border p-4">
        <div class="flex items-center gap-2 rounded-md border border-border/50 bg-muted/30 p-2">
          <div class="flex size-8 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-indigo-600 text-xs font-bold text-white">
            A
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-xs font-medium">{m.sidebar_adminUser()}</p>
            <p class="truncate text-[10px] text-muted-foreground">{m.sidebar_proEdition()}</p>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <LanguageSwitcher />
            <ModeToggle />
            <Button variant="ghost" size="icon" as={A} href="/settings" title={m.common_settings()}>
              <IconSettings class="size-[1.2rem]" />
            </Button>
          </div>
        </div>
      </div>

      <AddConnectionDialog isOpen={isDialogOpen()} onClose={() => setIsDialogOpen(false)} />
      <ImportWizard isOpen={isImportOpen()} onClose={() => setIsImportOpen(false)} />
    </nav>
  );
};

export default Sidebar;
