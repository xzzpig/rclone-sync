import * as m from '@/paraglide/messages.js';
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
import { Component, JSXElement, Show } from 'solid-js';
import { RichText } from '@/components/common/RichText';
import type { ConflictResolution, SyncDirection } from '@/lib/types';

// Form data type for task settings
export interface TaskSettingsFormData {
  name: string;
  direction: SyncDirection;
  schedule: string;
  realtime: boolean;
  options?: {
    conflictResolution?: ConflictResolution;
  };
}

export interface TaskSettingsFormProps {
  value: TaskSettingsFormData;
  onChange: (data: TaskSettingsFormData) => void;
  children?: JSXElement;
}

export const TaskSettingsForm: Component<TaskSettingsFormProps> = (props) => {
  const conflictResolution = () => props.value.options?.conflictResolution ?? 'NEWER';

  const updateField = <K extends keyof TaskSettingsFormData>(
    field: K,
    value: TaskSettingsFormData[K]
  ) => {
    const updates: Partial<TaskSettingsFormData> = { [field]: value };

    // When direction switches to 'DOWNLOAD', automatically disable realtime
    if (field === 'direction' && value === 'DOWNLOAD') {
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
        conflictResolution: value,
      },
    });
  };

  return (
    <div class="space-y-6">
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
            checked={props.value.realtime}
            onChange={(checked) => updateField('realtime', checked)}
          />
          <Label for="realtime" class="cursor-pointer">
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

      {/* Children slot for additional content (e.g., Task Summary) */}
      {props.children}
    </div>
  );
};
