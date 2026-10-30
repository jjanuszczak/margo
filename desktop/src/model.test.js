import { test } from 'node:test';
import assert from 'node:assert/strict';
import { reconcileBuffer, selectedAfterRefresh } from './model.js';
test('external edits reload clean buffers and preserve dirty text for reconciliation', () => {
 const buffer={path:'slides/a/index.md',base:'old',content:'old',revision:'1'};
 const disk={path:buffer.path,content:'agent',revision:'2'};
 assert.equal(reconcileBuffer(buffer,disk).content,'agent');
 const dirty=reconcileBuffer({...buffer,content:'my work'},disk);
 assert.equal(dirty.content,'my work');assert.equal(dirty.base,'old');assert.deepEqual(dirty.conflict,disk);
});
test('selection survives insertion and falls back predictably on deletion',()=>{
 assert.equal(selectedAfterRefresh([{id:'new'},{id:'a'},{id:'b'}],'b',1),'b');
 assert.equal(selectedAfterRefresh([{id:'a'},{id:'c'}],'b',1),'c');
 assert.equal(selectedAfterRefresh([],'a',0),'');
});

test('preview fits within its border at narrow, wide and portrait container sizes',async()=>{
 const {fitSlideRect}=await import('./model.js');
 for(const container of [{x:250,y:150,width:1000,height:350},{x:250,y:150,width:500,height:700},{x:0,y:0,width:1600,height:900}]){
  const fit=fitSlideRect(container);
  assert.ok(fit.x>=container.x+1&&fit.y>=container.y+1);
  assert.ok(fit.x+fit.width<=container.x+container.width-1.0000000000+1e-7);
  assert.ok(fit.y+fit.height<=container.y+container.height-1.0000000000+1e-7);
  assert.ok(Math.abs(fit.width/fit.height-16/9)<1e-10);
 }
});
