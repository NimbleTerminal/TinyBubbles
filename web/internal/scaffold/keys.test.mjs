import test from 'node:test';
import assert from 'node:assert/strict';
import {keyFromEvent,allowedKeys,textFromEvent} from './files/keys.js';

test('only enabled unmodified help gestures are permitted',()=>{
 const allowed=allowedKeys({actions:[{enabled:true,keys:[{key:'n',modifiers:[]},{key:'p',modifiers:['ctrl']}]},{enabled:false,keys:[{key:'q',modifiers:[]}]}]});
 assert.deepEqual([...allowed],['n']);
 assert.deepEqual([...allowedKeys(null)],[]);
});
test('KeyboardEvent mapping preserves typed characters and declines modified commands',()=>{
 const keys={ArrowUp:'up',ArrowDown:'down',ArrowLeft:'left',ArrowRight:'right',Enter:'enter',Escape:'escape',' ':'space',Tab:'tab',Backspace:'backspace',Home:'home',End:'end',Insert:'insert',Delete:'delete',PageUp:'pageup',PageDown:'pagedown'};
 for(let n=1;n<=12;n++)keys[`F${n}`]=`f${n}`;
 for(const [key,expected]of Object.entries(keys))assert.equal(keyFromEvent({key}),expected);
 assert.equal(keyFromEvent({key:'+',shiftKey:true}),'+');
 for (const key of ['=','é','😀']) assert.equal(keyFromEvent({key}),key);
 for (const key of ['\x1b','\x00','\x7f','ab']) assert.equal(keyFromEvent({key}),null);
 for(const event of [{key:'ArrowUp',shiftKey:true},{key:'n',ctrlKey:true},{key:'n',altKey:true},{key:'n',metaKey:true},{key:'n',isComposing:true},{key:'Dead'},{key:'F13'}])assert.equal(keyFromEvent(event),null);
});
test('TextInput mapping sends typed text and Ctrl control bytes, not named keys',()=>{
 for (const key of ['a','A',' ','+','é','😀']) assert.equal(textFromEvent({key}),key);
 assert.equal(textFromEvent({key:'c',ctrlKey:true}),'\x03');
 assert.equal(textFromEvent({key:'L',ctrlKey:true,shiftKey:true}),'\x0c');
 assert.equal(textFromEvent({key:']',ctrlKey:true}),'\x1d');
 assert.equal(textFromEvent({key:' ',ctrlKey:true}),'\x00');
 for (const event of [{key:'Enter'},{key:'ArrowUp'},{key:'Tab'},{key:'x',altKey:true},{key:'v',metaKey:true},{key:'1',ctrlKey:true},{key:'ArrowUp',ctrlKey:true},{key:'n',isComposing:true},{key:'Dead'}])
  assert.equal(textFromEvent(event),null);
});
