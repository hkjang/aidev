import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../lib/api'
import { useDialog } from '../lib/useDialog'

export type FunctionDoc={name:string;category:string;syntax:string;summary:string}

/**
 * Lists the functions the formula engine understands. The catalog comes from
 * the server so the list can never drift from what actually evaluates.
 */
export function FunctionListDialog({onClose,onInsert}:{onClose:()=>void;onInsert?:(name:string)=>void}){
  const [query,setQuery]=useState('')
  const dialog=useDialog<HTMLElement>(onClose)
  const functions=useQuery({queryKey:['formula-functions'],queryFn:()=>api<{items:FunctionDoc[]}>('/api/v1/formula/functions'),staleTime:60*60*1000})
  const needle=query.trim().toLowerCase()
  const items=(functions.data?.items??[]).filter(item=>!needle||item.name.toLowerCase().includes(needle)||item.summary.includes(needle)||item.category.includes(needle))
  const categories=Array.from(new Set(items.map(item=>item.category)))
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal function-list-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="함수 목록">
      <h2>함수 목록</h2>
      <p>kanpic 수식 엔진이 계산할 수 있는 함수입니다. 목록에 없는 함수는 <code>#NAME?</code> 오류를 반환합니다.</p>
      <input autoFocus aria-label="함수 검색" placeholder="함수 이름이나 설명으로 검색" value={query} onChange={event=>setQuery(event.target.value)}/>
      <div className="function-list">
        {functions.isLoading&&<p className="function-empty">함수 목록을 불러오는 중…</p>}
        {!functions.isLoading&&items.length===0&&<p className="function-empty">검색 결과가 없습니다.</p>}
        {categories.map(category=><section key={category}>
          <h3>{category}</h3>
          {items.filter(item=>item.category===category).map(item=><div className="function-row" key={item.name}>
            <div><strong>{item.name}</strong><code>{item.syntax}</code></div>
            <p>{item.summary}</p>
            {onInsert&&<button onClick={()=>{onInsert(item.name);onClose()}}>삽입</button>}
          </div>)}
        </section>)}
      </div>
      <div className="modal-actions"><button className="primary" onClick={onClose}>닫기</button></div>
    </section>
  </div>
}
