import { Component, createSignal, onCleanup, For, Show } from "solid-js";
import { createResource } from "solid-js";
import { getJobs, Job } from "../api/jobs";
import { API_BASE } from "../api/config";

const Dashboard: Component = () => {
    const [jobs, { refetch }] = createResource(async () => await getJobs(10, 0));
    const [stats, setStats] = createSignal<Record<string, { files: number, bytes: number }>>({});

    // SSE Connection for realtime updates
    const evtSource = new EventSource(`${API_BASE}/events`);

    evtSource.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'job_progress') {
                setStats(prev => ({
                    ...prev,
                    [data.data.job_id]: {
                        files: data.data.files_transferred,
                        bytes: data.data.bytes_transferred
                    }
                }));
            }
        } catch (e) {
            console.error("Failed to parse SSE event", e);
        }
    };

    onCleanup(() => {
        evtSource.close();
    });

    const formatBytes = (bytes: number) => {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    return (
        <div class="p-6">
            <h1 class="text-2xl font-bold mb-6">Dashboard</h1>

            {/* Recent Jobs */}
            <div class="bg-white rounded shadow overflow-hidden">
                <div class="px-6 py-4 border-b border-gray-200">
                    <h2 class="text-lg font-semibold text-gray-800">Recent Activity</h2>
                </div>
                <table class="min-w-full">
                    <thead>
                        <tr class="bg-gray-50 border-b">
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Task</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Start Time</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Trigger</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Files</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Size</th>
                            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-200">
                        <For each={jobs()}>
                            {(job) => {
                                const liveStats = stats()[job.id];
                                const files = liveStats ? liveStats.files : job.files_transferred;
                                const bytes = liveStats ? liveStats.bytes : job.bytes_transferred;

                                return (
                                    <tr>
                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <span class={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                                ${job.status === 'success' ? 'bg-green-100 text-green-800' :
                                                    job.status === 'failed' ? 'bg-red-100 text-red-800' :
                                                        job.status === 'running' ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-800'}`}>
                                                {job.status}
                                            </span>
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{job.edges?.task?.name || 'Unknown'}</td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(job.start_time).toLocaleString()}</td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{job.trigger}</td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{files}</td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{formatBytes(bytes)}</td>
                                        <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                            <a href={`/jobs/${job.id}`} class="text-indigo-600 hover:text-indigo-900">Details</a>
                                        </td>
                                    </tr>
                                );
                            }}
                        </For>
                        <Show when={!jobs() || jobs()?.length === 0}>
                            <tr>
                                <td colSpan={7} class="px-6 py-4 text-center text-sm text-gray-500">
                                    No recent activity found.
                                </td>
                            </tr>
                        </Show>
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default Dashboard;
