import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { formulaContext, suggestFunctions, type FormulaContext, type FunctionDoc } from '../lib/formulaSuggest'

/**
 * The function catalog, cached for the session. It is the same list the
 * function dialog shows, so what is suggested is always what evaluates.
 */
export function useFunctionCatalog(extra:FunctionDoc[]=[]){
  const query=useQuery({queryKey:['formula-functions'],queryFn:()=>api<{items:FunctionDoc[]}>('/api/v1/formula/functions'),staleTime:60*60*1000})
  const builtIn=query.data?.items??[]
  // 워크북에 저장해 둔 이름 있는 수식도 함께 권한다. 만들어 두고 아무도
  // 모르면 없는 것과 같다.
  return extra.length?[...builtIn,...extra]:builtIn
}

/** 저장해 둔 이름 있는 수식을 자동완성이 읽는 꼴로 바꾼다. */
export function namedFunctionDocs(items:Array<{name:string;parameters:string[];body:string;description?:string}>):FunctionDoc[]{
  return items.map(item=>({
    name:item.name,
    category:'이름 있는 수식',
    syntax:`${item.name}(${item.parameters.join(', ')})`,
    summary:item.description||item.body,
  }))
}

export type FormulaHint={context:FormulaContext;matches:FunctionDoc[];signature?:FunctionDoc}

/** Works out what to show under the cell being edited, if anything. */
export function formulaHint(functions:FunctionDoc[],text:string,caret:number):FormulaHint|undefined{
  const context=formulaContext(text,caret)
  if(!context)return undefined
  const matches=suggestFunctions(functions,context.token)
  const signature=context.call&&functions.find(item=>item.name===context.call?.name)
  if(matches.length===0&&!signature)return undefined
  return {context,matches,signature:signature||undefined}
}

/**
 * The suggestion list and signature hint Google Sheets users expect while
 * typing a formula. Selection is driven by the editor's key handling, so this
 * component only draws.
 */
export function FormulaAutocomplete({hint,active,left,top,onChoose}:{
  hint:FormulaHint
  active:number
  left:number
  top:number
  onChoose:(name:string)=>void
}){
  const argument=hint.context.call?.argument??0
  return <div className="formula-suggest" style={{left,top}} role="listbox" aria-label="함수 제안">
    {hint.matches.map((item,index)=><button key={item.name} role="option" aria-selected={index===active} className={index===active?'active':undefined}
      // The editor must keep focus, so the press is handled before blur.
      onMouseDown={event=>{event.preventDefault();onChoose(item.name)}}>
      <strong>{item.name}</strong><code>{item.syntax}</code><span>{item.summary}</span>
    </button>)}
    {hint.matches.length===0&&hint.signature&&<div className="formula-signature">
      <code>{highlightArgument(hint.signature.syntax,argument)}</code><span>{hint.signature.summary}</span>
    </div>}
  </div>
}

/** Marks the argument the caret is on inside the printed syntax. */
function highlightArgument(syntax:string,argument:number){
  const open=syntax.indexOf('(')
  if(open<0)return syntax
  const parts=syntax.slice(open+1,syntax.lastIndexOf(')')).split(', ')
  // `SUM(값1, 값2, …)` accepts more arguments than it names. Without this the
  // hint goes blank on the fourth value, which is when it is needed most.
  if(argument>=parts.length&&parts[parts.length-1]==='…')argument=parts.length-1
  return <>
    {syntax.slice(0,open+1)}
    {parts.map((part,index)=><span key={index}>
      <span className={index===argument?'current':undefined}>{part}</span>{index<parts.length-1?', ':''}
    </span>)}
    {')'}
  </>
}

/**
 * 표 이름을 자동완성이 읽는 꼴로 바꾼다. 표를 만들어 두고 이름을 아무도
 * 모르면 없는 것과 같다. 열 이름은 머리글에서 오므로 여기서는 알 수 없어,
 * 대괄호까지만 권하고 그 안은 사람이 적는다.
 */
export function sheetTableDocs(items:Array<{name:string;range:string;header_row:boolean;totals_row:boolean}>):FunctionDoc[]{
  return items.map(item=>({
    name:item.name,
    category:'표',
    syntax:`${item.name}[열 이름]`,
    summary:`${item.range} 의 자료 줄입니다. ${item.name} 은 표 전체, ${item.name}[#전체] 는 머리글까지${item.totals_row?`, ${item.name}[#합계] 는 합계 줄입니다`:'입니다'}.`,
  }))
}
