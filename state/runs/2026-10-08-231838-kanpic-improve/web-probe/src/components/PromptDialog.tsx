import { useState } from 'react'
import { useDialog } from '../lib/useDialog'
import './PromptDialog.css'

export type PromptRequest={
  title:string
  label:string
  value?:string
  placeholder?:string
  hint?:string
  confirmLabel?:string
  multiline?:boolean
  /** Return a message to refuse the value, or nothing to accept it. */
  validate?:(value:string)=>string|undefined
  onSubmit:(value:string)=>void
}

/**
 * The replacement for window.prompt in flows that actually collect something.
 * The browser dialog cannot be labelled, validated or reached by assistive
 * technology, and some embedded contexts suppress it entirely — a rename that
 * silently does nothing is worse than one that is a little more code.
 */
export function PromptDialog({request,onClose}:{request:PromptRequest;onClose:()=>void}){
  const [value,setValue]=useState(request.value??'')
  const [error,setError]=useState<string>()
  const dialog=useDialog<HTMLElement>(onClose)
  const submit=()=>{
    const message=request.validate?.(value)
    if(message){setError(message);return}
    request.onSubmit(value)
    onClose()
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal prompt-modal" ref={dialog as React.RefObject<never>} role="dialog" aria-modal="true" aria-label={request.title}>
      <h2>{request.title}</h2>
      <label>{request.label}
        {request.multiline
          ?<textarea aria-label={request.label} value={value} placeholder={request.placeholder} onChange={event=>{setValue(event.target.value);setError(undefined)}} autoFocus/>
          :<input aria-label={request.label} value={value} placeholder={request.placeholder} autoFocus
            onChange={event=>{setValue(event.target.value);setError(undefined)}}
            onKeyDown={event=>{if(event.key==='Enter'){event.preventDefault();submit()}}}/>}
      </label>
      {request.hint&&!error&&<p className="prompt-hint">{request.hint}</p>}
      {error&&<p className="prompt-error" role="alert">{error}</p>}
      <div className="modal-actions">
        <button onClick={onClose}>취소</button>
        <button className="primary" onClick={submit}>{request.confirmLabel??'저장'}</button>
      </div>
    </section>
  </div>
}
