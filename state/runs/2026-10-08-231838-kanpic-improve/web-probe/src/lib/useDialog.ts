import { useEffect, useRef } from 'react'

const FOCUSABLE='a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])'

function focusableElements(root:HTMLElement|null){
  if(!root)return []
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(element=>element.offsetParent!==null||element===document.activeElement)
}

/**
 * Shared dialog behaviour so every modal in kanpic feels the same:
 * Escape closes it, Tab cycles inside it, the page behind stops scrolling, the
 * first meaningful field takes focus, and the element that opened the dialog
 * gets focus back when it closes.
 *
 * Attach the returned ref to the dialog element itself, not to the backdrop.
 */
export function useDialog<T extends HTMLElement=HTMLElement>(onClose:()=>void,options:{closeOnEscape?:boolean}={}){
  const ref=useRef<T>(null)
  const close=useRef(onClose)
  close.current=onClose
  const escapeEnabled=options.closeOnEscape!==false
  useEffect(()=>{
    const dialog=ref.current
    const opener=document.activeElement instanceof HTMLElement?document.activeElement:null
    const previousOverflow=document.body.style.overflow
    document.body.style.overflow='hidden'
    // Prefer a real input over the close button so typing starts immediately.
    if(dialog&&!dialog.contains(document.activeElement)){
      const candidates=focusableElements(dialog)
      const preferred=candidates.find(element=>element.hasAttribute('autofocus'))
        ??candidates.find(element=>/^(INPUT|SELECT|TEXTAREA)$/.test(element.tagName))
        ??candidates[0]
      preferred?.focus()
    }
    const keydown=(event:KeyboardEvent)=>{
      if(!dialog)return
      if(event.key==='Escape'&&escapeEnabled&&!event.defaultPrevented){
        // A nested surface such as a context menu handles Escape first.
        event.preventDefault()
        event.stopPropagation()
        close.current()
        return
      }
      if(event.key!=='Tab')return
      const candidates=focusableElements(dialog)
      if(candidates.length===0)return
      const first=candidates[0],last=candidates[candidates.length-1]
      const active=document.activeElement
      if(!dialog.contains(active)){first.focus();event.preventDefault();return}
      if(event.shiftKey&&active===first){last.focus();event.preventDefault()}
      else if(!event.shiftKey&&active===last){first.focus();event.preventDefault()}
    }
    // Listening on the window keeps Escape working even when focus slipped out.
    window.addEventListener('keydown',keydown,true)
    return()=>{
      window.removeEventListener('keydown',keydown,true)
      document.body.style.overflow=previousOverflow
      if(opener&&document.body.contains(opener))opener.focus()
    }
  },[escapeEnabled])
  return ref
}
