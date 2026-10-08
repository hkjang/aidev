import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { CornerDownLeft, Search } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import './QuickSwitcher.css'

export type QuickItem={
  id:string
  group:string
  label:string
  hint?:string
  shortcut?:string
  icon?:ReactNode
  keywords?:string
  run:()=>void
}

/**
 * Scores an item against the query. A prefix match beats a word match, which
 * beats a scattered subsequence, so the most obvious target ranks first.
 */
export function scoreQuickItem(item:QuickItem,query:string){
  const needle=query.trim().toLowerCase()
  if(!needle)return 1
  const haystack=`${item.label} ${item.hint??''} ${item.keywords??''} ${item.group}`.toLowerCase()
  const label=item.label.toLowerCase()
  if(label.startsWith(needle))return 1000-label.length
  if(label.includes(needle))return 800-label.length
  const words=haystack.split(/[\s·/,()]+/).filter(Boolean)
  if(words.some(word=>word.startsWith(needle)))return 600
  if(haystack.includes(needle))return 400
  // Subsequence match keeps "매출보고" finding "2026 매출 보고서".
  let cursor=0
  for(const character of needle){
    const index=haystack.indexOf(character,cursor)
    if(index<0)return 0
    cursor=index+1
  }
  return 100
}

const GROUP_WEIGHT:Record<string,number>={'셀 이동':80,'시트':60,'명령':40,'이름 범위':30,'이동':20}

export function filterQuickItems(items:QuickItem[],query:string,limit=40){
  return items
    .map(item=>({item,score:scoreQuickItem(item,query)===0?0:scoreQuickItem(item,query)+(GROUP_WEIGHT[item.group]??0)}))
    .filter(entry=>entry.score>0)
    .sort((left,right)=>right.score-left.score)
    .slice(0,limit)
    .map(entry=>entry.item)
}

/**
 * A keyboard-first jump list: workbooks, sheets, cells and commands in one
 * dialog opened with Ctrl/⌘+K.
 */
export function QuickSwitcher({items,dynamicItems,onQuery,placeholder='워크북, 시트, 셀 주소 또는 명령 검색',onClose}:{items:QuickItem[];dynamicItems?:(query:string)=>QuickItem[];onQuery?:(query:string)=>void;placeholder?:string;onClose:()=>void}){
  const [query,setQuery]=useState(''),[active,setActive]=useState(0)
  // 적은 말을 밖으로 알린다. dynamicItems 는 그리는 중에 불리므로 그 안에서
  // 상태를 바꿀 수 없다. 목록에 없는 것을 서버에 물어보려면 이 길이 필요하다.
  useEffect(()=>{onQuery?.(query)},[query,onQuery])
  const input=useRef<HTMLInputElement>(null),list=useRef<HTMLDivElement>(null)
  const dialog=useDialog<HTMLElement>(onClose)
  // Some entries depend on what was typed, such as jumping to a cell address.
  const matches=useMemo(()=>[...(dynamicItems?.(query)??[]),...filterQuickItems(items,query)],[items,dynamicItems,query])
  useEffect(()=>{input.current?.focus()},[])
  useEffect(()=>setActive(0),[query])
  useEffect(()=>{
    const element=list.current?.querySelector<HTMLElement>('[data-active="true"]')
    element?.scrollIntoView({block:'nearest'})
  },[active,matches.length])
  const choose=(item?:QuickItem)=>{
    if(!item)return
    onClose()
    item.run()
  }
  const groups:Array<{group:string;items:QuickItem[]}>=[]
  for(const item of matches){
    const bucket=groups.find(entry=>entry.group===item.group)
    if(bucket)bucket.items.push(item)
    else groups.push({group:item.group,items:[item]})
  }
  let index=-1
  return <div className="quick-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="quick-switcher" ref={dialog as React.RefObject<any>} role="dialog" aria-modal="true" aria-label="빠른 이동">
      <div className="quick-input"><Search/>
        <input ref={input} value={query} maxLength={120} aria-label="빠른 이동 검색" placeholder={placeholder}
          onChange={event=>setQuery(event.target.value)}
          onKeyDown={event=>{
            if(event.key==='ArrowDown'){setActive(current=>Math.min(matches.length-1,current+1));event.preventDefault()}
            else if(event.key==='ArrowUp'){setActive(current=>Math.max(0,current-1));event.preventDefault()}
            else if(event.key==='Home'){setActive(0);event.preventDefault()}
            else if(event.key==='End'){setActive(Math.max(0,matches.length-1));event.preventDefault()}
            else if(event.key==='Enter'){choose(matches[active]);event.preventDefault()}
            else if(event.key==='Escape'){onClose();event.preventDefault()}
          }}/>
        <kbd>ESC</kbd>
      </div>
      <div className="quick-results" ref={list} role="listbox" aria-label="빠른 이동 결과">
        {matches.length===0
          ?<div className="quick-empty"><Search/><strong>일치하는 항목이 없습니다.</strong><span>워크북 이름, 시트 이름, A1 주소 또는 명령을 입력해 보세요.</span></div>
          :groups.map(bucket=><div className="quick-group" key={bucket.group}>
            <span className="quick-group-title">{bucket.group}</span>
            {bucket.items.map(item=>{
              index+=1
              const position=index
              return <button key={item.id} role="option" aria-selected={position===active} data-active={position===active}
                className={position===active?'active':''}
                onMouseEnter={()=>setActive(position)}
                onClick={()=>choose(item)}>
                <span className="quick-icon">{item.icon}</span>
                <span className="quick-label"><strong>{item.label}</strong>{item.hint&&<small>{item.hint}</small>}</span>
                {item.shortcut&&<kbd>{item.shortcut}</kbd>}
              </button>
            })}
          </div>)}
      </div>
      <footer><span><kbd>↑</kbd><kbd>↓</kbd> 이동</span><span><kbd><CornerDownLeft/></kbd> 실행</span><span>{matches.length.toLocaleString()}개 항목</span></footer>
    </section>
  </div>
}
