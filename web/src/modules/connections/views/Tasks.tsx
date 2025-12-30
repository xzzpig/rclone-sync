import StatusIcon from '@/components/common/StatusIcon';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { TextField, TextFieldInput } from '@/components/ui/text-field';
import { showToast } from '@/components/ui/toast';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { CreateTaskInput, StatusType, TaskListItem, UpdateTaskInput } from '@/lib/types';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useTasks } from '@/store/tasks';
import { useNavigate, useParams } from '@solidjs/router';
import { createSignal, For, Match, Show, Switch } from 'solid-js';
import IconArrowLeft from '~icons/lucide/arrow-left';
import IconArrowLeftRight from '~icons/lucide/arrow-left-right';
import IconArrowRight from '~icons/lucide/arrow-right';
import IconCalendar from '~icons/lucide/calendar';
import IconCalendarPlus from '~icons/lucide/calendar-plus';
import IconClock from '~icons/lucide/clock';
import IconCloud from '~icons/lucide/cloud';
import IconEdit from '~icons/lucide/edit';
import IconFilter from '~icons/lucide/filter';
import IconHardDrive from '~icons/lucide/hard-drive';
import IconHistory from '~icons/lucide/history';
import IconLayers from '~icons/lucide/layers';
import IconPlay from '~icons/lucide/play';
import IconSearch from '~icons/lucide/search';
import IconShieldCheck from '~icons/lucide/shield-check';
import IconTrash2 from '~icons/lucide/trash-2';
import { CreateTaskWizard } from '../components/CreateTaskWizard';
import { EditTaskDialog } from '../components/EditTaskDialog';
import ConnectionViewLayout from '../layouts/ConnectionViewLayout';

// Direction display helper using Lucide icons
const DirectionArrow = (props: { direction: string; class?: string }) => {
  const dir = () => props.direction.toUpperCase();
  return (
    <div class={cn('flex items-center justify-center', props.class)}>
      <Switch fallback={<IconArrowRight class="size-4" />}>
        <Match when={dir() === 'DOWNLOAD'}>
          <IconArrowLeft class="size-4" />
        </Match>
        <Match when={dir() === 'BIDIRECTIONAL'}>
          <IconArrowLeftRight class="size-4" />
        </Match>
      </Switch>
    </div>
  );
};

