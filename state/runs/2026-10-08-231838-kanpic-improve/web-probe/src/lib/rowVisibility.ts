export type RowVisibility = ReturnType<typeof createRowVisibility>

export function createRowVisibility(input:number[],totalRows:number){
  const hidden=Array.from(new Set(input.filter(row=>Number.isInteger(row)&&row>=1&&row<=totalRows))).sort((a,b)=>a-b)
  const hiddenSet=new Set(hidden)
  const lowerBound=(target:number)=>{let low=0,high=hidden.length;while(low<high){const middle=(low+high)>>1;if(hidden[middle]<target)low=middle+1;else high=middle}return low}
  const upperBound=(target:number)=>{let low=0,high=hidden.length;while(low<high){const middle=(low+high)>>1;if(hidden[middle]<=target)low=middle+1;else high=middle}return low}
  const visibleCount=Math.max(1,totalRows-hidden.length)
  const actualToDisplay=(actual:number)=>Math.max(1,Math.min(visibleCount,actual-upperBound(actual)))
  const displayToActual=(display:number)=>{
    const target=Math.max(1,Math.min(visibleCount,display));let low=1,high=totalRows
    while(low<high){const middle=Math.floor((low+high)/2),visibleThrough=middle-upperBound(middle);if(visibleThrough<target)low=middle+1;else high=middle}
    return low
  }
  const isHidden=(row:number)=>hiddenSet.has(row)
  const firstVisibleAtOrAfter=(row:number)=>{let next=Math.max(1,Math.min(totalRows,row));while(next<=totalRows&&hiddenSet.has(next))next+=1;return next<=totalRows?next:displayToActual(visibleCount)}
  const lastVisibleAtOrBefore=(row:number)=>{let next=Math.max(1,Math.min(totalRows,row));while(next>=1&&hiddenSet.has(next))next-=1;return next>=1?next:displayToActual(1)}
  const nextVisible=(row:number,direction:1|-1)=>direction===1?firstVisibleAtOrAfter(Math.min(totalRows,row+1)):lastVisibleAtOrBefore(Math.max(1,row-1))
  const countVisible=(start:number,end:number)=>{if(end<start)return 0;const boundedStart=Math.max(1,start),boundedEnd=Math.min(totalRows,end);if(boundedEnd<boundedStart)return 0;return boundedEnd-boundedStart+1-(upperBound(boundedEnd)-lowerBound(boundedStart))}
  return{hidden,hiddenSet,visibleCount,actualToDisplay,displayToActual,isHidden,firstVisibleAtOrAfter,lastVisibleAtOrBefore,nextVisible,countVisible}
}
