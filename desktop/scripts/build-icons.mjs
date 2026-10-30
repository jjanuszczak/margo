// Build native icons from the same approved wordmark used in the editor.
import { readFileSync, writeFileSync, mkdirSync, mkdtempSync, copyFileSync, rmSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const desktop=fileURLToPath(new URL('../',import.meta.url));
const icons=path.join(desktop,'src-tauri/icons');
const source=readFileSync(path.join(desktop,'../assets/new-logo-dark-bg.png')).toString('base64');
mkdirSync(icons,{recursive:true});
const svg=`<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024"><rect width="1024" height="1024" rx="210" fill="#191a1d"/><image x="60" y="374" width="904" height="275" preserveAspectRatio="xMidYMid meet" href="data:image/png;base64,${source}"/></svg>`;
writeFileSync(path.join(icons,'source.svg'),svg+'\n');
const temporary=mkdtempSync(path.join(tmpdir(),'margo-icons-'));
try {
 execFileSync(path.join(desktop,'node_modules/.bin/tauri'),['icon',path.join(icons,'source.svg'),'--output',temporary],{stdio:'ignore'});
 for(const name of ['icon.png','icon.icns','32x32.png','128x128.png','128x128@2x.png'])copyFileSync(path.join(temporary,name),path.join(icons,name));
} finally {rmSync(temporary,{recursive:true,force:true});}
