import * as m from '@/paraglide/messages.js';
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
import { formatRelativeTime } from '@/lib/date';
import { useHistory } from '@/store/history';
import { useTasks } from '@/store/tasks';
import { useParams, useSearchParams } from '@solidjs/router';
import { Component, For, Show, createEffect, createMemo, onMount } from 'solid-js';
import IconAlertCircle from '~icons/lucide/alert-circle';
import IconAlertTriangle from '~icons/lucide/alert-triangle';
import IconCheckCircle from '~icons/lucide/check-circle';
import IconInfo from '~icons/lucide/info';
import IconRefreshCw from '~icons/lucide/refresh-cw';
import ConnectionViewLayout from '../layouts/ConnectionViewLayout';

const Log: Component = () => {
  const params = useParams<{ connectionId: string }>();
  const [searchParams, setSearchParams] = useSearchParams<{
    job_id?: string;
    task_id?: string;
    level?: string;
    page?: string;
  }>();
  const [historyState, historyActions] = useHistory();
  const [taskState, taskActions] = useTasks();

  // Derived values from URL params
  const selectedTaskId = () => searchParams.task_id ?? '';
  const selectedJobId = () => searchParams.job_id ?? undefined;
  const levelFilter = () => searchParams.level ?? 'all';
  const currentPage = () => parseInt(searchParams.page ?? '1', 10);

  // Load tasks on mount
  onMount(() => {
    taskActions.loadTasks(params.connectionId);
  });

  // Load jobs when task is selected
  createEffect(() => {
    const taskId = selectedTaskId();
    if (taskId) {
      historyActions.loadJobs({
        connection_id: params.connectionId,
        task_id: taskId,
      });
    }
  });

  // Reload logs when filters or page changes
  createEffect(() => {
    const taskId = selectedTaskId();
    const jobId = selectedJobId();
    const level = levelFilter();
    const page = currentPage();

    historyActions.loadLogs({
      connection_id: params.connectionId,
      task_id: taskId,
      job_id: jobId,
      level: level === 'all' ? undefined : level,
      page,
    });
  });

  const setSelectedTaskId = (taskId: string) => {
    const currentTaskId = selectedTaskId();
    // Only clear job_id if task actually changed
    if (taskId === currentTaskId) return;

    setSearchParams({ task_id: taskId ?? undefined, job_id: undefined, page: '1' });
    historyActions.clearLogs();
  };

  const setSelectedJobId = (jobId: string | undefined) => {
    const currentJobId = selectedJobId();
    // Only update if job actually changed
    if (jobId === currentJobId) return;

    setSearchParams({ job_id: jobId ?? undefined, page: '1' });
    historyActions.clearLogs();
  };

  const setLevelFilter = (level: string) => {
    setSearchParams({ level: level === 'all' ? undefined : level, page: '1' });
  };

  const handleRefresh = () => {
    setSearchParams({ page: '1' });
    historyActions.loadLogs({
      connection_id: params.connectionId,
      task_id: selectedTaskId(),
      job_id: selectedJobId(),
      level: levelFilter() === 'all' ? undefined : levelFilter(),
      page: 1,
    });
  };

  const handlePageChange = (page: number) => {
    setSearchParams({ page: page.toString() });
  };

  const totalPages = createMemo(() =>
    Math.ceil(historyState.logsTotal / historyState.logsPageSize)
  );

  // Filter tasks by current connection
  const filteredTasks = createMemo(() => {
    return taskState.tasks.filter((t) => t.connection_id === params.connectionId);
  });

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'error':
        return <IconAlertCircle class="size-4 text-red-500" />;
      case 'warning':
        return <IconAlertTriangle class="size-4 text-yellow-500" />;
      case 'info':
        return <IconInfo class="size-4 text-blue-500" />;
      default:
        return <IconCheckCircle class="size-4 text-green-500" />;
    }
  };

  const getLevelBadge = (level: string) => {
    const variants: Record<
      string,
      'default' | 'secondary' | 'success' | 'warning' | 'error' | 'outline'
    > = {
      error: 'error',
      warning: 'warning',
      info: 'secondary',
    };
    return <Badge variant={variants[level] ?? 'outline'}>{level}</Badge>;
  };

  return (
    <ConnectionViewLayout
      title={m.log_syncLogs()}
      actions={
        <>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={historyState.isLoadingLogs}
          >
            <IconRefreshCw class="size-4" />
          </Button>
          <Select
            value={selectedTaskId()}
            onChange={(value) => setSelectedTaskId(value ?? '')}
            options={['', ...(filteredTasks().map((t) => t.id) ?? [])]}
            placeholder={m.log_selectTask()}
            itemComponent={(props) => (
              <SelectItem item={props.item}>
                {props.item.rawValue === ''
                  ? m.log_selectTask()
                  : (filteredTasks().find((t) => t.id === props.item.rawValue)?.name ??
                    props.item.rawValue)}
              </SelectItem>
            )}
          >
            <SelectTrigger class="w-[200px]">
              <SelectValue>
                {(state) => {
                  const taskId = state.selectedOption();
                  if (!taskId) return m.log_selectTask();
                  return filteredTasks().find((t) => t.id === taskId)?.name ?? m.log_selectTask();
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent />
          </Select>

          <Show when={selectedTaskId()}>
            <Select
              value={selectedJobId() ?? ''}
              onChange={(value) => setSelectedJobId(value ?? undefined)}
              options={['', ...(historyState.jobs.map((j) => j.id) ?? [])]}
              placeholder={m.log_selectExecution()}
              itemComponent={(props) => (
                <SelectItem item={props.item}>
                  {props.item.rawValue === ''
                    ? m.log_selectExecution()
                    : m.log_ranAgo({
                        time: formatRelativeTime(
                          historyState.jobs.find((j) => j.id === props.item.rawValue)?.start_time ??
                            ''
                        ),
                      })}
                </SelectItem>
              )}
            >
              <SelectTrigger class="w-[200px]">
                <SelectValue>
                  {(state) => {
                    const jobId = state.selectedOption() as string;
                    if (!jobId) return m.log_selectExecution();
                    const job = historyState.jobs.find((j) => j.id === jobId);
                    if (!job) return m.log_selectExecution();
                    return m.log_ranAgo({
                      time: formatRelativeTime(job.start_time),
                    });
                  }}
                </SelectValue>
              </SelectTrigger>
              <SelectContent />
            </Select>
          </Show>

          <Select
            value={levelFilter() ?? 'all'}
            onChange={(value) => setLevelFilter(value ?? 'all')}
            options={['all', 'info', 'warning', 'error']}
            placeholder={m.log_logLevel()}
            itemComponent={(props) => (
              <SelectItem item={props.item}>
                {props.item.rawValue === 'all' ? m.log_allLevels() : props.item.rawValue}
              </SelectItem>
            )}
          >
            <SelectTrigger class="w-[150px]">
              <SelectValue<string>>
                {(state) =>
                  (state.selectedOption() === 'all'
                    ? m.log_allLevels()
                    : state.selectedOption()) as string
                }
              </SelectValue>
            </SelectTrigger>
            <SelectContent />
          </Select>
        </>
      }
    >
      <Show
        when={historyState.logs.length > 0 || historyState.isLoadingLogs}
        fallback={
          <div class="flex flex-1 items-center justify-center text-muted-foreground">
            {m.log_noLogs()}
          </div>
        }
      >
        {/* Log Table */}
        <div class="relative min-h-0 flex-1 overflow-auto">
          <Table>
            <TableHeader class="sticky top-0 z-10 bg-card shadow-sm">
              <TableRow>
                <TableHead class="w-[100px] whitespace-nowrap">{m.common_level()}</TableHead>
                <TableHead class="w-[160px] whitespace-nowrap">{m.log_tableTime()}</TableHead>
                <TableHead class="w-[120px] whitespace-nowrap">{m.log_tableAction()}</TableHead>
                <TableHead class="w-[100px] whitespace-nowrap">{m.common_size()}</TableHead>
                <TableHead class="min-w-[300px]">{m.log_tablePath()}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <Show
                when={!historyState.isLoadingLogs}
                fallback={<TableSkeleton columns={5} rows={historyState.logsPageSize} />}
              >
                <For each={historyState.logs}>
                  {(log) => (
                    <TableRow>
                      <TableCell class="py-2 align-top">
                        <div class="flex items-center gap-2">
                          {getLevelIcon(log.level)}
                          {getLevelBadge(log.level)}
                        </div>
                      </TableCell>
                      <TableCell class="whitespace-nowrap py-2 align-top text-sm text-muted-foreground">
                        {formatRelativeTime(log.time)}
                      </TableCell>
                      <TableCell class="py-2 align-top text-sm">
                        <Badge variant="outline">{log.what}</Badge>
                      </TableCell>
                      <TableCell class="py-2 align-top text-sm text-muted-foreground">
                        <Show
                          when={log.size !== undefined && log.size > 0}
                          fallback={<span>-</span>}
                        >
                          {formatBytes(log.size)}
                        </Show>
                      </TableCell>
                      <TableCell class="py-2 align-top">
                        <Show
                          when={log.path}
                          fallback={<span class="text-muted-foreground">-</span>}
                        >
                          <div class="whitespace-pre-wrap break-all font-mono text-xs text-muted-foreground">
                            {log.path}
                          </div>
                        </Show>
                      </TableCell>
                    </TableRow>
                  )}
                </For>
              </Show>
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <Show when={totalPages() > 1}>
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

export default Log;
