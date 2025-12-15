import { For } from 'solid-js';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { locales, type Locale } from '@/paraglide/runtime';
import { useLocale } from '@/store/locale';
import IconLanguages from '~icons/lucide/languages';

const LOCALE_NAMES: Record<Locale, string> = {
  en: 'English',
  'zh-CN': '简体中文',
};

export default function LanguageSwitcher() {
  const [state, { setLocale }] = useLocale();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        as={Button}
        variant="ghost"
        size="icon"
        class="relative"
        aria-label="Switch language"
      >
        <IconLanguages class="size-[1.2rem]" />
        <span class="sr-only">Switch language</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <For each={locales}>
          {(locale) => (
            <DropdownMenuItem
              onClick={() => setLocale(locale)}
              class="gap-2"
              aria-label={`Switch to ${LOCALE_NAMES[locale]}`}
              aria-current={state.locale === locale ? 'true' : 'false'}
            >
              <span class={state.locale === locale ? 'font-semibold' : ''}>
                {LOCALE_NAMES[locale]}
              </span>
              {state.locale === locale && <span class="ml-auto text-xs">✓</span>}
            </DropdownMenuItem>
          )}
        </For>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
