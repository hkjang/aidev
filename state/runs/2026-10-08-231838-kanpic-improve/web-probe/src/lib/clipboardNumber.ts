/**
 * 스프레드시트 사이를 오가는 값은 계산된 숫자가 아니라 **보이던 글자**로
 * 건너온다. 엑셀에서 통화 서식이 붙은 열을 복사하면 클립보드의 평문에는
 * `₩1,234.50` 이 들어 있고, 이것을 글자로 저장하면 합계가 0이 된다.
 *
 * 글자를 뜯는 일은 spreadsheetNumber.ts 가 한 군데서 한다. 예전에는 여기서
 * 따로 뜯었고, 그래서 `5000 USD` 가 붙여넣기로는 글자로 남고 데이터 정리로는
 * 숫자가 되는 식으로 여덟 가지가 어긋나 있었다.
 */
import { decomposeNumberText } from './spreadsheetNumber'

export type ParsedNumber={value:number;numberFormat?:string}

/**
 * 붙여넣은 글자가 스프레드시트가 숫자로 다루었을 값이면 그 숫자와, 보이던
 * 모습을 되살리는 표시 형식을 함께 돌려준다. 숫자가 아니면 undefined.
 */
export function parsePastedNumber(raw:string):ParsedNumber|undefined{
  const parsed=decomposeNumberText(raw)
  return parsed===undefined?undefined:{value:parsed.value,numberFormat:parsed.format}
}
