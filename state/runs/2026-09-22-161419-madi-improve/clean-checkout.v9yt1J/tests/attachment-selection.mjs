import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import ts from '../web/node_modules/typescript/lib/typescript.js';

// Compile the actual UI helper, without a second implementation in this test.
const source=await readFile(new URL('../web/src/attachments/selection.ts',import.meta.url),'utf8');
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ES2022}}).outputText;
const {attachmentSelectionRange:range}=await import('data:text/javascript;base64,'+Buffer.from(compiled).toString('base64'));
assert.deepEqual(range('앞\r\n선택🚦\r\n끝',2,6),[3,7]);
assert.deepEqual(range('앞\r선택🚦\r끝',2,6),[2,6]);
assert.deepEqual(range('앞\n선택🚦\n끝',2,6),[2,6]);
assert.deepEqual(range('🚦',0,2),[0,2]);
assert.equal(range('🚦',1,2),null);
assert.equal(range('🚦',0,1),null);
assert.equal(range('abc',0,0),null);
assert.equal(range('abc',0,9),null);
assert.equal(range('abc',-1,2),null);
assert.equal(range('abc',NaN,2),null);
console.log('PASS attachment selection canonical CRLF/CR/LF, emoji and invalid boundaries');
