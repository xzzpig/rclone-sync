import { Button } from '@/components/ui/button';
import type { ImportResult } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { Component, Show } from 'solid-js';
import IconCheck from '~icons/lucide/check';
import IconAlertTriangle from '~icons/lucide/alert-triangle';

interface Step3ConfirmProps {
  result: ImportResult;
  onBack: () => void;
  onFinish: () => void;
}

export const Step3Confirm: Component<Step3ConfirmProps> = (props) => {
  const createdCount = () => (props.result.createdCount ?? 0) as number;
  const updatedCount = () => (props.result.updatedCount ?? 0) as number;
  const totalCount = () => createdCount() + updatedCount();

  const isFullSuccess = () => createdCount() > 0 && updatedCount() === 0;
  const isPartialSuccess = () => createdCount() > 0 && updatedCount() > 0;
  const isUpdateOnly = () => createdCount() === 0 && updatedCount() > 0;

  return (
    <div class="flex flex-col gap-4">
      {/* 成功状态 - 只创建新连接 */}
      <Show when={isFullSuccess()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-success/10">
            <IconCheck class="size-8 text-success-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_success()}</h3>
            <p class="text-sm text-muted-foreground">
              {m.import_successCount({ count: createdCount() })}
            </p>
          </div>
        </div>
      </Show>

      {/* 部分成功状态 - 既有创建又有更新 */}
      <Show when={isPartialSuccess()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-warning/10">
            <IconAlertTriangle class="size-8 text-warning-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_success()}</h3>
            <p class="text-sm text-muted-foreground">
              {m.import_partialSuccessDesc({
                created: createdCount(),
                updated: updatedCount(),
                total: totalCount(),
              })}
            </p>
          </div>
        </div>
      </Show>

      {/* 只更新状态 */}
      <Show when={isUpdateOnly()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-info/10">
            <IconAlertTriangle class="size-8 text-info-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_success()}</h3>
            <p class="text-sm text-muted-foreground">
              {m.import_successCount({ count: updatedCount() })}
            </p>
          </div>
        </div>
      </Show>

      {/* 统计汇总 */}
      <div class="grid grid-cols-3 gap-4 rounded-md border p-4">
        <div class="text-center">
          <div class="text-2xl font-semibold text-success-foreground">{createdCount()}</div>
          <div class="text-xs text-muted-foreground">{m.common_success()}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold text-info-foreground">{updatedCount()}</div>
          <div class="text-xs text-muted-foreground">{m.connection_update()}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold text-muted-foreground">{totalCount()}</div>
          <div class="text-xs text-muted-foreground">{m.overview_total()}</div>
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-4">
        <Button onClick={props.onFinish}>{m.common_finish()}</Button>
      </div>
    </div>
  );
};
