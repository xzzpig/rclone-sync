import { createConnection, testConnection } from '@/api/connections';
import { HelpTooltip } from '@/components/common/HelpTooltip';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import { Switch, SwitchControl, SwitchLabel, SwitchThumb } from '@/components/ui/switch';
import {
  TextField,
  TextFieldErrorMessage,
  TextFieldInput,
  TextFieldLabel,
} from '@/components/ui/text-field';
import { extractErrorDetails, extractErrorMessage } from '@/lib/api';
import type { RcloneOption } from '@/lib/types';
import { useQueryClient } from '@tanstack/solid-query';
import { createEffect, createMemo, createSignal, For, Show } from 'solid-js';
import { createStore } from 'solid-js/store';
import IconAlertCircle from '~icons/lucide/alert-circle';
import IconChevronDown from '~icons/lucide/chevron-down';

const groupOptions = (options: RcloneOption[]) => {
  const grouped: Record<string, RcloneOption[]> = {};
  const advanced: RcloneOption[] = [];

  if (!Array.isArray(options)) {
    return { grouped, advanced };
  }

  options.forEach((opt) => {
    if (opt.Advanced) {
      advanced.push(opt);
      return;
    }
    const groupName = opt.Groups ?? 'General';
    if (!grouped[groupName]) {
      grouped[groupName] = [];
    }
    grouped[groupName].push(opt);
  });

  return { grouped, advanced };
};

const DynamicConfigFormSkeleton = (props: { showBack?: boolean }) => (
  <div class="space-y-4 p-4">
    {/* Connection Name */}
    <div class="space-y-2">
      <Skeleton class="h-4 w-[140px]" />
      <Skeleton class="h-10 w-full" />
    </div>
    {/* Form Fields Group 1 */}
    <div class="space-y-2">
      <Skeleton class="h-5 w-[100px]" />
      <div class="space-y-2">
        <Skeleton class="h-4 w-[120px]" />
        <Skeleton class="h-10 w-full" />
      </div>
      <div class="space-y-2">
        <Skeleton class="h-4 w-[100px]" />
        <Skeleton class="h-10 w-full" />
      </div>
    </div>
    {/* Form Fields Group 2 */}
    <div class="space-y-2">
      <div class="space-y-2">
        <Skeleton class="h-4 w-[110px]" />
        <Skeleton class="h-10 w-full" />
      </div>
    </div>
    {/* Button Area */}
    <div class="flex items-center justify-between pt-4">
      <Show when={props.showBack !== false}>
        <Skeleton class="h-10 w-[60px]" />
      </Show>
      <div class="flex items-center gap-2">
        <Skeleton class="h-10 w-[140px]" />
        <Skeleton class="h-10 w-[100px]" />
      </div>
    </div>
  </div>
);

export const DynamicConfigForm = (props: {
  options: RcloneOption[];
  provider: string;
  onBack: () => void;
  onSave: () => void;
  initialValues?: Record<string, string>;
  isEditing?: boolean;
  showBack?: boolean;
  loading?: boolean;
}) => {
  const [formState, setFormState] = createStore<Record<string, string | undefined>>(
    props.initialValues ?? {}
  );
  const [errors, setErrors] = createStore<
    Record<string, { message: string; type: 'error' | 'success'; details?: string } | undefined>
  >({});
  const [isTesting, setIsTesting] = createSignal(false);
  const queryClient = useQueryClient();

  const groupedData = createMemo(() => groupOptions(props.options));

  const handleInputChange = (name: string, value?: string) => {
    setFormState(name, value);
  };

  createEffect(() => {
    setFormState(props.initialValues ?? {});
  });

  const handleTest = async () => {
    setIsTesting(true);
    setErrors('testResult', undefined);
    try {
      await testConnection(props.provider, { ...formState, type: props.provider });
      setErrors('testResult', { message: 'Connection successful!', type: 'success' });
    } catch (err: unknown) {
      setErrors('testResult', {
        message: extractErrorMessage(err) ?? 'Connection failed.',
        details: extractErrorDetails(err),
        type: 'error',
      });
    } finally {
      setIsTesting(false);
    }
  };

  const handleSave = async () => {
    const name = formState['name']; // Assuming a 'name' field exists
    if (!name) {
      setErrors('name', { message: 'Connection name is required.', type: 'error' });
      return;
    }
    try {
      await createConnection(name, { ...formState, type: props.provider });
      await queryClient.invalidateQueries({ queryKey: ['connections'] });
      props.onSave();
    } catch (err: unknown) {
      setErrors('saveResult', {
        message: extractErrorMessage(err) ?? 'Failed to save connection.',
        details: extractErrorDetails(err),
        type: 'error',
      });
    }
  };

  return (
    <Show when={!props.loading} fallback={<DynamicConfigFormSkeleton showBack={props.showBack} />}>
      <div class="space-y-4 p-4">
        <TextField
          value={formState['name'] ?? ''}
          onChange={(v?: string) => handleInputChange('name', v)}
          required
          disabled={props.isEditing}
        >
          <TextFieldLabel for="connection-name">Connection Name</TextFieldLabel>
          <TextFieldInput
            id="connection-name"
            aria-required="true"
            aria-invalid={!!errors.name}
            aria-describedby={errors.name ? 'connection-name-error' : undefined}
          />
          <Show when={errors.name}>
            <TextFieldErrorMessage id="connection-name-error" role="alert">
              {errors.name?.message}
            </TextFieldErrorMessage>
          </Show>
        </TextField>

        <For each={Object.entries(groupedData().grouped)}>
          {([groupName, opts]) => (
            <div class="space-y-2">
              <h3 class="font-semibold">{groupName}</h3>
              <For each={opts}>
                {(opt) => (
                  <FormField
                    option={opt}
                    value={formState[opt.Name]}
                    onChange={handleInputChange}
                  />
                )}
              </For>
            </div>
          )}
        </For>

        <Show when={groupedData().advanced.length > 0}>
          <Collapsible>
            <Separator class="my-4" />
            <CollapsibleTrigger
              as={Button}
              variant="link"
              class="group flex w-full items-center justify-between p-0"
            >
              <span>Advanced Options</span>
              <IconChevronDown class="size-4 transition-transform duration-200 group-data-[expanded]:rotate-180" />
            </CollapsibleTrigger>
            <CollapsibleContent class="space-y-2 pt-2">
              <For each={groupedData().advanced}>
                {(opt) => (
                  <FormField
                    option={opt}
                    value={formState[opt.Name]}
                    onChange={handleInputChange}
                  />
                )}
              </For>
            </CollapsibleContent>
          </Collapsible>
        </Show>

        <div class="flex items-center justify-between pt-4">
          <Show when={props.showBack !== false}>
            <Button
              variant="ghost"
              onClick={props.onBack}
              aria-label="Go back to provider selection"
            >
              Back
            </Button>
          </Show>
          <div class="flex items-center gap-2">
            <Show when={errors.testResult}>
              <div
                class={`flex items-center gap-1 text-sm ${errors.testResult?.type === 'success' ? 'text-green-500' : 'text-red-500'}`}
                role="status"
                aria-live="polite"
              >
                <span>{errors.testResult?.message}</span>
                <Show when={errors.testResult?.details}>
                  <HelpTooltip
                    content={errors.testResult?.details ?? ''}
                    trigger={<IconAlertCircle class="size-4 cursor-pointer" />}
                  />
                </Show>
              </div>
            </Show>
            <Button
              variant="outline"
              onClick={handleTest}
              disabled={isTesting()}
              aria-label={isTesting() ? 'Testing connection...' : 'Test connection configuration'}
            >
              {isTesting() ? 'Testing...' : 'Test Connection'}
            </Button>
            <Button
              onClick={handleSave}
              aria-label={
                props.isEditing ? 'Update connection configuration' : 'Create new connection'
              }
            >
              {props.isEditing ? 'Update' : 'Create Connection'}
            </Button>
          </div>
        </div>
        <Show when={errors.saveResult}>
          <div
            class="flex items-center gap-1 text-sm text-red-500"
            role="alert"
            aria-live="assertive"
          >
            <span>{errors.saveResult?.message}</span>
            <Show when={errors.saveResult?.details}>
              <HelpTooltip
                content={errors.saveResult?.details ?? ''}
                trigger={<IconAlertCircle class="size-4 cursor-pointer" />}
              />
            </Show>
          </div>
        </Show>
      </div>
    </Show>
  );
};

