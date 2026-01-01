import { client } from '@/api/graphql/client';
import { ConnectionCreateMutation, ConnectionsListQuery } from '@/api/graphql/queries/connections';
import { ProviderGetQuery, ProvidersListQuery } from '@/api/graphql/queries/providers';
import { RichText } from '@/components/common/RichText';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { ProviderListItem } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createQuery } from '@urql/solid';
import { createEffect, createSignal, Show } from 'solid-js';
import { DynamicConfigForm } from './DynamicConfigForm';
import { ProviderSelection } from './ProviderSelection';

export const AddConnectionDialog = (props: { isOpen: boolean; onClose: () => void }) => {
  const [step, setStep] = createSignal(1);
  const [selectedProvider, setSelectedProvider] = createSignal<ProviderListItem | null>(null);

  // Use GraphQL query for providers
  const [providersResult, reexecuteProviders] = createQuery({
    query: ProvidersListQuery,
    pause: () => !props.isOpen,
  });

  // Refetch when dialog opens
  createEffect(() => {
    if (props.isOpen) {
      reexecuteProviders({ requestPolicy: 'cache-first' });
    }
  });

  const providers = () => providersResult.data?.provider?.list ?? [];

  // Use GraphQL query for provider options
  const [optionsResult] = createQuery({
    query: ProviderGetQuery,
    variables: () => ({ name: selectedProvider()?.name ?? '' }),
    pause: () => !selectedProvider(),
  });

  const options = () => optionsResult.data?.provider?.get?.options ?? [];

  const handleSelectProvider = (provider: ProviderListItem) => {
    setSelectedProvider(provider);
    setStep(2);
  };

  const handleBack = () => {
    setStep(1);
    setSelectedProvider(null);
  };

  const handleSave = async (name: string | undefined, config: Record<string, string>) => {
    if (!name) {
      throw new Error('Connection name is required');
    }

    const result = await client.mutation(ConnectionCreateMutation, {
      input: {
        name,
        type: selectedProvider()!.name,
        config,
      },
    });

    if (result.error) {
      throw new Error(result.error.message);
    }

    // Invalidate connections cache by reexecuting query
    await client.query(ConnectionsListQuery, {}, { requestPolicy: 'network-only' });

    // Close the dialog after successful save
    handleClose();
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
          <DialogTitle id="dialog-title">{m.wizard_addNewConnection()}</DialogTitle>
          <DialogDescription id="dialog-description">
            <Show when={step() === 1}>{m.wizard_step1Of2()}</Show>
            <Show when={step() === 2}>
              <RichText text={m.wizard_step2Of2({ provider: selectedProvider()?.name ?? '' })} />
            </Show>
          </DialogDescription>
        </DialogHeader>

        <Show when={step() === 1}>
          <ProviderSelection providers={providers()} onSelect={handleSelectProvider} />
        </Show>

        <Show when={step() === 2}>
          <div class="min-h-0 flex-1 p-4">
            <DynamicConfigForm
              loading={optionsResult.fetching}
              options={options()}
              provider={selectedProvider()!.name}
              onBack={handleBack}
              onSave={handleSave}
            />
          </div>
        </Show>
      </DialogContent>
    </Dialog>
  );
};
