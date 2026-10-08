import { parsePastedNumber } from './clipboardNumber'

/**
 * 엑셀과 구글 시트는 복사할 때 표를 `text/html` 로도 올린다. 평문에는 보이던
 * 글자만 담기지만 이쪽에는 서식과 실제 값이 함께 들어 있다. 평문만 읽으면
 * 굵게·색·정렬이 전부 사라지고 `₩1,234` 같은 값은 글자가 된다.
 */
export type HtmlCell={
  rowOffset:number
  columnOffset:number
  value?:unknown
  style?:Record<string,unknown>
}

const COLOR=/^#[0-9a-f]{3}([0-9a-f]{3})?$/i

/**
 * 글꼴 이름은 CSS 선언 한가운데로 들어가고 나온다. 세미콜론이나 따옴표가 섞이면
 * 선언이 끊어지므로, 글자·숫자·공백·`._-` 만 통과시킨다. `\w` 로 거르면
 * `맑은 고딕` 같은 한글 이름이 전부 떨어진다 — 읽는 쪽과 쓰는 쪽이 같은 자를 쓴다.
 */
const FONT_FAMILY=/^[\p{L}\p{N} ._-]{1,64}$/u

/** 서버가 받아 주는 한도. `internal/workbook/style.go` 의 `ValidateStylePatch` 와
 *  같아야 한다. 여기서 더 넉넉하게 만들면 붙여넣기 전체가 400 으로 떨어진다. */
const MAX_FONT_FAMILY_BYTES=100
export const MIN_FONT_SIZE=6, MAX_FONT_SIZE=72

