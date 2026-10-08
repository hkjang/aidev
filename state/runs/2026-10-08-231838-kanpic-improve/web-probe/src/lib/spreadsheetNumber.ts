/**
 * 값이 숫자로 셈에 들어가는지, 들어간다면 얼마인지 정한다.
 *
 * 이 규칙은 서버의 수식 엔진이 정한다. 화면이 다르게 세면 =SUM(A1:A100) 이
 * 내는 값과 상태 줄에 보이는 합계가 달라지고, 사람은 어느 쪽을 믿어야 할지
 * 알 수 없다. 실제로 그런 일이 있었다 — 선택 요약과 열 통계와 엔진이 셋 다
 * 달랐다.
 *
 *   "2000"    셈한다        엔진도 그렇다
 *   " 300 "   셈한다        앞뒤 빈칸은 버린다
 *   TRUE      1 로 센다     엔진도 그렇다
 *   "1,234"   세지 않는다   쉼표가 붙으면 엔진은 글자로 본다
 *   "50%"     세지 않는다
 *   "1234원"  세지 않는다
 *
 * 쉼표가 붙은 것을 세지 않는 것이 야박해 보이지만, 세는 쪽이 더 나쁘다.
 * 화면에는 더해지는데 수식에서는 빠지므로 두 숫자가 어긋난 채로 남는다.
 * 그런 칸은 세는 대신 찾아서 알려 주고 숫자로 바꿔 주는 편이 낫다.
 */
/** 스프레드시트가 숫자라고 부르는 글자. 서버의 internal/formula/evaluator.go
 *  decimalText 와 같은 자다. testdata/numeric-text.json 이 둘을 붙들어 둔다. */
const DECIMAL_TEXT=/^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$/

export function spreadsheetNumber(value:unknown):number|undefined{
  if(typeof value==='number')return Number.isFinite(value)?value:undefined
  if(typeof value==='boolean')return value?1:0
  if(typeof value==='string'){
    const text=value.trim()
    // Number() 는 스프레드시트보다 넓다. `0x1F` 를 31 로, `0b101` 을 5 로 읽는다.
    // 엔진은 그것을 세지 않으므로, 그대로 두면 상태 줄 합계에만 들어가고
    // =SUM 에는 빠진 채 두 숫자가 어긋난다. 실제로 그랬다.
    if(!DECIMAL_TEXT.test(text))return undefined
    const parsed=Number(text)
    return Number.isFinite(parsed)?parsed:undefined
  }
  return undefined
}

/**
 * 숫자처럼 보이는데 글자로 담긴 칸인지 본다.
 *
 * 다른 곳에서 붙여 넣으면 "1,234" 나 "1,234원" 처럼 들어온다. 사람 눈에는
 * 숫자인데 =SUM 은 조용히 빼고 셈한다. 합계가 작게 나오는데 무엇이 빠졌는지는
 * 아무 데도 적히지 않는다.
 */
export function looksLikeNumberStoredAsText(value:unknown):boolean{
  if(typeof value!=='string')return false
  const text=value.trim()
  if(text==='')return false
  // 이미 셈에 들어가는 것은 고칠 것이 없다.
  if(spreadsheetNumber(text)!==undefined)return false
  return textToNumber(text)!==undefined
}

/**
 * 글자로 담긴 숫자를 숫자로 바꾼다. 바꿀 수 없으면 undefined 다.
 *
 * 쉼표, 원·달러 기호, 뒤에 붙은 단위, 괄호 음수, 백분율을 받는다. 괄호는
 * 회계에서 음수를 적는 꼴이고, 백분율은 100 으로 나눈다 — 화면에 보이던 것과
 * 셈하는 값이 같아야 한다.
 */
