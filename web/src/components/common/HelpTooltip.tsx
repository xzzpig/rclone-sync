import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useIsMobile } from '@/lib/media-query';
import { JSX, Show } from 'solid-js';
import IconInfo from '~icons/lucide/info';

interface HelpTooltipProps {
  content: string;
  trigger?: JSX.Element;
  children?: JSX.Element;
}

export const HelpTooltip = (props: HelpTooltipProps) => {
  const isMobile = useIsMobile();

  const Trigger = () =>
    props.trigger ?? props.children ?? <IconInfo class="size-[1em] text-muted-foreground" />;

  return (
    <Show
      when={isMobile()}
      fallback={
        <Tooltip>
          <TooltipTrigger class="flex cursor-help items-center" onClick={(e) => e.preventDefault()}>
            <Trigger />
          </TooltipTrigger>
          <TooltipContent class="whitespace-pre-wrap text-left font-normal">
            {props.content}
          </TooltipContent>
        </Tooltip>
      }
    >
      <Popover>
        <PopoverTrigger
          class="flex cursor-pointer items-center"
          onClick={(e) => e.preventDefault()}
        >
          <Trigger />
        </PopoverTrigger>
        <PopoverContent class="max-w-xs whitespace-pre-wrap text-left text-sm font-normal">
          {props.content}
        </PopoverContent>
      </Popover>
    </Show>
  );
};
