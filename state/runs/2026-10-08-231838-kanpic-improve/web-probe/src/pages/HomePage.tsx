import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Building2, ChevronRight, Clock3, CloudOff, Copy, FilePlus2, FileUp, Grid2X2, Link as LinkIcon, Lock, MoreHorizontal, Pencil, Plus, RotateCcw, Share2, SquareArrowOutUpRight, Star, Trash, Trash2, UploadCloud, Users, Grid3X3 } from 'lucide-react'
import { AppHeader } from '../components/AppHeader'
import { QuickSwitcher, type QuickItem } from '../components/QuickSwitcher'
import './HomePage.css'

/** 한 번에 받아 오는 워크북 수. 화면을 채우고도 남을 만큼이면 된다. */
const WORKBOOK_PAGE=60
import { api, newIdempotencyKey } from '../lib/api'
import { blockedOutbox, discardOutbox, flushOutbox, type OutboxOperation } from '../lib/outbox'
import type { BuildInfo, Session, ShareRole, Workbook } from '../types'
import { ShareDialog } from '../components/ShareDialog'
import { useUserDirectory, userLabel, userTooltip } from '../state/directory'
import { useDialog } from '../lib/useDialog'
import { ContextMenu,type MenuItem } from '../components/ContextMenu'
import { TemplateGallery,useTemplates,type WorkbookTemplate } from '../components/TemplateGallery'

const ROLE_CHIP:Record<ShareRole,string>={owner:'소유자',editor:'편집자',commenter:'댓글 작성자',viewer:'뷰어'}

/** Shows how the workbook reached this user, mirroring the Sheets home list. */
function accessChip(workbook:Workbook){
  if(!workbook.access_role||workbook.access_role==='owner')return workbook.link_access&&workbook.link_access!=='restricted'
    ?<span className="workbook-access shared" title="링크 액세스가 켜져 있습니다"><Users/> 공유 중</span>
    :null
  const source=workbook.access_source==='department'?<Building2/>:workbook.access_source==='link'?<Users/>:<Share2/>
  return <span className={`workbook-access ${workbook.access_role}`} title={`${ROLE_CHIP[workbook.access_role]} 권한`}>{source} {ROLE_CHIP[workbook.access_role]}</span>
}

/** Renaming a workbook shares the standard dialog behaviour: Escape, focus trap and focus restore. */
function RenameWorkbookDialog({title,pending,onTitle,onClose,onSave}:{title:string;pending:boolean;onTitle:(value:string)=>void;onClose:()=>void;onSave:()=>void}){
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <div className="modal" ref={dialog as React.RefObject<any>} role="dialog" aria-modal="true" aria-label="워크북 이름 변경">
      <h2>워크북 이름 변경</h2>
      <label>워크북 이름<input aria-label="워크북 이름" autoFocus value={title} onChange={event=>onTitle(event.target.value)} onKeyDown={event=>{if(event.key==='Enter')onSave()}}/></label>
      <div className="modal-actions"><button className="secondary" onClick={onClose}>취소</button><button className="primary" aria-label="워크북 이름 저장" disabled={!title.trim()||pending} onClick={onSave}>저장</button></div>
    </div>
  </div>
}

