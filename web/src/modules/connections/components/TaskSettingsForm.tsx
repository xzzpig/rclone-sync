import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { TextField, TextFieldInput, TextFieldLabel } from '@/components/ui/text-field';
import { SyncDirection, Task } from '@/lib/types';
import { Component, JSXElement, Show } from 'solid-js';

// Direction options
const directionOptions = [
  { value: 'upload', label: 'Upload (Local → Remote)' },
  { value: 'download', label: 'Download (Remote → Local)' },
  { value: 'bidirectional', label: 'Bidirectional (Sync Both)' },
] as const;

// Conflict resolution options (only applicable for bidirectional sync)
const conflictResolutionOptions = [
  { value: 'newer', label: 'Keep Newer' },
  { value: 'local', label: 'Keep Local' },
  { value: 'remote', label: 'Keep Remote' },
  { value: 'both', label: 'Keep Both' },
] as const;

export type ConflictResolution = (typeof conflictResolutionOptions)[number]['value'];

// Form data type - subset of Task type
export type TaskSettingsFormData = Pick<
  Task,
  'name' | 'direction' | 'schedule' | 'realtime' | 'options'
>;

export interface TaskSettingsFormProps {
  value: TaskSettingsFormData;
  onChange: (data: TaskSettingsFormData) => void;
  children?: JSXElement;
}

export const TaskSettingsForm: Component<TaskSettingsFormProps> = (props) => {
  const conflictResolution = () =>
    (props.value.options?.conflict_resolution as ConflictResolution) || 'newer';

  const updateField = <K extends keyof TaskSettingsFormData>(
    field: K,
    value: TaskSettingsFormData[K]
  ) => {
    const updates: Partial<TaskSettingsFormData> = { [field]: value };

    // When direction switches to 'download', automatically disable realtime
    if (field === 'direction' && value === 'download') {
      updates.realtime = false;
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
        conflict_resolution: value,
      },
    });
  };

  return (
    <div class="space-y-6">
      {/* Task Name */}
      <TextField>
        <TextFieldLabel for="name">Task Name</TextFieldLabel>
        <TextFieldInput
          id="name"
          value={props.value.name}
          onInput={(e: InputEvent) =>
            updateField('name', (e.currentTarget as HTMLInputElement).value)
          }
          placeholder="My Sync Task"
        />
      </TextField>

      {/* Direction */}
      <TextField>
        <TextFieldLabel for="direction">Sync Direction</TextFieldLabel>
        <Select
          value={props.value.direction}
          onChange={(value) => updateField('direction', value as SyncDirection)}
          options={['upload', 'download', 'bidirectional'] as const}
          placeholder="Select direction"
          itemComponent={(itemProps) => (
            <SelectItem item={itemProps.item}>
              {directionOptions.find((o) => o.value === itemProps.item.rawValue)?.label}
            </SelectItem>
          )}
        >
          <SelectTrigger id="direction">
            <SelectValue>
              {(state) => {
                const val = state.selectedOption();
                return directionOptions.find((o) => o.value === val)?.label ?? 'Select direction';
              }}
            </SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </TextField>

      {/* Schedule */}
      <TextField>
        <TextFieldLabel for="schedule">Schedule (Cron Expression)</TextFieldLabel>
        <TextFieldInput
          id="schedule"
          value={props.value.schedule ?? ''}
          onInput={(e: InputEvent) =>
            updateField('schedule', (e.currentTarget as HTMLInputElement).value)
          }
          placeholder="e.g., 0 */6 * * * (every 6 hours)"
        />
        <p class="text-xs text-muted-foreground">
          Leave empty to run manually only. Use{' '}
          <a
            href="https://crontab.guru"
            target="_blank"
            rel="noopener noreferrer"
            class="underline"
          >
            crontab.guru
          </a>{' '}
          for help.
        </p>
      </TextField>

      {/* Realtime Sync Toggle */}
      <Show when={props.value.direction !== 'download'}>
        <div class="flex items-center space-x-2">
          <Checkbox
            id="realtime"
            checked={props.value.realtime}
            onChange={(checked) => updateField('realtime', checked)}
          />
          <Label for="realtime" class="cursor-pointer">
            Enable Real-time Sync (Watch for local changes)
          </Label>
        </div>
      </Show>

      {/* Conflict Resolution (only for bidirectional sync) */}
      <Show when={props.value.direction === 'bidirectional'}>
        <TextField>
          <TextFieldLabel for="conflictResolution">Conflict Resolution</TextFieldLabel>
          <Select
            value={conflictResolution()}
            onChange={(value) => updateConflictResolution(value as ConflictResolution)}
            options={conflictResolutionOptions.map((o) => o.value)}
            placeholder="Select conflict resolution"
            itemComponent={(itemProps) => (
              <SelectItem item={itemProps.item}>
                {conflictResolutionOptions.find((o) => o.value === itemProps.item.rawValue)?.label}
              </SelectItem>
            )}
          >
            <SelectTrigger id="conflictResolution">
              <SelectValue>
                {(state) => {
                  const val = state.selectedOption();
                  return (
                    conflictResolutionOptions.find((o) => o.value === val)?.label ??
                    'Select conflict resolution'
                  );
                }}
              </SelectValue>
            </SelectTrigger>
            <SelectContent />
          </Select>
          <p class="text-xs text-muted-foreground">
            When both local and remote files are modified, choose which version to keep.
          </p>
        </TextField>
      </Show>

      {/* Children slot for additional content (e.g., Task Summary) */}
      {props.children}
    </div>
  );
};
