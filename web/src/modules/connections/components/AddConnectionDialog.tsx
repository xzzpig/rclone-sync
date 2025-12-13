import { getProviderOptions, getProviders } from '@/api/connections';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { RcloneProvider } from '@/lib/types';
import { useQuery } from '@tanstack/solid-query';
import { createSignal, Show } from 'solid-js';
import { DynamicConfigForm } from './DynamicConfigForm';
import { ProviderSelection } from './ProviderSelection';

export const AddConnectionDialog = (props: { isOpen: boolean; onClose: () => void }) => {
  const [step, setStep] = createSignal(1);
  const [selectedProvider, setSelectedProvider] = createSignal<RcloneProvider | null>(null);

  const providersQuery = useQuery(() => ({
    queryKey: ['providers'],
    queryFn: getProviders,
    enabled: props.isOpen,
  }));

  const optionsQuery = useQuery(() => ({
    queryKey: ['providerOptions', selectedProvider()?.name],
    queryFn: () => getProviderOptions(selectedProvider()!.name),
    enabled: !!selectedProvider(),
  }));

  const handleSelectProvider = (provider: RcloneProvider) => {
    setSelectedProvider(provider);
    setStep(2);
  };

  const handleBack = () => {
    setStep(1);
    setSelectedProvider(null);
  };

  const handleClose = () => {
    setStep(1);
    setSelectedProvider(null);
    props.onClose();
  };

  return (
    <Dialog open={props.isOpen} onOpenChange={handleClose}>
      <DialogContent class="flex max-h-[90vh] max-w-2xl flex-col overflow-y-auto">
        <DialogHeader>
          <DialogTitle id="dialog-title">Add New Connection</DialogTitle>
          <DialogDescription id="dialog-description">
            <Show when={step() === 1}>Step 1 of 2: Select a cloud provider.</Show>
            <Show when={step() === 2}>
              Step 2 of 2: Configure your{' '}
              <span class="font-bold text-primary">{selectedProvider()?.name}</span> connection.
            </Show>
          </DialogDescription>
        </DialogHeader>

        <Show when={step() === 1}>
          <ProviderSelection
            providers={providersQuery.data ?? []}
            onSelect={handleSelectProvider}
          />
        </Show>

        <Show when={step() === 2}>
          <div class="min-h-0 flex-1 p-4">
            <DynamicConfigForm
              loading={optionsQuery.isLoading}
              options={optionsQuery.data ?? []}
              provider={selectedProvider()!.name}
              onBack={handleBack}
              onSave={handleClose}
            />
          </div>
        </Show>
      </DialogContent>
    </Dialog>
  );
};
