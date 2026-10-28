export function allowedKeys(snapshot) {
  return new Set((snapshot?.actions || []).filter(a => a.enabled).flatMap(a =>
    a.keys.filter(g => g.modifiers.length === 0).map(g => g.key)));
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

// textFromEvent maps a keydown to raw input for TextInput demos: printable
// characters, including space, and Ctrl with a letter or [ \\ ] ^ _ @ space.
// Named keys return null; encode them as gestures. Alt and Meta are declined.
export function textFromEvent(event) {
  if (event.altKey || event.metaKey || event.isComposing) return null;
  if (event.ctrlKey) {
    const k = event.key.length === 1 ? event.key.toLowerCase() : '';
    if (k >= 'a' && k <= 'z') return String.fromCharCode(k.charCodeAt(0) - 96);
    const extra = {'[':'\x1b', '\\':'\x1c', ']':'\x1d', '^':'\x1e', '_':'\x1f', '@':'\x00', ' ':'\x00'};
    return Object.hasOwn(extra,k) ? extra[k] : null;
  }
  return /^\P{C}$/u.test(event.key) ? event.key : null;
}
