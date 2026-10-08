import type { OutboxOperation } from './outbox'

/**
 * 다섯 번 밀어 보고도 안 되면 3초마다 영원히 다시 붙지 않는다. 서버가 계속
 * 500 을 내거나 굳어 버린 기준 버전 때문에 409 가 되는 작업은 다시 보낸다고
 * 낫지 않는다. 조용히 되풀이하는 대신 멈추고 사람에게 알린다.
 */
export const MAX_ATTEMPTS=5

export function isBlocked(operation:OutboxOperation){return operation.attempts>=MAX_ATTEMPTS}

/**
 * 큐는 순서가 뜻을 가진다. 앞선 변경을 건너뛰고 뒤엣것을 적용하면 같은 셀이
 * 중간 값으로 남는다. 그래서 막힌 작업은 그 시트의 뒤엣것까지 함께 붙잡는다.
 * 다른 시트는 붙잡지 않는다 — 남의 워크북에서 막힌 하나가 여기까지 세울 이유가 없다.
 */
export function sendable(operations:OutboxOperation[]){
  const stalled=new Set<string>(),ready:OutboxOperation[]=[]
  for(const operation of operations){
    if(stalled.has(operation.sheetId))continue
    if(isBlocked(operation)){stalled.add(operation.sheetId);continue}
    ready.push(operation)
  }
  return ready
}

export function blocked(operations:OutboxOperation[]){return operations.filter(isBlocked)}
