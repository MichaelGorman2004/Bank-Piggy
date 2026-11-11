import js from '@eslint/js';
import babelParser from '@babel/eslint-parser';
import { defineConfig } from 'eslint/config';
import jsdoc from 'eslint-plugin-jsdoc';
import reactHooks from 'eslint-plugin-react-hooks';
import globals from 'globals';

export default defineConfig(
  { ignores: ['**/dist/**', '**/node_modules/**', 'packages/contracts/index.ts'] },
  js.configs.recommended,
  {
    files: ['apps/web/**/*.{ts,tsx}'],
    languageOptions: {
      globals: globals.browser,
      parser: babelParser,
      parserOptions: {
        requireConfigFile: false,
        babelOptions: {
          babelrc: false,
          configFile: false,
          parserOpts: { plugins: ['typescript'] },
        },
      },
    },
    plugins: { 'react-hooks': reactHooks, jsdoc },
    rules: {
      ...reactHooks.configs.recommended.rules,
      // Preserve strictTypeChecked's core overrides; Oxlint owns the TypeScript rules.
      'constructor-super': 'off',
      'getter-return': 'off',
      'no-class-assign': 'off',
      'no-const-assign': 'off',
      'no-dupe-args': 'off',
      'no-dupe-class-members': 'off',
      'no-dupe-keys': 'off',
      'no-func-assign': 'off',
      'no-import-assign': 'off',
      'no-new-native-nonconstructor': 'off',
      'no-new-symbol': 'off',
      'no-obj-calls': 'off',
      'no-redeclare': 'off',
      'no-setter-return': 'off',
      'no-this-before-super': 'off',
      'no-undef': 'off',
      'no-unreachable': 'off',
      'no-unsafe-negation': 'off',
      'no-var': 'error',
      'no-with': 'off',
      'prefer-const': 'error',
      'prefer-rest-params': 'error',
      'prefer-spread': 'error',
      'no-array-constructor': 'off',
      'no-implied-eval': 'off',
      'no-unused-expressions': 'off',
      'no-unused-vars': 'off',
      'no-useless-constructor': 'off',
      'no-throw-literal': 'off',
      'prefer-promise-reject-errors': 'off',
      'require-await': 'off',
      'no-return-await': 'off',
      'no-restricted-syntax': [
        'error',
        { selector: 'TSAsExpression', message: 'Validate or narrow the type instead of casting.' },
        { selector: 'TSTypeAssertion', message: 'Validate or narrow the type instead of casting.' },
        { selector: 'TSNonNullExpression', message: 'Handle the missing value explicitly.' },
      ],
      'jsdoc/require-jsdoc': [
        'error',
        {
          exemptEmptyConstructors: true,
          contexts: [
            'FunctionDeclaration',
            'MethodDefinition',
            'VariableDeclarator > ArrowFunctionExpression',
            'VariableDeclarator > FunctionExpression',
          ],
        },
      ],
      'jsdoc/require-description': 'error',
      'jsdoc/check-param-names': 'error',
      'jsdoc/check-tag-names': 'error',
      'jsdoc/require-param': 'error',
      'jsdoc/require-returns': 'error',
    },
  },
  {
    files: ['apps/web/**/*.tsx'],
    languageOptions: {
      parserOptions: { babelOptions: { parserOpts: { plugins: ['typescript', 'jsx'] } } },
    },
  },
  {
    files: ['**/*.{ts,tsx,mjs}'],
    rules: {
      'max-lines-per-function': ['error', { max: 50, skipBlankLines: true, skipComments: true }],
      'max-lines': ['error', { max: 300, skipBlankLines: true, skipComments: true }],
      complexity: ['error', 10],
      'max-depth': ['error', 3],
      'max-nested-callbacks': ['error', 3],
      'max-params': ['error', 4],
      eqeqeq: ['error', 'always'],
      'no-else-return': 'error',
      'no-duplicate-imports': 'error',
    },
  },
  { files: ['**/*.mjs', 'apps/web/vite.config.ts'], languageOptions: { globals: globals.node } },
);
