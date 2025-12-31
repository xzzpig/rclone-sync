import js from '@eslint/js';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';
import solid from 'eslint-plugin-solid';
import prettier from 'eslint-plugin-prettier';
import prettierConfig from 'eslint-config-prettier';
import tailwindcss from 'eslint-plugin-tailwindcss';

export default [
  // 忽略文件
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      '*.config.js',
      '*.config.ts',
      'vite.config.d.ts',
      'tsconfig.tsbuildinfo',
      'tsconfig.node.tsbuildinfo',
      'src/components/ui',
      'src/paraglide/**',
    ],
  },
  // JavaScript 基础配置
  js.configs.recommended,
  // TypeScript 配置
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        project: './tsconfig.json',
      },
      globals: {
        // 浏览器全局变量
        console: 'readonly',
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        setTimeout: 'readonly',
        setInterval: 'readonly',
        clearTimeout: 'readonly',
        clearInterval: 'readonly',
        fetch: 'readonly',
        EventSource: 'readonly',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
      solid,
      prettier,
      tailwindcss,
    },
    rules: {
      // TypeScript 推荐规则
      ...tseslint.configs.recommended.rules,
      // SolidJS 推荐规则
      ...solid.configs.recommended.rules,
      // Tailwind CSS 规则
      ...tailwindcss.configs.recommended.rules,
      // Prettier 规则
      'prettier/prettier': 'error',
      // 自定义规则
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
        },
      ],
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/explicit-module-boundary-types': 'off',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      'no-console': ['warn', { allow: ['info', 'warn', 'error'] }],
    },
  },
  // Prettier 配置（禁用与 Prettier 冲突的规则）
  prettierConfig,
];
