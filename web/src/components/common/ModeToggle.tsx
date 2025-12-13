import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useColorMode } from '@kobalte/core';
import IconMonitor from '~icons/lucide/monitor';
import IconMoon from '~icons/lucide/moon';
import IconSun from '~icons/lucide/sun';

export default function ModeToggle() {
  const { colorMode, setColorMode } = useColorMode();

  const getIcon = () => {
    const mode = colorMode();
    if (mode === 'light') {
      return <IconSun class="size-[1.2rem]" />;
    } else if (mode === 'dark') {
      return <IconMoon class="size-[1.2rem]" />;
    } else {
      return <IconMonitor class="size-[1.2rem]" />;
    }
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger as={Button} variant="ghost" size="icon" class="relative">
        {getIcon()}
        <span class="sr-only">Toggle theme</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={() => setColorMode('light')} class="gap-2">
          <IconSun class="size-4" />
          <span>Light</span>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setColorMode('dark')} class="gap-2">
          <IconMoon class="size-4" />
          <span>Dark</span>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setColorMode('system')} class="gap-2">
          <IconMonitor class="size-4" />
          <span>System</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
