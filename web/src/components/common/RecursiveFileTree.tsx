import { Button } from '@/components/ui/button';
import type { FileEntry } from '@/lib/types';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { Component, createEffect, createSignal, For, Match, on, Show, Switch } from 'solid-js';
import IconAlertTriangle from '~icons/lucide/alert-triangle';
import IconChevronDown from '~icons/lucide/chevron-down';
import IconChevronRight from '~icons/lucide/chevron-right';
import IconFile from '~icons/lucide/file';
import IconFolder from '~icons/lucide/folder';
import IconFolderOpen from '~icons/lucide/folder-open';
import IconRefreshCw from '~icons/lucide/refresh-cw';

export interface RecursiveFileTreeProps {
  /**
   * 根路径
   */
  path: string;
  /**
   * 加载目录的方法
   */
  loadDirectory: (path: string) => Promise<FileEntry[]>;
  /**
   * 刷新触发器
   */
  refreshTrigger?: number;
  /**
   * 自定义类名
   */
  class?: string;
  /**
   * 空列表提示
   */
  emptyMessage?: string;
  /**
   * 错误提示标题
   */
  errorTitle?: string;
}

// 目录项组件 - 基础属性
interface BaseDirectoryItemProps {
  /**
   * 目录路径
   */
  path: string;
  /**
   * 当前深度
   */
  depth: number;
  /**
   * 加载目录的方法
   */
  loadDirectory: (path: string) => Promise<FileEntry[]>;
}

// 根模式：隐藏自身行并自动加载
interface RootDirectoryItemProps extends BaseDirectoryItemProps {
  name?: never;
  isDir?: never;
  /**
   * 刷新触发器
   */
  refreshTrigger?: number;
  /**
   * 空列表提示
   */
  emptyMessage?: string;
  /**
   * 错误提示标题
   */
  errorTitle?: string;
  /**
   * 是否显示重试按钮
   */
  showRetry?: boolean;
}

// 子节点模式：显示文件/文件夹行
interface ChildDirectoryItemProps extends BaseDirectoryItemProps {
  /**
   * 显示名称
   */
  name: string;
  /**
   * 是否为文件夹，默认 true
   */
  isDir?: boolean;
  refreshTrigger?: never;
  emptyMessage?: never;
  errorTitle?: never;
  showRetry?: never;
}

type DirectoryItemProps = RootDirectoryItemProps | ChildDirectoryItemProps;

