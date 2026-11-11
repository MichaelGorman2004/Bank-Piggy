import js from '@eslint/js';
import globals from 'globals';

export default [
  { ignores: ['**/dist/**', '**/node_modules/**', 'packages/contracts/index.ts'] },
  js.configs.recommended,
  {
    files: ['**/*.mjs'],
    languageOptions: { globals: globals.node },
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
];
