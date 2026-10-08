/**
 * 수식을 쓰다 F4 를 누르면 참조가 상대 → 절대 → 행 고정 → 열 고정 순으로
 * 돌아간다. 조회표를 붙들어 두려면 `$` 를 매번 손으로 치는 수밖에 없었다.
 *
 * `A1` → `$A$1` → `A$1` → `$A1` → `A1`
 */
const REFERENCE=/(\$?)([A-Za-z]{1,3})(\$?)([1-9]\d*)/g

export type CycleResult={text:string;start:number;end:number}

type Found={start:number;end:number;column:string;row:string}

function references(text:string):Found[]{
  const found:Found[]=[]
  REFERENCE.lastIndex=0
  for(let match=REFERENCE.exec(text);match!==null;match=REFERENCE.exec(text)){
    const start=match.index,end=start+match[0].length
    // 낱말의 일부는 참조가 아니다: A1B 의 A1.
    const before=start>0?text[start-1]:''
    const after=end<text.length?text[end]:''
    if(/[A-Za-z0-9_.$]/.test(before)||/[A-Za-z0-9_.]/.test(after))continue
    // 여는 괄호가 뒤따르면 함수 이름이다. LOG10 은 LOG 열의 10행이 아니라
    // 함수이고, 열 이름과 겹치는 이름은 그 밖에도 있다.
    if(after==='(')continue
    found.push({start,end,column:match[2],row:match[4]})
  }
  return found
}

/** 상대 → 절대 → 행 고정 → 열 고정 → 상대. */
function nextForm(columnAbsolute:boolean,rowAbsolute:boolean){
  if(!columnAbsolute&&!rowAbsolute)return {column:true,row:true}
  if(columnAbsolute&&rowAbsolute)return {column:false,row:true}
  if(!columnAbsolute&&rowAbsolute)return {column:true,row:false}
  return {column:false,row:false}
}

function written(text:string,found:Found){
  const columnAbsolute=text[found.start]==='$'
  const rowAbsolute=text.slice(found.start,found.end).includes('$',columnAbsolute?1:0)
  return {columnAbsolute,rowAbsolute}
}

/**
 * 캐럿이 놓인(또는 선택한) 참조를 다음 형태로 바꾼다. 참조가 여러 개 선택되어
 * 있으면 모두 같은 단계로 옮긴다. 바꿀 참조가 없으면 undefined.
 */
export function cycleReference(text:string,selectionStart:number,selectionEnd:number):CycleResult|undefined{
  const all=references(text)
  if(all.length===0)return
  const start=Math.min(selectionStart,selectionEnd),end=Math.max(selectionStart,selectionEnd)
  // 범위를 골랐으면 그 안에 걸친 참조 전부, 아니면 캐럿이 닿은 하나.
  let targets=start===end
    ?all.filter(item=>start>=item.start&&start<=item.end)
    :all.filter(item=>item.start<end&&item.end>start)
  if(targets.length===0){
    // 캐럿이 참조에 닿지 않았으면 바로 앞의 참조를 집는다. `=SUM(A1` 처럼
    // 막 쓴 참조를 그대로 고정할 수 있어야 한다.
    const previous=all.filter(item=>item.end<=start).pop()
    if(!previous)return
    targets=[previous]
  }
  // 첫 참조가 어떤 형태인지에 따라 다음 단계를 정하고, 나머지도 같이 옮긴다.
  const form=nextForm(written(text,targets[0]).columnAbsolute,written(text,targets[0]).rowAbsolute)
  let result=text
  let shift=0
  let first=-1
  let last=-1
  for(const target of targets){
    const replacement=`${form.column?'$':''}${target.column}${form.row?'$':''}${target.row}`
    const from=target.start+shift,to=target.end+shift
    result=result.slice(0,from)+replacement+result.slice(to)
    if(first<0)first=from
    last=from+replacement.length
    shift+=replacement.length-(target.end-target.start)
  }
  return {text:result,start:first,end:last}
}
