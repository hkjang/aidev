import { CloudOff } from 'lucide-react'

/**
 * 저장 큐가 다섯 번 밀어 보고도 안 되면 더 보내지 않는다. 예전에는 3초마다
 * 영원히 다시 붙었고, 화면은 계속 "저장 중" 이라고만 했다. 멈춘 것을 말하지
 * 않으면 사람은 저장된 줄 알고 창을 닫는다. 다시 시도할지 버릴지는 사람이 정한다.
 */
export function StalledSaveNotice({count,onRetry,onDiscard}:{count:number;onRetry:()=>void;onDiscard:()=>void}){
  if(count===0)return null
  return <div className="formula-issue stalled-save" role="alert">
    <CloudOff/>
    <div>
      <strong>변경 {count.toLocaleString()}건을 저장하지 못했습니다</strong>
      <small>여러 번 다시 보냈지만 서버가 받지 않아 멈췄습니다. 버리면 이 변경은 사라지고 서버에 있는 값이 남습니다.</small>
    </div>
    <button className="link-button" onClick={onRetry}>다시 시도</button>
    <button className="link-button" onClick={onDiscard}>버리기</button>
  </div>
}
