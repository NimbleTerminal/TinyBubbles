// Add working demos here; each entry can load its own WASM program.
// Actions remain owned by the Go model's help map.
export const demos = [
  {
    id: 'hello',
    runtimes: ['bubbletea', 'tinytea'],
    title: 'Hello, tiny world',
    category: 'Getting started',
    description: 'Animation, custom fonts, and a counter on a tiny display.',
    instructions: 'Pop a bubble, try a new palette, or pause the scene.',
    wasm: 'app.wasm',
    source: 'https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/hello',
  },
];
