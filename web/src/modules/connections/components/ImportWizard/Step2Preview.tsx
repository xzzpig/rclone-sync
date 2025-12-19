import { testUnsavedConnection } from '@/api/connections';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { TextField, TextFieldInput } from '@/components/ui/text-field';
import type { ImportPreviewItem } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { Component, createSignal, For, Show } from 'solid-js';
import IconArrowLeft from '~icons/lucide/arrow-left';
import IconArrowRight from '~icons/lucide/arrow-right';
import IconAlertTriangle from '~icons/lucide/alert-triangle';
import IconCheck from '~icons/lucide/check';
import IconLoader from '~icons/lucide/loader-2';
import IconPencil from '~icons/lucide/pencil';
import IconPlay from '~icons/lucide/play';
import IconTrash from '~icons/lucide/trash-2';
import IconX from '~icons/lucide/x';
import { EditImportConfigDialog } from './EditImportConfigDialog';

interface Step2PreviewProps {
  items: ImportPreviewItem[];
  onBack: () => void;
  onNext: (items: ImportPreviewItem[]) => void;
  onItemsChange: (items: ImportPreviewItem[]) => void;
  loading?: boolean;
}

type TestStatus = 'idle' | 'testing' | 'success' | 'failed';

interface ItemTestStatus {
  status: TestStatus;
  error?: string;
}

