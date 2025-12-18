import { getProviderOptions } from '@/api/connections';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { ImportPreviewItem } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { createQuery } from '@tanstack/solid-query';
import { Component, Show } from 'solid-js';
import { DynamicConfigForm } from '../DynamicConfigForm';

interface EditImportConfigDialogProps {
  isOpen: boolean;
  item: ImportPreviewItem | null;
  onClose: () => void;
  onSave: (config: Record<string, string>) => void;
}

export const EditImportConfigDialog: Component<EditImportConfigDialogProps> = (props) => {
  // Query provider options when dialog opens and item type is available
  const providerOptionsQuery = createQuery(() => ({
    queryKey: ['providers', props.item?.type],
    queryFn: () => getProviderOptions(props.item!.type),
    enabled: props.isOpen && !!props.item?.type,
  }));

  const handleSave = async (_name: string | undefined, config: Record<string, string>) => {
    props.onSave(config);
  };

  const handleClose = () => {
    props.onClose();
  };

  return (
    <Dialog open={props.isOpen} onOpenChange={handleClose}>
      <DialogContent class="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{m.import_editConnectionConfig()}</DialogTitle>
          <DialogDescription>
            <Show when={props.item}>
              {(item) => (
                <span>
                  {item().editedName ?? item().name} ({item().type})
                </span>
              )}
            </Show>
          </DialogDescription>
        </DialogHeader>

        <Show
          when={props.item}
          fallback={
            <div class="p-4 text-center text-muted-foreground">
              {m.import_noConnectionSelected()}
            </div>
          }
        >
          {(item) => (
            <Show
              when={!providerOptionsQuery.isLoading && !providerOptionsQuery.isError}
              fallback={
                <div class="space-y-4 p-4">
                  <Show when={providerOptionsQuery.isLoading}>
                    <div class="text-center text-muted-foreground">{m.import_loadingOptions()}</div>
                  </Show>
                  <Show when={providerOptionsQuery.isError}>
                    <div class="space-y-2 text-center">
                      <div class="text-error-foreground">
                        {m.import_loadOptionsFailed({ error: String(providerOptionsQuery.error) })}:
                      </div>
                      <Button variant="outline" onClick={() => providerOptionsQuery.refetch()}>
                        {m.common_retry()}
                      </Button>
                    </div>
                  </Show>
                </div>
              }
            >
              <DynamicConfigForm
                options={providerOptionsQuery.data ?? []}
                provider={item().type}
                initialValues={item().editedConfig ?? item().config}
                onBack={handleClose}
                onSave={handleSave}
                hideName
                saveButtonText={m.common_save()}
                showBack
                loading={providerOptionsQuery.isLoading}
              />
            </Show>
          )}
        </Show>
      </DialogContent>
    </Dialog>
  );
};
