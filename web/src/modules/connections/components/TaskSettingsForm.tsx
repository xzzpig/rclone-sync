import { RichText } from '@/components/common/RichText';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsIndicator, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { TextField, TextFieldInput, TextFieldLabel } from '@/components/ui/text-field';
import {
  type ConflictResolution,
  type HookInput,
  type HookListItem,
  type NestedUpdateHookInput,
  type SyncDirection,
  type UpdateTaskInput,
} from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { Component, createEffect, createMemo, createSignal, JSXElement, Show } from 'solid-js';
import { FilterPreviewPanel } from './FilterPreviewPanel';
import { FilterRulesEditor } from './FilterRulesEditor';
import { HookDialog } from './HookDialog';
import { HookList } from './HookList';

import { AppConfigQuery } from '@/api/graphql/queries/system';
import { Button } from '@/components/ui/button';
import { createQuery } from '@urql/solid';
import { hookToHookInput } from '../utils/transformers';
import { useTasks } from '@/store/tasks';

export interface TaskSettingsFormProps {
  value: UpdateTaskInput;
  onChange: (data: UpdateTaskInput) => void;
  taskId?: string;
  connectionId?: string;
  remotePath?: string;
  sourcePath?: string;
  children?: JSXElement;
}

export const TaskSettingsForm: Component<TaskSettingsFormProps> = (props) => {
  const [, actions] = useTasks();
  const [activeTab, setActiveTab] = createSignal<'basic' | 'filters' | 'hooks'>('basic');
  const [hookDialogOpen, setHookDialogOpen] = createSignal(false);
  const [editingHookId, setEditingHookId] = createSignal<string | undefined>(undefined);

  const [appConfigResult] = createQuery({
    query: AppConfigQuery,
  });

  const isHookEnabled = () => appConfigResult.data?.appConfig?.hook?.enabled ?? true;

  const [hasLoadedHooks, setHasLoadedHooks] = createSignal(false);
  const [isHooksLoading, setIsHooksLoading] = createSignal(false);
  createEffect(() => {
    if (!props.taskId) {
      setHasLoadedHooks(false);
      return;
    }

    if (activeTab() === 'hooks' && !hasLoadedHooks()) {
      setHasLoadedHooks(true);
      setIsHooksLoading(true);
      actions.loadTaskDetail(props.taskId).finally(() => {
        setIsHooksLoading(false);
      });
    }
  });

  const handleAddHook = () => {
    setEditingHookId(undefined);
    setHookDialogOpen(true);
  };

  const handleEditHook = (hook: HookListItem) => {
    setEditingHookId(hook.id);
    setHookDialogOpen(true);
  };

  const currentHooks = () => props.value.hooks ?? [];

  const hookListItems = createMemo<HookListItem[]>(() => {
    return currentHooks()
      .filter((h) => !h.delete)
      .map((h, index) => {
        const id = h.id ?? `local-${index}-${h.hook?.event}-${h.hook?.type}`;
        return {
          id,
          ...h.hook,
        };
      }) as HookListItem[];
  });

  const handleSaveHook = (data: HookInput) => {
    const hooks = [...currentHooks()];
    const editingId = editingHookId();

    if (editingId) {
      const listItems = hookListItems();
      const listItemIndex = listItems.findIndex((item) => item.id === editingId);
      if (listItemIndex !== -1) {
        let nonDeletedCount = 0;
        const actualIndex = hooks.findIndex((h) => {
          if (h.delete) return false;
          if (nonDeletedCount === listItemIndex) return true;
          nonDeletedCount++;
          return false;
        });

        if (actualIndex !== -1) {
          hooks[actualIndex] = {
            ...hooks[actualIndex],
            hook: data,
          };
        }
      }
    } else {
      hooks.push({
        hook: data,
      });
    }

    props.onChange({
      ...props.value,
      hooks,
    });
  };

  const handleHooksChange = (newListItems: HookListItem[]) => {
    const oldHooks = currentHooks();
    const resultHooks: NestedUpdateHookInput[] = [];

    oldHooks.forEach((oldH) => {
      if (oldH.id) {
        const isStillPresent = newListItems.some((item) => item.id === oldH.id);
        if (!isStillPresent) {
          resultHooks.push({ id: oldH.id, delete: true });
        }
      }
    });

    newListItems.forEach((item) => {
      const isLocal = item.id.startsWith('local-');
      resultHooks.push({
        id: isLocal ? undefined : item.id,
        hook: hookToHookInput(item),
      });
    });

    props.onChange({
      ...props.value,
      hooks: resultHooks,
    });
  };

  const conflictResolution = () => props.value.options?.conflictResolution;
  const filters = () => props.value.options?.filters ?? [];
  const noDelete = () => props.value.options?.noDelete ?? undefined;
  const transfers = () => props.value.options?.transfers;

  const updateField = <K extends keyof UpdateTaskInput>(field: K, value: UpdateTaskInput[K]) => {
    const updates: Partial<UpdateTaskInput> = { [field]: value };

    if (field === 'direction' && value === 'DOWNLOAD') {
      updates.realtime = undefined;
    }

    props.onChange({
      ...props.value,
      ...updates,
    });
  };

  const updateOptions = <K extends keyof NonNullable<UpdateTaskInput['options']>>(
    field: K,
    value: NonNullable<UpdateTaskInput['options']>[K]
  ) => {
    props.onChange({
      ...props.value,
      options: {
        ...props.value.options,
        [field]: value,
      },
    });
  };

  const isUnidirectional = () =>
    props.value.direction === 'UPLOAD' || props.value.direction === 'DOWNLOAD';

  const initialHookData = createMemo<HookInput | undefined>(() => {
    const id = editingHookId();
    if (!id) return undefined;
    return hookListItems().find((h) => h.id === id);
  });

  return (
    <Tabs
      value={activeTab()}
      onChange={(value) => {
        if (value === 'basic' || value === 'filters' || value === 'hooks') {
          setActiveTab(value);
        }
      }}
    >
      <TabsList class="mb-4 w-full">
        <TabsTrigger value="basic" class="flex-1">
          {m.task_taskSettings()}
        </TabsTrigger>
        <TabsTrigger value="filters" class="flex-1">
          {m.task_filters()}
        </TabsTrigger>
        <Show when={isHookEnabled()}>
          <TabsTrigger value="hooks" class="flex-1">
            {m.hook_config()}
          </TabsTrigger>
        </Show>
        <TabsIndicator />
      </TabsList>

      <TabsContent value="basic" class="space-y-6">
        {/* Task Name */}
        <TextField>
          <TextFieldLabel for="name">{m.form_taskName()}</TextFieldLabel>
          <TextFieldInput
            id="name"
            value={props.value.name ?? undefined}
            onInput={(e) => updateField('name', e.currentTarget.value)}
            placeholder={m.form_taskNamePlaceholder()}
          />
        </TextField>

        {/* Direction */}
        <TextField>
          <TextFieldLabel for="direction">{m.form_syncDirection()}</TextFieldLabel>
          <Select<SyncDirection>
            value={props.value.direction}
            onChange={(value) => value && updateField('direction', value)}
            options={['UPLOAD', 'DOWNLOAD', 'BIDIRECTIONAL']}
            placeholder={m.form_selectDirection()}
            itemComponent={(itemProps) => (
              <SelectItem item={itemProps.item}>
                {itemProps.item.rawValue === 'UPLOAD'
                  ? m.form_directionUpload()
                  : itemProps.item.rawValue === 'DOWNLOAD'
                    ? m.form_directionDownload()
                    : m.form_directionBidirectional()}
              </SelectItem>
            )}
          >
            <SelectTrigger id="direction">
              <SelectValue>
                {(state) => {
                  const val = state.selectedOption();
                  return val === 'UPLOAD'
                    ? m.form_directionUpload()
                    : val === 'DOWNLOAD'
                      ? m.form_directionDownload()
                      : val === 'BIDIRECTIONAL'
                        ? m.form_directionBidirectional()
                        : m.form_selectDirection();
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent />
          </Select>
        </TextField>

        {/* Schedule */}
        <TextField>
          <TextFieldLabel for="schedule">{m.form_scheduleCron()}</TextFieldLabel>
          <TextFieldInput
            id="schedule"
            value={props.value.schedule ?? ''}
            onInput={(e) => updateField('schedule', e.currentTarget.value)}
            placeholder={m.form_scheduleExample()}
          />
          <p class="text-xs text-muted-foreground">
            <RichText text={m.form_scheduleHelp({ link: m.form_crontabGuru() })} />
          </p>
        </TextField>

        {/* Realtime Sync Toggle */}
        <Show when={props.value.direction !== 'DOWNLOAD'}>
          <div class="flex items-center space-x-2">
            <Checkbox
              id="realtime"
              checked={props.value.realtime ?? undefined}
              onChange={(checked) => updateField('realtime', checked)}
            />
            <Label for="realtime-input" class="cursor-pointer">
              {m.form_enableRealtime()}
            </Label>
          </div>
        </Show>

        {/* Conflict Resolution (only for bidirectional sync) */}
        <Show when={props.value.direction === 'BIDIRECTIONAL'}>
          <TextField>
            <TextFieldLabel for="conflictResolution">{m.form_conflictResolution()}</TextFieldLabel>
            <Select<ConflictResolution>
              value={conflictResolution()}
              onChange={(value) => value && updateOptions('conflictResolution', value)}
              options={['NEWER', 'LOCAL', 'REMOTE', 'BOTH']}
              placeholder={m.form_selectConflictResolution()}
              itemComponent={(itemProps) => (
                <SelectItem item={itemProps.item}>
                  {itemProps.item.rawValue === 'NEWER'
                    ? m.form_keepNewer()
                    : itemProps.item.rawValue === 'LOCAL'
                      ? m.form_keepLocal()
                      : itemProps.item.rawValue === 'REMOTE'
                        ? m.form_keepRemote()
                        : m.form_keepBoth()}
                </SelectItem>
              )}
            >
              <SelectTrigger id="conflictResolution">
                <SelectValue>
                  {(state) => {
                    const val = state.selectedOption();
                    return val === 'NEWER'
                      ? m.form_keepNewer()
                      : val === 'LOCAL'
                        ? m.form_keepLocal()
                        : val === 'REMOTE'
                          ? m.form_keepRemote()
                          : val === 'BOTH'
                            ? m.form_keepBoth()
                            : m.form_selectConflictResolution();
                  }}
                </SelectValue>
              </SelectTrigger>
              <SelectContent />
            </Select>
            <p class="text-xs text-muted-foreground">{m.form_conflictHelp()}</p>
          </TextField>
        </Show>

        {/* No Delete Option (only for unidirectional sync) */}
        <Show when={isUnidirectional()}>
          <div class="flex items-center space-x-2">
            <Checkbox
              id="noDelete"
              checked={noDelete()}
              onChange={(v) => updateOptions('noDelete', v)}
            />
            <Label for="noDelete-input" class="cursor-pointer">
              {m.filter_noDelete()}
            </Label>
          </div>
          <p class="text-xs text-muted-foreground">{m.filter_noDeleteHelp()}</p>
        </Show>

        {/* Parallel Transfers */}
        <TextField>
          <TextFieldLabel for="transfers">{m.filter_transfers()}</TextFieldLabel>
          <TextFieldInput
            id="transfers"
            type="number"
            min={1}
            max={64}
            value={transfers() ?? ''}
            onInput={(e) => {
              const inputValue = e.currentTarget.value;
              updateOptions(
                'transfers',
                inputValue ? Math.max(1, Math.min(64, parseInt(inputValue, 10))) : undefined
              );
            }}
          />
          <p class="text-xs text-muted-foreground">{m.filter_transfersHelp()}</p>
        </TextField>

        {/* Children slot for additional content (e.g., Task Summary) */}
        {props.children}
      </TabsContent>

      <TabsContent value="filters" class="space-y-6">
        {/* Filter Rules Editor */}
        <FilterRulesEditor value={filters()} onChange={(v) => updateOptions('filters', v)} />

        {/* Filter Preview Panel */}
        <Show when={props.connectionId && props.remotePath}>
          <FilterPreviewPanel
            connectionId={props.connectionId ?? ''}
            sourcePath={props.sourcePath ?? ''}
            remotePath={props.remotePath ?? ''}
            filters={filters()}
          />
        </Show>
      </TabsContent>

      <TabsContent value="hooks" class="space-y-6">
        <div class="flex justify-end">
          <Button variant="outline" size="sm" onClick={handleAddHook}>
            {m.hook_addHook()}
          </Button>
        </div>
        <HookList
          hooks={hookListItems()}
          fetching={isHooksLoading()}
          onEdit={handleEditHook}
          onEnabledChange={(id, enabled) => {
            const newList = hookListItems().map((h) => (h.id === id ? { ...h, enabled } : h));
            handleHooksChange(newList);
          }}
          onDelete={(id) => {
            const newList = hookListItems().filter((h) => h.id !== id);
            handleHooksChange(newList);
          }}
        />
        <HookDialog
          open={hookDialogOpen()}
          onOpenChange={setHookDialogOpen}
          initialData={initialHookData()}
          isEdit={!!editingHookId()}
          onSave={handleSaveHook}
        />
      </TabsContent>
    </Tabs>
  );
};
