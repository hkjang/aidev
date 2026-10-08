import { FileSpreadsheet, Replace, Search, X } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { api, newIdempotencyKey } from '../lib/api'
import { collaborationClientId } from '../lib/client'
import type { ReplaceResult, WorkbookSearchMatch, WorkbookSearchResult } from '../types'
import './WorkbookSearchDialog.css'

function valueLabel(value:unknown){
  if(value==null)return '빈 값'
  if(typeof value==='string')return value
  try{return JSON.stringify(value)}catch{return String(value)}
}

function matchLabel(item:WorkbookSearchMatch){
  if(item.matched_fields.includes('formula')&&item.formula)return item.formula
  return valueLabel(item.value)
}

export type SearchOptions={matchCase:boolean;wholeCell:boolean;useRegex:boolean;skipFormulas:boolean;currentSheetOnly:boolean}
const DEFAULT_OPTIONS:SearchOptions={matchCase:false,wholeCell:false,useRegex:false,skipFormulas:false,currentSheetOnly:false}

/** Only non-default options reach the query string so the default search keeps a stable URL. */
export function searchQuery(query:string,options:SearchOptions,sheetId:string|undefined,limit:number,offset?:number){
  const parameters=new URLSearchParams({q:query,limit:String(limit)})
  if(offset!=null)parameters.set('offset',String(offset))
  if(options.matchCase)parameters.set('match_case','true')
  if(options.wholeCell)parameters.set('whole_cell','true')
  if(options.useRegex)parameters.set('regex','true')
  if(options.skipFormulas)parameters.set('skip_formulas','true')
  if(options.currentSheetOnly&&sheetId)parameters.set('sheet_id',sheetId)
  return parameters.toString()
}