export function fontFamilyName(value:string|undefined){
  const first=(value??'').split(',')[0].trim().replace(/^["']|["']$/g,'').trim()
  if(first===''||!FONT_FAMILY.test(first))return
  return new TextEncoder().encode(first).length<=MAX_FONT_FAMILY_BYTES?first:undefined
}

/** 표 하나를 찾아 셀의 값과 서식을 읽는다. 표가 없으면 undefined 를 돌려
 *  평문 경로가 그대로 처리하게 둔다. */
export function parseClipboardHtml(html:string,maxCells:number):HtmlCell[]|undefined{
  if(!html.trim())return
  let document:Document
  try{
    document=new DOMParser().parseFromString(html,'text/html')
  }catch{return}
  const table=document.querySelector('table')
  if(!table)return
  const cells:HtmlCell[]=[]
  // rowspan/colspan 이 걸린 셀은 아래·오른쪽 자리를 차지한다. 그 자리를
  // 비워 두지 않으면 뒤따르는 셀이 왼쪽으로 밀려 표가 어긋난다.
  const occupied=new Set<string>()
  let rowOffset=0
  for(const row of Array.from(table.querySelectorAll('tr'))){
    let columnOffset=0
    for(const cell of Array.from(row.children)){
      if(cell.tagName!=='TD'&&cell.tagName!=='TH')continue
      while(occupied.has(`${rowOffset}:${columnOffset}`))columnOffset+=1
      const span=Math.max(1,Math.min(64,Number(cell.getAttribute('colspan'))||1))
      const down=Math.max(1,Math.min(1024,Number(cell.getAttribute('rowspan'))||1))
      for(let extraRow=0;extraRow<down;extraRow+=1)for(let extraColumn=0;extraColumn<span;extraColumn+=1){
        if(extraRow||extraColumn)occupied.add(`${rowOffset+extraRow}:${columnOffset+extraColumn}`)
      }
      const parsed=readCell(cell as HTMLElement)
      if(parsed.value!==undefined||parsed.style)cells.push({rowOffset,columnOffset,...parsed})
      if(cells.length>maxCells)return cells.slice(0,maxCells)
      columnOffset+=span
    }
    rowOffset+=1
  }
  return cells
}

function readCell(element:HTMLElement){
  const text=(element.textContent??'').replace(/ /g,' ').trim()
  const style=readStyle(element)
  const declared=declaredValue(element)
  if(declared!==undefined)return {value:declared,style}
  if(text==='')return {value:undefined,style}
  const number=parsePastedNumber(text)
  if(!number)return {value:text,style}
  if(number.numberFormat)return {value:number.value,style:{...(style??{}),number_format:number.numberFormat}}
  return {value:number.value,style}
}

/**
 * 엑셀은 `x:num`, 구글 시트는 `data-sheets-value` 로 셀이 실제로 담고 있는
 * 값을 알려 준다. 보이던 글자보다 이쪽이 정확하다.
 */
function declaredValue(element:HTMLElement):unknown{
  const excel=element.getAttribute('x:num')
  if(excel!==null){
    const parsed=Number(excel===''?element.textContent??'':excel)
    if(Number.isFinite(parsed))return parsed
  }
  if(element.getAttribute('x:str')!==null)return (element.textContent??'').trim()
  const sheets=element.getAttribute('data-sheets-value')
  if(sheets!==null){
    try{
      const payload=JSON.parse(sheets) as Record<string,unknown>
      // 구글 시트는 종류를 "1" 에, 문자열을 "2", 숫자를 "3" 에 넣는다.
      if(typeof payload['3']==='number')return payload['3']
      if(typeof payload['2']==='string')return payload['2']
    }catch{/* 알아볼 수 없으면 보이던 글자를 쓴다 */}
  }
  return undefined
}

/**
 * 인라인 스타일은 `element.style` 로 읽지 않는다. 이 앱의 CSP 에는
 * `style-src 'unsafe-inline'` 이 없고, 그러면 크롬은 style 속성을 CSSOM 에
 * 넣지 않아 `element.style` 이 전부 빈 값으로 읽힌다. 여기서는 스타일을
 * 적용하는 것이 아니라 읽기만 하므로 속성 문자열을 직접 해석한다.
 */
function declarations(element:HTMLElement){
  const raw=element.getAttribute('style')
  const parsed=new Map<string,string>()
  if(!raw)return parsed
  for(const part of raw.split(';')){
    const separator=part.indexOf(':')
    if(separator<1)continue
    const property=part.slice(0,separator).trim().toLowerCase()
    const value=part.slice(separator+1).trim()
    if(property!==''&&value!=='')parsed.set(property,value)
  }
  return parsed
}

function readStyle(element:HTMLElement):Record<string,unknown>|undefined{
  const style:Record<string,unknown>={}
  const inline=declarations(element)
  const weight=inline.get('font-weight')??inheritedTag(element,'B','STRONG')
  if(weight==='bold'||weight==='bolder'||Number(weight)>=600||weight==='inherited')style.bold=true
  const italic=inline.get('font-style')??inheritedTag(element,'I','EM')
  if(italic==='italic'||italic==='oblique'||italic==='inherited')style.italic=true
  const decoration=(inline.get('text-decoration')??inline.get('text-decoration-line')??'').toLowerCase()
  if(decoration.includes('underline')||inheritedTag(element,'U'))style.underline=true
  // 취소선은 이 앱 어디에서나 `strike` 다. 읽지 않으면 엑셀에서 그은 줄이 사라진다.
  if(decoration.includes('line-through')||inheritedTag(element,'S','STRIKE','DEL'))style.strike=true
  const color=hexColor(inline.get('color')??'')
  if(color)style.color=color
  const background=hexColor(inline.get('background-color')??firstColorToken(inline.get('background')??''))
  if(background)style.background=background
  const align=(inline.get('text-align')??element.getAttribute('align')??'').toLowerCase()
  if(align==='left'||align==='center'||align==='right')style.horizontal_align=align
  const vertical=(inline.get('vertical-align')??element.getAttribute('valign')??'').toLowerCase()
  if(vertical==='top'||vertical==='bottom')style.vertical_align=vertical
  if(vertical==='middle'||vertical==='center')style.vertical_align='middle'
  const wrap=(inline.get('white-space')??'').toLowerCase()
  if(wrap==='normal'||wrap==='pre-wrap'||wrap==='pre-line')style.text_mode='wrap'
  const size=fontSize(inline.get('font-size')??'')
  if(size!==undefined)style.font_size=size
  const family=fontFamilyName(inline.get('font-family')??element.getAttribute('face')??undefined)
  if(family)style.font_family=family
  return Object.keys(style).length>0?style:undefined
}

/** `background:#DBEAFE none repeat` 처럼 축약형에 섞여 있는 색을 꺼낸다.
 *  `rgb(254, 226, 226)` 은 안에 공백이 있으므로 토큰으로 자를 수 없다. */
function firstColorToken(value:string){
  const functional=value.match(/rgba?\([^)]*\)/i)
  if(functional)return functional[0]
  const hex=value.match(/#[0-9a-f]{3}(?:[0-9a-f]{3})?\b/i)
  return hex?hex[0]:''
}

function fontSize(value:string){
  const parsed=Number.parseFloat(value)
  if(!Number.isFinite(parsed))return
  // 엑셀은 pt, 구글 시트는 px 로 쓴다. kanpic 은 pt 로 센다.
  const points=value.trim().endsWith('px')?parsed*0.75:parsed
  const rounded=Math.round(points)
  if(!(rounded>=MIN_FONT_SIZE))return
  // 서버 한도를 넘는 크기는 버리지 않고 한도에 맞춘다. 80pt 제목을 떨어뜨리면
  // 기본 12pt 로 앉지만, 72pt 로 두면 "크다" 는 뜻은 남는다.
  return Math.min(rounded,MAX_FONT_SIZE)
}

function inheritedTag(element:HTMLElement,...tags:string[]){
  return element.querySelector(tags.join(','))?'inherited':undefined
}

/** 인라인 스타일의 색은 브라우저가 rgb() 로 정규화한다. */
function hexColor(value:string|undefined):string|undefined{
  const text=(value??'').trim()
  if(text===''||text==='transparent'||text==='rgba(0, 0, 0, 0)')return
  if(COLOR.test(text))return text.length===4?`#${text[1]}${text[1]}${text[2]}${text[2]}${text[3]}${text[3]}`.toUpperCase():text.toUpperCase()
  const match=text.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)$/)
  if(!match)return
  if(match[4]!==undefined&&Number(match[4])===0)return
  const channels=[match[1],match[2],match[3]].map(part=>Number(part))
  if(channels.some(part=>!Number.isFinite(part)||part<0||part>255))return
  return '#'+channels.map(part=>part.toString(16).padStart(2,'0')).join('').toUpperCase()
}
