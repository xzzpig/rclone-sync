import { Component, Show, createMemo } from 'solid-js';
import { useQuery } from '@tanstack/solid-query';
import { getConnections } from '@/api/connections';
import { getTasks } from '@/api/tasks';
import { getJobs } from '@/api/history';
import IconLink2 from '~icons/lucide/link-2';
import IconListTodo from '~icons/lucide/list-todo';
import IconCheckCircle2 from '~icons/lucide/check-circle-2';
import IconAlertCircle from '~icons/lucide/alert-circle';
import StatCard from '../components/StatCard';
import RecentActivity from '../components/RecentActivity';
import { Skeleton } from '@/components/ui/skeleton';

const WelcomeView: Component = () => {
  // Fetch all connections
  const connectionsQuery = useQuery(() => ({
    queryKey: ['connections'],
    queryFn: getConnections,
  }));

  // Fetch all tasks
  const tasksQuery = useQuery(() => ({
    queryKey: ['tasks'],
    queryFn: () => getTasks(),
  }));

  // Fetch recent job records
  const jobsQuery = useQuery(() => ({
    queryKey: ['jobs', 'recent'],
    queryFn: () => getJobs({ limit: 10 }),
  }));

  // Calculate today's sync count
  const todaySyncCount = createMemo(() => {
    const jobs = jobsQuery.data?.data ?? [];
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    return jobs.filter((job) => {
      const jobDate = new Date(job.start_time);
      jobDate.setHours(0, 0, 0, 0);
      return (
        jobDate.getTime() === today.getTime() &&
        ['success', 'finished', 'done'].includes(job.status.toLowerCase())
      );
    }).length;
  });

  // Calculate running and failed job counts
  const runningCount = createMemo(() => {
    const jobs = jobsQuery.data?.data ?? [];
    return jobs.filter((job) =>
      ['running', 'processing', 'queued'].includes(job.status.toLowerCase())
    ).length;
  });

  const failedCount = createMemo(() => {
    const jobs = jobsQuery.data?.data ?? [];
    return jobs.filter((job) => ['failed', 'error'].includes(job.status.toLowerCase())).length;
  });

  const isLoading = () => connectionsQuery.isLoading || tasksQuery.isLoading || jobsQuery.isLoading;

  return (
    <div class="h-full space-y-6 overflow-auto">
      {/* <div class="max-w-7xl mx-auto space-y-6"> */}
      {/* Welcome Header */}
      <div class="rounded-lg bg-card p-6 shadow-sm">
        <h1 class="text-3xl font-bold text-foreground">Welcome to Cloud Sync</h1>
        <p class="mt-2 text-muted-foreground">Manage your cloud storage sync tasks</p>
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
            title="Connections"
            value={connectionsQuery.data?.length ?? 0}
            color="blue"
          />
          <StatCard
            icon={<IconListTodo class="size-6" />}
            title="Tasks"
            value={tasksQuery.data?.length ?? 0}
            color="green"
          />
          <StatCard
            icon={<IconCheckCircle2 class="size-6" />}
            title="Today's Syncs"
            value={todaySyncCount()}
            description="Completed"
            color="green"
          />
          <StatCard
            icon={<IconAlertCircle class="size-6" />}
            title="Attention Needed"
            value={runningCount() + failedCount()}
            description={`Running ${runningCount()} / Failed ${failedCount()}`}
            color={failedCount() > 0 ? 'red' : 'orange'}
          />
        </div>
      </Show>

      {/* Recent Activity */}
      <Show when={!jobsQuery.isLoading} fallback={<Skeleton height={400} />}>
        <RecentActivity jobs={jobsQuery.data?.data ?? []} />
      </Show>
      {/* </div> */}
    </div>
  );
};

export default WelcomeView;