export function HomePage({build,session}:{build?:BuildInfo;session?:Session}) {
  const client=useQueryClient()
  const inputRef=useRef<HTMLInputElement>(null)
  const [importFile,setImportFile]=useState<File>()
  const [preview,setPreview]=useState<{format:string;total_cells:number;sheets:Array<{name:string;rows:number;columns:number;non_empty_cells:number}>;warnings:string[]} >()
  const [importing,setImporting]=useState(false)
  const [filter,setFilter]=useState<'recent'|'favorite'|'owned'|'shared'|'trash'>('recent')
  const [shareTarget,setShareTarget]=useState<Workbook>()
  const [cardMenu,setCardMenu]=useState<{x:number;y:number;items:MenuItem[];label:string}>()
  const [renameTarget,setRenameTarget]=useState<Workbook>()
  const [renameTitle,setRenameTitle]=useState('')
  // 목록은 한 번에 한 페이지만 받는다. 예전에는 열 수 있는 워크북을 전부
  // 받아 브라우저에서 걸러 냈는데, 수천 개가 되면 몇 MB를 받아 카드를 전부
  // 그리게 된다. 그래서 검색과 필터도 서버에 함께 넘긴다.
  const [search,setSearch]=useState('')
  const [query,setQuery]=useState('')
  const [limit,setLimit]=useState(WORKBOOK_PAGE)
  // 검색어가 실제로 바뀔 때만 첫 페이지로 돌아간다. 조건 없이 되돌리면
  // 화면이 열린 직후 도는 이 타이머가 방금 누른 "더 보기" 를 취소한다.
  useEffect(()=>{
    const next=search.trim()
    if(next===query)return
    const timer=window.setTimeout(()=>{setQuery(next);setLimit(WORKBOOK_PAGE)},250)
    return()=>window.clearTimeout(timer)
  },[search,query])
  useEffect(()=>{setLimit(WORKBOOK_PAGE)},[filter])
  const serverFilter=filter==='favorite'||filter==='owned'||filter==='shared'?filter:''
  const workbooks=useQuery({
    queryKey:['workbooks',query,serverFilter,limit],
    queryFn:()=>api<{items:Workbook[];total:number;has_more:boolean}>(
      `/api/v1/workbooks?limit=${limit}${query?`&q=${encodeURIComponent(query)}`:''}${serverFilter?`&filter=${serverFilter}`:''}`),
    placeholderData:previous=>previous,
  })
  const create=useMutation({
    mutationFn:(input?:{title?:string;templateId?:string})=>api<Workbook>('/api/v1/workbooks',{method:'POST',body:JSON.stringify({
      title:input?.title?.trim()||(input?.templateId?undefined:'제목 없는 워크북'),workspace_id:'default',template_id:input?.templateId,
    })}),
    onSuccess:(wb)=>{client.invalidateQueries({queryKey:['workbooks']});window.location.href=`/workbooks/${wb.id}`},
    onError:(error)=>alert(error instanceof Error?error.message:'워크북을 만들지 못했습니다.'),
  })
  const [galleryOpen,setGalleryOpen]=useState(false)
  const templates=useTemplates()
  const featured=['monthly-sales','project-status','invoice'].map(id=>templates.find(item=>item.id===id)).filter(Boolean) as WorkbookTemplate[]
  const useTemplate=(template:WorkbookTemplate)=>create.mutate({templateId:template.id})
  const update=useMutation({mutationFn:({id,input}:{id:string;input:Record<string,unknown>})=>api<Workbook>(`/api/v1/workbooks/${id}`,{method:'PATCH',body:JSON.stringify(input)}),onSuccess:()=>client.invalidateQueries({queryKey:['workbooks']})})
  const favorite=useMutation({mutationFn:({id,value}:{id:string;value:boolean})=>api<Workbook>(`/api/v1/workbooks/${id}/favorite`,{method:'PUT',body:JSON.stringify({favorite:value})}),onSuccess:()=>client.invalidateQueries({queryKey:['workbooks']})})
  const trash=useQuery({queryKey:['workbook-trash'],queryFn:()=>api<{items:Workbook[]}>('/api/v1/workbooks/trash'),enabled:filter==='trash'})
  const restore=useMutation({mutationFn:(workbook:Workbook)=>api<Workbook>(`/api/v1/workbooks/${workbook.id}/restore`,{method:'POST'}),onSuccess:()=>{client.invalidateQueries({queryKey:['workbook-trash']});client.invalidateQueries({queryKey:['workbooks']})}})
  const purge=useMutation({mutationFn:(workbook:Workbook)=>api(`/api/v1/workbooks/${workbook.id}/purge`,{method:'DELETE'}),onSuccess:()=>client.invalidateQueries({queryKey:['workbook-trash']})})
  const duplicate=useMutation({mutationFn:(workbook:Workbook)=>api<Workbook>(`/api/v1/workbooks/${workbook.id}/duplicate`,{method:'POST',body:JSON.stringify({title:`${workbook.title} 복사본`})}),onSuccess:()=>client.invalidateQueries({queryKey:['workbooks']})})
  const remove=useMutation({mutationFn:(workbook:Workbook)=>api(`/api/v1/workbooks/${workbook.id}`,{method:'DELETE'}),onSuccess:()=>client.invalidateQueries({queryKey:['workbooks']})})
  const chooseImport=async(file?:File)=>{if(!file)return;setImportFile(file);const form=new FormData();form.append('file',file);const result=await api<{format:string;total_cells:number;sheets:Array<{name:string;rows:number;columns:number;non_empty_cells:number}>;warnings:string[]}>('/api/v1/imports:preview',{method:'POST',body:form});setPreview(result)}
  const executeImport=async()=>{if(!importFile)return;setImporting(true);try{const form=new FormData();form.append('file',importFile);form.append('workspace_id','default');const created=await api<Workbook>('/api/v1/imports',{method:'POST',body:form,headers:{'Idempotency-Key':newIdempotencyKey()}});window.location.href=`/workbooks/${created.id}`}finally{setImporting(false)}}
  const openRename=(workbook:Workbook)=>{setRenameTarget(workbook);setRenameTitle(workbook.title)}
  const rename=async()=>{if(!renameTarget||!renameTitle.trim())return;await update.mutateAsync({id:renameTarget.id,input:{title:renameTitle.trim()}});setRenameTarget(undefined)}
  const copyWorkbook=async(workbook:Workbook)=>{await duplicate.mutateAsync(workbook)}
  const deleteWorkbook=async(workbook:Workbook)=>{if(!confirm(`'${workbook.title}' 워크북과 모든 시트를 삭제할까요?`))return;await remove.mutateAsync(workbook)}
  const shareWorkbook=(workbook:Workbook)=>{setShareTarget(workbook)}
  const cardMenuItems=(workbook:Workbook):MenuItem[]=>[
    {kind:'label',label:workbook.title},
    {kind:'item',label:'열기',icon:<Grid2X2/>,onSelect:()=>{window.location.href=`/workbooks/${workbook.id}`}},
    {kind:'item',label:'새 탭에서 열기',icon:<SquareArrowOutUpRight/>,onSelect:()=>window.open(`/workbooks/${workbook.id}`,'_blank','noopener')},
    {kind:'separator'},
    {kind:'item',label:'공유…',icon:<Share2/>,onSelect:()=>shareWorkbook(workbook)},
    {kind:'item',label:workbook.favorite?'즐겨찾기 해제':'즐겨찾기에 추가',icon:<Star/>,onSelect:()=>favorite.mutate({id:workbook.id,value:!workbook.favorite})},
    {kind:'item',label:'이름 변경',icon:<Pencil/>,disabled:workbook.access_role!=='owner'&&workbook.access_role!=='editor',onSelect:()=>openRename(workbook)},
    {kind:'item',label:'복제',icon:<Copy/>,onSelect:()=>void copyWorkbook(workbook)},
    {kind:'item',label:'링크 복사',icon:<LinkIcon/>,onSelect:()=>{
      const link=`${window.location.origin}/workbooks/${workbook.id}`
      void navigator.clipboard?.writeText(link).catch(()=>window.prompt('링크를 복사하세요.',link))
    }},
    {kind:'separator'},
    {kind:'item',label:'휴지통으로 이동',icon:<Trash2/>,danger:true,disabled:workbook.access_role!=='owner',onSelect:()=>void deleteWorkbook(workbook)},
  ]
  const openCardMenu=(workbook:Workbook,point:{clientX:number;clientY:number})=>{
    setCardMenu({x:point.clientX,y:point.clientY,items:cardMenuItems(workbook),label:`${workbook.title} 메뉴`})
  }
  const directory=useUserDirectory((workbooks.data?.items??[]).map(item=>item.owner_id))
  // 걸러 내는 일은 서버가 한다. 화면은 받은 페이지를 그대로 보여 준다.
  const visibleWorkbooks=workbooks.data?.items??[]

  /**
   * 저장 큐를 비우는 일은 편집기 화면에서만 돌았다. 그래서 다시 열지 않는 워크북에
   * 갇힌 편집은 영영 나가지 못했다. 목록 화면에서도 비워 준다. 여기까지 와서도
   * 나가지 못한 것만 사람에게 보인다.
   */
  const [stranded,setStranded]=useState<OutboxOperation[]>([])
  useEffect(()=>{
    let alive=true
    const sweep=async()=>{
      const sent=await flushOutbox()
      if(!alive)return
      if(sent>0)void client.invalidateQueries({queryKey:['workbooks']})
      setStranded(await blockedOutbox())
    }
    void sweep()
    const timer=window.setInterval(()=>void sweep(),5000)
    return()=>{alive=false;window.clearInterval(timer)}
  },[client])
  const strandedGroups=useMemo(()=>{
    const counts=new Map<string,number>()
    for(const operation of stranded)counts.set(operation.workbookId??'',(counts.get(operation.workbookId??'')??0)+1)
    return[...counts.entries()].map(([workbookId,count])=>({workbookId,count}))
  },[stranded])
  /**
   * 편집기 화면은 서버가 거절한 편집을 alert 로 알렸다. 목록 화면에서 큐를 비우기
   * 시작했으니 여기서도 말해야 한다. 권한이 사라진 워크북의 편집이 조용히 사라지면
   * 사람은 저장된 줄 안다.
   */
  const [rejected,setRejected]=useState('')
  useEffect(()=>{
    const notice=(event:Event)=>setRejected((event as CustomEvent<{message?:string}>).detail?.message??'서버가 저장을 거절했습니다.')
    window.addEventListener('kanpic:outbox-rejected',notice)
    return()=>window.removeEventListener('kanpic:outbox-rejected',notice)
  },[])
  const discardStranded=async(workbookId:string)=>{
    const group=stranded.filter(operation=>(operation.workbookId??'')===workbookId)
    if(!confirm(`저장하지 못한 편집 ${group.length.toLocaleString()}건을 버립니다. 이 편집은 사라지고 서버에 있는 값이 남습니다.`))return
    await discardOutbox(group)
    setStranded(await blockedOutbox())
  }

  /**
   * 빠른 이동은 편집기에만 있었다. 목록 화면에서 Ctrl/⌘+K 를 누르면 아무
   * 일도 일어나지 않아, 단축키가 고장 난 것처럼 보였다.
   *
   * 여기서 찾을 것은 워크북과 이 화면에서 할 수 있는 일이다. 시트나 셀
   * 주소는 워크북을 연 뒤에야 뜻이 생기므로 넣지 않는다.
   */
  const [quickOpen,setQuickOpen]=useState(false)
  const [quickQuery,setQuickQuery]=useState('')
  const searchResults=useQuery({
    queryKey:['workbook-quick-search',quickQuery],
    queryFn:()=>api<{items:Workbook[]}>(`/api/v1/workbooks?limit=20&q=${encodeURIComponent(quickQuery)}`),
    enabled:quickOpen&&quickQuery.trim().length>=2,
  })
  useEffect(()=>{
    const onKeyDown=(event:KeyboardEvent)=>{
      if(!(event.metaKey||event.ctrlKey)||event.key.toLowerCase()!=='k')return
      event.preventDefault()
      setQuickOpen(true)
    }
    window.addEventListener('keydown',onKeyDown)
    return()=>window.removeEventListener('keydown',onKeyDown)
  },[])
  const quickItems:QuickItem[]=[
    ...visibleWorkbooks.map(workbook=>({
      id:`workbook:${workbook.id}`,group:'워크북',label:workbook.title,
      hint:workbook.deleted_at?'휴지통':undefined,icon:<Grid2X2/>,keywords:'workbook 워크북 열기',
      run:()=>{window.location.href=`/workbooks/${workbook.id}`},
    })),
    {id:'home:new',group:'명령',label:'새 워크북',icon:<Plus/>,keywords:'new create 새로 만들기',run:()=>create.mutate(undefined)},
    {id:'home:import',group:'명령',label:'파일 가져오기',icon:<FileUp/>,keywords:'import upload xlsx csv 가져오기 업로드',run:()=>inputRef.current?.click()},
    {id:'home:gallery',group:'명령',label:'템플릿 갤러리',icon:<Grid3X3/>,keywords:'template gallery 템플릿 갤러리',run:()=>setGalleryOpen(true)},
    {id:'home:recent',group:'목록',label:'최근 워크북',icon:<Clock3/>,keywords:'recent 최근',run:()=>setFilter('recent')},
    {id:'home:owned',group:'목록',label:'내 소유',icon:<Lock/>,keywords:'owned mine 내 소유',run:()=>setFilter('owned')},
    {id:'home:shared',group:'목록',label:'나와 공유됨',icon:<Users/>,keywords:'shared 공유',run:()=>setFilter('shared')},
    {id:'home:favorite',group:'목록',label:'즐겨찾기',icon:<Star/>,keywords:'favorite star 즐겨찾기',run:()=>setFilter('favorite')},
    {id:'home:trash',group:'목록',label:'휴지통',icon:<Trash/>,keywords:'trash deleted 휴지통 삭제',run:()=>setFilter('trash')},
  ]

  return <div className="page-shell"><AppHeader build={build} session={session}/><main className="home-content">
    {rejected&&<section className="stranded-edits" role="alert">
      <CloudOff/>
      <div><strong>서버가 저장을 거절했습니다</strong><small>{rejected}</small></div>
      <div className="stranded-actions"><span><button className="link-button" onClick={()=>setRejected('')}>확인</button></span></div>
    </section>}
    {strandedGroups.length>0&&<section className="stranded-edits" role="alert">
      <CloudOff/>
      <div>
        <strong>서버에 닿지 못한 편집이 {stranded.length.toLocaleString()}건 남아 있습니다</strong>
        <small>여러 번 다시 보냈지만 서버가 받지 않았습니다. 워크북을 열어 다시 시도하거나, 여기서 버릴 수 있습니다.</small>
      </div>
      <div className="stranded-actions">{strandedGroups.map(group=>{
        const title=visibleWorkbooks.find(workbook=>workbook.id===group.workbookId)?.title
        return <span key={group.workbookId||'unknown'}>
          {group.workbookId
            ?<a href={`/workbooks/${group.workbookId}`}>{title??'워크북'} {group.count.toLocaleString()}건 열기</a>
            :<em>워크북을 알 수 없는 편집 {group.count.toLocaleString()}건</em>}
          <button className="link-button" onClick={()=>void discardStranded(group.workbookId)}>버리기</button>
        </span>
      })}</div>
    </section>}
    <section className="home-title"><div><span className="eyebrow">WORKSPACE</span><h1>좋은 아침이에요.</h1><p>오늘도 데이터에서 더 나은 답을 만들어 보세요.</p></div><div className="home-title-actions"><input ref={inputRef} type="file" hidden accept=".csv,.tsv,.xlsx" onChange={event=>chooseImport(event.target.files?.[0])}/><button className="secondary" onClick={()=>inputRef.current?.click()}><FileUp size={18}/> 파일 가져오기</button><button className="primary" onClick={()=>create.mutate(undefined)}><Plus size={18}/> 새 워크북</button></div></section>
    <section className="quick-start"><div className="section-heading"><h2>빠른 시작</h2><button className="link-button" onClick={()=>setGalleryOpen(true)}>템플릿 갤러리{templates.length>0?` (${templates.length})`:''} <ChevronRight size={14}/></button></div><div className="template-row">
      <button className="template blank" onClick={()=>create.mutate(undefined)}><span><FilePlus2/></span><strong>빈 워크북</strong><small>새로운 데이터 작업 시작</small></button>
      {featured.map((template,index)=><button className={`template template-${index}`} key={template.id} disabled={create.isPending} onClick={()=>useTemplate(template)}><span><Grid2X2/></span><strong>{template.name}</strong><small>{template.summary}</small></button>)}
      {featured.length===0&&['월간 매출 분석','프로젝트 현황','거래명세서'].map((name,index)=><button className={`template template-${index}`} key={name} disabled><span><Grid2X2/></span><strong>{name}</strong><small>템플릿을 불러오는 중…</small></button>)}
    </div></section>
    <section className="recent-section" onContextMenu={event=>{
      if(!(event.target as HTMLElement).closest('.workbook-card')){
        event.preventDefault()
        setCardMenu({x:event.clientX,y:event.clientY,label:'워크북 목록 메뉴',items:[
          {kind:'item',label:'새 워크북',icon:<Plus/>,onSelect:()=>create.mutate(undefined)},
          {kind:'item',label:'파일 가져오기',icon:<FileUp/>,onSelect:()=>inputRef.current?.click()},
          {kind:'item',label:'휴지통 보기',icon:<Trash/>,onSelect:()=>setFilter('trash')},
        ]})
      }
    }}><div className="section-heading"><h2>최근 워크북</h2><div className="segmented"><button className={filter==='recent'?'active':''} onClick={()=>setFilter('recent')}><Clock3/> 최근</button><button className={filter==='owned'?'active':''} onClick={()=>setFilter('owned')}><Lock/> 내 소유</button><button className={filter==='shared'?'active':''} onClick={()=>setFilter('shared')}><Users/> 나와 공유됨</button><button className={filter==='favorite'?'active':''} onClick={()=>setFilter('favorite')}><Star/> 즐겨찾기</button><button className={filter==='trash'?'active':''} onClick={()=>setFilter('trash')}><Trash/> 휴지통</button></div></div>
      {filter!=='trash'&&<div className="home-search"><input aria-label="워크북 검색" type="search" value={search} placeholder="워크북 이름으로 찾기" onChange={event=>setSearch(event.target.value)}/>{workbooks.data&&<span>{query||serverFilter?`${workbooks.data.total.toLocaleString()}개 중 ${visibleWorkbooks.length.toLocaleString()}개`:`전체 ${workbooks.data.total.toLocaleString()}개 중 ${visibleWorkbooks.length.toLocaleString()}개`}</span>}</div>}
      {filter==='trash'
        ?trash.isLoading?<div className="loading-card">휴지통을 불러오는 중…</div>
          :(trash.data?.items??[]).length===0?<div className="empty-state"><Trash/><h3>휴지통이 비어 있습니다</h3><p>삭제한 워크북은 여기에서 복원하거나 완전히 지울 수 있습니다.</p></div>
          :<div className="trash-list">{(trash.data?.items??[]).map(item=><article className="trash-row" key={item.id}>
            <span className="trash-info"><strong>{item.title}</strong><small>{item.deleted_at?`${new Date(item.deleted_at).toLocaleString('ko-KR')} 삭제`:'삭제됨'}{item.deleted_by?` · ${item.deleted_by}`:''}</small></span>
            <button disabled={restore.isPending} onClick={()=>void restore.mutateAsync(item)}><RotateCcw/> 복원</button>
            <button className="danger" disabled={purge.isPending} onClick={()=>{if(confirm(`'${item.title}' 워크북을 완전히 삭제할까요? 이 작업은 되돌릴 수 없습니다.`))void purge.mutateAsync(item)}}><Trash2/> 완전 삭제</button>
          </article>)}</div>
        :workbooks.isLoading?<div className="loading-card">워크북을 불러오는 중…</div>:visibleWorkbooks.length===0?<div className="empty-state"><FilePlus2/><h3>{query?`'${query}'와 맞는 워크북이 없습니다`:filter==='favorite'?'즐겨찾기한 워크북이 없습니다':filter==='shared'?'나와 공유된 워크북이 없습니다':filter==='owned'?'내가 소유한 워크북이 없습니다':'첫 워크북을 만들어 보세요'}</h3><p>{query?'이름의 일부만 넣어도 찾습니다.':filter==='favorite'?'워크북 메뉴에서 즐겨찾기에 추가할 수 있습니다.':filter==='shared'?'동료가 사용자, 부서 또는 역할로 공유하면 여기에 표시됩니다.':'셀 편집 내용은 자동으로 안전하게 저장됩니다.'}</p></div>:<div className="workbook-grid">{visibleWorkbooks.map(workbook=><article className="workbook-card" key={workbook.id} onContextMenu={event=>{event.preventDefault();openCardMenu(workbook,event)}}>
        <a href={`/workbooks/${workbook.id}`} className="workbook-preview"><Grid2X2/><span>{workbook.sheets.length} sheets</span>{workbook.favorite&&<Star className="favorite-star" fill="currentColor"/>}</a>
        <div className="workbook-meta"><a href={`/workbooks/${workbook.id}`}><strong>{workbook.title}</strong><small>{new Date(workbook.updated_at).toLocaleString('ko-KR')} 수정{workbook.access_role&&workbook.access_role!=='owner'?` · ${userLabel(workbook.owner_id,directory)} 소유`:''}</small></a>{accessChip(workbook)}<button aria-label={`${workbook.title} 더보기`} aria-haspopup="menu" onClick={event=>{const rect=event.currentTarget.getBoundingClientRect();openCardMenu(workbook,{clientX:rect.right-8,clientY:rect.bottom})}}><MoreHorizontal/></button></div>
      </article>)}</div>}
      {workbooks.data?.has_more&&<div className="home-more"><button className="secondary" disabled={workbooks.isFetching} onClick={()=>setLimit(current=>current+WORKBOOK_PAGE)}>{workbooks.isFetching?'불러오는 중…':`더 보기 (${(workbooks.data.total-visibleWorkbooks.length).toLocaleString()}개 남음)`}</button></div>}
    </section>
    {renameTarget&&<RenameWorkbookDialog title={renameTitle} pending={update.isPending} onTitle={setRenameTitle} onClose={()=>setRenameTarget(undefined)} onSave={()=>void rename()}/>}
    {preview&&importFile&&<div className="modal-backdrop"><div className="modal import-modal"><div className="import-preview-icon"><UploadCloud/></div><h2>{importFile.name}</h2><p>{preview.format.toUpperCase()} · 비어 있지 않은 셀 {preview.total_cells.toLocaleString()}개</p><div className="import-sheet-list">{preview.sheets.map(sheet=><div key={sheet.name}><Grid2X2/><div><strong>{sheet.name}</strong><small>{sheet.rows.toLocaleString()}행 × {sheet.columns.toLocaleString()}열 · {sheet.non_empty_cells.toLocaleString()}개 셀</small></div></div>)}</div>{preview.warnings.length>0&&<div className="import-warnings">{preview.warnings.map(warning=><span key={warning}>{warning}</span>)}</div>}<div className="modal-actions"><button className="secondary" onClick={()=>{setPreview(undefined);setImportFile(undefined)}}>취소</button><button className="primary" disabled={importing} onClick={executeImport}>{importing?'가져오는 중…':'워크북으로 가져오기'}</button></div></div></div>}
    {quickOpen&&<QuickSwitcher items={quickItems} placeholder="워크북 또는 명령 검색" onClose={()=>setQuickOpen(false)} onQuery={setQuickQuery} dynamicItems={typed=>{
      // 받아 온 페이지에 없는 워크북은 이름으로 검색해 닿게 한다. 목록에
      // 이미 보이는 것과 겹치지 않도록 같은 제목은 내보내지 않는다.
      const needle=typed.trim()
      if(needle.length<2)return []
      const shown=new Set(visibleWorkbooks.map(workbook=>workbook.title))
      return (searchResults.data?.items??[])
        .filter(workbook=>!shown.has(workbook.title))
        .map(workbook=>({id:`found:${workbook.id}`,group:'워크북 검색',label:workbook.title,
          hint:'목록에 없는 워크북',icon:<Grid2X2/>,run:()=>{window.location.href=`/workbooks/${workbook.id}`}}))
    }}/>}
    {galleryOpen&&<TemplateGallery onClose={()=>setGalleryOpen(false)} onCreate={useTemplate} pending={create.isPending?create.variables?.templateId:undefined}/>}
    {cardMenu&&<ContextMenu x={cardMenu.x} y={cardMenu.y} items={cardMenu.items} label={cardMenu.label} onClose={()=>setCardMenu(undefined)}/>}
    {shareTarget&&<ShareDialog workbook={shareTarget} onClose={()=>setShareTarget(undefined)} onChanged={()=>{void client.invalidateQueries({queryKey:['workbooks']})}}/>}
  </main></div>
}
