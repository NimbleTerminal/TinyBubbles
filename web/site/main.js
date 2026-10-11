import { BoobaTerminal, installActionBar } from './booba/booba.js';
import { encodeGesture } from './keys.js';
import { demos } from './demos.js';

const status = document.getElementById('status');
const canvas = document.getElementById('display');
const select = document.getElementById('screen');
const pageURL = new URL(location.href);
const demo = demos.find(entry => entry.id === pageURL.searchParams.get('demo')) || demos[0];
const runtimeSelect = document.getElementById('runtime');
const runtimeInfo = {
  bubbletea: {label:'Bubble Tea', note:'Commands and timers run through Bubble Tea’s event loop.', display:'BUBBLE TEA 2.1'},
  tinytea: {label:'TinyTea', note:'Cooperative commands; the host supplies animation ticks.', display:'TINYTEA'},
};
const supportedRuntimes = demo.runtimes || ['bubbletea'];
const requestedRuntime = pageURL.searchParams.get('runtime');
const runtime = supportedRuntimes.includes(requestedRuntime) ? requestedRuntime : supportedRuntimes[0];
for (const id of supportedRuntimes) {
  const option = document.createElement('option');
  option.value = id; option.textContent = runtimeInfo[id].label;
  runtimeSelect.append(option);
}
runtimeSelect.value = runtime;
runtimeSelect.disabled = supportedRuntimes.length === 1;
window.tinybubblesRuntime = runtime;
document.getElementById('runtime-note').textContent = `${runtimeInfo[runtime].note} Switching restarts the demo.`;
document.getElementById('runtime-label').textContent = runtimeInfo[runtime].display;
runtimeSelect.addEventListener('change', () => {
  const url = new URL(location.href);
  url.searchParams.set('runtime', runtimeSelect.value);
  location.assign(url);
});
for (const category of new Set(demos.map(entry => entry.category))) {
  const heading = document.createElement('h2');
  heading.textContent = category;
  document.getElementById('demo-nav').append(heading);
  for (const entry of demos.filter(entry => entry.category === category)) {
    const link = document.createElement('a');
    const url = new URL(pageURL);
    url.searchParams.set('demo', entry.id);
    link.href = url.href;
    link.textContent = entry.title;
    if (entry === demo) link.setAttribute('aria-current', 'page');
    document.getElementById('demo-nav').append(link);
  }
}
document.title = `${demo.title} · TinyBubbles`;
document.getElementById('demo-title').textContent = demo.title;
document.getElementById('demo-description').textContent = demo.description;
document.getElementById('demo-instructions').textContent = demo.instructions;
document.getElementById('demo-source').href = demo.source;
const requested = pageURL.searchParams.get('screen');
const screen = ['feather', 'oled', 'large'].includes(requested) ? requested : 'feather';
select.value = screen;
window.tinybubblesScreen = screen;
document.getElementById('device-label').textContent = {feather:'REVERSE TFT FEATHER',oled:'MONOCHROME OLED',large:'COLOR TFT'}[screen];
let terminal, disposeControls, unsubscribe;
let failed = false;
function stop(message, error = false) {
  failed = error;
  terminal?.disconnect();
  terminal?.actions.clear();
  status.textContent = message;
  canvas.dataset.state = error ? 'error' : 'stopped';
}
window.tinybubblesStopped = message => stop(message);
function fail(error) {
  console.error(error);
  stop(`Could not run the demo: ${error.message || error}. Try reloading.`, true);
}
select.addEventListener('change', () => {
  const url = new URL(location.href);
  url.searchParams.set('screen', select.value);
  location.assign(url);
});
document.getElementById('restart').addEventListener('click', () => location.reload());
try {
  terminal = new BoobaTerminal('protocol-terminal', {allowOSC7501Actions:true,renderer:'canvas2d',fontSize:12});
  await terminal.init();
  if (!terminal.supportsProgramStatus) throw new Error('Booba Terminal Actions is unavailable');
  const encode = gesture => encodeGesture(gesture, terminal.getMode(1));
  terminal.setActionKeyEncoder({canEncode: gesture => encode(gesture) !== null, encode});
  disposeControls = installActionBar(terminal.actions, {parent:document.getElementById('actions')});
  unsubscribe = terminal.actions.subscribe(() => {
    if (terminal.actions.snapshot && !failed) {
      status.textContent = 'Live · controls supplied by the Go model';
      canvas.dataset.state = 'ready';
    }
  });
  terminal.onActionError = fail;
  canvas.addEventListener('keydown', event => {
    if (event.ctrlKey || event.altKey || event.metaKey || event.shiftKey) return;
    const key = {Enter:'enter',Escape:'escape',ArrowRight:'right',q:'q'}[event.key];
    if (!key) return;
    const encoded = encode({key,modifiers:[]});
    if (encoded === null || canvas.dataset.state !== 'ready') return;
    event.preventDefault();
    terminal.adapter?.boobaWrite(encoded);
  });
  const go = new Go();
  const response = await fetch(new URL(demo.wasm, import.meta.url));
  if (!response.ok) throw new Error(`WASM download returned HTTP ${response.status}`);
  const {instance} = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
  go.run(instance).catch(fail);
  terminal.connectWasm();
  setTimeout(() => {
    if (!failed && canvas.dataset.state !== 'ready' && canvas.dataset.state !== 'stopped') {
      fail(new Error('Timed out waiting for the action handshake'));
    }
  }, 15000);
} catch (error) { fail(error); }
window.addEventListener('pagehide', () => {
  unsubscribe?.(); disposeControls?.(); terminal?.dispose();
}, {once:true});
