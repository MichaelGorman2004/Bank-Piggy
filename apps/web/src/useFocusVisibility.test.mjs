import assert from 'node:assert/strict';
import { getEventListeners } from 'node:events';
import { test } from 'node:test';
import { startFocusVisibility } from './useFocusVisibility.ts';

/** Supplies document events and attributes without loading a browser or household data. */
function focusPage() {
  const attributes = new Map();
  const documentElement = {
    setAttribute: (name, value) => attributes.set(name, value),
    removeAttribute: (name) => attributes.delete(name),
    hasAttribute: (name) => attributes.has(name),
  };
  return Object.assign(new EventTarget(), { documentElement });
}

/** Supplies the keyboard fields that Node's generic Event does not contain. */
function keydown(key, modifiers = {}) {
  return Object.assign(new Event('keydown', { cancelable: true }), {
    key,
    altKey: false,
    ctrlKey: false,
    metaKey: false,
    ...modifiers,
  });
}

test('pointer and keyboard switching preserves native default actions', (context) => {
  const page = focusPage();
  context.after(startFocusVisibility(page));
  assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), false);
  for (const key of ['Tab', 'ArrowDown', 'Enter', ' ', 'Escape']) {
    const pointer = new Event('pointerdown', { cancelable: true });
    page.dispatchEvent(pointer);
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), true);
    const keyboard = keydown(key);
    page.dispatchEvent(keyboard);
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), false);
    assert.equal(pointer.defaultPrevented, false);
    assert.equal(keyboard.defaultPrevented, false);
  }
  for (const modifiers of [
    { shiftKey: true },
    { altKey: true },
    { altKey: true, shiftKey: true },
  ]) {
    page.dispatchEvent(new Event('pointerdown'));
    page.dispatchEvent(keydown('Tab', modifiers));
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), false);
  }
});

test('modifier-only keys keep pointer mode while key combinations retain native actions', (context) => {
  const page = focusPage();
  context.after(startFocusVisibility(page));
  page.dispatchEvent(new Event('pointerdown'));
  for (const key of ['Alt', 'Control', 'Meta', 'Shift']) {
    page.dispatchEvent(keydown(key));
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), true);
  }
  for (const event of [
    keydown('r', { metaKey: true }),
    keydown('l', { ctrlKey: true }),
    keydown('ArrowLeft', { altKey: true }),
  ]) {
    page.dispatchEvent(new Event('pointerdown'));
    page.dispatchEvent(event);
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), false);
    assert.equal(event.defaultPrevented, false);
  }
});

test('cleanup and remount restore the native default without retaining listeners', () => {
  const page = focusPage();
  for (let mount = 0; mount < 2; mount += 1) {
    const stop = startFocusVisibility(page);
    assert.equal(getEventListeners(page, 'pointerdown').length, 1);
    assert.equal(getEventListeners(page, 'keydown').length, 1);
    page.dispatchEvent(new Event('pointerdown'));
    stop();
    assert.equal(getEventListeners(page, 'pointerdown').length, 0);
    assert.equal(getEventListeners(page, 'keydown').length, 0);
    page.dispatchEvent(new Event('pointerdown'));
    assert.equal(page.documentElement.hasAttribute('data-pointer-focus'), false);
  }
});
