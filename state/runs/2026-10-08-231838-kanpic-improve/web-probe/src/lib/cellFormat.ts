export type BorderStyle='thin'|'medium'|'thick'|'dashed'|'dotted'|'double'
export type BorderSide={style:BorderStyle;color:string}
export type CellBorders=Partial<Record<'top'|'right'|'bottom'|'left',BorderSide>>

export function formatCellValue(value:unknown,style?:Record<string,unknown>,locale='ko-KR'){
  if(value==null)return''
  // A chart value belongs on the canvas, never in a text cell or an input.
  if(typeof value==='object')return''
  const format=typeof style?.number_format==='string'?style.number_format.trim():''
  if(!format||format.toLowerCase()==='general')return String(value)
  if(format==='@')return String(value)
  if(isDateFormat(format)){
    const date=spreadsheetDate(value)
    if(date)return formatDate(date,format,locale)
  }
  // 서식은 값의 부호마다 다른 구역을 담는다. 음수 구역이 따로 있으면 그
  // 구역이 부호를 스스로 적는다. (#,##0) 은 괄호로 적겠다는 뜻이므로 빼기
  // 부호를 덧붙이면 -(5) 가 된다.
  const sections=formatSections(format)
  if(typeof value!=='number'||!Number.isFinite(value)){
    // 넷째 구역은 글자 값의 서식이다. 구역이 넷이 안 되면 엑셀은 글자를
    // 손대지 않는다.
    const text=textSection(sections)
    if(text!==undefined&&typeof value!=='number')return renderTextSection(text,String(value))
    return String(value)
  }
  const negativeSection=value<0&&sections.length>1
  const zeroSection=value===0&&sections.length>2
  const section=(negativeSection?sections[1]:zeroSection?sections[2]:sections[0])??format
  // 비워 둔 구역은 "그 값은 적지 말라" 는 뜻이다. "0;;" 은 음수와 0 을 감춘다.
  if(section==='')return ''
  // @ 는 값을 있는 그대로 적으라는 자리다. 수에도 쓴다 — 5 에 @"원" 은
  // "5원" 이다. 예전에는 자리 기호가 아니라고 보아 "@원" 이라고 그렸다.
  if(hasTextPlaceholder(section))return renderTextSection(section,String(value))
  const scientific=section.match(/^0(?:\.(0+))?[eE]([+-])(0+)$/)
  if(scientific){
    // 지수 자리는 서식에 적은 0 개수만큼 채운다. toExponential 은 자리를
    // 채우지 않아 "5.00E-1" 이 되었다. 엑셀·시트는 "5.00E-01" 이다.
    const mantissaDecimals=scientific[1]?.length??0,exponentDigits=scientific[3].length
    // toExponential 은 이진 실수를 반올림하므로 1.005가 1.00E+00이 된다.
    // 소수점만 옮겨 십진수 자리에서 반올림하면 1.01E+00이 되고, 서버가
    // 내는 답과 같아진다.
    let exponent=value===0?0:Math.floor(Math.log10(Math.abs(value)))
    const unit=10**mantissaDecimals
    let scaled=value===0?0:Math.round(Math.abs(shiftDecimalPoint(value,mantissaDecimals-exponent)))
    // log10 은 10의 거듭제곱에서 한 자리 어긋날 수 있다. 자릿수를 보고 맞춘다.
    if(scaled>=unit*10){scaled=Math.round(scaled/10);exponent+=1}
    else if(scaled>0&&scaled<unit){scaled=Math.round(scaled*10);exponent-=1}
    const digits=String(scaled).padStart(mantissaDecimals+1,'0')
    const point=digits.length-mantissaDecimals
    const mantissa=(value<0?'-':'')+(mantissaDecimals?digits.slice(0,point)+'.'+digits.slice(point):digits)
    const marker=exponent<0?'-':scientific[2]==='-'?'':'+'
    return `${mantissa}E${marker}${String(Math.abs(exponent)).padStart(exponentDigits,'0')}`
  }
  // # ?/? 처럼 빗금 양쪽에 자리 기호가 붙은 서식은 분수로 적는다.
  const fraction=parseFractionFormat(section)
  if(fraction&&Math.abs(value)<MAX_FRACTION_VALUE)return renderFraction(value,fraction,locale)
  // 백분율은 100을 곱한 뒤 반올림한다. 이진 실수로 곱하면 1.005가
  // 100.49999…가 되어 100%로 내려앉는다. 사람이 적은 십진수 자리에서
  // 소수점만 옮기면 100.5가 되어 101%가 된다. 서버가 내는 답과 같다.
  const parsed=parseFormatSection(section)
  // 자리 기호가 하나도 없는 구역은 수를 적지 않고 적어 둔 글자만 적는다.
  // 회계 서식의 0 구역이 그 꼴이다 — #,##0;(#,##0);"-" 의 0 은 "-" 이다.
  if(!/[0#?]/.test(parsed.core)&&!isDateFormat(section))return parsed.prefix+parsed.suffix
  // 날짜 서식에 날짜가 될 수 없는 수가 들어오면 여기까지 흘러오는데, 그때 y 나
  // m 을 글자로 적으면 "-yyyy-mm-dd1" 이 된다. 서버도 "-1" 을 적는다.
  const bare=!/[0#]/.test(parsed.core)
  const {core}=parsed,prefix=bare?'':parsed.prefix,suffix=bare?'':parsed.suffix
  const percent=core.includes('%')
  const base=negativeSection?-value:value
  const {scale,useGrouping}=thousandsScale(core)
  const percented=percent?shiftDecimalPoint(base,2):base
  const numeric=scale?shiftDecimalPoint(percented,-3*scale):percented
  const decimals=core.match(/\.(0+)/)?.[1].length??0
  const minimumIntegerDigits=Math.min(21,Math.max(1,(core.split('.')[0].match(/0/g)??[]).length))
  let rendered=new Intl.NumberFormat(locale,{useGrouping,minimumIntegerDigits,minimumFractionDigits:decimals,maximumFractionDigits:decimals}).format(Math.abs(numeric))
  // -0.2를 정수 자리로 반올림하면 0이 된다. 칸에 "-0"이 보이면 잘못 그린
  // 것처럼 읽히므로 부호를 붙이지 않는다. 서버도 그렇게 적는다.
  const roundedToZero=/^[0.,]*$/.test(rendered)
  if(percent)rendered+='%'
  rendered=prefix+rendered+suffix
  // 빼기 부호는 기호 앞에 온다. 엑셀은 -₩5 로 적지 ₩-5 로 적지 않는다.
  if(numeric<0&&!roundedToZero&&!/^-/.test(rendered))rendered='-'+rendered
  return rendered
}

/**
 * 서식을 구역으로 나눈다. 값의 부호마다 구역이 다르다.
 *
 *   한 구역    모든 값
 *   두 구역    양수와 0 ; 음수
 *   세 구역    양수 ; 음수 ; 0
 *
 * 따옴표 안과 대괄호 안의 ";" 는 구역을 가르지 않는다 — #,##0;"내려감;"
 * 처럼 글자로 적은 쌍반점이 있다. 예전에는 그냥 잘라서 그 글자가 구역을
 * 하나 더 만들었다. 서버의 internal/formula 의 formatSections 와 같은 규칙이다.
 */
export function formatSections(format:string){
  const sections:string[]=[]
  let current=''
  for(let index=0;index<format.length;index+=1){
    const character=format[index]
    if(character==='\\'){current+=character+(format[index+1]??'');index+=1;continue}
    if(character==='"'||character==='['){
      const closer=character==='['?']':'"'
      const end=format.indexOf(closer,index+1)
      current+=format.slice(index,end<0?format.length:end+1)
      index=end<0?format.length:end
      continue
    }
    if(character===';'){sections.push(current);current='';continue}
    current+=character
  }
  sections.push(current)
  return sections
}

/**
 * 글자 값을 그릴 구역을 고른다.
 *
 * 엑셀 서식의 넷째 구역은 글자 값의 서식이다 — 회계 서식의 마지막 토막
 * `_-@_-` 와 `#,##0;(#,##0);"-";"["@"]"` 의 `"["@"]"` 가 그것이다. 구역이
 * 넷이 안 되면 엑셀은 글자를 손대지 않는다. 다만 구역이 하나뿐인데 그 안에
 * @ 가 있으면 그 구역이 곧 글자 구역이다 — `@" 님"` 처럼 이름 뒤에 말을
 * 붙이는 데 흔히 쓴다.
 *
 * 예전에는 넷째 구역을 아예 읽지 않아 글자 칸이 서식 없이 그대로 나왔다.
 * 서버의 internal/formula 의 textSection 과 같은 규칙이다.
 */
function textSection(sections:string[]){
  if(sections.length>3)return sections[3]
  if(sections.length===1&&hasTextPlaceholder(sections[0]))return sections[0]
}

// 따옴표 안과 대괄호 안의 @ 는 그냥 글자다 — [$@-409] 같은 나라 코드가 있다.
function hasTextPlaceholder(section:string){
  let found=false
  scanFormatSection(section,character=>{if(character==='@')found=true})
  return found
}

// 글자 구역을 그린다. @ 자리에 값이 들어가고, 나머지는 parseFormatSection 과
// 같은 규칙으로 읽는다 — 따옴표 안은 그대로 적고, 대괄호 안의 색·나라 코드는
// 그리지 않으며(통화 기호만 꺼낸다), _ 와 * 는 자리 맞추기라 그리지 않는다.
function renderTextSection(section:string,text:string){
  let out=''
  scanFormatSection(section,(character,literal)=>{out+=character===undefined?literal:character==='@'?text:character})
  return out
}

// 구역을 한 글자씩 읽어 넘긴다. 따옴표·대괄호·역빗금으로 묶인 것은
// 글자(literal)로, 나머지는 글자 하나(character)로 넘어온다.
function scanFormatSection(section:string,emit:(character:string|undefined,literal:string)=>void){
  for(let index=0;index<section.length;index+=1){
    const character=section[index]
    if(character==='"'){
      const end=section.indexOf('"',index+1)
      emit(undefined,section.slice(index+1,end<0?section.length:end))
      index=end<0?section.length:end
      continue
    }
    if(character==='['){
      const end=section.indexOf(']',index+1)
      const inside=section.slice(index+1,end<0?section.length:end)
      if(inside.startsWith('$')){
        const symbol=inside.slice(1).split('-')[0]
        if(symbol!=='')emit(undefined,symbol)
      }
      index=end<0?section.length:end
      continue
    }
    // _x 는 x 만큼 자리를 비우고, *x 는 x 로 채운다. 둘 다 값이 아니다.
    if(character==='_'||character==='*'){index+=1;continue}
    if(character==='\\'){if(index+1<section.length)emit(undefined,section[index+1]??'');index+=1;continue}
    emit(character,'')
  }
}

/**
 * 자리 기호 뒤에 붙은 쉼표는 자릿점이 아니라 "천 단위로 줄여 적으라" 는
 * 뜻이다 — #,##0, 은 천 단위, #,##0,, 은 백만 단위다. 자리 기호 사이의
 * 쉼표만 자릿점이므로 함께 가려낸다. 예전에는 쉼표가 있기만 하면 자릿점으로
 * 보아 백만 원 단위 표가 통째로 백만 배로 보였다.
 *
 * 서버의 internal/formula 의 thousandsScale 이 같은 셈을 한다.
 */
function thousandsScale(core:string){
  const last=Math.max(core.lastIndexOf('0'),core.lastIndexOf('#'),core.lastIndexOf('?'))
  if(last<0)return{scale:0,useGrouping:core.includes(',')}
  return{scale:(core.slice(last+1).match(/,/g)??[]).length,useGrouping:core.slice(0,last).includes(',')}
}

/**
 * 서식 한 구역을 앞 글자·숫자 자리·뒤 글자로 나눈다.
 *
 * 엑셀에서 온 파일은 자리 기호만 적혀 있지 않다.
 *
 *   [$-409]#,##0     나라 코드다. 409 의 0 을 자릿수로 세면 5 가 05 가 된다.
 *   [$₩-412]#,##0    통화 기호는 대괄호 안의 $ 와 - 사이에 있다. 문자열에서
 *                    처음 만난 기호를 집으면 원화 파일이 달러로 보인다.
 *   #,##0"원"        따옴표 안은 그대로 적는 글자다.
 *   _-* #,##0_-      _ 는 자리 비우기, * 는 채우기다. 그리지 않는다.
 *
 * 자리 기호가 아닌 것을 세면 숫자가 달라 보이고, 사람은 그것을 값으로 읽는다.
 */
export function parseFormatSection(section:string){
  let prefix='',core='',suffix=''
  const put=(text:string)=>{if(core==='')prefix+=text;else suffix+=text}
  for(let index=0;index<section.length;index+=1){
    const character=section[index]
    if(character==='"'){
      const end=section.indexOf('"',index+1)
      put(section.slice(index+1,end<0?section.length:end))
      index=end<0?section.length:end
      continue
    }
    if(character==='['){
      const end=section.indexOf(']',index+1)
      const inside=section.slice(index+1,end<0?section.length:end)
      // [$기호-나라코드] 에서 기호만 꺼낸다. [Red] 나 [$-409] 는 그릴 것이 없다.
      if(inside.startsWith('$')){
        const symbol=inside.slice(1).split('-')[0]
        if(symbol!=='')put(symbol)
      }
      index=end<0?section.length:end
      continue
    }
    // _x 는 x 만큼 자리를 비우고, *x 는 x 로 채운다. 둘 다 값이 아니다.
    if(character==='_'||character==='*'){index+=1;continue}
    if(character==='\\'){put(section[index+1]??'');index+=1;continue}
    if('0#?,.%'.includes(character)){core+=character;continue}
    // 빈칸은 음수 구역의 괄호와 자리를 맞추려고 넣는 것이다. 그리면 "1,234 "
    // 처럼 꼬리가 붙는다. 서버도 그리지 않는다.
    if(character===' ')continue
    put(character)
  }
  return {prefix,core,suffix}
}

/**
 * 분수 서식은 값을 소수가 아니라 분수로 적는다 — 2.75 는 "2 3/4" 다.
 * 엑셀 기본 서식 12·13 번(`# ?/?`, `# ??/??`)이라 XLSX 로 그냥 들어온다.
 *
 * 예전에는 자리 기호만 세고 "/" 를 그냥 글자로 흘려 보내, `# ?/?` 가 0.5 를
 * "1/2" 가 아니라 "1/" 로 그렸다 — 반올림한 값 뒤에 빗금만 남은 꼴이다.
 *
 * 서버의 internal/formula 의 parseFractionFormat·renderFraction 이 같은
 * 규칙을 쓴다. testdata/cell-formats.json 이 둘을 함께 붙잡는다.
 */
type FractionFormat={prefix:string;suffix:string;integer:string;denominator:number;maxDenominator:number}

// 이보다 큰 값은 분수로 적지 않는다. 그만큼 큰 수에서는 소수 부분이 이미
// 이진 실수의 어긋남뿐이다.
const MAX_FRACTION_VALUE=1e15

// 분모 자리 기호를 세는 상한. internal/formula/format_fraction.go 의
// maxFractionPlaces 와 같은 수여야 한다 — 다르면 격자와 서버가 같은 칸을
// 다르게 그린다. 왜 여섯인지는 parseFractionFormat 안에 적었다.
const MAX_FRACTION_PLACES=6

function parseFractionFormat(section:string):FractionFormat|undefined{
  let slash=-1
  for(let index=0;index<section.length&&slash<0;index+=1){
    const character=section[index]
    if(character==='\\'){index+=1;continue}
    if(character==='"'||character==='['){
      const closer=character==='['?']':'"'
      const end=section.indexOf(closer,index+1)
      index=end<0?section.length:end
      continue
    }
    if(character==='/')slash=index
  }
  if(slash<0)return
  // 분자는 빗금 바로 앞에 이어진 자리 기호다.
  let start=slash
  while(start>0&&'0#?'.includes(section[start-1]))start-=1
  if(start===slash)return
  // 분모는 빗금 바로 뒤에 이어진 자리 기호이거나, 못 박은 숫자다.
  let end=slash+1
  while(end<section.length&&'0#?'.includes(section[end]))end+=1
  let denominator=0,maxDenominator=0
  const places=end-slash-1
  // 자리 기호를 여섯 개까지만 센다. bestFraction 이 분모를 1 부터 상한까지
  // 하나씩 재므로 상한이 그대로 한 칸을 그리는 비용이고, 이 루프는 메인
  // 스레드에서 돈다 — 배정밀도 값에서 1e-12 안으로 들어오는 분수는 여섯 자리
  // 분모 안에 거의 다 있고, 그보다 넓히면 칸 하나가 그리드를 멈춘다(아홉
  // 자리는 10억 번이다). internal/formula/format_fraction.go 의
  // maxFractionPlaces 와 같은 수여야 한다.
  if(places>0)maxDenominator=10**Math.min(places,MAX_FRACTION_PLACES)-1
  else{
    while(end<section.length&&section[end]>='0'&&section[end]<='9')end+=1
    denominator=Number(section.slice(slash+1,end))
    if(!Number.isInteger(denominator)||denominator<=0)return
  }
  const head=parseFormatSection(section.slice(0,start)),tail=parseFormatSection(section.slice(end))
  return {
    prefix:head.prefix+head.suffix,
    suffix:tail.prefix+tail.suffix,
    // 정수 자리가 없으면 가분수로 적는다 — #/# 은 5.25 를 21/4 로 적는다.
    integer:/[0#?]/.test(head.core)?head.core:'',
    denominator,maxDenominator,
  }
}

function renderFraction(value:number,spec:FractionFormat,locale:string){
  let rest=Math.abs(value),whole=0
  if(spec.integer){whole=Math.floor(rest);rest-=whole}
  let {numerator,denominator}=bestFraction(rest,spec)
  // 0.99 를 `# ?/?` 로 적으면 분자가 분모까지 올라간다. 정수 자리로 올린다.
  if(spec.integer&&numerator>=denominator){whole+=Math.floor(numerator/denominator);numerator%=denominator}
  const useGrouping=spec.integer.includes(',')
  const wholeText=()=>new Intl.NumberFormat(locale,{useGrouping,maximumFractionDigits:0}).format(whole)
  let body:string
  if(spec.integer&&numerator===0)body=wholeText()
  else if(spec.integer&&(whole!==0||spec.integer.includes('0')))body=`${wholeText()} ${numerator}/${denominator}`
  else body=`${numerator}/${denominator}`
  let rendered=spec.prefix+body+spec.suffix
  // -0.02 를 `# ?/?` 로 적으면 0 이 된다. "-0" 은 잘못 그린 것처럼 읽힌다.
  if(value<0&&!(whole===0&&numerator===0)&&!/^-/.test(rendered))rendered='-'+rendered
  return rendered
}

// 값에 가장 가까운 분수를 고른다. 분모를 못 박은 서식이면 그 분모를 그대로
// 쓰고(`?/8` 은 0.5 를 4/8 로 적는다), 아니면 자리 기호가 허락하는 분모를
// 모두 재어 가장 덜 어긋나는 것을 고른다. 같은 만큼 어긋나면 분모가 작은
// 쪽이다 — 0.5 는 `# ??/??` 에서도 1/2 다.
function bestFraction(value:number,spec:FractionFormat){
  if(spec.denominator>0)return{numerator:Math.round(value*spec.denominator),denominator:spec.denominator}
  let denominator=1,numerator=Math.round(value),error=Math.abs(value-numerator)
  for(let candidate=2;candidate<=spec.maxDenominator;candidate+=1){
    const top=Math.round(value*candidate),difference=Math.abs(value-top/candidate)
    if(difference<error-1e-12){denominator=candidate;numerator=top;error=difference}
  }
  return{numerator,denominator}
}

export function wrapText(text:string,maxWidth:number,measure:(text:string)=>number){
  if(maxWidth<=0)return['']
  const lines:string[]=[]
  for(const paragraph of text.split('\n')){
    if(paragraph===''){lines.push('');continue}
    let line=''
    for(const token of paragraph.split(/(\s+)/).filter(Boolean)){
      const candidate=line+token
      if(line&&measure(candidate)>maxWidth){lines.push(line.trimEnd());line=token.trimStart()}else line=candidate
      while(line&&measure(line)>maxWidth){let length=1;while(length<line.length&&measure(line.slice(0,length+1))<=maxWidth)length+=1;lines.push(line.slice(0,length));line=line.slice(length)}
    }
    if(line||paragraph)lines.push(line.trimEnd())
  }
  return lines.length?lines:['']
}

// 날 수를 날짜로 바꾸는 셈은 한 곳에만 둔다. 검증(lib/validation.ts)도
// 이것을 쓴다. 따로 세면 격자에 보이는 날짜와 검증이 읽는 날짜가 어긋난다.
export function spreadsheetDate(value:unknown){
  // 엑셀은 1900년을 윤년으로 잘못 센다. 일련번호 60은 없는 날(1900-02-29)을
  // 가리키므로, 그보다 작은 번호는 하루 뒤에서 세기 시작해야 1900-01-01이
  // 1번이 된다. 서버의 internal/formula 의 serialDate 가 같은 셈을 한다.
  // 음수는 날짜가 아니다. 표 프로그램에 1900년보다 앞선 날은 없다.
  // 서버의 serialDate 도 음수를 되돌린다.
  if(typeof value==='number'&&Number.isFinite(value)&&value>=0)
    return new Date((value<60?Date.UTC(1899,11,31):Date.UTC(1899,11,30))+Math.round(value*86400)*1000)
  if(typeof value==='string'){
    const parsed=new Date(value)
    if(Number.isFinite(parsed.getTime()))return parsed
  }
}

// 서식 기호를 하나씩 읽어 그대로 적는다.
//
// 예전에는 서식을 공백으로 잘라 "날짜 부분 하나"와 "시각 부분 하나"만
// 그렸다. 그래서 "yyyy년 m월 d일" 처럼 토막이 셋인 서식은 앞 토막만
// 그려져 "2024년 m월 d일" 이 되었다. 한국어 날짜 서식이 그대로 깨졌다.
// "dddd" 와 "mmmm" 처럼 요일·달 이름만 적은 서식도 그리지 못했다.
//
// 표 서식에서 m 은 앞뒤를 봐야 뜻이 정해진다. 시 뒤에 오거나 초 앞에
// 오면 분이고, 그 밖에는 달이다. 서버의 internal/formula 의
// renderDatePattern 이 같은 규칙을 쓴다.
const DATE_TOKENS=['am/pm','a/p','yyyy','yy','mmmm','mmm','mm','m','dddd','ddd','dd','d','hh','h','ss','s']

function nextDateToken(format:string,index:number){
  const rest=format.slice(index).toLowerCase()
  for(const token of DATE_TOKENS)if(rest.startsWith(token))return token
  return ''
}

function secondFollows(format:string,index:number){
  while(index<format.length){
    const token=nextDateToken(format,index)
    if(!token){index+=1;continue}
    return token==='ss'||token==='s'
  }
  return false
}

function formatDate(date:Date,format:string,locale:string){
  const normalized=format.replace(/\[([hms]+)]/gi,'$1').replace(/\[[^\]]*]/g,'')
  const twelve=/am\/pm|a\/p/i.test(normalized)
  const pad=(value:number,two:boolean)=>two?String(value).padStart(2,'0'):String(value)
  const monthName=(long:boolean)=>new Intl.DateTimeFormat(locale,{timeZone:'UTC',month:long?'long':'short'}).format(date)
  const dayName=(long:boolean)=>new Intl.DateTimeFormat(locale,{timeZone:'UTC',weekday:long?'long':'short'}).format(date)
  let out='',index=0,previousWasHour=false
  while(index<normalized.length){
    // 따옴표 안은 그대로 적는 글자다. 미리 걷어내면 "day" 의 d 가 날짜
    // 기호로 읽히므로, 여기서 통째로 옮긴다. 한국 파일의 yyyy"년" 이
    // 2023"년" 으로 보이던 것이 이것 때문이다.
    if(normalized[index]==='"'){
      const end=normalized.indexOf('"',index+1)
      out+=normalized.slice(index+1,end<0?undefined:end)
      index=end<0?normalized.length:end+1
      continue
    }
    if(normalized[index]==='\\'){out+=normalized[index+1]??'';index+=2;continue}
    const token=nextDateToken(normalized,index)
    if(!token){out+=normalized[index];index+=1;continue}
    switch(token){
      case'am/pm':out+=date.getUTCHours()<12?'AM':'PM';break
      case'a/p':out+=date.getUTCHours()<12?'A':'P';break
      case'yyyy':out+=String(date.getUTCFullYear());break
      case'yy':out+=pad(date.getUTCFullYear()%100,true);break
      case'mmmm':out+=monthName(true);break
      case'mmm':out+=monthName(false);break
      case'dddd':out+=dayName(true);break
      case'ddd':out+=dayName(false);break
      case'dd':case'd':out+=pad(date.getUTCDate(),token==='dd');break
      case'hh':case'h':{const hour=twelve?(date.getUTCHours()%12||12):date.getUTCHours();out+=pad(hour,token==='hh');break}
      case'ss':case's':out+=pad(date.getUTCSeconds(),token==='ss');break
      case'mm':case'm':
        // 시 뒤에 오거나 초 앞에 오면 분이다. 그 밖에는 달이다.
        out+=previousWasHour||secondFollows(normalized,index+token.length)
          ?pad(date.getUTCMinutes(),token==='mm')
          :pad(date.getUTCMonth()+1,token==='mm')
        break
    }
    previousWasHour=token==='hh'||token==='h'
    index+=token.length
  }
  return out
}

// shiftDecimalPoint 는 소수점만 옮긴다. 100을 곱하면 이진 실수의 어긋남이
// 따라오지만, 사람이 적은 십진수 표기에서 점을 옮기면 따라오지 않는다.
// 서버는 같은 일을 분수로 한다(internal/formula 의 decimalRound).
function shiftDecimalPoint(value:number,places:number){
  if(!Number.isFinite(value)||value===0)return value
  const text=value.toPrecision(15)
  if(text.includes('e')||text.includes('E'))return value*10**places
  const negative=text.startsWith('-'),body=negative?text.slice(1):text
  const point=body.indexOf('.'),digits=body.replace('.',''),whole=point<0?body.length:point
  const moved=whole+places
  let shifted:string
  if(moved<=0)shifted='0.'+'0'.repeat(-moved)+digits
  else if(moved>=digits.length)shifted=digits+'0'.repeat(moved-digits.length)
  else shifted=digits.slice(0,moved)+'.'+digits.slice(moved)
  return Number((negative?'-':'')+shifted)
}

function isDateFormat(format:string){
  const cleaned=format.replace(/"[^"]*"/g,'').replace(/\\./g,'').replace(/\[(?!h+\]|m+\]|s+\])[^\]]*]/gi,'')
  // mmm 과 mmmm 은 달 이름이므로 단위로 오해할 일이 없다. m 하나만으로는
  // 가르지 않는다 — "0.0 m" 처럼 단위로 적은 것을 달로 읽으면 안 된다.
  // 서버의 internal/formula 의 isDatePattern 이 같은 기준을 쓴다.
  return /[ydhs]/i.test(cleaned)||/mmm/i.test(cleaned)
}
