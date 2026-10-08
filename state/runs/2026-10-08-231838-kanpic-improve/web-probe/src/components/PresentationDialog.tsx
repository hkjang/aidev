import { Download,ExternalLink,Presentation } from 'lucide-react'
import { useEffect,useState, useMemo} from 'react'
import { address } from '../lib/api'
import type { MergeRange } from '../lib/merge'
import { useDialog } from '../lib/useDialog'
import './PresentationDialog.css'

export type PresentationTemplate = { id:string; name:string; built_in?:boolean }
export type PresentationSlide = { kind:string; title:string; lead?:string; bullets?:string[]; component?:{kind:string;caption?:string;rows:Array<{label:string;fields?:string[]}>}; notes?:string }
export type PresentationDeck = { title:string; subtitle?:string; slides:PresentationSlide[] }
export type PresentationAnalysis = { shape:string; chart:string; row_count:number; has_header:boolean; headline:string; columns:Array<{name:string;kind:string;role:string}> }
export type PresentationResult = { id:string; title:string; status:string; slide_count:number; template?:string; edit_url?:string; warnings:string[] }

const roleLabels:Record<string,string>={dimension:'항목',measure:'값',change:'증감',attainment:'달성률',share:'비중'}
const shapeLabels:Record<string,string>={categories:'항목별 비교',series:'시간에 따른 추이',figures:'지표 몇 개',table:'표',empty:'내용 없음'}
const componentLabels:Record<string,string>={kpi:'지표 타일',bars:'막대 차트',line:'선 차트',share:'비중',comparison:'비교',table:'표'}

/** 마지막으로 고른 디자인을 기억해 두는 자리. 브라우저마다 따로 남는다. */
const LAST_TEMPLATE_KEY='kanpic.presentation.template'

