import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { LayoutTemplate, Search, X } from 'lucide-react'
import { api } from '../lib/api'
import { useDialog } from '../lib/useDialog'
import './TemplateGallery.css'

export type WorkbookTemplate={id:string;name:string;category:string;summary:string;columns:string[];sheets:string[]}

export function useTemplates(){
  const templates=useQuery({queryKey:['templates'],queryFn:()=>api<{items:WorkbookTemplate[]}>('/api/v1/templates'),staleTime:60*60*1000})
  return templates.data?.items??[]
}

/**
 * Browses the template catalog by category. Each card shows the columns the
 * template starts with so the choice can be made without opening it first.
 */
export function TemplateGallery({onClose,onCreate,pending}:{onClose:()=>void;onCreate:(template:WorkbookTemplate)=>void;pending?:string}){
  const [query,setQuery]=useState(''),[category,setCategory]=useState('전체')
  const dialog=useDialog<HTMLElement>(onClose)
  const templates=useTemplates()
  const categories=['전체',...Array.from(new Set(templates.map(item=>item.category)))]
  const needle=query.trim().toLowerCase()
  const visible=templates.filter(item=>
    (category==='전체'||item.category===category)&&
    (!needle||item.name.toLowerCase().includes(needle)||item.summary.toLowerCase().includes(needle)||item.columns.join(' ').toLowerCase().includes(needle)))
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal template-gallery" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="템플릿 갤러리">
      <header>
        <div><h2>템플릿 갤러리</h2><p>바로 쓸 수 있는 {templates.length}개 템플릿입니다. 표와 수식, 서식이 이미 들어 있습니다.</p></div>
        <button aria-label="템플릿 갤러리 닫기" onClick={onClose}><X/></button>
      </header>
      <div className="template-filters">
        <div className="template-search"><Search/><input autoFocus aria-label="템플릿 검색" placeholder="템플릿 이름이나 항목으로 검색" value={query} onChange={event=>setQuery(event.target.value)}/></div>
        <div className="template-categories" role="tablist" aria-label="템플릿 분류">
          {categories.map(item=><button key={item} role="tab" aria-selected={category===item} className={category===item?'active':''} onClick={()=>setCategory(item)}>{item}</button>)}
        </div>
      </div>
      <div className="template-list">
        {visible.length===0&&<p className="template-empty">검색 결과가 없습니다.</p>}
        {visible.map(template=><article className="template-card" key={template.id}>
          <span className="template-mark"><LayoutTemplate/></span>
          <div>
            <strong>{template.name}</strong>
            <small>{template.summary}</small>
            <ul>{template.columns.slice(0,6).map(column=><li key={column}>{column}</li>)}</ul>
          </div>
          <button className="primary" disabled={Boolean(pending)} onClick={()=>onCreate(template)}>{pending===template.id?'만드는 중…':'사용하기'}</button>
        </article>)}
      </div>
    </section>
  </div>
}