const OptionLabel = (props: { option: RcloneOption }) => {
  return (
    <span class="flex items-center gap-1">
      {props.option.Name}
      <Show when={props.option.Required}>
        <span class="text-red-500">*</span>
      </Show>
      <HelpTooltip content={props.option.Help} />
    </span>
  );
};

const FormField = (props: {
  option: RcloneOption;
  value?: string;
  onChange: (name: string, value?: string) => void;
}) => {
  const val = () => props.value ?? props.option.DefaultStr;
  const fieldId = () => `field-${props.option.Name}`;
  const isRequired = () => props.option.Required;
  const changeValue = (v?: string | null) => {
    if (v === '') {
      return props.onChange(props.option.Name, undefined);
    }
    if (v === null) {
      return props.onChange(props.option.Name, undefined);
    }
    if (v === props.option.DefaultStr) {
      return props.onChange(props.option.Name, '');
    }
    return props.onChange(props.option.Name, v);
  };

  const label = (
    <label for={fieldId()} class="text-sm font-medium">
      <OptionLabel option={props.option} />
    </label>
  );

  return (
    <div class="mb-2">
      <Show
        when={props.option.Examples?.length > 0 && props.option.Exclusive}
        fallback={
          <Show
            when={props.option.Type === 'bool'}
            fallback={
              <TextField
                class="w-full"
                value={props.value ?? ''}
                onChange={(v: string) => changeValue(v)}
                required={isRequired()}
              >
                {label}
                <TextFieldInput
                  type={props.option.IsPassword ? 'password' : 'text'}
                  id={fieldId()}
                  class="mt-1"
                  placeholder={props.option.DefaultStr}
                  aria-required={isRequired() ? 'true' : undefined}
                  aria-describedby={props.option.Help ? `${fieldId()}-help` : undefined}
                />
              </TextField>
            }
          >
            <div class="mt-2">
              <Switch
                id={fieldId()}
                checked={val() === 'true'}
                onChange={(c) => changeValue(String(c))}
                class="relative flex w-full items-center justify-between"
                aria-describedby={props.option.Help ? `${fieldId()}-help` : undefined}
              >
                <SwitchLabel
                  class="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                  for={fieldId()}
                >
                  <OptionLabel option={props.option} />
                </SwitchLabel>
                <SwitchControl>
                  <SwitchThumb />
                </SwitchControl>
              </Switch>
            </div>
          </Show>
        }
      >
        {label}
        <Select
          value={val()}
          onChange={(v) => changeValue(v)}
          options={props.option.Examples.map((ex: { Value: string; Help: string }) => ex.Value)}
          placeholder="Select an option..."
          itemComponent={(p) => <SelectItem item={p.item}>{p.item.rawValue}</SelectItem>}
          required={isRequired()}
        >
          <SelectTrigger
            id={fieldId()}
            class="mt-1 w-full"
            aria-required={isRequired() ? 'true' : undefined}
            aria-describedby={props.option.Help ? `${fieldId()}-help` : undefined}
          >
            <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </Show>
    </div>
  );
};
