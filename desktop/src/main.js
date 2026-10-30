import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';
import { createSourceEditor } from './source-editor.js';
import { editorThemes, resolveAppearance, resolveEditorTheme, readPreference, savePreference } from './appearance.js';
import { getCurrentWindow } from '@tauri-apps/api/window';
import { reconcileBuffer, selectedAfterRefresh, isDirty, fitSlideRect } from './model.js';
import logoDark from '../../assets/new-logo-dark-bg.png';
import logoLight from '../../assets/new-logo-light-bg.png';
import './style.css';


const $ = id => document.getElementById(id);
let snapshot = null, selected = '', buffer = null, previewURL = null, overview = false, presentation = false, busy = false, autosaveTimer, diagnostics = [], renderGeneration = 0;
let notesPreview=false;
let polling=false, thumbnailRevision='', thumbnailCursor=0, thumbnailsDone=true, thumbnailPolling=false;
const thumbnails=new Map();
function applyThumbnails(result) {
 if(result.revision && result.revision!==thumbnailRevision)return;
 for(const item of result.thumbnails||[]) {thumbnails.set(item.id,item.image);thumbnailCursor++;}
 thumbnailsDone=!!result.thumbnails_done;
 for(const image of document.querySelectorAll('img[data-thumbnail]')) {
  const source=thumbnails.get(image.dataset.thumbnail);
  if(source){image.src=source;image.hidden=false;image.parentElement.classList.add('ready');}
 }
 if(result.thumbnail_error){thumbnailsDone=true;status('Thumbnails: '+result.thumbnail_error);}
}
async function startThumbnails() {
 if(!snapshot || thumbnailRevision===snapshot.revision)return;
 thumbnailRevision=snapshot.revision;thumbnailCursor=0;thumbnailsDone=false;
 try{const result=await request('thumbnails',{revision:thumbnailRevision});applyThumbnails(result);}
 catch(error){thumbnailsDone=true;status(error.message);}
}
async function pollThumbnails(){
 if(thumbnailsDone||thumbnailPolling||busy||!snapshot)return;
 thumbnailPolling=true;
 try{applyThumbnails(await request('thumbnail-status',{position:thumbnailCursor}));}
 catch(error){thumbnailsDone=true;status(error.message);}finally{thumbnailPolling=false;}
}

