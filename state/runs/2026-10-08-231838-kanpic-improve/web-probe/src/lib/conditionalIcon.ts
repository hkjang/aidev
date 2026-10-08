import type { ConditionalIcon,ConditionalIconStyle } from '../types'

// 아이콘은 색만으로 뜻을 전하지 않는다. 신호등은 색이 다르지만 화살표는
// 방향이, 기호는 모양이, 사분원은 채운 넓이가 같은 말을 한 번 더 한다.
// 색맹인 사람이나 흑백으로 인쇄한 종이에서도 순서를 읽을 수 있어야 한다.
export type IconGlyph =
  |{kind:'circle';color:string}
  |{kind:'arrow';color:string;angle:number}
  |{kind:'symbol';color:string;glyph:'check'|'bang'|'cross'}
  |{kind:'quarter';color:string;filled:number}

const RED='#dc2626',AMBER='#d97706',GREEN='#16a34a',SLATE='#475569'

// 낮은 아이콘부터 차례로 적는다. 서버가 준 index 가 곧 이 배열의 자리다.
const CATALOG:Record<ConditionalIconStyle,IconGlyph[]>={
  '3TrafficLights1':[{kind:'circle',color:RED},{kind:'circle',color:AMBER},{kind:'circle',color:GREEN}],
  '3Arrows':[{kind:'arrow',color:RED,angle:90},{kind:'arrow',color:AMBER,angle:0},{kind:'arrow',color:GREEN,angle:-90}],
  '3Symbols':[{kind:'symbol',color:RED,glyph:'cross'},{kind:'symbol',color:AMBER,glyph:'bang'},{kind:'symbol',color:GREEN,glyph:'check'}],
  '4Arrows':[{kind:'arrow',color:RED,angle:90},{kind:'arrow',color:AMBER,angle:45},{kind:'arrow',color:AMBER,angle:-45},{kind:'arrow',color:GREEN,angle:-90}],
  '5Arrows':[{kind:'arrow',color:RED,angle:90},{kind:'arrow',color:AMBER,angle:45},{kind:'arrow',color:SLATE,angle:0},{kind:'arrow',color:AMBER,angle:-45},{kind:'arrow',color:GREEN,angle:-90}],
  '5Quarters':[{kind:'quarter',color:SLATE,filled:0},{kind:'quarter',color:SLATE,filled:.25},{kind:'quarter',color:SLATE,filled:.5},{kind:'quarter',color:SLATE,filled:.75},{kind:'quarter',color:SLATE,filled:1}],
}

export const ICON_STYLES=Object.keys(CATALOG) as ConditionalIconStyle[]

export const ICON_STYLE_LABELS:Record<ConditionalIconStyle,string>={
  '3TrafficLights1':'신호등 3개','3Arrows':'화살표 3개','3Symbols':'기호 3개',
  '4Arrows':'화살표 4개','5Arrows':'화살표 5개','5Quarters':'사분원 5개',
}

export function iconGlyph(icon:ConditionalIcon):IconGlyph|null{
  const set=CATALOG[icon.style]
  if(!set)return null
  const index=Math.min(Math.max(Math.round(icon.index),0),set.length-1)
  return set[index]
}

export function iconSetPreview(style:ConditionalIconStyle):IconGlyph[]{return CATALOG[style]??[]}

// 아이콘을 그린 칸은 글자를 그만큼 오른쪽으로 민다. 값을 가리면 아이콘이
// 도움이 아니라 방해가 된다.
export function iconGutter(size:number):number{return Math.round(size)+6}

export function drawConditionalIcon(context:CanvasRenderingContext2D,glyph:IconGlyph,x:number,y:number,size:number){
  const half=size/2,cx=x+half,cy=y+half
  context.save()
  context.fillStyle=glyph.color
  context.strokeStyle=glyph.color
  context.lineWidth=Math.max(1.4,size*.14)
  context.lineCap='round'
  context.lineJoin='round'
  if(glyph.kind==='circle'){
    context.beginPath()
    context.arc(cx,cy,half*.82,0,Math.PI*2)
    context.fill()
  }else if(glyph.kind==='arrow'){
    context.translate(cx,cy)
    context.rotate(glyph.angle*Math.PI/180)
    const arm=half*.78
    context.beginPath()
    context.moveTo(-arm,0)
    context.lineTo(arm,0)
    context.stroke()
    context.beginPath()
    context.moveTo(arm,0)
    context.lineTo(arm-half*.6,-half*.5)
    context.lineTo(arm-half*.6,half*.5)
    context.closePath()
    context.fill()
  }else if(glyph.kind==='symbol'){
    const arm=half*.6
    context.beginPath()
    if(glyph.glyph==='check'){
      context.moveTo(cx-arm,cy)
      context.lineTo(cx-arm*.2,cy+arm*.7)
      context.lineTo(cx+arm,cy-arm*.7)
      context.stroke()
    }else if(glyph.glyph==='cross'){
      context.moveTo(cx-arm,cy-arm)
      context.lineTo(cx+arm,cy+arm)
      context.moveTo(cx+arm,cy-arm)
      context.lineTo(cx-arm,cy+arm)
      context.stroke()
    }else{
      context.moveTo(cx,cy-arm)
      context.lineTo(cx,cy+arm*.3)
      context.stroke()
      context.beginPath()
      context.arc(cx,cy+arm*.8,Math.max(1,size*.09),0,Math.PI*2)
      context.fill()
    }
  }else{
    context.beginPath()
    context.arc(cx,cy,half*.82,0,Math.PI*2)
    context.stroke()
    if(glyph.filled>0){
      context.beginPath()
      context.moveTo(cx,cy)
      context.arc(cx,cy,half*.82,-Math.PI/2,-Math.PI/2+Math.PI*2*glyph.filled)
      context.closePath()
      context.fill()
    }
  }
  context.restore()
}
