import assert from 'node:assert/strict';
import { getEventListeners } from 'node:events';
import { test } from 'node:test';
import { setImmediate } from 'node:timers/promises';
import { startAccountStatusUpdates } from './useAccountStatusUpdates.ts';

const STATUS_INTERVAL = 30 * 60_000;

/** Gives each test control of visibility, focus, and timer advancement. */
function surfaces(context, visibilityState = 'visible') {
  context.mock.timers.enable({ apis: ['setInterval'] });
  const page = Object.assign(new EventTarget(), { visibilityState });
  const browser = Object.assign(new EventTarget(), {
    setInterval: globalThis.setInterval,
    clearInterval: globalThis.clearInterval,
  });
  context.mock.method(browser, 'setInterval');
  context.mock.method(browser, 'clearInterval');
  return { page, browser };
}

/** Keeps a request unresolved until the test accepts its result. */
function deferred() {
  let resolve;
  const promise = new Promise((accept) => {
    resolve = accept;
  });
  return { promise, resolve };
}

test('reads on entry and every 30 minutes without an earlier timer read', async (context) => {
  const { page, browser } = surfaces(context);
  let reads = 0;
  const stop = startAccountStatusUpdates(
    async () => {
      reads += 1;
      return null;
    },
    page,
    browser,
  );
  context.after(stop);
  await setImmediate();
  assert.equal(reads, 1);
  context.mock.timers.tick(STATUS_INTERVAL - 1);
  await setImmediate();
  assert.equal(reads, 1);
  context.mock.timers.tick(1);
  await setImmediate();
  assert.equal(reads, 2);
  context.mock.timers.tick(STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 3);
});

test('focus, visibility, and timer triggers cannot overlap an unfinished read', async (context) => {
  const { page, browser } = surfaces(context);
  const first = deferred();
  let reads = 0;
  const stop = startAccountStatusUpdates(
    () => {
      reads += 1;
      return reads === 1 ? first.promise : Promise.resolve(null);
    },
    page,
    browser,
  );
  context.after(stop);
  await setImmediate();
  browser.dispatchEvent(new Event('focus'));
  page.dispatchEvent(new Event('visibilitychange'));
  context.mock.timers.tick(3 * STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 1);
  first.resolve(null);
  await setImmediate();
  assert.equal(reads, 1);
  browser.dispatchEvent(new Event('focus'));
  await setImmediate();
  assert.equal(reads, 2);
});

test('a hidden page waits for visibility and pauses again when hidden', async (context) => {
  const { page, browser } = surfaces(context, 'hidden');
  let reads = 0;
  const stop = startAccountStatusUpdates(
    async () => {
      reads += 1;
      return null;
    },
    page,
    browser,
  );
  context.after(stop);
  browser.dispatchEvent(new Event('focus'));
  context.mock.timers.tick(2 * STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 0);
  page.visibilityState = 'visible';
  page.dispatchEvent(new Event('visibilitychange'));
  await setImmediate();
  assert.equal(reads, 1);
  page.visibilityState = 'hidden';
  page.dispatchEvent(new Event('visibilitychange'));
  context.mock.timers.tick(STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 1);
  page.visibilityState = 'visible';
  page.dispatchEvent(new Event('visibilitychange'));
  await setImmediate();
  assert.equal(reads, 2);
});

test('a page hidden between queuing and dispatch does not start its queued GET', async (context) => {
  const { page, browser } = surfaces(context);
  let reads = 0;
  const stop = startAccountStatusUpdates(
    async () => {
      reads += 1;
      return null;
    },
    page,
    browser,
  );
  context.after(stop);
  page.visibilityState = 'hidden';
  await setImmediate();
  assert.equal(reads, 0);
  page.visibilityState = 'visible';
  page.dispatchEvent(new Event('visibilitychange'));
  await setImmediate();
  assert.equal(reads, 1);
});

test('a rejected read releases the lock and retains feedback owned by the callback', async (context) => {
  const { page, browser } = surfaces(context);
  let reads = 0;
  let feedback = null;
  const stop = startAccountStatusUpdates(
    async () => {
      reads += 1;
      if (reads === 1) {
        feedback = 'Account status could not load.';
        throw new Error(feedback);
      }
      return null;
    },
    page,
    browser,
  );
  context.after(stop);
  await setImmediate();
  assert.equal(feedback, 'Account status could not load.');
  context.mock.timers.tick(STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 2);
  assert.equal(feedback, 'Account status could not load.');
});

test('immediate cleanup removes resources and cancels the queued entry read', async (context) => {
  const { page, browser } = surfaces(context);
  let reads = 0;
  const stop = startAccountStatusUpdates(
    async () => {
      reads += 1;
      return null;
    },
    page,
    browser,
  );
  assert.equal(getEventListeners(page, 'visibilitychange').length, 1);
  assert.equal(getEventListeners(browser, 'focus').length, 1);
  stop();
  assert.equal(browser.clearInterval.mock.callCount(), 1);
  assert.equal(
    browser.clearInterval.mock.calls[0].arguments[0],
    browser.setInterval.mock.calls[0].result,
  );
  assert.equal(getEventListeners(page, 'visibilitychange').length, 0);
  assert.equal(getEventListeners(browser, 'focus').length, 0);
  context.mock.timers.tick(3 * STATUS_INTERVAL);
  browser.dispatchEvent(new Event('focus'));
  page.dispatchEvent(new Event('visibilitychange'));
  await setImmediate();
  assert.equal(reads, 0);
});

test('cleanup during a read cannot schedule more work after that read finishes', async (context) => {
  const { page, browser } = surfaces(context);
  const first = deferred();
  let reads = 0;
  const stop = startAccountStatusUpdates(
    () => {
      reads += 1;
      return first.promise;
    },
    page,
    browser,
  );
  await setImmediate();
  assert.equal(reads, 1);
  stop();
  first.resolve(null);
  await setImmediate();
  browser.dispatchEvent(new Event('focus'));
  page.dispatchEvent(new Event('visibilitychange'));
  context.mock.timers.tick(2 * STATUS_INTERVAL);
  await setImmediate();
  assert.equal(reads, 1);
});
