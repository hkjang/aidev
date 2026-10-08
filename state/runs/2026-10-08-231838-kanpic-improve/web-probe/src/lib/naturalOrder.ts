/**
 * 사람이 읽는 대로 글자를 견준다. 값 안의 숫자를 글자가 아니라 하나의 수로
 * 센다. 그냥 글자로 견주면 `10월` 과 `12월` 이 `1월` 보다 앞에 오고 `항목10`
 * 이 `항목2` 보다 앞에 온다. `0` 이 `월` 보다, `1` 이 `2` 보다 앞이기
 * 때문인데, 그렇게 정렬하고 싶은 사람은 없다.
 *
 * 엑셀과 구글 시트는 이렇게 하지 않는다 — 둘 다 글자 그대로 견주어 위의
 * 순서를 낸다. 일부러 다르게 두었고, 정렬 대화상자에서 끌 수 있다.
 *
 * 서버의 internal/workbook/natural_order.go 와 같은 답을 내야 한다. 정렬은
 * 화면에서 먼저 반영하고 서버가 다시 확정하므로, 둘이 어긋나면 값이 한 번
 * 튄다.
 */
export function compareNatural(left:string,right:string){
  let leftIndex=0,rightIndex=0
  while(leftIndex<left.length&&rightIndex<right.length){
    const leftDigit=isDigit(left[leftIndex]),rightDigit=isDigit(right[rightIndex])
    if(leftDigit&&rightDigit){
      const leftEnd=digitRunEnd(left,leftIndex),rightEnd=digitRunEnd(right,rightIndex)
      const comparison=compareDigitRuns(left.slice(leftIndex,leftEnd),right.slice(rightIndex,rightEnd))
      if(comparison!==0)return comparison
      leftIndex=leftEnd;rightIndex=rightEnd
      continue
    }
    if(leftDigit!==rightDigit)return leftDigit?-1:1
    // 자바스크립트의 문자열 비교는 UTF-16 조각을 견주므로, U+FFFF 를 넘는
    // 글자(이모지)가 ￦ 나 ＀ 같은 글자보다 **앞** 에 온다. 서버는 UTF-8
    // 바이트를 견주어 코드포인트 차례대로 놓는다. 그대로 두면 화면에서 한 번,
    // 서버가 확정하며 또 한 번, 줄이 서로 다른 자리에 선다.
    const leftPoint=left.codePointAt(leftIndex) as number
    const rightPoint=right.codePointAt(rightIndex) as number
    if(leftPoint!==rightPoint)return leftPoint<rightPoint?-1:1
    leftIndex+=leftPoint>0xffff?2:1
    rightIndex+=rightPoint>0xffff?2:1
  }
  if(leftIndex<left.length)return 1
  if(rightIndex<right.length)return -1
  return 0
}

/**
 * 숫자 뭉치를 실수로 바꾸지 않고 견준다. 마흔 자리 계좌번호도 정확히
 * 정렬된다. 앞의 0을 떼면 자릿수가 많은 쪽이 큰 수이고, 자릿수가 같으면
 * 글자 그대로 견주면 된다.
 */
function compareDigitRuns(left:string,right:string){
  const leftDigits=left.replace(/^0+/,''),rightDigits=right.replace(/^0+/,'')
  if(leftDigits.length!==rightDigits.length)return leftDigits.length<rightDigits.length?-1:1
  if(leftDigits!==rightDigits)return leftDigits<rightDigits?-1:1
  // 같은 수라면 자릿수를 맞춰 쓴 쪽(007)을 앞에 두어 순서가 흔들리지 않게 한다.
  if(left.length!==right.length)return left.length>right.length?-1:1
  return 0
}

const isDigit=(character:string)=>character>='0'&&character<='9'

function digitRunEnd(value:string,index:number){
  let end=index
  while(end<value.length&&isDigit(value[end]))end+=1
  return end
}
