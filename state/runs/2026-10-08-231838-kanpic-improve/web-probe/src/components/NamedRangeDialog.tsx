import { Braces,CornerDownRight,Plus } from 'lucide-react'
import { useState } from 'react'
import { address } from '../lib/api'
import type { MergeRange } from '../lib/merge'
import type { NamedRange,Sheet } from '../types'
import './NamedRangeDialog.css'
import { useDialog } from '../lib/useDialog'

type Draft={name:string;sheetId:string;range:string}
const rangeText=(range:MergeRange)=>`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`
const cellPosition=(value:string):[number,number]|undefined=>{const match=/^\$?([A-Za-z]+)\$?([1-9]\d*)$/.exec(value.trim());if(!match)return;let column=0;for(const letter of match[1].toUpperCase())column=column*26+letter.charCodeAt(0)-64;const row=Number(match[2]);if(column>16384||row>1048576)return;return[row,column]}
const validTarget=(value:string)=>{const parts=value.split(':');if(parts.length<1||parts.length>2)return false;const start=cellPosition(parts[0]),end=cellPosition(parts[1]??parts[0]);return !!start&&!!end&&start[0]<=end[0]&&start[1]<=end[1]}
const validName=(value:string)=>{const name=value.trim();return /^[\p{L}_][\p{L}\p{N}_.]*$/u.test(name)&&!cellPosition(name)&&!/^(true|false)$/i.test(name)}

export function NamedRangeDialog({selection,activeSheetId,sheets,ranges,onClose,onCreate,onUpdate,onDelete,onNavigate}:{
  selection:MergeRange;activeSheetId:string;sheets:Sheet[];ranges:NamedRange[];onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<NamedRange>;onUpdate:(id:string,input:Record<string,unknown>)=>Promise<NamedRange>;onDelete:(item:NamedRange)=>Promise<void>;onNavigate:(item:Pick<NamedRange,'sheet_id'|'range'>)=>void
}){
  const initial=():Draft=>({name:'',sheetId:activeSheetId,range:rangeText(selection)})
  const [selectedId,setSelectedId]=useState<string>(),[draft,setDraft]=useState<Draft>(initial),[saving,setSaving]=useState(false)
  const selected=ranges.find(item=>item.id===selectedId)
  const choose=(item?:NamedRange)=>{setSelectedId(item?.id);setDraft(item?{name:item.name,sheetId:item.sheet_id,range:item.range}:initial())}
  const save=async()=>{if(!validName(draft.name))return alert('이름은 문자 또는 밑줄로 시작하고 문자, 숫자, 밑줄, 마침표만 사용할 수 있습니다.');if(!validTarget(draft.range))return alert('올바른 A1 셀 또는 범위를 입력하세요.');setSaving(true);try{const input={name:draft.name.trim(),sheet_id:draft.sheetId,range:draft.range.toUpperCase()};const saved=selected?await onUpdate(selected.id,{...input,expected_revision:selected.revision}):await onCreate(input);choose(saved)}catch(error){alert(error instanceof Error?error.message:'이름 범위를 저장하지 못했습니다.')}finally{setSaving(false)}}
  const remove=async(item:NamedRange)=>{if(!confirm(`${item.name} 이름 범위를 삭제할까요?`))return;setSaving(true);try{await onDelete(item);choose()}catch(error){alert(error instanceof Error?error.message:'이름 범위를 삭제하지 못했습니다.')}finally{setSaving(false)}}
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop"><div className="modal named-range-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="이름 범위"><header><div><Braces/><div><h2>이름 범위</h2><p>셀 범위에 기억하기 쉬운 이름을 붙여 수식과 API에서 재사용합니다.</p></div></div><button aria-label="이름 범위 닫기" onClick={onClose}>×</button></header><div className="named-range-layout"><aside><button className={!selected?'active':''} onClick={()=>choose()}><Plus/> 새 이름 범위</button>{ranges.map(item=><button key={item.id} className={selected?.id===item.id?'active':''} onClick={()=>choose(item)}><span>{item.name}</span><em>{sheets.find(sheet=>sheet.id===item.sheet_id)?.name??'삭제된 시트'}!{item.range} · r{item.revision}</em></button>)}</aside><section><label>이름<input aria-label="이름 범위 이름" value={draft.name} maxLength={255} onChange={event=>setDraft(current=>({...current,name:event.target.value}))} placeholder="Quarter_Sales"/></label><div className="named-range-target"><label>시트<select aria-label="이름 범위 시트" value={draft.sheetId} onChange={event=>setDraft(current=>({...current,sheetId:event.target.value}))}>{sheets.map(sheet=><option key={sheet.id} value={sheet.id}>{sheet.name}</option>)}</select></label><label>범위<input aria-label="이름 범위 대상" value={draft.range} onChange={event=>setDraft(current=>({...current,range:event.target.value}))} placeholder="A1:B20"/></label></div><p className="named-range-preview">수식 예: <code>=SUM({draft.name||'Quarter_Sales'})</code></p><div className="modal-actions named-range-actions">{selected&&<><button className="secondary" onClick={()=>onNavigate(selected)}><CornerDownRight/> 범위로 이동</button><button className="danger" disabled={saving} onClick={()=>remove(selected)}>삭제</button></>}<span/><button className="secondary" onClick={onClose}>닫기</button><button className="primary" disabled={saving||!validName(draft.name)||!validTarget(draft.range)} onClick={save}>{saving?'저장 중…':'저장'}</button></div></section></div></div></div>
}
