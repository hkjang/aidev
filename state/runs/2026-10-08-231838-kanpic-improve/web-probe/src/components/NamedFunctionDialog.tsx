import { FunctionSquare,Plus } from 'lucide-react'
import { useState } from 'react'
import type { NamedFunction } from '../types'
import './NamedFunctionDialog.css'
import { useDialog } from '../lib/useDialog'

type Draft={name:string;parameters:string;body:string;description:string}

// 이름 규칙은 서버와 같아야 한다. 여기서만 느슨하면 저장할 때 거절당하고,
// 여기서만 빡빡하면 쓸 수 있는 이름을 막는다.
const cellPosition=(value:string)=>/^\$?[A-Za-z]{1,3}\$?[1-9]\d*$/.test(value.trim())
const validName=(value:string)=>{const name=value.trim();return /^[\p{L}_][\p{L}\p{N}_.]*$/u.test(name)&&!cellPosition(name)&&!/^(true|false)$/i.test(name)}
const parseParameters=(value:string)=>value.split(',').map(item=>item.trim()).filter(Boolean)
const validParameters=(value:string)=>{const items=parseParameters(value);return items.every(validName)&&new Set(items.map(item=>item.toUpperCase())).size===items.length}

export function NamedFunctionDialog({functions,onClose,onCreate,onUpdate,onDelete}:{
  functions:NamedFunction[];onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<NamedFunction>
  onUpdate:(id:string,input:Record<string,unknown>)=>Promise<NamedFunction>
  onDelete:(item:NamedFunction)=>Promise<void>
}){
  const initial=():Draft=>({name:'',parameters:'',body:'',description:''})
  const [selectedId,setSelectedId]=useState<string>(),[draft,setDraft]=useState<Draft>(initial),[saving,setSaving]=useState(false)
  const selected=functions.find(item=>item.id===selectedId)
  const choose=(item?:NamedFunction)=>{setSelectedId(item?.id);setDraft(item?{name:item.name,parameters:item.parameters.join(', '),body:item.body,description:item.description??''}:initial())}
  const save=async()=>{
    if(!validName(draft.name))return alert('이름은 문자나 밑줄로 시작하고 문자·숫자·밑줄·마침표만 쓸 수 있습니다.')
    if(!validParameters(draft.parameters))return alert('매개변수 이름을 확인하세요. 같은 이름을 두 번 쓸 수 없습니다.')
    if(!draft.body.trim())return alert('수식을 입력하세요.')
    setSaving(true)
    try{
      const input={name:draft.name.trim(),parameters:parseParameters(draft.parameters),body:draft.body.trim(),description:draft.description.trim()}
      const saved=selected?await onUpdate(selected.id,{...input,expected_revision:selected.revision}):await onCreate(input)
      choose(saved)
    }catch(error){alert(error instanceof Error?error.message:'이름 있는 수식을 저장하지 못했습니다.')}
    finally{setSaving(false)}
  }
  const remove=async(item:NamedFunction)=>{
    // 지우면 그것을 쓰던 칸이 모두 #NAME? 이 된다. 되돌릴 수 없는 일은 아니지만
    // 무엇이 깨지는지 미리 말해 준다.
    if(!confirm(`${item.name} 을(를) 지울까요? 이 수식을 쓰는 칸은 #NAME? 이 됩니다.`))return
    setSaving(true)
    try{await onDelete(item);choose()}catch(error){alert(error instanceof Error?error.message:'이름 있는 수식을 지우지 못했습니다.')}finally{setSaving(false)}
  }
  const dialog=useDialog<HTMLElement>(onClose)
  const example=`=${draft.name||'마진율'}(${parseParameters(draft.parameters).map((_,index)=>index===0?'A1':`${String.fromCharCode(66+index-1)}1`).join(', ')||''})`
  return <div className="modal-backdrop"><div className="modal named-function-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="이름 있는 수식">
    <header><div><FunctionSquare/><div><h2>이름 있는 수식</h2><p>자주 쓰는 셈을 한 번 정의해 두고 함수처럼 부릅니다.</p></div></div><button aria-label="이름 있는 수식 닫기" onClick={onClose}>×</button></header>
    <div className="named-function-layout">
      <aside>
        <button className={!selected?'active':''} onClick={()=>choose()}><Plus/> 새 수식</button>
        {functions.map(item=><button key={item.id} className={selected?.id===item.id?'active':''} onClick={()=>choose(item)}>
          <span>{item.name}({item.parameters.join(', ')})</span><em>{item.description||item.body}</em>
        </button>)}
      </aside>
      <section>
        <label>이름<input aria-label="수식 이름" value={draft.name} maxLength={255} onChange={event=>setDraft(current=>({...current,name:event.target.value}))} placeholder="마진율"/></label>
        <label>매개변수<input aria-label="매개변수" value={draft.parameters} onChange={event=>setDraft(current=>({...current,parameters:event.target.value}))} placeholder="매출, 원가"/></label>
        <label>수식<textarea aria-label="수식 본문" value={draft.body} rows={3} onChange={event=>setDraft(current=>({...current,body:event.target.value}))} placeholder="(매출-원가)/매출"/></label>
        <label>설명<input aria-label="수식 설명" value={draft.description} maxLength={500} onChange={event=>setDraft(current=>({...current,description:event.target.value}))} placeholder="어떤 셈인지 한 줄로"/></label>
        <p className="named-function-preview">쓰는 법: <code>{example}</code></p>
        <div className="modal-actions named-function-actions">
          {selected&&<button className="danger" disabled={saving} onClick={()=>remove(selected)}>삭제</button>}
          <span/>
          <button className="secondary" onClick={onClose}>닫기</button>
          <button className="primary" disabled={saving||!validName(draft.name)||!validParameters(draft.parameters)||!draft.body.trim()} onClick={save}>{saving?'저장 중…':'저장'}</button>
        </div>
      </section>
    </div>
  </div></div>
}
