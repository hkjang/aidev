import { address } from './api'
import { formatCellValue } from './cellFormat'
import { cellKey } from '../state/editor'
import type { Cell } from '../types'
import type { GridRegion } from './dataRegion'

export type PrintOptions={
  title:string
  sheetName:string
  region?:GridRegion
  // printArea 는 시트에 정해 둔 인쇄 영역이다. "A1:D20" 꼴.
  printArea?:string
  gridlines:boolean
  headers:boolean
  /**
   * Rows the reader cannot see — filtered out, hidden, or folded into a
   * collapsed group. Printing them would hand somebody a page that disagrees
   * with the screen they printed it from.
   */
  hiddenRows?:Set<number>
  /**
   * 조건부 서식이 화면에서 칠한 결과. 셀에 저장된 서식이 아니라 값에 따라
   * 그때그때 정해지는 것이라 따로 받는다. 이것이 없으면 사람이 읽으라고
   * 칠해 놓은 표가 종이에서는 아무 표시 없는 숫자 뭉치가 된다.
   */
  conditional?:Map<string,{style?:Record<string,unknown>;bar?:{color:string;ratio:number};icon?:{style:string;index:number;count:number}}>
  /**
   * 열의 실제 너비(px). 주지 않으면 기본 너비로 셈한다. 이것이 있어야 종이
   * 한 장에 몇 열이 들어가는지 알 수 있다.
   */
  columnWidth?:(column:number)=>number
  /**
   * 화면에서 고정해 둔 행의 수. 사용자가 "여기까지가 머리글" 이라고 이미
   * 말해 둔 것이므로, 종이에서도 장마다 다시 찍는다. 엑셀의 인쇄 제목이다.
   */
  frozenRows?:number
  /**
   * 종이를 세로로 쓸지 가로로 쓸지. 열이 많은 표는 가로로 놓으면 한 장에
   * 훨씬 많이 들어간다.
   */
  orientation?:PrintOrientation
  /**
   * 여백. 좁게 잡으면 한 장에 더 들어가지만 프린터가 잘라 먹을 수 있다.
   */
  margin?:PrintMargin
  /**
   * 'width' 로 하면 열을 다음 장으로 넘기는 대신 표 전체를 줄여 한 장
   * 너비에 맞춘다. 열이 서른 개인 표를 장마다 나누어 보는 것보다 한눈에
   * 보는 편이 나을 때가 있다.
   */
  fit?:PrintFit
}

export type PrintOrientation='portrait'|'landscape'
export type PrintMargin='narrow'|'normal'|'wide'
export type PrintFit='none'|'width'

/** 여백의 실제 크기(mm). 좌우 합쳐 이만큼이 종이에서 빠진다. */
export const PRINT_MARGINS:Record<PrintMargin,number>={narrow:8,normal:14,wide:22}

/** A4 의 짧은 쪽과 긴 쪽(mm). 브라우저가 실제로 어떤 용지를 쓸지는 알 수
 *  없으므로 가장 흔한 A4 를 기준으로 삼는다. */
const A4_SHORT=210
const A4_LONG=297
/** 96dpi 에서 1mm 는 약 3.7795px 이다. */
const PX_PER_MM=3.7795

/** 한 장에 들어가는 가로 폭(px). 방향과 여백에 따라 달라진다. */
export function printableWidth(orientation:PrintOrientation='portrait',margin:PrintMargin='normal'){
  const paper=orientation==='landscape'?A4_LONG:A4_SHORT
  return Math.floor((paper-PRINT_MARGINS[margin]*2)*PX_PER_MM)
}

/**
 * 장마다 다시 찍는 머리글 행의 최대 수. 너무 많이 반복하면 정작 데이터가
 * 들어갈 자리가 없어진다. 다섯 줄이면 한 장의 십분의 일쯤이다.
 */
const MAX_REPEATED_HEADER_ROWS=5

