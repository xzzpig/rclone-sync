import IntervalUpdated from '@/components/common/IntervalUpdated';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { formatRelativeTime } from '@/lib/date';
import type { JobStatus } from '@/lib/types';
import { formatBytes } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useJobProgress } from '@/store/jobProgress';
import { useTasks } from '@/store/tasks';
import { useNavigate } from '@solidjs/router';
import {
  Accessor,
  Component,
  createEffect,
  createMemo,
  createSignal,
  Index,
  onCleanup,
  Show,
} from 'solid-js';
import IconLoader from '~icons/lucide/loader';

interface RunningJobsCardProps {
  connectionId: Accessor<string | undefined>;
}

/**
 * Running job entry with progress data
 */
interface RunningJob {
  jobId: string;
  taskId: string;
  status: JobStatus;
  startTime: string;
  filesTransferred: number;
  filesTotal: number;
  bytesTransferred: number;
  bytesTotal: number;
}

const RunningJobsCard: Component<RunningJobsCardProps> = (props) => {
  const navigate = useNavigate();
  const [taskState] = useTasks();
  const jobProgress = useJobProgress();

  // Maintain local state for running jobs
  const [runningJobsMap, setRunningJobsMap] = createSignal<Map<string, RunningJob>>(new Map());

  // Create task name map for lookup
  const taskNameMap = createMemo(() => {
    const map = new Map<string, string>();
    taskState.tasks.forEach((task) => {
      map.set(task.id, task.name);
    });
    return map;
  });

  // Subscribe to job progress events using the global store
  // Use createEffect to react to connectionId changes
  createEffect(() => {
    const connectionId = props.connectionId();
    // Clear running jobs when connectionId changes
    setRunningJobsMap(new Map());

    const subscription = jobProgress.subscribe(
      (event) => {
        setRunningJobsMap((prev) => {
          const newMap = new Map(prev);

          if (event.status === 'RUNNING') {
            // Update or add running job
            newMap.set(event.jobId, {
              jobId: event.jobId,
              taskId: event.taskId,
              status: event.status,
              startTime: event.startTime,
              filesTransferred: event.filesTransferred,
              filesTotal: event.filesTotal,
              bytesTransferred: event.bytesTransferred,
              bytesTotal: event.bytesTotal,
            });
          } else {
            // Job completed, remove it
            newMap.delete(event.jobId);
          }

          return newMap;
        });
      },
      // Filter by connectionId - only receive events for this connection
      { connectionId }
    );

    onCleanup(() => {
      subscription.unsubscribe();
    });
  });

  // Clear running jobs on WebSocket reconnection
  const unsubscribeReconnect = jobProgress.onReconnect(() => {
    console.info('WebSocket reconnected, clearing running jobs state');
    setRunningJobsMap(new Map());
  });

  onCleanup(() => {
    unsubscribeReconnect();
  });

  // Convert map to array for rendering
  const runningJobs = createMemo(() => Array.from(runningJobsMap().values()));

  // Get task name by taskId
  const getTaskName = (taskId: string) => {
    return taskNameMap().get(taskId) ?? m.error_taskNotFound();
  };

  // Handle click to navigate to log page with task filter
  const handleJobClick = (job: RunningJob) => {
    const queryParams = new URLSearchParams();
    queryParams.set('task_id', job.taskId);
    queryParams.set('job_id', job.jobId);
    navigate(`/connections/${props.connectionId()}/log?${queryParams.toString()}`);
  };

  // Calculate progress percentage
  const calcPercent = (bytes: number, total: number) => {
    if (total <= 0) return 0;
    return Math.min(100, (bytes / total) * 100);
  };

  // Hide card when no running jobs
  return (
    <Show when={runningJobs().length > 0}>
      <Card>
        <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle class="text-sm font-medium">{m.overview_runningJobs()}</CardTitle>
          <IconLoader class="size-4 animate-spin text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div>
            <Index each={runningJobs()}>
              {(job, idx) => {
                const percent = () => calcPercent(job().bytesTransferred, job().bytesTotal);
                const taskName = () => getTaskName(job().taskId);
                return (
                  <>
                    <Show when={idx > 0}>
                      <Separator class="my-1" />
                    </Show>
                    <div
                      class="cursor-pointer space-y-2 transition-colors hover:bg-muted/50"
                      onClick={() => handleJobClick(job())}
                    >
                      {/* Header: Task name and status */}
                      <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                          <span class="font-medium" title={taskName()}>
                            {taskName()}
                          </span>
                        </div>
                        <span class="text-xs text-muted-foreground">
                          <IntervalUpdated when={true} interval={60 * 1000}>
                            {() => formatRelativeTime(job().startTime)}
                          </IntervalUpdated>
                        </span>
                      </div>

                      {/* Progress info */}
                      <div class="flex items-center justify-between text-xs text-muted-foreground">
                        <span>
                          {job().filesTotal > 0
                            ? `${job().filesTransferred}/${job().filesTotal} ${m.common_files().toLowerCase()}`
                            : `${job().filesTransferred} ${m.common_files().toLowerCase()}`}
                        </span>
                        <span>
                          {job().bytesTotal > 0
                            ? `${formatBytes(job().bytesTransferred)} / ${formatBytes(job().bytesTotal)}`
                            : formatBytes(job().bytesTransferred)}
                        </span>
                      </div>

                      {/* Progress bar */}
                      <div class="flex items-center gap-2">
                        <Progress
                          value={percent()}
                          minValue={0}
                          maxValue={100}
                          class="h-1.5 flex-1"
                        />
                        <span class="w-10 text-right text-xs text-muted-foreground">
                          {percent().toFixed(0)}%
                        </span>
                      </div>
                    </div>
                  </>
                );
              }}
            </Index>
          </div>
        </CardContent>
      </Card>
    </Show>
  );
};

export default RunningJobsCard;
