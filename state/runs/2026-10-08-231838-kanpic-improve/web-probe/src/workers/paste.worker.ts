/// <reference lib="webworker" />
import { materializePaste, type PasteMode } from '../lib/clipboard'
import type { HtmlCell } from '../lib/clipboardHtml'

// The clipboard HTML is parsed on the main thread: a worker has no DOMParser.
self.onmessage = (event: MessageEvent<{text:string;internal?:string;startRow:number;startColumn:number;mode?:PasteMode;htmlCells?:HtmlCell[]}>) => {
  try{
    self.postMessage({cells:materializePaste(event.data.text,event.data.internal,event.data.startRow,event.data.startColumn,event.data.mode??'all',event.data.htmlCells)})
  }catch(error){
    self.postMessage({error:error instanceof Error?error.message:'붙여넣기 데이터를 처리하지 못했습니다.'})
  }
}