const DirectoryItem: Component<DirectoryItemProps> = (props) => {
  // 根模式：name 为空时，隐藏自身行并自动展开
  const isRootMode = () => !props.name;
  const isDirectory = () => props.isDir ?? true;

  const [isExpanded, setIsExpanded] = createSignal(isRootMode());
  const [children, setChildren] = createSignal<FileEntry[]>([]);
  const [isLoading, setIsLoading] = createSignal(isRootMode());
  const [error, setError] = createSignal<string | null>(null);

  const loadChildren = async () => {
    setIsLoading(true);
    setError(null);

    try {
      const entries = await props.loadDirectory(props.path);
      setChildren(entries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  };

  const toggleExpand = async () => {
    if (!isDirectory()) return;

    if (isExpanded()) {
      setIsExpanded(false);
      return;
    }

    setIsExpanded(true);
    await loadChildren();
  };

  // 根模式下自动加载
  createEffect(() => {
    if (isRootMode()) {
      loadChildren();
    }
  });

  // 监听 refreshTrigger 变化重新加载（仅在已展开时）
  createEffect(
    on(
      () => props.refreshTrigger,
      () => {
        if (isExpanded()) {
          loadChildren();
        }
      }
    )
  );

  const FileIcon = () => {
    return (
      <Switch fallback={<IconFile class="size-4" />}>
        <Match when={isDirectory() && isExpanded()}>
          <IconFolderOpen class="size-4 text-blue-500" />
        </Match>
        <Match when={isDirectory() && !isExpanded()}>
          <IconFolder class="size-4 text-blue-500" />
        </Match>
      </Switch>
    );
  };

  const childDepth = () => (isRootMode() ? props.depth : props.depth + 1);
  const contentPadding = () => `${childDepth() * 16 + 8}px`;

  return (
    <div>
      {/* 非根模式显示自身行 */}
      <Show when={!isRootMode()}>
        <div
          class={cn(
            'flex w-fit min-w-full items-center gap-2 py-1.5 px-2 hover:bg-accent cursor-pointer rounded transition-colors',
            !isDirectory() && 'cursor-default'
          )}
          style={{ 'padding-left': `${props.depth * 16 + 8}px` }}
          onClick={toggleExpand}
        >
          <Show when={isDirectory()} fallback={<div class="w-4" />}>
            {isExpanded() ? (
              <IconChevronDown class="size-4 text-muted-foreground" />
            ) : (
              <IconChevronRight class="size-4 text-muted-foreground" />
            )}
          </Show>
          <FileIcon />
          <span class="flex-1 whitespace-nowrap text-sm" title={props.name}>
            {props.name}
          </span>
        </div>
      </Show>

      {/* 展开状态下的内容 */}
      <Show when={isExpanded()}>
        <Show when={isLoading()}>
          <div
            class={cn(
              'flex w-fit min-w-full items-center gap-2 py-1.5 text-sm text-muted-foreground',
              isRootMode() && 'justify-center py-8'
            )}
            style={{ 'padding-left': isRootMode() ? undefined : contentPadding() }}
          >
            <IconRefreshCw class={cn('animate-spin', isRootMode() ? 'size-4' : 'size-3.5')} />
            {m.common_loading()}
          </div>
        </Show>

        <Show when={error()}>
          <Show
            when={isRootMode() && props.showRetry}
            fallback={
              <div
                class={cn(
                  'flex w-fit min-w-full items-center gap-2 py-1.5 text-sm text-destructive',
                  isRootMode() && 'justify-center py-8'
                )}
                style={{ 'padding-left': isRootMode() ? undefined : contentPadding() }}
              >
                <IconAlertTriangle class={cn(isRootMode() ? 'size-4' : 'size-3.5')} />
                <Show when={isRootMode() && props.errorTitle}>{props.errorTitle}: </Show>
                {error()}
              </div>
            }
          >
            <div class="flex flex-col items-center justify-center gap-3 py-8">
              <div class="flex items-center gap-2 text-sm text-destructive">
                <IconAlertTriangle class="size-4" />
                {props.errorTitle ?? m.filter_previewError()}: {error()}
              </div>
              <Button variant="outline" size="sm" onClick={loadChildren}>
                {m.common_retry()}
              </Button>
            </div>
          </Show>
        </Show>

        <Show when={!isLoading() && !error() && children().length === 0}>
          <div
            class={cn(
              'w-fit min-w-full py-1.5 text-sm italic text-muted-foreground',
              isRootMode() && 'py-8 text-center'
            )}
            style={{ 'padding-left': isRootMode() ? undefined : contentPadding() }}
          >
            {props.emptyMessage ?? m.file_browser_empty()}
          </div>
        </Show>

        <Show when={!isLoading() && !error() && children().length > 0}>
          <div class={cn(isRootMode() && 'w-fit min-w-full divide-y')}>
            <For each={children()}>
              {(child) => (
                <DirectoryItem
                  path={child.path}
                  name={child.name}
                  isDir={child.isDir}
                  depth={childDepth()}
                  loadDirectory={props.loadDirectory}
                />
              )}
            </For>
          </div>
        </Show>
      </Show>
    </div>
  );
};

export const RecursiveFileTree: Component<RecursiveFileTreeProps> = (props) => {
  return (
    <div
      class={cn(
        'max-h-64 w-0 min-w-full overflow-auto rounded-lg border bg-background',
        props.class
      )}
    >
      <DirectoryItem
        path={props.path}
        depth={0}
        loadDirectory={props.loadDirectory}
        refreshTrigger={props.refreshTrigger}
        emptyMessage={props.emptyMessage}
        errorTitle={props.errorTitle}
        showRetry={true}
      />
    </div>
  );
};
