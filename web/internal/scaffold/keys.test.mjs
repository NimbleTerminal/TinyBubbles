import test from 'node:test';
import assert from 'node:assert/strict';
import {encodeGesture,keyFromEvent,allowedKeys} from './files/keys.js';
const named={up:'\x1b[A',down:'\x1b[B',right:'\x1b[C',left:'\x1b[D',home:'\x1b[H',end:'\x1b[F',insert:'\x1b[2~',delete:'\x1b[3~',pageup:'\x1b[5~',pagedown:'\x1b[6~',enter:'\r',escape:'\x1b',space:' ',tab:'\t',backspace:'\x7f',f1:'\x1bOP',f2:'\x1bOQ',f3:'\x1bOR',f4:'\x1bOS',f5:'\x1b[15~',f6:'\x1b[17~',f7:'\x1b[18~',f8:'\x1b[19~',f9:'\x1b[20~',f10:'\x1b[21~',f11:'\x1b[23~',f12:'\x1b[24~'};
const encode=(key,app=false,allowed=new Set(Object.keys(named)))=>encodeGesture({key,modifiers:[]},app,allowed);
test('all Booba named keys encode',()=>{for(const [key,bytes] of Object.entries(named))assert.equal(encode(key),bytes,key);});
test('application cursor mode covers arrows and home/end',()=>{
 for(const [key,letter] of Object.entries({up:'A',down:'B',right:'C',left:'D',home:'H',end:'F'}))assert.equal(encode(key,true),`\x1bO${letter}`);
 assert.equal(encode('f5',true),'\x1b[15~');
});
test('printable Unicode and punctuation pass, control bytes do not',()=>{
 for(const key of ['+','-','n','p','A','é','😀'])assert.equal(encode(key,false,new Set([key])),key);
 for(const key of ['\x1b','\x00','\x7f','ab','F13'])assert.equal(encode(key,false,new Set([key])),null);
});
test('only enabled unmodified help gestures are permitted',()=>{
 const allowed=allowedKeys({actions:[{enabled:true,keys:[{key:'n',modifiers:[]},{key:'p',modifiers:['ctrl']}]},{enabled:false,keys:[{key:'q',modifiers:[]}]}]});
 assert.equal(encode('n',false,allowed),'n');
 for(const key of ['p','q','x'])assert.equal(encode(key,false,allowed),null);
 assert.equal(encodeGesture({key:'n',modifiers:['shift']},false,allowed),null);
 assert.equal(encodeGesture({key:'n',modifiers:[]}),null);
});
test('KeyboardEvent mapping preserves typed characters and declines modified commands',()=>{
 const keys={ArrowUp:'up',ArrowDown:'down',ArrowLeft:'left',ArrowRight:'right',Enter:'enter',Escape:'escape',' ':'space',Tab:'tab',Backspace:'backspace',Home:'home',End:'end',Insert:'insert',Delete:'delete',PageUp:'pageup',PageDown:'pagedown'};
 for(let n=1;n<=12;n++)keys[`F${n}`]=`f${n}`;
 for(const [key,expected]of Object.entries(keys))assert.equal(keyFromEvent({key}),expected);
 assert.equal(keyFromEvent({key:'+',shiftKey:true}),'+');
 assert.equal(keyFromEvent({key:'=',shiftKey:false}),'=');
 for(const event of [{key:'ArrowUp',shiftKey:true},{key:'n',ctrlKey:true},{key:'n',altKey:true},{key:'n',metaKey:true},{key:'n',isComposing:true},{key:'Dead'},{key:'F13'}])assert.equal(keyFromEvent(event),null);
});
