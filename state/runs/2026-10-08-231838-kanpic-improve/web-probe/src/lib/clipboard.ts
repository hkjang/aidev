import type { HtmlCell } from './clipboardHtml'
import { leadingNumberSeriesValue, listSeriesValue } from './fillSeries'
import { parsePastedNumber } from './clipboardNumber'

export const KANPIC_CLIPBOARD_TYPE = 'application/x-kanpic-cells+json'
export const MAX_PASTE_CELLS = 10_000
// 정렬은 범위 전체를 한 번에 다시 쓰지만, 편집을 이어서 하는 것이 아니라
// 이미 가진 데이터에 대한 한 번의 동작이라 자체 한도를 가진다. 서버의
// MaxSortCells와 같은 값이어야 화면에서 막힌 것이 서버에서도 막힌다.
export const MAX_SORT_CELLS = 60_000
// 편집기가 보여 주는 행의 수. 서버는 이보다 훨씬 큰 시트를 저장하고
// 내보내기에도 담지만, 이 값 너머의 행은 화면에서 아예 닿을 수 없었다.
// 2만 행짜리 시트에서 A20000으로 이동하면 A10000에 멈췄다.
//
// 200,000으로 둔 이유: 격자는 전체 높이만큼의 스크롤 영역을 만든다.
// 200,000행 × 27px × 최대 확대 2배 = 약 1,080만 px로, 브라우저가 요소
// 높이를 잘라 내기 시작하는 지점(파이어폭스 약 1,790만 px)보다 낮다.
// 엑셀과 같은 1,048,576행은 확대 상태에서 그 한계를 넘어 스크롤 위치가
// 조용히 어긋난다.
export const MAX_GRID_ROWS = 200_000
// 편집기가 보여 주는 열의 수. 서버는 엑셀과 같은 16,384열(XFD)까지 받고
// 저장하지만, 500번째 열(SF) 너머는 화면에서 닿을 수 없었다. 이름 상자에
// SG1을 넣으면 SF1로 잘렸다.
//
// 행과 달리 여기서는 타협할 이유가 없다. 16,384열 × 108px × 최대 확대
// 2배는 약 354만 px로, 브라우저의 요소 크기 한계에서 한참 멀다.
export const MAX_GRID_COLUMNS = 16_384

export type ClipboardCell = {
  rowOffset:number
  columnOffset:number
  value?:unknown
  formula?:string
  style?:Record<string,unknown>
}

export type KanpicClipboard = {
  version:1
  sourceRow:number
  sourceColumn:number
  rows:number
  columns:number
  cells:ClipboardCell[]
}

export type PastedCell = {
  row:number
  column:number
  value?:unknown
  formula?:string
  style?:Record<string,unknown>
}

export type FillRange = {
  startRow:number
  startColumn:number
  endRow:number
  endColumn:number
}

function columnNumber(name:string){
  let value=0
  for(const character of name.toUpperCase())value=value*26+character.charCodeAt(0)-64
  return value
}

function columnName(column:number){
  let value=column,result=''
  while(value>0){value-=1;result=String.fromCharCode(65+value%26)+result;value=Math.floor(value/26)}
  return result
}

function shiftFormulaSegment(segment:string,rowDelta:number,columnDelta:number){
  return segment.replace(/(^|[^A-Za-z0-9_.])(\$?)([A-Za-z]{1,3})(\$?)([1-9]\d*)(?![A-Za-z0-9_])/g,(
    _match:string,prefix:string,columnAbsolute:string,columnLetters:string,rowAbsolute:string,rowDigits:string,
  )=>{
    const column=columnNumber(columnLetters)+(columnAbsolute?0:columnDelta)
    const row=Number(rowDigits)+(rowAbsolute?0:rowDelta)
    if(column<1||row<1)return `${prefix}#REF!`
    return `${prefix}${columnAbsolute}${columnName(column)}${rowAbsolute}${row}`
  })
}

export function shiftFormula(formula:string,rowDelta:number,columnDelta:number){
  if(!formula||(!rowDelta&&!columnDelta))return formula
  let result='',segment='',inString=false
  const flush=()=>{result+=inString?segment:shiftFormulaSegment(segment,rowDelta,columnDelta);segment=''}
  for(let index=0;index<formula.length;index+=1){
    const character=formula[index]
    if(character==='"'){
      if(inString&&formula[index+1]==='"'){segment+='""';index+=1;continue}
      flush();result+='"';inString=!inString
    }else segment+=character
  }
  flush()
  return result
}

