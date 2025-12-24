import { client } from '@/api/graphql/client';
import {
  ConnectionDeleteMutation,
  ConnectionGetConfigQuery,
  ConnectionUpdateMutation,
} from '@/api/graphql/queries/connections';
import { ProviderGetQuery } from '@/api/graphql/queries/providers';
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
import * as m from '@/paraglide/messages.js';
import { useNavigate, useParams } from '@solidjs/router';
import { createMutation, createQuery } from '@urql/solid';
import { Component, Show, createMemo, createSignal } from 'solid-js';
import IconLoader2 from '~icons/lucide/loader-2';
import { DynamicConfigForm } from '../components/DynamicConfigForm';

const Settings: Component = () => {
  const params = useParams();
  const navigate = useNavigate();
  const connectionId = () => params.connectionId;

  // Fetch connection info with config using GraphQL
  const [connectionResult] = createQuery({
    query: ConnectionGetConfigQuery,
    variables: () => ({ id: connectionId()! }),
    pause: () => !connectionId(),
  });

  // Extract connection data
  const connection = () => connectionResult.data?.connection?.get;
  const connectionName = () => connection()?.name ?? connectionId() ?? '';
  const connectionType = () => connection()?.type;
  const connectionConfig = () => connection()?.config;

  // Fetch provider options when connection type is available
  const [providerResult] = createQuery({
    query: ProviderGetQuery,
    variables: () => ({ name: connectionType()! }),
    pause: () => !connectionType(),
  });

  // Extract provider options directly from GraphQL (lowercase property names)
  const providerOptions = () => providerResult.data?.provider?.get?.options ?? [];

  // Build initial values by spreading config into flat Record<string, string>
  const initialValues = createMemo(() => {
    const config = connectionConfig();
    return {
      name: connectionName(),
      ...(config ?? {}),
    };
  });

  // Mutations
  const [, executeUpdateConnection] = createMutation(ConnectionUpdateMutation);
  const [, executeDeleteConnection] = createMutation(ConnectionDeleteMutation);

  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);

  const handleSave = async (_name: string | undefined, config: Record<string, string>) => {
    const id = connectionId();
    if (!id) {
      throw new Error('Connection ID is required');
    }

    const result = await executeUpdateConnection({
      id,
      input: { config },
    });

    if (result.error) {
      throw new Error(result.error.message);
    }

    // Invalidate connection queries
    client.query(ConnectionGetConfigQuery, { id }, { requestPolicy: 'network-only' });

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
        const result = await executeDeleteConnection({ id });
        if (result.error) {
          throw new Error(result.error.message);
        }
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

  const isLoading = () => connectionResult.fetching || providerResult.fetching;

  return (
    <div class="h-full space-y-6 overflow-auto [scrollbar-gutter:stable]">
      <Card>
        <CardHeader>
          <Show when={!isLoading()} fallback={<Skeleton class="h-6 w-[180px]" />}>
            <CardTitle>{m.common_settings()}</CardTitle>
          </Show>
        </CardHeader>
        <CardContent>
          <DynamicConfigForm
            loading={isLoading()}
            initialValues={initialValues()}
            options={providerOptions()}
            provider={connectionType() ?? ''}
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
