import { useState } from 'react'
import { UploadCloud, AlertTriangle } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { api } from '../lib/api'
import { readFileText } from '../lib/fileText'
import './UserImportDialog.css'

type Row={line:number;user_id:string;display_name?:string;email?:string;note?:string;action:'create'|'update'|'skip';reason?:string}
type Result={items:Row[];applied:boolean;created?:number;updated?:number}

const SAMPLE=`user_id,display_name,email\nkim.nara,김나라,kim.nara@corp.example\npark.dam,박다음,park.dam@corp.example`

/**
 * 사용자를 CSV 로 한꺼번에 등록한다. 팀 하나를 들이려고 스무 번을 누르는
 * 일을 없앤다.
 *
 * 먼저 미리 보여 준다. 사람을 스무 명 만드는 일은 되돌리기 번거롭고,
 * 이미 있는 사람의 이름을 덮어쓰는 것은 더 그렇다.
 */
export function UserImportDialog({onClose,onDone}:{onClose:()=>void;onDone:(message:string)=>void}){
  const dialog=useDialog<HTMLElement>(onClose)
  const [text,setText]=useState('')
  const [rows,setRows]=useState<Row[]>()
  const [error,setError]=useState('')
  const [busy,setBusy]=useState(false)

  const call=async(preview:boolean)=>{
    setBusy(true);setError('')
    try{
      const result=await api<Result>('/api/v1/admin/users:import',{method:'POST',body:JSON.stringify({csv:text,preview})})
      setRows(result.items)
      if(result.applied)onDone(`${result.created ?? 0}명을 등록하고 ${result.updated ?? 0}명을 갱신했습니다.`)
    }catch(reason){setError(reason instanceof Error?reason.message:'읽지 못했습니다.');setRows(undefined)}
    finally{setBusy(false)}
  }
  const readFile=async(file?:File)=>{if(!file)return;setText(await readFileText(file));setRows(undefined)}
  const counts={
    create:rows?.filter(row=>row.action==='create').length??0,
    update:rows?.filter(row=>row.action==='update').length??0,
    skip:rows?.filter(row=>row.action==='skip').length??0,
  }

  return <div className="modal-backdrop"><div className="modal user-import-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="사용자 일괄 등록">
    <header><div><UploadCloud/><div><h2>사용자 일괄 등록</h2><p>CSV를 붙여 넣거나 파일을 고르면 먼저 무엇이 바뀌는지 보여 줍니다.</p></div></div><button aria-label="사용자 일괄 등록 닫기" onClick={onClose}>×</button></header>
    <div className="user-import-body">
      {error&&<p className="user-import-error"><AlertTriangle/> {error}</p>}
      <label className="user-import-field">CSV
        <textarea aria-label="CSV 내용" rows={6} value={text} placeholder={SAMPLE} onChange={event=>{setText(event.target.value);setRows(undefined)}}/>
      </label>
      <div className="user-import-row">
        {/* 서버는 탭·세미콜론으로 가른 명단도 읽는다. 엑셀의 '텍스트(탭으로 분리)'
            저장과 AD 에서 뽑은 .txt 가 그 꼴이라 파일 고르기도 그것을 보여 준다. */}
        <input type="file" aria-label="CSV 파일" accept=".csv,.tsv,.txt,text/csv,text/tab-separated-values,text/plain" onChange={event=>void readFile(event.target.files?.[0])}/>
        <button className="secondary" onClick={()=>{setText(SAMPLE);setRows(undefined)}}>보기 채우기</button>
      </div>
      <p className="user-import-hint">머리글 줄이 있어야 합니다. <code>user_id</code>(필수) · <code>display_name</code> · <code>email</code> · <code>note</code> 를 읽고, <code>사용자 ID</code> · <code>이름</code> · <code>이메일</code> · <code>메모</code> 로 적어도 됩니다. 역할과 부서는 등록한 뒤 사용자별로 지정합니다.</p>
      {rows&&<>
        <p className="user-import-counts">새로 만듦 <b>{counts.create}</b> · 갱신 <b>{counts.update}</b>{counts.skip>0&&<> · 건너뜀 <b>{counts.skip}</b></>}</p>
        <ul className="user-import-list">{rows.slice(0,20).map(row=><li key={row.line} className={row.action}>
          <b>{row.line}행</b><code>{row.user_id||'(빈 값)'}</code><span>{row.display_name}</span>
          <em>{row.action==='create'?'새로 만듦':row.action==='update'?'갱신':'건너뜀'}</em>
          {row.reason&&<small>{row.reason}</small>}
        </li>)}
        {rows.length>20&&<li className="user-import-more">… 그리고 {rows.length-20}줄 더</li>}</ul>
      </>}
    </div>
    <div className="modal-actions"><span/>
      <button className="secondary" onClick={onClose}>닫기</button>
      <button className="secondary" disabled={busy||text.trim()===''} onClick={()=>void call(true)}>{busy?'읽는 중…':'미리 보기'}</button>
      <button className="primary" disabled={busy||!rows||counts.create+counts.update===0} onClick={()=>void call(false)}>{counts.create+counts.update}명 등록</button>
    </div>
  </div></div>
}
