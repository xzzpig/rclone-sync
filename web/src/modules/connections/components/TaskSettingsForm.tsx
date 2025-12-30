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
import type { ConflictResolution, SyncDirection, Task, UpdateTaskInput } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { Component, createSignal, JSXElement, Show } from 'solid-js';
import { FilterPreviewPanel } from './FilterPreviewPanel';
import { FilterRulesEditor } from './FilterRulesEditor';

/**
 * Converts a Task object (from GraphQL) to an UpdateTaskInput for form state.
 * Handles null values and nested options.
 */
export function taskToUpdateInput(task: Task): UpdateTaskInput {
  return {
    name: task.name,
    direction: task.direction as SyncDirection,
    schedule: task.schedule,
    realtime: task.realtime,
    options: {
      conflictResolution: task.options?.conflictResolution,
      filters: task.options?.filters
        ? [...task.options.filters].filter((f): f is string => f !== null)
        : [],
      noDelete: task.options?.noDelete,
      transfers: task.options?.transfers,
    },
  };
}

export interface TaskSettingsFormProps {
  value: UpdateTaskInput;
  onChange: (data: UpdateTaskInput) => void;
  /**
   * Connection ID for filter preview (optional)
   */
  connectionId?: string;
  /**
   * Remote path for filter preview (optional)
   */
  remotePath?: string;
  /**
   * Source (local) path for filter preview (optional)
   */
  sourcePath?: string;
  children?: JSXElement;
}

export const TaskSettingsForm: Component<TaskSettingsFormProps> = (props) => {
  const [activeTab, setActiveTab] = createSignal<'basic' | 'filters'>('basic');
  const conflictResolution = () => props.value.options?.conflictResolution;
  const filters = () => (props.value.options?.filters ?? []) as string[];
  const noDelete = () => props.value.options?.noDelete ?? undefined;
  const transfers = () => props.value.options?.transfers;

  const updateField = <K extends keyof UpdateTaskInput>(field: K, value: UpdateTaskInput[K]) => {
    const updates: Partial<UpdateTaskInput> = { [field]: value };

    // When direction switches to 'DOWNLOAD', automatically disable realtime
    if (field === 'direction' && value === 'DOWNLOAD') {
      updates.realtime = undefined;
    }

    props.onChange({
      ...props.value,
      ...updates,
    });
  };

  const updateConflictResolution = (value: ConflictResolution) => {
    props.onChange({
      ...props.value,
      options: {
        ...props.value.options,
        conflictResolution: value,
      },
    });
  };

  const updateFilters = (newFilters: string[]) => {
    props.onChange({
      ...props.value,
      options: {
        ...props.value.options,
        filters: newFilters,
      },
    });
  };

  const updateNoDelete = (value: boolean) => {
    props.onChange({
      ...props.value,
      options: {
        ...props.value.options,
        noDelete: value,
      },
    });
  };

  const updateTransfers = (value: number | undefined) => {
    // If value is 0 or undefined, set to undefined to let backend handle default
    const clampedValue = value ? Math.max(1, Math.min(64, value)) : undefined;
    props.onChange({
      ...props.value,
      options: {
        ...props.value.options,
        transfers: clampedValue,
      },
    });
  };

  // Check if this is unidirectional sync (not bidirectional) for noDelete option
  const isUnidirectional = () =>
    props.value.direction === 'UPLOAD' || props.value.direction === 'DOWNLOAD';

  return (
    <Tabs value={activeTab()} onChange={(value) => setActiveTab(value as 'basic' | 'filters')}>
      <TabsList class="mb-4 w-full">
        <TabsTrigger value="basic" class="flex-1">
          {m.task_taskSettings()}
        </TabsTrigger>
        <TabsTrigger value="filters" class="flex-1">
          {m.task_filters()}
        </TabsTrigger>
        <TabsIndicator />
      </TabsList>

      <TabsContent value="basic" class="space-y-6">
        {/* Task Name */}
        <TextField>
          <TextFieldLabel for="name">{m.form_taskName()}</TextFieldLabel>
          <TextFieldInput
            id="name"
            value={props.value.name}
            onInput={(e: InputEvent) =>
              updateField('name', (e.currentTarget as HTMLInputElement).value)
            }
            placeholder={m.form_taskNamePlaceholder()}
          />
        </TextField>

        {/* Direction */}
        <TextField>
          <TextFieldLabel for="direction">{m.form_syncDirection()}</TextFieldLabel>
          <Select
            value={props.value.direction}
            onChange={(value) => updateField('direction', value as SyncDirection)}
            options={['UPLOAD', 'DOWNLOAD', 'BIDIRECTIONAL'] as const}
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
            onInput={(e: InputEvent) =>
              updateField('schedule', (e.currentTarget as HTMLInputElement).value)
            }
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
            <Select
              value={conflictResolution()}
              onChange={(value) => updateConflictResolution(value as ConflictResolution)}
              options={['NEWER', 'LOCAL', 'REMOTE', 'BOTH'] as const}
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
            <Checkbox id="noDelete" checked={noDelete()} onChange={updateNoDelete} />
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
            onInput={(e: InputEvent) => {
              const inputValue = (e.currentTarget as HTMLInputElement).value;
              updateTransfers(inputValue ? parseInt(inputValue, 10) : undefined);
            }}
          />
          <p class="text-xs text-muted-foreground">{m.filter_transfersHelp()}</p>
        </TextField>

        {/* Children slot for additional content (e.g., Task Summary) */}
        {props.children}
      </TabsContent>

      <TabsContent value="filters" class="space-y-6">
        {/* Filter Rules Editor */}
        <FilterRulesEditor value={filters()} onChange={updateFilters} />

        {/* Filter Preview Panel */}
        <Show when={props.connectionId && props.remotePath}>
          <FilterPreviewPanel
            connectionId={props.connectionId!}
            sourcePath={props.sourcePath ?? ''}
            remotePath={props.remotePath!}
            filters={filters()}
          />
        </Show>
      </TabsContent>
    </Tabs>
  );
};
