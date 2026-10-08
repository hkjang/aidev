import { BadgeCheck,Plus,Trash2 } from 'lucide-react'
import { useState } from 'react'
import { address } from '../lib/api'
import { parseFilterInput,parseFilterRange } from '../lib/filter'
import { comparisonNeedsSecond,validationDraftValue,validationOptionInput } from '../lib/validation'
import type { MergeRange } from '../lib/merge'
import type { DataValidation,ValidationEvaluation,ValidationOperator,ValidationRuleType } from '../types'
import { useDialog } from '../lib/useDialog'

type DraftOption={value:string;label:string;color:string}
type Draft={range:string;ruleType:ValidationRuleType;operator:ValidationOperator;options:DraftOption[];sourceRange:string;value:string;value2:string;formula:string;allowBlank:boolean;rejectInput:boolean;showDropdown:boolean;displayStyle:'chip'|'arrow'|'plain';helpText:string}

const comparisons:Array<[ValidationOperator,string]>=[['between','사이'],['not_between','사이 아님'],['equal','같음'],['not_equal','같지 않음'],['greater_than','보다 큼'],['greater_or_equal','이상'],['less_than','보다 작음'],['less_or_equal','이하']]
const types:Array<[ValidationRuleType,string]>=[['list','드롭다운 목록'],['list_range','드롭다운(범위에서)'],['checkbox','체크박스'],['number','숫자'],['date','날짜'],['custom_formula','사용자 지정 수식']]
const initialOption=():DraftOption=>({value:'',label:'',color:'#dcfce7'})
const defaultDraft=(range:MergeRange):Draft=>({range:`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`,ruleType:'list',operator:'in_list',options:[initialOption()],sourceRange:'',value:'',value2:'',formula:'=',allowBlank:true,rejectInput:true,showDropdown:true,displayStyle:'chip',helpText:''})
const draftFromRule=(rule:DataValidation):Draft=>({range:rule.range,ruleType:rule.rule_type,operator:rule.operator,sourceRange:rule.source_range??'',options:(rule.options??[]).map(option=>({value:validationDraftValue(option.value),label:option.label??'',color:option.color??'#dcfce7'})),value:validationDraftValue(rule.value),value2:validationDraftValue(rule.value2),formula:rule.formula??'=',allowBlank:rule.allow_blank,rejectInput:rule.reject_input,showDropdown:rule.show_dropdown,displayStyle:rule.display_style,helpText:rule.help_text??''})
const typeLabel=(type:ValidationRuleType)=>types.find(([value])=>value===type)?.[1]??type

