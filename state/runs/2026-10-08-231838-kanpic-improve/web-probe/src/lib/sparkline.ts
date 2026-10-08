export type SparklineSpec={
  chart:'line'|'column'|'bar'|'winloss'
  values:number[]
  color:string
  negativeColor?:string
  highColor?:string
  axis?:boolean
  max?:number
  min?:number
  lineWidth?:number
}

/**
 * Recognises the value a SPARKLINE formula produces. Everything else in a cell
 * is text or a number, so the marker is what tells the grid to draw instead of
 * writing.
 */
export function parseSparkline(value:unknown):SparklineSpec|undefined{
  if(!value||typeof value!=='object'||Array.isArray(value))return undefined
  const record=value as Record<string,unknown>
  if(record.kanpic!=='sparkline')return undefined
  const values=Array.isArray(record.values)?record.values.filter((item):item is number=>typeof item==='number'&&Number.isFinite(item)):[]
  if(values.length===0)return undefined
  const chart=record.chart==='column'||record.chart==='bar'||record.chart==='winloss'?record.chart:'line'
  return {
    chart,values,
    color:typeof record.color==='string'&&record.color?record.color:'#0f766e',
    negativeColor:typeof record.negativeColor==='string'?record.negativeColor:undefined,
    highColor:typeof record.highColor==='string'?record.highColor:undefined,
    axis:record.axis===true,
    max:typeof record.max==='number'?record.max:undefined,
    min:typeof record.min==='number'?record.min:undefined,
    lineWidth:typeof record.lineWidth==='number'?record.lineWidth:undefined,
  }
}

/**
 * Draws the chart inside the cell rectangle. The scale covers the values
 * unless the formula pinned it, and zero is kept on the axis so negative
 * values read as negative.
 */
export function drawSparkline(context:CanvasRenderingContext2D,spec:SparklineSpec,x:number,y:number,width:number,height:number,zoom=1){
  const padding=Math.max(2,3*zoom)
  const left=x+padding,top=y+padding,plotWidth=Math.max(1,width-padding*2),plotHeight=Math.max(1,height-padding*2)
  const values=spec.values
  const highest=spec.max??Math.max(...values)
  const lowest=spec.min??Math.min(...values)
  // A flat series still needs a range, and a series that never crosses zero
  // keeps the baseline at its own low so the bars fill the cell.
  const top_=Math.max(highest,spec.chart==='line'?highest:Math.max(highest,0))
  const bottom=Math.min(lowest,spec.chart==='line'?lowest:Math.min(lowest,0))
  const span=top_-bottom||Math.abs(top_)||1
  const positionY=(value:number)=>top+plotHeight-((value-bottom)/span)*plotHeight
  context.save()
  context.beginPath();context.rect(x+1,y+1,Math.max(0,width-2),Math.max(0,height-2));context.clip()
  if(spec.chart==='line'){
    context.strokeStyle=spec.color
    context.lineWidth=(spec.lineWidth??1.5)*zoom
    context.lineJoin='round'
    context.beginPath()
    values.forEach((value,index)=>{
      const pointX=values.length===1?left+plotWidth/2:left+(index/(values.length-1))*plotWidth
      const pointY=positionY(value)
      if(index===0)context.moveTo(pointX,pointY)
      else context.lineTo(pointX,pointY)
    })
    context.stroke()
  }else if(spec.chart==='bar'){
    // One stacked bar: each value takes a share of the width.
    const total=values.reduce((sum,value)=>sum+Math.abs(value),0)||1
    let offset=left
    values.forEach((value,index)=>{
      const size=Math.abs(value)/total*plotWidth
      context.fillStyle=index%2===0?spec.color:(spec.negativeColor??shade(spec.color))
      context.fillRect(offset,top+plotHeight*0.2,Math.max(0.5,size),Math.max(1,plotHeight*0.6))
      offset+=size
    })
  }else{
    const gap=values.length>40?0:Math.max(1,plotWidth/values.length*0.2)
    const barWidth=Math.max(1,(plotWidth-gap*(values.length-1))/values.length)
    const zeroY=positionY(spec.chart==='winloss'?0:Math.max(bottom,Math.min(0,top_)))
    values.forEach((value,index)=>{
      const barX=left+index*(barWidth+gap)
      const negative=value<0
      context.fillStyle=negative?(spec.negativeColor??'#c2413b'):(spec.highColor&&value===Math.max(...values)?spec.highColor:spec.color)
      if(spec.chart==='winloss'){
        const half=Math.max(1,plotHeight*0.38)
        context.fillRect(barX,negative?top+plotHeight/2:top+plotHeight/2-half,barWidth,half)
        return
      }
      const barY=positionY(value)
      context.fillRect(barX,Math.min(barY,zeroY),barWidth,Math.max(1,Math.abs(zeroY-barY)))
    })
  }
  if(spec.axis&&bottom<0&&top_>0){
    context.strokeStyle='#b9c3c8';context.lineWidth=1
    const axisY=Math.round(positionY(0))+0.5
    context.beginPath();context.moveTo(left,axisY);context.lineTo(left+plotWidth,axisY);context.stroke()
  }
  context.restore()
}

/** A dimmer partner colour for the alternating segments of a stacked bar. */
function shade(color:string){
  return color.length===7?color+'99':color
}

/**
 * Describes the chart for people who cannot see it. A drawing in a cell is
 * invisible to a screen reader, so the shape is put into words.
 */
export function describeSparkline(spec:SparklineSpec){
  const kind=spec.chart==='line'?'선형':spec.chart==='column'?'막대':spec.chart==='bar'?'누적 막대':'승패'
  const first=spec.values[0],last=spec.values[spec.values.length-1]
  const direction=last>first?'상승':last<first?'하락':'변화 없음'
  const round=(value:number)=>value.toLocaleString('ko-KR',{maximumFractionDigits:2})
  return `${kind} 미니 차트, 값 ${spec.values.length}개, ${round(Math.min(...spec.values))}부터 ${round(Math.max(...spec.values))}까지, ${direction}`
}
