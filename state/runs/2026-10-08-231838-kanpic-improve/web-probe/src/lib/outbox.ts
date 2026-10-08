import { blocked, isBlocked, sendable } from './outboxQueue'

export type OutboxOperation = {
  id: string
  sheetId: string
  /** 어느 워크북의 편집인지. 목록 화면은 시트만으로는 워크북을 찾을 수 없다.
   *  이 칸이 생기기 전에 쌓인 작업에는 없을 수 있다. */
  workbookId?: string
  endpoint?: 'batch'|'paste'|'fill'|'format'|'note'|'merge'|'unmerge'|'sort'
  body: Record<string, unknown>
  attempts: number
  createdAt: number
}

const databaseName = 'kanpic-local'
const storeName = 'outbox'

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(databaseName, 1)
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(storeName)) request.result.createObjectStore(storeName, { keyPath: 'id' })
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

async function transaction<T>(mode: IDBTransactionMode, callback: (store: IDBObjectStore, resolve: (value:T)=>void, reject:(reason?:unknown)=>void)=>void) {
  const db = await openDatabase()
  return new Promise<T>((resolve, reject) => {
    const tx = db.transaction(storeName, mode)
    callback(tx.objectStore(storeName), resolve, reject)
    tx.oncomplete = () => db.close()
    tx.onerror = () => reject(tx.error)
  })
}

export function enqueue(operation: OutboxOperation) {
  return transaction<void>('readwrite', (store, resolve, reject) => { const request=store.put(operation); request.onsuccess=()=>resolve(); request.onerror=()=>reject(request.error) })
}

export function remove(id: string) {
  return transaction<void>('readwrite', (store, resolve, reject) => { const request=store.delete(id); request.onsuccess=()=>resolve(); request.onerror=()=>reject(request.error) })
}

export function listOutbox() {
  return transaction<OutboxOperation[]>('readonly', (store, resolve, reject) => { const request=store.getAll(); request.onsuccess=()=>resolve(request.result.sort((a,b)=>a.createdAt-b.createdAt)); request.onerror=()=>reject(request.error) })
}

/** 실패를 세어 두어야 포기할 때를 안다. 세지 않으면 3초마다 영원히 다시 붙는다. */
async function recordFailure(operation:OutboxOperation){
  const next={...operation,attempts:operation.attempts+1}
  await enqueue(next)
  if(isBlocked(next))window.dispatchEvent(new CustomEvent('kanpic:outbox-blocked',{detail:{operation:next}}))
}

export async function flushOutbox(onApplied?: (operation:OutboxOperation, result:unknown)=>void) {
  if (!navigator.onLine) return 0
  const items = sendable(await listOutbox())
  const stalled = new Set<string>()
  let flushed = 0
  for (const item of items) {
    // 한 시트에서 한 번 막히면 그 시트의 뒤엣것은 이번 차례에 보내지 않는다.
    // 순서를 건너뛰면 같은 셀이 중간 값으로 남는다.
    if (stalled.has(item.sheetId)) continue
    try {
      const action=item.endpoint==='paste'?'paste':item.endpoint==='fill'?'fill':'batch'
      const rangeAction=item.endpoint==='format'||item.endpoint==='note'||item.endpoint==='merge'||item.endpoint==='unmerge'||item.endpoint==='sort'
      const path=rangeAction?`/api/v1/sheets/${item.sheetId}/ranges:${item.endpoint}`:`/api/v1/sheets/${item.sheetId}/cells:${action}`
      const response = await fetch(path, { method:'PATCH', credentials:'same-origin', headers:{'Content-Type':'application/json'}, body:JSON.stringify(item.body) })
      if (!response.ok) {
        if (response.status >= 400 && response.status < 500 && response.status !== 409) {
          const payload=await response.json().catch(()=>null) as {error?:{code?:string;message?:string}}|null
          await remove(item.id)
          window.dispatchEvent(new CustomEvent('kanpic:outbox-rejected',{detail:{operation:item,status:response.status,code:payload?.error?.code??'request_rejected',message:payload?.error?.message??`요청 실패 (${response.status})`}}))
        } else {
          await recordFailure(item)
        }
        stalled.add(item.sheetId)
        continue
      }
      const result = await response.json()
      await remove(item.id)
      flushed += 1
      onApplied?.(item, result)
    } catch {
      await recordFailure(item)
      stalled.add(item.sheetId)
    }
  }
  return flushed
}

/** 더 보내지 않기로 한 작업들. 사람이 다시 시도할지 버릴지 정해야 한다. */
export async function blockedOutbox(){ return blocked(await listOutbox()) }

/** 실패 횟수를 지워 다음 차례부터 다시 보낸다. */
export async function retryOutbox(operations:OutboxOperation[]){
  for(const operation of operations)await enqueue({...operation,attempts:0})
  return operations.length
}

/** 사람이 버리기로 한 변경만 큐에서 뺀다. */
export async function discardOutbox(operations:OutboxOperation[]){
  for(const operation of operations)await remove(operation.id)
  return operations.length
}