export function PresentationDialog({range,onClose,onPreview,onCreate,onLoadTemplates,onDownload}:{
  range:MergeRange
  onClose:()=>void
  onPreview:(input:Record<string,unknown>)=>Promise<{deck:PresentationDeck;analysis:PresentationAnalysis}>
  onCreate:(input:Record<string,unknown>)=>Promise<{presentation:PresentationResult}>
  onLoadTemplates:()=>Promise<PresentationTemplate[]>
  onDownload:(id:string)=>Promise<void>
}){
  const selected=`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`
  const [title,setTitle]=useState('')
  const [templateId,setTemplateId]=useState('')
  const [includeTable,setIncludeTable]=useState(true)
  const [templates,setTemplates]=useState<PresentationTemplate[]>([])
  const [preview,setPreview]=useState<{deck:PresentationDeck;analysis:PresentationAnalysis}>()
  const [result,setResult]=useState<PresentationResult>()
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)

  useEffect(()=>{onLoadTemplates().then(setTemplates).catch(()=>setTemplates([]))},[onLoadTemplates])
  /**
   * 쉰 가지가 한 줄로 늘어서면 고르기가 아니라 훑기가 된다. 이름이
   * "Ptium <색> <배치>" 로 되어 있으므로 색으로 묶어 준다.
   */
  const templateGroups=useMemo(()=>{
    const families=new Map<string,PresentationTemplate[]>()
    for(const template of templates){
      const parts=template.name.split(/\s+/)
      const family=parts.length>2?parts[1]:'기타'
      const bucket=families.get(family)
      if(bucket)bucket.push(template)
      else families.set(family,[template])
    }
    return [...families.entries()]
  },[templates])
  /**
   * 고르는 순간 기억한다. 만들 때만 기억하면, 골라 두고 닫았다가 다시 연
   * 사람에게는 아무것도 남지 않는다.
   */
  const rememberTemplate=(id:string)=>{
    setTemplateId(id)
    try{window.localStorage.setItem(LAST_TEMPLATE_KEY,id)}catch{/* 기억하지 못해도 고르는 데는 지장이 없다. */}
  }
  /**
   * 고른 디자인을 기억한다. 기억하지 않으면 열 때마다 "기본 디자인" 으로
   * 돌아가고, 그러면 아무것도 보내지 않아 만드는 덱이 늘 같은 모습이 된다.
   */
  useEffect(()=>{
    if(templateId||templates.length===0)return
    try{
      const remembered=window.localStorage.getItem(LAST_TEMPLATE_KEY)
      if(remembered&&templates.some(template=>template.id===remembered))setTemplateId(remembered)
    }catch{/* 저장소를 못 읽어도 고르는 데는 지장이 없다. */}
  },[templates,templateId])

  // 미리보기는 서버가 만든다. 여기서 따로 그리면 언젠가 만들어지는 것과 다른
  // 것을 보여 주게 된다.
  useEffect(()=>{
    let cancelled=false
    setError('')
    onPreview({range:selected,title:title.trim(),include_table:includeTable,preview:true})
      .then(next=>{if(!cancelled)setPreview(next)})
      .catch(problem=>{if(!cancelled)setError(problem instanceof Error?problem.message:'범위를 읽지 못했습니다.')})
    return()=>{cancelled=true}
  },[onPreview,selected,title,includeTable])

  const create=async()=>{
    setBusy(true);setError('')
    try{
      const made=await onCreate({range:selected,title:title.trim(),include_table:includeTable,template_id:templateId})
      setResult(made.presentation)
    }catch(problem){
      setError(problem instanceof Error?problem.message:'프레젠테이션을 만들지 못했습니다.')
    }finally{setBusy(false)}
  }

  return <div className="modal-backdrop"><div className="modal validation-modal presentation-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="프레젠테이션 만들기">
    <header><div><Presentation/><div><h2>프레젠테이션 만들기</h2><p>{selected} 범위를 읽어 슬라이드로 만듭니다. 값이 무엇을 뜻하는지 kanpic이 먼저 판단합니다.</p></div></div><button aria-label="프레젠테이션 닫기" onClick={onClose}>×</button></header>
    <div className="presentation-body">
      <section className="presentation-form">
        <label>제목<input aria-label="프레젠테이션 제목" maxLength={200} value={title} placeholder={preview?.deck.title??'범위에서 자동으로 정합니다'} onChange={event=>setTitle(event.target.value)}/></label>
        <label>디자인<select aria-label="프레젠테이션 템플릿" value={templateId} onChange={event=>rememberTemplate(event.target.value)}>
          <option value="">기본 디자인</option>
          {templateGroups.map(([family,items])=><optgroup key={family} label={family}>
            {items.map(template=><option key={template.id} value={template.id}>{template.name}</option>)}
          </optgroup>)}
        </select></label>
        <label className="presentation-check"><input aria-label="상세 표 넣기" type="checkbox" checked={includeTable} onChange={event=>setIncludeTable(event.target.checked)}/> 원본 표를 슬라이드로 넣기</label>
        {preview&&<div className="presentation-reading">
          <strong>이 범위를 이렇게 읽었습니다</strong>
          <p>{shapeLabels[preview.analysis.shape]??preview.analysis.shape}{preview.analysis.chart?` · ${componentLabels[preview.analysis.chart]??preview.analysis.chart}`:''} · {preview.analysis.row_count.toLocaleString()}행</p>
          <ul>{preview.analysis.columns.map((column,index)=><li key={index}><em>{column.name||`열 ${index+1}`}</em><span>{roleLabels[column.role]??column.role}</span></li>)}</ul>
          {!preview.analysis.has_header&&<small>첫 줄을 열 이름이 아니라 자료로 읽었습니다.</small>}
        </div>}
      </section>
      <section className="presentation-preview" aria-label="슬라이드 미리보기">
        {preview?preview.deck.slides.map((slide,index)=><article key={index} className="presentation-slide">
          <span className="presentation-slide-number">{index+1}</span>
          <strong>{slide.title}</strong>
          {slide.lead&&<p>{slide.lead}</p>}
          {slide.component&&<div className="presentation-component"><span>{componentLabels[slide.component.kind]??slide.component.kind}</span>{slide.component.rows.slice(0,4).map((row,at)=><code key={at}>{[row.label,...(row.fields??[])].filter(Boolean).join(' · ')}</code>)}{slide.component.rows.length>4&&<code>…{slide.component.rows.length-4}줄 더</code>}</div>}
          {slide.bullets&&slide.bullets.length>0&&<ul>{slide.bullets.map((bullet,at)=><li key={at}>{bullet}</li>)}</ul>}
        </article>):<p className="presentation-empty">범위를 읽는 중…</p>}
      </section>
    </div>
    {result&&<div className="presentation-result" role="status">
      <strong>{result.slide_count}장을 만들었습니다{result.template?` · ${result.template}`:''}</strong>
      {result.warnings.length>0&&<ul className="presentation-warnings">{result.warnings.map((warning,index)=><li key={index}>{warning}</li>)}</ul>}
    </div>}
    {error&&<div className="presentation-error" role="alert">{error}</div>}
    <div className="modal-actions">
      <span/>
      <button className="secondary" onClick={onClose}>닫기</button>
      {result
        ?<><a className="secondary presentation-link" href={result.edit_url} target="_blank" rel="noreferrer"><ExternalLink/> 편집기에서 열기</a><button className="primary" onClick={()=>void onDownload(result.id)}><Download/> PowerPoint 내려받기</button></>
        :<button className="primary" disabled={busy||!preview||preview.deck.slides.length===0} onClick={()=>void create()}>{busy?'만드는 중…':'프레젠테이션 만들기'}</button>}
    </div>
  </div></div>
}
