import { Palette,Plus,Trash2 } from 'lucide-react'
import { useState } from 'react'
import { address } from '../lib/api'
import { parseFilterInput,parseFilterRange } from '../lib/filter'
import type { MergeRange } from '../lib/merge'
import type { ConditionalFormat,ConditionalFormatOperator,ConditionalFormatRuleType,ConditionalIconStyle } from '../types'
import { ICON_STYLES,ICON_STYLE_LABELS,drawConditionalIcon,iconSetPreview } from '../lib/conditionalIcon'
import './ConditionalFormatDialog.css'
import { useDialog } from '../lib/useDialog'

type Draft={name:string;range:string;ruleType:ConditionalFormatRuleType;operator:ConditionalFormatOperator;formula:string;value:string;value2:string;background:string;color:string;bold:boolean;italic:boolean;minColor:string;midColor:string;maxColor:string;useMidColor:boolean;barColor:string;iconStyle:ConditionalIconStyle;iconReverse:boolean;priority:number;stopIfTrue:boolean}

const types:Array<[ConditionalFormatRuleType,string]>=[['value','값 조건'],['custom_formula','맞춤 수식'],['duplicate','중복·고유 값'],['rank','상위·하위 N개'],['color_scale','색상 범위'],['data_bar','데이터 막대'],['icon_set','아이콘 집합']]
const rankOperators:Array<[string,string]>=[['top','상위 N개'],['bottom','하위 N개'],['top_percent','상위 N%'],['bottom_percent','하위 N%']]
const valueOperators:Array<[ConditionalFormatOperator,string]>=[['greater_than','보다 큼'],['greater_or_equal','이상'],['less_than','보다 작음'],['less_or_equal','이하'],['equals','같음'],['not_equals','같지 않음'],['between','사이'],['not_between','사이 아님'],['contains','텍스트 포함'],['not_contains','텍스트 미포함'],['is_blank','비어 있음'],['not_blank','비어 있지 않음']]
const needsValue=(operator:ConditionalFormatOperator)=>operator!=='is_blank'&&operator!=='not_blank'
const needsSecond=(operator:ConditionalFormatOperator)=>operator==='between'||operator==='not_between'
const ruleTypeLabel=(type:ConditionalFormatRuleType)=>types.find(([value])=>value===type)?.[1]??type
const draftValue=(value:unknown)=>value==null?'':String(value)
const defaultDraft=(range:MergeRange):Draft=>({name:'',range:`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`,ruleType:'value',operator:'greater_than',formula:'',value:'',value2:'',background:'#fee2e2',color:'#991b1b',bold:false,italic:false,minColor:'#dcfce7',midColor:'#fef3c7',maxColor:'#ef4444',useMidColor:false,barColor:'#38a3a5',iconStyle:'3TrafficLights1',iconReverse:false,priority:1,stopIfTrue:false})
const draftFromRule=(rule:ConditionalFormat):Draft=>({name:rule.name,range:rule.range,ruleType:rule.rule_type,operator:rule.operator??(rule.rule_type==='duplicate'?'duplicate':'greater_than'),formula:rule.formula??'',value:draftValue(rule.value),value2:draftValue(rule.value2),background:typeof rule.style?.background==='string'?rule.style.background:'#fee2e2',color:typeof rule.style?.color==='string'?rule.style.color:'#991b1b',bold:rule.style?.bold===true,italic:rule.style?.italic===true,minColor:rule.min_color??'#dcfce7',midColor:rule.mid_color??'#fef3c7',maxColor:rule.max_color??'#ef4444',useMidColor:Boolean(rule.mid_color),barColor:rule.bar_color??'#38a3a5',iconStyle:rule.icon_style??'3TrafficLights1',iconReverse:rule.icon_reverse===true,priority:rule.priority,stopIfTrue:rule.stop_if_true})

// 미리보기는 격자와 같은 그리기 코드를 쓴다. 따로 그리면 언젠가 서로
// 다른 그림이 된다.
function IconSetPreview({style,reverse}:{style:ConditionalIconStyle;reverse:boolean}){
  const size=18,gap=8
  const draw=(canvas:HTMLCanvasElement|null)=>{
    if(!canvas)return
    const glyphs=reverse?[...iconSetPreview(style)].reverse():iconSetPreview(style)
    const ratio=window.devicePixelRatio||1
    canvas.width=(size+gap)*glyphs.length*ratio;canvas.height=(size+4)*ratio
    canvas.style.width=`${(size+gap)*glyphs.length}px`;canvas.style.height=`${size+4}px`
    const context=canvas.getContext('2d')
    if(!context)return
    context.scale(ratio,ratio)
    glyphs.forEach((glyph,index)=>drawConditionalIcon(context,glyph,index*(size+gap)+gap/2,2,size))
  }
  return <canvas className="conditional-icon-preview" aria-label={`${ICON_STYLE_LABELS[style]} 미리보기`} ref={draw}/>
}