/**
 * 한 장에 들어가는 가로 폭(px). A4 세로에 좌우 14mm 여백이면 210 − 28 =
 * 182mm 이고, 96dpi 에서 1mm 는 약 3.78px 이므로 688px 이다. 브라우저가
 * 실제로 어떤 용지를 쓸지는 알 수 없으므로 가장 흔한 A4 세로를 기준으로
 * 삼는다. 넘치면 다음 장으로 넘길 뿐 잘리지는 않는다.
 */
const PRINTABLE_WIDTH=688
const ROW_HEAD_WIDTH=34
const DEFAULT_PRINT_COLUMN_WIDTH=108

/**
 * 종이 한 장에 들어갈 만큼씩 열을 끊는다. 예전에는 표를 종이 폭에 맞춰
 * 통째로 눌러 담아, 열이 서른 개면 한 열이 몇 밀리미터로 찌그러졌다.
 */
export function columnPages(startColumn:number,endColumn:number,widthOf:(column:number)=>number,pageWidth:number=PRINTABLE_WIDTH){
  const pages:Array<{start:number;end:number;widths:number[]}>=[]
  let current:{start:number;end:number;widths:number[]}|undefined
  let used=0
  for(let column=startColumn;column<=endColumn;column+=1){
    const width=Math.max(1,Math.round(widthOf(column)))
    // 한 열이 종이보다 넓어도 혼자서는 한 장을 차지해야 한다. 그렇지 않으면
    // 어느 장에도 들어가지 못한다.
    if(current&&used+width>pageWidth-ROW_HEAD_WIDTH){current=undefined;used=0}
    if(!current){current={start:column,end:column,widths:[]};pages.push(current)}
    current.end=column
    current.widths.push(width)
    used+=width
  }
  return pages
}

/**
 * 표가 종이에서 가로로 몇 장에 나뉘는지, 어느 열이 어느 장에 가는지.
 *
 * 미리보기와 실제 인쇄가 이 하나를 같이 쓴다. 따로 세면 "한 장에 들어갑니다"
 * 라고 보여 주고 두 장을 뱉는 일이 생긴다 — 미리보기가 틀리면 없느니만
 * 못하다.
 */
export function printColumnPages(region:GridRegion,widthOf:(column:number)=>number,orientation:PrintOrientation='portrait',margin:PrintMargin='normal',fit:PrintFit='none'){
  // 너비에 맞추라고 하면 열을 넘기지 않는다. 표 전체를 줄여 한 장에 담는다.
  if(fit==='width')return [{start:region.startColumn,end:region.endColumn,
    widths:Array.from({length:region.endColumn-region.startColumn+1},(_,index)=>Math.max(1,Math.round(widthOf(region.startColumn+index))))}]
  return columnPages(region.startColumn,region.endColumn,widthOf,printableWidth(orientation,margin))
}

