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
import { SyncDirection, Task } from '@/lib/types';
import { Component, JSXElement, Show } from 'solid-js';
import { RichText } from '@/components/common/RichText';

export type ConflictResolution = 'newer' | 'local' | 'remote' | 'both';

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
    (props.value.options?.conflict_resolution as ConflictResolution) ?? 'newer';

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
          options={['upload', 'download', 'bidirectional'] as const}
          placeholder={m.form_selectDirection()}
          itemComponent={(itemProps) => (
            <SelectItem item={itemProps.item}>
              {itemProps.item.rawValue === 'upload'
                ? m.form_directionUpload()
                : itemProps.item.rawValue === 'download'
                  ? m.form_directionDownload()
                  : m.form_directionBidirectional()}
            </SelectItem>
          )}
        >
          <SelectTrigger id="direction">
            <SelectValue>
              {(state) => {
                const val = state.selectedOption();
                return val === 'upload'
                  ? m.form_directionUpload()
                  : val === 'download'
                    ? m.form_directionDownload()
                    : val === 'bidirectional'
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
      <Show when={props.value.direction !== 'download'}>
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
      <Show when={props.value.direction === 'bidirectional'}>
        <TextField>
          <TextFieldLabel for="conflictResolution">{m.form_conflictResolution()}</TextFieldLabel>
          <Select
            value={conflictResolution()}
            onChange={(value) => updateConflictResolution(value as ConflictResolution)}
            options={['newer', 'local', 'remote', 'both'] as const}
            placeholder={m.form_selectConflictResolution()}
            itemComponent={(itemProps) => (
              <SelectItem item={itemProps.item}>
                {itemProps.item.rawValue === 'newer'
                  ? m.form_keepNewer()
                  : itemProps.item.rawValue === 'local'
                    ? m.form_keepLocal()
                    : itemProps.item.rawValue === 'remote'
                      ? m.form_keepRemote()
                      : m.form_keepBoth()}
              </SelectItem>
            )}
          >
            <SelectTrigger id="conflictResolution">
              <SelectValue>
                {(state) => {
                  const val = state.selectedOption();
                  return val === 'newer'
                    ? m.form_keepNewer()
                    : val === 'local'
                      ? m.form_keepLocal()
                      : val === 'remote'
                        ? m.form_keepRemote()
                        : val === 'both'
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
