import { formatCellValue } from './cellFormat'
import type { KanpicClipboard } from './clipboard'
import { fontFamilyName } from './clipboardHtml'

/**
 * 복사할 때 클립보드에 `text/html` 표도 함께 올린다. 평문만 올리면 엑셀,
 * 구글 시트, 워드 어디에 붙여넣어도 굵게·색·정렬이 전부 사라진 글자만
 * 남는다. 스프레드시트끼리 서식이 오가는 길은 이 표다.
 */
export function clipboardHtml(payload:KanpicClipboard){
  const byOffset=new Map(payload.cells.map(cell=>[`${cell.rowOffset}:${cell.columnOffset}`,cell]))
  const rows:string[]=[]
  for(let row=0;row<payload.rows;row+=1){
    const cells:string[]=[]
    for(let column=0;column<payload.columns;column+=1){
      const cell=byOffset.get(`${row}:${column}`)
      const style=cell?.style as Record<string,unknown>|undefined
      const text=cell?.value===undefined||cell.value===null?'':formatCellValue(cell.value,style)
      const attributes=[cssFor(style,cell?.value),numeric(cell?.value)].filter(Boolean).join(' ')
      cells.push(`<td${attributes?' '+attributes:''}>${escapeHtml(text)}</td>`)
    }
    rows.push(`<tr>${cells.join('')}</tr>`)
  }
  // xmlns:x 는 엑셀이 x:num 을 자기 이름 공간으로 알아보게 한다.
  return '<meta charset="utf-8"><table xmlns:x="urn:schemas-microsoft-com:office:excel">'
    +`<tbody>${rows.join('')}</tbody></table>`
}

/**
 * 보이던 글자와 함께 셀이 담고 있던 숫자를 붙여 둔다. 받는 쪽이 통화나
 * 백분율 표기를 알아보지 못해도 숫자는 숫자로 들어간다.
 */
function numeric(value:unknown){
  return typeof value==='number'&&Number.isFinite(value)?`x:num="${value}"`:''
}

function cssFor(style:Record<string,unknown>|undefined,value:unknown){
  const declarations:string[]=[]
  if(style?.bold===true)declarations.push('font-weight:700')
  if(style?.italic===true)declarations.push('font-style:italic')
  const decoration=[style?.underline===true?'underline':'',style?.strike===true?'line-through':''].filter(Boolean)
  if(decoration.length>0)declarations.push(`text-decoration:${decoration.join(' ')}`)
  const color=hex(style?.color)
  if(color)declarations.push(`color:${color}`)
  const background=hex(style?.background)
  if(background)declarations.push(`background-color:${background}`)
  const horizontal=style?.horizontal_align
  if(horizontal==='left'||horizontal==='center'||horizontal==='right')declarations.push(`text-align:${horizontal}`)
  else if(typeof value==='number')declarations.push('text-align:right')
  const vertical=style?.vertical_align
  if(vertical==='top'||vertical==='bottom')declarations.push(`vertical-align:${vertical}`)
  if(vertical==='middle')declarations.push('vertical-align:middle')
  if(style?.text_mode==='wrap'||style?.wrap===true)declarations.push('white-space:pre-wrap')
  const size=typeof style?.font_size==='number'?style.font_size:undefined
  if(size!==undefined&&size>=6&&size<=96)declarations.push(`font-size:${size}pt`)
  const family=fontFamilyName(typeof style?.font_family==='string'?style.font_family:undefined)
  if(family)declarations.push(`font-family:${family}`)
  // 엑셀은 붙여넣을 때 이 표시 형식을 읽어 셀에 그대로 적용한다. 세미콜론은
  // 선언을 끊어 버리므로 그런 형식은 넘기지 않는다(조건부 형식 구문).
  const format=typeof style?.number_format==='string'?style.number_format.trim():''
  if(format!==''&&format.length<=64&&!format.includes(';')&&!format.includes("'"))declarations.push(`mso-number-format:'${format}'`)
  return declarations.length>0?`style="${escapeHtml(declarations.join(';'))}"`:''
}

function hex(value:unknown){
  return typeof value==='string'&&/^#[0-9a-f]{3}([0-9a-f]{3})?$/i.test(value.trim())?value.trim():undefined
}

function escapeHtml(value:string){
  return value.replace(/[&<>"]/g,character=>character==='&'?'&amp;':character==='<'?'&lt;':character==='>'?'&gt;':'&quot;')
}