export const Step2Preview: Component<Step2PreviewProps> = (props) => {
  const [testStatuses, setTestStatuses] = createSignal<Record<string, ItemTestStatus>>({});
  const [editingItem, setEditingItem] = createSignal<ImportPreviewItem | null>(null);
  const [editingIndex, setEditingIndex] = createSignal<number | null>(null);

  const openEditDialog = (index: number) => {
    setEditingItem(props.items[index]);
    setEditingIndex(index);
  };

  const closeEditDialog = () => {
    setEditingItem(null);
    setEditingIndex(null);
  };

  const handleEditSave = (config: Record<string, string>) => {
    const index = editingIndex();
    if (index !== null) {
      const newItems = props.items.map((item, i) =>
        i === index ? { ...item, editedConfig: config } : item
      );
      props.onItemsChange(newItems);
    }
    closeEditDialog();
  };

  const toggleItem = (index: number) => {
    const newItems = props.items.map((item, i) =>
      i === index ? { ...item, selected: !item.selected } : item
    );
    props.onItemsChange(newItems);
  };

  const updateItemName = (index: number, newName: string) => {
    const newItems = props.items.map((item, i) =>
      i === index ? { ...item, editedName: newName } : item
    );
    props.onItemsChange(newItems);
  };

  const removeItem = (index: number) => {
    const newItems = props.items.filter((_, i) => i !== index);
    props.onItemsChange(newItems);
  };

  const testItem = async (index: number) => {
    const item = props.items[index];
    const key = item.name;

    setTestStatuses((prev) => ({
      ...prev,
      [key]: { status: 'testing' },
    }));

    try {
      await testUnsavedConnection(item.type, item.editedConfig ?? item.config);
      setTestStatuses((prev) => ({
        ...prev,
        [key]: { status: 'success' },
      }));
    } catch (err) {
      setTestStatuses((prev) => ({
        ...prev,
        [key]: {
          status: 'failed',
          error: err instanceof Error ? err.message : m.import_testFailed(),
        },
      }));
    }
  };

  const getTestStatus = (name: string): ItemTestStatus => {
    return testStatuses()[name] ?? { status: 'idle' };
  };

  const selectedCount = () => props.items.filter((item) => item.selected).length;
  const conflictCount = () => props.items.filter((item) => item.isConflict).length;

  const handleNext = () => {
    props.onNext(props.items);
  };

  const getDisplayName = (item: ImportPreviewItem) => {
    return item.editedName ?? item.name;
  };

  const isNameChanged = (item: ImportPreviewItem) => {
    return item.editedName && item.editedName !== item.name;
  };

  return (
    <div class="flex flex-col gap-4">
      <div class="text-sm text-muted-foreground">
        {m.import_foundConnections({ count: props.items.length })}
      </div>

      <Show when={conflictCount() > 0}>
        <div class="flex items-center gap-2 rounded-md bg-warning/10 p-3 text-sm text-warning-foreground">
          <IconAlertTriangle class="size-4 shrink-0" />
          <span>{m.import_conflictWarning({ count: conflictCount() })}</span>
        </div>
      </Show>

      <div class="max-h-[350px] space-y-2 overflow-y-auto rounded-md border p-2">
        <For each={props.items}>
          {(item, index) => {
            const testStatus = () => getTestStatus(item.name);

            return (
              <div
                class="flex items-start gap-3 rounded-md border p-3 transition-colors hover:bg-muted/50"
                classList={{ 'border-warning': item.isConflict && item.selected }}
              >
                <Checkbox
                  checked={item.selected}
                  onChange={() => toggleItem(index())}
                  class="mt-2"
                />

                <div class="flex flex-1 flex-col gap-2">
                  <div class="flex items-center gap-2">
                    <TextField
                      value={getDisplayName(item)}
                      onChange={(value: string) => updateItemName(index(), value)}
                      class="flex-1"
                      disabled={!item.selected}
                    >
                      <TextFieldInput class="h-8" placeholder={m.connection_connectionName()} />
                    </TextField>
                    <span class="text-xs text-muted-foreground">{item.type}</span>
                  </div>

                  <Show when={item.isConflict}>
                    <div class="flex items-center gap-1 text-xs text-warning-foreground">
                      <IconAlertTriangle class="size-3" />
                      <span>
                        {isNameChanged(item)
                          ? m.import_willCreateNew()
                          : item.selected
                            ? m.import_willOverwrite()
                            : m.import_skipped()}
                      </span>
                    </div>
                  </Show>

                  {/* 测试状态显示 */}
                  <Show when={testStatus().status !== 'idle'}>
                    <div
                      class="flex items-center gap-1 text-xs"
                      classList={{
                        'text-muted-foreground': testStatus().status === 'testing',
                        'text-success-foreground': testStatus().status === 'success',
                        'text-error-foreground': testStatus().status === 'failed',
                      }}
                    >
                      <Show when={testStatus().status === 'testing'}>
                        <IconLoader class="size-3 animate-spin" />
                        <span>{m.connection_statusLoading()}</span>
                      </Show>
                      <Show when={testStatus().status === 'success'}>
                        <IconCheck class="size-3" />
                        <span>{m.connection_statusLoaded()}</span>
                      </Show>
                      <Show when={testStatus().status === 'failed'}>
                        <IconX class="size-3" />
                        <span>{testStatus().error ?? m.import_testFailed()}</span>
                      </Show>
                    </div>
                  </Show>
                </div>

                {/* 操作按钮 */}
                <div class="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    class="size-8 p-0"
                    onClick={() => testItem(index())}
                    disabled={testStatus().status === 'testing'}
                    title={m.common_test()}
                  >
                    <IconPlay class="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="size-8 p-0"
                    onClick={() => openEditDialog(index())}
                    title={m.import_editConfig()}
                  >
                    <IconPencil class="size-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    class="size-8 p-0 text-error-foreground hover:text-error-foreground"
                    onClick={() => removeItem(index())}
                    title={m.import_removeFromList()}
                  >
                    <IconTrash class="size-4" />
                  </Button>
                </div>
              </div>
            );
          }}
        </For>

        <Show when={props.items.length === 0}>
          <div class="py-8 text-center text-muted-foreground">{m.import_noConnections()}</div>
        </Show>
      </div>

      <div class="text-sm text-muted-foreground">
        {m.import_selectedCount({ count: selectedCount() })}
      </div>

      <div class="flex justify-between gap-2 pt-4">
        <Button variant="outline" onClick={props.onBack} class="gap-2">
          <IconArrowLeft class="size-4" />
          {m.common_back()}
        </Button>
        <Button
          onClick={handleNext}
          disabled={selectedCount() === 0 || props.loading}
          class="gap-2"
        >
          {props.loading ? m.import_importing() : m.common_import()}
          <IconArrowRight class="size-4" />
        </Button>
      </div>

      {/* 编辑配置对话框 */}
      <EditImportConfigDialog
        item={editingItem()}
        isOpen={editingItem() !== null}
        onClose={closeEditDialog}
        onSave={handleEditSave}
      />
    </div>
  );
};