export function parseTabularText(text:string){
  const rows:string[][]=[]
  let row:string[]=[],value='',quoted=false
  for(let index=0;index<text.length;index+=1){
    const character=text[index]
    if(character==='"'){
      if(quoted&&text[index+1]==='"'){value+='"';index+=1}else quoted=!quoted
    }else if(character==='\t'&&!quoted){row.push(value);value=''}
    else if((character==='\n'||character==='\r')&&!quoted){
      if(character==='\r'&&text[index+1]==='\n')index+=1
      row.push(value);rows.push(row);row=[];value=''
    }else value+=character
  }
  row.push(value)
  if(row.length>1||row[0]!==''||rows.length===0)rows.push(row)
  return rows
}

export function parseClipboardPayload(raw:string|undefined):KanpicClipboard|undefined{
  if(!raw)return
  try{
    const value=JSON.parse(raw) as Partial<KanpicClipboard>
    if(value.version!==1||!Number.isInteger(value.sourceRow)||!Number.isInteger(value.sourceColumn)||!Number.isInteger(value.rows)||!Number.isInteger(value.columns)||!Array.isArray(value.cells))return
    if(value.sourceRow!<1||value.sourceColumn!<1)return
    if(value.rows!<1||value.columns!<1||value.rows!>MAX_PASTE_CELLS||value.columns!>MAX_PASTE_CELLS)return
    if(value.rows!*value.columns!>MAX_PASTE_CELLS)return
    if(value.cells.length!==value.rows!*value.columns!)return
    if(value.cells.some(cell=>!Number.isInteger(cell.rowOffset)||!Number.isInteger(cell.columnOffset)||cell.rowOffset<0||cell.columnOffset<0||cell.rowOffset>=value.rows!||cell.columnOffset>=value.columns!))return
    return value as KanpicClipboard
  }catch{return}
}

function parsedValue(raw:string):{value:unknown;style?:Record<string,unknown>}{
  if(raw==='')return {value:undefined}
  if(raw.toLowerCase()==='true')return {value:true}
  if(raw.toLowerCase()==='false')return {value:false}
  // 다른 스프레드시트의 평문에는 보이던 글자가 담긴다. `₩1,234` 를 글자로
  // 저장하면 합계가 0이 되므로, 숫자로 읽고 보이던 모습은 서식으로 남긴다.
  const number=parsePastedNumber(raw)
  if(number)return {value:number.value,style:number.numberFormat?{number_format:number.numberFormat}:undefined}
  return {value:raw}
}

/**
 * How a paste treats the copied cells. `values` drops formulas, `format` keeps
 * only the styling and `transpose` swaps rows with columns, matching the paste
 * special options people expect from a spreadsheet.
 */
export type PasteMode='all'|'values'|'format'|'transpose'

