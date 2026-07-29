// ESLint flat configuration with layer-boundary enforcement.
//
// Layer model:
//   L0  src/gen                         generated proto (immutable)
//   L1  src/api                         raw network I/O
//   L2  src/contexts/<ctx>/domain       pure domain
//   L3  src/contexts/<ctx>/adapters     pure converters L1<->L2
//   L4  src/contexts/<ctx>/application  React Query + stateful clients
//   L5  src/components, router, main    UI
//
// shared/ + utils/ are cross-cutting dependencies.
// stores/ are persisted / UI-state containers.

import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import boundaries from 'eslint-plugin-boundaries'
import importX from 'eslint-plugin-import-x'
import { createTypeScriptImportResolver } from 'eslint-import-resolver-typescript'
import tanstackQuery from '@tanstack/eslint-plugin-query'
import jsxA11y from 'eslint-plugin-jsx-a11y'

// Semantic-token guard: raw Tailwind palette classes with an exact
// semantic-token equivalent (web/tailwind.config.js) are banned so new
// code can't regress the finished token migration.
const BANNED_PALETTE_CLASSES = [
  ['bg-white', 'bg-surface-primary'],
  ['bg-gray-100', 'bg-surface-tertiary'],
  ['bg-gray-900', 'bg-surface-inverse'],
  ['bg-blue-600', 'bg-accent'],
  ['bg-blue-700', 'bg-accent-hover'],
  ['bg-blue-100', 'bg-accent-muted'],
  ['bg-red-100', 'bg-status-error-light'],
  ['bg-amber-50', 'bg-status-warning-bg'],
  ['bg-amber-100', 'bg-status-warning-light'],
  ['text-gray-900', 'text-content-primary'],
  ['text-gray-500', 'text-content-tertiary'],
  ['text-white', 'text-content-inverse'],
  ['text-blue-600', 'text-accent'],
  ['text-red-600', 'text-status-error-text'],
  ['text-amber-600', 'text-status-warning-text'],
  ['text-emerald-600', 'text-status-success-text'],
  ['border-gray-200', 'border-border'],
  ['border-gray-300', 'border-border-strong'],
  ['border-blue-500', 'border-border-focus'],
  ['border-red-500', 'border-status-error-border'],
  ['border-amber-500', 'border-status-warning-border'],
  ['border-emerald-500', 'border-status-success-border'],
  ['divide-gray-200', 'divide-border'],
  ['divide-gray-300', 'divide-border-strong'],
  ['divide-blue-500', 'divide-border-focus'],
  ['divide-blue-600', 'divide-accent'],
  ['divide-red-500', 'divide-status-error-border'],
  ['divide-amber-500', 'divide-status-warning-border'],
  ['divide-emerald-500', 'divide-status-success-border'],
  ['ring-gray-200', 'ring-border'],
  ['ring-gray-300', 'ring-border-strong'],
  ['ring-blue-500', 'ring-border-focus'],
  ['ring-blue-600', 'ring-accent'],
  ['ring-emerald-500', 'ring-status-success-border'],
  ['placeholder-gray-900', 'placeholder-content-primary'],
  ['placeholder-gray-500', 'placeholder-content-tertiary'],
  ['placeholder-white', 'placeholder-content-inverse'],
  ['bg-gray-200', 'bg-surface-hover'],
  ['bg-gray-50', 'bg-surface-secondary'],
  ['bg-blue-50', 'bg-accent-light'],
  ['bg-red-50', 'bg-status-error-bg'],
  ['text-gray-600', 'text-content-secondary'],
  ['text-blue-700', 'text-accent-text'],
  ['bg-emerald-50', 'bg-status-success-bg'],
  ['bg-emerald-100', 'bg-status-success-light'],
  ['text-gray-400', 'text-content-muted'],
  ['border-blue-600', 'border-accent'],
  ['ring-red-500', 'ring-status-error-border'],
  ['ring-amber-500', 'ring-status-warning-border'],
  ['placeholder-gray-400', 'placeholder-content-muted'],
  ['placeholder-gray-600', 'placeholder-content-secondary'],
]

const paletteClassRestrictions = BANNED_PALETTE_CLASSES.flatMap(([raw, replacement]) => {
  const pattern = `(^|[^\\w-])${raw}([^\\w-]|$)`
  const message = `Use ${replacement} (semantic token) instead of ${raw}`
  return [
    { selector: `Literal[value=/${pattern}/]`, message },
    { selector: `TemplateElement[value.raw=/${pattern}/]`, message },
  ]
})

