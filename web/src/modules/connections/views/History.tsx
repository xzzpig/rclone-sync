import IntervalUpdated from '@/components/common/IntervalUpdated';
import StatusIcon from '@/components/common/StatusIcon';
import TableSkeleton from '@/components/common/TableSkeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Pagination,
  PaginationEllipsis,
  PaginationItem,
  PaginationItems,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { formatBytes } from '@/lib/utils';
import { useHistory } from '@/store/history';
import { useTasks } from '@/store/tasks';
import { useNavigate, useParams, useSearchParams } from '@solidjs/router';
import { formatDistanceToNow } from 'date-fns';
import { enUS } from 'date-fns/locale';
import { Component, For, Show, createEffect, createMemo } from 'solid-js';
import IconFileText from '~icons/lucide/file-text';
import IconRefreshCw from '~icons/lucide/refresh-cw';
import ConnectionViewLayout from '../layouts/ConnectionViewLayout';

const History: Component = () => {
  const params = useParams<{ connectionName: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams<{ task_id?: string; page?: string }>();
  const [historyState, historyActions] = useHistory();
  const [taskState, _] = useTasks();

  // Derived values from URL params
  const selectedTaskId = () => searchParams.task_id;
  const currentPage = () => parseInt(searchParams.page ?? '1', 10);

  // Reload jobs when task filter or page changes
  createEffect(() => {
    const taskId = selectedTaskId();
    const page = currentPage();
    historyActions.loadJobs({
      remote_name: params.connectionName,
      task_id: taskId,
      page,
    });
  });

  const setSelectedTaskId = (taskId: string | undefined) => {
    setSearchParams({ task_id: taskId, page: '1' });
  };

  const formatDuration = (start: string, end?: string, now: number = Date.now()) => {
    const startTime = new Date(start);
    const endTime = end ? new Date(end) : new Date(now);
    const duration = endTime.getTime() - startTime.getTime();
    const seconds = Math.floor(duration / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  };

  // Filter tasks by current connection
  const filteredTasks = createMemo(() => {
    return taskState.tasks.filter((t) => t.remote_name === params.connectionName);
  });

  const taskNameMap = createMemo(() => {
    const map = new Map<string, string>();
    filteredTasks().forEach((task) => {
      map.set(task.id, task.name);
    });
    return map;
  });

  const handleViewLogs = (jobId: string, taskId?: string) => {
    const queryParams = new URLSearchParams();
    if (jobId) queryParams.set('job_id', jobId);
    if (taskId) queryParams.set('task_id', taskId);
    navigate(`/connections/${params.connectionName}/log?${queryParams.toString()}`);
  };

  const handleRefresh = () => {
    historyActions.loadJobs({
      remote_name: params.connectionName,
      task_id: selectedTaskId(),
      page: currentPage(),
    });
  };

  const totalPages = createMemo(() =>
    Math.ceil(historyState.jobsTotal / historyState.jobsPageSize)
  );

  const handlePageChange = (page: number) => {
    setSearchParams({ page: page.toString() });
  };

  return (
    <ConnectionViewLayout
      title="Sync History"
      actions={
        <>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={historyState.isLoadingJobs}
          >
            <IconRefreshCw class="size-4" />
          </Button>
          <Select
            value={selectedTaskId() ?? ''}
            onChange={(value) => setSelectedTaskId(value ?? undefined)}
            options={['', ...(filteredTasks().map((t) => t.id) ?? [])]}
            placeholder="Filter Task"
            itemComponent={(props) => (
              <SelectItem item={props.item}>
                {props.item.rawValue === ''
                  ? 'All Tasks'
                  : (taskNameMap().get(props.item.rawValue as string) ?? props.item.rawValue)}
              </SelectItem>
            )}
          >
            <SelectTrigger class="w-[200px]">
              <SelectValue<string>>
                {(state) => {
                  const selectedId = state.selectedOption() as string;
                  return selectedId === ''
                    ? 'All Tasks'
                    : (taskNameMap().get(selectedId) ?? 'Select Task');
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent />
          </Select>
        </>
      }
    >
      <Show
        when={historyState.jobs.length > 0 || historyState.isLoadingJobs}
        fallback={
          <div class="flex flex-1 items-center justify-center text-muted-foreground">
            No execution history
          </div>
        }
      >
        <div class="relative min-h-0 flex-1 overflow-auto">
          <Table>
            <TableHeader class="sticky top-0 z-10 bg-card shadow-sm">
              <TableRow>
                <TableHead class="w-[200px] whitespace-nowrap">Task</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">Trigger</TableHead>
                <TableHead class="w-[150px] whitespace-nowrap">Started</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">Duration</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">Files</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">Data</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <Show
                when={!historyState.isLoadingJobs}
                fallback={<TableSkeleton columns={7} rows={historyState.jobsPageSize} />}
              >
                <For each={historyState.jobs}>
                  {(job) => {
                    const task = job.edges?.task;
                    return (
                      <TableRow>
                        <TableCell class="flex items-center gap-2 py-2 align-top font-medium">
                          <StatusIcon status={job.status} class="inline-block" />
                          <div
                            class="max-w-[200px] truncate"
                            title={task ? task.name : 'Unknown Task'}
                          >
                            {task ? task.name : 'Unknown Task'}
                          </div>
                        </TableCell>
                        <TableCell class="py-2 align-top">
                          <Badge variant="outline">{job.trigger}</Badge>
                        </TableCell>
                        <TableCell class="whitespace-nowrap py-2 align-top">
                          <IntervalUpdated when={true} interval={60 * 1000}>
                            {() =>
                              formatDistanceToNow(new Date(job.start_time), {
                                addSuffix: true,
                                locale: enUS,
                              })
                            }
                          </IntervalUpdated>
                        </TableCell>
                        <TableCell class="whitespace-nowrap py-2 align-top">
                          <IntervalUpdated when={job.status === 'running'}>
                            {(now) =>
                              formatDuration(
                                job.start_time,
                                job.status === 'running' ? undefined : job.end_time,
                                now
                              )
                            }
                          </IntervalUpdated>
                        </TableCell>
                        <TableCell class="py-2 align-top">
                          {job.status === 'running' ? 'N/A' : (job.files_transferred ?? 0)}
                        </TableCell>
                        <TableCell class="whitespace-nowrap py-2 align-top">
                          {formatBytes(job.bytes_transferred)}
                        </TableCell>
                        <TableCell class="py-2 text-right align-top">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleViewLogs(job.id, task?.id)}
                          >
                            <IconFileText class="size-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  }}
                </For>
              </Show>
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <Show when={historyState.jobsTotal > historyState.jobsPageSize}>
          <div class="mt-4 flex shrink-0 justify-center">
            <Pagination
              count={totalPages()}
              page={currentPage()}
              onPageChange={handlePageChange}
              itemComponent={(props) => (
                <PaginationItem page={props.page}>{props.page}</PaginationItem>
              )}
              ellipsisComponent={() => <PaginationEllipsis />}
            >
              <PaginationPrevious />
              <PaginationItems />
              <PaginationNext />
            </Pagination>
          </div>
        </Show>
      </Show>
    </ConnectionViewLayout>
  );
};

export default History;
