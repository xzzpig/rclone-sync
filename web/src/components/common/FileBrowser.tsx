import { Button } from '@/components/ui/button';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { TextField, TextFieldInput } from '@/components/ui/text-field';
import { ApiError } from '@/lib/api';
import { FileEntry } from '@/lib/types';
import { cn } from '@/lib/utils';
import { useQuery } from '@tanstack/solid-query';
import { Component, createEffect, createSignal, For, Show } from 'solid-js';
import IconFolder from '~icons/lucide/folder';
import IconRefreshCw from '~icons/lucide/refresh-cw';

export interface FileBrowserProps {
  initialPath?: string;
  rootLabel?: string;
  icon?: Component<{ class?: string }>;
  loadDirectory: (path: string) => Promise<FileEntry[]>;
  onSelect: (path: string) => void;
  class?: string;
}

interface BreadcrumbItem {
  name: string;
  path: string;
}

export const FileBrowser: Component<FileBrowserProps> = (props) => {
  const defaultPath = props.initialPath ?? '/';
  const [currentPath, setCurrentPath] = createSignal(defaultPath);
  const [inputPath, setInputPath] = createSignal(defaultPath);
  const [selectedPath, setSelectedPath] = createSignal<string | null>(null);

  // Use createQuery for caching file list
  const query = useQuery(() => ({
    queryKey: ['files', props.rootLabel, currentPath()],
    queryFn: () => props.loadDirectory(currentPath()),
    staleTime: 1000 * 60 * 5, // 5 minutes cache
    retry: 1,
  }));

  // Sync inputPath with currentPath
  createEffect(() => {
    setInputPath(currentPath());
  });

  const loadDirectory = (path: string) => {
    setCurrentPath(path);
  };

  const handleEntryClick = (entry: FileEntry) => {
    if (entry.is_dir) {
      // Normalize the path to avoid double slashes
      let normalizedPath = entry.path;

      // Remove duplicate slashes
      while (normalizedPath.includes('//')) {
        normalizedPath = normalizedPath.replace('//', '/');
      }

      loadDirectory(normalizedPath);
    }
  };

  const handleSelect = () => {
    const path = inputPath().trim();
    if (!path) return;

    // If path is different from current, load the directory first
    if (path !== currentPath()) {
      loadDirectory(path);
    } else {
      // Path hasn't changed, select it directly
      props.onSelect(path);
    }
  };

  const getBreadcrumbs = (): BreadcrumbItem[] => {
    let path = currentPath();

    // Handle null or undefined path
    if (!path) {
      return [{ name: props.rootLabel ?? 'Root', path: '/' }];
    }

    // Normalize path - remove duplicate slashes
    while (path.includes('//')) {
      path = path.replace('//', '/');
    }

    const parts = path.split('/').filter(Boolean);

    const breadcrumbs: BreadcrumbItem[] = [];
    let accumulated = '';

    if (path.startsWith('/')) {
      breadcrumbs.push({ name: props.rootLabel ?? 'Root', path: '/' });
    }

    for (const part of parts) {
      if (breadcrumbs.length === 0) {
        accumulated += part;
        breadcrumbs.push({ name: part, path: accumulated });
      } else {
        accumulated += '/' + part;
        breadcrumbs.push({ name: part, path: accumulated });
      }
    }

    return breadcrumbs;
  };

  return (
    <div class={cn('flex flex-col h-full min-h-0', props.class)}>
      {/* Breadcrumb Navigation */}
      <div class="flex items-center gap-1 overflow-x-auto border-b px-4 py-2">
        <Show when={props.icon}>
          {(IconComponent) => {
            const Icon = IconComponent();
            return <Icon class="size-4 shrink-0" />;
          }}
        </Show>
        <Breadcrumb class="flex-1 overflow-x-auto">
          <BreadcrumbList class="flex-nowrap whitespace-nowrap sm:gap-1">
            <For each={getBreadcrumbs()}>
              {(item, index) => (
                <>
                  <Show when={index() > 0}>
                    <BreadcrumbSeparator />
                  </Show>
                  <BreadcrumbItem>
                    <BreadcrumbLink as="button" onClick={() => loadDirectory(item.path)}>
                      {item.name}
                    </BreadcrumbLink>
                  </BreadcrumbItem>
                </>
              )}
            </For>
          </BreadcrumbList>
        </Breadcrumb>
        <Button
          variant="ghost"
          size="sm"
          class="shrink-0"
          onClick={() => query.refetch()}
          disabled={query.isFetching}
          title="Refresh"
        >
          <IconRefreshCw class={cn('w-4 h-4', query.isFetching && 'animate-spin')} />
        </Button>
      </div>

      {/* File List */}
      <div class="flex-1 overflow-y-auto">
        <Show when={query.isPending}>
          <div class="flex h-32 items-center justify-center">
            <div class="text-sm text-muted-foreground">Loading...</div>
          </div>
        </Show>

        <Show when={query.isError}>
          <div class="flex h-32 items-center justify-center">
            <div class="text-sm text-destructive">
              {query.error instanceof ApiError
                ? (query.error.details ?? query.error.message)
                : query.error instanceof Error
                  ? query.error.message
                  : 'Failed to load directory'}
            </div>
          </div>
        </Show>

        <Show when={query.isSuccess}>
          <div class="divide-y">
            <For each={query.data}>
              {(entry) => (
                <div
                  class={cn(
                    'flex items-center gap-3 px-4 py-3 hover:bg-accent cursor-pointer transition-colors',
                    selectedPath() === entry.path && 'bg-accent'
                  )}
                  onClick={() => {
                    setSelectedPath(entry.path);
                    handleEntryClick(entry);
                  }}
                >
                  <IconFolder class="size-5 shrink-0 text-blue-500" />
                  <span class="flex-1 truncate">{entry.name}</span>
                </div>
              )}
            </For>

            <Show when={(query.data?.length ?? 0) === 0}>
              <div class="px-4 py-8 text-center text-sm text-muted-foreground">
                This directory is empty
              </div>
            </Show>
          </div>
        </Show>
      </div>

      {/* Selection Bar */}
      <div class="flex items-center gap-2 border-t bg-muted/50 px-4 py-3">
        <TextField class="flex-1">
          <TextFieldInput
            value={inputPath()}
            onInput={(e: InputEvent) => setInputPath((e.currentTarget as HTMLInputElement).value)}
            placeholder="Enter path (e.g., /home/user or /Documents)"
            class="h-9"
            onKeyPress={(e: KeyboardEvent) => {
              if (e.key === 'Enter') {
                handleSelect();
              }
            }}
          />
        </TextField>
        <Button onClick={handleSelect} disabled={!inputPath().trim()} class="shrink-0">
          <Show when={currentPath() === inputPath().trim()}>Select</Show>
          <Show when={currentPath() !== inputPath().trim()}>Go</Show>
        </Button>
      </div>
    </div>
  );
};
