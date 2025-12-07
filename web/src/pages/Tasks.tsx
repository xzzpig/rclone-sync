import { Component, createSignal, For, Show } from "solid-js";
import { CreateTaskRequest, createTask, deleteTask, getTasks, Task, runTask } from "../api/tasks";
import { createResource } from "solid-js";
import { fetchRemotes } from "../api/remotes";

const Tasks: Component = () => {
    const [tasks, { refetch }] = createResource(getTasks);
    const [remotes] = createResource<string[]>(fetchRemotes);
    const [isCreating, setIsCreating] = createSignal(false);
    const [newTask, setNewTask] = createSignal<CreateTaskRequest>({
        name: "",
        source_path: "",
        remote_name: "",
        remote_path: "",
        direction: "bidirectional",
        realtime: false,
        schedule: "",
    });

    const handleCreate = async (e: Event) => {
        e.preventDefault();
        try {
            await createTask(newTask());
            setIsCreating(false);
            refetch();
            setNewTask({
                name: "",
                source_path: "",
                remote_name: "",
                remote_path: "",
                direction: "bidirectional",
                realtime: false,
                schedule: "",
            });
        } catch (err) {
            console.error(err);
            alert("Failed to create task");
        }
    };

    const handleDelete = async (id: string) => {
        if (!confirm("Are you sure you want to delete this task?")) return;
        try {
            await deleteTask(id);
            refetch();
        } catch (err) {
            console.error(err);
            alert("Failed to delete task");
        }
    };

    const handleRun = async (id: string, name: string) => {
        try {
            await runTask(id);
            alert(`Task "${name}" started successfully!`);
        } catch (err: any) {
            console.error(err);
            alert(`Failed to start task: ${err.message}`);
        }
    };

    return (
        <div class="p-6">
            <div class="flex justify-between items-center mb-6">
                <h1 class="text-2xl font-bold">Sync Tasks</h1>
                <button
                    class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
                    onClick={() => setIsCreating(true)}
                >
                    New Task
                </button>
            </div>

            <Show when={isCreating()}>
                <div class="bg-white p-6 rounded shadow mb-6">
                    <h2 class="text-lg font-semibold mb-4">Create New Task</h2>
                    <form onSubmit={handleCreate}>
                        <div class="grid grid-cols-2 gap-4">
                            <div>
                                <label class="block text-sm font-medium mb-1">Task Name</label>
                                <input
                                    type="text"
                                    required
                                    class="w-full p-2 border rounded"
                                    value={newTask().name}
                                    onInput={(e) =>
                                        setNewTask({ ...newTask(), name: e.currentTarget.value })
                                    }
                                />
                            </div>
                            <div>
                                <label class="block text-sm font-medium mb-1">Source Path (Local)</label>
                                <input
                                    type="text"
                                    required
                                    class="w-full p-2 border rounded"
                                    value={newTask().source_path}
                                    onInput={(e) =>
                                        setNewTask({ ...newTask(), source_path: e.currentTarget.value })
                                    }
                                />
                            </div>
                            <div>
                                <label class="block text-sm font-medium mb-1">Remote</label>
                                <select
                                    required
                                    class="w-full p-2 border rounded"
                                    value={newTask().remote_name}
                                    onChange={(e) =>
                                        setNewTask({ ...newTask(), remote_name: e.currentTarget.value })
                                    }
                                >
                                    <option value="">Select Remote</option>
                                    <For each={remotes() || []}>
                                        {(remote) => <option value={remote}>{remote}</option>}
                                    </For>
                                </select>
                            </div>
                            <div>
                                <label class="block text-sm font-medium mb-1">Remote Path</label>
                                <input
                                    type="text"
                                    required
                                    class="w-full p-2 border rounded"
                                    value={newTask().remote_path}
                                    onInput={(e) =>
                                        setNewTask({ ...newTask(), remote_path: e.currentTarget.value })
                                    }
                                />
                            </div>
                            <div>
                                <label class="block text-sm font-medium mb-1">Direction</label>
                                <select
                                    class="w-full p-2 border rounded"
                                    value={newTask().direction}
                                    onChange={(e) =>
                                        setNewTask({
                                            ...newTask(),
                                            direction: e.currentTarget.value as any,
                                        })
                                    }
                                >
                                    <option value="bidirectional">Bidirectional</option>
                                    <option value="upload">Upload (Local to Remote)</option>
                                    <option value="download">Download (Remote to Local)</option>
                                </select>
                            </div>
                            <div>
                                <label class="block text-sm font-medium mb-1">Schedule (Cron)</label>
                                <input
                                    type="text"
                                    class="w-full p-2 border rounded"
                                    placeholder="e.g. @daily or * */1 * * *"
                                    value={newTask().schedule}
                                    onInput={(e) =>
                                        setNewTask({ ...newTask(), schedule: e.currentTarget.value })
                                    }
                                />
                            </div>
                        </div>
                        <div class="mt-4">
                            <label class="flex items-center space-x-2">
                                <input
                                    type="checkbox"
                                    checked={newTask().realtime}
                                    onChange={(e) => setNewTask({ ...newTask(), realtime: e.currentTarget.checked })}
                                />
                                <span>Enable Realtime Sync (Watch)</span>
                            </label>
                        </div>
                        <div class="mt-6 flex justify-end space-x-2">
                            <button
                                type="button"
                                class="bg-gray-200 hover:bg-gray-300 px-4 py-2 rounded"
                                onClick={() => setIsCreating(false)}
                            >
                                Cancel
                            </button>
                            <button
                                type="submit"
                                class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
                            >
                                Create
                            </button>
                        </div>
                    </form>
                </div>
            </Show>

            <div class="bg-white rounded shadow overflow-hidden">
                <table class="min-w-full">
                    <thead>
                        <tr class="bg-gray-50 border-b">
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Name
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Source
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Destination
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Direction
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Schedule
                            </th>
                            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                                Actions
                            </th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-200">
                        <For each={tasks()}>
                            {(task) => (
                                <tr>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                        {task.name}
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {task.source_path}
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {task.remote_name}:{task.remote_path}
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        <span
                                            class={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                        ${task.direction === 'bidirectional' ? 'bg-purple-100 text-purple-800' :
                                                    task.direction === 'upload' ? 'bg-green-100 text-green-800' : 'bg-blue-100 text-blue-800'}`}
                                        >
                                            {task.direction}
                                        </span>
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                        {task.realtime ? <span class="bg-yellow-100 text-yellow-800 text-xs px-2 rounded-full mr-1">Watch</span> : null}
                                        {task.schedule || "Manual"}
                                    </td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                        <button
                                            class="text-indigo-600 hover:text-indigo-900"
                                            onClick={() => handleRun(task.id, task.name)}
                                        >
                                            Run Now
                                        </button>
                                        <button
                                            class="text-red-600 hover:text-red-900 ml-4"
                                            onClick={() => handleDelete(task.id)}
                                        >
                                            Delete
                                        </button>
                                    </td>
                                </tr>
                            )}
                        </For>
                        <Show when={!tasks() || tasks()?.length === 0}>
                            <tr>
                                <td colSpan={6} class="px-6 py-4 text-center text-sm text-gray-500">
                                    No tasks found. Create one to get started.
                                </td>
                            </tr>
                        </Show>
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default Tasks;
