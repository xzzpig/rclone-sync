import { Button } from '@/components/ui/button';
import type { ImportResult } from '@/lib/types';
import * as m from '@/paraglide/messages';
import { Component, For, Show } from 'solid-js';
import IconCheck from '~icons/lucide/check';
import IconAlertTriangle from '~icons/lucide/alert-triangle';
import IconX from '~icons/lucide/x';

interface Step3ConfirmProps {
  result: ImportResult;
  onBack: () => void;
  onFinish: () => void;
}

export const Step3Confirm: Component<Step3ConfirmProps> = (props) => {
  const isFullSuccess = () => props.result.failed === 0 && props.result.skipped === 0;
  const isPartialSuccess = () =>
    props.result.imported > 0 && (props.result.failed > 0 || props.result.skipped > 0);
  const isFullFailure = () => props.result.imported === 0 && props.result.failed > 0;

  return (
    <div class="flex flex-col gap-4">
      {/* 成功状态 */}
      <Show when={isFullSuccess()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-success/10">
            <IconCheck class="size-8 text-success-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_success()}</h3>
            <p class="text-sm text-muted-foreground">
              {m.import_successCount({ count: props.result.imported })}
            </p>
          </div>
        </div>
      </Show>

      {/* 部分成功状态 */}
      <Show when={isPartialSuccess()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-warning/10">
            <IconAlertTriangle class="size-8 text-warning-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_partialSuccess()}</h3>
            <p class="text-sm text-muted-foreground">
              {m.import_partialSuccessDesc({
                imported: props.result.imported,
                skipped: props.result.skipped,
                failed: props.result.failed,
              })}
            </p>
          </div>
        </div>
      </Show>

      {/* 完全失败状态 */}
      <Show when={isFullFailure()}>
        <div class="flex flex-col items-center gap-3 py-6">
          <div class="flex size-16 items-center justify-center rounded-full bg-error/10">
            <IconX class="size-8 text-error-foreground" />
          </div>
          <div class="text-center">
            <h3 class="text-lg font-medium">{m.import_failed()}</h3>
            <p class="text-sm text-muted-foreground">{m.import_allFailed()}</p>
          </div>
        </div>
      </Show>

      {/* 错误详情 */}
      <Show when={props.result.errors && props.result.errors.length > 0}>
        <div class="rounded-md border border-error/20 bg-error/5 p-4">
          <h4 class="mb-2 font-medium text-error-foreground">{m.import_errorDetails()}</h4>
          <ul class="space-y-1 text-sm text-error-foreground">
            <For each={props.result.errors}>
              {(error) => (
                <li class="flex items-start gap-2">
                  <IconX class="mt-0.5 size-4 shrink-0" />
                  <span>{error}</span>
                </li>
              )}
            </For>
          </ul>
        </div>
      </Show>

      {/* 统计汇总 */}
      <div class="grid grid-cols-3 gap-4 rounded-md border p-4">
        <div class="text-center">
          <div class="text-2xl font-semibold text-success-foreground">{props.result.imported}</div>
          <div class="text-xs text-muted-foreground">{m.common_success()}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold text-muted-foreground">{props.result.skipped}</div>
          <div class="text-xs text-muted-foreground">{m.common_skipped()}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold text-error-foreground">{props.result.failed}</div>
          <div class="text-xs text-muted-foreground">{m.common_failed()}</div>
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-4">
        <Show when={props.result.failed > 0}>
          <Button variant="outline" onClick={props.onBack}>
            {m.import_backToRetry()}
          </Button>
        </Show>
        <Button onClick={props.onFinish}>{m.common_finish()}</Button>
      </div>
    </div>
  );
};