/**
 * 글자 하나를 숫자와 표시 형식으로 뜯는다.
 *
 * 같은 글자가 이 앱에 들어오는 길은 셋이다 — 붙여넣기, 데이터 정리, 그리고
 * 셀에 직접 치는 것. 셋이 따로 뜯으면 같은 "₩5,000" 이 어떻게 들어왔는지에
 * 따라 다른 값이나 다른 서식으로 저장된다. 사람에게는 같은 자료인데 파일에서는
 * 다르다. 그래서 뜯는 일은 여기 한 군데서만 한다.
 *
 * 뜯은 것은 반드시 원래 글자로 다시 그려져야 한다. 기호와 단위에 붙어 있던
 * 빈칸까지 서식 안에 담는 것은 그 때문이다 — `5000 USD` 를 `5000USD` 로
 * 되돌리면 사람이 옮긴 표와 다른 것을 보게 된다.
 */
export type NumberText={value:number;format?:string}

const CURRENCY_SIGN=/^([₩$€¥£¢])(\s*)/
const UNIT_SUFFIX=/(\s*)(원|달러|USD|KRW|won)$/i
const GROUPED_INTEGER=/^\d{1,3}(,\d{3})+$/
const PLAIN_INTEGER=/^\d+$/
/**
 * 앞의 0 이 자리가 아니라 번호를 뜻하는 평문 정수. `0`·`0.5`·`-0`·`007.5` 의
 * 0 은 자리이고, `00123`·`007` 처럼 0 뒤에 숫자가 더 오면서 소수점이 없을
 * 때만 번호다. 자릿점이나 통화 기호가 붙은 것은 번호가 아니라 금액이므로
 * 여기서 걸리지 않는다. 서버의 delimited.HasSignificantLeadingZero 와 같은
 * 자다.
 */
const LEADING_ZERO_NUMBER=/^[+-]?0\d+$/

export function decomposeNumberText(value:string):NumberText|undefined{
  let text=value.trim()
  // 마흔 자를 넘는 것은 금액이 아니라 문장이다.
  if(text===''||text.length>40)return undefined
  // 우편번호·사번의 앞자리 0 은 숫자로 바꾸는 순간 사라지고 되돌릴 길이 없다.
  // 열여섯 자리 넘는 번호를 아래에서 놓아주는 것과 같은 까닭이다. 파일로
  // 들어오는 문(internal/delimited 의 Number — 업로드 가져오기와 IMPORTDATA)이
  // 이미 그렇게 하므로, 같은 표를 CSV 로 올리느냐 복사해 붙이느냐에 따라
  // 다른 값이 남지 않으려면 여기서도 글자로 두어야 한다.
  // testdata/incoming-number.json 이 두 문을 붙들어 둔다.
  //
  // 칸에 직접 쳐 넣는 것은 다른 문이다 — cellEntry 의 parseTypedCellValue 는
  // 여기에 닿기 전에 위의 spreadsheetNumber 로 읽으므로 사람이 친 `00123` 은
  // 예전대로 123 이다. 파일은 되돌릴 길이 없지만 친 사람은 다시 칠 수 있다.
  if(LEADING_ZERO_NUMBER.test(text))return undefined
  let negative=false
  const parenthesized=/^\(.*\)$/.test(text)
  if(parenthesized){negative=true;text=text.slice(1,-1).trim()}
  const leading=(prefix:string)=>{
    if(!text.startsWith(prefix))return false
    text=text.slice(1).trim();return true
  }
  if(leading('-'))negative=!negative
  else leading('+')
  let currency='',currencyGap=''
  const sign=text.match(CURRENCY_SIGN)
  if(sign){currency=sign[1];currencyGap=sign[2];text=text.slice(sign[0].length)}
  // `$-5` 처럼 기호 뒤에 부호가 오는 표기도 있다.
  if(currency!==''){if(leading('-'))negative=!negative;else leading('+')}
  let percent=false
  if(text.endsWith('%')){percent=true;text=text.slice(0,-1).trim()}
  let unit='',unitGap=''
  const suffix=text.match(UNIT_SUFFIX)
  if(suffix){unitGap=suffix[1];unit=suffix[2];text=text.slice(0,text.length-suffix[0].length)}
  // 백분율에 단위를 붙인 것은 뜻이 없다. `5%원` 을 억지로 읽지 않는다.
  if(percent&&unit!=='')return undefined
  const [integer,fraction,...rest]=text.trim().split('.')
  if(rest.length>0)return undefined
  if(fraction!==undefined&&!PLAIN_INTEGER.test(fraction))return undefined
  // 쉼표는 세 자리마다 하나여야 한다. 아무 데나 찍힌 `1,2` 는 소수점으로
  // 쉼표를 쓰는 표기이지 자릿수 구분이 아니다.
  const grouped=GROUPED_INTEGER.test(integer)
  if(!grouped&&!PLAIN_INTEGER.test(integer))return undefined
  const digits=integer.replace(/,/g,'')+(fraction===undefined?'':'.'+fraction)
  // 배정밀도가 정확히 담는 것은 열다섯 자리까지다. 그보다 긴 것은 계좌번호나
  // 전표번호이지 금액이 아니다. 숫자로 바꾸면 뒤가 뭉개져, 적은 사람이 보던
  // 것과 다른 값이 칸에 들어간다. 글자로 두면 적어도 그대로 남는다.
  if(significantDigits(digits)>15)return undefined
  const parsed=Number(digits)
  if(!Number.isFinite(parsed))return undefined
  const value_=(negative?-parsed:parsed)/(percent?100:1)
  // 서식이 필요 없는 평범한 숫자에는 서식을 붙이지 않는다. 붙이면 칸마다
  // 쓸모없는 스타일이 하나씩 생긴다.
  if(!grouped&&currency===''&&unit===''&&!percent&&!parenthesized)return {value:value_}
  return {value:value_,format:numberTextFormat({grouped,currency,currencyGap,unit,unitGap,percent,parenthesized,decimals:fraction?.length??0})}
}

