/**
 * FilterRulesEditor - 过滤器规则可视化编辑器
 *
 * 允许用户配置文件过滤规则，每条规则包含类型（Include/Exclude）和模式。
 * 规则按从上到下顺序匹配，第一个匹配的规则生效。
 */
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { TextField, TextFieldInput } from '@/components/ui/text-field';
import { cn } from '@/lib/utils';
import * as m from '@/paraglide/messages.js';
import { Component, Index, Show, createMemo } from 'solid-js';
import IconArrowDown from '~icons/lucide/arrow-down';
import IconArrowUp from '~icons/lucide/arrow-up';
import IconExternalLink from '~icons/lucide/external-link';
import IconPlus from '~icons/lucide/plus';
import IconTrash2 from '~icons/lucide/trash-2';

// 规则类型
type RuleType = 'include' | 'exclude';

// 解析后的规则
interface FilterRule {
  type: RuleType;
  pattern: string;
}

export interface FilterRulesEditorProps {
  /**
   * 过滤规则列表（rclone filter 语法，如 ["- node_modules/**", "+ **"]）
   */
  value: string[];
  /**
   * 规则变更回调
   */
  onChange: (rules: string[]) => void;
  /**
   * 是否禁用编辑
   */
  disabled?: boolean;
  /**
   * 自定义类名
   */
  class?: string;
}

/**
 * 解析 rclone filter 规则字符串为结构化数据
 */
function parseRule(rule: string): FilterRule {
  const trimmed = rule.trim();
  // 使用更宽松的匹配，支持 "+ " 或 "+" 开头
  if (trimmed.startsWith('+')) {
    const pattern = trimmed.startsWith('+ ') ? trimmed.slice(2) : trimmed.slice(1).trimStart();
    return { type: 'include', pattern };
  } else if (trimmed.startsWith('-')) {
    const pattern = trimmed.startsWith('- ') ? trimmed.slice(2) : trimmed.slice(1).trimStart();
    return { type: 'exclude', pattern };
  }
  // 默认为 exclude，pattern 为整个字符串
  return { type: 'exclude', pattern: trimmed };
}

/**
 * 将结构化规则转换为 rclone filter 语法字符串
 */
function formatRule(rule: FilterRule): string {
  const prefix = rule.type === 'include' ? '+ ' : '- ';
  return prefix + rule.pattern;
}

export const FilterRulesEditor: Component<FilterRulesEditorProps> = (props) => {
  // 解析规则列表
  const rules = createMemo(() => props.value.map(parseRule));

  // 更新规则
  const updateRules = (newRules: FilterRule[]) => {
    props.onChange(newRules.map(formatRule));
  };

  // 添加新规则
  const addRule = () => {
    const newRules = [...rules(), { type: 'exclude' as RuleType, pattern: '' }];
    updateRules(newRules);
  };

  // 删除规则
  const removeRule = (index: number) => {
    const newRules = rules().filter((_, i) => i !== index);
    updateRules(newRules);
  };

  // 更新规则类型
  const updateRuleType = (index: number, type: RuleType) => {
    const newRules = rules().map((rule, i) => (i === index ? { ...rule, type } : rule));
    updateRules(newRules);
  };

  // 更新规则模式
  const updateRulePattern = (index: number, pattern: string) => {
    const newRules = rules().map((rule, i) => (i === index ? { ...rule, pattern } : rule));
    updateRules(newRules);
  };

  // 上移规则
  const moveRuleUp = (index: number) => {
    if (index <= 0) return;
    const newRules = [...rules()];
    [newRules[index - 1], newRules[index]] = [newRules[index], newRules[index - 1]];
    updateRules(newRules);
  };

  // 下移规则
  const moveRuleDown = (index: number) => {
    if (index >= rules().length - 1) return;
    const newRules = [...rules()];
    [newRules[index], newRules[index + 1]] = [newRules[index + 1], newRules[index]];
    updateRules(newRules);
  };

  return (
    <div class={cn('space-y-3', props.class)}>
      {/* 标题栏 */}
      <div class="flex items-center justify-between">
        <h4 class="text-sm font-medium">{m.filter_rules()}</h4>
        <Button
          variant="outline"
          size="sm"
          onClick={addRule}
          disabled={props.disabled}
          class="h-7 gap-1 px-2 text-xs"
        >
          <IconPlus class="size-3" />
          {m.common_add()}
        </Button>
      </div>

      {/* 规则列表 */}
      <div class="space-y-2">
        <Show when={rules().length === 0}>
          <div class="rounded-lg border border-dashed p-4 text-center text-sm text-muted-foreground">
            {m.filter_noRules()}
          </div>
        </Show>

        <Index each={rules()}>
          {(rule, index) => (
            <div class="flex items-center gap-2 rounded-lg border bg-muted/30 p-2">
              {/* 类型选择 */}
              <Select
                value={rule().type}
                onChange={(value) => value && updateRuleType(index, value as RuleType)}
                options={['exclude', 'include'] as const}
                disabled={props.disabled}
                itemComponent={(itemProps) => (
                  <SelectItem item={itemProps.item}>
                    {itemProps.item.rawValue === 'exclude'
                      ? m.filter_typeExclude()
                      : m.filter_typeInclude()}
                  </SelectItem>
                )}
              >
                <SelectTrigger class="h-8 w-24 text-xs">
                  <SelectValue>
                    {(state) =>
                      state.selectedOption() === 'exclude'
                        ? m.filter_typeExclude()
                        : m.filter_typeInclude()
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent />
              </Select>

              {/* 模式输入 */}
              <TextField class="flex-1">
                <TextFieldInput
                  value={rule().pattern}
                  onInput={(e: InputEvent) =>
                    updateRulePattern(index, (e.currentTarget as HTMLInputElement).value)
                  }
                  placeholder={m.filter_patternPlaceholder()}
                  disabled={props.disabled}
                  class="h-8 font-mono text-xs"
                />
              </TextField>

              {/* 操作按钮 */}
              <div class="flex items-center gap-0.5">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => moveRuleUp(index)}
                  disabled={props.disabled ?? index === 0}
                  class="size-7"
                  title={m.filter_moveUp()}
                >
                  <IconArrowUp class="size-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => moveRuleDown(index)}
                  disabled={props.disabled ?? index === rules().length - 1}
                  class="size-7"
                  title={m.filter_moveDown()}
                >
                  <IconArrowDown class="size-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => removeRule(index)}
                  disabled={props.disabled}
                  class="size-7 text-destructive hover:bg-destructive/10 hover:text-destructive"
                  title={m.common_delete()}
                >
                  <IconTrash2 class="size-3.5" />
                </Button>
              </div>
            </div>
          )}
        </Index>
      </div>

      {/* 帮助信息 */}
      <div class="space-y-1 text-xs text-muted-foreground">
        <p>{m.filter_orderHelp()}</p>
        <a
          href="https://rclone.org/filtering/#filter-add-a-file-filtering-rule"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1 text-primary hover:underline"
        >
          <IconExternalLink class="size-3" />
          {m.filter_syntaxDoc()}
        </a>
      </div>
    </div>
  );
};
