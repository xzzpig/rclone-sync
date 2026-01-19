---
name: sort-i18n-keys
description: 对项目的 i18n JSON 文件 (zh-CN.json, en.json) 进行 key 字典序排序。当用户需要整理、排序国际化翻译文件，或者在添加新翻译后需要保持 key 顺序一致时使用此 skill。
---

# Sort i18n Keys

对项目中的国际化 (i18n) JSON 文件进行 key 字典序排序，确保翻译文件的 key 保持一致的顺序，便于 diff 对比和代码审查。

## 使用场景

- 添加新翻译 key 后，需要整理文件顺序
- 合并翻译文件后，需要统一 key 排序
- 代码审查前，确保翻译文件格式一致

## 使用方法

运行以下命令对 i18n 文件进行排序：

使用 skill 内置脚本：

```bash
node .claude/skills/sort-i18n-keys/scripts/sort-i18n-keys.js
```

## 处理的文件

脚本会自动处理以下文件：

- `web/project.inlang/messages/zh-CN.json`
- `web/project.inlang/messages/en.json`

## 脚本行为

1. 读取 JSON 文件
2. 对所有顶层 key 进行字典序 (alphabetical) 排序
3. 使用 2 空格缩进写回文件
4. 输出处理结果和 key 数量

## Resources

### scripts/

- `sort-i18n-keys.js` - 排序脚本，可直接执行
