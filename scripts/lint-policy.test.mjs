import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const ROOT = fileURLToPath(new URL('..', import.meta.url));
const FUNCTION_LINE_LIMIT = 50;
const LINT_TIMEOUT_MS = 30_000;
const TOOLS = {
  oxlint: join(ROOT, 'node_modules/oxlint/bin/oxlint'),
  eslint: join(ROOT, 'node_modules/eslint/bin/eslint.js'),
};

/** Runs the installed CLI with the repository configuration and no automatic fixes. */
function lint(tool, paths) {
  const flags = tool === 'oxlint' ? ['--type-aware'] : ['--max-warnings=0'];
  const result = spawnSync(
    process.execPath,
    [TOOLS[tool], ...flags, '--format', 'json', ...paths],
    {
      cwd: ROOT,
      encoding: 'utf8',
      timeout: LINT_TIMEOUT_MS,
    },
  );
  assert.ifError(result.error);
  assert.equal(result.signal, null, result.stderr);
  const output = JSON.parse(result.stdout);
  const diagnostics =
    tool === 'oxlint' ? output.diagnostics : output.flatMap((file) => file.messages);
  return {
    status: result.status,
    diagnostics,
    rules: diagnostics.map((diagnostic) => diagnostic.ruleId ?? diagnostic.code),
    output: result.stdout + result.stderr,
  };
}

/** Creates isolated project files and registers cleanup before any file writes. */
async function fixtures(context, sources) {
  const directory = await mkdtemp(join(ROOT, 'apps/web/src/lint-policy-'));
  context.after(() => rm(directory, { recursive: true, force: true }));
  const paths = {};
  for (const [name, source] of Object.entries(sources)) {
    paths[name] = join(directory, name);
    await writeFile(paths[name], source);
  }
  return paths;
}

/** Requires a lint rejection from each specified rule, rather than a process or parse failure. */
function rejectsRules(result, expected) {
  assert.equal(result.status, 1, result.output);
  for (const rule of expected) assert.ok(result.rules.includes(rule), `${rule}\n${result.output}`);
}

/** Builds a function with an exact number of nonblank source lines. */
function countedFunction(lineCount) {
  return [
    'export function counted() {',
    '  let value = 0;',
    ...Array.from({ length: lineCount - 4 }, () => '  value += 1;'),
    '  return value;',
    '}',
  ].join('\n');
}

/** Gives each branch one independent decision in the function. */
function branchFunction(branchCount) {
  return [
    'export function branches(value) {',
    ...Array.from(
      { length: branchCount },
      (_, index) => `  if (value === ${index}) return ${index};`,
    ),
    '  return -1;',
    '}',
  ].join('\n');
}

test('native TypeScript lint accepts safe types and rejects semantic failures', async (context) => {
  const paths = await fixtures(context, {
    'safe.ts': "import type { ReactNode } from 'react';\nexport const value: ReactNode = 'ready';",
    'unsafe.ts': [
      "import { ReactNode } from 'react';",
      'export const node: ReactNode = null;',
      "export const unvalidated: any = JSON.parse('null');",
      'export const unsafe: string = unvalidated.value;',
      'Promise.resolve(unsafe);',
      'export const unnecessary = true ? 1 : 0;',
    ].join('\n'),
  });
  const safe = lint('oxlint', [paths['safe.ts']]);
  assert.equal(safe.status, 0, safe.output);
  assert.equal(safe.diagnostics.length, 0, safe.output);
  rejectsRules(lint('oxlint', [paths['unsafe.ts']]), [
    'typescript(no-explicit-any)',
    'typescript(no-unsafe-assignment)',
    'typescript(no-unsafe-member-access)',
    'typescript(no-floating-promises)',
    'typescript(consistent-type-imports)',
    'typescript(no-unnecessary-condition)',
  ]);
});

test('ESLint rejects every forbidden TypeScript assertion form', async (context) => {
  const paths = await fixtures(context, {
    'assertions.ts': [
      "export const cast = 'ready' as string;",
      "export const legacy = <string>'ready';",
      'export const nonnull = document.body!;',
    ].join('\n'),
  });
  const result = lint('eslint', Object.values(paths));
  rejectsRules(result, ['no-restricted-syntax']);
  assert.equal(
    result.rules.filter((rule) => rule === 'no-restricted-syntax').length,
    3,
    result.output,
  );
});

