import { BarChart3,CornerDownRight,Plus,Settings2 } from 'lucide-react'
import type { Chart,Sheet } from '../types'
import './ChartPanel.css'

const typeLabels:Record<Chart['type'],string>={bar:'막대',line:'선',area:'영역',pie:'원형',scatter:'분산',histogram:'히스토그램',stacked_bar:'누적 막대',stacked_area:'누적 영역',combo:'혼합',timeline:'일정표',waterfall:'폭포'}
export function ChartPanel({charts,sheets,onClose,onCreate,onEdit,onNavigate}:{charts:Chart[];sheets:Sheet[];onClose:()=>void;onCreate:()=>void;onEdit:(chart:Chart)=>void;onNavigate:(chart:Chart)=>void}){
  return <aside className="chart-panel"><header><span><BarChart3/> 차트</span><button aria-label="차트 패널 닫기" onClick={onClose}>×</button></header><div className="chart-panel-intro"><p>셀 범위를 시각화하고 시트 위에서 자유롭게 이동하거나 크기를 조절합니다.</p><button className="primary" onClick={onCreate}><Plus/> 새 차트</button></div><div className="chart-panel-list">{charts.length===0?<div className="chart-panel-empty"><BarChart3/><strong>이 시트에 차트가 없습니다</strong><span>범위를 선택한 뒤 새 차트를 만드세요.</span></div>:charts.map(chart=><article key={chart.id}><span className={`chart-type chart-type-${chart.type}`}><BarChart3/></span><div><strong>{chart.title||`${typeLabels[chart.type]} 차트`}</strong><small>{sheets.find(sheet=>sheet.id===chart.source_sheet_id)?.name??'삭제된 시트'}!{chart.source_range} · r{chart.revision}</small></div><button title="원본 범위로 이동" onClick={()=>onNavigate(chart)}><CornerDownRight/></button><button title="차트 설정" onClick={()=>onEdit(chart)}><Settings2/></button></article>)}</div></aside>
}
