import { executeImport, parseImport } from '@/api/connections';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { ImportPreviewItem, ImportResult } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { useQueryClient } from '@tanstack/solid-query';
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
  const queryClient = useQueryClient();

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

  // Step 1: 解析配置内容
  const handleParseContent = async (inputContent: string) => {
    setLoading(true);
    setError(null);

    try {
      const parseResult = await parseImport(inputContent);

      if (parseResult.connections.length === 0) {
        setError(m.import_noValidConnections());
        return;
      }

      // 检查内部重复
      if (parseResult.validation?.internal_duplicates?.length) {
        setError(
          m.import_duplicateNames({ names: parseResult.validation.internal_duplicates.join(', ') })
        );
        return;
      }

      // 转换为前端 ImportPreviewItem
      const conflictSet = new Set(parseResult.validation?.conflicts ?? []);

      const items: ImportPreviewItem[] = parseResult.connections.map((conn) => ({
        ...conn,
        selected: !conflictSet.has(conn.name), // 默认选中非冲突项
        isConflict: conflictSet.has(conn.name),
        isDuplicate: false,
      }));

      setPreviewItems(items);
      setStep(2);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : m.import_parseFailed());
    } finally {
      setLoading(false);
    }
  };

  // Step 2: 确认选择后执行导入
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

      const importResult = await executeImport({
        connections,
        overwrite: hasOverwrite,
      });

      setResult(importResult);
      setStep(3);

      // 刷新连接列表
      await queryClient.invalidateQueries({ queryKey: ['connections'] });
      // 刷新单个连接查询（用于 Settings 页面等）
      await queryClient.invalidateQueries({ queryKey: ['connection'] });
      await queryClient.invalidateQueries({ queryKey: ['connectionConfig'] });
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
