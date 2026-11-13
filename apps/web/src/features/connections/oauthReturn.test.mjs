import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';

const html = await readFile(new URL('../../../index.html', import.meta.url), 'utf8');
const script = /<script>([\s\S]*?)<\/script>/.exec(html)?.[1];

test('institution OAuth return is scrubbed before assets and kept only in transient memory', () => {
  assert.ok(script);
  assert.ok(html.indexOf('<script>') < html.indexOf('<link'));
  const original =
    'https://bank-piggy.example/plaid-return?oauth_state_id=simulation-only&other=value';
  const calls = [];
  const window = {
    location: {
      pathname: '/plaid-return',
      search: '?oauth_state_id=simulation-only&other=value',
      href: original,
    },
    history: {
      replaceState: (...args) => {
        calls.push(args);
      },
    },
  };
  runInNewContext(script, { window, URLSearchParams });
  assert.deepEqual(calls, [[null, '', '/']]);
  assert.equal(window.consumeBankPiggyPlaidReturn(), original);
  assert.equal(window.consumeBankPiggyPlaidReturn(), null);
  assert.equal(Object.hasOwn(window, 'localStorage'), false);
  assert.equal(Object.hasOwn(window, 'sessionStorage'), false);
});

test('a callback without OAuth state does not start provider recovery', () => {
  const window = {
    location: { pathname: '/plaid-return', search: '' },
    history: {
      replaceState() {
        assert.fail('unexpected rewrite');
      },
    },
  };
  runInNewContext(script, { window, URLSearchParams });
  assert.equal(window.consumeBankPiggyPlaidReturn, undefined);
});
