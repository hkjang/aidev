/**
 * 채우기 손잡이가 아는 이름 목록. `1월` 을 끌면 2월, 3월이 나와야 하고
 * 12월 다음은 13월이 아니라 1월이어야 한다. 요일도 같다. 숫자만으로는
 * 이 되돌아옴을 표현할 수 없어 목록으로 둔다.
 */
const LISTS:string[][]=[
  ['월요일','화요일','수요일','목요일','금요일','토요일','일요일'],
  ['월','화','수','목','금','토','일'],
  ['1월','2월','3월','4월','5월','6월','7월','8월','9월','10월','11월','12월'],
  ['1분기','2분기','3분기','4분기'],
  ['Monday','Tuesday','Wednesday','Thursday','Friday','Saturday','Sunday'],
  ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'],
  ['January','February','March','April','May','June','July','August','September','October','November','December'],
  ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'],
]

type Position={list:string[];index:number}

function locate(value:string):Position|undefined{
  const text=value.trim()
  if(text==='')return
  const lower=text.toLowerCase()
  for(const list of LISTS){
    const index=list.findIndex(item=>item.toLowerCase()===lower)
    if(index>=0)return {list,index}
  }
  return
}

/** `MON` 이면 `TUE`, `mon` 이면 `tue` 로 돌려준다. */
function matchCase(source:string,value:string){
  if(source===source.toUpperCase()&&source!==source.toLowerCase())return value.toUpperCase()
  if(source===source.toLowerCase())return value.toLowerCase()
  return value
}

/**
 * 이름 목록을 따라 이어지는 값. 씨앗이 여럿이면 그 간격을 지킨다
 * (월, 수 → 금, 일). 목록에 없는 값이 섞여 있으면 아무것도 돌려주지 않는다.
 */
export function listSeriesValue(values:unknown[],position:number):string|undefined{
  if(values.length===0)return
  const texts:string[]=[]
  for(const value of values){
    if(typeof value!=='string')return
    texts.push(value)
  }
  const first=locate(texts[0])
  if(!first)return
  const indexes:number[]=[first.index]
  for(const text of texts.slice(1)){
    const found=locate(text)
    // 같은 목록 안에서만 이어진다. 월요일과 Jan 을 한 줄로 보지 않는다.
    if(!found||found.list!==first.list)return
    indexes.push(found.index)
  }
  let step=1
  if(indexes.length>1){
    const size=first.list.length
    step=(indexes[1]-indexes[0]+size)%size
    for(let at=1;at<indexes.length;at+=1){
      if((indexes[at]-indexes[at-1]+size)%size!==step)return
    }
    if(step===0)return
  }
  const size=first.list.length
  const next=(((first.index+step*position)%size)+size)%size
  return matchCase(texts[0].trim(),first.list[next])
}

/**
 * `1분기` 처럼 숫자가 앞에 오는 이름. 뒤에 오는 경우(`항목 1`)는 이미
 * 이어졌지만 앞에 오면 그대로 복사되었다. 되돌아올 자리가 없으므로 그냥
 * 세어 나간다.
 */
export function leadingNumberSeriesValue(values:unknown[],position:number):string|undefined{
  if(values.length===0)return
  const parsed:Array<{number:number;suffix:string;width:number}>=[]
  for(const value of values){
    if(typeof value!=='string')return
    const match=value.match(/^(\d+)(\D.*)$/)
    if(!match)return
    parsed.push({number:Number(match[1]),suffix:match[2],width:match[1].length})
  }
  if(parsed.some(item=>item.suffix!==parsed[0].suffix))return
  let step=1
  if(parsed.length>1){
    step=parsed[1].number-parsed[0].number
    for(let at=1;at<parsed.length;at+=1){
      if(parsed[at].number-parsed[at-1].number!==step)return
    }
  }
  const next=parsed[0].number+step*position
  if(next<0)return
  return String(next).padStart(parsed[0].width,'0')+parsed[0].suffix
}
