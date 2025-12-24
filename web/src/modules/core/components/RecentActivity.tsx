import StatusIcon from '@/components/common/StatusIcon';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatRelativeTime } from '@/lib/date';
import type { JobListItem, JobStatus } from '@/lib/types';
import { formatBytes } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { useNavigate } from '@solidjs/router';
import { Component, For, Show } from 'solid-js';
import IconClock from '~icons/lucide/clock';

interface RecentActivityProps {
  jobs: JobListItem[];
}

const RecentActivity: Component<RecentActivityProps> = (props) => {
  const navigate = useNavigate();

  const handleJobClick = (job: JobListItem) => {
    const taskId = job.task?.id;
    const connectionId = job.task?.connection?.id;
    const jobId = job.id;

    if (connectionId && taskId && jobId) {
      // Navigate to Log page with task and job pre-selected
      navigate(`/connections/${connectionId}/log?task_id=${taskId}&job_id=${jobId}`);
    }
  };

  const getStatusText = (status: string) => {
    switch (status.toUpperCase()) {
      case 'SUCCESS':
        return m.status_completed();
      case 'FAILED':
        return m.status_failed();
      case 'RUNNING':
        return m.status_running();
      case 'CANCELLED':
        return m.task_status_cancelled();
      case 'PENDING':
        return m.status_idle();
      default:
        return status;
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{m.recentActivity_title()}</CardTitle>
      </CardHeader>
      <CardContent>
        <Show
          when={props.jobs.length > 0}
          fallback={
            <div class="py-8 text-center text-muted-foreground">
              <IconClock class="mx-auto mb-2 size-12 text-muted-foreground" />
              <p>{m.recentActivity_noActivity()}</p>
            </div>
          }
        >
          <div class="space-y-3">
            <For each={props.jobs}>
              {(job) => (
                <div
                  class="flex cursor-pointer items-start gap-3 rounded-lg p-3 transition-colors hover:bg-accent"
                  onClick={() => handleJobClick(job)}
                >
                  <div class="mt-0.5">
                    <StatusIcon status={job.status as JobStatus} class="size-5" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex items-start justify-between gap-2">
                      <div class="flex-1">
                        <p class="truncate font-medium text-foreground">
                          {job.task?.name ?? 'Unnamed Task'}
                        </p>
                        <p class="text-sm text-muted-foreground">
                          {job.task?.connection?.name ?? 'Unknown Connection'}
                        </p>
                      </div>
                      <span class="whitespace-nowrap text-xs text-muted-foreground">
                        {formatRelativeTime(job.startTime)}
                      </span>
                    </div>
                    <div class="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{getStatusText(job.status)}</span>
                      <Show when={job.filesTransferred > 0}>
                        <span>•</span>
                        <span>{job.filesTransferred} files</span>
                      </Show>
                      <Show when={job.bytesTransferred > 0}>
                        <span>•</span>
                        <span>{formatBytes(job.bytesTransferred)}</span>
                      </Show>
                    </div>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>
      </CardContent>
    </Card>
  );
};

export default RecentActivity;