export function DataValidationDialog({range,rules,onClose,onCreate,onUpdate,onDelete,onEvaluate}:{
  range:MergeRange;rules:DataValidation[];onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<DataValidation>;onUpdate:(id:string,input:Record<string,unknown>)=>Promise<DataValidation>;onDelete:(rule:DataValidation)=>Promise<void>;onEvaluate:(id:string)=>Promise<ValidationEvaluation>
}){
  const [selectedId,setSelectedId]=useState<string>(),[draft,setDraft]=useState<Draft>(()=>defaultDraft(range)),[saving,setSaving]=useState(false),[evaluation,setEvaluation]=useState<ValidationEvaluation>()
  const selected=rules.find(rule=>rule.id===selectedId)
  const choose=(rule?:DataValidation)=>{setSelectedId(rule?.id);setDraft(rule?draftFromRule(rule):defaultDraft(range));setEvaluation(undefined)}
  const patchOption=(index:number,patch:Partial<DraftOption>)=>setDraft(current=>({...current,options:current.options.map((option,optionIndex)=>optionIndex===index?{...option,...patch}:option)}))
  const changeType=(ruleType:ValidationRuleType)=>setDraft(current=>({...current,ruleType,
    operator:ruleType==='list'||ruleType==='list_range'||ruleType==='checkbox'?'in_list':ruleType==='custom_formula'?'custom':'between',
    showDropdown:ruleType==='list'||ruleType==='list_range'?current.showDropdown:false,
    displayStyle:ruleType==='list'||ruleType==='list_range'?current.displayStyle:'plain',
    // A checkbox is the two values it toggles between, seeded with TRUE/FALSE.
    options:ruleType==='checkbox'&&current.options.length!==2?[{value:'TRUE',label:'',color:'#dcfce7'},{value:'FALSE',label:'',color:'#fee2e2'}]:current.options}))
  const save=async()=>{
    if(!parseFilterRange(draft.range))return alert('올바른 A1 범위를 입력하세요.')
    if(draft.ruleType==='list'&&(draft.options.length===0||draft.options.some(option=>option.value.trim()==='')))return alert('드롭다운 값을 하나 이상 입력하세요.')
    if(draft.ruleType==='checkbox'&&draft.options.some(option=>option.value.trim()===''))return alert('체크 상태와 해제 상태의 값을 모두 입력하세요.')
    if(draft.ruleType==='list_range'&&!parseFilterRange(draft.sourceRange.replace(/^.*!/,'')))return alert('목록을 읽어올 범위를 입력하세요. 예: 코드!A2:A50')
    const input:Record<string,unknown>={range:draft.range.toUpperCase(),rule_type:draft.ruleType,operator:draft.operator,allow_blank:draft.allowBlank,reject_input:draft.rejectInput,show_dropdown:(draft.ruleType==='list'||draft.ruleType==='list_range')&&draft.showDropdown,display_style:draft.ruleType==='list'||draft.ruleType==='list_range'?draft.displayStyle:'plain',help_text:draft.helpText}
    if(draft.ruleType==='list_range')input.source_range=draft.sourceRange.trim().toUpperCase()
    else if(draft.ruleType==='list'||draft.ruleType==='checkbox')input.options=draft.options.slice(0,draft.ruleType==='checkbox'?2:undefined).map(option=>validationOptionInput(option.value,option.label,option.color))
    else if(draft.ruleType==='number'){input.value=parseFilterInput(draft.value);if(comparisonNeedsSecond(draft.operator))input.value2=parseFilterInput(draft.value2)}
    else if(draft.ruleType==='date'){input.value=draft.value;if(comparisonNeedsSecond(draft.operator))input.value2=draft.value2}
    else input.formula=draft.formula
    setSaving(true);try{const saved=selected?await onUpdate(selected.id,{...input,expected_revision:selected.revision}):await onCreate(input);choose(saved)}catch(error){alert(error instanceof Error?error.message:'검증 규칙을 저장하지 못했습니다.')}finally{setSaving(false)}
  }
  const remove=async(rule:DataValidation)=>{if(!confirm(`${rule.range} 검증 규칙을 삭제할까요?`))return;setSaving(true);try{await onDelete(rule);choose()}catch(error){alert(error instanceof Error?error.message:'검증 규칙을 삭제하지 못했습니다.')}finally{setSaving(false)}}
  const evaluate=async(rule:DataValidation)=>{setSaving(true);try{setEvaluation(await onEvaluate(rule.id))}catch(error){alert(error instanceof Error?error.message:'기존 데이터를 검사하지 못했습니다.')}finally{setSaving(false)}}
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop"><div className="modal validation-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="데이터 검증"><header><div><BadgeCheck/><div><h2>데이터 검증</h2><p>범위 입력을 서버에서 검사하고 목록은 컬러 드롭다운으로 표시합니다.</p></div></div><button aria-label="데이터 검증 닫기" onClick={onClose}>×</button></header><div className="validation-layout"><aside><button className={!selected?'active':''} onClick={()=>choose()}><Plus/> 새 검증 규칙</button>{rules.map(rule=><button key={rule.id} className={selected?.id===rule.id?'active':''} onClick={()=>choose(rule)}><span>{rule.range}</span><em>{typeLabel(rule.rule_type)} · r{rule.revision}</em></button>)}</aside><section><div className="validation-fields"><label>적용 범위<input aria-label="데이터 검증 범위" value={draft.range} onChange={event=>setDraft(current=>({...current,range:event.target.value}))}/></label><label>검증 유형<select aria-label="검증 유형" value={draft.ruleType} onChange={event=>changeType(event.target.value as ValidationRuleType)}>{types.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label></div>
    {draft.ruleType==='checkbox'&&<div className="validation-fields">
      <label>체크했을 때 저장할 값<input aria-label="체크 값" value={draft.options[0]?.value??''} onChange={event=>patchOption(0,{value:event.target.value})}/></label>
      <label>해제했을 때 저장할 값<input aria-label="해제 값" value={draft.options[1]?.value??''} onChange={event=>patchOption(1,{value:event.target.value})}/></label>
    </div>}
    {draft.ruleType==='list_range'&&<div className="validation-fields"><label>목록 범위<input aria-label="드롭다운 목록 범위" value={draft.sourceRange} onChange={event=>setDraft(current=>({...current,sourceRange:event.target.value}))} placeholder="예: 코드!A2:A50"/><small>이 범위의 값이 그대로 드롭다운 항목이 됩니다. 목록을 고치면 이 규칙을 쓰는 모든 셀에 바로 반영됩니다.</small></label></div>}
    {draft.ruleType==='list'&&<><div className="validation-list-head"><strong>드롭다운 항목</strong><button className="secondary" disabled={draft.options.length>=500} onClick={()=>setDraft(current=>({...current,options:[...current.options,initialOption()]}))}><Plus/> 항목 추가</button></div><div className="validation-options">{draft.options.map((option,index)=><div className="validation-option" key={index}><input aria-label={`목록 항목 ${index+1} 색상`} type="color" value={option.color} onChange={event=>patchOption(index,{color:event.target.value})}/><input aria-label={`목록 항목 ${index+1} 값`} placeholder="저장 값" value={option.value} onChange={event=>patchOption(index,{value:event.target.value})}/><input aria-label={`목록 항목 ${index+1} 라벨`} placeholder="표시 라벨 (선택)" value={option.label} onChange={event=>patchOption(index,{label:event.target.value})}/><button aria-label={`목록 항목 ${index+1} 삭제`} disabled={draft.options.length===1} onClick={()=>setDraft(current=>({...current,options:current.options.filter((_,optionIndex)=>optionIndex!==index)}))}><Trash2/></button></div>)}</div><div className="validation-fields"><label>표시 방식<select aria-label="드롭다운 표시 방식" value={draft.displayStyle} onChange={event=>setDraft(current=>({...current,displayStyle:event.target.value as Draft['displayStyle']}))}><option value="chip">컬러 칩</option><option value="arrow">값과 화살표</option><option value="plain">일반 텍스트</option></select></label></div></>}
    {(draft.ruleType==='number'||draft.ruleType==='date')&&<div className="validation-fields three"><label>조건<select aria-label="검증 조건" value={draft.operator} onChange={event=>setDraft(current=>({...current,operator:event.target.value as ValidationOperator}))}>{comparisons.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label><label>기준값<input aria-label="검증 기준값" type={draft.ruleType==='date'?'date':'number'} value={draft.value} onChange={event=>setDraft(current=>({...current,value:event.target.value}))}/></label>{comparisonNeedsSecond(draft.operator)&&<label>두 번째 기준값<input aria-label="검증 두 번째 기준값" type={draft.ruleType==='date'?'date':'number'} value={draft.value2} onChange={event=>setDraft(current=>({...current,value2:event.target.value}))}/></label>}</div>}
    {draft.ruleType==='custom_formula'&&<label className="validation-formula">사용자 지정 수식<input aria-label="검증 사용자 지정 수식" value={draft.formula} onChange={event=>setDraft(current=>({...current,formula:event.target.value}))} placeholder="=A1>0"/><small>범위의 왼쪽 위 셀을 기준으로 상대 참조가 이동합니다.</small></label>}
    <label className="validation-help">도움말<input aria-label="검증 도움말" value={draft.helpText} maxLength={500} onChange={event=>setDraft(current=>({...current,helpText:event.target.value}))} placeholder="잘못된 입력일 때 보여줄 안내"/></label><div className="validation-toggles"><label><input type="checkbox" checked={draft.allowBlank} onChange={event=>setDraft(current=>({...current,allowBlank:event.target.checked}))}/> 빈 값 허용</label><label><input aria-label="잘못된 입력 거부" type="checkbox" checked={draft.rejectInput} onChange={event=>setDraft(current=>({...current,rejectInput:event.target.checked}))}/> 잘못된 입력 거부</label>{draft.ruleType==='list'&&<label><input aria-label="드롭다운 표시" type="checkbox" checked={draft.showDropdown} onChange={event=>setDraft(current=>({...current,showDropdown:event.target.checked}))}/> 셀에 드롭다운 표시</label>}</div>
    {evaluation&&<div className={`validation-result ${evaluation.invalid_cells.length?'invalid':'valid'}`}>검사 {evaluation.checked_cells.toLocaleString()}셀 · 정상 {evaluation.valid_cells.toLocaleString()}셀 · 오류 {(evaluation.checked_cells-evaluation.valid_cells).toLocaleString()}셀{evaluation.truncated?' (오류 목록 일부만 표시)':''}</div>}
    <div className="modal-actions validation-actions">{selected&&<><button className="secondary" disabled={saving} onClick={()=>evaluate(selected)}>기존 데이터 검사</button><button className="danger" disabled={saving} onClick={()=>remove(selected)}>삭제</button></>}<span/><button className="secondary" onClick={onClose}>닫기</button><button className="primary" disabled={saving||!parseFilterRange(draft.range)} onClick={save}>{saving?'저장 중…':'규칙 저장'}</button></div></section></div></div></div>
}
