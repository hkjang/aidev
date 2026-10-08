import { useMemo } from 'react'
import { Stethoscope, AlertTriangle, Check } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { inspectTable, type TableFinding } from '../lib/inspectTable'
import type { Cell } from '../types'
import type { GridRegion } from '../lib/dataRegion'
import { address } from '../lib/api'
import './InspectDialog.css'

export function InspectDialog({cells,region,headerRows,onClose,onFix}:{
  cells:Map<string,Cell>;region:GridRegion;headerRows:number
  onClose:()=>void;onFix:(kind:TableFinding['kind'])=>void
}){
  const dialog=useDialog<HTMLElement>(onClose)
  const findings=useMemo(()=>inspectTable(cells,region,headerRows),[cells,region,headerRows])
  const label=`${address(region.startRow,region.startColumn)}:${address(region.endRow,region.endColumn)}`

  return <div className="modal-backdrop"><div className="modal inspect-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="표 검사">
    <header><div><Stethoscope/><div><h2>표 검사</h2><p>{label} 범위에서 고칠 만한 것을 세어 봅니다. 검사는 아무것도 바꾸지 않습니다.</p></div></div><button aria-label="표 검사 닫기" onClick={onClose}>×</button></header>
    <div className="inspect-body">
      {findings.length===0
        ?<p className="inspect-clean"><Check/> 고칠 것을 찾지 못했습니다.</p>
        :<ul>{findings.map(finding=><li key={finding.kind} className={finding.changesTotals?'weighty':''}>
          <div className="inspect-head">
            {finding.changesTotals&&<AlertTriangle aria-label="셈이 달라짐"/>}
            <strong>{finding.title}</strong>
            <b>{finding.count.toLocaleString()}{finding.kind==='duplicates'||finding.kind==='subtotals'?'행':finding.kind==='unmerge'?'곳':'칸'}</b>
            <button className="secondary" onClick={()=>onFix(finding.kind)}>고치기…</button>
          </div>
          <p>{finding.detail}</p>
        </li>)}</ul>}
      {findings.some(finding=>finding.changesTotals)&&
        <p className="inspect-note">△ 표시는 고치면 <b>셈이 달라지는</b> 것입니다. 나머지는 모양만 다듬습니다.</p>}
    </div>
    <div className="modal-actions"><span/><button className="primary" onClick={onClose}>닫기</button></div>
  </div></div>
}