export function materializePaste(text:string,internalRaw:string|undefined,startRow:number,startColumn:number,mode:PasteMode|boolean='all',htmlCells?:HtmlCell[]){
  const resolved:PasteMode=mode===true?'values':mode===false?'all':mode
  const valuesOnly=resolved==='values',transpose=resolved==='transpose'
  const internal=parseClipboardPayload(internalRaw)
  if(internal){
    if(internal.rows*internal.columns>MAX_PASTE_CELLS)throw new Error(`붙여넣기는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`)
    return validateGridBounds(internal.cells.map(cell=>{
      const rowOffset=transpose?cell.columnOffset:cell.rowOffset,columnOffset=transpose?cell.rowOffset:cell.columnOffset
      if(resolved==='format')return {row:startRow+rowOffset,column:startColumn+columnOffset,style:cell.style}
      return {
        row:startRow+rowOffset,
        column:startColumn+columnOffset,
        value:valuesOnly?cell.value:cell.formula?undefined:cell.value,
        // A transposed formula cannot keep its relative references straight, so
        // it is pasted as text-free value the same way values-only pastes are.
        formula:valuesOnly||transpose?undefined:cell.formula?shiftFormula(cell.formula,startRow-internal.sourceRow,startColumn-internal.sourceColumn):undefined,
        style:cell.style,
      }
    }))
  }
  // 다른 스프레드시트에서 온 표는 평문보다 HTML 쪽이 더 많이 알고 있다.
  // kanpic에서 복사한 셀이 있을 때는 그쪽이 언제나 더 정확하므로 뒤에 온다.
  if(htmlCells&&htmlCells.length>0){
    if(htmlCells.length>MAX_PASTE_CELLS)throw new Error(`붙여넣기는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`)
    if(resolved==='format')return []
    return validateGridBounds(htmlCells.map(cell=>{
      const rowOffset=transpose?cell.columnOffset:cell.rowOffset,columnOffset=transpose?cell.rowOffset:cell.columnOffset
      const formula=!valuesOnly&&!transpose&&typeof cell.value==='string'&&cell.value.startsWith('=')?cell.value:undefined
      return {row:startRow+rowOffset,column:startColumn+columnOffset,value:formula?undefined:cell.value,formula,style:valuesOnly?undefined:cell.style}
    }))
  }
  const parsed=parseTabularText(text)
  const rows=transpose?transposeRows(parsed):parsed
  const count=rows.reduce((total,row)=>total+row.length,0)
  if(count>MAX_PASTE_CELLS)throw new Error(`붙여넣기는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`)
  if(resolved==='format')return []
  return validateGridBounds(rows.flatMap((row,rowOffset)=>row.map((raw,columnOffset)=>{
    const formula=!valuesOnly&&!transpose&&raw.startsWith('=')?raw:undefined
    if(formula)return {row:startRow+rowOffset,column:startColumn+columnOffset,formula}
    const parsedCell=parsedValue(raw)
    return {row:startRow+rowOffset,column:startColumn+columnOffset,value:parsedCell.value,style:valuesOnly?undefined:parsedCell.style}
  })))
}

function transposeRows(rows:string[][]){
  const width=rows.reduce((max,row)=>Math.max(max,row.length),0)
  const result:string[][]=[]
  for(let column=0;column<width;column+=1)result.push(rows.map(row=>row[column]??''))
  return result
}

function positiveModulo(value:number,divisor:number){return((value%divisor)+divisor)%divisor}
function arithmeticStep(values:number[]){if(values.length<2)return;const step=values[1]-values[0];return values.every((value,index)=>index===0||value-values[index-1]===step)?step:undefined}
function isoDateValue(value:unknown){if(typeof value!=='string'||!/^\d{4}-\d{2}-\d{2}$/.test(value))return;const date=new Date(`${value}T00:00:00Z`);if(Number.isNaN(date.getTime())||date.toISOString().slice(0,10)!==value)return;return date.getTime()}
function trailingNumber(value:unknown){if(typeof value!=='string')return;const match=value.match(/^(.*?)(-?\d+)$/);if(!match)return;return{prefix:match[1],number:Number(match[2]),width:match[2].replace('-','').length}}
function seriesValue(values:ClipboardCell[],position:number){
  if(values.some(cell=>cell.formula))return{matched:false as const}
  const raw=values.map(cell=>cell.value)
  if(raw.every(value=>typeof value==='number')){
    const numbers=raw as number[],step=arithmeticStep(numbers)
    if(step!==undefined)return{matched:true as const,value:numbers[0]+step*position}
  }
  const dates=raw.map(isoDateValue)
  if(dates.every(value=>value!==undefined)){
    const timestamps=dates as number[],day=86_400_000,step=timestamps.length===1?day:arithmeticStep(timestamps)
    if(step!==undefined)return{matched:true as const,value:new Date(timestamps[0]+step*position).toISOString().slice(0,10)}
  }
  // 이름 목록이 먼저다. `1월` 은 앞자리 숫자 규칙에도 걸리지만 12월 다음은
  // 13월이 아니라 1월이어야 하고, 그 되돌아옴은 목록만 안다.
  const listed=listSeriesValue(raw,position)
  if(listed!==undefined)return {matched:true as const,value:listed}
  const numbered=raw.map(trailingNumber)
  if(numbered.every(value=>value!==undefined)){
    const items=numbered as Array<{prefix:string;number:number;width:number}>,samePrefix=items.every(item=>item.prefix===items[0].prefix)
    const step=items.length===1?1:arithmeticStep(items.map(item=>item.number))
    if(samePrefix&&step!==undefined){const number=items[0].number+step*position,sign=number<0?'-':'';return{matched:true as const,value:`${items[0].prefix}${sign}${String(Math.abs(number)).padStart(items[0].width,'0')}`}}
  }
  const leading=leadingNumberSeriesValue(raw,position)
  if(leading!==undefined)return {matched:true as const,value:leading}
  return{matched:false as const}
}