function Tasks() {
  const [state, actions] = useTasks();
  const [selectedTaskId, setSelectedTaskId] = createSignal<string | null>(null);
  const [searchQuery, setSearchQuery] = createSignal('');
  const [isDeleteConfirmOpen, setDeleteConfirmOpen] = createSignal(false);
  const [isEditDialogOpen, setEditDialogOpen] = createSignal(false);
  const [isCreateDialogOpen, setCreateDialogOpen] = createSignal(false);
  const params = useParams();
  const navigate = useNavigate();

  const filteredTasks = () => {
    const id = params.connectionId;
    let tasks = state.tasks;
    if (id) {
      tasks = tasks.filter((task) => task.connection?.id === id);
    }

    const query = searchQuery().toLowerCase().trim();
    if (query) {
      tasks = tasks.filter(
        (task) =>
          task.name.toLowerCase().includes(query) ||
          task.sourcePath.toLowerCase().includes(query) ||
          task.remotePath.toLowerCase().includes(query)
      );
    }
    return tasks;
  };

  const selectedTask = () => {
    const id = selectedTaskId();
    if (!id) return null;
    return state.tasks.find((task) => task.id === id) ?? null;
  };

  const handleRunTask = async (task: TaskListItem) => {
    try {
      await actions.runTask(task.id);
      showToast({
        title: m.toast_taskStarted(),
        description: m.toast_taskStartedDesc({ name: task.name }),
      });
    } catch (error) {
      showToast({
        title: m.toast_failedToStartTask(),
        description: error instanceof Error ? error.message : m.error_unknownError(),
        variant: 'destructive',
      });
    }
  };

  const handleDeleteTask = async () => {
    const task = selectedTask();
    if (task) {
      try {
        await actions.deleteTask(task.id);
        showToast({
          title: m.toast_taskDeleted(),
          description: m.toast_taskDeletedDesc({ name: task.name }),
        });
        setDeleteConfirmOpen(false);
        setSelectedTaskId(null);
      } catch (error) {
        showToast({
          title: m.toast_failedToDeleteTask(),
          description: error instanceof Error ? error.message : m.error_unknownError(),
          variant: 'destructive',
        });
      }
    }
  };

  const handleEditTask = (taskId: string) => {
    setSelectedTaskId(taskId);
    setEditDialogOpen(true);
  };

  const handleHistory = (task: TaskListItem) => {
    navigate(`/connections/${params.connectionId}/history?task_id=${task.id}`);
  };

  const handleSaveTask = async (id: string, updates: UpdateTaskInput) => {
    await actions.updateTask(id, updates);
  };

  const handleCreateTask = async (input: CreateTaskInput) => {
    await actions.createTask(input);
  };

  return (
    <>
      <ConnectionViewLayout
        title={m.task_title()}
        actions={
          <>
            <div class="relative max-w-[160px] md:max-w-[240px]">
              <IconSearch class="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
              <TextField>
                <TextFieldInput
                  placeholder={m.common_search()}
                  class="h-9 pl-9"
                  value={searchQuery()}
                  onInput={(e) => setSearchQuery(e.currentTarget.value)}
                />
              </TextField>
            </div>
            <Button
              onClick={() => setCreateDialogOpen(true)}
              size="sm"
              aria-label={m.task_create()}
              class="shrink-0"
            >
              <IconCalendarPlus class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.task_create()}</span>
            </Button>
          </>
        }
      >
        <div class="min-h-0 flex-1 overflow-auto px-1">
          <Show
            when={filteredTasks().length > 0 || state.isLoading}
            fallback={
              <div class="flex h-32 flex-col items-center justify-center rounded-lg border border-dashed text-muted-foreground">
                <p>{searchQuery() ? (m.common_no?.() ?? 'No results found') : m.task_noTasks()}</p>
                <Show when={!searchQuery()}>
                  <Button
                    variant="link"
                    onClick={() => setCreateDialogOpen(true)}
                    class="mt-2 text-primary"
                  >
                    {m.task_create()}
                  </Button>
                </Show>
              </div>
            }
          >
            <div class="grid grid-cols-1 gap-4 pb-4">
              <Show
                when={!state.isLoading}
                fallback={
                  <For each={Array(3)}>
                    {() => (
                      <Card class="border-dashed opacity-50">
                        <CardContent class="p-6">
                          <div class="flex items-center gap-4">
                            <Skeleton class="size-10 rounded-full" />
                            <div class="flex-1 space-y-2">
                              <Skeleton class="h-4 w-1/4" />
                              <Skeleton class="h-3 w-3/4" />
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    )}
                  </For>
                }
              >
                <For each={filteredTasks()}>
                  {(task) => {
                    const latestJob = () => task.latestJob;
                    const status = (): StatusType => latestJob()?.status ?? 'IDLE';
                    const lastRun = () => {
                      const job = latestJob();
                      return job?.endTime ?? job?.startTime;
                    };

                    return (
                      <Card
                        class={cn(
                          'group transition-all hover:border-primary/50 hover:shadow-md',
                          selectedTaskId() === task.id && 'border-primary bg-primary/5'
                        )}
                        onClick={() =>
                          setSelectedTaskId((prev) => (prev === task.id ? null : task.id))
                        }
                      >
                        <CardContent class="p-3 md:px-4 md:py-3">
                          {/* Top Row: Info and Actions */}
                          <div class="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
                            <div class="flex min-w-0 flex-1 items-start gap-3">
                              <StatusIcon status={status()} class="mt-0.5 size-5 shrink-0" />
                              <div class="min-w-0 flex-1">
                                <div class="flex flex-col md:flex-row md:items-baseline md:gap-4">
                                  <h3
                                    class="truncate text-base font-bold tracking-tight md:max-w-[300px]"
                                    title={task.name}
                                  >
                                    {task.name}
                                  </h3>
                                  <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[11px] text-muted-foreground">
                                    <div class="flex items-center gap-1">
                                      <IconClock class="size-3" />
                                      <span>
                                        {lastRun()
                                          ? new Date(lastRun()!).toLocaleString()
                                          : m.history_notApplicable()}
                                      </span>
                                    </div>
                                    <div class="flex items-center gap-1">
                                      <IconCalendar class="size-3" />
                                      <Show when={task.realtime}>
                                        <Badge
                                          variant="outline"
                                          class="h-4 border-primary/30 bg-primary/5 px-1 text-[10px] text-primary"
                                        >
                                          {m.task_scheduleRealtime()}
                                        </Badge>
                                      </Show>
                                      <Show
                                        when={
                                          !task.realtime && (!task.schedule || task.schedule === '')
                                        }
                                      >
                                        <span>{m.task_scheduleManual()}</span>
                                      </Show>
                                      <Show
                                        when={
                                          !task.realtime && task.schedule && task.schedule !== ''
                                        }
                                      >
                                        <span class="font-mono">{task.schedule}</span>
                                      </Show>
                                    </div>

                                    {/* Extended Options */}
                                    <Show
                                      when={
                                        task.options?.filters && task.options.filters.length > 0
                                      }
                                    >
                                      <div class="flex items-center gap-1">
                                        <IconFilter class="size-3" />
                                        <Badge
                                          variant="outline"
                                          class="h-4 border-blue-500/30 bg-blue-500/5 px-1 text-[10px] text-blue-600"
                                        >
                                          {m.task_filterRulesCount({
                                            count: task.options?.filters?.length ?? 0,
                                          })}
                                        </Badge>
                                      </div>
                                    </Show>
                                    <Show when={task.options?.noDelete}>
                                      <div class="flex items-center gap-1">
                                        <IconShieldCheck class="size-3" />
                                        <Badge
                                          variant="outline"
                                          class="h-4 border-green-500/30 bg-green-500/5 px-1 text-[10px] text-green-600"
                                        >
                                          {m.task_noDeleteEnabled()}
                                        </Badge>
                                      </div>
                                    </Show>
                                    <Show
                                      when={task.options?.transfers && task.options.transfers !== 4}
                                    >
                                      <div class="flex items-center gap-1">
                                        <IconLayers class="size-3" />
                                        <Badge
                                          variant="outline"
                                          class="h-4 border-purple-500/30 bg-purple-500/5 px-1 text-[10px] text-purple-600"
                                        >
                                          {m.task_transfersCount({
                                            count: task.options?.transfers ?? 4,
                                          })}
                                        </Badge>
                                      </div>
                                    </Show>
                                  </div>
                                </div>
                              </div>
                            </div>

                            {/* Actions Group */}
                            <div
                              class={cn(
                                'flex items-center gap-0.5 self-end transition-opacity md:self-start md:opacity-0 md:group-hover:opacity-100',
                                selectedTaskId() === task.id && 'md:opacity-100'
                              )}
                            >
                              <Tooltip>
                                <TooltipTrigger
                                  as={Button}
                                  size="icon"
                                  variant="ghost"
                                  onClick={(e: MouseEvent) => {
                                    e.stopPropagation();
                                    handleRunTask(task);
                                  }}
                                  class="size-7 text-blue-500 hover:bg-blue-500/10 hover:text-blue-600"
                                >
                                  <IconPlay class="size-3.5" />
                                </TooltipTrigger>
                                <TooltipContent>{m.task_syncNow()}</TooltipContent>
                              </Tooltip>

                              <Tooltip>
                                <TooltipTrigger
                                  as={Button}
                                  size="icon"
                                  variant="ghost"
                                  onClick={(e: MouseEvent) => {
                                    e.stopPropagation();
                                    handleHistory(task);
                                  }}
                                  class="size-7"
                                >
                                  <IconHistory class="size-3.5" />
                                </TooltipTrigger>
                                <TooltipContent>{m.history_title()}</TooltipContent>
                              </Tooltip>

                              <Tooltip>
                                <TooltipTrigger
                                  as={Button}
                                  size="icon"
                                  variant="ghost"
                                  onClick={(e: MouseEvent) => {
                                    e.stopPropagation();
                                    handleEditTask(task.id);
                                  }}
                                  class="size-7"
                                >
                                  <IconEdit class="size-3.5" />
                                </TooltipTrigger>
                                <TooltipContent>{m.common_edit()}</TooltipContent>
                              </Tooltip>

                              <div class="mx-1 h-3 w-px bg-border" />

                              <Tooltip>
                                <TooltipTrigger
                                  as={Button}
                                  size="icon"
                                  variant="ghost"
                                  onClick={(e: MouseEvent) => {
                                    e.stopPropagation();
                                    setSelectedTaskId(task.id);
                                    setDeleteConfirmOpen(true);
                                  }}
                                  class="size-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
                                >
                                  <IconTrash2 class="size-3.5" />
                                </TooltipTrigger>
                                <TooltipContent>{m.common_delete()}</TooltipContent>
                              </Tooltip>
                            </div>
                          </div>

                          {/* Middle Row: Path Visualization */}
                          <div class="mt-2 grid grid-cols-1 items-center gap-2 rounded-lg border border-muted/50 bg-muted/30 p-2.5 transition-colors group-hover:bg-muted/50 md:grid-cols-[1fr_auto_1fr]">
                            <div class="min-w-0 space-y-0.5">
                              <div class="flex items-center gap-1 text-[9px] font-bold uppercase tracking-wider text-muted-foreground/80">
                                <IconHardDrive class="size-2.5" />
                                {m.task_source()}
                              </div>
                              <div
                                class="truncate font-mono text-xs font-medium"
                                title={task.sourcePath}
                              >
                                {task.sourcePath}
                              </div>
                            </div>

                            <div class="flex items-center justify-center py-0.5 md:py-0">
                              <div class="flex size-6 rotate-90 items-center justify-center rounded-full border bg-background shadow-sm ring-2 ring-muted/20 md:rotate-0">
                                <DirectionArrow
                                  direction={task.direction}
                                  class="scale-75 text-primary"
                                />
                              </div>
                            </div>

                            <div class="min-w-0 space-y-0.5 md:text-right">
                              <div class="flex items-center gap-1 text-[9px] font-bold uppercase tracking-wider text-muted-foreground/80 md:justify-end">
                                {m.task_destination()}
                                <IconCloud class="size-2.5" />
                              </div>
                              <div
                                class="truncate font-mono text-xs font-medium"
                                title={`${task.connection?.name ?? '?'}:${task.remotePath}`}
                              >
                                <span class="text-primary/70">{task.connection?.name ?? '?'}</span>
                                <span class="mx-0.5 text-muted-foreground">:</span>
                                {task.remotePath}
                              </div>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    );
                  }}
                </For>
              </Show>
            </div>
          </Show>
        </div>
      </ConnectionViewLayout>
      <Dialog open={isDeleteConfirmOpen()} onOpenChange={setDeleteConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle id="delete-dialog-title">{m.task_delete()}</DialogTitle>
          </DialogHeader>
          <div id="delete-dialog-description">
            <p>{m.task_deleteConfirm()}</p>
            <p class="mt-2 text-sm text-gray-500">{m.task_deleteWarning()}</p>
          </div>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setDeleteConfirmOpen(false)}
              aria-label={m.common_cancel()}
            >
              {m.common_cancel()}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteTask}
              aria-label={m.task_confirmDeletion({ name: selectedTask()?.name ?? '' })}
            >
              {m.common_delete()}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <EditTaskDialog
        task={selectedTask()}
        open={isEditDialogOpen()}
        onOpenChange={setEditDialogOpen}
        onSave={handleSaveTask}
      />
      <CreateTaskWizard
        connectionId={params.connectionId ?? ''}
        open={isCreateDialogOpen()}
        onClose={() => setCreateDialogOpen(false)}
        onSubmit={handleCreateTask}
      />
    </>
  );
}

export default Tasks;
