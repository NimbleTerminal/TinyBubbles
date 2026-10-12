import test from 'node:test';
import assert from 'node:assert/strict';
import {keyFromEvent,allowedKeys} from './files/keys.js';

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
