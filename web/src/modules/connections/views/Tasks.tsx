import StatusIcon from '@/components/common/StatusIcon';
import TableSkeleton from '@/components/common/TableSkeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { showToast } from '@/components/ui/toast';
import type { CreateTaskInput, StatusType, UpdateTaskInput } from '@/lib/types';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useTasks } from '@/store/tasks';
import { useNavigate, useParams } from '@solidjs/router';
import { createSignal, For, Show } from 'solid-js';
import IconCalendarPlus from '~icons/lucide/calendar-plus';
import IconEdit from '~icons/lucide/edit';
import IconHistory from '~icons/lucide/history';
import IconPlay from '~icons/lucide/play';
import IconTrash2 from '~icons/lucide/trash-2';
import { CreateTaskWizard } from '../components/CreateTaskWizard';
import { EditTaskDialog } from '../components/EditTaskDialog';
import ConnectionViewLayout from '../layouts/ConnectionViewLayout';

// Direction display helper - handles both GraphQL enum and legacy formats
const DirectionArrow = (props: { direction: string; class?: string }) => {
  const getArrow = () => {
    const dir = props.direction.toUpperCase();
    switch (dir) {
      case 'UPLOAD':
        return '→';
      case 'DOWNLOAD':
        return '←';
      case 'BIDIRECT':
      case 'BIDIRECTIONAL':
        return '↔';
      default:
        return '?';
    }
  };

  return (
    <span class={cn('mx-1 text-lg font-bold', props.class)} title={props.direction}>
      {getArrow()}
    </span>
  );
};

const PathDisplay = (props: { path: string; type: 'local' | 'remote'; class?: string }) => {
  return (
    <div class={cn('flex flex-col gap-0.5', props.class)}>
      <span class="hidden text-xs text-muted-foreground md:inline">
        {props.type === 'local' ? m.task_source() : m.task_destination()}
      </span>
      <span class="max-w-full truncate font-mono text-sm md:max-w-[400px]" title={props.path}>
        {props.path}
      </span>
    </div>
  );
};

