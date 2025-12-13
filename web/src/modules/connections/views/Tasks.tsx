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
import { cn } from '@/lib/utils';
import { Task } from '@/lib/types';
import { useTasks } from '@/store/tasks';
import IconPlay from '~icons/lucide/play';
import IconHistory from '~icons/lucide/history';
import IconEdit from '~icons/lucide/edit';
import IconTrash2 from '~icons/lucide/trash-2';
import IconCalendarPlus from '~icons/lucide/calendar-plus';
import { useNavigate, useParams } from '@solidjs/router';
import { createSignal, For, Show } from 'solid-js';
import { CreateTaskWizard } from '../components/CreateTaskWizard';
import { EditTaskDialog } from '../components/EditTaskDialog';
import ConnectionViewLayout from '../layouts/ConnectionViewLayout';

const DirectionArrow = (props: { direction: string; class?: string }) => {
  const getArrow = () => {
    switch (props.direction) {
      case 'upload':
        return '→';
      case 'download':
        return '←';
      case 'bidirectional':
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
        {props.type === 'local' ? 'Local' : 'Remote'}
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
  // Global SSE subscription in AppShell handles real-time updates
  const filteredTasks = () => {
    const name = params.connectionName;
    if (!name) return state.tasks;
    return state.tasks.filter((task) => task.remote_name === name);
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
          title: 'Task started',
          description: `Task "${task.name}" has been started.`,
        });
      } catch (error) {
        showToast({
          title: 'Failed to start task',
          description: error instanceof Error ? error.message : 'An unknown error occurred.',
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
          title: 'Task deleted',
          description: `Task "${task.name}" has been deleted successfully.`,
        });
        setDeleteConfirmOpen(false);
        setSelectedTaskId(null);
      } catch (error) {
        showToast({
          title: 'Failed to delete task',
          description: error instanceof Error ? error.message : 'An unknown error occurred.',
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
      navigate(`/connections/${params.connectionName}/history?task_id=${task.id}`);
    }
  };

  const handleSaveTask = async (id: string, updates: Partial<Task>) => {
    await actions.updateTask(id, updates);
  };

  const handleCreateTask = async (task: Omit<Task, 'id' | 'edges'>) => {
    await actions.createTask(task);
  };

  return (
    <>
      <ConnectionViewLayout
        title="Tasks"
        actions={
          <>
            <div class="hidden text-sm text-muted-foreground md:block">
              {selectedTask() ? '1 selected' : ''}
            </div>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleRunTask}
              title="Run task"
              aria-label="Run task"
            >
              <IconPlay class="size-4 md:mr-2" />
              <span class="hidden md:inline">Run</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleHistory}
              title="View history"
              aria-label="View history"
            >
              <IconHistory class="size-4 md:mr-2" />
              <span class="hidden md:inline">History</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="outline"
              size="sm"
              onClick={handleEditTask}
              title="Edit task"
              aria-label="Edit task"
            >
              <IconEdit class="size-4 md:mr-2" />
              <span class="hidden md:inline">Edit</span>
            </Button>
            <Button
              disabled={!selectedTask()}
              variant="destructive"
              size="sm"
              onClick={() => setDeleteConfirmOpen(true)}
              title="Delete task"
              aria-label="Delete task"
            >
              <IconTrash2 class="size-4 md:mr-2" />
              <span class="hidden md:inline">Delete</span>
            </Button>
            <div class="mx-1 h-8 w-px bg-border" />
            <Button
              onClick={() => setCreateDialogOpen(true)}
              size="sm"
              aria-label="Create new task"
            >
              <IconCalendarPlus class="size-4 md:mr-2" />
              <span class="hidden md:inline">Create Task</span>
            </Button>
          </>
        }
      >
        <div class="min-h-0 flex-1 overflow-auto">
          <Show
            when={filteredTasks().length > 0 || state.isLoading}
            fallback={
              <div class="flex h-24 items-center justify-center text-muted-foreground">
                No tasks found.
              </div>
            }
          >
            <Table>
              <TableHeader class="sticky top-0 z-10 bg-card shadow-sm">
                <TableRow>
                  <TableHead class="whitespace-nowrap">Task Name</TableHead>
                  <TableHead class="whitespace-nowrap">Sync Path</TableHead>
                  <TableHead class="hidden whitespace-nowrap md:table-cell">Last Run</TableHead>
                  <TableHead class="hidden whitespace-nowrap md:table-cell">Schedule</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <Show
                  when={!state.isLoading}
                  fallback={<TableSkeleton columns={4} hiddenColumns={[2, 3]} />}
                >
                  <For each={filteredTasks()}>
                    {(task) => {
                      const latestJob = () => task.edges?.jobs?.[0];
                      const status = () => latestJob()?.status ?? 'idle';
                      const lastRun = () => {
                        const job = latestJob();
                        return job?.end_time ?? job?.start_time;
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
                              <PathDisplay path={task.source_path} type="local" class="flex-1" />
                              <DirectionArrow
                                direction={task.direction}
                                class="rotate-90 md:rotate-0"
                              />
                              <PathDisplay
                                path={`${task.remote_name}:${task.remote_path}`}
                                type="remote"
                                class="flex-1"
                              />
                            </div>
                          </TableCell>
                          <TableCell class="hidden py-2 md:table-cell">
                            {lastRun() ? new Date(lastRun()!).toLocaleString() : 'N/A'}
                          </TableCell>
                          <TableCell class="hidden py-2 md:table-cell">
                            <Show when={task.realtime}>
                              <Badge variant="default">Real-time</Badge>
                            </Show>
                            <Show when={!task.realtime && (!task.schedule || task.schedule === '')}>
                              <Badge variant="secondary">Manual</Badge>
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
            <DialogTitle id="delete-dialog-title">Delete Task</DialogTitle>
          </DialogHeader>
          <div id="delete-dialog-description">
            <p>
              Are you sure you want to delete the task <strong>"{selectedTask()?.name}"</strong>?
            </p>
            <p class="mt-2 text-sm text-gray-500">
              This action cannot be undone. This will permanently delete the task and all its
              history.
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setDeleteConfirmOpen(false)}
              aria-label="Cancel task deletion"
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteTask}
              aria-label={`Confirm deletion of task ${selectedTask()?.name}`}
            >
              Delete
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
        remoteName={params.connectionName ?? ''}
        open={isCreateDialogOpen()}
        onClose={() => setCreateDialogOpen(false)}
        onSubmit={handleCreateTask}
      />
    </>
  );
}

export default Tasks;
