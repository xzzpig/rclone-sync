import { client } from '@/api/graphql/client';
import { ConnectionGetBasicQuery } from '@/api/graphql/queries/connections';
import { FilesListQuery } from '@/api/graphql/queries/files';
import { FileBrowser } from '@/components/common/FileBrowser';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { showToast } from '@/components/ui/toast';
import type { CreateTaskInput, UpdateTaskInput } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createQuery } from '@urql/solid';
import { Component, createSignal, Show } from 'solid-js';
import IconChevronLeft from '~icons/lucide/chevron-left';
import IconChevronRight from '~icons/lucide/chevron-right';
import IconHardDrive from '~icons/lucide/hard-drive';
import { TaskSettingsForm } from './TaskSettingsForm';

type TaskSettingsFormData = UpdateTaskInput;

// Re-export CreateTaskInput for backward compatibility
export type { CreateTaskInput };

export interface CreateTaskWizardProps {
  connectionId: string;
  open: boolean;
  onClose: () => void;
  onSubmit: (task: CreateTaskInput) => void | Promise<void>;
}

type WizardStep = 'paths' | 'settings';

export const CreateTaskWizard: Component<CreateTaskWizardProps> = (props) => {
  // Fetch connection info to get the name for display using GraphQL
  const [connectionResult] = createQuery({
    query: ConnectionGetBasicQuery,
    variables: () => ({ id: props.connectionId }),
    pause: () => !props.connectionId || !props.open,
  });

  const connectionName = () => connectionResult.data?.connection?.get?.name ?? props.connectionId;

  // Helper functions to load directory contents via GraphQL
  // Using unified FilesListQuery - connectionId: null for local, connectionId for remote
  const loadLocalFiles = async (path: string, refresh?: boolean) => {
    const result = await client.query(
      FilesListQuery,
      { connectionId: null, path },
      { requestPolicy: refresh ? 'network-only' : 'cache-first' }
    );
    if (result.error) throw new Error(result.error.message);
    return result.data?.file?.list ?? [];
  };

  const loadRemoteFiles = async (path: string, refresh?: boolean) => {
    const result = await client.query(
      FilesListQuery,
      { connectionId: props.connectionId, path },
      { requestPolicy: refresh ? 'network-only' : 'cache-first' }
    );
    if (result.error) throw new Error(result.error.message);
    return result.data?.file?.list ?? [];
  };

  const [currentStep, setCurrentStep] = createSignal<WizardStep>('paths');
  const [sourcePath, setSourcePath] = createSignal('');
  const [remotePath, setRemotePath] = createSignal('');
  const [formData, setFormData] = createSignal<TaskSettingsFormData>({
    name: '',
    direction: 'UPLOAD',
    schedule: '',
    realtime: false,
    options: {},
  });
  const [submitting, setSubmitting] = createSignal(false);

  const resetForm = () => {
    setCurrentStep('paths');
    setSourcePath('');
    setRemotePath('');
    setFormData({
      name: '',
      direction: 'UPLOAD',
      schedule: '',
      realtime: false,
      options: {},
    });
    setSubmitting(false);
  };

  const handleClose = () => {
    resetForm();
    props.onClose();
  };

  const handleNext = () => {
    if (currentStep() === 'paths') {
      setCurrentStep('settings');
    }
  };

  const handleBack = () => {
    if (currentStep() === 'settings') {
      setCurrentStep('paths');
    }
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      const data = formData();
      await props.onSubmit({
        name: data.name ?? `task-${new Date().getTime()}`,
        sourcePath: sourcePath(),
        connectionId: props.connectionId,
        remotePath: remotePath(),
        direction: data.direction ?? 'UPLOAD',
        schedule: data.schedule,
        realtime: data.realtime,
        options: data.options,
      });

      handleClose();
    } catch (error) {
      showToast({
        title: m.toast_failedToCreateTask(),
        description: error instanceof Error ? error.message : m.error_unknownError(),
        variant: 'destructive',
      });
    } finally {
      setSubmitting(false);
    }
  };

  const canProceed = () => {
    if (currentStep() === 'paths') {
      return sourcePath() !== '' && remotePath() !== '';
    }
    return true;
  };

  return (
    <Dialog open={props.open} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent class="flex h-[80vh] max-w-4xl flex-col">
        <DialogHeader>
          <DialogTitle>{m.wizard_createTask()}</DialogTitle>
          <DialogDescription>
            {currentStep() === 'paths'
              ? m.wizard_selectDirectories()
              : m.wizard_configureSettings()}
          </DialogDescription>
        </DialogHeader>

        <div class="min-h-0 flex-1">
          <Show when={currentStep() === 'paths'}>
            <div class="grid h-full grid-cols-2 gap-4">
              {/* Local Path Browser */}
              <div class="flex h-full min-h-0 flex-col rounded-lg border">
                <div class="border-b bg-muted px-4 py-3">
                  <h3 class="font-semibold">{m.wizard_localDirectory()}</h3>
                  <p class="text-sm text-muted-foreground">
                    {sourcePath() ?? m.wizard_noDirectorySelected()}
                  </p>
                </div>
                <div class="min-h-0 flex-1">
                  <FileBrowser
                    initialPath="/"
                    rootLabel={m.file_browser_root()}
                    icon={IconHardDrive}
                    loadDirectory={loadLocalFiles}
                    onSelect={setSourcePath}
                    class="h-full"
                  />
                </div>
              </div>

              {/* Remote Path Browser */}
              <div class="flex h-full min-h-0 flex-col rounded-lg border">
                <div class="border-b bg-muted px-4 py-3">
                  <h3 class="font-semibold">{m.wizard_remoteDirectory()}</h3>
                  <p class="text-sm text-muted-foreground">
                    {remotePath() ?? m.wizard_noDirectorySelected()}
                  </p>
                </div>
                <div class="min-h-0 flex-1">
                  <FileBrowser
                    initialPath="/"
                    rootLabel={`${connectionName()}:`}
                    loadDirectory={loadRemoteFiles}
                    onSelect={setRemotePath}
                    class="h-full"
                  />
                </div>
              </div>
            </div>
          </Show>

          <Show when={currentStep() === 'settings'}>
            <div class="h-full overflow-y-auto p-6">
              <TaskSettingsForm value={formData()} onChange={setFormData}>
                {/* Path Summary */}
                <div class="space-y-2 rounded-lg bg-muted p-4">
                  <h4 class="text-sm font-semibold">{m.wizard_taskSummary()}</h4>
                  <div class="space-y-1 text-sm">
                    <div>
                      <span class="text-muted-foreground">{m.wizard_local()}:</span> {sourcePath()}
                    </div>
                    <div>
                      <span class="text-muted-foreground">{m.wizard_remote()}:</span> {remotePath()}
                    </div>
                    <div>
                      <span class="text-muted-foreground">{m.wizard_direction()}:</span>{' '}
                      {formData().direction === 'UPLOAD'
                        ? m.form_directionUpload()
                        : formData().direction === 'DOWNLOAD'
                          ? m.form_directionDownload()
                          : m.form_directionBidirectional()}
                    </div>
                  </div>
                </div>
              </TaskSettingsForm>
            </div>
          </Show>
        </div>

        <DialogFooter class="gap-2">
          <Show when={currentStep() === 'settings'}>
            <Button variant="outline" onClick={handleBack}>
              <IconChevronLeft class="mr-2 size-4" />
              {m.common_back()}
            </Button>
          </Show>
          <Button variant="outline" onClick={handleClose}>
            {m.common_cancel()}
          </Button>
          <Show
            when={currentStep() === 'paths'}
            fallback={
              <Button onClick={handleSubmit} disabled={submitting()}>
                {submitting() ? m.wizard_creating() : m.task_create()}
              </Button>
            }
          >
            <Button onClick={handleNext} disabled={!canProceed()}>
              {m.common_next()}
              <IconChevronRight class="ml-2 size-4" />
            </Button>
          </Show>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