test('ESLint retains the six JSDoc checks', async (context) => {
  const paths = await fixtures(context, {
    'documentation.ts': [
      'export function missingComment(value: number) { return value; }',
      '/**\n * @param value - Number.\n * @returns Number.\n */',
      'export function missingDescription(value: number) { return value; }',
      '/** Uses a number.\n * @param wrong - Number.\n * @returns Number.\n */',
      'export function wrongName(value: number) { return value; }',
      '/** Uses a number.\n * @unknownPolicyTag Invalid.\n * @returns Number.\n */',
      'export function missingParam(value: number) { return value; }',
      '/** Uses a number.\n * @param value - Number.\n */',
      'export function missingReturn(value: number) { return value; }',
    ].join('\n'),
  });
  rejectsRules(lint('eslint', Object.values(paths)), [
    'jsdoc/require-jsdoc',
    'jsdoc/require-description',
    'jsdoc/check-param-names',
    'jsdoc/check-tag-names',
    'jsdoc/require-param',
    'jsdoc/require-returns',
  ]);
});

test('ESLint rejects conditional hooks and treats dependency warnings as failures', async (context) => {
  const description =
    '/** Shows a fixture.\n * @param value - Input.\n * @returns A placeholder.\n */';
  const paths = await fixtures(context, {
    'conditional.tsx': [
      "import { useState } from 'react';",
      description,
      'export function ConditionalHook(value: boolean) {',
      '  if (value) useState(0);',
      '  return null;',
      '}',
    ].join('\n'),
    'dependencies.tsx': [
      "import { useEffect } from 'react';",
      description,
      'export function WarningHook(value: number) {',
      '  useEffect(() => { console.log(value); }, []);',
      '  return null;',
      '}',
    ].join('\n'),
  });
  rejectsRules(lint('eslint', [paths['conditional.tsx']]), ['react-hooks/rules-of-hooks']);
  const warnings = lint('eslint', [paths['dependencies.tsx']]);
  rejectsRules(warnings, ['react-hooks/exhaustive-deps']);
  assert.ok(
    warnings.diagnostics.every(({ severity }) => severity === 1),
    warnings.output,
  );
});

const STRUCTURAL_LIMITS = [
  [
    'max-lines-per-function',
    countedFunction(FUNCTION_LINE_LIMIT),
    countedFunction(FUNCTION_LINE_LIMIT + 1),
  ],
  ['complexity', branchFunction(9), branchFunction(10)],
];

for (const [rule, valid, invalid] of STRUCTURAL_LIMITS) {
  test(`ESLint preserves the ${rule} boundary`, async (context) => {
    const paths = await fixtures(context, { 'valid.mjs': valid, 'invalid.mjs': invalid });
    const accepted = lint('eslint', [paths['valid.mjs']]);
    assert.equal(accepted.status, 0, accepted.output);
    assert.equal(accepted.diagnostics.length, 0, accepted.output);
    rejectsRules(lint('eslint', [paths['invalid.mjs']]), [rule]);
  });
}

test('the installed dependency graph uses only the TypeScript 7 compiler', () => {
  const result = spawnSync(
    'npm',
    ['ls', 'typescript', '@typescript/typescript6', '--all', '--json'],
    {
      cwd: ROOT,
      encoding: 'utf8',
      timeout: LINT_TIMEOUT_MS,
    },
  );
  assert.ifError(result.error);
  assert.equal(result.status, 0, result.stdout + result.stderr);
  const pending = [JSON.parse(result.stdout)];
  const compilers = [];
  while (pending.length) {
    for (const [name, dependency] of Object.entries(pending.pop().dependencies ?? {})) {
      assert.notEqual(name, '@typescript/typescript6');
      if (name === 'typescript') compilers.push(dependency.version);
      pending.push(dependency);
    }
  }
  assert.ok(compilers.length > 0);
  for (const version of compilers) assert.match(version, /^7\./);
});
