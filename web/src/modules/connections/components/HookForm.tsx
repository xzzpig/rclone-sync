import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch, SwitchControl, SwitchLabel, SwitchThumb } from '@/components/ui/switch';
import {
  TextField,
  TextFieldDescription,
  TextFieldErrorMessage,
  TextFieldInput,
  TextFieldTextArea,
} from '@/components/ui/text-field';
import * as m from '@/paraglide/messages.js';
import { Index, Show, createEffect, createMemo, createSignal, untrack } from 'solid-js';

import { type HookEvent, type HookInput, type HookOnError, type HookType } from '@/lib/types';
import { getEventLabel, getOnErrorLabel, getTypeLabel } from '../utils/hooks';

interface HookFormProps {
  value: HookInput;
  onChange: (value: HookInput) => void;
  onSubmit?: () => void;
  onValidate?: (isValid: boolean) => void;
}

export function HookForm(props: HookFormProps) {
  const [headers, setHeaders] = createSignal<Array<{ key: string; value: string }>>([]);

  createEffect(() => {
    const configHeaders = props.value.config.headers;
    const initialHeaders = configHeaders
      ? Object.entries(configHeaders).map(([key, value]) => ({ key, value }))
      : [];

    const isSame = untrack(() => {
      const current = headers().filter((h) => h.key);
      if (current.length !== initialHeaders.length) return false;
      return current.every(
        (h, i) => h.key === initialHeaders[i].key && h.value === initialHeaders[i].value
      );
    });

    if (!isSame) {
      setHeaders(initialHeaders);
    }
  });

  const [errors, setErrors] = createSignal<Record<string, string>>({});

  const updateField = <K extends keyof HookInput>(field: K, value: HookInput[K]) => {
    props.onChange({ ...props.value, [field]: value });
  };

  const updateConfig = <K extends keyof HookInput['config']>(
    field: K,
    value: HookInput['config'][K]
  ) => {
    props.onChange({
      ...props.value,
      config: { ...props.value.config, [field]: value },
    });
  };

  const addHeader = () => {
    setHeaders([...headers(), { key: '', value: '' }]);
  };

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const newHeaders = [...headers()];
    newHeaders[index] = { ...newHeaders[index], [field]: value };
    setHeaders(newHeaders);

    const headersObj = newHeaders.reduce<Record<string, string>>((acc, h) => {
      if (h.key) acc[h.key] = h.value;
      return acc;
    }, {});
    updateConfig('headers', headersObj);
  };

  const removeHeader = (index: number) => {
    const newHeaders = headers().filter((_, i) => i !== index);
    setHeaders(newHeaders);

    const headersObj = newHeaders.reduce<Record<string, string>>((acc, h) => {
      if (h.key) acc[h.key] = h.value;
      return acc;
    }, {});
    updateConfig('headers', headersObj);
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (props.value.type === 'HTTP' && !props.value.config.url) {
      newErrors.url = m.hook_urlRequired();
    }

    if (props.value.type === 'COMMAND' && !props.value.config.command) {
      newErrors.command = m.hook_commandRequired();
    }

    setErrors(newErrors);
    const isValid = Object.keys(newErrors).length === 0;
    props.onValidate?.(isValid);
    return isValid;
  };

  createEffect(() => {
    validate();
  });

  const isHttpType = createMemo(() => props.value.type === 'HTTP');
  const isCommandType = createMemo(() => props.value.type === 'COMMAND');

  return (
    <div class="space-y-4">
      <Switch
        class="relative flex w-full items-center justify-between"
        checked={props.value.enabled ?? true}
        onChange={(checked) => updateField('enabled', checked)}
      >
        <SwitchLabel class="cursor-pointer text-sm font-medium">{m.hook_enabled()}</SwitchLabel>
        <SwitchControl>
          <SwitchThumb />
        </SwitchControl>
      </Switch>

      <TextField
        value={props.value.event}
        onChange={(value) => {
          if (
            value === 'ON_START' ||
            value === 'ON_SUCCESS' ||
            value === 'ON_FAILURE' ||
            value === 'ON_END'
          ) {
            updateField('event', value);
          }
        }}
      >
        <Label class="mb-1 block">{m.hook_event()}</Label>
        <Select<HookEvent>
          value={props.value.event}
          onChange={(value) => value && updateField('event', value)}
          options={['ON_START', 'ON_SUCCESS', 'ON_FAILURE', 'ON_END']}
          placeholder={m.common_select()}
          itemComponent={(itemProps) => (
            <SelectItem item={itemProps.item}>{getEventLabel(itemProps.item.rawValue)}</SelectItem>
          )}
        >
          <SelectTrigger>
            <SelectValue<HookEvent>>{(state) => getEventLabel(state.selectedOption())}</SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </TextField>

      <TextField
        value={props.value.type}
        onChange={(value) => {
          if (value === 'HTTP' || value === 'COMMAND') {
            updateField('type', value);
          }
        }}
      >
        <Label class="mb-1 block">{m.hook_type()}</Label>
        <Select<HookType>
          value={props.value.type}
          onChange={(value) => value && updateField('type', value)}
          options={['HTTP', 'COMMAND']}
          placeholder={m.common_select()}
          itemComponent={(itemProps) => (
            <SelectItem item={itemProps.item}>{getTypeLabel(itemProps.item.rawValue)}</SelectItem>
          )}
        >
          <SelectTrigger>
            <SelectValue<HookType>>{(state) => getTypeLabel(state.selectedOption())}</SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </TextField>

      <TextField
        value={props.value.priority?.toString() ?? ''}
        onChange={(value) => updateField('priority', value ? parseInt(value, 10) : null)}
        validationState={errors().priority ? 'invalid' : 'valid'}
      >
        <Label class="mb-1 block">{m.hook_priority()}</Label>
        <TextFieldInput type="number" />
        <TextFieldDescription>{m.hook_priorityHelp()}</TextFieldDescription>
      </TextField>

      <TextField
        value={props.value.onError ?? 'IGNORE'}
        onChange={(value) => {
          if (value === 'IGNORE' || value === 'CANCEL' || value === 'FATAL') {
            updateField('onError', value);
          }
        }}
      >
        <Label class="mb-1 block">{m.hook_onError()}</Label>
        <Select<HookOnError>
          value={props.value.onError ?? 'IGNORE'}
          onChange={(value) => value && updateField('onError', value)}
          options={['IGNORE', 'CANCEL', 'FATAL']}
          placeholder={m.common_select()}
          itemComponent={(itemProps) => (
            <SelectItem item={itemProps.item}>
              {getOnErrorLabel(itemProps.item.rawValue)}
            </SelectItem>
          )}
        >
          <SelectTrigger>
            <SelectValue<HookOnError>>
              {(state) => getOnErrorLabel(state.selectedOption())}
            </SelectValue>
          </SelectTrigger>
          <SelectContent />
        </Select>
      </TextField>

      <div class="border-t pt-4">
        <h3 class="mb-4 text-sm font-medium">{m.hook_config()}</h3>

        <Show when={isHttpType()}>
          <div class="space-y-4">
            <TextField
              value={props.value.config.url ?? ''}
              onChange={(value) => updateConfig('url', value)}
              validationState={errors().url ? 'invalid' : 'valid'}
            >
              <Label class="mb-1 block">{m.hook_url()}</Label>
              <TextFieldInput type="url" placeholder={m.hook_urlPlaceholder()} />
              <Show when={errors().url}>
                <TextFieldErrorMessage>{errors().url}</TextFieldErrorMessage>
              </Show>
            </TextField>

            <TextField
              value={props.value.config.method ?? 'POST'}
              onChange={(value) => updateConfig('method', value)}
            >
              <Label class="mb-1 block">{m.hook_httpMethod()}</Label>
              <Select
                value={props.value.config.method ?? 'POST'}
                onChange={(value) => value && updateConfig('method', value)}
                options={['GET', 'POST', 'PUT']}
                placeholder={m.common_select()}
                itemComponent={(itemProps) => (
                  <SelectItem item={itemProps.item}>{itemProps.item.rawValue}</SelectItem>
                )}
              >
                <SelectTrigger>
                  <SelectValue<string>>{(state) => state.selectedOption() ?? 'POST'}</SelectValue>
                </SelectTrigger>
                <SelectContent />
              </Select>
            </TextField>

            <div>
              <div class="mb-2 flex items-center justify-between">
                <Label>{m.hook_headers()}</Label>
                <Button type="button" variant="outline" size="sm" onClick={addHeader}>
                  {m.hook_addHeader()}
                </Button>
              </div>
              <Show
                when={headers().length > 0}
                fallback={<p class="text-sm text-muted-foreground">{m.hook_noHeaders()}</p>}
              >
                <div class="space-y-2">
                  <Index each={headers()}>
                    {(header, index) => (
                      <div class="flex gap-2">
                        <TextField
                          class="flex-1"
                          value={header().key}
                          onChange={(value) => updateHeader(index, 'key', value)}
                        >
                          <TextFieldInput placeholder={m.hook_headerKey()} />
                        </TextField>
                        <TextField
                          class="flex-1"
                          value={header().value}
                          onChange={(value) => updateHeader(index, 'value', value)}
                        >
                          <TextFieldInput placeholder={m.hook_headerValue()} />
                        </TextField>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => removeHeader(index)}
                        >
                          {m.common_remove()}
                        </Button>
                      </div>
                    )}
                  </Index>
                </div>
              </Show>
            </div>

            <TextField
              value={props.value.config.body ?? ''}
              onChange={(value) => updateConfig('body', value)}
            >
              <Label class="mb-1 block">{m.hook_body()}</Label>
              <TextFieldTextArea placeholder={m.hook_bodyPlaceholder()} rows={4} />
            </TextField>
          </div>
        </Show>

        <Show when={isCommandType()}>
          <div class="space-y-4">
            <TextField
              value={props.value.config.command ?? ''}
              onChange={(value) => updateConfig('command', value)}
              validationState={errors().command ? 'invalid' : 'valid'}
            >
              <Label class="mb-1 block">{m.hook_command()}</Label>
              <TextFieldInput placeholder={m.hook_commandPlaceholder()} />
              <Show when={errors().command}>
                <TextFieldErrorMessage>{errors().command}</TextFieldErrorMessage>
              </Show>
            </TextField>

            <TextField
              value={props.value.config.workDir ?? ''}
              onChange={(value) => updateConfig('workDir', value)}
            >
              <Label class="mb-1 block">{m.hook_workDir()}</Label>
              <TextFieldInput placeholder={m.hook_workDirPlaceholder()} />
            </TextField>

            <TextField
              value={props.value.config.timeout?.toString() ?? '30'}
              onChange={(value) => updateConfig('timeout', value ? parseInt(value, 10) : 30)}
            >
              <Label class="mb-1 block">{m.hook_timeout()}</Label>
              <TextFieldInput type="number" />
              <TextFieldDescription>{m.hook_timeoutHelp()}</TextFieldDescription>
            </TextField>
          </div>
        </Show>
      </div>
    </div>
  );
}
