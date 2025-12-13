import { JobStatus, TaskStatus } from '@/lib/types';
import { cn } from '@/lib/utils';
import { Component, Match, Show, Switch } from 'solid-js';
import IconBan from '~icons/lucide/ban';
import IconCheckCircle2 from '~icons/lucide/check-circle-2';
import IconClock from '~icons/lucide/clock';
import IconPauseCircle from '~icons/lucide/pause-circle';
import IconRefresh from '~icons/lucide/refresh-cw';
import IconXCircle from '~icons/lucide/x-circle';
import { HelpTooltip } from './HelpTooltip';

interface StatusIconProps {
  status: TaskStatus | JobStatus;
  class?: string;
  showIdle?: boolean;
}

const StatusIcon: Component<StatusIconProps> = (props) => {
  // 如果 showIdle 为 false 且状态为 idle，则不显示任何内容
  const shouldShow = () => props.showIdle !== false || props.status !== 'idle';

  return (
    <Show when={shouldShow()}>
      <Switch>
        <Match when={props.status === 'running'}>
          <HelpTooltip content="Running">
            <IconRefresh class={cn('h-5 w-5 animate-spin text-blue-500', props.class)} />
          </HelpTooltip>
        </Match>
        <Match when={props.status === 'success'}>
          <HelpTooltip content="Success">
            <IconCheckCircle2 class={cn('h-5 w-5 text-green-500', props.class)} />
          </HelpTooltip>
        </Match>
        <Match when={props.status === 'failed'}>
          <HelpTooltip content="Failed">
            <IconXCircle class={cn('h-5 w-5 text-red-500', props.class)} />
          </HelpTooltip>
        </Match>
        <Match when={props.status === 'pending'}>
          <HelpTooltip content="Pending">
            <IconClock class={cn('h-5 w-5 text-yellow-500', props.class)} />
          </HelpTooltip>
        </Match>
        <Match when={props.status === 'canceled'}>
          <HelpTooltip content="Canceled">
            <IconBan class={cn('h-5 w-5 text-gray-500', props.class)} />
          </HelpTooltip>
        </Match>
        <Match when={props.status === 'idle'}>
          <HelpTooltip content="Idle">
            <IconPauseCircle class={cn('h-5 w-5 text-gray-400', props.class)} />
          </HelpTooltip>
        </Match>
      </Switch>
    </Show>
  );
};

export default StatusIcon;
