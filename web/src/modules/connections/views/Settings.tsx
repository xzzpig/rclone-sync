import { deleteConnection, getProviderOptions, getRemoteConfig } from '@/api/connections';
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
  const connectionName = () => params.connectionName;

  // 1. Fetch remote config
  const configQuery = useQuery(() => ({
    queryKey: ['remoteConfig', connectionName()],
    queryFn: () => getRemoteConfig(connectionName()!),
    enabled: !!connectionName(),
  }));

  // 2. Fetch provider options when config is available
  const optionsQuery = useQuery(() => ({
    queryKey: ['providerOptions', configQuery.data?.type],
    queryFn: () => getProviderOptions(configQuery.data!.type),
    enabled: !!configQuery.data?.type,
  }));

  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      const name = connectionName();
      if (name) {
        await deleteConnection(name);
        await queryClient.invalidateQueries({ queryKey: ['connections'] });
        navigate('/');
      }
    } catch (error) {
      console.error('Failed to delete connection:', error);
      showToast({
        title: 'Error',
        description: 'Failed to delete connection. Please try again.',
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
            <CardTitle>Connection Settings</CardTitle>
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
            onSave={() => {
              showToast({
                title: 'Connection updated',
                description: `Connection "${connectionName()}" has been updated successfully.`,
              });
            }}
          />
        </CardContent>
      </Card>

      <Card class="border-red-200">
        <CardHeader>
          <CardTitle class="text-red-500">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          <p class="mb-4 text-sm text-muted-foreground">
            Deleting this connection will remove the configuration properly. This action cannot be
            undone.
          </p>
          <Button variant="destructive" onClick={() => setIsDeleteDialogOpen(true)}>
            Delete Connection
          </Button>
        </CardContent>
      </Card>

      <Dialog open={isDeleteDialogOpen()} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Are you sure?</DialogTitle>
            <DialogDescription>
              This will permanently delete the connection "{connectionName()}".
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={isDeleting()}>
              {isDeleting() ? <IconLoader2 class="mr-2 size-4 animate-spin" /> : null}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};

export default Settings;
