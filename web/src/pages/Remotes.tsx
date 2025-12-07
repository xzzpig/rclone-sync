import { Component, createSignal, createResource, For, Show } from 'solid-js';
import { fetchRemotes, createRemote, deleteRemote, fetchProviders, fetchProviderOptions, Provider, RemoteInfo } from '../api/remotes';
import { FiPlus, FiTrash2, FiSettings } from 'solid-icons/fi';
import clsx from 'clsx';

const Remotes: Component = () => {
    const [remotes, { refetch: refetchRemotes }] = createResource(fetchRemotes);
    const [isCreating, setIsCreating] = createSignal(false);

    // Creation State
    const [step, setStep] = createSignal(1); // 1: Select Type, 2: Configure
    const [newRemoteName, setNewRemoteName] = createSignal('');
    const [selectedProvider, setSelectedProvider] = createSignal<string>('');
    const [configParams, setConfigParams] = createSignal<RemoteInfo>({});

    const [providers] = createResource(fetchProviders);
    const [providerOptions] = createResource(selectedProvider, fetchProviderOptions);

    const handleCreate = async () => {
        try {
            if (!newRemoteName() || !selectedProvider()) return;

            const config = {
                ...configParams(),
                type: selectedProvider()
            };

            await createRemote(newRemoteName(), config);
            await refetchRemotes();
            resetForm();
        } catch (e) {
            console.error(e);
            alert('Failed to create remote');
        }
    };

    const handleDelete = async (name: string) => {
        if (!confirm(`Are you sure you want to delete remote "${name}"?`)) return;
        try {
            await deleteRemote(name);
            await refetchRemotes();
        } catch (e) {
            console.error(e);
            alert('Failed to delete remote');
        }
    };

    const resetForm = () => {
        setIsCreating(false);
        setStep(1);
        setNewRemoteName('');
        setSelectedProvider('');
        setConfigParams({});
    };

    return (
        <div>
            <div class="flex justify-between items-center mb-6">
                <h1 class="text-2xl font-bold text-gray-800">Remotes</h1>
                <button
                    onClick={() => setIsCreating(true)}
                    class="flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                >
                    <FiPlus class="mr-2" />
                    New Remote
                </button>
            </div>

            <Show when={isCreating()}>
                <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto p-6">
                        <h2 class="text-xl font-bold mb-4">
                            {step() === 1 ? 'Select Storage Provider' : 'Configure Remote'}
                        </h2>

                        <Show when={step() === 1}>
                            <div class="mb-4">
                                <label class="block text-sm font-medium text-gray-700 mb-1">Remote Name</label>
                                <input
                                    type="text"
                                    value={newRemoteName()}
                                    onInput={(e) => setNewRemoteName(e.currentTarget.value)}
                                    class="w-full px-3 py-2 border rounded-md focus:ring-blue-500 focus:border-blue-500"
                                    placeholder="e.g., my-drive"
                                />
                            </div>

                            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 max-h-96 overflow-y-auto mb-6">
                                <For each={providers()}>
                                    {(provider) => (
                                        <button
                                            onClick={() => {
                                                setSelectedProvider(provider.name);
                                                setStep(2);
                                            }}
                                            class={clsx(
                                                "p-4 border rounded-lg text-left hover:bg-blue-50 hover:border-blue-300 transition-all",
                                                selectedProvider() === provider.name && "ring-2 ring-blue-500 border-transparent"
                                            )}
                                        >
                                            <div class="font-bold text-gray-900">{provider.description}</div>
                                            <div class="text-xs text-gray-500 font-mono mt-1">{provider.name}</div>
                                        </button>
                                    )}
                                </For>
                            </div>
                        </Show>

                        <Show when={step() === 2}>
                            <div class="mb-6 space-y-4">
                                <Show when={providerOptions.loading}>
                                    <p>Loading configuration options...</p>
                                </Show>

                                <For each={providerOptions()?.options}>
                                    {(opt: any) => (
                                        <div>
                                            <label class="block text-sm font-medium text-gray-700 mb-1">
                                                {opt.Name} <span class="text-gray-400 font-normal">({opt.Type})</span>
                                            </label>
                                            <p class="text-xs text-gray-500 mb-1">{opt.Help}</p>

                                            {opt.Type === 'bool' ? (
                                                <select
                                                    onChange={(e) => setConfigParams({ ...configParams(), [opt.Name]: e.currentTarget.value })}
                                                    class="w-full px-3 py-2 border rounded-md"
                                                >
                                                    <option value="false">False</option>
                                                    <option value="true">True</option>
                                                </select>
                                            ) : (
                                                <input
                                                    type={opt.Type === 'password' ? 'password' : 'text'}
                                                    onInput={(e) => setConfigParams({ ...configParams(), [opt.Name]: e.currentTarget.value })}
                                                    class="w-full px-3 py-2 border rounded-md"
                                                    placeholder={opt.Examples?.[0]?.Value || ''}
                                                />
                                            )}
                                        </div>
                                    )}
                                </For>
                            </div>
                        </Show>

                        <div class="flex justify-end space-x-3">
                            <button
                                onClick={resetForm}
                                class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-md"
                            >
                                Cancel
                            </button>

                            <Show when={step() === 2}>
                                <button
                                    onClick={() => setStep(1)}
                                    class="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-md"
                                >
                                    Back
                                </button>
                                <button
                                    onClick={handleCreate}
                                    class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                                >
                                    Create Remote
                                </button>
                            </Show>
                        </div>
                    </div>
                </div>
            </Show>

            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                <For each={remotes()} fallback={<p class="text-gray-500 col-span-full">No remotes configured yet.</p>}>
                    {(remoteName) => (
                        <div class="bg-white p-6 rounded-lg shadow-sm border hover:shadow-md transition-shadow">
                            <div class="flex justify-between items-start mb-4">
                                <div class="flex items-center">
                                    <div class="p-2 bg-blue-100 rounded-lg text-blue-600 mr-3">
                                        <FiSettings class="w-5 h-5" />
                                    </div>
                                    <h3 class="text-lg font-bold text-gray-800">{remoteName}</h3>
                                </div>
                                <button
                                    onClick={() => handleDelete(remoteName)}
                                    class="text-gray-400 hover:text-red-500 transition-colors"
                                >
                                    <FiTrash2 class="w-5 h-5" />
                                </button>
                            </div>
                            <div class="text-sm text-gray-500 mt-2">
                                <span class="font-medium">Type:</span>
                                {/* We would fetch detailed info here if needed, or pass it from list */}
                                Configured
                            </div>
                        </div>
                    )}
                </For>
            </div>
        </div>
    );
};

export default Remotes;
