import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { test } from 'node:test';
import { runInNewContext } from 'node:vm';

const html = await readFile(new URL('../../../index.html', import.meta.url), 'utf8');
const script = /<script>([\s\S]*?)<\/script>/.exec(html)?.[1];

/** Runs the real early document script without allowing storage or a network request. */
function capture(pathname, hash) {
  const calls = [];
  const window = {
    location: { pathname, hash, search: '' },
    history: {
      replaceState: (...args) => {
        calls.push(args);
      },
    },
  };
  runInNewContext(script, { window, URLSearchParams });
  return { window, calls };
}

test('join fragment is scrubbed before font or module assets and consumed only once', () => {
  assert.ok(script);
  assert.ok(html.indexOf('<script>') < html.indexOf('<link'));
  const token = 'a'.repeat(43);
  const result = capture('/join', `#invite=${token}`);
  assert.deepEqual(result.calls, [[null, '', '/join']]);
  assert.equal(result.window.consumeBankPiggyInvitation(), token);
  assert.equal(result.window.consumeBankPiggyInvitation(), null);
});

test('malformed join tokens are removed and unrelated routes are unchanged', () => {
  const invalid = capture('/join', '#invite=bad-token');
  assert.equal(invalid.calls.length, 1);
  assert.equal(invalid.window.consumeBankPiggyInvitation(), null);
  const ordinary = capture('/', '#other');
  assert.equal(ordinary.calls.length, 0);
  assert.equal(ordinary.window.consumeBankPiggyInvitation, undefined);
});
