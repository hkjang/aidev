import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Check, Globe2, Link2, Lock, Mail, ShieldCheck, UserPlus, Users, X } from 'lucide-react'
import { api } from '../lib/api'
import { useUserDirectory, useUserSearch, userLabel, userTooltip } from '../state/directory'
import type { AccessRequest, Department, LinkAccess, PrincipalType, ShareRole, SharingResponse, Workbook, WorkbookShare } from '../types'
import './ShareDialog.css'
import { useDialog } from '../lib/useDialog'

const ROLE_LABELS:Record<ShareRole,string>={viewer:'뷰어',commenter:'댓글 작성자',editor:'편집자',owner:'소유자'}
const ROLE_HINTS:Record<ShareRole,string>={viewer:'보기와 검색만 가능합니다.',commenter:'댓글을 남길 수 있고 셀은 편집할 수 없습니다.',editor:'셀·서식·구조를 편집할 수 있습니다.',owner:'공유와 소유권까지 관리합니다.'}
const LINK_LABELS:Record<LinkAccess,string>={restricted:'제한됨',organization:'조직 내 모든 사용자',anyone:'링크가 있는 모든 사용자'}
const LINK_HINTS:Record<LinkAccess,string>={
  restricted:'추가된 사용자·부서·역할만 열 수 있습니다.',
  organization:'로그인한 조직 구성원이면 누구나 링크로 열 수 있습니다.',
  anyone:'링크를 아는 모든 로그인 사용자가 열 수 있습니다.',
}
const ASSIGNABLE:ShareRole[]=['viewer','commenter','editor']

export function principalIcon(type:PrincipalType){
  if(type==='department')return <Building2/>
  if(type==='role')return <ShieldCheck/>
  return <Mail/>
}

function shareLabel(share:WorkbookShare,departments:Department[],directory?:Map<string,{user_id:string;display_name?:string;email?:string}>){
  if(share.principal_type==='user')return share.principal_label?.trim()||userLabel(share.principal_id,directory)
  if(share.principal_type==='department'){
    const department=departments.find(item=>item.id===share.principal_id)
    return department?.path||department?.name||share.principal_label||share.principal_id
  }
  return share.principal_label||share.principal_id
}

function principalTypeLabel(type:PrincipalType){
  return type==='department'?'부서':type==='role'?'역할':'사용자'
}

export function accessSummary(access:{role:ShareRole|'';source?:string;source_label?:string}){
  if(!access.role)return '권한 없음'
  const role=ROLE_LABELS[access.role as ShareRole]
  switch(access.source){
    case 'owner':return `${role} · 이 워크북의 소유자입니다`
    case 'admin':return `${role} · 관리자 권한으로 접근 중입니다`
    case 'department':return `${role} · ${access.source_label||'부서'} 공유`
    case 'role':return `${role} · ${access.source_label||'역할'} 공유`
    case 'link':return `${role} · 링크 액세스`
    default:return `${role} · 직접 공유`
  }
}

/**
 * Google Sheets style sharing: people, departments and identity provider roles
 * on top, general link access below, and owner-only controls for locking
 * sharing, viewer downloads, ownership transfer and pending access requests.
 */