const escapeHTML=(value:string)=>value.replace(/[&<>"]/g,character=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[character] as string))

// 시트가 가진 넓이. 서버도 같은 값으로 인쇄 영역을 받지만, 그 검사가 생기기
// 전에 저장된 값이 남아 있을 수 있어 읽는 쪽에서도 본다. 아래 layout 은 영역의
// 줄마다 한 줄씩 짜므로, 시트에 없는 줄을 세면 화면이 멈춘다.
const SHEET_ROWS=1048576, SHEET_COLUMNS=16384

/** The smallest rectangle that holds every non-empty cell of the sheet. */
// printAreaRegion 은 "A1:D20" 을 격자 범위로 옮긴다. 읽을 수 없는 값이면
// 아무것도 돌려주지 않아 내용이 있는 곳 전체를 내게 한다.
export function printAreaRegion(area:string|undefined):GridRegion|undefined{
  const parsed=/^([A-Z]+)([1-9]\d*):([A-Z]+)([1-9]\d*)$/.exec((area??'').trim().toUpperCase())
  if(!parsed)return
  const column=(letters:string)=>{
    let value=0
    for(const letter of letters){
      value=value*26+letter.charCodeAt(0)-64
      // 루프 안에서 본다. 글자가 몇 개 더 붙으면 수가 배정밀도의 정수 범위를
      // 넘어, 빠져나온 뒤에 재면 멀쩡한 열 번호로 보일 수 있다.
      if(value>SHEET_COLUMNS)return 0
    }
    return value
  }
  const startRow=Number(parsed[2]),endRow=Number(parsed[4])
  const startColumn=column(parsed[1]),endColumn=column(parsed[3])
  // 시트 밖을 가리키는 값이면 인쇄 영역이 없는 것으로 친다 — 내용이 있는
  // 곳 전체를 내는 쪽이, 아무것도 못 내는 것보다 낫다.
  if(startColumn<1||endColumn<1||startRow>SHEET_ROWS||endRow>SHEET_ROWS)return
  return{startRow:Math.min(startRow,endRow),startColumn:Math.min(startColumn,endColumn),
         endRow:Math.max(startRow,endRow),endColumn:Math.max(startColumn,endColumn)}
}

export function usedRegion(cells:Map<string,Cell>):GridRegion|undefined{
  let startRow=Infinity,startColumn=Infinity,endRow=0,endColumn=0
  cells.forEach(cell=>{
    const empty=(cell.value==null||cell.value==='')&&!cell.formula
    if(empty)return
    startRow=Math.min(startRow,cell.row);startColumn=Math.min(startColumn,cell.column)
    endRow=Math.max(endRow,cell.row);endColumn=Math.max(endColumn,cell.column)
  })
  if(endRow===0||endColumn===0)return undefined
  return {startRow,startColumn,endRow,endColumn}
}

// 셀 서식의 값은 사람이 적은 것이 그대로 온다. 인쇄물의 style 속성에 그대로
// 넣으면 값 하나가 속성을 빠져나가 문서 전체의 모양을 바꿀 수 있다. 인쇄
// 문서에서는 스크립트도 바깥 연결도 막혀 있으므로 훔칠 것은 없지만, 자기
// 인쇄물이 엉뚱하게 나오는 것도 고장이다.
const safeColor=(value:string)=>/^(#[0-9a-fA-F]{3,8}|[a-zA-Z]{3,20}|rgba?\([0-9.,%\s]+\))$/.test(value.trim())?value.trim():''
const safeFont=(value:string)=>/^[\w\s가-힣,'-]{1,120}$/.test(value.trim())?value.trim():''
const safeKeyword=(value:string,allowed:string[])=>allowed.includes(value)?value:''

// cellCSSForTest exposes the style builder so the escaping above can be tested
// on its own; building a whole document to check one attribute would hide what
// the test is actually about.
export function cellCSSForTest(style?:Record<string,unknown>){return cellCSS(style)}

function cellCSS(style?:Record<string,unknown>){
  if(!style)return ''
  const rules:string[]=[]
  if(style.bold===true)rules.push('font-weight:700')
  if(style.italic===true)rules.push('font-style:italic')
  const decorations=[style.underline===true?'underline':'',style.strike===true?'line-through':''].filter(Boolean)
  if(decorations.length)rules.push(`text-decoration:${decorations.join(' ')}`)
  if(typeof style.color==='string'&&safeColor(style.color))rules.push(`color:${safeColor(style.color)}`)
  if(typeof style.background==='string'&&safeColor(style.background))rules.push(`background:${safeColor(style.background)}`)
  if(typeof style.font_size==='number'&&Number.isFinite(style.font_size))rules.push(`font-size:${Math.max(4,Math.min(400,style.font_size))}px`)
  if(typeof style.font_family==='string'&&safeFont(style.font_family))rules.push(`font-family:${safeFont(style.font_family)}`)
  if(typeof style.horizontal_align==='string'){
    const align=safeKeyword(style.horizontal_align,['left','center','right','justify'])
    if(align)rules.push(`text-align:${align}`)
  }
  if(typeof style.vertical_align==='string'){
    const align=safeKeyword(style.vertical_align,['top','middle','bottom'])
    if(align)rules.push(`vertical-align:${align}`)
  }
  if(style.text_mode==='wrap'||style.wrap===true)rules.push('white-space:pre-wrap')
  return rules.join(';')
}

function rowCells(cells:Map<string,Cell>,row:number,startColumn:number,endColumn:number,headers:boolean,conditional?:PrintOptions['conditional']){
  const columns:string[]=headers?[`<th class="row-head">${row}</th>`]:[]
  for(let column=startColumn;column<=endColumn;column+=1){
    const key=cellKey(row,column)
    const cell=cells.get(key)
    const painted=conditional?.get(key)
    const text=cell?formatCellValue(cell.value,cell.style):''
    // 조건부 서식이 셀 서식 위에 얹힌다 — 화면과 같은 차례다.
    const style={...(cell?.style??{}),...(painted?.style??{})}
    const declarations=[cellCSS(style),dataBarCSS(painted?.bar)].filter(Boolean).join(';')
    const mark=painted?.icon?`<span class="mark">${escapeHTML(printIcon(painted.icon))}</span>`:''
    columns.push(`<td style="${declarations}">${mark}${escapeHTML(text)}</td>`)
  }
  return columns.join('')
}

// 데이터 막대는 종이에서 그라디언트로 그린다. 화면의 반투명 막대와 같은 자리에
// 같은 비율로 선다.
function dataBarCSS(bar?:{color:string;ratio:number}){
  if(!bar||!safeColor(bar.color))return ''
  const width=Math.round(Math.max(0,Math.min(1,bar.ratio))*100)
  return `background:linear-gradient(to right,${safeColor(bar.color)} ${width}%,transparent ${width}%)`
}

// 아이콘은 글자로 찍는다. 흑백으로 인쇄해도 모양이 남는 것을 고른다.
const PRINT_ICONS:Record<string,string[]>={
  '3TrafficLights1':['●','●','●'],
  '3Arrows':['▼','▶','▲'],
  '3Symbols':['✗','!','✓'],
  '4Arrows':['▼','↘','↗','▲'],
  '5Arrows':['▼','↘','▶','↗','▲'],
  '5Quarters':['○','◔','◑','◕','●'],
}

function printIcon(icon:{style:string;index:number;count:number}){
  const set=PRINT_ICONS[icon.style]
  if(!set)return ''
  return set[Math.min(Math.max(icon.index,0),set.length-1)]??''
}

/**
 * Renders the sheet as a plain HTML table. The canvas grid cannot be printed
 * directly, so printing goes through a document the browser can paginate.
 */
export function printableDocument(cells:Map<string,Cell>,options:PrintOptions){
  const orientation=options.orientation??'portrait'
  const margin=options.margin??'normal'
  const fit=options.fit??'none'
  const pageWidth=printableWidth(orientation,margin)
  // 인쇄 영역을 정해 두었으면 그 범위만 낸다. 정해 두지 않았으면 내용이
  // 있는 곳 전체를 낸다 — 사람이 따로 말하기 전까지는 그쪽이 기대하는 바다.
  const region=options.region??printAreaRegion(options.printArea)??usedRegion(cells)
  const widthOf=options.columnWidth??(()=>DEFAULT_PRINT_COLUMN_WIDTH)
  const tables:string[]=[]
  let printedRows=0
  // 가장 넓은 표가 종이보다 넓으면 그만큼 줄여야 한다.
  let widestTable=0
  if(region){
    // 종이 폭을 넘는 열은 다음 장으로 넘긴다. 장마다 열 머리글과 행 번호를
    // 다시 찍어야 어느 칸이 무슨 열의 몇 행인지 알 수 있다.
    // 너비에 맞추라고 하면 열을 다음 장으로 넘기지 않는다. 대신 아래에서
    // 표 전체를 줄여 한 장 너비에 담는다.
    const pages=printColumnPages(region,widthOf,orientation,margin,fit)
    for(const page of pages){
      const columnGroup=[options.headers?`<col style="width:${ROW_HEAD_WIDTH}px">`:'']
        .concat(page.widths.map(width=>`<col style="width:${width}px">`)).join('')
      const headLines:string[]=[]
      if(options.headers){
        const headerCells=[`<th class="corner"></th>`]
        for(let column=page.start;column<=page.end;column+=1)headerCells.push(`<th>${escapeHTML(address(1,column).replace(/\d+$/,''))}</th>`)
        headLines.push(`<tr>${headerCells.join('')}</tr>`)
      }
      // 고정한 행은 thead 안에 둔다. 브라우저가 장마다 다시 찍어 주므로
      // 둘째 장부터도 어느 칸이 무슨 뜻인지 알 수 있다.
      const frozenEnd=Math.min(region.startRow+Math.min(options.frozenRows??0,MAX_REPEATED_HEADER_ROWS)-1,region.endRow)
      for(let row=region.startRow;row<=frozenEnd;row+=1){
        if(options.hiddenRows?.has(row))continue
        headLines.push(`<tr class="frozen">${rowCells(cells,row,page.start,page.end,options.headers,options.conditional)}</tr>`)
      }
      const head=headLines.length>0?`<thead>${headLines.join('')}</thead>`:''
      const rows:string[]=[]
      for(let row=frozenEnd+1;row<=region.endRow;row+=1){
        if(options.hiddenRows?.has(row))continue
        rows.push(`<tr>${rowCells(cells,row,page.start,page.end,options.headers,options.conditional)}</tr>`)
      }
      printedRows=rows.length+headLines.length
      if(rows.length===0)continue
      widestTable=Math.max(widestTable,page.widths.reduce((total,width)=>total+width,0)+(options.headers?ROW_HEAD_WIDTH:0))
      tables.push(`<table><colgroup>${columnGroup}</colgroup>${head}<tbody>${rows.join('')}</tbody></table>`)
    }
  }
  // 표가 종이보다 넓으면 줄여서 담는다. transform 대신 zoom 을 쓰는 이유는
  // zoom 은 줄인 크기로 다시 흘려 주어 장 나눔이 그대로 살기 때문이다.
  const zoomRule=fit==='width'&&widestTable>pageWidth
    ? `;zoom:${(pageWidth/widestTable).toFixed(4)}`
    : ''
  const empty=printedRows===0?'<p class="empty">인쇄할 데이터가 없습니다.</p>':''
  return `<!doctype html><html lang="ko"><head><meta charset="utf-8"><title>${escapeHTML(options.title)}</title><style>
    td .mark{margin-right:4px}
  @page{size:A4 ${orientation};margin:${PRINT_MARGINS[margin]}mm}
  body{font:12px Inter,Pretendard,'Malgun Gothic',sans-serif;color:#1c2b33;margin:0${zoomRule}}
  header{display:flex;justify-content:space-between;align-items:baseline;margin-bottom:10px}
  h1{font-size:15px;margin:0}
  header span{font-size:11px;color:#61727c}
  /* 폭에 맞춰 눌러 담지 않는다. 열은 화면에서와 같은 너비로 찍히고,
     한 장에 안 들어가면 다음 장으로 넘어간다. */
  table{border-collapse:collapse;table-layout:fixed}
  table+table{page-break-before:always;margin-top:10px}
  td,th{border:${options.gridlines?'1px solid #d6dee2':'none'};padding:3px 6px;font-weight:inherit;text-align:left;vertical-align:bottom}
  th{background:#f1f5f7;font-weight:600;text-align:center;color:#4a5b65}
  th.row-head{width:${ROW_HEAD_WIDTH}px}
  td,th{overflow:hidden;text-overflow:ellipsis}
  .empty{color:#7d8b94}
  tr{page-break-inside:avoid}
  thead{display:table-header-group}
  thead tr.frozen th.row-head{background:#f1f5f7}
  thead tr.frozen td{background:#fafcfd;font-weight:600}
  </style></head><body><header><h1>${escapeHTML(options.title)}</h1><span>${escapeHTML(options.sheetName)}${region?` · ${address(region.startRow,region.startColumn)}:${address(region.endRow,region.endColumn)}`:''}</span></header>${empty||tables.join('')}</body></html>`
}
