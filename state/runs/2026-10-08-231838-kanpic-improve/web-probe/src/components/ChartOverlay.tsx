import { useQuery } from '@tanstack/react-query'
import { Download,ImageDown,Maximize2,Settings2,Table2,Trash2 } from 'lucide-react'
import { useEffect,useRef,useState } from 'react'
import { createPortal } from 'react-dom'
import { api } from '../lib/api'
import { spreadsheetDate } from '../lib/cellFormat'
import type { Chart,ChartData,ChartPosition,ChartSeries } from '../types'
import './ChartOverlay.css'
import { ContextMenu,type MenuItem } from './ContextMenu'
import { chartShape,shapeExtent,stackedBases } from '../lib/chartLayout'

const palette=['#0f766e','#5268a6','#d97706','#9333ea','#dc4f4f','#0891b2','#65a30d','#db2777']
const safeFileName=(value:string)=>value.trim().replace(/[\\/:*?"<>|]+/g,'-')||'kanpic-chart'
const finiteValues=(series:ChartSeries[])=>series.flatMap(item=>item.points.flatMap(point=>point.value==null?[]:[point.value]))
const chartExtents=(series:ChartSeries[])=>{const values=finiteValues(series);if(values.length===0)return{min:0,max:1};let min=Math.min(0,...values),max=Math.max(0,...values);if(min===max){min-=1;max+=1}return{min,max}}
const pointY=(value:number,min:number,max:number,top:number,height:number)=>top+(max-value)/(max-min)*height
const pointX=(index:number,count:number,left:number,width:number)=>left+(count<=1?width/2:index*width/(count-1))
const pieArc=(cx:number,cy:number,r:number,start:number,end:number)=>{const sx=cx+r*Math.cos(start),sy=cy+r*Math.sin(start),ex=cx+r*Math.cos(end),ey=cy+r*Math.sin(end),large=end-start>Math.PI?1:0;return `M${cx},${cy} L${sx},${sy} A${r},${r} 0 ${large} 1 ${ex},${ey} Z`}

/** 축과 값에 적는 수. 다른 차트가 쓰는 표기를 그대로 따른다. */
const formatTick=(value:number)=>String(Number(value.toPrecision(4)))

export function ChartPlot({data}: {data:ChartData}) {
  const chart=data.chart,series=data.series,width=520,height=280,left=chart.legend_position==='left'?125:52,right=chart.legend_position==='right'?112:24,top=chart.legend_position==='top'?42:20,bottom=chart.legend_position==='bottom'?62:48,plotWidth=width-left-right,plotHeight=height-top-bottom
  // Which series are bars and which are lines depends on the chart kind, and a
  // combination chart may put its line on a scale of its own.
  const shape=chartShape(chart.type,series,chart.secondary_axis===true)
  const extents=shape.bars.length>0||shape.lines.length>0
    ?shapeExtent(shape.secondary?shape.bars:[...shape.bars,...shape.lines],shape.stacked,{min:chart.y_axis_min,max:chart.y_axis_max})
    :chartExtents(series)
  const lineExtent=shape.secondary?shapeExtent(shape.lines,false):extents
  const categories=series[0]?.points.map(point=>point.category)??[],ticks=[0,.25,.5,.75,1].map(portion=>extents.min+(extents.max-extents.min)*portion)
  const legend=chart.legend_position!=='none'&&series.map((item,index)=><g key={item.name} transform={chart.legend_position==='right'?`translate(${width-right+12},${top+index*18})`:chart.legend_position==='left'?`translate(8,${top+index*18})`:chart.legend_position==='bottom'?`translate(${left+index*95},${height-20})`:`translate(${left+index*95},14)`}><rect width="9" height="9" rx="2" fill={palette[index%palette.length]}/><text x="14" y="8" fontSize="9" fill="#53636d">{item.name.slice(0,14)}</text></g>)
  if(data.warning||series.length===0||finiteValues(series).length===0)return <svg className="chart-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={chart.title||'차트'}><title>{chart.title||'차트'}</title><rect width={width} height={height} fill="white"/><text x={width/2} y={height/2} textAnchor="middle" fill="#89969e" fontSize="12">{data.warning||'표시할 숫자 데이터가 없습니다.'}</text></svg>
  if(chart.type==='pie'){
    const points=series[0].points.filter(point=>point.value!=null&&point.value>=0),total=points.reduce((sum,point)=>sum+(point.value??0),0),vertical=chart.legend_position==='left'||chart.legend_position==='right',cx=chart.legend_position==='left'?355:chart.legend_position==='right'?165:260,cy=chart.legend_position==='top'?165:chart.legend_position==='bottom'?115:140,r=90;let angle=-Math.PI/2
    const pieLegend=chart.legend_position==='none'?null:points.map((point,index)=>{const x=chart.legend_position==='left'?18:chart.legend_position==='right'?330:65+(index%5)*90,y=vertical?55+index*18:chart.legend_position==='top'?18+Math.floor(index/5)*16:245+Math.floor(index/5)*16;return <g key={`legend-${point.category}-${index}`} transform={`translate(${x},${y})`}><rect width="9" height="9" rx="2" fill={palette[index%palette.length]}/><text x="13" y="8" fontSize="8" fill="#53636d">{point.category.slice(0,12)} · {point.value}</text></g>})
    return <svg className="chart-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={chart.title||'원형 차트'}><title>{chart.title||'원형 차트'}</title><rect width={width} height={height} fill="white"/>{total>0?points.map((point,index)=>{const start=angle,end=angle+(point.value??0)/total*Math.PI*2;angle=end;return <path key={`${point.category}-${index}`} d={pieArc(cx,cy,r,start,end)} fill={palette[index%palette.length]} stroke="white" strokeWidth="2"><title>{point.category}: {point.value}</title></path>}):null}{pieLegend}</svg>
  }
  if(chart.type==='waterfall'){
    // 폭포는 앞줄까지의 누계 위에 다음 막대를 얹어, 증감이 쌓여 결과에
    // 이르는 길을 보여 준다. 막대의 아래위가 곧 그 줄의 시작과 끝이다.
    const steps=series[0].points.filter(point=>point.x!=null&&point.value!=null)
    const levels=steps.flatMap(point=>[point.x as number,point.value as number]).concat(0)
    const low=Math.min(...levels),high=Math.max(...levels)
    // 모든 값이 같으면 높이가 0 이 되어 아무것도 안 보인다.
    const span=high>low?high-low:Math.abs(high)||1
    const bottom0=high>low?low:low-span/2
    const atValue=(value:number)=>top+plotHeight-((value-bottom0)/span)*plotHeight
    const slot=plotWidth/Math.max(1,steps.length)
    const barWidth=Math.max(3,Math.min(46,slot*0.62))
    const marks=[0,.25,.5,.75,1].map(portion=>bottom0+span*portion)
    // 늘어난 것과 줄어든 것을 색으로 가른다. 폭포에서 방향은 값 자체만큼
    // 중요하다. 합계 줄은 증감이 아니므로 또 다른 색으로 둔다.
    const paint=(point:typeof steps[number])=>point.total?'#3b6ea5':(point.value as number)>=(point.x as number)?'#2f9e5f':'#d1495b'
    return <svg className="chart-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={chart.title||'폭포 차트'}><title>{chart.title||'폭포 차트'}</title>
      <rect width={width} height={height} fill="white"/>
      {marks.map(mark=><g key={mark}><line x1={left} x2={width-right} y1={atValue(mark)} y2={atValue(mark)} stroke="#e6eaee"/><text x={left-6} y={atValue(mark)+3} textAnchor="end" className="chart-axis">{formatTick(mark)}</text></g>)}
      {low<0&&high>0&&<line x1={left} x2={width-right} y1={atValue(0)} y2={atValue(0)} stroke="#b7c0c7"/>}
      {steps.map((point,index)=>{
        const start=point.x as number,end=point.value as number
        const x=left+index*slot+(slot-barWidth)/2
        const barTop=atValue(Math.max(start,end)),barBottom=atValue(Math.min(start,end))
        const previous=index>0?steps[index-1]:undefined
        return <g key={`${point.category}-${index}`}>
          {previous&&<line x1={left+(index-1)*slot+(slot+barWidth)/2} x2={x} y1={atValue(previous.value as number)} y2={atValue(point.total?end:start)} stroke="#c3ccd3" strokeDasharray="2 2"/>}
          <rect x={x} y={barTop} width={barWidth} height={Math.max(1,barBottom-barTop)} fill={paint(point)} rx="1"><title>{point.category}: {formatTick(end-start)}</title></rect>
          {chart.data_labels&&<text x={x+barWidth/2} y={barTop-4} textAnchor="middle" className="chart-axis">{formatTick(point.total?end:end-start)}</text>}
          <text x={x+barWidth/2} y={height-bottom+14} textAnchor="middle" className="chart-axis">{point.category.length>7?`${point.category.slice(0,6)}…`:point.category}</text>
        </g>
      })}
    </svg>
  }
  if(chart.type==='timeline'){
    // 일정표는 한 줄에 하나씩 가로 막대로 그린다. 시작과 끝이 곧 막대의
    // 양 끝이므로 세로축은 눈금이 아니라 일감의 이름이다.
    const tasks=series[0].points.filter(point=>point.x!=null&&point.value!=null)
    const starts=tasks.map(point=>point.x as number),ends=tasks.map(point=>point.value as number)
    const from=Math.min(...starts),toRaw=Math.max(...ends)
    // 하루짜리 일감만 있으면 폭이 0 이 되어 아무것도 안 보인다.
    const to=toRaw>from?toRaw:from+1
    // 일감이 많으면 한 줄에 줄 수 있는 높이가 줄어든다. 6픽셀보다 얇아지면
    // 막대도 이름도 읽을 수 없으므로, 그릴 수 있는 만큼만 그리고 몇 개가
    // 남았는지 적는다. 넘쳐 흘러 그림 밖으로 나가는 것보다 낫다.
    const rowHeight=Math.min(26,plotHeight/Math.max(1,tasks.length))
    const shown=rowHeight>=6?tasks.length:Math.max(1,Math.floor(plotHeight/6))
    const drawHeight=rowHeight>=6?rowHeight:6
    const hidden=tasks.length-shown
    const timelineLeft=Math.max(left,96)
    const timelineWidth=width-right-timelineLeft
    const atDate=(value:number)=>timelineLeft+((value-from)/(to-from))*timelineWidth
    const ticks=[0,.5,1].map(portion=>from+(to-from)*portion)
    return <svg className="chart-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={chart.title||'일정표'}><title>{chart.title||'일정표'}</title>
      {chart.title&&<text x={width/2} y={16} textAnchor="middle" className="chart-title">{chart.title}</text>}
      {ticks.map((value,index)=><g key={`tick-${index}`}>
        <line x1={atDate(value)} y1={top} x2={atDate(value)} y2={top+plotHeight} stroke="#e4e8ec"/>
        <text x={atDate(value)} y={top+plotHeight+14} textAnchor={index===0?'start':index===ticks.length-1?'end':'middle'} className="chart-axis">{dateText(value)}</text>
      </g>)}
      {hidden>0&&<text x={width-right} y={top-6} textAnchor="end" className="chart-axis">외 {hidden}개 더</text>}
      {tasks.slice(0,shown).map((point,index)=>{
        const y=top+index*drawHeight+2,barHeight=Math.max(3,drawHeight-6)
        const x=atDate(point.x as number),end=atDate(point.value as number)
        return <g key={`${point.category}-${index}`}>
          <text x={timelineLeft-6} y={y+barHeight/2+3} textAnchor="end" className="chart-axis">{point.category.length>10?`${point.category.slice(0,9)}…`:point.category}</text>
          <rect x={x} y={y} width={Math.max(2,end-x)} height={barHeight} rx={2} fill={palette[index%palette.length]}/>
        </g>
      })}
    </svg>
  }
  const isBar=shape.bars.length>0,barGroup=plotWidth/Math.max(1,categories.length)
  const barWidth=shape.stacked?Math.max(2,Math.min(48,barGroup-8)):Math.max(2,Math.min(40,(barGroup-6)/Math.max(1,shape.bars.length)))
  const stacks=shape.stacked?stackedBases(shape.bars.length>0?shape.bars:shape.lines,categories.length):undefined
  const colourOf=(item:ChartSeries)=>palette[Math.max(0,series.indexOf(item))%palette.length]
  const xValues=chart.type==='scatter'?series.flatMap(item=>item.points.flatMap(point=>point.x==null?[]:[point.x])):[],xMin=xValues.length?Math.min(...xValues):0,xMaxRaw=xValues.length?Math.max(...xValues):1,xMax=xMaxRaw===xMin?xMin+1:xMaxRaw
  return <svg className="chart-svg" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={chart.title||`${chart.type} 차트`}><title>{chart.title||`${chart.type} 차트`}</title><rect width={width} height={height} fill="white"/>{ticks.map((tick,index)=>{const y=pointY(tick,extents.min,extents.max,top,plotHeight);return <g key={tick}><line x1={left} y1={y} x2={left+plotWidth} y2={y} stroke="#e5e9eb"/><text x={left-7} y={y+3} textAnchor="end" fill="#80909a" fontSize="8">{Number(tick.toPrecision(4))}</text>{index===0&&<line x1={left} y1={top} x2={left} y2={top+plotHeight} stroke="#b9c3c8"/>}</g>})}{isBar&&shape.bars.map((item,seriesIndex)=>item.points.map((point,index)=>{if(point.value==null)return null
      const base=stacks?stacks.bases[seriesIndex][index]:0
      const from=pointY(base,extents.min,extents.max,top,plotHeight),to=pointY(base+point.value,extents.min,extents.max,top,plotHeight)
      const x=shape.stacked?left+index*barGroup+(barGroup-barWidth)/2:left+index*barGroup+(barGroup-barWidth*shape.bars.length)/2+seriesIndex*barWidth
      return <g key={`${item.name}-${index}`}>{chart.data_labels&&<text x={x+(barWidth-1)/2} y={Math.min(from,to)-3} textAnchor="middle" className="chart-label">{labelText(point.value)}</text>}<rect key={`${item.name}-${index}`} x={x} y={Math.min(from,to)} width={barWidth-1} height={Math.max(1,Math.abs(from-to))} rx="2" fill={colourOf(item)}><title>{point.category} · {item.name}: {point.value}</title></rect></g>}))}{shape.lines.map((item,seriesIndex)=>{
      const scale=shape.secondary?lineExtent:extents
      const heightOf=(value:number,index:number)=>pointY((shape.stacked&&stacks?stacks.bases[seriesIndex][index]:0)+value,scale.min,scale.max,top,plotHeight)
      const points=item.points.flatMap((point,index)=>point.value==null?[]:[[pointX(index,item.points.length,left,plotWidth),heightOf(point.value,index)] as [number,number]])
      const line=points.map((point,index)=>`${index?'L':'M'}${point[0]},${point[1]}`).join(' ')
      const zero=pointY(0,scale.min,scale.max,top,plotHeight)
      const area=points.length?`${line} L${points[points.length-1][0]},${zero} L${points[0][0]},${zero} Z`:''
      return <g key={item.name}>{shape.filled&&<path d={area} fill={`${colourOf(item)}33`}/>}<path d={line} fill="none" stroke={colourOf(item)} strokeWidth="2"/>{points.map((point,index)=><circle key={index} cx={point[0]} cy={point[1]} r="3" fill="white" stroke={colourOf(item)} strokeWidth="2"/>)}{chart.data_labels&&points.map((point,index)=><text key={`label-${index}`} x={point[0]} y={point[1]-6} textAnchor="middle" className="chart-label">{labelText(item.points.filter(candidate=>candidate.value!=null)[index]?.value)}</text>)}</g>})}
    {shape.secondary&&[0,.5,1].map(portion=>{const value=lineExtent.min+(lineExtent.max-lineExtent.min)*portion,y=pointY(value,lineExtent.min,lineExtent.max,top,plotHeight);return <text key={portion} x={left+plotWidth+6} y={y+3} fontSize="8" fill={colourOf(shape.lines[0])}>{Number(value.toPrecision(3))}</text>})}{chart.type==='scatter'&&series.map((item,seriesIndex)=>item.points.map((point,index)=>point.value==null||point.x==null?null:<circle key={`${item.name}-${index}`} cx={left+(point.x-xMin)/(xMax-xMin)*plotWidth} cy={pointY(point.value,extents.min,extents.max,top,plotHeight)} r="4" fill={palette[seriesIndex%palette.length]} opacity=".85"><title>{item.name}: ({point.x}, {point.value})</title></circle>))}{categories.slice(0,12).map((category,index)=>{const x=isBar?left+index*barGroup+barGroup/2:pointX(index,categories.length,left,plotWidth);return <text key={`${category}-${index}`} x={x} y={top+plotHeight+15} textAnchor="middle" fill="#74838c" fontSize="8">{category.slice(0,10)}</text>})}{chart.x_axis_title&&<text x={left+plotWidth/2} y={height-5} textAnchor="middle" fontSize="9" fill="#596871">{chart.x_axis_title}</text>}{chart.y_axis_title&&<text transform={`translate(12 ${top+plotHeight/2}) rotate(-90)`} textAnchor="middle" fontSize="9" fill="#596871">{chart.y_axis_title}</text>}{legend}</svg>
}

// 값 표시는 눈금을 짚어 읽는 수고를 덜자는 것이므로 자릿수가 길면 도리어
// 읽기 어렵다. 소수는 두 자리까지만 적는다.
function labelText(value:number|null|undefined){
  if(value==null||!Number.isFinite(value))return ''
  return Number.isInteger(value)?String(value):value.toFixed(2)
}

async function downloadChart(svg:SVGSVGElement|null,title:string,format:'svg'|'png'){
  if(!svg)return
  const source=new XMLSerializer().serializeToString(svg),blob=new Blob([source],{type:'image/svg+xml;charset=utf-8'}),name=safeFileName(title)
  if(format==='svg'){const url=URL.createObjectURL(blob),link=document.createElement('a');link.href=url;link.download=`${name}.svg`;link.click();URL.revokeObjectURL(url);return}
  const url=URL.createObjectURL(blob),image=new Image();image.onload=()=>{const canvas=document.createElement('canvas');canvas.width=1040;canvas.height=560;const context=canvas.getContext('2d');if(context){context.fillStyle='white';context.fillRect(0,0,canvas.width,canvas.height);context.drawImage(image,0,0,canvas.width,canvas.height);canvas.toBlob(output=>{if(output){const outputUrl=URL.createObjectURL(output),link=document.createElement('a');link.href=outputUrl;link.download=`${name}.png`;link.click();URL.revokeObjectURL(outputUrl)}},'image/png')}URL.revokeObjectURL(url)};image.src=url
}

function ChartCard({chart,version,onEdit,onUpdate,onDelete,onNavigate}:{chart:Chart;version:number;onEdit:(chart:Chart)=>void;onUpdate:(chart:Chart,input:Record<string,unknown>)=>Promise<Chart>;onDelete?:(chart:Chart)=>Promise<void>;onNavigate?:(chart:Chart)=>void}){
  const [position,setPosition]=useState(chart.position),svg=useRef<SVGSVGElement>(null)
  const [menu,setMenu]=useState<{x:number;y:number}>()
  const menuItems=():MenuItem[]=>[
    {kind:'label',label:chart.title||'제목 없는 차트'},
    {kind:'item',label:'차트 설정…',icon:<Settings2/>,onSelect:()=>onEdit(chart)},
    ...(onNavigate&&chart.source_sheet_id&&chart.source_range!=='#REF!'?[{kind:'item',label:`원본 데이터로 이동 (${chart.source_range})`,icon:<Table2/>,onSelect:()=>onNavigate(chart)} as MenuItem]:[]),
    {kind:'separator'},
    {kind:'item',label:'SVG로 내보내기',icon:<Download/>,onSelect:()=>downloadChart(svg.current,chart.title,'svg')},
    {kind:'item',label:'PNG로 내보내기',icon:<ImageDown/>,onSelect:()=>downloadChart(svg.current,chart.title,'png')},
    ...(onDelete?[{kind:'separator'} as MenuItem,{kind:'item',label:'차트 삭제',icon:<Trash2/>,danger:true,onSelect:()=>{
      if(window.confirm(`'${chart.title||'제목 없는 차트'}' 차트를 삭제할까요?`))void onDelete(chart).catch(error=>alert(error instanceof Error?error.message:'차트를 삭제하지 못했습니다.'))
    }} as MenuItem]:[]),
  ]
  const data=useQuery({queryKey:['chart-data',chart.id,version],queryFn:()=>api<ChartData>(`/api/v1/charts/${chart.id}/data`)})
  useEffect(()=>setPosition(chart.position),[chart.position])
  const startGesture=(kind:'move'|'resize',event:React.PointerEvent)=>{event.preventDefault();event.stopPropagation();const startX=event.clientX,startY=event.clientY,start=position;let latest=start;const move=(next:PointerEvent)=>{const dx=next.clientX-startX,dy=next.clientY-startY;latest=kind==='move'
      ?{...start,x:Math.round(Math.max(0,start.x+dx)),y:Math.round(Math.max(0,start.y+dy))}
      :{...start,width:Math.round(Math.max(240,Math.min(1600,start.width+dx))),height:Math.round(Math.max(160,Math.min(1200,start.height+dy)))};setPosition(latest)};const stop=()=>{window.removeEventListener('pointermove',move);window.removeEventListener('pointerup',stop);if(JSON.stringify(latest)!==JSON.stringify(chart.position))void onUpdate(chart,{position:latest,expected_revision:chart.revision}).catch(error=>{setPosition(chart.position);alert(error instanceof Error?error.message:'차트 배치를 저장하지 못했습니다.')})};window.addEventListener('pointermove',move);window.addEventListener('pointerup',stop,{once:true})}
  return <article className="chart-card" style={{left:position.x,top:position.y,width:position.width,height:position.height}} data-chart-id={chart.id}
    onContextMenu={event=>{event.preventDefault();event.stopPropagation();setMenu({x:event.clientX,y:event.clientY})}}>
    {menu&&<ContextMenu x={menu.x} y={menu.y} items={menuItems()} label="차트 메뉴" onClose={()=>setMenu(undefined)}/>}<header onPointerDown={event=>startGesture('move',event)}><strong>{chart.title||'제목 없는 차트'}</strong><span><button aria-label="SVG로 내보내기" title="SVG로 내보내기" onPointerDown={event=>event.stopPropagation()} onClick={()=>downloadChart(svg.current,chart.title,'svg')}><Download/></button><button aria-label="PNG로 내보내기" title="PNG로 내보내기" onPointerDown={event=>event.stopPropagation()} onClick={()=>downloadChart(svg.current,chart.title,'png')}><ImageDown/></button><button aria-label="차트 설정" title="차트 설정" onPointerDown={event=>event.stopPropagation()} onClick={()=>onEdit(chart)}><Settings2/></button></span></header><div className="chart-card-body">{data.data?<div ref={node=>{svg.current=node?.querySelector('svg')??null}}><ChartPlot data={data.data}/></div>:<span className="chart-loading">차트 데이터를 불러오는 중…</span>}</div><button className="chart-resize" aria-label="차트 크기 조정" onPointerDown={event=>startGesture('resize',event)}><Maximize2/></button></article>
}

export function ChartOverlay({charts,version,onEdit,onUpdate,onDelete,onNavigate}:{charts:Chart[];version:number;onEdit:(chart:Chart)=>void;onUpdate:(chart:Chart,input:Record<string,unknown>)=>Promise<Chart>;onDelete?:(chart:Chart)=>Promise<void>;onNavigate?:(chart:Chart)=>void}){
  const [target,setTarget]=useState<Element|null>(null)
  useEffect(()=>setTarget(document.querySelector('.sheet-area')),[charts.length])
  return target?createPortal(<div className="chart-overlay-layer" aria-label="시트 차트 레이어">{charts.map(chart=><ChartCard key={chart.id} chart={chart} version={version} onEdit={onEdit} onUpdate={onUpdate} onDelete={onDelete} onNavigate={onNavigate}/>)}</div>,target):null
}

// 일련번호를 날짜로 적는다. 날 수를 날짜로 바꾸는 셈은 lib/cellFormat.ts
// 한 곳에만 두므로 여기서도 그것을 쓴다 — 따로 세면 격자에 보이는 날짜와
// 일정표의 눈금이 어긋난다.
function dateText(value:number){
  const date=spreadsheetDate(value)
  if(!date)return String(Math.round(value))
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth()+1).padStart(2,'0')}-${String(date.getUTCDate()).padStart(2,'0')}`
}
