import * as m from '@/paraglide/messages.js';
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
import { formatDuration, formatRelativeTime } from '@/lib/date';
import { useHistory } from '@/store/history';
import { useTasks } from '@/store/tasks';
import { useNavigate, useParams, useSearchParams } from '@solidjs/router';
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

  const calculateDuration = (start: string, end?: string, now: number = Date.now()) => {
    const startTime = new Date(start);
    const endTime = end ? new Date(end) : new Date(now);
    const duration = endTime.getTime() - startTime.getTime();
    return formatDuration(duration);
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
      title={m.history_title()}
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
            placeholder={m.task_filters()}
            itemComponent={(props) => (
              <SelectItem item={props.item}>
                {props.item.rawValue === ''
                  ? m.task_noTasks()
                  : (taskNameMap().get(props.item.rawValue as string) ?? props.item.rawValue)}
              </SelectItem>
            )}
          >
            <SelectTrigger class="w-[200px]">
              <SelectValue<string>>
                {(state) => {
                  const selectedId = state.selectedOption() as string;
                  return selectedId === ''
                    ? m.task_noTasks()
                    : (taskNameMap().get(selectedId) ?? m.task_selectDestination());
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
            {m.history_noExecutionHistory()}
          </div>
        }
      >
        <div class="relative min-h-0 flex-1 overflow-auto">
          <Table>
            <TableHeader class="sticky top-0 z-10 bg-card shadow-sm">
              <TableRow>
                <TableHead class="w-[200px] whitespace-nowrap">{m.history_tableTask()}</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">
                  {m.history_tableTrigger()}
                </TableHead>
                <TableHead class="w-[150px] whitespace-nowrap">
                  {m.history_tableStarted()}
                </TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">{m.common_duration()}</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">{m.common_files()}</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">{m.common_data()}</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap text-right">
                  {m.common_actions()}
                </TableHead>
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
                            title={task ? task.name : m.error_taskNotFound()}
                          >
                            {task ? task.name : m.error_taskNotFound()}
                          </div>
                        </TableCell>
                        <TableCell class="py-2 align-top">
                          <Badge variant="outline">{job.trigger}</Badge>
                        </TableCell>
                        <TableCell class="whitespace-nowrap py-2 align-top">
                          <IntervalUpdated when={true} interval={60 * 1000}>
                            {() => formatRelativeTime(job.start_time)}
                          </IntervalUpdated>
                        </TableCell>
                        <TableCell class="whitespace-nowrap py-2 align-top">
                          <IntervalUpdated when={job.status === 'running'}>
                            {(now) =>
                              calculateDuration(
                                job.start_time,
                                job.status === 'running' ? undefined : job.end_time,
                                now
                              )
                            }
                          </IntervalUpdated>
                        </TableCell>
                        <TableCell class="py-2 align-top">
                          {job.status === 'running'
                            ? m.history_notApplicable()
                            : (job.files_transferred ?? 0)}
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
