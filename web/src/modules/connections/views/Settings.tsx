import * as m from '@/paraglide/messages.js';
import {
  deleteConnection,
  getConnection,
  getConnectionConfig,
  getProviderOptions,
  updateConnection,
} from '@/api/connections';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { showToast } from '@/components/ui/toast';
import { useNavigate, useParams } from '@solidjs/router';
import { useQuery, useQueryClient } from '@tanstack/solid-query';
import { Component, Show, createSignal } from 'solid-js';
import IconLoader2 from '~icons/lucide/loader-2';
import { DynamicConfigForm } from '../components/DynamicConfigForm';

const Settings: Component = () => {
  const params = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const connectionId = () => params.connectionId;

  // Fetch connection info
  const connectionQuery = useQuery(() => ({
    queryKey: ['connection', connectionId()],
    queryFn: () => getConnection(connectionId()!),
    enabled: !!connectionId(),
  }));

  // Fetch connection config
  const configQuery = useQuery(() => ({
    queryKey: ['connectionConfig', connectionId()],
    queryFn: () => getConnectionConfig(connectionId()!),
    enabled: !!connectionId(),
  }));

  // Fetch provider options when config is available
  const optionsQuery = useQuery(() => ({
    queryKey: ['providerOptions', configQuery.data?.type],
    queryFn: () => getProviderOptions(configQuery.data!.type),
    enabled: !!configQuery.data?.type,
  }));

  const connectionName = () => connectionQuery.data?.name ?? connectionId() ?? '';

  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);

  const handleSave = async (_name: string | undefined, config: Record<string, string>) => {
    const id = connectionId();
    if (!id) {
      throw new Error('Connection ID is required');
    }
    await updateConnection(id, { config });
    await queryClient.invalidateQueries({ queryKey: ['connections'] });
    await queryClient.invalidateQueries({ queryKey: ['connectionConfig', id] });

    // Show success toast
    showToast({
      title: m.toast_taskUpdated(),
      description: m.toast_taskUpdatedDesc({ name: connectionName() }),
    });
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      const id = connectionId();
      if (id) {
        await deleteConnection(id);
        await queryClient.invalidateQueries({ queryKey: ['connections'] });
        navigate('/');
      }
    } catch (error) {
      console.error('Failed to delete connection:', error);
      showToast({
        title: m.common_error(),
        description: m.connection_failedToDelete(),
        variant: 'error',
      });
    } finally {
      setIsDeleting(false);
      setIsDeleteDialogOpen(false);
    }
  };

  return (
    <div class="h-full space-y-6 overflow-auto [scrollbar-gutter:stable]">
      <Card>
        <CardHeader>
          <Show when={!configQuery.isLoading} fallback={<Skeleton class="h-6 w-[180px]" />}>
            <CardTitle>{m.common_settings()}</CardTitle>
          </Show>
        </CardHeader>
        <CardContent>
          <DynamicConfigForm
            loading={configQuery.isLoading || optionsQuery.isLoading}
            initialValues={{ ...configQuery.data, name: connectionName() ?? '' }}
            options={optionsQuery.data ?? []}
            provider={configQuery.data?.type ?? ''}
            isEditing={true}
            showBack={false}
            onBack={() => navigate('..')}
            onSave={handleSave}
          />
        </CardContent>
      </Card>

      <Card class="border-red-200">
        <CardHeader>
          <CardTitle class="text-red-500">{m.connection_dangerZone()}</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="mb-4 text-sm text-muted-foreground">{m.connection_deleteWarning()}</p>
          <Button variant="destructive" onClick={() => setIsDeleteDialogOpen(true)}>
            {m.common_delete()}
          </Button>
        </CardContent>
      </Card>

      <Dialog open={isDeleteDialogOpen()} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{m.connection_deleteConfirmTitle()}</DialogTitle>
            <DialogDescription>
              {m.connection_deleteConfirmDesc({ name: connectionName() ?? '' })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteDialogOpen(false)}>
              {m.common_cancel()}
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={isDeleting()}>
              {isDeleting() ? <IconLoader2 class="mr-2 size-4 animate-spin" /> : null}
              {m.common_delete()}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Settings;
