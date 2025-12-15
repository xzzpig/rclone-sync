import ModeToggle from '@/components/common/ModeToggle';
import { Button } from '@/components/ui/button';
import * as m from '@/paraglide/messages.js';
import { A } from '@solidjs/router';
import { Component, Show } from 'solid-js';
import IconArrowLeft from '~icons/lucide/arrow-left';

interface MobileHeaderProps {
  title?: string;
  showBack?: boolean;
}

const MobileHeader: Component<MobileHeaderProps> = (props) => {
  return (
    <div class="flex items-center justify-between border-b border-border bg-background p-4 text-foreground md:hidden">
      <div class="flex items-center">
        <Show when={props.showBack}>
          <Button
            as={A}
            href="/"
            variant="ghost"
            size="icon"
            class="mr-2"
            aria-label={m.common_back()}
          >
            <IconArrowLeft class="size-6" />
          </Button>
        </Show>
        <h1 class="truncate text-lg font-bold">{props.title ?? m.app_title()}</h1>
      </div>
      <ModeToggle />
    </div>
  );
};

export default MobileHeader;