export function ShareDialog({workbook,onClose,onChanged}:{workbook:Workbook;onClose:()=>void;onChanged?:(sharing:SharingResponse)=>void}){
  const client=useQueryClient()
  const [principalType,setPrincipalType]=useState<PrincipalType>('user')
  const [principal,setPrincipal]=useState('')
  const [role,setRole]=useState<ShareRole>('editor')
  const [busy,setBusy]=useState(false),[error,setError]=useState(''),[status,setStatus]=useState('')
  const [transferTo,setTransferTo]=useState(''),[transferring,setTransferring]=useState(false)
  const sharingQuery=useQuery({queryKey:['workbook-sharing',workbook.id],queryFn:()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/sharing`)})
  const departments=useQuery({queryKey:['departments'],queryFn:()=>api<{items:Department[]}>('/api/v1/departments')})
  const access=sharingQuery.data?.access
  const sharing=sharingQuery.data?.sharing
  const canManage=access?.can_manage===true
  const isOwner=access?.role==='owner'
  const requests=useQuery({
    queryKey:['access-requests',workbook.id],
    queryFn:()=>api<{items:AccessRequest[]}>(`/api/v1/workbooks/${workbook.id}/access-requests?status=pending`),
    enabled:canManage,
  })
  const link=useMemo(()=>`${window.location.origin}/workbooks/${workbook.id}`,[workbook.id])
  const directory=useUserDirectory([sharing?.owner_id,...(sharing?.shares??[]).filter(share=>share.principal_type==='user').map(share=>share.principal_id)])
  const suggestions=useUserSearch(principalType==='user'?principal:'')
  const departmentOptions=departments.data?.items??[]

  const apply=async(action:()=>Promise<SharingResponse>)=>{
    setBusy(true);setError('');setStatus('')
    try{
      const result=await action()
      sharingQuery.refetch()
      client.invalidateQueries({queryKey:['workbooks']})
      client.invalidateQueries({queryKey:['workbook',workbook.id]})
      onChanged?.(result)
      return result
    }catch(reason){
      setError(reason instanceof Error?reason.message:'공유 설정을 저장하지 못했습니다.')
      return undefined
    }finally{setBusy(false)}
  }
  const addShare=async()=>{
    const value=principal.trim()
    if(!value)return
    const label=principalType==='department'
      ?departmentOptions.find(item=>item.id===value)?.name??''
      :principalType==='user'?suggestions.find(item=>item.user_id.toLowerCase()===value.toLowerCase())?.display_name??'':value
    const result=await apply(()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/shares`,{method:'PUT',body:JSON.stringify({principal_type:principalType,principal_id:value,principal_label:label,role})}))
    if(result){setPrincipal('');setStatus(`${principalTypeLabel(principalType)} 공유를 저장했습니다.`)}
  }
  const changeRole=(share:WorkbookShare,next:ShareRole)=>void apply(()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/shares`,{method:'PUT',body:JSON.stringify({principal_type:share.principal_type,principal_id:share.principal_id,principal_label:share.principal_label,role:next})}))
  const removeShare=(share:WorkbookShare)=>void apply(()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/shares/${share.id}`,{method:'DELETE'}))
  const patchSharing=(input:Record<string,unknown>)=>void apply(()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/sharing`,{method:'PATCH',body:JSON.stringify(input)}))
  const transfer=async()=>{
    const value=transferTo.trim()
    if(!value)return
    if(!window.confirm(`${value}에게 소유권을 넘기면 이 워크북의 공유 설정을 더 이상 변경할 수 없습니다. 계속할까요?`))return
    setTransferring(true)
    try{await apply(()=>api<SharingResponse>(`/api/v1/workbooks/${workbook.id}/sharing:transfer-ownership`,{method:'POST',body:JSON.stringify({new_owner_id:value,keep_as_editor:true})}));setTransferTo('')}
    finally{setTransferring(false)}
  }
  const decide=async(request:AccessRequest,approve:boolean)=>{
    setBusy(true);setError('')
    try{
      await api<AccessRequest>(`/api/v1/access-requests/${request.id}:${approve?'approve':'deny'}`,{method:'POST',body:JSON.stringify(approve?{role:request.requested_role}:{})})
      await requests.refetch();await sharingQuery.refetch()
      setStatus(approve?`${request.requester_id}에게 권한을 부여했습니다.`:'액세스 요청을 거부했습니다.')
    }catch(reason){setError(reason instanceof Error?reason.message:'요청을 처리하지 못했습니다.')}
    finally{setBusy(false)}
  }
  const copyLink=async()=>{
    try{await navigator.clipboard?.writeText(link);setStatus('링크를 복사했습니다.')}
    catch{window.prompt('링크를 복사하세요.',link)}
  }

  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal share-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label={`${workbook.title} 공유`}>
      <header className="share-header"><div><h2>공유</h2><p>{workbook.title}</p></div><button aria-label="공유 창 닫기" onClick={onClose}><X/></button></header>

      {sharingQuery.isLoading?<div className="share-loading">공유 설정을 불러오는 중…</div>:<>
        <div className="share-role-summary" role="status">{accessSummary(access??{role:''})}{access?.role&&<small>{ROLE_HINTS[access.role as ShareRole]}</small>}</div>

        {canManage&&<div className="share-add">
          <div className="share-add-row">
            <select aria-label="공유 대상 유형" value={principalType} onChange={event=>{setPrincipalType(event.target.value as PrincipalType);setPrincipal('')}}>
              <option value="user">사용자</option>
              <option value="department">부서</option>
              <option value="role">역할</option>
            </select>
            {principalType==='department'
              ?<select aria-label="부서 선택" value={principal} onChange={event=>setPrincipal(event.target.value)}>
                <option value="">부서를 선택하세요</option>
                {departmentOptions.map(item=><option key={item.id} value={item.id}>{item.path||item.name} ({item.member_count}명)</option>)}
              </select>
              :<><input aria-label={principalType==='role'?'역할 이름':'사용자 ID 또는 이메일'} list={principalType==='user'?'share-user-suggestions':undefined} placeholder={principalType==='role'?'예: kanpic-analyst':'이름, 사용자 ID 또는 이메일'} value={principal} onChange={event=>setPrincipal(event.target.value)} onKeyDown={event=>{if(event.key==='Enter'){event.preventDefault();void addShare()}}}/>
                {principalType==='user'&&<datalist id="share-user-suggestions">{suggestions.map(item=><option key={item.user_id} value={item.user_id}>{item.display_name?`${item.display_name}${item.email?` · ${item.email}`:''}`:item.email??item.user_id}</option>)}</datalist>}</>}
            <select aria-label="부여할 권한" value={role} onChange={event=>setRole(event.target.value as ShareRole)}>
              {ASSIGNABLE.map(item=><option key={item} value={item}>{ROLE_LABELS[item]}</option>)}
            </select>
            <button className="primary" disabled={busy||!principal.trim()} onClick={()=>void addShare()}><UserPlus/> 추가</button>
          </div>
          <small>{ROLE_HINTS[role]}</small>
        </div>}

        <div className="share-people">
          <h3>액세스 권한이 있는 사용자</h3>
          <ul>
            <li className="share-owner">
              <span className="share-avatar owner">{(sharing?.owner_id??'?').slice(0,1).toUpperCase()}</span>
              <span className="share-identity"><strong title={userTooltip(sharing?.owner_id??'',directory)}>{userLabel(sharing?.owner_id??'',directory)}</strong><small>소유자</small></span>
              <span className="share-role-static">{ROLE_LABELS.owner}</span>
            </li>
            {(sharing?.shares??[]).map(share=><li key={share.id}>
              <span className={`share-avatar ${share.principal_type}`}>{principalIcon(share.principal_type)}</span>
              <span className="share-identity"><strong title={share.principal_type==='user'?userTooltip(share.principal_id,directory):share.principal_id}>{shareLabel(share,departmentOptions,directory)}</strong><small>{principalTypeLabel(share.principal_type)}{share.principal_type==='user'&&share.principal_id!==shareLabel(share,departmentOptions,directory)?` · ${share.principal_id}`:''}</small></span>
              {canManage
                ?<><select aria-label={`${shareLabel(share,departmentOptions,directory)} 권한`} value={share.role} disabled={busy} onChange={event=>changeRole(share,event.target.value as ShareRole)}>
                    {ASSIGNABLE.map(item=><option key={item} value={item}>{ROLE_LABELS[item]}</option>)}
                  </select>
                  <button className="share-remove" aria-label={`${shareLabel(share,departmentOptions,directory)} 공유 제거`} disabled={busy} onClick={()=>removeShare(share)}><X/></button></>
                :<span className="share-role-static">{ROLE_LABELS[share.role]}</span>}
            </li>)}
            {(sharing?.shares??[]).length===0&&<li className="share-empty">아직 개별 공유 대상이 없습니다.</li>}
          </ul>
        </div>

        <div className="share-general">
          <h3>일반 액세스</h3>
          <div className="share-general-row">
            <span className={`share-avatar link ${sharing?.link_access==='restricted'?'restricted':''}`}>{sharing?.link_access==='restricted'?<Lock/>:sharing?.link_access==='anyone'?<Globe2/>:<Users/>}</span>
            <div className="share-general-controls">
              {canManage
                ?<select aria-label="일반 액세스 범위" value={sharing?.link_access??'restricted'} disabled={busy} onChange={event=>patchSharing({link_access:event.target.value})}>
                  {(Object.keys(LINK_LABELS) as LinkAccess[]).map(item=><option key={item} value={item}>{LINK_LABELS[item]}</option>)}
                </select>
                :<strong>{LINK_LABELS[sharing?.link_access??'restricted']}</strong>}
              <small>{LINK_HINTS[sharing?.link_access??'restricted']}</small>
            </div>
            {sharing?.link_access!=='restricted'&&(canManage
              ?<select aria-label="링크 액세스 권한" value={sharing?.link_role??'viewer'} disabled={busy} onChange={event=>patchSharing({link_role:event.target.value})}>
                {ASSIGNABLE.map(item=><option key={item} value={item}>{ROLE_LABELS[item]}</option>)}
              </select>
              :<span className="share-role-static">{ROLE_LABELS[sharing?.link_role??'viewer']}</span>)}
          </div>
        </div>

        {isOwner&&<div className="share-policies">
          <label><input type="checkbox" checked={sharing?.sharing_locked!==true} disabled={busy} onChange={event=>patchSharing({sharing_locked:!event.target.checked})}/><span>편집자가 권한을 변경하고 공유할 수 있음</span></label>
          <label><input type="checkbox" checked={sharing?.viewer_can_copy!==false} disabled={busy} onChange={event=>patchSharing({viewer_can_copy:event.target.checked})}/><span>뷰어와 댓글 작성자에게 내보내기·복사 허용</span></label>
        </div>}

        {canManage&&(requests.data?.items??[]).length>0&&<div className="share-requests">
          <h3>대기 중인 액세스 요청 {(requests.data?.items??[]).length}건</h3>
          {(requests.data?.items??[]).map(request=><div className="share-request" key={request.id}>
            <span><strong>{request.requester_name||request.requester_id}</strong><small>{ROLE_LABELS[request.requested_role]} 요청{request.message?` · ${request.message}`:''}</small></span>
            <button className="primary" disabled={busy} onClick={()=>void decide(request,true)}><Check/> 승인</button>
            <button disabled={busy} onClick={()=>void decide(request,false)}>거부</button>
          </div>)}
        </div>}

        {isOwner&&<div className="share-transfer">
          <h3>소유권 이전</h3>
          <div className="share-transfer-row">
            <input aria-label="새 소유자" placeholder="새 소유자의 사용자 ID 또는 이메일" value={transferTo} onChange={event=>setTransferTo(event.target.value)}/>
            <button disabled={transferring||busy||!transferTo.trim()} onClick={()=>void transfer()}>소유권 넘기기</button>
          </div>
          <small>이전 후 현재 소유자는 편집자로 남습니다.</small>
        </div>}
      </>}

      {error&&<div className="share-error" role="alert">{error}</div>}
      {status&&<div className="share-status" role="status">{status}</div>}
      <footer><button className="secondary" onClick={()=>void copyLink()}><Link2/> 링크 복사</button><button className="primary" onClick={onClose}>완료</button></footer>
    </section>
  </div>
}
