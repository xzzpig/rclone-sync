import { client } from '@/api/graphql/client';
import { ConnectionTestUnsavedMutation } from '@/api/graphql/queries/connections';
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
import type { ProviderOption } from '@/lib/types';
import * as m from '@/paraglide/messages.js';
import { createEffect, createMemo, createSignal, For, Show } from 'solid-js';
import { createStore, unwrap } from 'solid-js/store';
import IconAlertCircle from '~icons/lucide/alert-circle';
import IconChevronDown from '~icons/lucide/chevron-down';

const groupOptions = (options: ProviderOption[]) => {
  const grouped: Record<string, ProviderOption[]> = {};
  const advanced: ProviderOption[] = [];

  if (!Array.isArray(options)) {
    return { grouped, advanced };
  }

  options.forEach((opt) => {
    // Use advanced field to separate advanced options
    if (opt.advanced) {
      advanced.push(opt);
    } else {
      // Use groups field for grouping, default to 'General'
      const groupName = opt.groups ?? 'General';
      grouped[groupName] ??= [];
      grouped[groupName].push(opt);
    }
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
  options: ProviderOption[];
  provider: string;
  onBack: () => void;
  /** Save handler - receives name and config */
  onSave: (name: string | undefined, config: Record<string, string>) => Promise<void>;
  initialValues?: Record<string, string>;
  isEditing?: boolean;
  showBack?: boolean;
  loading?: boolean;
  /** Hide the connection name field */
  hideName?: boolean;
  /** Custom text for save button */
  saveButtonText?: string;
}) => {
  const [formState, setFormState] = createStore<Record<string, string | undefined>>(
    props.initialValues ?? {}
  );
  const [errors, setErrors] = createStore<
    Record<string, { message: string; type: 'error' | 'success'; details?: string } | undefined>
  >({});
  const [isTesting, setIsTesting] = createSignal(false);
  const [isSaving, setIsSaving] = createSignal(false);

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
      // Build config without name field - use unwrap to get current snapshot
      const currentFormState = unwrap(formState);
      const { name: _, ...config } = currentFormState;

      const result = await client.mutation(ConnectionTestUnsavedMutation, {
        input: {
          type: props.provider,
          config: config as Record<string, string>,
        },
      });

      if (result.error) {
        throw new Error(result.error.message);
      }

      const testResult = result.data?.connection?.testUnsaved;
      if (testResult?.__typename === 'ConnectionTestSuccess') {
        setErrors('testResult', { message: m.connection_testSuccess(), type: 'success' });
      } else if (testResult?.__typename === 'ConnectionTestFailure') {
        setErrors('testResult', {
          message: testResult.error ?? m.error_connectionFailed(),
          type: 'error',
        });
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : m.error_connectionFailed();
      setErrors('testResult', {
        message,
        type: 'error',
      });
    } finally {
      setIsTesting(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setErrors('saveResult', undefined);

    // Build config without name field - use unwrap to get current snapshot
    const currentFormState = unwrap(formState);
    const { name: nameValue, ...config } = currentFormState;

    // Validate name if not hidden
    if (!props.hideName && !nameValue) {
      setErrors('name', { message: m.connection_nameRequired(), type: 'error' });
      setIsSaving(false);
      return;
    }

    try {
      await props.onSave(nameValue, config as Record<string, string>);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : m.connection_saveError();
      setErrors('saveResult', {
        message,
        type: 'error',
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Show when={!props.loading} fallback={<DynamicConfigFormSkeleton showBack={props.showBack} />}>
      <div class="space-y-4 p-4">
        <Show when={!props.hideName}>
          <TextField
            value={formState['name'] ?? ''}
            onChange={(v?: string) => handleInputChange('name', v)}
            required
            disabled={props.isEditing}
          >
            <TextFieldLabel for="connection-name">{m.connection_connectionName()}</TextFieldLabel>
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
        </Show>

        <For each={Object.entries(groupedData().grouped)}>
          {([groupName, opts]) => (
            <div class="space-y-2">
              <h3 class="font-semibold">{groupName}</h3>
              <For each={opts}>
                {(opt) => (
                  <FormField
                    option={opt}
                    value={formState[opt.name]}
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
              <span>{m.connection_advancedOptions()}</span>
              <IconChevronDown class="size-4 transition-transform duration-200 group-data-[expanded]:rotate-180" />
            </CollapsibleTrigger>
            <CollapsibleContent class="space-y-2 pt-2">
              <For each={groupedData().advanced}>
                {(opt) => (
                  <FormField
                    option={opt}
                    value={formState[opt.name]}
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
              {m.common_back()}
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
              {isTesting() ? m.connection_testing() : m.connection_testConnection()}
            </Button>
            <Button
              onClick={handleSave}
              disabled={isSaving()}
              aria-label={
                props.saveButtonText ??
                (props.isEditing ? 'Update connection configuration' : 'Create new connection')
              }
            >
              {props.saveButtonText ??
                (props.isEditing ? m.connection_update() : m.connection_createConnection())}
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

const OptionLabel = (props: { option: ProviderOption }) => {
  return (
    <span class="flex items-center gap-1">
      {props.option.name}
      <Show when={props.option.required}>
        <span class="text-red-500">*</span>
      </Show>
      <HelpTooltip content={props.option.help} />
    </span>
  );
};

const FormField = (props: {
  option: ProviderOption;
  value?: string;
  onChange: (name: string, value?: string) => void;
}) => {
  const val = () => props.value ?? props.option.default ?? '';
  const fieldId = () => `field-${props.option.name}`;
  const isRequired = () => props.option.required;
  const changeValue = (v?: string | null) => {
    if (v === '') {
      return props.onChange(props.option.name, undefined);
    }
    if (v === null) {
      return props.onChange(props.option.name, undefined);
    }
    if (v === props.option.default) {
      return props.onChange(props.option.name, '');
    }
    return props.onChange(props.option.name, v);
  };

  const label = (
    <label for={fieldId()} class="text-sm font-medium">
      <OptionLabel option={props.option} />
    </label>
  );

  // Check if this is a select field (exclusive option with examples)
  const hasExamples = () => props.option.examples && props.option.examples.length > 0;
  const isExclusive = () => props.option.exclusive && hasExamples();
  const isBool = () => props.option.type === 'bool';

  return (
    <div class="mb-2">
      <Show
        when={isExclusive()}
        fallback={
          <Show
            when={isBool()}
            fallback={
              <TextField
                class="w-full"
                value={props.value ?? ''}
                onChange={(v: string) => changeValue(v)}
                required={isRequired()}
              >
                {label}
                <TextFieldInput
                  type={props.option.isPassword ? 'password' : 'text'}
                  id={fieldId()}
                  class="mt-1"
                  placeholder={props.option.default ?? ''}
                  aria-required={isRequired() ? 'true' : undefined}
                  aria-describedby={props.option.help ? `${fieldId()}-help` : undefined}
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
                aria-describedby={props.option.help ? `${fieldId()}-help` : undefined}
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
          options={props.option.examples?.map((ex) => ex.value) ?? []}
          placeholder="Select an option..."
          itemComponent={(p) => <SelectItem item={p.item}>{p.item.rawValue}</SelectItem>}
          required={isRequired()}
        >
          <SelectTrigger
            id={fieldId()}
            class="mt-1 w-full"
            aria-required={isRequired() ? 'true' : undefined}
            aria-describedby={props.option.help ? `${fieldId()}-help` : undefined}
          >
            <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </Show>
    </div>
  );
};
