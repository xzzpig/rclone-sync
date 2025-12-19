import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { LoadStatus } from '@/lib/types';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages';
import { Match, Show, Switch } from 'solid-js';
import IconCheckCircle from '~icons/lucide/check-circle';
import IconLoader from '~icons/lucide/loader-2';
import IconAlertCircle from '~icons/lucide/alert-circle';

interface ConnectionStatusBadgeProps {
  status: LoadStatus;
  error?: string;
  size?: 'sm' | 'md';
}

export const ConnectionStatusBadge = (props: ConnectionStatusBadgeProps) => {
  const iconClass = () => (props.size === 'sm' ? 'h-3 w-3' : 'h-3.5 w-3.5');
  const textClass = () => (props.size === 'sm' ? 'text-xs' : 'text-sm');

  const badge = () => (
    <Switch>
      <Match when={props.status === 'loaded'}>
        <Badge variant="success" class={cn('gap-1', textClass())}>
          <IconCheckCircle class={iconClass()} />
          <span>{m.connection_statusLoaded()}</span>
        </Badge>
      </Match>
      <Match when={props.status === 'loading'}>
        <Badge variant="secondary" class={cn('gap-1', textClass())}>
          <IconLoader class={cn(iconClass(), 'animate-spin')} />
          <span>{m.connection_statusLoading()}</span>
        </Badge>
      </Match>
      <Match when={props.status === 'error'}>
        <Badge variant="error" class={cn('gap-1', textClass())}>
          <IconAlertCircle class={iconClass()} />
          <span>{m.connection_statusError()}</span>
        </Badge>
      </Match>
    </Switch>
  );

  return (
    <Show when={props.status === 'error' && props.error} fallback={badge()}>
      <Tooltip>
        <TooltipTrigger>{badge()}</TooltipTrigger>
        <TooltipContent>
          <p class="max-w-xs text-sm">{props.error}</p>
        </TooltipContent>
      </Tooltip>
    </Show>
  );
};
