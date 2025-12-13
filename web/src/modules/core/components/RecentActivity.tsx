import StatusIcon from '@/components/common/StatusIcon';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Job } from '@/lib/types';
import { formatBytes } from '@/lib/utils';
import { useNavigate } from '@solidjs/router';
import { formatDistanceToNow } from 'date-fns';
import { Component, For, Show } from 'solid-js';
import IconClock from '~icons/lucide/clock';

interface RecentActivityProps {
  jobs: Job[];
}

const RecentActivity: Component<RecentActivityProps> = (props) => {
  const navigate = useNavigate();

  const handleJobClick = (job: Job) => {
    const taskId = job.edges?.task?.id;
    const remoteName = job.edges?.task?.remote_name;
    const jobId = job.id;

    if (remoteName && taskId && jobId) {
      // Navigate to Log page with task and job pre-selected
      navigate(`/connections/${remoteName}/log?task_id=${taskId}&job_id=${jobId}`);
    }
  };

  const getStatusText = (status: string) => {
    switch (status.toLowerCase()) {
      case 'success':
      case 'finished':
      case 'done':
        return 'Success';
      case 'failed':
      case 'error':
        return 'Failed';
      case 'running':
      case 'processing':
        return 'Running';
      case 'queued':
        return 'Queued';
      default:
        return status;
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent Activity</CardTitle>
      </CardHeader>
      <CardContent>
        <Show
          when={props.jobs.length > 0}
          fallback={
            <div class="py-8 text-center text-muted-foreground">
              <IconClock class="mx-auto mb-2 size-12 text-muted-foreground" />
              <p>No recent activity</p>
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
                    <StatusIcon status={job.status} class="size-5" />
                    {/* {getStatusIcon(job.status)} */}
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="flex items-start justify-between gap-2">
                      <div class="flex-1">
                        <p class="truncate font-medium text-foreground">
                          {job.edges?.task?.name ?? 'Unnamed Task'}
                        </p>
                        <p class="text-sm text-muted-foreground">
                          {job.edges?.task?.remote_name ?? 'Unknown Connection'}
                        </p>
                      </div>
                      <span class="whitespace-nowrap text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(job.start_time), { addSuffix: true })}
                      </span>
                    </div>
                    <div class="mt-1 flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{getStatusText(job.status)}</span>
                      <Show when={job.files_transferred > 0}>
                        <span>•</span>
                        <span>{job.files_transferred} files</span>
                      </Show>
                      <Show when={job.bytes_transferred > 0}>
                        <span>•</span>
                        <span>{formatBytes(job.bytes_transferred)}</span>
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