function numberTextFormat({grouped,currency,currencyGap,unit,unitGap,percent,parenthesized,decimals}:{
  grouped:boolean;currency:string;currencyGap:string;unit:string;unitGap:string;percent:boolean;parenthesized:boolean;decimals:number}){
  // 기호는 따옴표로 묶는다. 엑셀 서식 말에서 따옴표 안은 언제나 그대로 적는
  // 글자이므로, `$` 처럼 다른 뜻으로 읽힐 수 있는 기호도 안전하다. 빈칸도
  // 따옴표 안에 넣어야 그려질 때 살아남는다.
  const body=(currency!==''?`"${currency}${currencyGap}"`:'')
    +(grouped?'#,##0':'0')
    +(decimals>0?'.'+'0'.repeat(decimals):'')
    +(percent?'%':'')
    +(unit!==''?`"${unitGap}${unit}"`:'')
  // 회계 표기의 괄호는 음수를 적는 꼴이다. 떼어 버리면 (500) 이 -500 으로
  // 보이고, 사람은 자기가 옮긴 표와 다른 것을 보게 된다.
  return parenthesized?`${body};(${body})`:body
}

export function textToNumber(value:string):number|undefined{
  return decomposeNumberText(value)?.value
}

/**
 * 글자로 담겼던 숫자를 숫자로 바꾼 뒤에도 보이던 대로 보이게 할 표시 형식을
 * 고른다. 고를 것이 없으면 undefined 다.
 *
 * 이것이 없으면 바꾸는 순간 화면이 달라진다. "50%" 가 "0.5" 가 되고
 * "₩5,000" 이 "5000" 이 된다. 값은 맞지만 사람은 자기 자료가 망가졌다고
 * 읽는다 — 특히 백분율은 수가 100분의 1로 줄어든 것처럼 보인다.
 *
 * 쓸 수 있는 것은 formatCellValue 가 그릴 줄 아는 것뿐이다. "1,234원" 의
 * "원" 같은 뒤에 붙는 단위는 서식 말에 없으므로 자릿점만 남기고 놓아준다.
 * 그리지 못할 서식을 적어 두면 칸이 빈 것처럼 보인다.
 */
export function numberFormatForText(value:string):string|undefined{
  return decomposeNumberText(value)?.format
}

/** 앞의 0과 소수점을 뺀 자릿수. 배정밀도가 정확히 담는 한도를 재는 데 쓴다. */
export function significantDigits(text:string){
  return text.replace('.','').replace(/^0+/,'').length
}
