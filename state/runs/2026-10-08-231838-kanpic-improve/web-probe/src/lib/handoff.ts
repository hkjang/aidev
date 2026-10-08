import { api, ApiError } from './api'

// 서비스 간 문서 넘기기(HANDOFF-STANDARD)의 보내는 쪽. 사람이 파일을
// 내려받지 않는다 — 표(claim)를 발급받아 받는 쪽의 /handoff 를 새 창에서
// 열면 그쪽이 여기서 직접 받아 간다.

export type HandoffFormat = 'csv' | 'xlsx'
export type HandoffTarget = { name: string; origin: string; formats: HandoffFormat[] }
export type HandoffClaim = { claim: string; source: string; filename: string; content_type: string; bytes: number; expires_at: string }

export const handoffFormatLabel: Record<HandoffFormat, string> = { csv: '현재 시트 CSV', xlsx: 'XLSX' }

/** 받는 쪽이 여는 주소. source 와 claim 은 질의에 실리므로 그대로 넣지 않는다. */
export function handoffAddress(target: Pick<HandoffTarget, 'origin'>, claim: Pick<HandoffClaim, 'source' | 'claim'>) {
  const address = new URL('/handoff', target.origin)
  address.searchParams.set('source', claim.source)
  address.searchParams.set('claim', claim.claim)
  return address.toString()
}

/** 표에 적는 문서 id. 시트 하나(CSV)는 워크북 뒤에 시트를 붙인다. */
export function handoffResource(workbookId: string, format: HandoffFormat, sheetId?: string) {
  return format === 'csv' && sheetId ? `${workbookId}/sheets/${sheetId}` : workbookId
}

/** 넘겨받은 워크북의 출처. service 는 허용 목록의 이름이라 그 사이 목록에서 빠졌으면 없다. */
export type HandoffOrigin = { workbook_id: string; source: string; service?: string; filename: string; received_by: string; received_at: string }

/** 어디서 왔는지 묻는다. 넘겨받은 워크북이 아니면(404) null 이다 — 대부분의 워크북이 그렇다. */
export async function fetchHandoffOrigin(workbookId: string) {
  try {
    return await api<HandoffOrigin>(`/api/v1/workbooks/${workbookId}/handoff`)
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null
    throw error
  }
}

/** 편집기 제목 아래 한 줄. 이름이 없으면 오리진의 호스트로 부른다. */
export function handoffOriginLabel(origin: Pick<HandoffOrigin, 'source' | 'service'>) {
  return `${handoffOriginName(origin)} 에서 받음`
}

/** 그 한 줄에 마우스를 올렸을 때: 어느 파일을 언제 받았는지. */
export function handoffOriginDetail(origin: Pick<HandoffOrigin, 'source' | 'service' | 'filename' | 'received_at'>) {
  return `${origin.source} 에서 ${origin.filename} 을(를) ${new Date(origin.received_at).toLocaleString('ko-KR')} 에 받았습니다.`
}

function handoffOriginName(origin: Pick<HandoffOrigin, 'source' | 'service'>) {
  if (origin.service) return origin.service
  try {
    return new URL(origin.source).host
  } catch {
    return origin.source
  }
}

/**
 * 표를 발급받아 새 창을 받는 쪽으로 보낸다. 창은 누르는 순간에 미리 열어
 * 둔다 — 표를 기다린 뒤에 열면 팝업 차단에 걸린다. 표를 받지 못하면 그
 * 창을 닫고 까닭을 던진다.
 */
export async function sendHandoff(target: HandoffTarget, resource: string, format: HandoffFormat, open: (address?: string) => { location: { href: string }; close: () => void } | null = address => window.open(address, '_blank')) {
  const popup = open()
  try {
    const issued = await api<HandoffClaim>('/api/v1/handoff/claims', { method: 'POST', body: JSON.stringify({ resource, format }) })
    const address = handoffAddress(target, issued)
    if (popup) popup.location.href = address
    else open(address)
    return issued
  } catch (error) {
    popup?.close()
    throw error
  }
}
