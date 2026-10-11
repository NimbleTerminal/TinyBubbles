// This app never requests extended keyboard/release reporting. Decline every
// gesture outside the Go help map's bounded subset, including modifiers.
export function encodeGesture(gesture, applicationCursor = false) {
  if (gesture.modifiers.length) return null;
  switch (gesture.key) {
    case 'enter': return '\r';
    case 'escape': return '\x1b';
    case 'right': return `\x1b${applicationCursor ? 'O' : '['}C`;
    case 'q': return 'q';
    default: return null;
  }
}