const editor=createSourceEditor($('editor'),content=>{
 if(!buffer)return;
 buffer.content=content;updateSaveState();
 clearTimeout(autosaveTimer);
 if($('autosave').checked&&!buffer.conflict&&!buffer.manualSave)autosaveTimer=setTimeout(()=>act(save),800);
});
const systemAppearance=matchMedia('(prefers-color-scheme: dark)');
let appearanceChoice=readPreference('margo.appearance',['dark','light','system'],'dark');
let editorThemeChoice=readPreference('margo.editor-theme',Object.keys(editorThemes),'auto');
for(const [value,label] of Object.entries(editorThemes)){const option=document.createElement('option');option.value=value;option.textContent=label;$('editor-theme').append(option);}
function applyAppearance(){
 const appearance=resolveAppearance(appearanceChoice,systemAppearance.matches);
 document.documentElement.dataset.appearance=appearance;
 $('appearance').value=appearanceChoice;$('editor-theme').value=editorThemeChoice;
 for(const image of document.querySelectorAll('.margo-logo'))image.src=appearance==='light'?logoLight:logoDark;
 editor.setTheme(resolveEditorTheme(editorThemeChoice,appearance));
 getCurrentWindow().setTheme(appearanceChoice==='system'?null:appearance).catch(error=>console.warn('Native theme:',error));
 requestAnimationFrame(()=>syncPreview());
}
$('appearance').onchange=()=>{appearanceChoice=$('appearance').value;savePreference('margo.appearance',appearanceChoice);applyAppearance();};
$('editor-theme').onchange=()=>{editorThemeChoice=$('editor-theme').value;savePreference('margo.editor-theme',editorThemeChoice);applyAppearance();};
systemAppearance.addEventListener('change',()=>{if(appearanceChoice==='system')applyAppearance();});
function status(message) { $('status').textContent=message; }
function updateSaveState() {
 $('save-state').textContent=buffer?.conflict?'Conflict':isDirty(buffer)?'Unsaved':'Saved';
 $('save').disabled=busy||!isDirty(buffer)||!!buffer?.conflict;
 $('conflict').hidden=!buffer?.conflict;
 if (isDirty(buffer)) markStale('Unsaved changes · preview shows saved source');
}
function markStale(message) { $('preview-state').textContent=message; $('preview-state').classList.add('stale'); }
function showDiagnostics(items) {
 diagnostics=items||[]; $('messages').replaceChildren();
 for (const d of diagnostics) {
  const button=document.createElement('button');button.className=`diagnostic ${d.severity}`;
  button.textContent=`${d.severity}: ${d.message}${d.path?` (${d.path}${d.line?`:${d.line}`:''})`:''}`;
  button.onclick=()=>act(async()=>{
   if(!d.path)return;
   let path=d.path.startsWith(snapshot.root+'/')?d.path.slice(snapshot.root.length+1):d.path;
   if(!await resolveDirty())return;
   const slide=snapshot.slides.find(s=>s.path===path||s.notes.some(n=>n.path===path));
   if(slide){selected=slide.id; renderNavigation();renderSourceOptions();}
   await loadSource(path);
   editor.goToLine(d.line||1);
  }); $('messages').append(button);
 }
}
async function request(op, params={}) {
 const result=await invoke('engine_request',{request:{protocol:1,op,...params}});
 if(result.protocol!==1)throw new Error('Incompatible engine protocol');
 if(result.error){
  if(result.conflict && result.file && buffer)buffer.conflict=result.file;
  if(result.diagnostics?.length)showDiagnostics(result.diagnostics);
  updateSaveState();throw new Error(result.error);
 }
 return result;
}
async function act(fn) {
 if(busy)return;busy=true;updateSaveState();
 for(const id of ['open','empty-open','recent','add','duplicate','delete','move-up','move-down','build','source','slides','overview','present','media','new-note','preview-notes'])$(id).inert=true;
 try{await fn();}catch(error){status(error.message||String(error));console.error(error);}
 finally{busy=false;for(const id of ['open','empty-open','recent','add','duplicate','delete','move-up','move-down','build','source','slides','overview','present','media','new-note','preview-notes'])$(id).inert=false;updateSaveState();}
}
$('dialog-cancel').onclick=()=>$('dialog').close('cancel');
function dialog(title, description, initial, comparison) {
 return new Promise(resolve=>{
  $('dialog').returnValue='';
  $('dialog-title').textContent=title;$('dialog-description').textContent=description;
  $('dialog-input').hidden=initial===undefined;$('dialog-input').value=initial||'';
  $('comparison').hidden=!comparison;
  if(comparison){$('disk-text').value=comparison.disk;$('local-text').value=comparison.local;}
  $('dialog-ok').textContent=initial!==undefined?'Create':'Continue';
  $('dialog').showModal();syncPreview();
  if(initial!==undefined)$('dialog-input').focus();
  $('dialog').addEventListener('close',()=>{syncPreview();resolve($('dialog').returnValue==='ok'?(initial!==undefined?$('dialog-input').value:true):null);},{once:true});
 });
}
async function resolveDirty() {
 clearTimeout(autosaveTimer);
 if(!isDirty(buffer))return true;
 if(buffer.conflict){status('Resolve the source conflict before leaving this file.');return false;}
 await save();return !isDirty(buffer);
}
function setEditor(file, {preserveHistory=false}={}) {
 buffer={...file,base:file.content,conflict:null};
 notesPreview=file.path!==currentSlide()?.path;updateNotesControl();
 editor.load(file,{preserveHistory});updateSaveState();
 $('source').value=file.path;
}
async function loadSource(path) {
 const {file}=await request('read',{path});setEditor(file,{preserveHistory:true});
}
function updateNotesControl(){
 const button=$('preview-notes');button.disabled=!currentSlide()?.notes.length;button.textContent=notesPreview?'Back to slide':'Preview notes';button.classList.toggle('active',notesPreview);
}
function currentSlide(){return snapshot?.slides.find(s=>s.id===selected);}
function renderSourceOptions() {
 const slide=currentSlide();$('source').replaceChildren();if(!slide)return;
 const option=(value,label)=>{const o=document.createElement('option');o.value=value;o.textContent=label;$('source').append(o);};
 option(slide.path,'Markdown · '+slide.path);
 for(const n of slide.notes)if(n.path)option(n.path,'Note · '+n.name);
 $('asset-list').textContent=[`${(slide.assets||[]).length} assets`,...(slide.assets||[])].join(' · ');
 // Legacy inline notes remain in the full Markdown source and are never rewritten.
 const inline=slide.notes.filter(n=>!n.path);if(inline.length)$('asset-list').textContent+=' · Inline notes are editable in Markdown';
}
function renderNavigation() {
 $('slides').replaceChildren();$('grid').replaceChildren();if(!snapshot)return;
 $('count').textContent=snapshot.slides.length;
 snapshot.slides.forEach((slide,i)=>{
  const button=document.createElement('button');button.className='slide-item'+(slide.id===selected?' selected':'');
  button.setAttribute('aria-current',slide.id===selected?'true':'false');button.draggable=true;
  const number=document.createElement('span');number.className='number';number.textContent=String(i+1).padStart(2,'0');
  const label=document.createElement('span');label.className='label';const title=document.createElement('strong');title.textContent=slide.title;label.append(title);
  const flags=[slide.draft?'Draft':'',slide.visibility&&slide.visibility!=='visible'?slide.visibility:''].filter(Boolean);
  if(flags.length){const small=document.createElement('small');small.textContent=flags.join(' · ');label.append(small);}
  const frame=document.createElement('span');frame.className='thumbnail-frame';
  const thumbnail=document.createElement('img');thumbnail.dataset.thumbnail=slide.id;thumbnail.alt='Rendered slide '+(i+1)+': '+slide.title;thumbnail.draggable=false;
  const image=thumbnails.get(slide.id);thumbnail.hidden=!image;if(image){thumbnail.src=image;frame.classList.add('ready');}
  const placeholder=document.createElement('span');placeholder.className='thumbnail-loading';placeholder.textContent='Rendering slide…';
  frame.append(thumbnail,placeholder);button.append(number,frame,label);button.onclick=()=>act(()=>selectSlide(slide.id));
  button.ondragstart=e=>e.dataTransfer.setData('text/plain',slide.id);
  button.ondragover=e=>e.preventDefault();button.ondrop=e=>{e.preventDefault();const id=e.dataTransfer.getData('text/plain');if(id&&id!==slide.id)act(()=>mutate('move',{slide:id,position:i+1}));};
  $('slides').append(button);
  const card=button.cloneNode(true);card.className='grid-card'+(slide.id===selected?' selected':'');card.querySelector('strong').className='grid-title';
  card.onclick=()=>act(async()=>{await selectSlide(slide.id);overview=false;updateModes();});
  card.ondragstart=button.ondragstart;card.ondragover=button.ondragover;card.ondrop=button.ondrop;$('grid').append(card);
 });
 updateNotesControl();
 const slide=currentSlide();$('slide-title').textContent=slide?.title||'';$('slide-meta').textContent=slide?.layout||'';
 $('move-up').disabled=snapshot.slides[0]?.id===selected;$('move-down').disabled=snapshot.slides.at(-1)?.id===selected;
}
async function selectSlide(id) {
 if(!await resolveDirty())return;
 selected=id;renderNavigation();renderSourceOptions();
 const slide=currentSlide();if(slide)await loadSource(slide.path);
 await renderPreview();
}
async function openProject(root) {
 if(!await resolveDirty())return;
 status('Opening project…');
 const result=await request('open',{root});snapshot=result.snapshot;selected=snapshot.slides[0]?.id||'';
 buffer=null;editor.reset();previewURL=null;thumbnails.clear();thumbnailRevision='';thumbnailCursor=0;overview=false;presentation=false;showDiagnostics([]);
 $('project').firstElementChild.textContent=snapshot.title||'Untitled deck';$('project').lastElementChild.textContent=`${snapshot.root} · ${snapshot.theme}`;$('project').title=snapshot.root;
 for(const id of ['overview','present','build','add'])$(id).disabled=false;
 $('format').replaceChildren();for(const f of snapshot.formats){const o=document.createElement('option');o.value=f;o.textContent=f.toUpperCase();$('format').append(o);}
 renderNavigation();updateModes();if(selected)await selectSlide(selected);
 await updateRecent(snapshot.root);await startThumbnails();status(`Opened ${snapshot.title} · ${snapshot.slides.length} slides`);
}
async function pickProject(){const path=await invoke('choose_path',{kind:'project'});if(path)await openProject(path);}
async function updateRecent(root=null){
 const recent=await invoke('recent_projects',{root});$('recent').replaceChildren();const empty=document.createElement('option');empty.value='';empty.textContent='Recent projects';$('recent').append(empty);
 for(const path of recent){const o=document.createElement('option');o.value=path;o.textContent=path.split('/').at(-1);o.title=path;$('recent').append(o);}
}
async function save() {
 clearTimeout(autosaveTimer);if(!isDirty(buffer)||buffer.conflict)return;
 const path=buffer.path,content=buffer.content;
 status('Saving source…');
 const result=await request('save',{path,content,revision:buffer.revision});
 if(buffer.path!==path)return;
 buffer.revision=result.file.revision;buffer.base=content;buffer.manualSave=false;updateSaveState();
 // Preserve editor state and undo history across saves.
 await refreshSnapshot();await renderPreview();await startThumbnails();status('Saved · '+path);
}
async function refreshSnapshot(){
 const {snapshot:next}=await request('snapshot');const index=snapshot?.slides.findIndex(s=>s.id===selected)||0;
 selected=selectedAfterRefresh(next.slides,selected,index);snapshot=next;renderNavigation();
 const path=buffer?.path;renderSourceOptions();if(path)$('source').value=path;
}
async function poll(){
 if(busy||polling||!snapshot||$('dialog').open||presentation)return;
 polling=true;
 const root=snapshot.root, path=buffer?.path;
 try{
  // Background checks never disable navigation. Ignore replies after a user action.
  if(path){
   try{
    const {file}=await request('read',{path});
    if(busy||snapshot.root!==root||buffer?.path!==path)return;
    const reconciled=reconcileBuffer(buffer,file);
    if(reconciled.conflict){buffer=reconciled;updateSaveState();status('External change conflicts with unsaved text. Choose Reload, Compare, or Keep.');}
    else if(file.revision!==buffer.revision){setEditor(file);status('Reloaded external source change');}
   }catch(error){if(busy||snapshot.root!==root)return;if(isDirty(buffer)){buffer.conflict={path,content:'[File missing or unreadable on disk]',revision:''};updateSaveState();}status(error.message);}
  }
  const {snapshot:next}=await request('snapshot');
  if(busy||snapshot.root!==root)return;
  if(next.revision!==snapshot.revision){
   const previous=selected;selected=selectedAfterRefresh(next.slides,selected,snapshot.slides.findIndex(s=>s.id===selected));snapshot=next;renderNavigation();renderSourceOptions();
   if(buffer)$('source').value=buffer.path;
   if(previous!==selected&&!isDirty(buffer)&&currentSlide())await loadSource(currentSlide().path);
   if(busy)return;
   await renderPreview();if(!busy)await startThumbnails();
  }
 }catch(error){if(!busy)status(error.message);}finally{polling=false;}
}
async function renderPreview(){
 if(!selected)return;
 const generation=++renderGeneration;markStale('Rendering through Margo…');
 try{
  const result=await request('preview',{slide:selected,presentation,notes:notesPreview&&!presentation,path:notesPreview&&buffer?.path!==currentSlide()?.path?buffer?.path:undefined});if(generation!==renderGeneration)return;
  previewURL=result.preview;showDiagnostics(result.diagnostics);
  await syncPreview(true);
  $('preview-state').textContent=isDirty(buffer)?'Unsaved changes · preview shows saved source':notesPreview&&!presentation?'Notes preview · '+snapshot.theme:'Margo preview · '+snapshot.theme;
  $('preview-state').classList.toggle('stale',isDirty(buffer));
 }catch(error){markStale('Preview failed · last successful render remains visible');throw error;}
}
let previewLayoutPending=false, previewLayoutAgain=false, pendingNavigation=false;
async function syncPreview(navigate=false){
 pendingNavigation ||= navigate;
 if(previewLayoutPending){previewLayoutAgain=true;return;}
 previewLayoutPending=true;
 try{
  do {
   previewLayoutAgain=false;
   const target=presentation?$('presentation-target'):$('preview-target');
   const rect=fitSlideRect(target.getBoundingClientRect());
   const visible=!!previewURL&&!overview&&!$('dialog').open&&!!snapshot&&rect.width>1&&rect.height>1;
   const url=pendingNavigation?previewURL:null;pendingNavigation=false;
   await invoke('preview_layout',{rect,visible,url,viewport:{width:window.innerWidth,height:window.innerHeight}});
  } while(previewLayoutAgain||pendingNavigation);
 }catch(error){if(previewURL)status('Preview window: '+error);}finally{previewLayoutPending=false;}
}
function updateModes(){
 $('empty').hidden=!!snapshot;$('workspace').hidden=!snapshot||overview;$('grid').hidden=!snapshot||!overview;
 $('overview').classList.toggle('active',overview);$('presentation').hidden=!presentation;requestAnimationFrame(()=>syncPreview());
}
async function mutate(op,params={}){
 if(!await resolveDirty())return;
 status(`${op} slide…`);
 const result=await request(op,{slide:selected,revision:snapshot.revision,...params});
 if(result.snapshot){snapshot=result.snapshot;selected=result.selected||selected;buffer=null;editor.reset();renderNavigation();renderSourceOptions();if(currentSlide())await loadSource(currentSlide().path);await renderPreview();await startThumbnails();}
 status('Slide '+op+' complete');
}
async function namedMutation(op){const name=await dialog(op==='create'?'Add a slide':'Duplicate slide','Enter a unique name for the new slide bundle.',op==='create'?'New slide':currentSlide().title+' copy');if(name?.trim())await mutate(op,{name});}
$('open').onclick=$('empty-open').onclick=()=>act(pickProject);
$('recent').onchange=()=>{const path=$('recent').value;if(path)act(()=>openProject(path));$('recent').value='';};
$('save').onclick=()=>act(save);
$('source').onchange=()=>act(async()=>{const target=$('source').value;if(await resolveDirty()){await loadSource(target);await renderPreview();}else $('source').value=buffer.path;});
$('overview').onclick=()=>act(async()=>{if(!await resolveDirty())return;overview=!overview;updateModes();});
$('add').onclick=()=>act(()=>namedMutation('create'));$('duplicate').onclick=()=>act(()=>namedMutation('duplicate'));
$('move-up').onclick=()=>act(()=>mutate('move',{position:snapshot.slides.findIndex(s=>s.id===selected)}));
$('move-down').onclick=()=>act(()=>mutate('move',{position:snapshot.slides.findIndex(s=>s.id===selected)+2}));
$('delete').onclick=()=>act(async()=>{if(await dialog('Delete this slide?',`“${currentSlide().title}” and its notes/assets will move to .margo-trash. You can recover the bundle from that folder.`))await mutate('delete');});
$('preview-notes').onclick=()=>act(async()=>{if(!await resolveDirty())return;notesPreview=!notesPreview;updateNotesControl();await renderPreview();});
$('new-note').onclick=()=>act(async()=>{
 if(!await resolveDirty())return;const name=await dialog('Add named note','This creates a separate Markdown file in the slide’s notes folder.','Speaker notes');if(!name?.trim())return;
 const result=await request('note',{slide:selected,name,revision:snapshot.revision});await refreshSnapshot();renderSourceOptions();setEditor(result.file);editor.focus();await renderPreview();await startThumbnails();status('Note created · '+result.file.path);
});
$('media').onclick=()=>act(async()=>{
 if(!await resolveDirty())return;const source=await invoke('choose_path',{kind:'media'});if(!source)return;
 const result=await request('media',{slide:selected,source,revision:snapshot.revision});await refreshSnapshot();
 if(buffer.path!==currentSlide().path)await loadSource(currentSlide().path);
 editor.insert('\n'+result.markdown+'\n');editor.focus();await save();
});
$('build').onclick=()=>act(async()=>{
 if(!await resolveDirty())return;status(`Exporting ${$('format').value.toUpperCase()}…`);
 const result=await request('build',{format:$('format').value});showDiagnostics(result.diagnostics);status(result.log||`Built ${result.artifact}`);
});
$('reload').onclick=()=>act(async()=>{
 if(!await dialog('Reload disk version?','This discards your unsaved text. Use Compare first if you need to copy part of it.'))return;
 await loadSource(buffer.path);await refreshSnapshot();await renderPreview();
});
$('compare').onclick=()=>act(async()=>{await dialog('Compare external changes','Copy any text you need, then cancel to return to your editor.',undefined,{disk:buffer.conflict.content,local:buffer.content});});
$('keep').onclick=()=>act(async()=>{
 if(!await dialog('Keep your text?', 'Your next explicit Save will replace the current disk version with your editor text. Compare first to merge changes. Auto-save pauses for this file until you press Save.'))return;
 const {file}=await request('read',{path:buffer.path});buffer.base=file.content;buffer.revision=file.revision;buffer.conflict=null;buffer.manualSave=true;updateSaveState();status('Your text kept. Press Save when ready to replace the disk version.');
});
async function setPresentation(value){if(!await resolveDirty())return;presentation=value;overview=false;updateModes();await renderPreview();}
$('present').onclick=()=>act(()=>setPresentation(true));$('exit-presentation').onclick=()=>act(()=>setPresentation(false));
function resizePreview(percent){percent=Math.max(20,Math.min(72,percent));document.documentElement.style.setProperty('--preview',percent+'%');$('divider').setAttribute('aria-valuenow',Math.round(percent));syncPreview();}
$('divider').onpointerdown=event=>{
 event.preventDefault();$('divider').focus();
 $('divider').setPointerCapture(event.pointerId);
 $('divider').onpointermove=e=>{const r=$('workspace').getBoundingClientRect();resizePreview((e.clientY-r.top-44)/r.height*100);};
 $('divider').onpointerup=()=>{$('divider').onpointermove=null;};
};
$('divider').onkeydown=e=>{if(e.key==='ArrowUp'||e.key==='ArrowDown'){e.preventDefault();resizePreview(Number($('divider').getAttribute('aria-valuenow'))+(e.key==='ArrowDown'?3:-3));}};
new ResizeObserver(()=>syncPreview()).observe($('preview-target'));new ResizeObserver(()=>syncPreview()).observe($('presentation-target'));
window.addEventListener('resize',()=>syncPreview());
window.addEventListener('keydown',event=>{
 if((event.metaKey||event.ctrlKey)&&event.key==='s'){event.preventDefault();act(save);return;}
 if(event.key==='Escape'&&presentation){act(()=>setPresentation(false));return;}
 if(event.defaultPrevented||editor.hasFocus()||$('dialog').open||event.target.closest('.monaco-editor,input,textarea,select'))return;
 if(event.key==='ArrowDown'||event.key==='ArrowUp'){
  event.preventDefault();const index=snapshot?.slides.findIndex(s=>s.id===selected);const slide=snapshot?.slides[index+(event.key==='ArrowDown'?1:-1)];if(slide)act(()=>selectSlide(slide.id));
 }
});
window.addEventListener('beforeunload',event=>{if(isDirty(buffer)){event.preventDefault();event.returnValue='Unsaved changes';}});
$('autosave').onchange=()=>{if($('autosave').checked&&isDirty(buffer)&&!buffer.conflict&&!buffer.manualSave)act(save);};
updateRecent().catch(error=>status(String(error)));setInterval(poll,1500);setInterval(pollThumbnails,700);

listen('margo-editor-history',event=>editor.history(event.payload)).catch(error=>status(String(error)));

listen('margo-close-requested',()=>act(async()=>{
 clearTimeout(autosaveTimer);
 if(isDirty(buffer)) {
  if(buffer.conflict) { if(!await dialog('Close with unsaved changes?', 'Your unsaved editor text will be discarded. The disk version remains unchanged.'))return; }
  else if(!await resolveDirty())return;
 }
 await invoke('close_app');
})).catch(error=>status(String(error)));

window.addEventListener('scroll',()=>syncPreview(),true);

applyAppearance();
