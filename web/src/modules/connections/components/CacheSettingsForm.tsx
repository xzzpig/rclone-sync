import { Component, Show } from 'solid-js';
import { Switch, SwitchControl, SwitchLabel, SwitchThumb } from '@/components/ui/switch';
import {
  TextField,
  TextFieldDescription,
  TextFieldInput,
  TextFieldLabel,
} from '@/components/ui/text-field';
import * as m from '@/paraglide/messages.js';
import type { CacheOptions } from '@/lib/types';
import { cn } from '@/lib/utils';

interface CacheSettingsFormProps {
  options: CacheOptions;
  onChange: (options: CacheOptions) => void;
  class?: string;
}

export const CacheSettingsForm: Component<CacheSettingsFormProps> = (props) => {
  return (
    <div class={cn('space-y-6', props.class)}>
      <Switch
        class="relative flex w-full items-center justify-between"
        checked={props.options.enabled}
        onChange={(checked) => props.onChange({ ...props.options, enabled: checked })}
      >
        <SwitchLabel class="cursor-pointer text-sm font-medium">
          {m.connection_cache_enabled()}
        </SwitchLabel>
        <SwitchControl>
          <SwitchThumb />
        </SwitchControl>
      </Switch>

      <Show when={props.options.enabled}>
        <div class="grid gap-4 sm:grid-cols-2">
          <TextField>
            <TextFieldLabel>{m.connection_cache_infoAge()}</TextFieldLabel>
            <TextFieldInput
              value={props.options.infoAge ?? ''}
              onInput={(e) => props.onChange({ ...props.options, infoAge: e.currentTarget.value })}
              placeholder="24h"
            />
            <TextFieldDescription>{m.connection_cache_infoAge_help()}</TextFieldDescription>
          </TextField>

          <TextField>
            <TextFieldLabel>{m.connection_cache_poll()}</TextFieldLabel>
            <TextFieldInput
              value={props.options.changeNotifyPoll ?? ''}
              onInput={(e) =>
                props.onChange({ ...props.options, changeNotifyPoll: e.currentTarget.value })
              }
              placeholder="1m"
            />
            <TextFieldDescription>{m.connection_cache_poll_help()}</TextFieldDescription>
          </TextField>
        </div>
      </Show>
    </div>
  );
};
