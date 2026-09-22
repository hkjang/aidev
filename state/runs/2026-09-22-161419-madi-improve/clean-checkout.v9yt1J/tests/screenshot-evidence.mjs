import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {lstat, readFile, readdir, realpath} from 'node:fs/promises';
import path from 'node:path';

// Reviewed normal product captures only. A passing suite never implicitly
// approves arbitrary new diagnostic PNGs for the public gallery.
export const approvedScreenshots = Object.fromEntries(Object.entries({
 'document-async':'',
 'document-foundations':'profile-expanded document-history document-diff workspace-audit mobile-workspace-audit mobile-document-history mobile-profile-expanded',
 browser:'login editor-source document workspace documents search favorites graph tasks databases templates trash import members profile api-keys admin-dashboard admin-users admin-settings admin-settings-oidc admin-settings-ai admin-settings-security admin-settings-workflow admin-settings-storage admin-settings-history admin-audit admin-backup database-table database-board database-calendar profile-dark profile-menu mobile-workspace mobile-menu mobile-admin',
 navigation:'presentation command-palette keyboard-settings source-location block-organizer mobile-navigation',
 'document-read-mode':'',
 'database-advanced':'database-advanced database-gallery database-list database-timeline database-board database-calendar mobile-database-gallery',
 operations:'workspace-branding workspace-features admin-user-features admin-telemetry admin-operations mobile-operations mobile-telemetry mobile-branding',
 'automation-browser':'automation-editor automations webhook-editor webhooks jobs job-detail admin-jobs automations-mobile webhooks-mobile jobs-mobile admin-jobs-mobile jobs-mobile-actions',
 'notification-browser':'notification-channel-editor admin-notification-channels notification-preferences notification-delivery-history notification-preferences-mobile',
 'inbound-capture-browser':'admin-inbound-capture imap-capture-editor inbound-capture-history captured-private-document inbound-capture-mobile',
 'storage-browser':'storage-local-editor admin-storage workspace-storage storage-s3-editor backup-schedule admin-storage-mobile workspace-storage-mobile backup-schedule-mobile',
 'migration-browser':'migration-center migration-preview migration-result admin-migration migration-result-mobile migration-center-mobile migration-history-mobile-actions',
 spaces:'spaces space-members workspace-settings organizations mobile-spaces mobile-workspace-settings',
 graph:'graph graph-local mobile-graph',
 templates:'template-library template-history templates mobile-templates template-editor',
 tasks:'task-properties tasks-list tasks-kanban tasks-calendar mobile-tasks-list mobile-tasks-kanban mobile-tasks-calendar',
 'tasks-html':'tasks-html-source',
 inbox:'inbox inbox-classify mobile-inbox mobile-inbox-classify',
 discussion:'teams document-discussion document-lifecycle mobile-teams mobile-document-discussion',
 knowledge:'document-knowledge-policy document-knowledge-history knowledge-health knowledge-quality-score mobile-knowledge-health mobile-document-knowledge',
 canvas:'canvas mobile-canvas',
 plugins:'admin-plugins plugin-permissions plugins plugin-block mobile-plugins',
 'enterprise-browser':'entity-create entities entity-profile enterprise entity-profile-mobile enterprise-mobile',
 'transfer-browser':'admin-export-policy migration-folder-preview migration-folder-result export-center export-results mobile-export mobile-import mobile-export-policy',
 pwa:'devices offline-vault mobile-devices',
}).map(([suite,names])=>[suite,names.split(' ').filter(Boolean).map(name=>name+'.png')]));

export const digest = bytes => createHash('sha256').update(bytes).digest('hex');
export const validBundle = name => /^(main|index)-[A-Za-z0-9_-]+\.js$/.test(name);
export const normalPNG = name => /^[a-z0-9][a-z0-9-]*\.png$/.test(name) && !/(fail|error|diagnostic|debug|raw|unapproved)/i.test(name);
const magic=Buffer.from([137,80,78,71,13,10,26,10]);

export async function safeDirectory(directory) {
 const resolved=path.resolve(directory);
 assert.equal(await realpath(resolved),resolved,'Screenshot directory must not traverse symlinks');
 assert.ok((await lstat(resolved)).isDirectory(),'Screenshot directory is not a directory');
 return resolved;
}
export async function readPNG(file) {
 const info=await lstat(file);
 assert.ok(info.isFile()&&!info.isSymbolicLink()&&info.size>24&&info.size<=16*1024*1024,'Screenshot must be a bounded regular PNG');
 const bytes=await readFile(file);
 assert.equal(bytes.length,info.size,'Screenshot changed during reading');
 assert.ok(bytes.subarray(0,8).equals(magic)&&bytes.toString('ascii',12,16)==='IHDR','Invalid PNG signature/header');
 const width=bytes.readUInt32BE(16),height=bytes.readUInt32BE(20);
 assert.ok(width>0&&height>0&&width<=32768&&height<=100000,'Invalid screenshot dimensions');
 return {bytes,mtime_ms:info.mtimeMs,ctime_ms:info.ctimeMs,sha256:digest(bytes),size:bytes.length};
}
export async function snapshotScreenshots(directory) {
 await safeDirectory(directory);
 const result={};
 for(const name of await readdir(directory)) {
  if(!normalPNG(name))continue;
  const file=await readPNG(path.join(directory,name));
  result[name]={mtime_ms:file.mtime_ms,ctime_ms:file.ctime_ms,size:file.size,sha256:file.sha256};
 }
 return result;
}
export async function freshScreenshots(directory,before,started,finished) {
 const after=await snapshotScreenshots(directory),result=[];
 for(const [name,item] of Object.entries(after)) {
  const old=before[name];
  if(old&&JSON.stringify(old)===JSON.stringify(item))continue;
  if(item.mtime_ms<started-2000||item.mtime_ms>finished+2000)continue;
  result.push({name,...item});
 }
 return result.sort((a,b)=>a.name.localeCompare(b.name));
}
