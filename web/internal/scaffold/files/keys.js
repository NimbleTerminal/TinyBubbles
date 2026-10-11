export function allowedKeys(snapshot) {
  return new Set((snapshot?.actions || []).filter(a => a.enabled).flatMap(a =>
    a.keys.filter(g => g.modifiers.length === 0).map(g => g.key)));
}

export function encodeGesture(gesture, applicationCursor = false, allowed = new Set()) {
  if (gesture.modifiers?.length || !allowed.has(gesture.key)) return null;
  const key = gesture.key;
  const cursor = letter => `\x1b${applicationCursor ? 'O' : '['}${letter}`;
  const cursorKeys = {up:'A', down:'B', right:'C', left:'D', home:'H', end:'F'};
  if (cursorKeys[key]) return cursor(cursorKeys[key]);
  const simple = {enter:'\r', escape:'\x1b', space:' ', tab:'\t', backspace:'\x7f'};
  if (Object.hasOwn(simple, key)) return simple[key];
  const tilde = {insert:2, delete:3, pageup:5, pagedown:6, f5:15, f6:17, f7:18, f8:19, f9:20, f10:21, f11:23, f12:24};
  if (tilde[key]) return `\x1b[${tilde[key]}~`;
  if (/^f[1-4]$/.test(key)) return `\x1bO${'PQRS'[Number(key[1])-1]}`;
  return /^\P{C}$/u.test(key) ? key : null;
}

export function keyFromEvent(event) {
  if (event.ctrlKey || event.altKey || event.metaKey || event.isComposing) return null;
  // Shift used to type a character is already represented by event.key.
  if (/^\P{C}$/u.test(event.key) && event.key !== ' ') return event.key;
  if (event.shiftKey) return null;
  const named = {ArrowUp:'up', ArrowDown:'down', ArrowLeft:'left', ArrowRight:'right',
    Enter:'enter', Escape:'escape', ' ':'space', Tab:'tab', Backspace:'backspace',
    Home:'home', End:'end', Insert:'insert', Delete:'delete', PageUp:'pageup', PageDown:'pagedown'};
  if (Object.hasOwn(named,event.key)) return named[event.key];
  return /^F([1-9]|1[0-2])$/.test(event.key) ? event.key.toLowerCase() : null;
}
