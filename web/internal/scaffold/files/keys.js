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
