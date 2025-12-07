import { Component, createResource, Show, For } from "solid-js";
import { useParams } from "@solidjs/router";
import { getJob } from "../api/jobs";

const JobDetails: Component = () => {
    const params = useParams();
    const [job] = createResource(() => params.id, getJob);

    return (
        <div class="p-6">
            <div class="mb-6">
                <a href="/" class="text-blue-600 hover:underline">← Back to Dashboard</a>
            </div>

            <Show when={job()} fallback={<div class="p-4">Loading job details...</div>}>
                <div class="bg-white rounded shadow overflow-hidden mb-6">
                    <div class="px-6 py-4 border-b border-gray-200">
                        <h1 class="text-xl font-bold text-gray-800">Job Details</h1>
                    </div>
                    <div class="p-6 grid grid-cols-2 gap-4">
                        <div>
                            <span class="block text-sm font-medium text-gray-500">ID</span>
                            <span class="block text-gray-900">{job()?.id}</span>
                        </div>
                        <div>
                            <span class="block text-sm font-medium text-gray-500">Task Name</span>
                            <span class="block text-gray-900">{job()?.edges?.task?.name || 'Unknown'}</span>
                        </div>
                        <div>
                            <span class="block text-sm font-medium text-gray-500">Status</span>
                            <span class={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                ${job()?.status === 'success' ? 'bg-green-100 text-green-800' :
                                    job()?.status === 'failed' ? 'bg-red-100 text-red-800' :
                                        job()?.status === 'running' ? 'bg-blue-100 text-blue-800' : 'bg-gray-100 text-gray-800'}`}>
                                {job()?.status}
                            </span>
                        </div>
                        <div>
                            <span class="block text-sm font-medium text-gray-500">Start Time</span>
                            <span class="block text-gray-900">{new Date(job()?.start_time!).toLocaleString()}</span>
                        </div>
                        <div class="col-span-2">
                            <span class="block text-sm font-medium text-gray-500">Summary</span>
                            <span class="block text-gray-900">
                                Transferred {job()?.files_transferred} files ({job()?.bytes_transferred} bytes)
                            </span>
                        </div>
                        <Show when={job()?.errors}>
                            <div class="col-span-2 bg-red-50 p-4 rounded border border-red-200">
                                <span class="block text-sm font-medium text-red-700">Errors</span>
                                <pre class="text-red-600 whitespace-pre-wrap text-sm">{job()?.errors}</pre>
                            </div>
                        </Show>
                    </div>
                </div>

                <div class="bg-white rounded shadow overflow-hidden">
                    <div class="px-6 py-4 border-b border-gray-200">
                        <h2 class="text-lg font-semibold text-gray-800">Logs</h2>
                    </div>
                    <div class="bg-gray-900 p-4 font-mono text-sm h-96 overflow-y-auto">
                        <For each={job()?.edges?.logs}>
                            {(log) => (
                                <div class="mb-1">
                                    <span class="text-gray-500 mr-2">[{new Date(log.time).toLocaleTimeString()}]</span>
                                    <span class={`${log.level === 'ERROR' ? 'text-red-400' : 'text-gray-300'}`}>
                                        {log.level}: {log.message}
                                    </span>
                                    <Show when={log.path}>
                                        <span class="text-gray-600 ml-2">({log.path})</span>
                                    </Show>
                                </div>
                            )}
                        </For>
                        <Show when={!job()?.edges?.logs || job()?.edges?.logs.length === 0}>
                            <div class="text-gray-600 italic">No logs available for this job.</div>
                        </Show>
                    </div>
                </div>
            </Show>
        </div>
    );
};

export default JobDetails;