export default tseslint.config(
  // Ignores ------------------------------------------------------------
  {
    ignores: [
      'dist',
      '.vite',
      'coverage',
      'test-results',
      'playwright-report',
      'blob-report',
      'src/gen/**',
      'node_modules',
    ],
  },

  // Base rules -----------------------------------------------------------
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    extends: [tseslint.configs.recommended],
    languageOptions: {
      parser: tseslint.parser,
      ecmaVersion: 2021,
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.es2021,
      },
      parserOptions: {
        ecmaFeatures: { jsx: true },
      },
    },
  },

  // react-hooks / react-refresh -------------------------------------------
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    rules: {
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },

  // @tanstack/query --------------------------------------------------------
  {
    ...tanstackQuery.configs['flat/recommended'][0],
    files: ['**/*.{ts,tsx}'],
  },

  // jsx-a11y ---------------------------------------------------------------
  {
    ...jsxA11y.flatConfigs.recommended,
    files: ['**/*.{ts,tsx}'],
  },

  // import-x: TypeScript resolver, no-cycle only (parity-first) -----------
  {
    files: ['**/*.{ts,tsx}'],
    plugins: {
      'import-x': importX,
    },
    settings: {
      'import-x/resolver-next': [createTypeScriptImportResolver({ project: './tsconfig.json' })],
    },
    rules: {
      'import-x/no-cycle': ['error', { maxDepth: 4 }],
    },
  },

  // Layer boundaries --------------------------------------------------------
  {
    files: ['**/*.{ts,tsx}'],
    plugins: { boundaries },
    settings: {
      // eslint-plugin-boundaries still resolves modules via
      // eslint-module-utils, which reads the legacy `import/resolver` key
      // (separate from import-x's `import-x/resolver-next`).
      'import/resolver': {
        typescript: { project: './tsconfig.json' },
      },
      'boundaries/elements': [
        { type: 'gen', pattern: 'src/gen/**' },
        { type: 'api', pattern: 'src/api/**' },
        { type: 'ctx-domain', pattern: 'src/contexts/*/domain/**' },
        { type: 'ctx-adapter', pattern: 'src/contexts/*/adapters/**' },
        { type: 'ctx-app', pattern: 'src/contexts/*/application/**' },
        { type: 'ctx-barrel', pattern: 'src/contexts/*/index.ts', mode: 'file' },
        { type: 'shared', pattern: 'src/shared/**' },
        { type: 'stores', pattern: 'src/stores/**' },
        { type: 'utils', pattern: 'src/utils/**' },
        { type: 'hooks', pattern: 'src/hooks/**' },
        { type: 'types', pattern: 'src/types/**' },
        { type: 'test', pattern: 'src/test/**' },
        // router.tsx / main.tsx are FILES: a `src/{components,router.tsx}/**`
        // glob never matches them, which left both unclassified (= unchecked).
        { type: 'ui', pattern: 'src/components/**' },
        { type: 'ui', pattern: 'src/router.tsx', mode: 'file' },
        { type: 'ui', pattern: 'src/main.tsx', mode: 'file' },
      ],
      'boundaries/ignore': ['**/*.test.ts', '**/*.test.tsx'],
    },
    rules: {
      // Also enabled (bare) by the old `plugin:boundaries/recommended` (v4.2.2)
      // extends: entry-point, external, no-private. Ported for parity.
      'boundaries/entry-point': ['error'],
      'boundaries/external': ['error'],
      'boundaries/no-private': ['error', { allowUncles: true }],
      'boundaries/element-types': [
        'error',
        {
          default: 'disallow',
          rules: [
            { from: 'api', allow: ['api', 'gen', 'utils', 'shared', 'types'] },
            // ctx-domain occasionally re-uses api-layer TYPES (e.g. `SavedConnection`);
            // runtime code is still forbidden because domain files contain no IO.
            { from: 'ctx-domain', allow: ['ctx-domain', 'shared', 'utils', 'types', 'api'] },
            {
              from: 'ctx-adapter',
              allow: ['ctx-domain', 'ctx-adapter', 'gen', 'api', 'utils', 'shared', 'types'],
            },
            { from: 'hooks', allow: ['hooks', 'api', 'utils', 'shared', 'types', 'stores'] },
            {
              from: 'ctx-app',
              allow: [
                'ctx-domain',
                'ctx-adapter',
                'ctx-app',
                'ctx-barrel',
                'api',
                'gen',
                'stores',
                'hooks',
                'utils',
                'shared',
                'types',
              ],
            },
            { from: 'ctx-barrel', allow: ['ctx-domain', 'ctx-adapter', 'ctx-app'] },
            {
              from: 'ui',
              allow: [
                'ui',
                'ctx-barrel',
                'ctx-domain',
                'ctx-adapter',
                'ctx-app',
                'hooks',
                'utils',
                'stores',
                'shared',
                'types',
                'api',
              ],
            },
            { from: 'stores', allow: ['ctx-domain', 'ctx-barrel', 'utils', 'shared', 'stores', 'types'] },
            { from: 'shared', allow: ['shared', 'utils'] },
            { from: 'utils', allow: ['utils'] },
            { from: 'test', allow: ['test', 'ctx-domain', 'ctx-adapter', 'ctx-app', 'ctx-barrel', 'shared', 'utils', 'hooks', 'stores', 'api', 'gen', 'types', 'ui'] },
          ],
        },
      ],
    },
  },

  // Project-specific rules --------------------------------------------------
  {
    files: ['**/*.{ts,tsx}'],
    rules: {
      // Belt & suspenders — UI may not import from `@/gen/**`.
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            {
              // `@/gen/*` only matches one segment — real imports are deep
              // (`@/gen/types/nats/...`), so the guard needs `**`.
              group: ['@/gen/*', '@/gen/**'],
              message:
                'UI and contexts (outside adapters) must not import generated proto directly. Go through adapters -> domain.',
            },
          ],
        },
      ],

      'import-x/order': [
        'warn',
        {
          groups: ['builtin', 'external', 'internal', 'parent', 'sibling', 'index'],
          pathGroups: [
            { pattern: '@/gen/**', group: 'internal', position: 'before' },
            { pattern: '@/api/**', group: 'internal' },
            { pattern: '@/contexts/**', group: 'internal' },
            { pattern: '@/shared/**', group: 'internal' },
            { pattern: '@/stores/**', group: 'internal' },
            { pattern: '@/components/**', group: 'internal' },
            { pattern: '@/utils/**', group: 'internal' },
          ],
          'newlines-between': 'ignore',
        },
      ],

      // Allow unused names prefixed with _, common in injected params.
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],

      // Semantic-token guard --------------------------------------------------
      'no-restricted-syntax': ['error', ...paletteClassRestrictions],
    },
  },

  // Overrides ----------------------------------------------------------------
  {
    files: ['**/*.test.ts', '**/*.test.tsx', 'src/test/**'],
    rules: {
      'boundaries/element-types': 'off',
      'no-restricted-imports': 'off',
    },
  },
  {
    // Adapters are pure L1<->L2 converters: `@/gen/**` is their job, but
    // React Query hooks are NOT — those belong in the application layer.
    files: ['src/contexts/*/adapters/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@tanstack/react-query',
              message: 'Adapters are pure converters. React Query hooks belong in contexts/<ctx>/application.',
            },
          ],
        },
      ],
    },
  },
  {
    // Application-layer streaming clients (e.g. LiveStreamClient) consume
    // gen event union types directly — there is no meaningful conversion.
    files: ['src/contexts/*/application/**', 'src/api/**'],
    rules: {
      'no-restricted-imports': 'off',
    },
  },

  // Parity: migration-only carve-outs, see migration report ------------------
  {
    // Playwright fixture `use` callback, not a React hook.
    files: ['e2e/**'],
    rules: {
      'react-hooks/rules-of-hooks': 'off',
    },
  },
  {
    // Route table mixes lazy-loaded page refs with a non-component export;
    // react-refresh's newer local-component check isn't applicable here.
    files: ['src/router.tsx'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
  {
    // Generic query wrapper: `opts` intentionally excluded from queryKey.
    files: ['src/hooks/useConnectionQuery.ts'],
    rules: {
      '@tanstack/query/exhaustive-deps': 'off',
    },
  },
  {
    // Pre-existing type-only re-export of @/types/management, self-documented
    // as temporary debt. v7 now tracks type-only re-exports; v4.2.2 didn't.
    files: ['src/contexts/kv/index.ts', 'src/contexts/objects/index.ts'],
    rules: {
      'boundaries/element-types': 'off',
    },
  },
  {
    // jsx-a11y is new in this pass; these rules fire beyond the trivial-fix
    // budget on existing markup (counts in the migration report).
    files: ['**/*.{ts,tsx}'],
    rules: {
      'jsx-a11y/click-events-have-key-events': 'off',
      'jsx-a11y/no-static-element-interactions': 'off',
      'jsx-a11y/no-noninteractive-element-interactions': 'off',
      'jsx-a11y/label-has-associated-control': 'off',
      'jsx-a11y/no-autofocus': 'off',
      'jsx-a11y/interactive-supports-focus': 'off',
      'jsx-a11y/no-noninteractive-tabindex': 'off',
      'jsx-a11y/no-redundant-roles': 'off',
    },
  },

  {
    files: ['src/components/ui/Tabs.tsx'],
    rules: {
      'no-useless-assignment': 'off',
    },
  },

  {
    linterOptions: {
      reportUnusedDisableDirectives: 'error',
    },
  },
)
