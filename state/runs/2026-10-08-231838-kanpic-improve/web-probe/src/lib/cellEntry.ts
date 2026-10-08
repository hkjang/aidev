/**
 * 칸에 **직접 쳐 넣은** 글자가 무엇인지 정한다.
 *
 * 같은 글자가 이 앱에 들어오는 길은 셋이다 — 파일(업로드·IMPORTDATA),
 * 붙여넣기, 그리고 칸에 직접 치는 것. 앞의 둘은 spreadsheetNumber.ts 의
 * decomposeNumberText 한 군데를 지나는데 이 문만 컴포넌트 안에 숨어 있었다.
 * 그래서 이 규칙도 여기 한 파일에 두고 테스트가 붙들게 한다.
 */
import { parsePastedNumber } from './clipboardNumber'
import { spreadsheetNumber } from './spreadsheetNumber'

/**
 * 입력한 글자가 무엇인지 정한다. 스프레드시트는 `1,234` 나 `12%` 를 글자가
 * 아니라 숫자로 받고 보이던 모습은 표시 형식으로 남긴다. 그렇지 않으면
 * 합계에 들어가지 않는다.
 */
export function parseTypedCellValue(raw:string):{value:unknown;numberFormat?:string}{
  if(raw==='')return {value:undefined}
  if(raw.toLowerCase()==='true')return {value:true}
  if(raw.toLowerCase()==='false')return {value:false}
  // 숫자로 담을지는 셈하는 자가 정한다. 예전에는 여기서 Number() 를 바로
  // 불렀는데, 그것은 스프레드시트보다 넓어 `0x1F` 를 31 로, `0b101` 을 5 로
  // 읽었다 — 사람이 친 글자는 사라지고, 같은 글자가 파일이나 붙여넣기로
  // 들어오면 글자로 남으므로 한 표 안에 두 가지가 섞였다. 셈하는 자로 읽으면
  // 상태 줄 합계와 =SUM 이 갈리는 일도 없다.
  const counted=spreadsheetNumber(raw)
  if(counted!==undefined)return {value:counted}
  const number=parsePastedNumber(raw)
  if(number)return {value:number.value,numberFormat:number.numberFormat}
  return {value:raw}
}