export function WorkbookSearchDialog({open,workbookId,version,sheetId,sheetName,replaceMode=false,onClose,onNavigate,onReplaced}:{open:boolean;workbookId:string;version:number;sheetId?:string;sheetName?:string;replaceMode?:boolean;onClose:()=>void;onNavigate:(item:WorkbookSearchMatch)=>void;onReplaced?:(result:ReplaceResult)=>void}){
  const [query,setQuery]=useState(''),[replacement,setReplacement]=useState(''),[showReplace,setShowReplace]=useState(replaceMode)
  const [options,setOptions]=useState<SearchOptions>(DEFAULT_OPTIONS)
  const [result,setResult]=useState<WorkbookSearchResult>(),[loading,setLoading]=useState(false),[loadingMore,setLoadingMore]=useState(false),[error,setError]=useState(''),[status,setStatus]=useState(''),[activeIndex,setActiveIndex]=useState(0),[replacing,setReplacing]=useState(false)
  const inputRef=useRef<HTMLInputElement>(null)
  const optionKey=useMemo(()=>JSON.stringify(options),[options])
  useEffect(()=>{if(open){setShowReplace(current=>current||replaceMode);const timer=window.setTimeout(()=>inputRef.current?.focus(),0);return()=>window.clearTimeout(timer)}setActiveIndex(0);setStatus('')},[open,replaceMode])
  useEffect(()=>{
    if(!open)return
    const normalized=options.useRegex?query:query.trim()
    if(!normalized){setResult(undefined);setError('');setLoading(false);return}
    const controller=new AbortController(),timer=window.setTimeout(()=>{
      setLoading(true);setError('')
      api<WorkbookSearchResult>(`/api/v1/workbooks/${workbookId}/search?${searchQuery(normalized,options,sheetId,50)}`,{signal:controller.signal})
        .then(value=>{setResult(value);setActiveIndex(0)})
        .catch(reason=>{if(!controller.signal.aborted)setError(reason instanceof Error?reason.message:'검색하지 못했습니다.')})
        .finally(()=>{if(!controller.signal.aborted)setLoading(false)})
    },250)
    return()=>{window.clearTimeout(timer);controller.abort()}
  },[open,query,workbookId,version,optionKey,sheetId])
  useEffect(()=>{if(!open)return;const close=(event:KeyboardEvent)=>{if(event.key==='Escape')onClose()};window.addEventListener('keydown',close);return()=>window.removeEventListener('keydown',close)},[open,onClose])
  if(!open)return null
  const items=result?.items??[]
  const choose=(item:WorkbookSearchMatch)=>{onNavigate(item);onClose()}
  const preview=(item:WorkbookSearchMatch)=>{onNavigate(item)}
  const loadMore=async()=>{if(result?.next_offset==null||loadingMore)return;setLoadingMore(true);setError('');try{const page=await api<WorkbookSearchResult>(`/api/v1/workbooks/${workbookId}/search?${searchQuery(result.query,options,sheetId,50,result.next_offset)}`);setResult(current=>current?.query===page.query?{...page,items:[...current.items,...page.items]}:current)}catch(reason){setError(reason instanceof Error?reason.message:'다음 결과를 불러오지 못했습니다.')}finally{setLoadingMore(false)}}
  const replaceBody=(scope?:WorkbookSearchMatch)=>({
    query:options.useRegex?query:query.trim(),replacement,
    match_case:options.matchCase,whole_cell:options.wholeCell,use_regex:options.useRegex,skip_formulas:options.skipFormulas,
    ...(scope?{sheet_id:scope.sheet_id,range:scope.address}:options.currentSheetOnly&&sheetId?{sheet_id:sheetId}:{}),
  })
  const runReplace=async(scope?:WorkbookSearchMatch)=>{
    if(!query.trim()||replacing)return
    setReplacing(true);setError('');setStatus('')
    try{
      const planned=await api<ReplaceResult>(`/api/v1/workbooks/${workbookId}/search:replace`,{method:'POST',body:JSON.stringify({...replaceBody(scope),preview:true})})
      if(planned.planned_cells===0){setStatus('바꿀 셀이 없습니다.');return}
      if(!scope&&!window.confirm(`${planned.planned_cells.toLocaleString()}개 셀을 바꿉니다. 계속할까요?`))return
      const key=newIdempotencyKey()
      const applied=await api<ReplaceResult>(`/api/v1/workbooks/${workbookId}/search:replace`,{method:'POST',headers:{'Idempotency-Key':key},body:JSON.stringify({...replaceBody(scope),idempotency_key:key,client_id:collaborationClientId()})})
      setStatus(`${applied.replaced_cells.toLocaleString()}개 셀을 바꿨습니다.${applied.skipped_cells>0?` (${applied.skipped_cells.toLocaleString()}개 건너뜀)`:''}`)
      onReplaced?.(applied)
    }catch(reason){setError(reason instanceof Error?reason.message:'바꾸지 못했습니다.')}finally{setReplacing(false)}
  }
  const toggle=(key:keyof SearchOptions)=>setOptions(current=>({...current,[key]:!current[key]}))
  const optionRow:Array<{key:keyof SearchOptions;label:string;title:string}>=[
    {key:'matchCase',label:'Aa',title:'대소문자 구분'},
    {key:'wholeCell',label:'[ ]',title:'셀 전체 일치'},
    {key:'useRegex',label:'.*',title:'정규식 사용'},
    {key:'skipFormulas',label:'fx',title:'수식 검색 제외'},
    {key:'currentSheetOnly',label:'⊞',title:sheetName?`${sheetName} 시트만 검색`:'현재 시트만 검색'},
  ]
  return <div className="search-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="workbook-search" role="dialog" aria-modal="true" aria-label="워크북 찾기 및 바꾸기">
      <div className="workbook-search-input"><Search/><input ref={inputRef} value={query} maxLength={256} placeholder="값 또는 수식 검색" aria-label="검색어" onChange={event=>setQuery(event.target.value)} onKeyDown={event=>{if(event.key==='ArrowDown'){setActiveIndex(index=>items.length?Math.min(items.length-1,index+1):0);event.preventDefault()}else if(event.key==='ArrowUp'){setActiveIndex(index=>Math.max(0,index-1));event.preventDefault()}else if(event.key==='Enter'&&items[activeIndex]){if(event.shiftKey){preview(items[activeIndex]);setActiveIndex(index=>Math.min(items.length-1,index+1))}else choose(items[activeIndex]);event.preventDefault()}}}/><kbd>ESC</kbd><button aria-label="검색 닫기" onClick={onClose}><X/></button></div>
      <div className="workbook-search-options">
        <button type="button" className={showReplace?'active':''} aria-pressed={showReplace} aria-label="바꾸기 입력 표시" title="바꾸기 입력 표시" onClick={()=>setShowReplace(current=>!current)}><Replace/> 바꾸기</button>
        <span className="workbook-search-divider"/>
        {optionRow.map(option=><button key={option.key} type="button" className={options[option.key]?'active':''} aria-pressed={options[option.key]} aria-label={option.title} title={option.title} disabled={option.key==='currentSheetOnly'&&!sheetId} onClick={()=>toggle(option.key)}>{option.label}</button>)}
      </div>
      {showReplace&&<div className="workbook-search-replace">
        <input value={replacement} maxLength={256} placeholder={options.useRegex?'바꿀 내용 ($1로 그룹 참조)':'바꿀 내용'} aria-label="바꿀 내용" onChange={event=>setReplacement(event.target.value)}/>
        <button type="button" disabled={replacing||!items[activeIndex]} onClick={()=>void runReplace(items[activeIndex])}>바꾸기</button>
        <button type="button" className="primary" disabled={replacing||!query.trim()} onClick={()=>void runReplace()}>모두 바꾸기</button>
      </div>}
      <div className="workbook-search-meta"><span>{loading?'서버 저장 상태를 검색하는 중…':result?`${result.items.length.toLocaleString()}개 결과${result.next_offset!=null?' 이상':''}`:'현재 워크북의 값과 수식을 검색합니다.'}</span>{result&&<small>워크북 v{result.workbook_version}</small>}</div>
      <div className="workbook-search-results" role="listbox" aria-label="검색 결과">
        {!loading&&query.trim()&&items.length===0&&!error&&<div className="workbook-search-empty"><Search/><strong>일치하는 셀이 없습니다.</strong><span>검색 옵션을 바꾸거나 다른 검색어를 입력해 보세요.</span></div>}
        {!query.trim()&&<div className="workbook-search-empty"><FileSpreadsheet/><strong>워크북 전체 검색</strong><span>대소문자 구분, 셀 전체 일치, 정규식과 시트 범위를 조합할 수 있습니다.</span></div>}
        {items.map((item,index)=><button key={`${item.sheet_id}:${item.address}`} role="option" aria-selected={index===activeIndex} className={index===activeIndex?'active':''} onMouseEnter={()=>setActiveIndex(index)} onClick={()=>choose(item)}><span className="search-location"><FileSpreadsheet/><strong>{item.sheet_name}</strong><code>{item.address}</code></span><span className="search-preview">{matchLabel(item)}</span><span className="search-fields">{item.matched_fields.map(field=><em key={field}>{field==='formula'?'수식':'값'}</em>)}</span></button>)}
      </div>
      {error&&<div className="workbook-search-error" role="alert">{error}</div>}
      {status&&<div className="workbook-search-status" role="status">{status}</div>}
      {result?.next_offset!=null&&<button className="workbook-search-more" disabled={loadingMore} onClick={()=>void loadMore()}>{loadingMore?'불러오는 중…':'결과 더 보기'}</button>}
      <footer><span><kbd>↑</kbd><kbd>↓</kbd> 이동</span><span><kbd>Enter</kbd> 셀 열기</span><span><kbd>Shift</kbd>+<kbd>Enter</kbd> 미리 보기</span><span>서버 권위 검색</span></footer>
    </section>
  </div>
}
