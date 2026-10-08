import type { DimensionRange,DimensionSize } from '../types'

export type DimensionAxis=ReturnType<typeof createDimensionAxis>

function lowerBound(values:number[],target:number){let low=0,high=values.length;while(low<high){const middle=(low+high)>>1;if(values[middle]<target)low=middle+1;else high=middle}return low}
function upperBound(values:number[],target:number){let low=0,high=values.length;while(low<high){const middle=(low+high)>>1;if(values[middle]<=target)low=middle+1;else high=middle}return low}

export function mergeDimensionRanges(input:DimensionRange[],total:number):DimensionRange[]{
  const sorted=input.map(item=>({start:Math.max(1,Math.trunc(item.start)),end:Math.min(total,Math.trunc(item.end))})).filter(item=>item.start<=item.end).sort((a,b)=>a.start-b.start||a.end-b.end)
  const result:DimensionRange[]=[]
  for(const item of sorted){const previous=result.at(-1);if(!previous||item.start>previous.end+1)result.push({...item});else previous.end=Math.max(previous.end,item.end)}
  return result
}

export function createDimensionAxis({total,defaultSize,sizes=[],hiddenRanges=[],hiddenIndexes=[],zoom=1}:{total:number;defaultSize:number;sizes?:DimensionSize[];hiddenRanges?:DimensionRange[];hiddenIndexes?:number[];zoom?:number}){
  const hidden=mergeDimensionRanges([...hiddenRanges,...hiddenIndexes.map(index=>({start:index,end:index}))],total)
  const starts=hidden.map(item=>item.start),ends=hidden.map(item=>item.end),hiddenPrefix:number[]=[]
  hidden.reduce((sum,item,index)=>(hiddenPrefix[index]=sum+item.end-item.start+1,hiddenPrefix[index]),0)
  const intervalAt=(index:number)=>{const position=upperBound(starts,index)-1;return position>=0&&ends[position]>=index?position:-1}
  const isHidden=(index:number)=>index<1||index>total||intervalAt(index)>=0
  const hiddenBefore=(index:number)=>{const position=lowerBound(ends,index);const completed=position>0?hiddenPrefix[position-1]:0;if(position<hidden.length&&hidden[position].start<index)return completed+Math.min(index-1,hidden[position].end)-hidden[position].start+1;return completed}
  const custom=new Map<number,number>()
  for(const item of sizes)if(Number.isInteger(item.index)&&item.index>=1&&item.index<=total&&Number.isFinite(item.size)&&item.size>0&&!isHidden(item.index))custom.set(item.index,item.size*zoom)
  const customIndexes=Array.from(custom.keys()).sort((a,b)=>a-b),customPrefix:number[]=[]
  customIndexes.reduce((sum,index,position)=>(customPrefix[position]=sum+(custom.get(index)!-defaultSize*zoom),customPrefix[position]),0)
  const customDeltaBefore=(index:number)=>{const position=lowerBound(customIndexes,index);return position>0?customPrefix[position-1]:0}
  const scaledDefault=defaultSize*zoom
  const offsetOf=(index:number)=>{const bounded=Math.max(1,Math.min(total+1,Math.trunc(index)));return (bounded-1-hiddenBefore(bounded))*scaledDefault+customDeltaBefore(bounded)}
  const sizeOf=(index:number)=>isHidden(index)?0:custom.get(index)??scaledDefault
  const extent=offsetOf(total+1)
  const firstVisibleAtOrAfter=(raw:number):number=>{let index=Math.max(1,Math.min(total,Math.trunc(raw)));const interval=intervalAt(index);if(interval>=0)index=hidden[interval].end+1;return index<=total?index:lastVisibleAtOrBefore(total)}
  const lastVisibleAtOrBefore=(raw:number):number=>{let index=Math.max(1,Math.min(total,Math.trunc(raw)));const interval=intervalAt(index);if(interval>=0)index=hidden[interval].start-1;return index>=1?index:firstVisibleAtOrAfter(1)}
  const nextVisible=(index:number,direction:1|-1)=>direction===1?firstVisibleAtOrAfter(Math.min(total,index+1)):lastVisibleAtOrBefore(Math.max(1,index-1))
  const indexAtOffset=(raw:number)=>{const target=Math.max(0,Math.min(Math.max(0,extent-.0001),raw));let low=1,high=total;while(low<high){const middle=Math.floor((low+high)/2);if(offsetOf(middle+1)<=target)low=middle+1;else high=middle}return firstVisibleAtOrAfter(low)}
  const countVisible=(start:number,end:number)=>{const boundedStart=Math.max(1,start),boundedEnd=Math.min(total,end);if(boundedEnd<boundedStart)return 0;return boundedEnd-boundedStart+1-(hiddenBefore(boundedEnd+1)-hiddenBefore(boundedStart))}
  const rangeSize=(start:number,end:number)=>end<start?0:offsetOf(Math.min(total,end)+1)-offsetOf(Math.max(1,start))
  return{total,hidden,extent,isHidden,sizeOf,offsetOf,indexAtOffset,countVisible,rangeSize,firstVisibleAtOrAfter,lastVisibleAtOrBefore,nextVisible}
}

export function axisViewportPosition(axis:DimensionAxis,index:number,scroll:number,frozen:number){return axis.offsetOf(index)-(index<=frozen?0:scroll)}
export function axisIndexAtViewport(axis:DimensionAxis,position:number,scroll:number,frozen:number){const frozenExtent=axis.offsetOf(Math.min(axis.total,frozen)+1);return axis.indexAtOffset(position<frozenExtent?position:position+scroll)}

/**
 * 행 머리글은 그 자리에 보이는 가장 큰 행 번호를 담을 만큼 넓어야 합니다.
 * 기본 폭은 다섯 자리까지만 들어가므로, 십오만 행짜리 시트를 내려가면
 * 번호의 앞자리가 잘려 나갑니다. 네 자리까지는 기본 폭 그대로 두어 작은
 * 시트에서 자리를 낭비하지 않습니다.
 */
export function rowHeaderWidth(base:number,lastRowInView:number,zoom=1){
  const digits=Math.max(4,String(Math.max(1,Math.trunc(lastRowInView))).length)
  return base+(digits-4)*Math.round(8*zoom)
}

/**
 * 함께 보고 있는 사람의 이름표를 놓을 세로 위치입니다. 셀 위에 붙이되,
 * 맨 윗줄이라 위에 자리가 없으면 아래로 내립니다. 머리글 아래로 밀어
 * 넣으면 그 셀의 내용을 가려, 누가 어디에 있는지는 알려 주면서 거기에
 * 무엇이 있는지는 감추게 됩니다.
 */
export function presenceLabelTop(cellTop:number,cellHeight:number,labelHeight:number,headerHeight:number){
  const above=cellTop-labelHeight-1
  return above>=headerHeight?above:cellTop+cellHeight+1
}
