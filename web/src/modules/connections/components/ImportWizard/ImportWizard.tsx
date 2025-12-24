import { IMPORT_PARSE, IMPORT_EXECUTE } from '@/api/graphql/queries/import';
import { ConnectionsListQuery } from '@/api/graphql/queries/connections';
import { client } from '@/api/graphql/client';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { ImportPreviewItem, ImportResult } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { Component, createSignal, Match, Switch } from 'solid-js';
import { Step1Input } from './Step1Input';
import { Step2Preview } from './Step2Preview';
import { Step3Confirm } from './Step3Confirm';

interface ImportWizardProps {
  isOpen: boolean;
  onClose: () => void;
}

type Step = 1 | 2 | 3;

export const ImportWizard: Component<ImportWizardProps> = (props) => {
  const [step, setStep] = createSignal<Step>(1);
  const [loading, setLoading] = createSignal(false);
  const [previewItems, setPreviewItems] = createSignal<ImportPreviewItem[]>([]);
  const [result, setResult] = createSignal<ImportResult | null>(null);
  const [error, setError] = createSignal<string | null>(null);

  const resetState = () => {
    setStep(1);
    setLoading(false);
    setPreviewItems([]);
    setResult(null);
    setError(null);
  };

  const handleClose = () => {
    resetState();
    props.onClose();
  };

  // Step 1: 解析配置内容 using GraphQL
  const handleParseContent = async (inputContent: string) => {
    setLoading(true);
    setError(null);

    try {
      const parseResult = await client.mutation(IMPORT_PARSE, {
        input: { content: inputContent },
      });

      if (parseResult.error) {
        throw new Error(parseResult.error.message);
      }

      const data = parseResult.data?.import?.parse;

      // Check if it's an error response
      if (data?.__typename === 'ImportParseError') {
        const errorMsg = data.line ? `Line ${data.line}: ${data.error}` : data.error;
        setError(errorMsg);
        return;
      }

      // It's a success response
      if (data?.__typename === 'ImportParseSuccess') {
        const connections = data.connections ?? [];

        if (connections.length === 0) {
          setError(m.import_noValidConnections());
          return;
        }

        // Check for internal duplicates (same name appearing multiple times)
        const nameCount: Record<string, number> = {};
        for (const conn of connections) {
          nameCount[conn.name] = (nameCount[conn.name] || 0) + 1;
        }
        const duplicates = Object.entries(nameCount)
          .filter(([_, count]) => count > 1)
          .map(([name]) => name);

        if (duplicates.length > 0) {
          setError(m.import_duplicateNames({ names: duplicates.join(', ') }));
          return;
        }

        // Fetch existing connections to check for conflicts
        const existingResult = await client.query(ConnectionsListQuery, {});
        const existingItems = existingResult.data?.connection?.list?.items ?? [];
        const existingNames = new Set(existingItems.map((c) => c.name));

        // 转换为前端 ImportPreviewItem
        const items: ImportPreviewItem[] = connections.map((conn) => ({
          name: conn.name,
          type: conn.type,
          config: (conn.config ?? {}) as Record<string, string>,
          selected: !existingNames.has(conn.name), // 默认选中非冲突项
          isConflict: existingNames.has(conn.name),
          isDuplicate: false,
        }));

        setPreviewItems(items);
        setStep(2);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : m.import_parseFailed());
    } finally {
      setLoading(false);
    }
  };

  // Step 2: 确认选择后执行导入 using GraphQL
  const handleConfirmSelection = async (items: ImportPreviewItem[]) => {
    setLoading(true);
    setError(null);

    try {
      const selectedItems = items.filter((item) => item.selected);

      // 检查是否有冲突项被选中（需要覆盖）
      const hasOverwrite = selectedItems.some((item) => item.isConflict);

      // 构建导入请求
      const connections = selectedItems.map((item) => ({
        name: item.editedName ?? item.name,
        type: item.type,
        config: item.editedConfig ?? item.config,
      }));

      const importResult = await client.mutation(IMPORT_EXECUTE, {
        input: {
          connections,
          overwrite: hasOverwrite,
        },
      });

      if (importResult.error) {
        throw new Error(importResult.error.message);
      }

      const data = importResult.data?.import?.execute;
      const importedCount = data?.connections?.length ?? 0;
      const skippedCount = data?.skippedCount ?? 0;

      // Convert to ImportResult format expected by Step3Confirm
      setResult({
        imported: importedCount,
        skipped: skippedCount,
        failed: 0, // GraphQL API doesn't have failed concept in current schema
        errors: [],
      });
      setStep(3);

      // 刷新连接列表 - invalidate GraphQL cache
      client.query(ConnectionsListQuery, {}, { requestPolicy: 'network-only' });
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : m.import_failed());
    } finally {
      setLoading(false);
    }
  };

  const handleBackToStep1 = () => {
    setStep(1);
    setError(null);
  };

  const handleBackToStep2 = () => {
    setStep(2);
    setError(null);
  };

  const getStepDescription = () => {
    switch (step()) {
      case 1:
        return m.import_step1Desc();
      case 2:
        return m.import_step2Desc();
      case 3:
        return m.import_step3Desc();
      default:
        return '';
    }
  };

  return (
    <Dialog open={props.isOpen} onOpenChange={handleClose}>
      <DialogContent class="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{m.import_title()}</DialogTitle>
          <DialogDescription>{getStepDescription()}</DialogDescription>
        </DialogHeader>

        <div class="py-4">
          <Switch>
            <Match when={step() === 1}>
              <Step1Input onNext={handleParseContent} onCancel={handleClose} loading={loading()} />
              {error() && (
                <div class="mt-4 rounded-md bg-error/10 p-3 text-sm text-error-foreground">
                  {error()}
                </div>
              )}
            </Match>

            <Match when={step() === 2}>
              <Step2Preview
                items={previewItems()}
                onBack={handleBackToStep1}
                onNext={handleConfirmSelection}
                onItemsChange={setPreviewItems}
                loading={loading()}
              />
              {error() && (
                <div class="mt-4 rounded-md bg-error/10 p-3 text-sm text-error-foreground">
                  {error()}
                </div>
              )}
            </Match>

            <Match when={step() === 3 && result()}>
              <Step3Confirm result={result()!} onBack={handleBackToStep2} onFinish={handleClose} />
            </Match>
          </Switch>
        </div>
      </DialogContent>
    </Dialog>
  );
};
