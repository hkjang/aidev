import {zipSync} from 'fflate';
const workerScope=self as unknown as {onmessage:(event:MessageEvent<Record<string,Uint8Array>>)=>void;postMessage:(value:unknown,transfer?:Transferable[])=>void};
workerScope.onmessage=(event)=>{try{const data=zipSync(event.data,{level:1});workerScope.postMessage({data},[data.buffer])}catch{workerScope.postMessage({error:'선택한 폴더를 ZIP으로 준비하지 못했습니다'})}};
