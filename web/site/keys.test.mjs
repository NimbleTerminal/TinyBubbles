import { test } from 'node:test';
import assert from 'node:assert/strict';
import { encodeGesture } from './keys.js';
const gesture = key => ({key, modifiers: []});
test('supported help gestures encode complete inputs', () => {
  assert.equal(encodeGesture(gesture('enter')), '\r');
  assert.equal(encodeGesture(gesture('escape')), '\x1b');
  assert.equal(encodeGesture(gesture('right')), '\x1b[C');
  assert.equal(encodeGesture(gesture('right'), true), '\x1bOC');
  assert.equal(encodeGesture(gesture('q')), 'q');
});
test('unsupported or modified gestures are declined', () => {
  assert.equal(encodeGesture(gesture('left')), null);
  assert.equal(encodeGesture({key:'q', modifiers:['ctrl']}), null);
});
