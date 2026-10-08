import type { OutboxOperation } from './outbox'

/**
 * 저장 큐는 지금 어떤 워크북이 열려 있는지 모른다. 밀려 있는 작업을 시트 구분 없이
 * 모두 보내기 때문에, 응답 하나가 어느 시트의 것인지 가려내지 않으면 다른 워크북의
 * 서버 버전과 작업 번호가 보고 있는 시트에 얹힌다. 그러면 Ctrl+Z 가 화면에 없는
 * 셀을 되돌린다. 보내는 것은 그대로 두고, 화면에 반영하는 쪽만 좁힌다.
 */
export function forSheet<T>(sheetId:string|undefined,apply:(result:T)=>void){
  return (operation:OutboxOperation,result:unknown)=>{
    if(!sheetId||operation.sheetId!==sheetId)return
    apply(result as T)
  }
}

/**
 * 이 워크북이 서버와 어긋나 있는지 셀 때는 이 워크북의 시트만 센다. 다른 워크북에서
 * 막힌 작업 하나가 여기의 행·열 변경까지 영원히 막아서는 안 된다.
 */
export function pendingInWorkbook(operations:OutboxOperation[],sheetIds:readonly string[]){
  const own=new Set(sheetIds)
  return operations.filter(operation=>own.has(operation.sheetId))
}