export function ConditionalFormatDialog({range,rules,onClose,onCreate,onUpdate,onDelete}:{
  range:MergeRange;rules:ConditionalFormat[];onClose:()=>void
  onCreate:(input:Record<string,unknown>)=>Promise<ConditionalFormat>;onUpdate:(id:string,input:Record<string,unknown>)=>Promise<ConditionalFormat>;onDelete:(rule:ConditionalFormat)=>Promise<void>
}){
  const [selectedId,setSelectedId]=useState<string>(),[draft,setDraft]=useState<Draft>(()=>defaultDraft(range)),[saving,setSaving]=useState(false)
  const selected=rules.find(rule=>rule.id===selectedId)
  const choose=(rule?:ConditionalFormat)=>{setSelectedId(rule?.id);setDraft(rule?draftFromRule(rule):defaultDraft(range))}
  const changeType=(ruleType:ConditionalFormatRuleType)=>setDraft(current=>({...current,ruleType,operator:ruleType==='duplicate'?'duplicate':ruleType==='rank'?'top':ruleType==='value'?'greater_than':current.operator,value:ruleType==='rank'?'10':current.value,stopIfTrue:ruleType==='value'||ruleType==='duplicate'||ruleType==='rank'||ruleType==='custom_formula'?current.stopIfTrue:false}))
  const save=async()=>{
    if(!parseFilterRange(draft.range))return alert('올바른 A1 범위를 입력하세요.')
    if(draft.ruleType==='value'&&needsValue(draft.operator)&&draft.value.trim()==='')return alert('조건 기준값을 입력하세요.')
    if(draft.ruleType==='value'&&needsSecond(draft.operator)&&draft.value2.trim()==='')return alert('두 번째 기준값을 입력하세요.')
    if(draft.ruleType==='value'&&needsSecond(draft.operator)&&(!Number.isFinite(Number(draft.value))||!Number.isFinite(Number(draft.value2))))return alert('사이 조건의 기준값은 숫자로 입력하세요.')
    const input:Record<string,unknown>={name:draft.name.trim(),range:draft.range.toUpperCase(),rule_type:draft.ruleType,priority:draft.priority,stop_if_true:draft.ruleType==='value'||draft.ruleType==='duplicate'||draft.ruleType==='rank'||draft.ruleType==='custom_formula'?draft.stopIfTrue:false}
    if(draft.ruleType==='value'){
      input.operator=draft.operator
      if(needsValue(draft.operator))input.value=needsSecond(draft.operator)?Number(draft.value):parseFilterInput(draft.value)
      if(needsSecond(draft.operator))input.value2=Number(draft.value2)
      input.style={background:draft.background,color:draft.color,bold:draft.bold,italic:draft.italic}
    }else if(draft.ruleType==='custom_formula'){
      if(!draft.formula.trim().startsWith('='))return alert('맞춤 수식은 =로 시작해야 합니다. 예: =$C1="완료"')
      input.formula=draft.formula.trim()
      input.style={background:draft.background,color:draft.color,bold:draft.bold,italic:draft.italic}
    }else if(draft.ruleType==='duplicate'){
      input.operator=draft.operator
      input.style={background:draft.background,color:draft.color,bold:draft.bold,italic:draft.italic}
    }else if(draft.ruleType==='rank'){
      input.operator=draft.operator
      input.value=Number(draft.value)
      input.style={background:draft.background,color:draft.color,bold:draft.bold,italic:draft.italic}
    }else if(draft.ruleType==='color_scale'){
      input.min_color=draft.minColor;input.mid_color=draft.useMidColor?draft.midColor:'';input.max_color=draft.maxColor
    }else if(draft.ruleType==='icon_set'){
      input.icon_style=draft.iconStyle;input.icon_reverse=draft.iconReverse
    }else input.bar_color=draft.barColor
    setSaving(true)
    try{const saved=selected?await onUpdate(selected.id,{...input,expected_revision:selected.revision}):await onCreate(input);choose(saved)}catch(error){alert(error instanceof Error?error.message:'조건부 서식 규칙을 저장하지 못했습니다.')}finally{setSaving(false)}
  }
  const remove=async(rule:ConditionalFormat)=>{if(!confirm(`${rule.range} 조건부 서식을 삭제할까요?`))return;setSaving(true);try{await onDelete(rule);choose()}catch(error){alert(error instanceof Error?error.message:'조건부 서식을 삭제하지 못했습니다.')}finally{setSaving(false)}}
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop"><div className="modal validation-modal conditional-format-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="조건부 서식"><header><div><Palette/><div><h2>조건부 서식</h2><p>값 비교, 맞춤 수식, 중복, 상위·하위 N개, 색상 범위, 데이터 막대와 아이콘 집합을 서버 권위 규칙으로 표시합니다.</p></div></div><button aria-label="조건부 서식 닫기" onClick={onClose}>×</button></header><div className="validation-layout"><aside><button className={!selected?'active':''} onClick={()=>choose()}><Plus/> 새 서식 규칙</button>{rules.map(rule=><button key={rule.id} className={selected?.id===rule.id?'active':''} onClick={()=>choose(rule)}><span>{rule.name||rule.range}</span><em>{rule.range} · {ruleTypeLabel(rule.rule_type)} · 우선순위 {rule.priority}</em></button>)}</aside><section><div className="validation-fields three"><label>규칙 이름<input aria-label="조건부 서식 이름" maxLength={200} value={draft.name} onChange={event=>setDraft(current=>({...current,name:event.target.value}))} placeholder="예: 목표 초과"/></label><label>적용 범위<input aria-label="조건부 서식 범위" value={draft.range} onChange={event=>setDraft(current=>({...current,range:event.target.value}))}/></label><label>규칙 유형<select aria-label="조건부 서식 유형" value={draft.ruleType} onChange={event=>changeType(event.target.value as ConditionalFormatRuleType)}>{types.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label></div>
    {draft.ruleType==='value'&&<div className="validation-fields three"><label>조건<select aria-label="조건부 서식 조건" value={draft.operator} onChange={event=>setDraft(current=>({...current,operator:event.target.value as ConditionalFormatOperator}))}>{valueOperators.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label>{needsValue(draft.operator)&&<label>기준값<input aria-label="조건부 서식 기준값" value={draft.value} onChange={event=>setDraft(current=>({...current,value:event.target.value}))}/></label>}{needsSecond(draft.operator)&&<label>두 번째 기준값<input aria-label="조건부 서식 두 번째 기준값" type="number" value={draft.value2} onChange={event=>setDraft(current=>({...current,value2:event.target.value}))}/></label>}</div>}
    {draft.ruleType==='custom_formula'&&<div className="validation-fields"><label>맞춤 수식<input aria-label="조건부 서식 맞춤 수식" value={draft.formula} onChange={event=>setDraft(current=>({...current,formula:event.target.value}))} placeholder={'예: =$C1="완료"'}/><small>범위의 첫 셀 기준으로 쓰면 나머지 셀에서는 상대 참조가 함께 움직입니다. 열을 고정하려면 <code>$C1</code> 처럼 적습니다.</small></label></div>}
    {draft.ruleType==='rank'&&<div className="validation-fields three"><label>순위 기준<select aria-label="순위 기준" value={draft.operator} onChange={event=>setDraft(current=>({...current,operator:event.target.value as ConditionalFormatOperator}))}>{rankOperators.map(([value,label])=><option value={value} key={value}>{label}</option>)}</select></label><label>개수{draft.operator.endsWith('_percent')?'(%)':''}<input aria-label="순위 개수" type="number" min={1} max={draft.operator.endsWith('_percent')?100:1000} value={draft.value} onChange={event=>setDraft(current=>({...current,value:event.target.value}))}/><small>같은 값이 문턱에 걸리면 함께 표시됩니다. 상위 3개를 물었는데 3등이 둘이면 둘 다 나옵니다.</small></label></div>}
    {draft.ruleType==='duplicate'&&<div className="validation-fields"><label>표시할 값<select aria-label="중복 값 조건" value={draft.operator} onChange={event=>setDraft(current=>({...current,operator:event.target.value as ConditionalFormatOperator}))}><option value="duplicate">중복 값</option><option value="unique">고유 값</option></select></label></div>}
    {(draft.ruleType==='value'||draft.ruleType==='duplicate'||draft.ruleType==='rank'||draft.ruleType==='custom_formula')&&<fieldset className="conditional-style"><legend>적용할 셀 서식</legend><label>배경색<input aria-label="조건부 배경색" type="color" value={draft.background} onChange={event=>setDraft(current=>({...current,background:event.target.value}))}/></label><label>글자색<input aria-label="조건부 글자색" type="color" value={draft.color} onChange={event=>setDraft(current=>({...current,color:event.target.value}))}/></label><label className="conditional-check"><input aria-label="조건부 굵게" type="checkbox" checked={draft.bold} onChange={event=>setDraft(current=>({...current,bold:event.target.checked}))}/> 굵게</label><label className="conditional-check"><input aria-label="조건부 기울임" type="checkbox" checked={draft.italic} onChange={event=>setDraft(current=>({...current,italic:event.target.checked}))}/> 기울임</label><span className="conditional-preview" style={{background:draft.background,color:draft.color,fontWeight:draft.bold?700:400,fontStyle:draft.italic?'italic':'normal'}}>미리보기 Aa 123</span></fieldset>}
    {draft.ruleType==='color_scale'&&<fieldset className="conditional-style color-scale"><legend>색상 범위</legend><label>최솟값 색상<input aria-label="최솟값 색상" type="color" value={draft.minColor} onChange={event=>setDraft(current=>({...current,minColor:event.target.value}))}/></label><label>최댓값 색상<input aria-label="최댓값 색상" type="color" value={draft.maxColor} onChange={event=>setDraft(current=>({...current,maxColor:event.target.value}))}/></label><label className="conditional-check"><input aria-label="중간 색상 사용" type="checkbox" checked={draft.useMidColor} onChange={event=>setDraft(current=>({...current,useMidColor:event.target.checked}))}/> 중간 색상 사용</label>{draft.useMidColor&&<label>중간값 색상<input aria-label="중간값 색상" type="color" value={draft.midColor} onChange={event=>setDraft(current=>({...current,midColor:event.target.value}))}/></label>}<span className="conditional-gradient" style={{background:`linear-gradient(90deg,${draft.minColor}${draft.useMidColor?`,${draft.midColor}`:''},${draft.maxColor})`}}/></fieldset>}
    {draft.ruleType==='icon_set'&&<fieldset className="conditional-style icon-set-style"><legend>아이콘 집합</legend><label>아이콘 종류<select aria-label="아이콘 종류" value={draft.iconStyle} onChange={event=>setDraft(current=>({...current,iconStyle:event.target.value as ConditionalIconStyle}))}>{ICON_STYLES.map(style=><option value={style} key={style}>{ICON_STYLE_LABELS[style]}</option>)}</select></label><label className="conditional-check"><input aria-label="아이콘 순서 뒤집기" type="checkbox" checked={draft.iconReverse} onChange={event=>setDraft(current=>({...current,iconReverse:event.target.checked}))}/> 순서 뒤집기(작은 값이 좋은 값)</label><IconSetPreview style={draft.iconStyle} reverse={draft.iconReverse}/><small>범위의 최솟값과 최댓값 사이를 엑셀과 같은 자리에서 나눕니다. 아이콘 3개는 33%·67%, 4개는 25%·50%·75%, 5개는 20%·40%·60%·80% 입니다.</small></fieldset>}
    {draft.ruleType==='data_bar'&&<fieldset className="conditional-style data-bar-style"><legend>데이터 막대</legend><label>막대 색상<input aria-label="데이터 막대 색상" type="color" value={draft.barColor} onChange={event=>setDraft(current=>({...current,barColor:event.target.value}))}/></label><span className="conditional-bar-preview"><i style={{background:draft.barColor}}/></span></fieldset>}
    <div className="validation-fields conditional-options"><label>우선순위<input aria-label="조건부 서식 우선순위" type="number" min={1} max={1000} value={draft.priority} onChange={event=>setDraft(current=>({...current,priority:Number(event.target.value)}))}/></label>{(draft.ruleType==='value'||draft.ruleType==='duplicate'||draft.ruleType==='custom_formula')&&<label className="conditional-check stop"><input aria-label="조건이 참이면 중지" type="checkbox" checked={draft.stopIfTrue} onChange={event=>setDraft(current=>({...current,stopIfTrue:event.target.checked}))}/> 조건이 참이면 낮은 우선순위 규칙 중지</label>}</div>
    <div className="modal-actions validation-actions">{selected&&<button className="danger" disabled={saving} onClick={()=>remove(selected)}><Trash2/> 삭제</button>}<span/><button className="secondary" onClick={onClose}>닫기</button><button className="primary" disabled={saving||!parseFilterRange(draft.range)||draft.priority<1||draft.priority>1000} onClick={save}>{saving?'저장 중…':'규칙 저장'}</button></div></section></div></div></div>
}