export function materializeFill(payload:KanpicClipboard,target:FillRange){
  const sourceEndRow=payload.sourceRow+payload.rows-1,sourceEndColumn=payload.sourceColumn+payload.columns-1
  if(target.startRow<1||target.startColumn<1||target.endRow>MAX_GRID_ROWS||target.endColumn>MAX_GRID_COLUMNS||target.startRow>payload.sourceRow||target.startColumn>payload.sourceColumn||target.endRow<sourceEndRow||target.endColumn<sourceEndColumn)throw new Error('자동 채우기 대상은 원본 선택 범위를 포함해야 합니다.')
  const targetCount=(target.endRow-target.startRow+1)*(target.endColumn-target.startColumn+1)
  if(targetCount>MAX_PASTE_CELLS)throw new Error(`자동 채우기는 원본을 포함해 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`)
  const byOffset=new Map(payload.cells.map(cell=>[`${cell.rowOffset}:${cell.columnOffset}`,cell]))
  const sourceCell=(rowOffset:number,columnOffset:number)=>byOffset.get(`${rowOffset}:${columnOffset}`)
  const vertical=payload.columns===1&&target.startColumn===payload.sourceColumn&&target.endColumn===sourceEndColumn
  const horizontal=payload.rows===1&&target.startRow===payload.sourceRow&&target.endRow===sourceEndRow
  const ordered=vertical?Array.from({length:payload.rows},(_,index)=>sourceCell(index,0)!):horizontal?Array.from({length:payload.columns},(_,index)=>sourceCell(0,index)!):[]
  const cells:PastedCell[]=[]
  for(let row=target.startRow;row<=target.endRow;row+=1)for(let column=target.startColumn;column<=target.endColumn;column+=1){
    if(row>=payload.sourceRow&&row<=sourceEndRow&&column>=payload.sourceColumn&&column<=sourceEndColumn)continue
    const rowOffset=positiveModulo(row-payload.sourceRow,payload.rows),columnOffset=positiveModulo(column-payload.sourceColumn,payload.columns)
    const source=sourceCell(rowOffset,columnOffset)
    if(!source)throw new Error('자동 채우기 원본 셀이 올바르지 않습니다.')
    const position=vertical?row-payload.sourceRow:horizontal?column-payload.sourceColumn:0
    const series=ordered.length?seriesValue(ordered,position):{matched:false as const}
    const formula=source.formula?shiftFormula(source.formula,row-(payload.sourceRow+source.rowOffset),column-(payload.sourceColumn+source.columnOffset)):undefined
    cells.push({row,column,value:formula?undefined:series.matched?series.value:source.value,formula,style:source.style?{...source.style}:undefined})
  }
  return cells
}

function validateGridBounds(cells:PastedCell[]){
  if(cells.some(cell=>cell.row<1||cell.row>MAX_GRID_ROWS||cell.column<1||cell.column>MAX_GRID_COLUMNS))throw new Error(`붙여넣을 범위가 시트 한도(${MAX_GRID_ROWS.toLocaleString()}행 × ${MAX_GRID_COLUMNS.toLocaleString()}열)를 벗어났습니다.`)
  return cells
}

function quoteTSV(value:string){
  return /[\t\r\n"]/.test(value)?`"${value.replace(/"/g,'""')}"`:value
}

export function clipboardText(payload:KanpicClipboard){
  const byOffset=new Map(payload.cells.map(cell=>[`${cell.rowOffset}:${cell.columnOffset}`,cell]))
  const rows:string[]=[]
  for(let row=0;row<payload.rows;row+=1){
    const values:string[]=[]
    for(let column=0;column<payload.columns;column+=1){
      const cell=byOffset.get(`${row}:${column}`)
      const value=cell?.formula||(cell?.value==null?'':String(cell.value))
      values.push(quoteTSV(value))
    }
    rows.push(values.join('\t'))
  }
  return rows.join('\n')
}
