// This bootstrap executes only inside an opaque-origin, allow-scripts iframe.
// It contains no cookie, token, provider secret or network access.
export const PLUGIN_SDK_SOURCE = String.raw`
(()=>{
"use strict";
const NONCE=self.MADI_NONCE;
const pending=new Map(),registrations=new Map(),events=new Map();let sequence=0,context=null;
let resolveReady;const readyPromise=new Promise(resolve=>resolveReady=resolve);
const send=(type,payload)=>postMessage({channel:"madi-plugin-v1",nonce:NONCE,type,...payload});
function request(operation,args={},onChunk){
 if(pending.size>=16)return Promise.reject(new Error("동시 요청은 16개까지 가능합니다"));
 const id=String(++sequence);return new Promise((resolve,reject)=>{
  const timer=setTimeout(()=>{pending.delete(id);send("cancel",{id});reject(new Error("플러그인 요청 시간이 초과되었습니다"))},180000);
  pending.set(id,{resolve,reject,timer,onChunk});send("request",{id,operation,args});
 });
}
function register(kind,id,handler){
 if(typeof id!=="string"||typeof handler!=="function")throw new Error("확장 ID와 실행 함수를 입력하세요");
 const declared=context?.manifest?.contributions?.[kind]?.some(item=>item.id===id);
 if(!declared)throw new Error("manifest에 등록되지 않은 확장입니다: "+id);
 registrations.set(kind+":"+id,handler);send("registered",{kind,id});
}
addEventListener("message",async event=>{
 const data=event.data;if(!data||data.channel!=="madi-plugin-v1"||data.nonce!==NONCE)return;
 if(data.type==="ping"){send("pong",{});return}
 if(data.type==="init"&&!context){context=Object.freeze(data.context);resolveReady(context);return}
 if(data.type==="response"||data.type==="chunk"){
  const p=pending.get(data.id);if(!p)return;
  if(data.type==="chunk"){if(p.onChunk)try{p.onChunk(data.chunk)}catch{};return}
  clearTimeout(p.timer);pending.delete(data.id);data.error?p.reject(new Error(data.error)):p.resolve(data.result);return;
 }
 if(data.type==="ui-event") {try{await events.get(data.id)?.(data.value)}catch(error){send("plugin-error",{error:String(error?.message||error)})};return}
 if(data.type==="invoke"){
  try{const handler=registrations.get(data.kind+":"+data.contributionId);if(!handler)throw new Error("확장 기능이 준비되지 않았습니다");const result=await handler(data.input);send("invocation-result",{id:data.id,result:result===undefined?null:result})}
  catch(error){send("invocation-result",{id:data.id,error:String(error?.message||error)})}
 }
});
const methods=(prefix,names)=>Object.freeze(Object.fromEntries(names.map(name=>[name,args=>request(prefix+"."+name,args)])));
const madi={
 ready:()=>readyPromise,
 get context(){return context},
 documents:methods("documents",["list","get","create","update","delete"]),
 databases:methods("databases",["list","get","query","createRow","updateRow","deleteRow"]),
 ai:Object.freeze({chat:(args,onChunk)=>request("ai.chat",args,onChunk)}),
 storage:Object.freeze({get:async key=>(await request("storage.get",{key})).value,set:(key,value)=>request("storage.set",{key,value}),delete:key=>request("storage.delete",{key})}),
 notify:message=>request("ui.notify",{message}),navigate:path=>request("ui.navigate",{path}),
 asset:path=>{const f=context?.assets?.[path];if(!f)throw new Error("등록된 에셋이 없습니다");return "data:"+f.mime+";base64,"+f.data},
 ui:Object.freeze({render:node=>{events.clear();let count=0;const serialize=(item,depth=0)=>{if(depth>20||++count>500)throw new Error("화면 요소 한도를 초과했습니다");if(!item||typeof item!=="object")return {type:"text",text:String(item??"")};const out={};for(const[key,value]of Object.entries(item)){if((key==="onClick"||key==="onChange")&&typeof value==="function"){const id="event-"+(++sequence);events.set(id,value);out[key]=id}else if(key==="children"){out.children=(Array.isArray(value)?value:[value]).map(v=>serialize(v,depth+1))}else if(typeof value!=="function")out[key]=value}return out};send("ui-render",{tree:serialize(node)})}}),
 registerBlock:(id,fn)=>register("blocks",id,fn),registerCommand:(id,fn)=>register("commands",id,fn),
 registerSidebar:(id,fn)=>register("sidebars",id,fn),registerMenu:(id,fn)=>register("menus",id,fn),
 registerImporter:(id,fn)=>register("importers",id,fn),registerExporter:(id,fn)=>register("exporters",id,fn),
 registerAIProvider:(id)=>register("ai_providers",id,args=>request("ai.chat",args)),
};Object.defineProperty(self,"madi",{value:Object.freeze(madi),writable:false,configurable:false});
addEventListener("unhandledrejection",event=>send("plugin-error",{error:String(event.reason?.message||event.reason)}));
send("ready",{});
})();`;

export function pluginWorkerSource(nonce: string, script: string) {
  const entry = new TextDecoder().decode(
    Uint8Array.from(atob(script), (character) => character.charCodeAt(0)),
  );
  return (
    "self.MADI_NONCE=" +
    JSON.stringify(nonce) +
    ";\n" +
    PLUGIN_SDK_SOURCE +
    "\n" +
    entry
  );
}
