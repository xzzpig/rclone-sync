/**
 * FilterPreviewPanel - 过滤器预览面板
 *
 * 显示过滤规则应用后的文件列表预览，支持源端和目标端切换，
 * 懒加载目录内容，规则修改后自动刷新（500ms 防抖）。
 */
import { client } from '@/api/graphql/client';
import { FilesListQuery } from '@/api/graphql/queries/files';
import { RecursiveFileTree } from '@/components/common/RecursiveFileTree';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Tabs, TabsContent, TabsIndicator, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { FileEntry } from '@/lib/types';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { Component, createEffect, createSignal, on, onCleanup, Show } from 'solid-js';
import IconChevronDown from '~icons/lucide/chevron-down';
import IconChevronRight from '~icons/lucide/chevron-right';

export interface FilterPreviewPanelProps {
  /**
   * 连接 ID
   */
  connectionId: string;
  /**
   * 源路径（本地路径）
   */
  sourcePath: string;
  /**
   * 远程路径
   */
  remotePath: string;
  /**
   * 过滤规则列表
   */
  filters: string[];
  /**
   * 是否禁用
   */
  disabled?: boolean;
  /**
   * 自定义类名
   */
  class?: string;
}

export const FilterPreviewPanel: Component<FilterPreviewPanelProps> = (props) => {
  const [isOpen, setIsOpen] = createSignal(false);
  const [activeTab, setActiveTab] = createSignal<'source' | 'remote'>('remote');
  const [refreshTrigger, setRefreshTrigger] = createSignal(0);

  // Load remote directory with filters
  const loadRemoteDirectory = async (path: string): Promise<FileEntry[]> => {
    const result = await client.query(FilesListQuery, {
      connectionId: props.connectionId,
      path,
      filters: props.filters.length > 0 ? props.filters : null,
      includeFiles: true,
      basePath: props.remotePath,
    });

    if (result.error) {
      throw new Error(result.error.message);
    }

    return result.data?.file.list ?? [];
  };

  // Load local directory with filters (connectionId: null)
  const loadLocalDirectory = async (path: string): Promise<FileEntry[]> => {
    const result = await client.query(FilesListQuery, {
      connectionId: null,
      path,
      filters: props.filters.length > 0 ? props.filters : null,
      includeFiles: true,
      basePath: props.sourcePath,
    });

    if (result.error) {
      throw new Error(result.error.message);
    }

    return result.data?.file.list ?? [];
  };

  // 防抖刷新 (500ms)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  createEffect(
    on(
      () => props.filters,
      () => {
        if (!isOpen()) return;

        if (debounceTimer) {
          clearTimeout(debounceTimer);
        }

        debounceTimer = setTimeout(() => {
          setRefreshTrigger((prev) => prev + 1);
        }, 500);
      },
      { defer: true }
    )
  );

  onCleanup(() => {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }
  });

  // 当面板打开时触发初始加载
  const handleOpenChange = (open: boolean) => {
    setIsOpen(open);
    if (open) {
      setRefreshTrigger((prev) => prev + 1);
    }
  };

  return (
    <Collapsible open={isOpen()} onOpenChange={handleOpenChange} class={cn('', props.class)}>
      <CollapsibleTrigger
        as={Button}
        variant="ghost"
        size="sm"
        class="w-full justify-start gap-2 px-0 font-normal hover:bg-transparent"
        disabled={props.disabled}
      >
        <Show when={isOpen()} fallback={<IconChevronRight class="size-4" />}>
          <IconChevronDown class="size-4" />
        </Show>
        {m.filter_previewTitle()}
      </CollapsibleTrigger>

      <CollapsibleContent class="mt-2">
        <Tabs value={activeTab()} onChange={(value) => setActiveTab(value as 'source' | 'remote')}>
          <TabsList class="w-full">
            <TabsIndicator />
            <TabsTrigger value="remote" class="flex-1">
              {m.wizard_remote()}
            </TabsTrigger>
            <TabsTrigger value="source" class="flex-1">
              {m.wizard_local()}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="remote" class="mt-2 min-w-0 p-0">
            <RecursiveFileTree
              path={props.remotePath}
              loadDirectory={loadRemoteDirectory}
              refreshTrigger={refreshTrigger()}
            />
          </TabsContent>

          <TabsContent value="source" class="mt-2 min-w-0 p-0">
            <RecursiveFileTree
              path={props.sourcePath}
              loadDirectory={loadLocalDirectory}
              refreshTrigger={refreshTrigger()}
            />
          </TabsContent>
        </Tabs>

        <p class="mt-2 text-xs text-muted-foreground">{m.filter_previewAutoRefresh()}</p>
      </CollapsibleContent>
    </Collapsible>
  );
};
