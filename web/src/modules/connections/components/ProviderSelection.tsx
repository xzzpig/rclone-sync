import { TextField, TextFieldInput } from '@/components/ui/text-field';
import type { RcloneProvider } from '@/lib/types';
import { createMemo, createSignal, For, Show } from 'solid-js';
import IconSearch from '~icons/lucide/search';

interface ProviderSelectionProps {
  providers: RcloneProvider[];
  onSelect: (provider: RcloneProvider) => void;
}

export const ProviderSelection = (props: ProviderSelectionProps) => {
  const [searchTerm, setSearchTerm] = createSignal('');

  const filteredProviders = createMemo(() => {
    const providers = props.providers;
    if (!providers) return [];
    const term = searchTerm().toLowerCase().trim();
    if (!term) return providers;
    return providers.filter(
      (p) => p.name.toLowerCase().includes(term) || p.description.toLowerCase().includes(term)
    );
  });

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-4 p-4">
      <TextField value={searchTerm()} onChange={setSearchTerm} class="w-full">
        <div class="relative">
          <IconSearch
            class="absolute left-2.5 top-2.5 size-4 text-muted-foreground"
            aria-hidden="true"
          />
          <TextFieldInput
            type="search"
            placeholder="Search providers..."
            class="pl-9"
            aria-label="Search cloud providers"
          />
        </div>
      </TextField>

      <div class="flex-1 overflow-y-auto pr-1" role="region" aria-label="Cloud provider selection">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3" role="group">
          <For each={filteredProviders()}>
            {(provider) => (
              <button
                class="flex h-auto min-h-20 flex-col items-start justify-start rounded-lg border border-border p-3 text-left outline-none ring-offset-background transition-all hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => props.onSelect(provider)}
                aria-label={`Select ${provider.name} cloud provider`}
              >
                <span class="mb-1 text-sm font-medium">{provider.name}</span>
                <span
                  class="line-clamp-2 text-xs leading-relaxed text-muted-foreground"
                  aria-hidden="true"
                >
                  {provider.description}
                </span>
              </button>
            )}
          </For>
          <Show when={filteredProviders().length === 0}>
            <div class="col-span-full py-8 text-center text-muted-foreground">
              No providers found.
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
};
