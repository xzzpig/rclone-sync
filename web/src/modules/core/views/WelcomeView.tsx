import { Component, Show, createMemo } from 'solid-js';
import { createQuery } from '@urql/solid';
import { ConnectionsListQuery } from '@/api/graphql/queries/connections';
import { TasksListQuery } from '@/api/graphql/queries/tasks';
import { JobsListQuery } from '@/api/graphql/queries/jobs';
import * as m from '@/paraglide/messages.js';
import IconLink2 from '~icons/lucide/link-2';
import IconListTodo from '~icons/lucide/list-todo';
import IconCheckCircle2 from '~icons/lucide/check-circle-2';
import IconAlertCircle from '~icons/lucide/alert-circle';
import StatCard from '../components/StatCard';
import RecentActivity from '../components/RecentActivity';
import { Skeleton } from '@/components/ui/skeleton';

const WelcomeView: Component = () => {
  // Fetch all connections using GraphQL
  const [connectionsResult] = createQuery({
    query: ConnectionsListQuery,
    variables: {},
  });

  // Fetch all tasks using GraphQL
  const [tasksResult] = createQuery({
    query: TasksListQuery,
  });

  // Fetch recent job records using GraphQL (limit 10)
  const [jobsResult] = createQuery({
    query: JobsListQuery,
    variables: { pagination: { limit: 10 }, withConnection: true },
  });

  // Extract data from GraphQL results
  const connections = () => connectionsResult.data?.connection?.list?.items ?? [];
  const tasks = () => tasksResult.data?.task?.list?.items ?? [];
  const jobs = () => jobsResult.data?.job?.list?.items ?? [];

  // Calculate today's sync count
  const todaySyncCount = createMemo(() => {
    const jobList = jobs();
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    return jobList.filter((job) => {
      const jobDate = new Date(job.startTime);
      jobDate.setHours(0, 0, 0, 0);
      return (
        jobDate.getTime() === today.getTime() &&
        ['SUCCESS', 'FINISHED', 'DONE'].includes(job.status)
      );
    }).length;
  });

  // Calculate running and failed job counts
  const runningCount = createMemo(() => {
    const jobList = jobs();
    return jobList.filter((job) => ['RUNNING', 'PROCESSING', 'QUEUED'].includes(job.status)).length;
  });

  const failedCount = createMemo(() => {
    const jobList = jobs();
    return jobList.filter((job) => ['FAILED', 'ERROR'].includes(job.status)).length;
  });

  const isLoading = () => connectionsResult.fetching ?? tasksResult.fetching ?? jobsResult.fetching;

  return (
    <div class="h-full space-y-6 overflow-auto">
      {/* Welcome Header */}
      <div class="rounded-lg bg-card p-6 shadow-sm">
        <h1 class="text-3xl font-bold text-foreground">{m.welcome_title()}</h1>
        <p class="mt-2 text-muted-foreground">{m.welcome_subtitle()}</p>
      </div>

      {/* Statistics Cards */}
      <Show
        when={!isLoading()}
        fallback={
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Skeleton height={120} />
            <Skeleton height={120} />
            <Skeleton height={120} />
            <Skeleton height={120} />
          </div>
        }
      >
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <StatCard
            icon={<IconLink2 class="size-6" />}
            title={m.common_connections()}
            value={connections().length}
            color="blue"
          />
          <StatCard
            icon={<IconListTodo class="size-6" />}
            title={m.common_tasks()}
            value={tasks().length}
            color="green"
          />
          <StatCard
            icon={<IconCheckCircle2 class="size-6" />}
            title={m.statCard_todaysSyncs()}
            value={todaySyncCount()}
            description={m.status_completed()}
            color="green"
          />
          <StatCard
            icon={<IconAlertCircle class="size-6" />}
            title={m.statCard_attentionNeeded()}
            value={runningCount() + failedCount()}
            description={m.welcome_runningAndFailed({
              running: runningCount(),
              failed: failedCount(),
            })}
            color={failedCount() > 0 ? 'red' : 'orange'}
          />
        </div>
      </Show>

      {/* Recent Activity */}
      <Show when={!jobsResult.fetching} fallback={<Skeleton height={400} />}>
        <RecentActivity jobs={jobs()} />
      </Show>
    </div>
  );
};

export default WelcomeView;
