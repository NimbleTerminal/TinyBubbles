import { BoobaTerminal, installActionBar } from './booba/booba.js';
import { encodeGesture, keyFromEvent, allowedKeys } from './keys.js';

const $ = id => document.getElementById(id);
const canvas = $('display');
const url = new URL(location.href);
let terminal, disposeControls, unsubscribe, timer;
let stopped = false;
function selection(id, choices, selected) {
  const element = $(id);
  element.replaceChildren();
  for (const choice of choices) {
    const option = document.createElement('option');
    option.value = choice.id; option.textContent = choice.label || choice.title;
    element.append(option);
  }
  element.value = selected.id;
  element.disabled = choices.length < 2;
  element.onchange = () => {
    const next = new URL(location.href);
    next.searchParams.set(id, element.value);
    location.assign(next);
  };
}
window.tinybubblesConfigure = json => {
  const config = JSON.parse(json);
  const pick = (choices, name) => choices.find(c => c.id === url.searchParams.get(name)) || choices[0];
  const screen = pick(config.screens, 'screen');
  const demos = config.demos?.length ? config.demos : [{id:'default', title:config.title, description:'', source:config.source}];
  const demo = pick(demos, 'demo');
  const runtimes = demo.runtimes?.length ? demo.runtimes : [{id:'bubbletea', label:'Bubble Tea', description:''}];
  const runtime = pick(runtimes, 'runtime');
  document.title = demo.title === config.title ? config.title : `${demo.title} · ${config.title}`;
  $('brand').textContent = config.title;
  $('source').href = config.source;
  $('intro').textContent = config.description;
  $('demo-title').textContent = demo.title;
  $('demo-description').textContent = demo.description;
  $('demo-source').href = demo.source || config.source;
  $('note').textContent = config.note || 'Controls come from the Go model’s help map.';
  $('device-label').textContent = screen.label;
  $('device').dataset.screen = screen.id;
  $('runtime-label').textContent = runtime.label;
  selection('screen',config.screens,screen);
  selection('runtime',runtimes,runtime);
  $('runtime-settings').hidden = runtimes.length < 2;
  $('runtime-note').textContent = `${runtime.description} Switching restarts the demo.`;
  $('demo-nav').replaceChildren();
  $('catalog').hidden = !config.demos?.length;
  $('gallery').classList.toggle('single',!config.demos?.length);
  for (const entry of demos) {
    const link = document.createElement('a');
    const next = new URL(location.href); next.searchParams.set('demo',entry.id);
    link.href = next.href; link.textContent = entry.title;
    if (entry.id === demo.id) link.setAttribute('aria-current','page');
    $('demo-nav').append(link);
  }
  $('leds').replaceChildren();
  for (let i=0; i<(config.ledCount || 0); i++) {
    const led = document.createElement('span'); led.className='led'; $('leds').append(led);
  }
  return JSON.stringify({screen,demo,runtime});
};
function stop(message, error=false) {
  stopped = true;
  clearTimeout(timer);
  terminal?.disconnect(); terminal?.actions.clear();
  $('status').textContent = message;
  canvas.dataset.state = error ? 'error' : 'stopped';
}
window.tinybubblesStopped = stop;
function fail(error) {
  console.error(error);
  stop(`Could not run: ${error.message || error}. Try restarting.`,true);
}
$('restart').onclick = () => location.reload();
try {
  terminal = new BoobaTerminal('protocol-terminal',{allowOSC7501Actions:true,renderer:'canvas2d',fontSize:12});
  await terminal.init();
  if (!terminal.supportsProgramStatus) throw new Error('Booba Terminal Actions is unavailable');
  const encode = gesture => encodeGesture(gesture,terminal.getMode(1),allowedKeys(terminal.actions.snapshot));
  terminal.setActionKeyEncoder({canEncode:g => encode(g)!==null, encode});
  disposeControls = installActionBar(terminal.actions,{parent:$('actions')});
  unsubscribe = terminal.actions.subscribe(() => {
    if (stopped) return;
    const ready = !!terminal.actions.snapshot;
    canvas.dataset.state = ready ? 'ready' : 'waiting';
    $('status').textContent = ready ? 'Running. Click the display to use the keyboard.' : 'Waiting for controls…';
  });
  terminal.onActionError = fail;
  canvas.addEventListener('keydown',event => {
    const key = keyFromEvent(event);
    if (!key || stopped || canvas.dataset.state!=='ready') return;
    const bytes = encode({key,modifiers:[]});
    if (bytes===null) return;
    event.preventDefault(); terminal.adapter?.boobaWrite(bytes);
  });
  const go = new Go();
  const response = await fetch(new URL('app.wasm',import.meta.url));
  if (!response.ok) throw new Error(`WASM download returned HTTP ${response.status}`);
  const {instance} = await WebAssembly.instantiate(await response.arrayBuffer(),go.importObject);
  go.run(instance).catch(fail);
  if (!stopped) terminal.connectWasm();
  timer = setTimeout(() => {
    if (!stopped && canvas.dataset.state!=='ready') fail(new Error('Timed out waiting for controls'));
  },15000);
} catch(error) { fail(error); }
window.addEventListener('pagehide',()=>{
  clearTimeout(timer); unsubscribe?.(); disposeControls?.(); terminal?.dispose();
},{once:true});
