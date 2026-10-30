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
let appearanceChoice=readPreference('margo.appearance',['dark','light','system','omarchy'],'dark');
let editorThemeChoice=readPreference('margo.editor-theme',Object.keys(editorThemes),'auto');
const appearanceLabels={light:'Light',dark:'Dark',system:'System',omarchy:'Omarchy'};
let omarchyPalette=null;
const omarchyColorKeys={
 '--background':'background','--panel':'lighter_background','--control':'background',
 '--control-hover':'lighter_background','--text':'foreground','--muted':'muted',
 '--line':'muted','--accent':'accent','--selected':'selection','--selected-border':'accent',
 '--well':'dark_background','--frame':'darker_background','--frame-line':'muted',
 '--divider':'muted','--warning':'yellow','--error':'red','--conflict':'selection',
 '--conflict-text':'foreground'
};
const appearanceIcons={
 light:'<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42"/></svg>',
 dark:'<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20.5 15.5A8.5 8.5 0 0 1 8.5 3.5 8.5 8.5 0 1 0 20.5 15.5z"/></svg>',
 system:'<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="4" width="18" height="13" rx="2"/><path d="M8 21h8m-4-4v4"/></svg>',
 omarchy:'<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18"/><circle cx="7" cy="6.5" r=".7"/><circle cx="10" cy="6.5" r=".7"/><path d="M7 13h4m-4 3h7"/></svg>'
};
for(const [value,label] of Object.entries(editorThemes)){const option=document.createElement('option');option.value=value;option.textContent=label;$('editor-theme').append(option);}
function applyAppearance(){
 const palette=appearanceChoice==='omarchy'?omarchyPalette:null;
 const appearance=palette?.available?palette.mode:resolveAppearance(appearanceChoice==='omarchy'?'system':appearanceChoice,systemAppearance.matches);
 document.documentElement.dataset.appearance=appearance;
 document.documentElement.dataset.appearanceSource=palette?.available?'omarchy':'system';
 document.documentElement.style.colorScheme=appearance;
 for(const [property,key] of Object.entries(omarchyColorKeys)){
  const value=palette?.available?palette.colors?.[key]:null;
  if(value)document.documentElement.style.setProperty(property,value);else document.documentElement.style.removeProperty(property);
 }
 $('appearance-icon').innerHTML=appearanceIcons[appearanceChoice];
 const unavailable=appearanceChoice==='omarchy'&&!palette?.available;
 $('appearance-toggle').title='Appearance: '+appearanceLabels[appearanceChoice]+(unavailable?' (Omarchy palette unavailable; using system colors)':'');
 $('appearance-toggle').setAttribute('aria-label','Appearance: '+appearanceLabels[appearanceChoice]+(unavailable?' (using system colors)':''));
 for(const option of document.querySelectorAll('[data-appearance-choice]'))option.setAttribute('aria-checked',String(option.dataset.appearanceChoice===appearanceChoice));
 $('editor-theme').value=editorThemeChoice;
 for(const image of document.querySelectorAll('.margo-logo'))image.src=appearance==='light'?logoLight:logoDark;
 editor.setTheme(resolveEditorTheme(editorThemeChoice,appearance));
 getCurrentWindow().setTheme(appearanceChoice==='system'?null:appearance).catch(error=>console.warn('Native theme:',error));
 requestAnimationFrame(()=>syncPreview());
}
async function refreshOmarchyPalette(){
 try{omarchyPalette=await invoke('omarchy_palette');}
 catch(error){omarchyPalette=null;console.warn('Omarchy palette:',error);}
 if(appearanceChoice==='omarchy')applyAppearance();
}
$('appearance-menu').querySelectorAll('[data-appearance-choice]').forEach(option=>option.onclick=()=>{
 appearanceChoice=option.dataset.appearanceChoice;savePreference('margo.appearance',appearanceChoice);$('appearance-menu').open=false;applyAppearance();if(appearanceChoice==='omarchy')refreshOmarchyPalette();
});
document.addEventListener('pointerdown',event=>{if($('appearance-menu').open&&!$('appearance-menu').contains(event.target))$('appearance-menu').open=false;});
document.addEventListener('keydown',event=>{if(event.key==='Escape'&&$('appearance-menu').open){$('appearance-menu').open=false;$('appearance-toggle').focus();}});
$('editor-theme').onchange=()=>{editorThemeChoice=$('editor-theme').value;savePreference('margo.editor-theme',editorThemeChoice);applyAppearance();};
systemAppearance.addEventListener('change',()=>{if(appearanceChoice==='system'||(appearanceChoice==='omarchy'&&!omarchyPalette?.available))applyAppearance();});
refreshOmarchyPalette();setInterval(()=>{if(appearanceChoice==='omarchy')refreshOmarchyPalette();},2500);
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
 for(const id of ['new-project','open','empty-new','empty-open','agent-skills','tile-content','add','duplicate','delete','move-up','move-down','source','slides','overview','present','media','new-note','preview-notes'])$(id).inert=true;
 try{await fn();}catch(error){status(error.message||String(error));console.error(error);}
 finally{busy=false;for(const id of ['new-project','open','empty-new','empty-open','agent-skills','tile-content','add','duplicate','delete','move-up','move-down','source','slides','overview','present','media','new-note','preview-notes'])$(id).inert=false;updateSaveState();}
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
 for(const id of ['overview','present','add'])$(id).disabled=false;
 $('agent-skills').disabled=false;$('tile-content').disabled=false;
 $('tile-content').title=snapshot.supports_tiles?'Insert a nested tile composition':'Theme does not provide the nested tile shortcodes';
 renderNavigation();updateModes();if(selected)await selectSlide(selected);
 await updateRecent(snapshot.root);await startThumbnails();status(`Opened ${snapshot.title} · ${snapshot.slides.length} slides`);
}
async function pickProject(){const path=await invoke('choose_path',{kind:'project'});if(path)await openProject(path);}
async function newProject(){
 if(!await resolveDirty())return;
 const parent=await invoke('choose_path',{kind:'project-parent'});if(!parent)return;
 const name=await dialog('Create a deck','Choose a project folder name. Margo will create a complete starter deck here.','My presentation');
 if(!name?.trim())return;
 const folderName=name.trim().replace(/[\\/:*?"<>|]/g,'-');
 const root=parent.replace(/[\\/]$/,'')+'/'+folderName;
 const repo=await dialog('Optional theme import','Enter a Git repository URL for a Margo theme, or leave blank to use the default theme.','');
 const themeRepo=repo?.trim()||'';
 const themeRef=themeRepo?(await dialog('Theme revision','Optional Git branch, tag, or commit. Leave blank for the repository default.','')||'').trim():'';
 status('Creating deck…');
 try { await request('create-project',{root,name:name.trim(),theme_repo:themeRepo,theme_ref:themeRef}); }
 catch(error) { try { await openProject(root); } catch {} throw error; }
 await openProject(root);status('Created deck · '+root);
}
async function updateRecent(root=null){
 await invoke('recent_projects',{root});
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
 if(busy||polling||!snapshot||appModalOpen()||presentation)return;
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
function appModalOpen(){return ['dialog','skills-dialog','tiling-dialog'].some(id=>$(id).open);}
async function syncPreview(navigate=false){
 pendingNavigation ||= navigate;
 if(previewLayoutPending){previewLayoutAgain=true;return;}
 previewLayoutPending=true;
 try{
  do {
   previewLayoutAgain=false;
   const target=presentation?$('presentation-target'):$('preview-target');
   const rect=fitSlideRect(target.getBoundingClientRect());
   const visible=!!previewURL&&!overview&&!appModalOpen()&&!!snapshot&&rect.width>1&&rect.height>1;
   const url=pendingNavigation?previewURL:null;pendingNavigation=false;
   await invoke('preview_layout',{rect,visible,url,viewport:{width:window.innerWidth,height:window.innerHeight}});
  } while(previewLayoutAgain||pendingNavigation);
 }catch(error){if(previewURL)status('Preview window: '+error);}finally{previewLayoutPending=false;}
}
for(const id of ['dialog','skills-dialog','tiling-dialog'])$(id).addEventListener('close',()=>requestAnimationFrame(()=>syncPreview()));
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
const tilingDialog=$('tiling-dialog');
let tilingLayout='stack';
const layoutRadios=[...document.querySelectorAll('input[name=tiling-layout]')];
function refreshTilingSettings(){
 $('tiling-ratio-label').hidden=tilingLayout!=='stack';
 $('tiling-track-label').hidden=tilingLayout!=='track';
 const count=Math.max(2,Math.min(6,Number($('tiling-count').value)||3));
 $('tiling-track-count').max=Math.min(3,count);
 if(Number($('tiling-track-count').value)>Number($('tiling-track-count').max))$('tiling-track-count').value=$('tiling-track-count').max;
}
for(const radio of layoutRadios)radio.onchange=()=>{tilingLayout=radio.value;refreshTilingSettings();};
$('tiling-count').oninput=refreshTilingSettings;
$('tiling-track-count').oninput=refreshTilingSettings;
$('tiling-cancel').onclick=()=>tilingDialog.close('cancel');
$('tiling-create').onclick=()=>tilingDialog.close('insert');
function composeTiles(layout,count,ratio,trackCount){
 const tile=(body,params='')=>'{{< tile'+(params?' '+params:'')+' >}}\n'+body+'\n{{< /tile >}}';
 const tiles=(axis,children)=>'{{< tiles axis="'+axis+'" >}}\n'+children.join('\n')+'\n{{< /tiles >}}';
 const fill=index=>'Add content for area '+index+' here.';
 const leaf=index=>tile(fill(index));
 if(layout==='columns')return tiles('horizontal',Array.from({length:count},(_,i)=>leaf(i+1)));
 if(layout==='stack'){
  const supporting=Array.from({length:count-1},(_,i)=>leaf(i+2));
  const stack=supporting.length===1?fill(2):tiles('vertical',supporting);
  return tiles('horizontal',[tile('Add primary content here.','width="'+ratio+'%"'),tile(stack)]);
 }
 if(layout==='bsp'){
  const split=(start,size,depth)=>{
   if(size===1)return fill(start);
   const leftSize=Math.ceil(size/2),rightSize=size-leftSize,axis=depth%2===0?'horizontal':'vertical';
   return tiles(axis,[tile(split(start,leftSize,depth+1)),tile(split(start+leftSize,rightSize,depth+1))]);
  };
  return split(1,count,0);
 }
 if(layout==='grid'){
  const columns=Math.ceil(Math.sqrt(count)),rows=[];
  for(let start=0;start<count;start+=columns){
   const row=Array.from({length:Math.min(columns,count-start)},(_,i)=>leaf(start+i+1));
   rows.push(tile(row.length===1?fill(start+1):tiles('horizontal',row)));
  }
  return tiles('vertical',rows);
 }
 if(layout==='track'){
  const tracks=[];let start=1;
  for(let track=0;track<trackCount;track++){
   const size=Math.floor(count/trackCount)+(track<count%trackCount?1:0);
   const content=Array.from({length:size},(_,i)=>leaf(start+i));
   start+=size;
   tracks.push(tile(content.length===1?fill(start-1):tiles('vertical',content)));
  }
  return tiles('horizontal',tracks);
 }
 return '';
}
$('tile-content').onclick=()=>act(async()=>{
 if(!snapshot?.supports_tiles){status('This theme does not provide nested tile shortcodes. Use Agent skills to hand off the explicit margo-theme-tiling workflow.');await showProjectSkills('margo-theme-tiling');return;}
 if(!await resolveDirty())return;
 refreshTilingSettings();
 tilingDialog.returnValue='';tilingDialog.showModal();syncPreview();
 await new Promise(resolve=>tilingDialog.addEventListener('close',resolve,{once:true}));
 if(tilingDialog.returnValue!=='insert')return;
 const count=Math.max(2,Math.min(6,Number($('tiling-count').value)||3));
 const ratio=Math.max(25,Math.min(75,Number($('tiling-ratio').value)||65));
 const trackCount=Math.max(2,Math.min(3,count,Number($('tiling-track-count').value)||2));
 const source=composeTiles(tilingLayout,count,ratio,trackCount)+'\n';
 editor.insert('\n'+source);editor.focus();await save();status('Tiling source inserted · edit the Markdown placeholders');
});
$('agent-skills').onclick=()=>act(()=>showProjectSkills());
let agentSkills=[];
async function showProjectSkills(preferred=''){
 const result=await request('skills');agentSkills=result.skills||[];const select=$('skills-list');select.replaceChildren();
 for(const skill of agentSkills){const option=document.createElement('option');option.value=skill.path;option.textContent=skill.name+' — '+skill.description;select.append(option);}
 if(!agentSkills.length){status('No project skills are available.');return;}
 if(preferred){const match=agentSkills.find(skill=>skill.name===preferred||skill.path.includes('/'+preferred+'/'));if(match)select.value=match.path;}
 const update=()=>{const skill=agentSkills.find(item=>item.path===select.value);$('skills-content').value=skill?buildAgentPrompt(skill):'';$('skills-copy').disabled=!$('agent-task').value.trim();};select.onchange=update;$('agent-task').oninput=update;update();
 $('skills-dialog').showModal();syncPreview();
}
function buildAgentPrompt(skill){
 const slide=currentSlide();
 const skillBody=skill.content.replace(/^---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$)/,'').trim();
 const context=[
  'Margo is a local-first Markdown presentation engine and CLI. A deck is a directory containing margo.yaml, slide bundles under slides/, and a selected theme under themes/. The Go engine and CLI define project behavior; source files are canonical.',
  'Project root: '+snapshot.root,
  'Deck title: '+(snapshot.title||'Untitled deck'),
  'Project configuration: '+snapshot.root+'/margo.yaml',
  'Active theme: '+snapshot.theme+' ('+snapshot.root+'/themes/'+snapshot.theme+'/)',
  'Project guidance: '+snapshot.root+'/AGENTS.md',
  'Selected slide: '+(slide?slide.title+' ['+slide.id+']':'none'),
  'Selected slide source: '+(slide?snapshot.root+'/'+slide.path:'none'),
  'Currently open source file: '+(buffer?snapshot.root+'/'+buffer.path:'none'),
  'Skill file: '+snapshot.root+'/'+skill.path,
  'The agent should have this project folder open or otherwise mounted at the path above.'
 ].join('\n');
 const references=(skill.references||[]).map(ref=>'Reference file: '+snapshot.root+'/'+ref.path+'\n'+ref.content.trim()).join('\n\n');
 return [
  'Task for the agent:\n'+$('agent-task').value.trim(),
  'Project context:\n'+context,
  'Project skill: '+skill.name+'\n'+skill.description+'\n\n'+skillBody,
  references?'Skill reference material:\n'+references:'',
  'Work within the project root. Follow AGENTS.md and the skill guidance, preserve unrelated source and user changes, and do not edit generated dist/ output. When finished, summarize what changed and any checks performed.'
 ].filter(Boolean).join('\n\n');
}
$('skills-close').onclick=()=>$('skills-dialog').close();
$('skills-copy').onclick=async()=>{
 if(!$('agent-task').value.trim()){status('Describe the task before copying the handoff prompt.');return;}
 try{await navigator.clipboard.writeText($('skills-content').value);$('skills-dialog').close();status('Complete agent prompt copied. Paste it into an agent with access to this deck folder.');}
 catch{$('skills-content').focus();$('skills-content').select();status('Clipboard access is unavailable. The skill text is selected; copy it manually.');}
};
$('open').onclick=$('empty-open').onclick=()=>act(pickProject);
$('new-project').onclick=$('empty-new').onclick=()=>act(newProject);
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
async function exportDeck(format){
 if(!snapshot||!snapshot.formats.includes(format)){status(format.toUpperCase()+' export is unavailable in this environment.');return;}
 if(!await resolveDirty())return;
 status('Exporting '+format.toUpperCase()+'…');
 const result=await request('build',{format});showDiagnostics(result.diagnostics);status(result.log||'Built '+result.artifact);
}
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
 if(event.defaultPrevented||editor.hasFocus()||appModalOpen()||event.target.closest('.monaco-editor,input,textarea,select'))return;
 if(event.key==='ArrowDown'||event.key==='ArrowUp'){
  event.preventDefault();const index=snapshot?.slides.findIndex(s=>s.id===selected);const slide=snapshot?.slides[index+(event.key==='ArrowDown'?1:-1)];if(slide)act(()=>selectSlide(slide.id));
 }
});
window.addEventListener('beforeunload',event=>{if(isDirty(buffer)){event.preventDefault();event.returnValue='Unsaved changes';}});
$('autosave').onchange=()=>{if($('autosave').checked&&isDirty(buffer)&&!buffer.conflict&&!buffer.manualSave)act(save);};
updateRecent().catch(error=>status(String(error)));setInterval(poll,1500);setInterval(pollThumbnails,700);

listen('margo-editor-history',event=>editor.history(event.payload)).catch(error=>status(String(error)));
listen('margo-native-menu',event=>{
 const {action,root,format}=event.payload||{};
 if(action==='new-project')act(newProject);
 else if(action==='open-project')act(root?()=>openProject(root):pickProject);
 else if(action==='export'&&format)act(()=>exportDeck(format));
}).catch(error=>status(String(error)));

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