function Tasks() {
  const [state, actions] = useTasks();
  const [selectedTaskId, setSelectedTaskId] = createSignal<string | null>(null);
  const [isDeleteConfirmOpen, setDeleteConfirmOpen] = createSignal(false);
  const [isEditDialogOpen, setEditDialogOpen] = createSignal(false);
  const [isCreateDialogOpen, setCreateDialogOpen] = createSignal(false);
  const params = useParams();
  const navigate = useNavigate();

  // Filter tasks for the current connection
  // GraphQL subscription handles real-time updates
  const filteredTasks = () => {
    const id = params.connectionId;
    if (!id) return state.tasks;
    return state.tasks.filter((task) => task.connection?.id === id);
  };

  const selectedTask = () => {
    const id = selectedTaskId();
    if (!id) return null;
    return state.tasks.find((task) => task.id === id) ?? null;
  };

  const handleRowClick = (taskId: string) => {
    setSelectedTaskId((current) => (current === taskId ? null : taskId));
  };

  const handleRunTask = async () => {
    const task = selectedTask();
    if (task) {
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

  const handleEditTask = () => {
    if (selectedTask()) {
      setEditDialogOpen(true);
    }
  };

  const handleHistory = () => {
    const task = selectedTask();
    if (task) {
      navigate(`/connections/${params.connectionId}/history?task_id=${task.id}`);
    }
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
            <div class="hidden text-sm text-muted-foreground md:block">
              {selectedTask() ? m.task_selectedCount({ count: 1 }) : ''}
            </div>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleRunTask}
              title={m.task_syncNow()}
              aria-label={m.task_syncNow()}
            >
              <IconPlay class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.task_syncNow()}</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleHistory}
              title={m.history_title()}
              aria-label={m.history_title()}
            >
              <IconHistory class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.history_title()}</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleEditTask}
              title={m.task_edit()}
              aria-label={m.task_edit()}
            >
              <IconEdit class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.common_edit()}</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="destructive"
              size="sm"
              onClick={() => setDeleteConfirmOpen(true)}
              title={m.task_delete()}
              aria-label={m.task_delete()}
            >
              <IconTrash2 class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.common_delete()}</span>
            </Button>
            <div class="mx-1 h-8 w-px bg-border" />
            <Button
              onClick={() => setCreateDialogOpen(true)}
              size="sm"
              aria-label={m.task_create()}
            >
              <IconCalendarPlus class="size-4 md:mr-2" />
              <span class="hidden md:inline">{m.task_create()}</span>
            </Button>
          </>
        }
      >
        <div class="min-h-0 flex-1 overflow-auto">
          <Show
            when={filteredTasks().length > 0 || state.isLoading}
            fallback={
              <div class="flex h-24 items-center justify-center text-muted-foreground">
                {m.task_noTasks()}
              </div>
            }
          >
            <Table>
              <TableHeader class="sticky top-0 z-10 bg-card shadow-sm">
                <TableRow>
                  <TableHead class="whitespace-nowrap">{m.form_taskName()}</TableHead>
                  <TableHead class="whitespace-nowrap">{m.task_syncMode()}</TableHead>
                  <TableHead class="hidden whitespace-nowrap md:table-cell">
                    {m.task_lastSync()}
                  </TableHead>
                  <TableHead class="hidden whitespace-nowrap md:table-cell">
                    {m.task_schedule()}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <Show
                  when={!state.isLoading}
                  fallback={<TableSkeleton columns={4} hiddenColumns={[2, 3]} />}
                >
                  <For each={filteredTasks()}>
                    {(task) => {
                      // Use latestJob from GraphQL/subscription
                      const latestJob = () => task.latestJob;
                      // Use GraphQL uppercase status directly, default to IDLE
                      const status = (): StatusType => latestJob()?.status ?? 'IDLE';
                      const lastRun = () => {
                        const job = latestJob();
                        return job?.endTime ?? job?.startTime;
                      };

                      return (
                        <TableRow
                          data-state={selectedTaskId() === task.id && 'selected'}
                          class="cursor-pointer data-[state=selected]:bg-primary/10"
                          onClick={() => handleRowClick(task.id)}
                        >
                          <TableCell class="py-2">
                            <div class="flex items-center gap-2">
                              <StatusIcon status={status()} class="inline-block" />
                              <div
                                class="max-w-[120px] truncate md:max-w-[200px]"
                                title={task.name}
                              >
                                {task.name}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell class="py-2">
                            <div class="flex w-full min-w-0 flex-col items-center gap-1 md:flex-row md:gap-2">
                              <PathDisplay path={task.sourcePath} type="local" class="flex-1" />
                              <DirectionArrow
                                direction={task.direction}
                                class="rotate-90 md:rotate-0"
                              />
                              <PathDisplay
                                path={`${task.connection?.name ?? '?'}:${task.remotePath}`}
                                type="remote"
                                class="flex-1"
                              />
                            </div>
                          </TableCell>
                          <TableCell class="hidden py-2 md:table-cell">
                            {lastRun()
                              ? new Date(lastRun()!).toLocaleString()
                              : m.history_notApplicable()}
                          </TableCell>
                          <TableCell class="hidden py-2 md:table-cell">
                            <Show when={task.realtime}>
                              <Badge variant="default">{m.task_scheduleRealtime()}</Badge>
                            </Show>
                            <Show when={!task.realtime && (!task.schedule || task.schedule === '')}>
                              <Badge variant="secondary">{m.task_scheduleManual()}</Badge>
                            </Show>
                            <Show when={!task.realtime && task.schedule && task.schedule !== ''}>
                              <span class="font-mono text-sm">{task.schedule}</span>
                            </Show>
                          </TableCell>
                        </TableRow>
                      );
                    }}
                  </For>
                </Show>
              </TableBody>
            </Table>
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
