import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ArrowDownAZ, ArrowUpAZ, BadgeCheck, BarChart3, Bot, ChevronsDownUp, ChevronsUpDown, Clipboard, ClipboardPaste, Copy, Eraser, EyeOff, Filter, History, Link2, MessageSquarePlus, Palette, PanelTop, Rows3, Sigma, Scissors, StickyNote, Table2, Trash2 } from 'lucide-react'
import { api, address, newIdempotencyKey } from '../lib/api'
import { ContextMenu, type MenuItem } from './ContextMenu'
import { FormulaAutocomplete, formulaHint, useFunctionCatalog } from './FormulaAutocomplete'
import { explainFormulaError, formulaErrorCode, type FormulaErrorExplanation } from '../lib/formulaError'
import { applySuggestion } from '../lib/formulaSuggest'
import { suggestColumnValues } from '../lib/valueSuggest'
import type { LayoutCommand } from './LayoutDialog'
import type { StructureCommand } from './StructureDialog'
import { dataRegion, populatedCell } from '../lib/dataRegion'
import { clampDimensionSize, parseTableRange, pointerRegion, resizeHandleAt, type GridGeometry, type ResizeTarget } from '../lib/gridGeometry'
import { spillRoom } from '../lib/textSpill'
import { clipboardText, KANPIC_CLIPBOARD_TYPE, materializeFill, MAX_GRID_COLUMNS, MAX_GRID_ROWS, MAX_PASTE_CELLS, type FillRange, type KanpicClipboard, type PasteMode, type PastedCell } from '../lib/clipboard'
import { drawConditionalIcon,iconGlyph,iconGutter } from '../lib/conditionalIcon'
import { hashesWhenTooNarrow } from '../lib/cellWidth'
import { parseClipboardHtml } from '../lib/clipboardHtml'
import { cycleReference } from '../lib/referenceCycle'
import { parseTypedCellValue } from '../lib/cellEntry'
import { clipboardHtml } from '../lib/clipboardHtmlOut'
import { collaborationClientId } from '../lib/client'
import { cellMerge,selectedMergedBounds,stripMergeStyle,type MergeRange } from '../lib/merge'
import { enqueue, flushOutbox } from '../lib/outbox'
import { forSheet } from '../lib/outboxScope'
import { axisIndexAtViewport,axisViewportPosition,createDimensionAxis,type DimensionAxis,rowHeaderWidth,presenceLabelTop} from '../lib/dimensionAxis'
import { formatCellValue,wrapText,type CellBorders,type BorderSide } from '../lib/cellFormat'
import { describeSparkline,drawSparkline,parseSparkline } from '../lib/sparkline'
import { collapsedIndexes,controlAt,controlIndexFor,groupsAt,innermostGroup,outlineSize,OUTLINE_STEP } from '../lib/outline'
import { cellLink, workbookRangeTarget } from '../lib/hyperlink'
import { checkboxState,optionForValue,optionLabel,ruleOptions,validateClientInputs,validateClientValue,validationForCell } from '../lib/validation'
import { presenceColor, useCollaborationStore } from '../state/collaboration'
import { cellKey, selectedBounds, useEditorStore } from '../state/editor'
import type { Cell, ConditionalFormat, ConditionalFormatCell, ConditionalFormatEvaluation, DataValidation, DimensionGroup, FilterView, MutationResult, SheetLayout, SheetTable } from '../types'
import { columnFiltered } from '../lib/columnFilter'
import { parseFilterRange } from '../lib/filter'

const ROW_HEADER_WIDTH=46
const COLUMN_HEADER_HEIGHT=27
const TOTAL_ROWS=MAX_GRID_ROWS
const TOTAL_COLUMNS=MAX_GRID_COLUMNS

export type GridShortcut=
  | {command:'fill-down'|'fill-right'|'select-all'|'select-row'|'select-column'|'move-first'|'move-last'}
  | {command:'select-data-region'|'clear-contents'|'auto-sum'|'insert-today'|'insert-now'|'copy'|'cut'|'paste'|'paste-values'}
  | {command:'paste-special';mode:PasteMode}
  | {command:'focus-grid'}
  | {command:'commit-draft'}
  | {command:'insert-text';text:string}
  | {command:'commit-text';text:string}
  | {command:'insert-function';name:string}
  | {command:'move-data-edge';direction:'up'|'down'|'left'|'right';extend:boolean}
  | {command:'move-page';direction:'up'|'down'|'left'|'right';extend:boolean}

/** Menu actions the editor page owns because they open dialogs or panels. */
export type GridMenuCommand=
  | {command:'sort-dialog'|'filter'|'comment'|'named-range'|'conditional-format'|'data-validation'|'chart'|'pivot'|'format-dialog'|'layout-dialog'|'structure-dialog'|'clear-format'|'find-replace'|'note'|'column-stats'|'subtotal'|'cleanup-duplicates'|'split-columns'}
  | {command:'cell-history';row:number;column:number}
  | {command:'agent';mode:'summarize'|'formula'|'explain'|'fix'|'clean'|'format'|'chart'|'agent';request:string}
  | {command:'merge';merge:boolean}
  | {command:'sort-column';column:number;direction:'asc'|'desc'}
  | {command:'column-filter';column:number;x:number;y:number}

const RESIZE_HANDLE=4,MIN_ROW_HEIGHT=16,MAX_ROW_HEIGHT=400,MIN_COLUMN_WIDTH=32,MAX_COLUMN_WIDTH=600,DEFAULT_ROW_HEIGHT=27,DEFAULT_COLUMN_WIDTH=108

function columnName(column:number){let value=column,result='';while(value){value--;result=String.fromCharCode(65+value%26)+result;value=Math.floor(value/26)}return result}
function parsedAddress(value:string){const match=/^([A-Z]+)([1-9]\d*)$/.exec(value.toUpperCase());if(!match)return;let column=0;for(const character of match[1])column=column*26+character.charCodeAt(0)-64;return{row:Number(match[2]),column}}
function formulaPreview(value:unknown){if(!Array.isArray(value))return value;const first=value[0];return Array.isArray(first)?first[0]:first}
// 표를 그리는 색. 선택 색이나 협업자 색과 겹치지 않아야 무엇이 무엇인지
// 알아볼 수 있다.
const TABLE_COLOR='#4c6ef5'
// 표마다 고른 색으로 그린다. 내보낼 때 엑셀의 표 서식으로 바뀌는 그 색이다 —
// 화면과 파일이 다른 색이면 어느 쪽이 맞는지 사람이 알 수 없다.
const TABLE_COLORS:Record<string,string>={blue:TABLE_COLOR,green:'#2f9e44',orange:'#e8590c',red:'#e03131'}
const tableColor=(theme?:string)=>TABLE_COLORS[(theme??'').toLowerCase()]??TABLE_COLOR

const DEFAULT_LAYOUT:SheetLayout={revision:1,frozen_rows:0,frozen_columns:0}
function indexesIn(axis:DimensionAxis,start:number,end:number){const result:number[]=[];if(end<start)return result;let index=axis.firstVisibleAtOrAfter(start);while(index<=end&&result.length<10000){result.push(index);const next=axis.nextVisible(index,1);if(next<=index)break;index=next}return result}
function paintCellBorders(context:CanvasRenderingContext2D,borders:CellBorders,x:number,y:number,width:number,height:number,zoom:number){
  const line=(side:'top'|'right'|'bottom'|'left',definition:BorderSide,offset=0)=>{const position=offset+.5;context.beginPath();if(side==='top'){context.moveTo(x,y+position);context.lineTo(x+width,y+position)}else if(side==='right'){context.moveTo(x+width-position,y);context.lineTo(x+width-position,y+height)}else if(side==='bottom'){context.moveTo(x,y+height-position);context.lineTo(x+width,y+height-position)}else{context.moveTo(x+position,y);context.lineTo(x+position,y+height)}context.strokeStyle=definition.color;context.lineWidth=Math.max(1,(definition.style==='medium'?2:definition.style==='thick'?3:1)*zoom);context.setLineDash(definition.style==='dashed'?[6*zoom,3*zoom]:definition.style==='dotted'?[2*zoom,2*zoom]:[]);context.stroke()}
  context.save();for(const side of ['top','right','bottom','left'] as const){const definition=borders[side];if(!definition)continue;if(definition.style==='double'){line(side,definition,1);line(side,definition,4)}else line(side,definition)}context.restore()
}

export function CanvasGrid({sheetId,workbookId,layout=DEFAULT_LAYOUT,version,onVersion,hiddenRows=[],validations=[],conditionalFormats=[],tables=[],filterView,formatBrush=false,onPaintFormat,showFormulas=false,showGridlines=true,readOnly=false,userLabels,onLayout,onStructure,onMenuCommand,onOpenRange,onResolveNumericRun,onPasteImage}:{sheetId:string;workbookId:string;layout?:SheetLayout;version:number;onVersion:(version:number)=>void;hiddenRows?:number[];validations?:DataValidation[];conditionalFormats?:ConditionalFormat[];tables?:SheetTable[];filterView?:FilterView;formatBrush?:boolean;onPaintFormat?:(range:{startRow:number;startColumn:number;endRow:number;endColumn:number})=>void;showFormulas?:boolean;showGridlines?:boolean;readOnly?:boolean;userLabels?:Record<string,string>;onLayout?:(command:LayoutCommand)=>Promise<void>;onStructure?:(command:StructureCommand)=>Promise<void>;onMenuCommand?:(command:GridMenuCommand)=>void;onOpenRange?:(sheetId:string,range:string)=>boolean;onResolveNumericRun?:(row:number,column:number)=>Promise<number|undefined>;onPasteImage?:(file:File)=>void}) {
  const viewport=useRef<HTMLDivElement>(null),editorInput=useRef<HTMLTextAreaElement>(null),composing=useRef(false),canvas=useRef<HTMLCanvasElement>(null),dragging=useRef(false),filling=useRef(false),fillPreviewRef=useRef<FillRange|undefined>(undefined),pasteAsValues=useRef(false)
  const headerDrag=useRef<{axis:'row'|'column';anchor:number}|null>(null),resizeDrag=useRef<{axis:'row'|'column';index:number;origin:number;start:number;count:number;size:number}|null>(null),internalClipboard=useRef<KanpicClipboard|undefined>(undefined)
  const moveDrag=useRef<{axis:'row'|'column';start:number;count:number;origin:number;destination:number;armed:boolean}|null>(null)
  const functionCatalog=useFunctionCatalog()
  const [caret,setCaret]=useState(0),[suggestion,setSuggestion]=useState(0)
  const [noteHover,setNoteHover]=useState<{row:number;column:number;text?:string;failure?:FormulaErrorExplanation;x:number;y:number}>()
  const [scroll,setScroll]=useState({left:0,top:0}),[size,setSize]=useState({width:900,height:500}),[fillPreview,setFillPreview]=useState<FillRange>(),[refreshToken,setRefreshToken]=useState(0),[conditionalCells,setConditionalCells]=useState<Map<string,ConditionalFormatCell>>(()=>new Map())
  const [movePreview,setMovePreview]=useState<{axis:'row'|'column';destination:number}>()
  const [resizePreview,setResizePreview]=useState<{axis:'row'|'column';index:number;size:number}>(),[menu,setMenu]=useState<{x:number;y:number;items:MenuItem[];label:string}>()
  const editor=useEditorStore()
  const {activeRow,activeColumn,anchorRow,anchorColumn,editing,draft,zoom,cells,select,setEditing,setDraft,replaceRange,putCells,putCell,setSaveState,recordOperation,reportEdit}=editor
  const selection=selectedMergedBounds(cells,selectedBounds(editor))
  const collaborators=useCollaborationStore(state=>state.users)
  const sendCursor=useCollaborationStore(state=>state.sendCursor),sendSelection=useCollaborationStore(state=>state.sendSelection)
  const hiddenRowsKey=hiddenRows.join(','),layoutKey=JSON.stringify(layout),conditionalRulesKey=conditionalFormats.map(rule=>`${rule.id}:${rule.revision}`).join(',')
  // A collapsed group folds its rows away without touching the ranges the user
  // hid by hand, so expanding it never reveals more than it hid.
  const collapsedRows=useMemo(()=>collapsedIndexes(layout.row_groups),[layoutKey])
  const collapsedColumns=useMemo(()=>collapsedIndexes(layout.column_groups),[layoutKey])
  const rowAxis=useMemo(()=>createDimensionAxis({total:TOTAL_ROWS,defaultSize:27,sizes:layout.row_heights,hiddenRanges:layout.hidden_rows,hiddenIndexes:[...hiddenRows,...collapsedRows],zoom}),[hiddenRowsKey,layoutKey,zoom])
  const columnAxis=useMemo(()=>createDimensionAxis({total:TOTAL_COLUMNS,defaultSize:108,sizes:layout.column_widths,hiddenRanges:layout.hidden_columns,hiddenIndexes:[...collapsedColumns],zoom}),[layoutKey,zoom])
  // The outline gutter sits between the sheet edge and the headers, one step
  // per level of nesting, and is absent entirely when nothing is grouped.
  const rowOutline=outlineSize(layout.row_groups),columnOutline=outlineSize(layout.column_groups)
  // 행 머리글은 그 자리에 보이는 가장 큰 행 번호를 담을 만큼 넓어야 한다.
  // 46px는 다섯 자리까지만 들어가서, 15만 행짜리 시트를 내려가면 번호의
  // 앞자리가 잘려 나갔다. 네 자리까지는 예전 폭 그대로다.
  const headerWidth=rowHeaderWidth(ROW_HEADER_WIDTH,rowAxis.indexAtOffset(scroll.top+size.height),zoom)+rowOutline,headerHeight=COLUMN_HEADER_HEIGHT+columnOutline
  const frozenRows=Math.min(layout.frozen_rows??0,TOTAL_ROWS),frozenColumns=Math.min(layout.frozen_columns??0,TOTAL_COLUMNS)
  const activeCell=cells.get(cellKey(activeRow,activeColumn))
  // The span the active filter covers, which is where the funnel buttons go.
  const filterColumns=useMemo(()=>{
    const range=filterView?parseFilterRange(filterView.range):undefined
    return range?{start:range.startColumn,end:range.endColumn,headerRow:range.startRow}:undefined
  },[filterView?.range])
  const activeValidation=validationForCell(validations,activeRow,activeColumn)
  // The checkbox under the cursor, which space, Enter and a click all flip.
  const activeCheckbox=activeValidation?checkboxState(activeValidation,activeCell?.value):undefined
  const activeText=activeCell?.formula || (activeCell?.value == null?'':String(activeCell.value))

  // Menu driven inserts move the active cell and seed the editor in one step,
  // so the pending draft survives the sync that normally mirrors the cell text.
  const pendingDraft=useRef<{row:number;column:number;text:string}|undefined>(undefined)
  useEffect(()=>{
    // 로컬에서 옮겨 심은 입력과, 다른 사람의 구조 변경 때문에 자리가 밀린
    // 입력은 같은 문제다. 둘 다 이 동기화를 넘겨야 살아남는다.
    const carried=useEditorStore.getState().carriedDraft
    const pending=pendingDraft.current??carried
    pendingDraft.current=undefined
    if(carried)useEditorStore.getState().carryDraft(undefined)
    if(pending&&pending.row===activeRow&&pending.column===activeColumn){setDraft(pending.text);return}
    setDraft(activeText)
  },[activeText,activeRow,activeColumn])
  useEffect(()=>{if(rowAxis.isHidden(activeRow)||columnAxis.isHidden(activeColumn))select(rowAxis.firstVisibleAtOrAfter(activeRow),columnAxis.firstVisibleAtOrAfter(activeColumn))},[rowAxis,columnAxis,activeRow,activeColumn,select])
  useEffect(()=>{dragging.current=false;filling.current=false;fillPreviewRef.current=undefined;setFillPreview(undefined)},[sheetId])
  useEffect(()=>{const rejected=(event:Event)=>{const detail=(event as CustomEvent<{message?:string}>).detail;setSaveState('error');setRefreshToken(value=>value+1);alert(detail?.message??'서버가 변경을 거부했습니다. 최신 값을 다시 불러옵니다.')};window.addEventListener('kanpic:outbox-rejected',rejected);return()=>window.removeEventListener('kanpic:outbox-rejected',rejected)},[setSaveState])
  // 저장하지 못한 변경을 사람이 버리면 화면에 남은 낙관적 값도 서버 값으로 되돌린다.
  useEffect(()=>{const discarded=()=>setRefreshToken(value=>value+1);window.addEventListener('kanpic:outbox-discarded',discarded);return()=>window.removeEventListener('kanpic:outbox-discarded',discarded)},[])
  useEffect(()=>{if(!viewport.current)return;const observer=new ResizeObserver(([entry])=>setSize({width:Math.floor(entry.contentRect.width),height:Math.floor(entry.contentRect.height)}));observer.observe(viewport.current);return()=>observer.disconnect()},[])
  useEffect(()=>{const element=viewport.current;if(!element)return;let left=element.scrollLeft,top=element.scrollTop;const bodyWidth=Math.max(1,element.clientWidth-headerWidth),bodyHeight=Math.max(1,element.clientHeight-headerHeight),frozenWidth=columnAxis.offsetOf(frozenColumns+1),frozenHeight=rowAxis.offsetOf(frozenRows+1);if(activeColumn>frozenColumns){const start=columnAxis.offsetOf(activeColumn),end=start+columnAxis.sizeOf(activeColumn),visibleStart=left+frozenWidth,visibleEnd=left+bodyWidth;if(start<visibleStart)left=Math.max(0,start-frozenWidth);else if(end>visibleEnd)left=Math.max(0,end-bodyWidth)}if(activeRow>frozenRows){const start=rowAxis.offsetOf(activeRow),end=start+rowAxis.sizeOf(activeRow),visibleStart=top+frozenHeight,visibleEnd=top+bodyHeight;if(start<visibleStart)top=Math.max(0,start-frozenHeight);else if(end>visibleEnd)top=Math.max(0,end-bodyHeight)}if(left!==element.scrollLeft||top!==element.scrollTop){element.scrollLeft=left;element.scrollTop=top;setScroll({left,top})}},[activeRow,activeColumn,sheetId,rowAxis,columnAxis,frozenRows,frozenColumns])

  const visibleRange=useMemo(()=>{const frozenHeight=rowAxis.offsetOf(frozenRows+1),frozenWidth=columnAxis.offsetOf(frozenColumns+1),startRow=rowAxis.firstVisibleAtOrAfter(Math.max(frozenRows+1,rowAxis.indexAtOffset(scroll.top+frozenHeight))),startColumn=columnAxis.firstVisibleAtOrAfter(Math.max(frozenColumns+1,columnAxis.indexAtOffset(scroll.left+frozenWidth))),endRow=rowAxis.lastVisibleAtOrBefore(rowAxis.indexAtOffset(scroll.top+Math.max(frozenHeight,size.height-headerHeight)+100*zoom)),endColumn=columnAxis.lastVisibleAtOrBefore(columnAxis.indexAtOffset(scroll.left+Math.max(frozenWidth,size.width-headerWidth)+250*zoom));return{startRow,startColumn,endRow:Math.max(startRow,endRow),endColumn:Math.max(startColumn,endColumn)}},[scroll,size,rowAxis,columnAxis,frozenRows,frozenColumns,zoom])
  useEffect(()=>{const controller=new AbortController(),ranges=[visibleRange];if(frozenRows>0)ranges.push({startRow:rowAxis.firstVisibleAtOrAfter(1),endRow:rowAxis.lastVisibleAtOrBefore(frozenRows),startColumn:visibleRange.startColumn,endColumn:visibleRange.endColumn});if(frozenColumns>0)ranges.push({startRow:visibleRange.startRow,endRow:visibleRange.endRow,startColumn:columnAxis.firstVisibleAtOrAfter(1),endColumn:columnAxis.lastVisibleAtOrBefore(frozenColumns)});if(frozenRows>0&&frozenColumns>0)ranges.push({startRow:rowAxis.firstVisibleAtOrAfter(1),endRow:rowAxis.lastVisibleAtOrBefore(frozenRows),startColumn:columnAxis.firstVisibleAtOrAfter(1),endColumn:columnAxis.lastVisibleAtOrBefore(frozenColumns)});for(const selected of ranges){if(selected.endRow<selected.startRow||selected.endColumn<selected.startColumn)continue;const range=`${address(selected.startRow,selected.startColumn)}:${address(selected.endRow,selected.endColumn)}`;api<{items:Cell[]}>(`/api/v1/sheets/${sheetId}/ranges/${range}`,{signal:controller.signal}).then(result=>replaceRange(result.items,selected.startRow,selected.startColumn,selected.endRow,selected.endColumn)).catch(()=>{})}return()=>controller.abort()},[sheetId,version,refreshToken,visibleRange.startRow,visibleRange.startColumn,visibleRange.endRow,visibleRange.endColumn,frozenRows,frozenColumns,rowAxis,columnAxis,replaceRange])
  useEffect(()=>{
    if(conditionalFormats.length===0){setConditionalCells(new Map());return}
    const controller=new AbortController(),ranges=[visibleRange]
    if(frozenRows>0)ranges.push({startRow:rowAxis.firstVisibleAtOrAfter(1),endRow:rowAxis.lastVisibleAtOrBefore(frozenRows),startColumn:visibleRange.startColumn,endColumn:visibleRange.endColumn})
    if(frozenColumns>0)ranges.push({startRow:visibleRange.startRow,endRow:visibleRange.endRow,startColumn:columnAxis.firstVisibleAtOrAfter(1),endColumn:columnAxis.lastVisibleAtOrBefore(frozenColumns)})
    if(frozenRows>0&&frozenColumns>0)ranges.push({startRow:rowAxis.firstVisibleAtOrAfter(1),endRow:rowAxis.lastVisibleAtOrBefore(frozenRows),startColumn:columnAxis.firstVisibleAtOrAfter(1),endColumn:columnAxis.lastVisibleAtOrBefore(frozenColumns)})
    const requests=ranges.filter(selected=>selected.endRow>=selected.startRow&&selected.endColumn>=selected.startColumn).map(selected=>{const range=`${address(selected.startRow,selected.startColumn)}:${address(selected.endRow,selected.endColumn)}`;return api<ConditionalFormatEvaluation>(`/api/v1/sheets/${sheetId}/conditional-formats:evaluate?range=${encodeURIComponent(range)}`,{signal:controller.signal})})
    Promise.all(requests).then(results=>{const next=new Map<string,ConditionalFormatCell>();for(const result of results)for(const item of result.items)next.set(cellKey(item.row,item.column),item);setConditionalCells(next)}).catch(()=>{})
    return()=>controller.abort()
  },[sheetId,version,refreshToken,conditionalRulesKey,visibleRange.startRow,visibleRange.startColumn,visibleRange.endRow,visibleRange.endColumn,frozenRows,frozenColumns,rowAxis,columnAxis])

  useEffect(()=>{
    const element=canvas.current
    if(!element)return
    const ratio=window.devicePixelRatio||1
    element.width=size.width*ratio;element.height=size.height*ratio;element.style.width=`${size.width}px`;element.style.height=`${size.height}px`
    const context=element.getContext('2d');if(!context)return
    context.scale(ratio,ratio);context.fillStyle='#fff';context.fillRect(0,0,size.width,size.height);context.font=`${12*zoom}px Inter, Pretendard, sans-serif`;context.textBaseline='middle'
    const rowPosition=(row:number)=>headerHeight+axisViewportPosition(rowAxis,row,scroll.top,frozenRows),columnPosition=(column:number)=>headerWidth+axisViewportPosition(columnAxis,column,scroll.left,frozenColumns)
    const rowVisible=(row:number)=>!rowAxis.isHidden(row)&&(row<=frozenRows||row>=visibleRange.startRow&&row<=visibleRange.endRow),columnVisible=(column:number)=>!columnAxis.isHidden(column)&&(column<=frozenColumns||column>=visibleRange.startColumn&&column<=visibleRange.endColumn)
    const geometry=(startRow:number,startColumn:number,endRow:number,endColumn:number)=>{if(rowAxis.countVisible(startRow,endRow)===0||columnAxis.countVisible(startColumn,endColumn)===0)return;const firstRow=rowAxis.firstVisibleAtOrAfter(startRow),lastRow=rowAxis.lastVisibleAtOrBefore(endRow),firstColumn=columnAxis.firstVisibleAtOrAfter(startColumn),lastColumn=columnAxis.lastVisibleAtOrBefore(endColumn),x=columnPosition(firstColumn),y=rowPosition(firstRow);return{x,y,width:columnPosition(lastColumn)+columnAxis.sizeOf(lastColumn)-x,height:rowPosition(lastRow)+rowAxis.sizeOf(lastRow)-y}}
    const mainRows=indexesIn(rowAxis,visibleRange.startRow,visibleRange.endRow),mainColumns=indexesIn(columnAxis,visibleRange.startColumn,visibleRange.endColumn),frozenRowIndexes=indexesIn(rowAxis,1,frozenRows),frozenColumnIndexes=indexesIn(columnAxis,1,frozenColumns)
    const rows=[...mainRows,...frozenRowIndexes],columns=[...mainColumns,...frozenColumnIndexes]
    context.fillStyle='#f7f9fb';context.fillRect(0,0,size.width,headerHeight);context.fillRect(0,0,headerWidth,size.height)
    context.strokeStyle='#e4e8ec';context.lineWidth=1
    const wholeColumns=selection.startRow<=1&&selection.endRow>=TOTAL_ROWS,wholeRows=selection.startColumn<=1&&selection.endColumn>=TOTAL_COLUMNS
    for(const column of columns){const x=columnPosition(column),width=columnAxis.sizeOf(column);if(x+width<headerWidth||x>size.width)continue
      const selected=column>=selection.startColumn&&column<=selection.endColumn
      context.fillStyle=selected&&wholeColumns?'#c7e3dd':selected?'#e6f2ef':'#f7f9fb';context.fillRect(x,0,width,headerHeight)
      context.beginPath();context.moveTo(Math.round(x)+.5,0);context.lineTo(Math.round(x)+.5,showGridlines?size.height:headerHeight);context.stroke()
      if(selected){context.fillStyle='#0f766e';context.fillRect(x,headerHeight-2,width,2)}
      context.fillStyle=selected?'#0b5c55':'#52606d';context.font=`${selected?'600 ':''}${12*zoom}px Inter, Pretendard, sans-serif`;context.textAlign='center';context.fillText(columnName(column),x+width/2,headerHeight/2)
      // A filtered range puts a funnel on every column it covers, filled in on
      // the columns that actually restrict something.
      if(filterColumns&&column>=filterColumns.start&&column<=filterColumns.end&&width>34){
        const glyphX=x+width-14,glyphY=headerHeight/2
        const restricted=columnFiltered(filterView,column)
        context.fillStyle=restricted?'#0f766e':'#93a2aa'
        context.beginPath()
        context.moveTo(glyphX-5,glyphY-4);context.lineTo(glyphX+5,glyphY-4)
        context.lineTo(glyphX+1,glyphY);context.lineTo(glyphX+1,glyphY+4)
        context.lineTo(glyphX-1,glyphY+3);context.lineTo(glyphX-1,glyphY)
        context.closePath();context.fill()
      }}
    context.font=`${12*zoom}px Inter, Pretendard, sans-serif`
    for(const row of rows){const y=rowPosition(row),height=rowAxis.sizeOf(row);if(y+height<headerHeight||y>size.height)continue
      const selected=row>=selection.startRow&&row<=selection.endRow
      context.fillStyle=selected&&wholeRows?'#c7e3dd':selected?'#e6f2ef':'#f7f9fb';context.fillRect(0,y,headerWidth,height)
      context.beginPath();context.moveTo(0,Math.round(y)+.5);context.lineTo(showGridlines?size.width:headerWidth,Math.round(y)+.5);context.stroke()
      if(selected){context.fillStyle='#0f766e';context.fillRect(headerWidth-2,y,2,height)}
      context.fillStyle=selected?'#0b5c55':'#73808c';context.font=`${selected?'600 ':''}${12*zoom}px Inter, Pretendard, sans-serif`;context.textAlign='right';context.fillText(String(row),headerWidth-8,y+height/2)}
    context.font=`${12*zoom}px Inter, Pretendard, sans-serif`
    // The outline gutter: a bracket for every group and a box to fold it.
    const paintOutline=(axis:'row'|'column')=>{
      const groups=axis==='row'?layout.row_groups:layout.column_groups
      if(!groups||groups.length===0)return
      const indexes=axis==='row'?rows:columns
      const depth=Math.max(...groups.map(group=>group.depth))
      context.save()
      context.fillStyle='#f2f5f6'
      if(axis==='row')context.fillRect(0,headerHeight,rowOutline,size.height-headerHeight)
      else context.fillRect(headerWidth,0,size.width-headerWidth,columnOutline)
      context.strokeStyle='#8fa3ab';context.lineWidth=1
      for(const index of indexes){
        const start=axis==='row'?rowPosition(index):columnPosition(index)
        const span=axis==='row'?rowAxis.sizeOf(index):columnAxis.sizeOf(index)
        const middle=Math.round(start+span/2)+.5
        for(let level=0;level<=depth;level++){
          const line=Math.round(level*OUTLINE_STEP+OUTLINE_STEP/2)+.5
          const control=controlAt(groups,index,level)
          const covered=groupsAt(groups,index).some(group=>group.depth===level)
          if(!covered&&!control)continue
          context.beginPath()
          if(axis==='row'){
            if(!control){context.moveTo(line,start);context.lineTo(line,start+span)}
            else{context.moveTo(line,start);context.lineTo(line,middle)}
          }else{
            if(!control){context.moveTo(start,line);context.lineTo(start+span,line)}
            else{context.moveTo(start,line);context.lineTo(middle,line)}
          }
          context.stroke()
          if(!control)continue
          const boxX=axis==='row'?line-5:middle-5,boxY=axis==='row'?middle-5:line-5
          context.fillStyle='#fff';context.fillRect(boxX,boxY,10,10)
          context.strokeRect(boxX+.5,boxY+.5,9,9)
          context.beginPath();context.moveTo(boxX+2.5,boxY+5);context.lineTo(boxX+7.5,boxY+5)
          if(control.collapsed){context.moveTo(boxX+5,boxY+2.5);context.lineTo(boxX+5,boxY+7.5)}
          context.stroke()
          context.fillStyle='#f2f5f6'
        }
      }
      context.restore()
    }
    paintOutline('row');paintOutline('column')
    context.save();context.beginPath();context.rect(headerWidth,headerHeight,size.width-headerWidth,size.height-headerHeight);context.clip()
    const mergedRanges=new Map<string,{range:MergeRange;representative:Cell}>()
    const drawCell=(cell:Cell,x:number,y:number,width:number,height:number)=>{
      const conditional=conditionalCells.get(cellKey(cell.row,cell.column)),style={...(cell.style??{}),...(conditional?.style??{})},validation=validationForCell(validations,cell.row,cell.column),validationOption=validation?.rule_type==='list'?optionForValue(validation,cell.value):undefined
      context.fillStyle=typeof style.background==='string'?style.background:'#fff';context.fillRect(x+1,y+1,width-2,height-2)
      if(conditional?.data_bar){context.save();context.globalAlpha=.3;context.fillStyle=conditional.data_bar.color;context.fillRect(x+3,y+4,Math.max(0,(width-6)*conditional.data_bar.ratio),Math.max(0,height-8));context.restore()}
      // 아이콘은 왼쪽 자리를 차지하고, 값은 그만큼 오른쪽으로 물러난다.
      const glyph=conditional?.icon?iconGlyph(conditional.icon):null
      const iconSize=glyph?Math.max(0,Math.min(13*zoom,height-6,width-10)):0
      const iconGap=glyph&&iconSize>4?iconGutter(iconSize):0
      if(glyph&&iconGap)drawConditionalIcon(context,glyph,x+4,y+(height-iconSize)/2,iconSize)
      if(validation?.display_style==='chip'&&validationOption?.color){context.fillStyle=validationOption.color;context.beginPath();context.roundRect(x+4,y+4,width-8,height-8,6);context.fill()}
      // A checkbox cell is drawn as a box, which is the only way the state is
      // readable at a glance and clickable without opening an editor.
      const checkbox=validation&&!showFormulas?checkboxState(validation,cell.value):undefined
      if(checkbox){
        const size=Math.min(14*zoom,height-6,width-6)
        const boxX=Math.round(x+(width-size)/2),boxY=Math.round(y+(height-size)/2)
        context.fillStyle=checkbox.checked?'#0f766e':'#ffffff'
        context.strokeStyle=checkbox.checked?'#0f766e':'#98a6ad'
        context.lineWidth=1
        context.beginPath();context.roundRect(boxX+.5,boxY+.5,size-1,size-1,3);context.fill();context.stroke()
        if(checkbox.checked){
          context.strokeStyle='#ffffff';context.lineWidth=Math.max(1.5,size/8)
          context.beginPath()
          context.moveTo(boxX+size*0.24,boxY+size*0.52)
          context.lineTo(boxX+size*0.43,boxY+size*0.72)
          context.lineTo(boxX+size*0.77,boxY+size*0.29)
          context.stroke()
        }
        if(cell.note){context.fillStyle='#e0a428';context.beginPath();context.moveTo(x+width-8,y+1);context.lineTo(x+width-1,y+1);context.lineTo(x+width-1,y+8);context.closePath();context.fill()}
        if(style.borders&&typeof style.borders==='object')paintCellBorders(context,style.borders as CellBorders,x,y,width,height,zoom)
        return
      }
      // A SPARKLINE result is a chart rather than text, so it is painted into
      // the cell and nothing else is written there.
      const sparkline=showFormulas?undefined:parseSparkline(cell.value)
      if(sparkline){
        drawSparkline(context,sparkline,x,y,width,height,zoom)
        if(style.borders&&typeof style.borders==='object')paintCellBorders(context,style.borders as CellBorders,x,y,width,height,zoom)
        return
      }
      const formulaError=typeof cell.value==='string'&&cell.value.startsWith('#')
      const fontSize=typeof style.font_size==='number'?style.font_size:12,fontFamily=typeof style.font_family==='string'?JSON.stringify(style.font_family):'Inter, Pretendard, sans-serif'
      // A cell that carries a link is drawn the way a link is drawn everywhere
      // else, so it reads as clickable without hovering it first.
      const link=showFormulas?undefined:cellLink(cell)
      context.fillStyle=formulaError?'#c2413b':link?'#1155cc':typeof style.color==='string'?style.color:'#1c2733';context.font=`${style.italic===true?'italic ':''}${style.bold||formulaError?'600':'400'} ${fontSize*zoom}px ${fontFamily}`
      const alignment=validation?.display_style==='chip'?'left':style.horizontal_align==='left'||style.horizontal_align==='center'||style.horizontal_align==='right'?style.horizontal_align:typeof cell.value==='number'?'right':'left'
      context.textAlign=alignment
      const text=showFormulas&&cell.formula?cell.formula:validationOption?optionLabel(validationOption):formatCellValue(cell.value,style),textX=alignment==='right'?x+width-7:alignment==='center'?x+iconGap+(width-iconGap)/2:x+iconGap+(validation?.display_style==='chip'?10:7)
      const vertical=style.vertical_align==='top'||style.vertical_align==='bottom'||style.vertical_align==='middle'?style.vertical_align:'middle'
      const textY=vertical==='top'?y+Math.max(4,fontSize*zoom/2+3):vertical==='bottom'?y+height-Math.max(4,fontSize*zoom/2+3):y+height/2
      // A value with a line break in it is wrapped whether or not wrapping was
      // asked for; drawing it on one line would hide the break entirely.
      const rotation=typeof style.text_rotation==='number'?style.text_rotation:0,maxTextWidth=Math.max(0,width-12-iconGap),textMode=style.text_mode==='wrap'||style.wrap===true||text.includes('\n')?'wrap':style.text_mode==='clip'?'clip':'overflow'
      // Overflowing text spills across empty neighbours and is cut off at the
      // first cell that holds something, instead of being condensed to fit.
      const room=textMode==='overflow'&&rotation===0
        ?spillRoom({row:cell.row,column:cell.column,alignment,maxColumn:TOTAL_COLUMNS,sizeOf:column=>columnAxis.sizeOf(column),populated:(candidateRow,candidateColumn)=>populatedCell(cells,candidateRow,candidateColumn)})
        :{left:0,right:0}
      const drawWidth=textMode==='overflow'&&rotation===0?maxTextWidth+room.left+room.right:maxTextWidth
      // 자리에 안 들어가는 숫자는 잘라 보여 주면 안 된다. 1,234,567 의 앞
      // 부분만 보이면 1,234 로 읽힌다. 스프레드시트가 `####` 를 쓰는 이유다.
      const painted=rotation===0&&textMode!=='wrap'&&typeof cell.value==='number'&&!showFormulas
        ?hashesWhenTooNarrow(text,drawWidth,value=>context.measureText(value).width)
        :text
      context.save();context.beginPath();context.rect(x-room.left+1,y+1,Math.max(0,width-2+room.left+room.right),Math.max(0,height-2));context.clip()
      if(textMode==='wrap'&&rotation===0){const lines=wrapText(text,maxTextWidth,value=>context.measureText(value).width),lineHeight=Math.max(fontSize*zoom*1.25,12*zoom),visibleLines=Math.max(1,Math.floor((height-6)/lineHeight)),shown=lines.slice(0,visibleLines),blockHeight=shown.length*lineHeight,startY=vertical==='top'?y+3+lineHeight/2:vertical==='bottom'?y+height-3-blockHeight+lineHeight/2:y+(height-blockHeight)/2+lineHeight/2;shown.forEach((line,index)=>context.fillText(line,textX,startY+index*lineHeight,maxTextWidth))}else{context.translate(textX,textY);if(rotation)context.rotate(rotation*Math.PI/180);context.fillText(painted,0,0);if(text&&(style.underline===true||style.strike===true)){const measured=Math.min(context.measureText(painted).width,drawWidth),start=alignment==='right'?-measured:alignment==='center'?-measured/2:0;context.strokeStyle=context.fillStyle;context.lineWidth=Math.max(1,zoom);if(style.underline===true){context.beginPath();context.moveTo(start,fontSize*zoom*.48);context.lineTo(start+measured,fontSize*zoom*.48);context.stroke()}if(style.strike===true){context.beginPath();context.moveTo(start,0);context.lineTo(start+measured,0);context.stroke()}}}
      if(link&&text!==''){
        const measured=Math.min(context.measureText(painted).width,drawWidth)
        const underlineX=alignment==='right'?textX-measured:alignment==='center'?textX-measured/2:textX
        context.fillRect(underlineX,textY+fontSize*zoom*.55,measured,Math.max(1,zoom))
      }
      context.restore();if(style.borders&&typeof style.borders==='object')paintCellBorders(context,style.borders as CellBorders,x,y,width,height,zoom)
      if(validation?.rule_type==='list'&&validation.show_dropdown&&validation.display_style!=='plain'){context.fillStyle='#52606d';context.beginPath();context.moveTo(x+width-13,y+height/2-2);context.lineTo(x+width-7,y+height/2-2);context.lineTo(x+width-10,y+height/2+2);context.closePath();context.fill()}
      if(validation){const checked=validateClientValue(validation,cell.value);if(!checked.valid&&!checked.deferred){context.fillStyle='#dc2626';context.beginPath();context.moveTo(x+width-9,y+1);context.lineTo(x+width-1,y+1);context.lineTo(x+width-1,y+9);context.closePath();context.fill()}}
      if(cell.note){
        // The corner marker is the only sign a note is attached; hovering the
        // cell shows the text itself.
        context.fillStyle='#e0a428'
        context.beginPath();context.moveTo(x+width-8,y+1);context.lineTo(x+width-1,y+1);context.lineTo(x+width-1,y+8);context.closePath();context.fill()
      }
      if(cell.spill_source){context.save();context.setLineDash([2,2]);context.strokeStyle='#38a3a5';context.lineWidth=1;context.strokeRect(Math.round(x)+2,Math.round(y)+2,Math.round(width)-4,Math.round(height)-4);context.restore()}
    }
    const renderCells=new Map(cells);conditionalCells.forEach(item=>{const key=cellKey(item.row,item.column);if(!renderCells.has(key))renderCells.set(key,{sheet_id:sheetId,row:item.row,column:item.column,updated_at:''})})
    const visibleCells=Array.from(renderCells.values()).filter(cell=>rowVisible(cell.row)&&columnVisible(cell.column)).sort((a,b)=>Number(a.row<=frozenRows)+Number(a.column<=frozenColumns)-Number(b.row<=frozenRows)-Number(b.column<=frozenColumns))
    visibleCells.forEach(cell=>{
      const merged=cellMerge(cell)
      if(merged){const key=`${merged.startRow}:${merged.startColumn}:${merged.endRow}:${merged.endColumn}`,current=mergedRanges.get(key);if(!current||cell.row===merged.startRow&&cell.column===merged.startColumn)mergedRanges.set(key,{range:merged,representative:cell});return}
      const x=columnPosition(cell.column),y=rowPosition(cell.row)
      drawCell(cell,x,y,columnAxis.sizeOf(cell.column),rowAxis.sizeOf(cell.row))
    })
    mergedRanges.forEach(({range,representative})=>{const box=geometry(range.startRow,range.startColumn,range.endRow,range.endColumn);if(!box)return;const anchor=cells.get(cellKey(range.startRow,range.startColumn)),display=anchor??{...representative,value:undefined,formula:undefined};drawCell(display,box.x,box.y,box.width,box.height);context.strokeStyle='#e4e8ec';context.lineWidth=1;context.strokeRect(Math.round(box.x)+.5,Math.round(box.y)+.5,Math.round(box.width),Math.round(box.height))})
    // 표는 격자 위에 테두리와 이름으로 그린다. 칸에 색을 칠해 두면 표가
    // 행을 따라 움직였을 때 칠이 제자리에 남아, 표가 아닌 곳이 표처럼
    // 보인다. 그려 주는 편은 언제나 지금의 자리를 따른다.
    tables.forEach(table=>{
      if(table.sheet_id!==sheetId)return
      const bounds=parseTableRange(table.range);if(!bounds)return
      const box=geometry(bounds.startRow,bounds.startColumn,bounds.endRow,bounds.endColumn);if(!box)return
      const colour=tableColor(table.theme)
      context.save()
      if(table.header_row){
        const header=geometry(bounds.startRow,bounds.startColumn,bounds.startRow,bounds.endColumn)
        if(header){context.globalAlpha=.12;context.fillStyle=colour;context.fillRect(header.x,header.y,header.width,header.height);context.globalAlpha=1}
      }
      // 합계 줄도 옅게 칠하고 그 위에 줄을 긋는다. 켰는데 눈에 띄는 변화가
      // 없으면 사람은 켜지지 않은 줄 안다.
      if(table.totals_row){
        const totals=geometry(bounds.endRow,bounds.startColumn,bounds.endRow,bounds.endColumn)
        if(totals){
          context.globalAlpha=.12;context.fillStyle=colour;context.fillRect(totals.x,totals.y,totals.width,totals.height);context.globalAlpha=1
          context.strokeStyle=colour;context.lineWidth=1
          context.beginPath();context.moveTo(totals.x,Math.round(totals.y)+.5);context.lineTo(totals.x+totals.width,Math.round(totals.y)+.5);context.stroke()
        }
      }
      context.strokeStyle=colour;context.lineWidth=2
      context.strokeRect(Math.round(box.x)+1,Math.round(box.y)+1,Math.round(box.width)-2,Math.round(box.height)-2)
      // 이름은 표 위쪽 바깥에 붙인다. 자리가 없으면 그리지 않는다 — 첫 줄을
      // 가리면 머리글을 못 읽는다.
      const labelHeight=14*zoom
      if(box.y-labelHeight>=headerHeight){
        context.font=`${Math.round(11*zoom)}px system-ui`
        const width=context.measureText(table.name).width+10*zoom
        context.fillStyle=colour;context.fillRect(box.x,box.y-labelHeight,width,labelHeight)
        context.fillStyle='#ffffff';context.textAlign='left';context.textBaseline='middle'
        context.fillText(table.name,box.x+5*zoom,box.y-labelHeight/2)
      }
      context.restore()
    })
    Object.values(collaborators).forEach(user=>{
      if(user.client_id===collaborationClientId()||user.selection?.sheet_id!==sheetId)return
      const remote=user.selection
      const startRow=Math.min(remote.start.row,remote.end.row),endRow=Math.max(remote.start.row,remote.end.row)
      const startColumn=Math.min(remote.start.column,remote.end.column),endColumn=Math.max(remote.start.column,remote.end.column)
      const box=geometry(startRow,startColumn,endRow,endColumn);if(!box)return
      const color=presenceColor(user.client_id)
      context.save();context.globalAlpha=.1;context.fillStyle=color;context.fillRect(box.x,box.y,box.width,box.height);context.restore();context.strokeStyle=color;context.lineWidth=2;context.strokeRect(Math.round(box.x)+1,Math.round(box.y)+1,Math.round(box.width)-2,Math.round(box.height)-2)
    })
    Object.values(collaborators).forEach(user=>{
      if(user.client_id===collaborationClientId()||user.cursor?.sheet_id!==sheetId)return
      const cursor=user.cursor;if(rowAxis.isHidden(cursor.row)||columnAxis.isHidden(cursor.column))return
      const x=columnPosition(cursor.column),y=rowPosition(cursor.row),cellWidth=columnAxis.sizeOf(cursor.column),cellHeight=rowAxis.sizeOf(cursor.row)
      if(x+cellWidth<headerWidth||y+cellHeight<headerHeight||x>size.width||y>size.height)return
      const color=presenceColor(user.client_id);context.strokeStyle=color;context.lineWidth=2;context.strokeRect(Math.round(x)+2,Math.round(y)+2,Math.round(cellWidth)-4,Math.round(cellHeight)-4);context.fillStyle=color;context.font=`600 ${9*zoom}px Inter, Pretendard, sans-serif`;context.textAlign='left'
      const label=userLabels?.[user.actor_id?.toLowerCase()??'']||user.actor_id||'사용자',labelWidth=Math.min(cellWidth,context.measureText(label).width+10)
      // 이름표는 셀 위에 붙인다. 맨 윗줄이라 위에 자리가 없으면 아래로
      // 내린다. 머리글 아래로 밀어 넣으면 그 셀의 내용을 가려서, 누가 어디에
      // 있는지는 알려 주고 거기에 무엇이 있는지는 감추게 된다.
      const labelHeight=14*zoom,labelTop=presenceLabelTop(y,cellHeight,labelHeight,headerHeight)
      context.fillRect(x+1,labelTop,labelWidth,labelHeight);context.fillStyle='#fff';context.fillText(label,x+5,labelTop+labelHeight/2,labelWidth-8)
    })
    if(fillPreview){const box=geometry(fillPreview.startRow,fillPreview.startColumn,fillPreview.endRow,fillPreview.endColumn);if(box){context.fillStyle='rgba(15,118,110,.045)';context.fillRect(box.x,box.y,box.width,box.height);context.save();context.setLineDash([5,3]);context.strokeStyle='#0f766e';context.lineWidth=1;context.strokeRect(Math.round(box.x)+1,Math.round(box.y)+1,Math.round(box.width)-2,Math.round(box.height)-2);context.restore()}}
    const selectionBox=geometry(selection.startRow,selection.startColumn,selection.endRow,selection.endColumn)
    if(selectionBox){context.fillStyle='rgba(15,118,110,.08)';context.fillRect(selectionBox.x,selectionBox.y,selectionBox.width,selectionBox.height);context.strokeStyle='#0f766e';context.lineWidth=2;context.strokeRect(Math.round(selectionBox.x)+1,Math.round(selectionBox.y)+1,Math.round(selectionBox.width)-2,Math.round(selectionBox.height)-2)}
    const activeMerge=cellMerge(activeCell),activeStartRow=activeMerge?.startRow??activeRow,activeStartColumn=activeMerge?.startColumn??activeColumn,activeEndRow=activeMerge?.endRow??activeRow,activeEndColumn=activeMerge?.endColumn??activeColumn
    const activeBox=geometry(activeStartRow,activeStartColumn,activeEndRow,activeEndColumn)
    if(activeBox){context.strokeStyle='#0f766e';context.lineWidth=2;context.strokeRect(Math.round(activeBox.x)+1,Math.round(activeBox.y)+1,Math.round(activeBox.width)-2,Math.round(activeBox.height)-2)}if(selectionBox){context.fillStyle='#0f766e';context.fillRect(selectionBox.x+selectionBox.width-4,selectionBox.y+selectionBox.height-4,6,6)}context.restore()
    const frozenHeight=rowAxis.offsetOf(frozenRows+1),frozenWidth=columnAxis.offsetOf(frozenColumns+1);context.strokeStyle='#98a9ad';context.lineWidth=2;if(frozenRows>0){context.beginPath();context.moveTo(0,headerHeight+frozenHeight+.5);context.lineTo(size.width,headerHeight+frozenHeight+.5);context.stroke()}if(frozenColumns>0){context.beginPath();context.moveTo(headerWidth+frozenWidth+.5,0);context.lineTo(headerWidth+frozenWidth+.5,size.height);context.stroke()}
    context.fillStyle='#edf7f5';context.fillRect(0,0,headerWidth,headerHeight);context.strokeStyle='#d9dfe5';context.strokeRect(.5,.5,headerWidth-.5,headerHeight-.5)
    if(resizePreview){
      context.save();context.strokeStyle='#0f766e';context.lineWidth=2;context.setLineDash([6,4])
      if(resizePreview.axis==='column'){const x=columnPosition(resizePreview.index)+resizePreview.size*zoom;context.beginPath();context.moveTo(Math.round(x)+.5,0);context.lineTo(Math.round(x)+.5,size.height)}
      else{const y=rowPosition(resizePreview.index)+resizePreview.size*zoom;context.beginPath();context.moveTo(0,Math.round(y)+.5);context.lineTo(size.width,Math.round(y)+.5)}
      context.stroke();context.restore()
      const label=`${Math.round(resizePreview.size)}px`
      context.font=`600 ${10*zoom}px Inter, Pretendard, sans-serif`;context.textAlign='left'
      const labelWidth=context.measureText(label).width+12
      const labelX=resizePreview.axis==='column'?Math.min(size.width-labelWidth-4,columnPosition(resizePreview.index)+resizePreview.size*zoom+6):headerWidth+6
      const labelY=resizePreview.axis==='column'?headerHeight+6:Math.min(size.height-22,rowPosition(resizePreview.index)+resizePreview.size*zoom+6)
      context.fillStyle='#0f766e';context.beginPath();context.roundRect(labelX,labelY,labelWidth,18,5);context.fill()
      context.fillStyle='#fff';context.fillText(label,labelX+6,labelY+9)
    }
    // The drop indicator is a solid line on the boundary the dragged band will
    // land in front of, so a move reads as "it goes here" before letting go.
    if(movePreview){
      context.save();context.strokeStyle='#0f766e';context.lineWidth=3
      if(movePreview.axis==='column'){const x=columnPosition(movePreview.destination);context.beginPath();context.moveTo(Math.round(x)+.5,0);context.lineTo(Math.round(x)+.5,size.height)}
      else{const y=rowPosition(movePreview.destination);context.beginPath();context.moveTo(0,Math.round(y)+.5);context.lineTo(size.width,Math.round(y)+.5)}
      context.stroke();context.restore()
    }
  },[size,scroll,rowAxis,columnAxis,frozenRows,frozenColumns,cells,conditionalCells,activeRow,activeColumn,activeCell,zoom,visibleRange,collaborators,userLabels,sheetId,selection.startRow,selection.startColumn,selection.endRow,selection.endColumn,fillPreview,validations,showFormulas,showGridlines,resizePreview,movePreview])

  const handleApplied=useMemo(()=>forSheet<MutationResult>(sheetId,applied=>{onVersion(applied.server_version);reportEdit(applied);if(!applied.duplicate&&applied.applied_cells>0)recordOperation(applied.operation_id);setSaveState(applied.conflicts?.length?'conflict':'saved',applied.conflicts?.length||0)}),[sheetId,onVersion,recordOperation,setSaveState,reportEdit])

  const readOnlyNotice=useCallback(()=>{setSaveState('error');alert('보기 전용 권한입니다. 소유자에게 편집 권한을 요청하세요.')},[setSaveState])
  const queueCells=useCallback(async(inputs:PastedCell[],endpoint:'batch'|'paste'|'fill')=>{
    if(inputs.length===0)return
    if(readOnly){readOnlyNotice();return}
    if(inputs.length>(endpoint==='batch'?1000:MAX_PASTE_CELLS)){setSaveState('error');alert(endpoint==='batch'?'한 번에 최대 1,000셀까지 변경할 수 있습니다.':`${endpoint==='fill'?'자동 채우기':'붙여넣기'}는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`);return}
    const spillChild=inputs.find(input=>cells.get(cellKey(input.row,input.column))?.spill_source)
    if(spillChild){const source=cells.get(cellKey(spillChild.row,spillChild.column))?.spill_source;setSaveState('error');alert(`${address(spillChild.row,spillChild.column)}은(는) ${source} 배열 수식의 결과입니다. 원본 수식 셀을 편집하세요.`);return}
    const checked=validateClientInputs(validations,inputs)
    if(checked.rejected.length){const first=checked.rejected[0];setSaveState('error');alert(`${address(first.row,first.column)}: ${first.message}`);return}
    if(checked.warnings.length&&!confirm(`${checked.warnings.length.toLocaleString()}개 셀이 데이터 검증 조건을 만족하지 않습니다. 그래도 입력할까요?`))return
    const updatedAt=new Date().toISOString()
    putCells(inputs.map(cell=>({sheet_id:sheetId,...cell,updated_at:updatedAt})))
    setSaveState(navigator.onLine?'saving':'offline')
    const id=newIdempotencyKey()
    await enqueue({id,sheetId,workbookId,endpoint,attempts:0,createdAt:Date.now(),body:{base_version:version,idempotency_key:id,client_id:collaborationClientId(),cells:inputs}})
    await flushOutbox(handleApplied)
  },[cells,handleApplied,putCells,setSaveState,sheetId,workbookId,version,validations,readOnly,readOnlyNotice])

  const saveCell=useCallback(async(value:unknown,formula:string,row:number,column:number,numberFormat?:string)=>{
    if(readOnly){readOnlyNotice();return false}
    const current=cells.get(cellKey(row,column))
    if(current?.spill_source){setSaveState('error');alert(`${address(row,column)}은(는) ${current.spill_source} 배열 수식의 결과입니다. 원본 수식 셀을 편집하세요.`);return false}
    // 입력한 모습을 되살리는 표시 형식은 셀에 이미 형식이 있으면 덮지 않는다.
    // 서식을 지정해 둔 칸에 값을 넣는 것은 서식을 바꾸겠다는 뜻이 아니다.
    const existing=current?.style as Record<string,unknown>|undefined
    const style=numberFormat&&typeof existing?.number_format!=='string'?{...(existing??{}),number_format:numberFormat}:existing
    const input={row,column,value,formula,style}
    const checked=validateClientInputs(validations,[input])
    if(checked.rejected.length){setSaveState('error');alert(`${address(row,column)}: ${checked.rejected[0].message}`);return false}
    if(checked.warnings.length&&!confirm(`${address(row,column)} 값이 데이터 검증 조건을 만족하지 않습니다. 그래도 입력할까요?`))return false
    const cell:Cell={sheet_id:sheetId,...input,updated_at:new Date().toISOString()}
    putCell(cell);setSaveState(navigator.onLine?'saving':'offline')
    const id=newIdempotencyKey()
    await enqueue({id,sheetId,workbookId,endpoint:'batch',attempts:0,createdAt:Date.now(),body:{base_version:version,idempotency_key:id,client_id:collaborationClientId(),cells:[input]}})
    await flushOutbox(handleApplied);return true
  },[sheetId,workbookId,version,cells,putCell,setSaveState,handleApplied,validations,readOnly,readOnlyNotice])

  const commit=useCallback(async(raw:string,row=activeRow,column=activeColumn)=>{
    const formula=raw.startsWith('=')?raw:''
    const parsed=formula?{value:undefined,numberFormat:undefined}:parseTypedCellValue(raw)
    let value:unknown=parsed.value
    if(formula&&navigator.onLine){
      // Formula evaluation happens before the outbox write. Mark that gap as
      // saving so workbook-wide actions can wait for the draft to be durable.
      setSaveState('saving')
      const formulaCells:Record<string,unknown>={}
      cells.forEach(candidate=>{formulaCells[address(candidate.row,candidate.column)]=candidate.value})
      try{const evaluated=await api<{value?:unknown;error?:{code:string}}>(`/api/v1/formulas:evaluate`,{method:'POST',body:JSON.stringify({formula,cells:formulaCells})});value=evaluated.error?.code??formulaPreview(evaluated.value)}catch{value='#ERROR!'}
    }
    return saveCell(value,formula,row,column,parsed.numberFormat)
  },[activeRow,activeColumn,cells,saveCell,setSaveState])

  useEffect(()=>{const sync=()=>flushOutbox(handleApplied);window.addEventListener('online',sync);const timer=window.setInterval(sync,3000);sync();return()=>{window.removeEventListener('online',sync);window.clearInterval(timer)}},[handleApplied])

  const selectionPayload=useCallback((range:FillRange=selection)=>{
    const rows=range.endRow-range.startRow+1,columns=range.endColumn-range.startColumn+1
    if(rows*columns>MAX_PASTE_CELLS)throw new Error(`복사와 잘라내기는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`)
    const payload:KanpicClipboard={version:1,sourceRow:range.startRow,sourceColumn:range.startColumn,rows,columns,cells:[]}
    for(let row=range.startRow;row<=range.endRow;row+=1)for(let column=range.startColumn;column<=range.endColumn;column+=1){
      const cell=cells.get(cellKey(row,column))
      payload.cells.push({rowOffset:row-range.startRow,columnOffset:column-range.startColumn,value:cell?.value,formula:cell?.formula||undefined,style:stripMergeStyle(cell?.style)})
    }
    return payload
  },[cells,selection])
  const fillSelection=useCallback(async(target:FillRange)=>{try{const inputs=materializeFill(selectionPayload(),target);await queueCells(inputs,'fill')}catch(error){setSaveState('error');alert(error instanceof Error?error.message:'자동 채우기를 적용하지 못했습니다.')}},[queueCells,selectionPayload,setSaveState])
  const fillFrom=useCallback(async(source:FillRange)=>{try{const inputs=materializeFill(selectionPayload(source),selection);await queueCells(inputs,'fill')}catch(error){setSaveState('error');alert(error instanceof Error?error.message:'선택 범위를 채우지 못했습니다.')}},[queueCells,selection,selectionPayload,setSaveState])
  const fillDown=useCallback(()=>void fillFrom({startRow:selection.startRow,startColumn:selection.startColumn,endRow:selection.startRow,endColumn:selection.endColumn}),[fillFrom,selection])
  const fillRight=useCallback(()=>void fillFrom({startRow:selection.startRow,startColumn:selection.startColumn,endRow:selection.endRow,endColumn:selection.startColumn}),[fillFrom,selection])
  const fillDraft=useCallback(async(raw:string)=>{try{const formula=raw.startsWith('=')?raw:undefined,current=cells.get(cellKey(activeRow,activeColumn)),parsed=formula?{value:undefined,numberFormat:undefined}:parseTypedCellValue(raw),existing=current?.style as Record<string,unknown>|undefined,style=parsed.numberFormat&&typeof existing?.number_format!=='string'?{...(existing??{}),number_format:parsed.numberFormat}:existing,payload:KanpicClipboard={version:1,sourceRow:activeRow,sourceColumn:activeColumn,rows:1,columns:1,cells:[{rowOffset:0,columnOffset:0,value:parsed.value,formula,style:stripMergeStyle(style)}]},inputs=[{row:activeRow,column:activeColumn,value:parsed.value,formula,style},...materializeFill(payload,selection)];await queueCells(inputs,'fill');setEditing(false)}catch(error){setSaveState('error');alert(error instanceof Error?error.message:'선택 범위에 값을 채우지 못했습니다.')}},[activeColumn,activeRow,cells,queueCells,selection,setEditing,setSaveState])

  const selectCell=useCallback((row:number,column:number,extend=false)=>{
    let nextRow=Math.max(1,Math.min(TOTAL_ROWS,row)),nextColumn=Math.max(1,Math.min(TOTAL_COLUMNS,column));if(rowAxis.isHidden(nextRow))nextRow=row>=activeRow?rowAxis.firstVisibleAtOrAfter(nextRow):rowAxis.lastVisibleAtOrBefore(nextRow);if(columnAxis.isHidden(nextColumn))nextColumn=column>=activeColumn?columnAxis.firstVisibleAtOrAfter(nextColumn):columnAxis.lastVisibleAtOrBefore(nextColumn)
    const merged=!extend?cellMerge(cells.get(cellKey(nextRow,nextColumn))):undefined
    if(merged){nextRow=merged.startRow;nextColumn=merged.startColumn}
    select(nextRow,nextColumn,extend);sendCursor({sheet_id:sheetId,row:nextRow,column:nextColumn})
    const start=extend?{row:Math.min(anchorRow,nextRow),column:Math.min(anchorColumn,nextColumn)}:{row:merged?.startRow??nextRow,column:merged?.startColumn??nextColumn}
    const end=extend?{row:Math.max(anchorRow,nextRow),column:Math.max(anchorColumn,nextColumn)}:{row:merged?.endRow??nextRow,column:merged?.endColumn??nextColumn}
    sendSelection({sheet_id:sheetId,start,end})
  },[anchorRow,anchorColumn,cells,select,sendCursor,sendSelection,sheetId,rowAxis,columnAxis,activeRow,activeColumn])
  // Keeping the active cell on a caller-chosen corner stops whole-row and
  // whole-column selections from scrolling to the end of the grid.
  const selectSpan=useCallback((startRow:number,startColumn:number,endRow:number,endColumn:number,active?:{row:number;column:number})=>{
    const target=active??{row:endRow,column:endColumn}
    const anchor={row:target.row===startRow?endRow:startRow,column:target.column===startColumn?endColumn:startColumn}
    select(anchor.row,anchor.column);select(target.row,target.column,true)
    sendCursor({sheet_id:sheetId,row:target.row,column:target.column})
    sendSelection({sheet_id:sheetId,start:{row:startRow,column:startColumn},end:{row:endRow,column:endColumn}})
  },[select,sendCursor,sendSelection,sheetId])
  const selectRange=useCallback((startRow:number,startColumn:number,endRow:number,endColumn:number)=>selectSpan(startRow,startColumn,endRow,endColumn,{row:startRow,column:startColumn}),[selectSpan])
  const populated=useCallback((row:number,column:number)=>{const cell=cells.get(cellKey(row,column));return cell?.value!=null||Boolean(cell?.formula)},[cells])
  const moveDataEdge=useCallback((direction:'up'|'down'|'left'|'right',extend:boolean)=>{const rowStep=direction==='up'?-1:direction==='down'?1:0,columnStep=direction==='left'?-1:direction==='right'?1:0,filled=populated(activeRow,activeColumn);let row=activeRow,column=activeColumn;for(let index=0;index<TOTAL_ROWS+TOTAL_COLUMNS;index+=1){const nextRow=rowStep?rowAxis.nextVisible(row,rowStep):row,nextColumn=columnStep?columnAxis.nextVisible(column,columnStep):column;if(nextRow===row&&nextColumn===column)break;if(populated(nextRow,nextColumn)!==filled)break;row=nextRow;column=nextColumn}selectCell(row,column,extend)},[activeColumn,activeRow,columnAxis,populated,rowAxis,selectCell])
  const commitAndMoveRef=useRef<(rowOffset:number,columnOffset:number)=>void>(()=>{})
  // Enter and Tab close the editor and move on straight away. Saving a formula
  // needs a server round trip, and waiting for it used to swallow whatever was
  // typed in the next cell before the save came back.
  const commitAndMove=(rowOffset:number,columnOffset:number)=>{
    const row=activeRow,column=activeColumn,text=draft
    setEditing(false)
    selectCell(rowOffset===0?row:rowAxis.nextVisible(row,rowOffset>0?1:-1),columnOffset===0?column:columnAxis.nextVisible(column,columnOffset>0?1:-1))
    void commit(text,row,column)
  }
  // The shortcut listener is registered once, so it reaches the current commit
  // through a ref instead of a stale closure over the draft.
  commitAndMoveRef.current=commitAndMove
  const pointerPosition=(event:{clientX:number;clientY:number})=>{const rect=canvas.current!.getBoundingClientRect();return{x:event.clientX-rect.left,y:event.clientY-rect.top}}
  const pointCell=(event:React.PointerEvent<HTMLCanvasElement>)=>{const{x,y}=pointerPosition(event);if(x<headerWidth||y<headerHeight)return;return{row:axisIndexAtViewport(rowAxis,y-headerHeight,scroll.top,frozenRows),column:axisIndexAtViewport(columnAxis,x-headerWidth,scroll.left,frozenColumns)}}
  const geometry:GridGeometry={rowAxis,columnAxis,scroll,frozenRows,frozenColumns,headerWidth:headerWidth,headerHeight:headerHeight}
  const rowPositionOf=(row:number)=>headerHeight+axisViewportPosition(rowAxis,row,scroll.top,frozenRows)
  const columnPositionOf=(column:number)=>headerWidth+axisViewportPosition(columnAxis,column,scroll.left,frozenColumns)
  const regionAt=(x:number,y:number)=>pointerRegion(geometry,x,y)
  /** The outline control under the pointer, if the pointer is in the gutter. */
  const outlineControlAt=(x:number,y:number):{axis:'row'|'column';group:DimensionGroup}|undefined=>{
    if(x<rowOutline&&y>=headerHeight&&layout.row_groups?.length){
      const level=Math.floor(x/OUTLINE_STEP)
      const index=axisIndexAtViewport(rowAxis,y-headerHeight,scroll.top,frozenRows)
      const group=controlAt(layout.row_groups,index,level)
      if(group)return{axis:'row',group}
    }
    if(y<columnOutline&&x>=headerWidth&&layout.column_groups?.length){
      const level=Math.floor(y/OUTLINE_STEP)
      const index=axisIndexAtViewport(columnAxis,x-headerWidth,scroll.left,frozenColumns)
      const group=controlAt(layout.column_groups,index,level)
      if(group)return{axis:'column',group}
    }
    return undefined
  }
  const toggleGroup=(axis:'row'|'column',group:DimensionGroup)=>
    void applyLayoutCommand({action:group.collapsed?'expand':'collapse',axis,start:group.start,count:group.end-group.start+1})
  const resizeTargetAt=(x:number,y:number)=>resizeHandleAt(geometry,x,y,RESIZE_HANDLE)
  const wholeColumnsSelected=selection.startRow<=1&&selection.endRow>=TOTAL_ROWS,wholeRowsSelected=selection.startColumn<=1&&selection.endColumn>=TOTAL_COLUMNS
  // Dragging a boundary inside a whole-row or whole-column selection resizes
  // every selected dimension at once, like Sheets and Excel do.
  const resizeSpan=(target:ResizeTarget)=>{
    if(target.axis==='column'&&wholeColumnsSelected&&target.index>=selection.startColumn&&target.index<=selection.endColumn)return{start:selection.startColumn,count:selection.endColumn-selection.startColumn+1}
    if(target.axis==='row'&&wholeRowsSelected&&target.index>=selection.startRow&&target.index<=selection.endRow)return{start:selection.startRow,count:selection.endRow-selection.startRow+1}
    return{start:target.index,count:1}
  }
  const onFillHandle=(event:React.PointerEvent<HTMLCanvasElement>)=>{if(readOnly)return false;if(rowAxis.hidden.length>0||columnAxis.hidden.length>0)return false;const{x,y}=pointerPosition(event),handleX=headerWidth+axisViewportPosition(columnAxis,selection.endColumn,scroll.left,frozenColumns)+columnAxis.sizeOf(selection.endColumn),handleY=headerHeight+axisViewportPosition(rowAxis,selection.endRow,scroll.top,frozenRows)+rowAxis.sizeOf(selection.endRow);return Math.abs(x-handleX)<=8&&Math.abs(y-handleY)<=8}
  const applyLayoutCommand=useCallback(async(command:LayoutCommand)=>{if(!onLayout)return;try{await onLayout(command)}catch{/* the editor page reports layout failures */}},[onLayout])
  const applyStructureCommand=useCallback(async(command:StructureCommand)=>{if(!onStructure)return;try{await onStructure(command)}catch{/* the editor page reports structure failures */}},[onStructure])
  // Selecting another cell must save what is being typed. The selection change
  // itself clears the editing flag, so the pending text is committed first.
  const flushEdit=()=>{
    const state=useEditorStore.getState()
    if(!state.editing)return
    setEditing(false)
    void commit(state.draft,state.activeRow,state.activeColumn)
  }
  const pointerDown=(event:React.PointerEvent<HTMLCanvasElement>)=>{
    if(event.button!==0)return
    setMenu(undefined)
    flushEdit()
    const{x,y}=pointerPosition(event),target=onLayout?resizeTargetAt(x,y):undefined
    // A click on an outline control folds or unfolds before anything else is
    // considered, because the control sits over the header area.
    const control=onLayout?outlineControlAt(x,y):undefined
    if(control){toggleGroup(control.axis,control.group);focusGrid();return}
    if(target){
      const axis=target.axis==='column'?columnAxis:rowAxis,span=resizeSpan(target),size=Math.round(axis.sizeOf(target.index)/zoom)
      resizeDrag.current={axis:target.axis,index:target.index,origin:target.axis==='column'?event.clientX:event.clientY,start:span.start,count:span.count,size}
      setResizePreview({axis:target.axis,index:target.index,size})
      event.currentTarget.setPointerCapture(event.pointerId);event.preventDefault();focusGrid();return
    }
    const region=regionAt(x,y)
    if(region.kind==='column'&&filterColumns&&onMenuCommand&&region.index>=filterColumns.start&&region.index<=filterColumns.end){
      const left=columnPositionOf(region.index),width=columnAxis.sizeOf(region.index)
      if(width>34&&x>=left+width-22&&x<=left+width-4){
        const rect=canvas.current!.getBoundingClientRect()
        onMenuCommand({command:'column-filter',column:region.index,x:rect.left+left+width-22,y:rect.top+headerHeight+2})
        focusGrid();return
      }
    }
    if(region.kind==='cell'&&!readOnly){
      // A click on a checkbox flips it in place, the way a checkbox behaves
      // everywhere else.
      const rule=validationForCell(validations,region.row,region.column)
      const state=rule?checkboxState(rule,cells.get(cellKey(region.row,region.column))?.value):undefined
      if(state){selectCell(region.row,region.column);focusGrid();void saveCell(state.next,'',region.row,region.column);return}
    }
    if(region.kind==='corner'){selectSpan(1,1,TOTAL_ROWS,TOTAL_COLUMNS,{row:1,column:1});focusGrid();return}
    if(region.kind==='column'||region.kind==='row'){
      // Pressing a header that is already part of the selected band picks the
      // band up to move it; pressing any other header starts a new selection.
      // That is the rule Sheets uses and it keeps both gestures on one button.
      const bandStart=region.kind==='column'?selection.startColumn:selection.startRow,bandEnd=region.kind==='column'?selection.endColumn:selection.endRow
      const whole=region.kind==='column'?wholeColumnsSelected:wholeRowsSelected
      if(!readOnly&&onStructure&&!event.shiftKey&&whole&&region.index>=bandStart&&region.index<=bandEnd){
        moveDrag.current={axis:region.kind,start:bandStart,count:bandEnd-bandStart+1,origin:region.kind==='column'?event.clientX:event.clientY,destination:bandStart,armed:false}
        event.currentTarget.setPointerCapture(event.pointerId);event.preventDefault();focusGrid();return
      }
      const anchor=event.shiftKey?(region.kind==='column'?anchorColumn:anchorRow):region.index
      headerDrag.current={axis:region.kind,anchor}
      if(region.kind==='column')selectSpan(1,Math.min(anchor,region.index),TOTAL_ROWS,Math.max(anchor,region.index),{row:1,column:region.index})
      else selectSpan(Math.min(anchor,region.index),1,Math.max(anchor,region.index),TOTAL_COLUMNS,{row:region.index,column:1})
      event.currentTarget.setPointerCapture(event.pointerId);event.preventDefault();focusGrid();return
    }
    if(onFillHandle(event)){filling.current=true;fillPreviewRef.current={...selection};setFillPreview({...selection});event.currentTarget.style.cursor='crosshair';event.currentTarget.setPointerCapture(event.pointerId);focusGrid();event.preventDefault();return}
    dragging.current=true;event.currentTarget.setPointerCapture(event.pointerId);selectCell(region.row,region.column,event.shiftKey);focusGrid()
  }
  const pointerMove=(event:React.PointerEvent<HTMLCanvasElement>)=>{
    const drag=resizeDrag.current
    if(drag){
      const axis=drag.axis==='column'?columnAxis:rowAxis
      const delta=((drag.axis==='column'?event.clientX:event.clientY)-drag.origin)/zoom
      const size=clampDimensionSize(drag.axis,axis.sizeOf(drag.index)/zoom+delta)
      setResizePreview({axis:drag.axis,index:drag.index,size});return
    }
    const moving=moveDrag.current
    if(moving){
      const travelled=Math.abs((moving.axis==='column'?event.clientX:event.clientY)-moving.origin)
      if(!moving.armed&&travelled<6)return
      moving.armed=true
      event.currentTarget.style.cursor=moving.axis==='column'?'grabbing':'grabbing'
      const{x,y}=pointerPosition(event)
      const axis=moving.axis==='column'?columnAxis:rowAxis
      const offset=moving.axis==='column'?Math.max(0,x-headerWidth):Math.max(0,y-headerHeight)
      const index=axisIndexAtViewport(axis,offset,moving.axis==='column'?scroll.left:scroll.top,moving.axis==='column'?frozenColumns:frozenRows)
      const edge=moving.axis==='column'?columnPositionOf(index):rowPositionOf(index)
      const past=(moving.axis==='column'?x:y)-edge>axis.sizeOf(index)/2
      const destination=Math.max(1,past?index+1:index)
      if(destination!==moving.destination){moving.destination=destination;setMovePreview({axis:moving.axis,destination})}
      else if(!movePreview)setMovePreview({axis:moving.axis,destination})
      return
    }
    if(headerDrag.current){
      const{x,y}=pointerPosition(event),header=headerDrag.current
      if(header.axis==='column'){const index=axisIndexAtViewport(columnAxis,Math.max(0,x-headerWidth),scroll.left,frozenColumns);selectSpan(1,Math.min(header.anchor,index),TOTAL_ROWS,Math.max(header.anchor,index),{row:1,column:index})}
      else{const index=axisIndexAtViewport(rowAxis,Math.max(0,y-headerHeight),scroll.top,frozenRows);selectSpan(Math.min(header.anchor,index),1,Math.max(header.anchor,index),TOTAL_COLUMNS,{row:index,column:1})}
      return
    }
    if(filling.current){const cell=pointCell(event);if(!cell)return;const next={startRow:Math.min(selection.startRow,cell.row),startColumn:Math.min(selection.startColumn,cell.column),endRow:Math.max(selection.endRow,cell.row),endColumn:Math.max(selection.endColumn,cell.column)};fillPreviewRef.current=next;setFillPreview(next);return}
    if(!dragging.current){
      const{x,y}=pointerPosition(event),target=onLayout?resizeTargetAt(x,y):undefined
      const hoveredRegion=target?undefined:regionAt(x,y)
      const overBand=!readOnly&&!!onStructure&&((hoveredRegion?.kind==='column'&&wholeColumnsSelected&&hoveredRegion.index>=selection.startColumn&&hoveredRegion.index<=selection.endColumn)||(hoveredRegion?.kind==='row'&&wholeRowsSelected&&hoveredRegion.index>=selection.startRow&&hoveredRegion.index<=selection.endRow))
      event.currentTarget.style.cursor=formatBrush?'copy':target?target.axis==='column'?'col-resize':'row-resize':overBand?'grab':onFillHandle(event)?'crosshair':'default'
      // Hovering a cell that carries a note shows the note, which is the only
      // way to read one. A cell holding a formula error is explained the same
      // way, because `#VALUE!` on its own says nothing about what to fix.
      const hovered=pointCell(event)
      const hoveredCell=hovered?cells.get(cellKey(hovered.row,hovered.column)):undefined
      const note=hoveredCell?.note
      const failure=hoveredCell?explainFormulaError(formulaErrorCode(hoveredCell.value)??''):undefined
      if(hovered&&(note||failure)){
        if(noteHover?.row!==hovered.row||noteHover?.column!==hovered.column)
          setNoteHover({row:hovered.row,column:hovered.column,text:note,failure,x:columnPositionOf(hovered.column),y:rowPositionOf(hovered.row)+rowAxis.sizeOf(hovered.row)})
      }else if(noteHover)setNoteHover(undefined)
      return
    }
    const cell=pointCell(event);if(cell)selectCell(cell.row,cell.column,true)
  }
  const finishGesture=(event:React.PointerEvent<HTMLCanvasElement>)=>{
    // A selection made while the brush is loaded takes the copied format and
    // the brush is put down, which is what one click of a format painter does.
    if(formatBrush&&onPaintFormat&&(dragging.current||headerDrag.current))onPaintFormat(selectedBounds(useEditorStore.getState()))
    dragging.current=false;headerDrag.current=null;moveDrag.current=null;filling.current=false;fillPreviewRef.current=undefined;setFillPreview(undefined);setMovePreview(undefined)
    event.currentTarget.style.cursor=formatBrush?'copy':'default'
    if(event.currentTarget.hasPointerCapture(event.pointerId))event.currentTarget.releasePointerCapture(event.pointerId)
  }
  const pointerUp=(event:React.PointerEvent<HTMLCanvasElement>)=>{
    const drag=resizeDrag.current,preview=resizePreview
    const move=moveDrag.current
    resizeDrag.current=null
    const target=fillPreviewRef.current,shouldFill=filling.current
    finishGesture(event)
    if(drag){setResizePreview(undefined);if(preview&&preview.size!==drag.size)void applyLayoutCommand({action:'resize',axis:drag.axis,start:drag.start,count:drag.count,size:preview.size});return}
    // A drop inside the band itself lands it exactly where it started, so it is
    // dismissed rather than sent to the server as a no-op version bump.
    if(move?.armed&&(move.destination<move.start||move.destination>move.start+move.count)){
      void applyStructureCommand({axis:move.axis,action:'move',index:move.start,count:move.count,destination:move.destination})
      return
    }
    if(shouldFill&&target)void fillSelection(target)
  }
  const pointerCancel=(event:React.PointerEvent<HTMLCanvasElement>)=>{resizeDrag.current=null;setResizePreview(undefined);finishGesture(event)}
  // The cell editor input stays mounted at the active cell even when nobody is
  // editing. Keeping one focused input is what makes IME composition work from
  // the very first keystroke and what keeps typing alive after a commit moves
  // the selection to the next cell.
  const focusGrid=useCallback(()=>{
    const input=editorInput.current
    if(input)input.focus({preventScroll:true})
    else viewport.current?.focus()
  },[])
  // Entering edit mode from a key, a menu or a double click puts the caret at
  // the end of the existing text. A composition in flight is left untouched so
  // the first Hangul syllable is not dropped.
  useEffect(()=>{
    const input=editorInput.current
    if(!input||!editing||composing.current)return
    // Editing can also be driven from the formula bar, which keeps its own caret.
    const active=document.activeElement
    if(active&&active!==input&&(active.tagName==='INPUT'||active.tagName==='TEXTAREA'))return
    input.focus({preventScroll:true})
    const end=input.value.length
    input.setSelectionRange(end,end)
  },[editing])
  const beginTyping=useCallback((text:string)=>{
    const input=editorInput.current
    const reject=()=>{if(input)input.value=''}
    if(readOnly){reject();readOnlyNotice();return}
    if(activeCell?.spill_source){
      reject()
      const source=parsedAddress(activeCell.spill_source)
      if(source)selectCell(source.row,source.column)
      alert(`${address(activeRow,activeColumn)}은(는) ${activeCell.spill_source} 배열 수식의 결과입니다. 원본 수식 셀에서 입력하세요.`)
      return
    }
    setDraft(text);setEditing(true)
  },[activeCell,activeColumn,activeRow,readOnly,readOnlyNotice,selectCell,setDraft,setEditing])
  const editActiveCell=useCallback(()=>{if(readOnly){readOnlyNotice();return}if(activeCell?.spill_source){const source=parsedAddress(activeCell.spill_source);if(source){selectCell(source.row,source.column);setEditing(true);return}}setEditing(true)},[activeCell,selectCell,setEditing,readOnly,readOnlyNotice])
  const keyDown=(event:React.KeyboardEvent)=>{if(editing)return;const primary=event.ctrlKey||event.metaKey
    if(primary&&event.shiftKey&&event.key.toLowerCase()==='v'){pasteAsValues.current=true;return}
    if(primary&&event.key.toLowerCase()==='a'){selectRange(1,1,TOTAL_ROWS,TOTAL_COLUMNS);event.preventDefault()}
    else if(primary&&event.code==='Space'){selectRange(1,activeColumn,TOTAL_ROWS,activeColumn);event.preventDefault()}
    else if(event.shiftKey&&event.code==='Space'){selectRange(activeRow,1,activeRow,TOTAL_COLUMNS);event.preventDefault()}
    else if(primary&&['ArrowUp','ArrowDown','ArrowLeft','ArrowRight'].includes(event.key)){const direction=event.key.replace('Arrow','').toLowerCase() as 'up'|'down'|'left'|'right';moveDataEdge(direction,event.shiftKey);event.preventDefault()}
    else if(primary&&!event.shiftKey&&event.key.toLowerCase()==='d'){if(readOnly)readOnlyNotice();else fillDown();event.preventDefault()}
    else if(primary&&!event.shiftKey&&event.key.toLowerCase()==='r'){if(readOnly)readOnlyNotice();else fillRight();event.preventDefault()}
    else if(primary&&event.key==='Home'){selectCell(1,1);event.preventDefault()}
    else if(primary&&event.key==='End'){selectCell(TOTAL_ROWS,TOTAL_COLUMNS);event.preventDefault()}
    else if(event.key==='Home'){selectCell(activeRow,columnAxis.firstVisibleAtOrAfter(1),event.shiftKey);event.preventDefault()}
    else if(event.key==='End'){moveDataEdge('right',event.shiftKey);event.preventDefault()}
    else if(event.key===' '&&activeCheckbox){event.preventDefault();void saveCell(activeCheckbox.next,'',activeRow,activeColumn)}
    else if(event.key==='Enter'&&activeCheckbox){event.preventDefault();void saveCell(activeCheckbox.next,'',activeRow,activeColumn)}
    else if(event.key==='Enter'||event.key==='F2'){editActiveCell();event.preventDefault()}
    else if(event.key==='ArrowDown'){selectCell(rowAxis.nextVisible(activeRow,1),activeColumn,event.shiftKey);event.preventDefault()}
    else if(event.key==='ArrowUp'){selectCell(rowAxis.nextVisible(activeRow,-1),activeColumn,event.shiftKey);event.preventDefault()}
    else if(event.key==='ArrowRight'||event.key==='Tab'){selectCell(activeRow,columnAxis.nextVisible(activeColumn,event.key==='Tab'&&event.shiftKey?-1:1),event.shiftKey);event.preventDefault()}
    else if(event.key==='ArrowLeft'){selectCell(activeRow,columnAxis.nextVisible(activeColumn,-1),event.shiftKey);event.preventDefault()}
    else if(event.key==='ContextMenu'||(event.shiftKey&&event.key==='F10')){
      const rect=canvas.current?.getBoundingClientRect()
      setMenu({
        x:(rect?.left??0)+headerWidth+axisViewportPosition(columnAxis,activeColumn,scroll.left,frozenColumns)+10,
        y:(rect?.top??0)+headerHeight+axisViewportPosition(rowAxis,activeRow,scroll.top,frozenRows)+10,
        items:cellMenuItems(selection),label:'셀 메뉴',
      })
      event.preventDefault()
    }
    else if(event.key==='Backspace'||event.key==='Delete'){if(readOnly){readOnlyNotice();event.preventDefault();return}const count=(selection.endRow-selection.startRow+1)*(selection.endColumn-selection.startColumn+1);if(count===1)void commit('');else void clearSelection();event.preventDefault()}
  }
  const writeClipboard=(event:React.ClipboardEvent)=>{try{const payload=selectionPayload();internalClipboard.current=payload;event.preventDefault();event.clipboardData.setData('text/plain',clipboardText(payload));event.clipboardData.setData('text/html',clipboardHtml(payload));event.clipboardData.setData(KANPIC_CLIPBOARD_TYPE,JSON.stringify(payload));return true}catch(error){event.preventDefault();alert(error instanceof Error?error.message:'선택 범위를 복사하지 못했습니다.');return false}}
  const copy=(event:React.ClipboardEvent)=>{if(editing)return;writeClipboard(event)}
  const clearSelection=useCallback(async()=>{
    const count=(selection.endRow-selection.startRow+1)*(selection.endColumn-selection.startColumn+1)
    if(count>MAX_PASTE_CELLS){alert(`잘라내기와 삭제는 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`);return}
    const empty:PastedCell[]=[]
    for(let row=selection.startRow;row<=selection.endRow;row+=1)for(let column=selection.startColumn;column<=selection.endColumn;column+=1)empty.push({row,column})
    await queueCells(empty,'paste')
  },[queueCells,selection.endColumn,selection.endRow,selection.startColumn,selection.startRow])
  const cut=(event:React.ClipboardEvent)=>{if(editing)return;if(writeClipboard(event))void clearSelection()}
  const runPaste=useCallback((text:string,internal:string,mode:PasteMode,html?:string)=>{
    // 다른 스프레드시트의 표는 HTML 쪽에 서식과 실제 값을 함께 담아 온다.
    // 워커에는 DOMParser 가 없으므로 여기서 읽어 넘긴다.
    const htmlCells=internal?undefined:html?parseClipboardHtml(html,MAX_PASTE_CELLS):undefined
    const worker=new Worker(new URL('../workers/paste.worker.ts',import.meta.url),{type:'module'})
    worker.onmessage=async(message:MessageEvent<{cells?:PastedCell[];error?:string}>)=>{try{
      if(message.data.error){setSaveState('error');alert(message.data.error);return}
      let pasted=message.data.cells??[]
      // Pasting formatting alone keeps whatever each target cell already holds.
      if(mode==='format'){
        if(pasted.length===0){setSaveState('error');alert('서식만 붙여넣기는 kanpic에서 복사한 셀에만 사용할 수 있습니다.');return}
        pasted=pasted.map(cell=>{const current=cells.get(cellKey(cell.row,cell.column));return {...cell,value:current?.formula?undefined:current?.value,formula:current?.formula}})
      }
      await queueCells(pasted,'paste')
    }finally{worker.terminate()}}
    worker.onerror=()=>{setSaveState('error');worker.terminate();alert('붙여넣기 데이터를 처리하지 못했습니다.')}
    worker.postMessage({text,internal,startRow:activeRow,startColumn:activeColumn,mode,htmlCells})
  },[activeColumn,activeRow,cells,queueCells,setSaveState])
  const paste=(event:React.ClipboardEvent)=>{
    // While a cell is open the paste belongs to the text being edited.
    if(editing)return
    event.preventDefault()
    // 클립보드에 그림이 있으면 글자가 아니라 그림을 넣는다. 화면을 캡처해
    // Ctrl+V 를 누른 사람은 그 그림이 시트에 놓이기를 바란다.
    const picture=Array.from(event.clipboardData.files).find(file=>file.type.startsWith('image/'))
    if(picture&&onPasteImage){onPasteImage(picture);return}
    const valuesOnly=pasteAsValues.current;pasteAsValues.current=false
    runPaste(event.clipboardData.getData('text/plain'),event.clipboardData.getData(KANPIC_CLIPBOARD_TYPE),valuesOnly?'values':'all',event.clipboardData.getData('text/html'))
  }
  // Menu-driven clipboard actions cannot rely on a browser clipboard event, so
  // they use the async Clipboard API and keep the last internal copy in memory
  // to preserve formulas and styles that plain text cannot carry.
  const copySelection=useCallback(async(cut:boolean)=>{
    try{
      const payload=selectionPayload(),text=clipboardText(payload)
      internalClipboard.current=payload
      // 서식이 따라가려면 표도 함께 올려야 한다. ClipboardItem 을 쓸 수 없는
      // 브라우저에서는 글자만이라도 올린다.
      if(navigator.clipboard?.write&&typeof ClipboardItem!=='undefined'){
        await navigator.clipboard.write([new ClipboardItem({
          'text/plain':new Blob([text],{type:'text/plain'}),
          'text/html':new Blob([clipboardHtml(payload)],{type:'text/html'}),
        })])
      }
      else if(navigator.clipboard?.writeText)await navigator.clipboard.writeText(text)
      else if(!document.execCommand('copy'))throw new Error('클립보드를 사용할 수 없습니다.')
      if(cut)await clearSelection()
    }catch(error){alert(error instanceof Error?error.message:'선택 범위를 복사하지 못했습니다.')}
  },[clearSelection,selectionPayload])
  const pasteFromClipboard=useCallback(async(mode:PasteMode)=>{
    try{
      if(!navigator.clipboard?.readText)throw new Error('이 브라우저에서는 Ctrl/⌘+V로 붙여넣어 주세요.')
      const text=await navigator.clipboard.readText()
      const cached=internalClipboard.current
      runPaste(text,cached&&clipboardText(cached)===text?JSON.stringify(cached):'',mode)
    }catch(error){alert(error instanceof Error?error.message:'클립보드를 읽지 못했습니다. Ctrl/⌘+V를 사용하세요.')}
  },[runPaste])
  const copySelectionLink=useCallback(async()=>{
    const parameters=new URLSearchParams({sheet_id:sheetId,range:selection.startRow===selection.endRow&&selection.startColumn===selection.endColumn?address(selection.startRow,selection.startColumn):`${address(selection.startRow,selection.startColumn)}:${address(selection.endRow,selection.endColumn)}`})
    const link=`${window.location.origin}${window.location.pathname}?${parameters.toString()}`
    try{await navigator.clipboard?.writeText(link)}catch{window.prompt('셀 링크를 복사하세요.',link)}
  },[selection,sheetId])
  // Moves by one visible screen, matching the Page Up and Page Down keys.
  const movePage=useCallback((direction:'up'|'down'|'left'|'right',extend:boolean)=>{
    if(direction==='up'||direction==='down'){
      const step=Math.max(1,Math.floor(Math.max(1,size.height-headerHeight-rowAxis.offsetOf(frozenRows+1))/Math.max(8,rowAxis.sizeOf(activeRow))))
      let row=activeRow
      for(let index=0;index<step;index+=1){const next=rowAxis.nextVisible(row,direction==='down'?1:-1);if(next===row)break;row=next}
      selectCell(row,activeColumn,extend);return
    }
    const step=Math.max(1,Math.floor(Math.max(1,size.width-headerWidth-columnAxis.offsetOf(frozenColumns+1))/Math.max(16,columnAxis.sizeOf(activeColumn))))
    let column=activeColumn
    for(let index=0;index<step;index+=1){const next=columnAxis.nextVisible(column,direction==='right'?1:-1);if(next===column)break;column=next}
    selectCell(activeRow,column,extend)
  },[activeColumn,activeRow,columnAxis,frozenColumns,frozenRows,rowAxis,selectCell,size.height,size.width])
  // Sums the contiguous numbers above the active cell, then to its left, and
  // leaves the formula in the editor so it can be adjusted before committing.
  /**
   * Builds an aggregate over the numbers directly above or to the left of the
   * active cell. A range selection wins over the scan, so choosing cells first
   * aggregates exactly those cells.
   */
  const insertAggregate=useCallback((name:string)=>{
    const multiple=selection.startRow!==selection.endRow||selection.startColumn!==selection.endColumn
    if(multiple){
      const target=selection.endRow+1<=TOTAL_ROWS?{row:selection.endRow+1,column:selection.startColumn}:{row:selection.startRow,column:selection.endColumn+1}
      const reference=`${address(selection.startRow,selection.startColumn)}:${address(selection.endRow,selection.endColumn)}`
      const formula=`=${name}(${reference})`
      pendingDraft.current={row:target.row,column:target.column,text:formula}
      selectCell(target.row,target.column)
      setDraft(formula);setEditing(true);return
    }
    let row=activeRow-1
    while(row>=1&&typeof cells.get(cellKey(row,activeColumn))?.value==='number')row-=1
    if(row<activeRow-1){
      // The walk stops where the loaded rows stop, which on a long column is
      // not where the numbers stop. The page reads the column from the server
      // and reports the real start; a SUM over the visible tail looks right
      // and is wrong.
      if(onResolveNumericRun){
        void onResolveNumericRun(activeRow,activeColumn).then(start=>{
          const first=start??row+1
          setDraft(`=${name}(${address(first,activeColumn)}:${address(activeRow-1,activeColumn)})`)
          setEditing(true)
        })
        return
      }
      setDraft(`=${name}(${address(row+1,activeColumn)}:${address(activeRow-1,activeColumn)})`);setEditing(true);return
    }
    let column=activeColumn-1
    while(column>=1&&typeof cells.get(cellKey(activeRow,column))?.value==='number')column-=1
    setDraft(column<activeColumn-1?`=${name}(${address(activeRow,column+1)}:${address(activeRow,activeColumn-1)})`:`=${name}()`)
    setEditing(true)
  },[activeColumn,activeRow,cells,onResolveNumericRun,selectCell,selection.endColumn,selection.endRow,selection.startColumn,selection.startRow,setEditing])
  const autoSum=useCallback(()=>insertAggregate('SUM'),[insertAggregate])
  useEffect(()=>{const shortcut=(event:Event)=>{
    const detail=(event as CustomEvent<GridShortcut>).detail
    if(!detail)return
    const pad=(value:number)=>String(value).padStart(2,'0'),now=new Date()
    switch(detail.command){
      case 'fill-down':fillDown();return
      case 'fill-right':fillRight();return
      case 'select-all':selectSpan(1,1,TOTAL_ROWS,TOTAL_COLUMNS,{row:1,column:1});return
      case 'select-row':selectSpan(activeRow,1,activeRow,TOTAL_COLUMNS,{row:activeRow,column:1});return
      case 'select-column':selectSpan(1,activeColumn,TOTAL_ROWS,activeColumn,{row:1,column:activeColumn});return
      case 'select-data-region':{const region=dataRegion(cells,activeRow,activeColumn,{rows:TOTAL_ROWS,columns:TOTAL_COLUMNS});selectSpan(region.startRow,region.startColumn,region.endRow,region.endColumn,{row:region.startRow,column:region.startColumn});return}
      case 'move-first':selectCell(1,1);return
      case 'move-last':selectCell(TOTAL_ROWS,TOTAL_COLUMNS);return
      case 'move-data-edge':moveDataEdge(detail.direction,detail.extend);return
      case 'move-page':movePage(detail.direction,detail.extend);return
      case 'clear-contents':void clearSelection();return
      case 'auto-sum':autoSum();return
      case 'insert-today':void commit(`${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}`);return
      case 'insert-now':void commit(`${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`);return
      case 'copy':void copySelection(false);return
      case 'cut':void copySelection(true);return
      case 'paste':void pasteFromClipboard('all');return
      case 'paste-values':void pasteFromClipboard('values');return
      case 'paste-special':void pasteFromClipboard(detail.mode);return
      case 'focus-grid':focusGrid();return
      case 'commit-draft':if(useEditorStore.getState().editing){focusGrid();commitAndMoveRef.current(1,0)}return
      case 'insert-text':if(readOnly){readOnlyNotice();return}setDraft(detail.text);setEditing(true);return
      // A dialog that already collected everything writes the cell outright
      // rather than leaving a half-typed formula waiting for Enter.
      case 'commit-text':if(readOnly){readOnlyNotice();return}void commit(detail.text);return
      case 'insert-function':if(readOnly){readOnlyNotice();return}insertAggregate(detail.name);return
    }
  };window.addEventListener('kanpic:grid-shortcut',shortcut);return()=>window.removeEventListener('kanpic:grid-shortcut',shortcut)},[activeColumn,activeRow,autoSum,cells,clearSelection,commit,copySelection,fillDown,fillRight,moveDataEdge,movePage,focusGrid,insertAggregate,pasteFromClipboard,readOnly,readOnlyNotice,selectCell,selectSpan,setDraft,setEditing])
  // Auto-fit measures the cells already loaded in the client store, which is
  // the same data the canvas paints.
  const autoFitSize=(axis:'row'|'column',start:number,count:number)=>{
    const context=canvas.current?.getContext('2d')
    if(!context)return undefined
    const end=start+count-1
    let measured=0
    context.save()
    cells.forEach(cell=>{
      const index=axis==='column'?cell.column:cell.row
      if(index<start||index>end)return
      const style=(cell.style??{}) as Record<string,unknown>
      const fontSize=typeof style.font_size==='number'?style.font_size:12,fontFamily=typeof style.font_family==='string'?JSON.stringify(style.font_family):'Inter, Pretendard, sans-serif'
      context.font=`${style.italic===true?'italic ':''}${style.bold===true?'600':'400'} ${fontSize}px ${fontFamily}`
      const text=showFormulas&&cell.formula?cell.formula:formatCellValue(cell.value,style)
      if(!text)return
      if(axis==='column'){measured=Math.max(measured,context.measureText(text).width+18);return}
      const width=columnAxis.sizeOf(cell.column)/zoom
      const wrapped=style.text_mode==='wrap'||style.wrap===true?wrapText(text,Math.max(0,width-12),value=>context.measureText(value).width).length:1
      measured=Math.max(measured,wrapped*Math.max(fontSize*1.25,12)+9)
    })
    context.restore()
    if(measured===0)return axis==='column'?DEFAULT_COLUMN_WIDTH:DEFAULT_ROW_HEIGHT
    return clampDimensionSize(axis,measured)
  }
  const autoFit=(target:ResizeTarget)=>{
    const span=resizeSpan(target),size=autoFitSize(target.axis,span.start,span.count)
    if(size===undefined)return
    void applyLayoutCommand({action:'resize',axis:target.axis,start:span.start,count:span.count,size})
  }
  const doubleClick=(event:React.MouseEvent<HTMLCanvasElement>)=>{
    const{x,y}=pointerPosition(event),target=onLayout?resizeTargetAt(x,y):undefined
    if(target){autoFit(target);return}
    if(regionAt(x,y).kind==='cell'&&!activeCheckbox)editActiveCell()
  }
  // The grid holds only the rows on screen, so the table to sort is worked out
  // on the editor page from what the server says the sheet contains. Deciding
  // it here would sort the visible window and leave the rest out of order.
  const sortByColumn=(column:number,direction:'asc'|'desc')=>onMenuCommand?.({command:'sort-column',column,direction})
  const clipboardMenuItems=():MenuItem[]=>[
    {kind:'item',label:'잘라내기',shortcut:'Ctrl+X',icon:<Scissors/>,disabled:readOnly,onSelect:()=>void copySelection(true)},
    {kind:'item',label:'복사',shortcut:'Ctrl+C',icon:<Copy/>,onSelect:()=>void copySelection(false)},
    {kind:'item',label:'붙여넣기',shortcut:'Ctrl+V',icon:<ClipboardPaste/>,disabled:readOnly,onSelect:()=>void pasteFromClipboard('all')},
    {kind:'submenu',label:'특수 붙여넣기',icon:<Clipboard/>,disabled:readOnly,items:[
      {kind:'item',label:'값만 붙여넣기',shortcut:'Ctrl+Shift+V',onSelect:()=>void pasteFromClipboard('values')},
      {kind:'item',label:'서식만 붙여넣기',onSelect:()=>void pasteFromClipboard('format')},
      {kind:'item',label:'행과 열 바꿔 붙여넣기',onSelect:()=>void pasteFromClipboard('transpose')},
    ]},
  ]
  const cellMenuItems=(range:FillRange):MenuItem[]=>{
    const rows=range.endRow-range.startRow+1,columns=range.endColumn-range.startColumn+1
    const merged=Boolean(cellMerge(cells.get(cellKey(range.startRow,range.startColumn))))
    const rowLabel=rows>1?`행 ${range.startRow}–${range.endRow}`:`행 ${range.startRow}`,columnLabel=columns>1?`열 ${columnName(range.startColumn)}–${columnName(range.endColumn)}`:`열 ${columnName(range.startColumn)}`
    return [...clipboardMenuItems(),{kind:'separator'},
      {kind:'item',label:`위에 행 ${rows}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis:'row',action:'insert',index:range.startRow,count:rows})},
      {kind:'item',label:`아래에 행 ${rows}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis:'row',action:'insert',index:range.endRow+1,count:rows})},
      {kind:'item',label:`왼쪽에 열 ${columns}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis:'column',action:'insert',index:range.startColumn,count:columns})},
      {kind:'item',label:`오른쪽에 열 ${columns}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis:'column',action:'insert',index:range.endColumn+1,count:columns})},
      {kind:'separator'},
      {kind:'item',label:`${rowLabel} 삭제`,icon:<Trash2/>,danger:true,disabled:readOnly||!onStructure,onSelect:()=>{if(window.confirm(`${rowLabel}을(를) 삭제할까요?`))void applyStructureCommand({axis:'row',action:'delete',index:range.startRow,count:rows})}},
      {kind:'item',label:`${columnLabel} 삭제`,icon:<Trash2/>,danger:true,disabled:readOnly||!onStructure,onSelect:()=>{if(window.confirm(`${columnLabel}을(를) 삭제할까요?`))void applyStructureCommand({axis:'column',action:'delete',index:range.startColumn,count:columns})}},
      {kind:'item',label:'내용 지우기',shortcut:'Delete',icon:<Eraser/>,disabled:readOnly,onSelect:()=>void clearSelection()},
      {kind:'item',label:'서식 지우기',shortcut:'Ctrl+\\',disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'clear-format'})},
      {kind:'separator'},
      {kind:'item',label:merged?'셀 병합 해제':'셀 병합',disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'merge',merge:!merged})},
      {kind:'submenu',label:'데이터',disabled:readOnly||!onMenuCommand,items:[
        {kind:'item',label:'범위 정렬…',icon:<ArrowUpAZ/>,onSelect:()=>onMenuCommand?.({command:'sort-dialog'})},
        {kind:'item',label:'필터 보기…',icon:<Filter/>,onSelect:()=>onMenuCommand?.({command:'filter'})},
        {kind:'item',label:'데이터 검증…',icon:<BadgeCheck/>,onSelect:()=>onMenuCommand?.({command:'data-validation'})},
        {kind:'item',label:'조건부 서식…',icon:<Palette/>,onSelect:()=>onMenuCommand?.({command:'conditional-format'})},
        {kind:'item',label:'이름 범위 지정…',icon:<Link2/>,onSelect:()=>onMenuCommand?.({command:'named-range'})},
        {kind:'separator'},
        {kind:'item',label:'부분합…',icon:<Sigma/>,onSelect:()=>onMenuCommand?.({command:'subtotal'})},
        {kind:'item',label:'중복 항목 삭제…',icon:<Rows3/>,onSelect:()=>onMenuCommand?.({command:'cleanup-duplicates'})},
        {kind:'item',label:'텍스트를 열로 분할…',onSelect:()=>onMenuCommand?.({command:'split-columns'})},
      ]},
      {kind:'submenu',label:'삽입',disabled:readOnly||!onMenuCommand,items:[
        {kind:'item',label:'차트 만들기…',icon:<BarChart3/>,onSelect:()=>onMenuCommand?.({command:'chart'})},
        {kind:'item',label:'피벗 테이블 만들기…',icon:<Table2/>,onSelect:()=>onMenuCommand?.({command:'pivot'})},
        {kind:'item',label:'댓글 추가',icon:<MessageSquarePlus/>,onSelect:()=>onMenuCommand?.({command:'comment'})},
        {kind:'item',label:activeCell?.note?'메모 편집…':'메모 삽입…',icon:<StickyNote/>,disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'note'})},
      ]},
	  {kind:'submenu',label:'Workbook Agent',icon:<Bot/>,disabled:!onMenuCommand,items:[
		{kind:'item',label:'Agent에게 질문',onSelect:()=>onMenuCommand?.({command:'agent',mode:'agent',request:''})},
		{kind:'item',label:'선택 범위 분석',onSelect:()=>onMenuCommand?.({command:'agent',mode:'summarize',request:'선택 범위의 핵심 지표와 패턴을 분석해줘'})},
		{kind:'separator'},
		{kind:'item',label:'수식 생성',onSelect:()=>onMenuCommand?.({command:'agent',mode:'formula',request:'선택 범위에 필요한 수식을 만들어줘'})},
		{kind:'item',label:'수식 설명',onSelect:()=>onMenuCommand?.({command:'agent',mode:'explain',request:'선택한 수식의 계산 방식과 참조를 설명해줘'})},
		{kind:'item',label:'수식 오류 수정',disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'agent',mode:'fix',request:'선택 범위의 잘못된 수식을 찾아 고쳐줘'})},
		{kind:'item',label:'데이터 정리',disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'agent',mode:'clean',request:'선택 범위의 중복과 형식 불일치를 정리해줘'})},
		{kind:'item',label:'표 서식 적용',disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'agent',mode:'format',request:'선택 범위를 읽기 쉬운 표 서식으로 정리해줘'})},
		{kind:'item',label:'차트 생성',disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'agent',mode:'chart',request:'선택 범위에 적합한 차트를 만들어줘'})},
		{kind:'item',label:'패턴 자동 채우기',disabled:readOnly,onSelect:()=>onMenuCommand?.({command:'agent',mode:'formula',request:'선택 범위의 패턴을 분석해 전체 행에 수식을 채워줘'})},
	  ]},
      {kind:'separator'},
      {kind:'item',label:'편집 기록 표시',icon:<History/>,disabled:!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'cell-history',row:range.startRow,column:range.startColumn})},
      {kind:'item',label:'셀 링크 복사',icon:<Link2/>,onSelect:()=>void copySelectionLink()},
    ]
  }
  const axisMenuItems=(axis:'row'|'column',index:number,span:{start:number;count:number}):MenuItem[]=>{
    const isColumn=axis==='column'
    const first=span.start,count=span.count
    const label=isColumn?count>1?`열 ${columnName(first)}–${columnName(first+count-1)}`:`열 ${columnName(first)}`:count>1?`행 ${first}–${first+count-1}`:`행 ${first}`
    const unit=isColumn?'열':'행'
    return [{kind:'label',label},...clipboardMenuItems(),{kind:'separator'},
      {kind:'item',label:isColumn?`왼쪽에 열 ${count}개 삽입`:`위에 행 ${count}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis,action:'insert',index:first,count})},
      {kind:'item',label:isColumn?`오른쪽에 열 ${count}개 삽입`:`아래에 행 ${count}개 삽입`,disabled:readOnly||!onStructure,onSelect:()=>void applyStructureCommand({axis,action:'insert',index:first+count,count})},
      {kind:'item',label:isColumn?'왼쪽으로 이동':'위로 이동',disabled:readOnly||!onStructure||first<=1,onSelect:()=>void applyStructureCommand({axis,action:'move',index:first,count,destination:first-1})},
      {kind:'item',label:isColumn?'오른쪽으로 이동':'아래로 이동',disabled:readOnly||!onStructure||first+count>(isColumn?TOTAL_COLUMNS:TOTAL_ROWS),onSelect:()=>void applyStructureCommand({axis,action:'move',index:first,count,destination:first+count+1})},
      {kind:'item',label:`${label} 삭제`,icon:<Trash2/>,danger:true,disabled:readOnly||!onStructure,onSelect:()=>{if(window.confirm(`${label}을(를) 삭제할까요?`))void applyStructureCommand({axis,action:'delete',index:first,count})}},
      {kind:'item',label:`${label} 내용 지우기`,icon:<Eraser/>,disabled:readOnly,onSelect:()=>void clearSelection()},
      {kind:'separator'},
      ...(isColumn?[{kind:'item',label:'이 열 통계 보기',icon:<BarChart3/>,disabled:!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'column-stats'})} as MenuItem]:[]),
      {kind:'item',label:isColumn?'열 너비 자동 맞춤':'행 높이 자동 맞춤',disabled:readOnly||!onLayout,onSelect:()=>autoFit({axis,index})},
      {kind:'item',label:isColumn?'열 너비 지정…':'행 높이 지정…',disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'layout-dialog'})},
      {kind:'item',label:`${label} 숨기기`,icon:<EyeOff/>,disabled:readOnly||!onLayout,onSelect:()=>void applyLayoutCommand({action:'hide',axis,start:first,count})},
      {kind:'item',label:`${label} 그룹화`,icon:<ChevronsDownUp/>,disabled:readOnly||!onLayout||count<2,onSelect:()=>void applyLayoutCommand({action:'group',axis,start:first,count})},
      {kind:'item',label:'그룹 해제',icon:<ChevronsUpDown/>,disabled:readOnly||!onLayout||!innermostGroup(isColumn?layout.column_groups:layout.row_groups,first,first+count-1),onSelect:()=>void applyLayoutCommand({action:'ungroup',axis,start:first,count})},
      {kind:'item',label:`모든 ${unit} 표시`,disabled:readOnly||!onLayout,onSelect:()=>void applyLayoutCommand({action:'show_all',axis})},
      {kind:'item',label:isColumn?`${columnName(index)}열까지 고정`:`${index}행까지 고정`,icon:<PanelTop/>,disabled:readOnly||!onLayout,onSelect:()=>void applyLayoutCommand({action:'freeze',frozen_rows:isColumn?frozenRows:index,frozen_columns:isColumn?index:frozenColumns})},
      {kind:'item',label:'고정 해제',disabled:readOnly||!onLayout||(frozenRows===0&&frozenColumns===0),onSelect:()=>void applyLayoutCommand({action:'freeze',frozen_rows:0,frozen_columns:0})},
      ...(isColumn?[{kind:'separator'} as MenuItem,
        {kind:'item',label:'이 열 기준 오름차순 정렬',icon:<ArrowUpAZ/>,disabled:readOnly||!onMenuCommand,onSelect:()=>sortByColumn(index,'asc')} as MenuItem,
        {kind:'item',label:'이 열 기준 내림차순 정렬',icon:<ArrowDownAZ/>,disabled:readOnly||!onMenuCommand,onSelect:()=>sortByColumn(index,'desc')} as MenuItem,
        {kind:'item',label:'필터 보기…',icon:<Filter/>,disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'filter'})} as MenuItem,
      ]:[]),
      {kind:'separator'},
      {kind:'item',label:'조건부 서식…',icon:<Palette/>,disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'conditional-format'})},
      {kind:'item',label:'서식 지우기',shortcut:'Ctrl+\\',disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'clear-format'})},
    ]
  }
  const cornerMenuItems=():MenuItem[]=>[
    {kind:'item',label:'전체 선택',shortcut:'Ctrl+A',onSelect:()=>selectSpan(1,1,TOTAL_ROWS,TOTAL_COLUMNS,{row:1,column:1})},
    {kind:'item',label:'붙여넣기',shortcut:'Ctrl+V',icon:<ClipboardPaste/>,disabled:readOnly,onSelect:()=>void pasteFromClipboard('all')},
    {kind:'separator'},
    {kind:'item',label:'모든 행 표시',icon:<Rows3/>,disabled:readOnly||!onLayout,onSelect:()=>void applyLayoutCommand({action:'show_all',axis:'row'})},
    {kind:'item',label:'모든 열 표시',disabled:readOnly||!onLayout,onSelect:()=>void applyLayoutCommand({action:'show_all',axis:'column'})},
    {kind:'item',label:'고정 해제',icon:<PanelTop/>,disabled:readOnly||!onLayout||(frozenRows===0&&frozenColumns===0),onSelect:()=>void applyLayoutCommand({action:'freeze',frozen_rows:0,frozen_columns:0})},
    {kind:'separator'},
    {kind:'item',label:'시트 레이아웃…',disabled:readOnly||!onMenuCommand,onSelect:()=>onMenuCommand?.({command:'layout-dialog'})},
  ]
  const openContextMenu=(event:React.MouseEvent<HTMLCanvasElement>)=>{
    event.preventDefault()
    const{x,y}=pointerPosition(event),region=regionAt(x,y)
    focusGrid()
    if(region.kind==='corner'){selectSpan(1,1,TOTAL_ROWS,TOTAL_COLUMNS,{row:1,column:1});setMenu({x:event.clientX,y:event.clientY,items:cornerMenuItems(),label:'시트 전체 메뉴'});return}
    if(region.kind==='column'){
      const inside=wholeColumnsSelected&&region.index>=selection.startColumn&&region.index<=selection.endColumn
      if(!inside)selectSpan(1,region.index,TOTAL_ROWS,region.index,{row:1,column:region.index})
      const span=inside?{start:selection.startColumn,count:selection.endColumn-selection.startColumn+1}:{start:region.index,count:1}
      setMenu({x:event.clientX,y:event.clientY,items:axisMenuItems('column',region.index,span),label:'열 메뉴'});return
    }
    if(region.kind==='row'){
      const inside=wholeRowsSelected&&region.index>=selection.startRow&&region.index<=selection.endRow
      if(!inside)selectSpan(region.index,1,region.index,TOTAL_COLUMNS,{row:region.index,column:1})
      const span=inside?{start:selection.startRow,count:selection.endRow-selection.startRow+1}:{start:region.index,count:1}
      setMenu({x:event.clientX,y:event.clientY,items:axisMenuItems('row',region.index,span),label:'행 메뉴'});return
    }
    const outside=region.row<selection.startRow||region.row>selection.endRow||region.column<selection.startColumn||region.column>selection.endColumn
    if(outside)selectCell(region.row,region.column)
    const range=outside?{startRow:region.row,startColumn:region.column,endRow:region.row,endColumn:region.column}:selection
    setMenu({x:event.clientX,y:event.clientY,items:cellMenuItems(range),label:'셀 메뉴'})
  }
  const activeMerge=cellMerge(activeCell),inputStartRow=activeMerge?.startRow??activeRow,inputStartColumn=activeMerge?.startColumn??activeColumn,inputEndRow=activeMerge?.endRow??activeRow,inputEndColumn=activeMerge?.endColumn??activeColumn
  const inputVisibleStart=rowAxis.firstVisibleAtOrAfter(inputStartRow),inputVisibleColumn=columnAxis.firstVisibleAtOrAfter(inputStartColumn),inputLeft=headerWidth+axisViewportPosition(columnAxis,inputVisibleColumn,scroll.left,frozenColumns),inputTop=headerHeight+axisViewportPosition(rowAxis,inputVisibleStart,scroll.top,frozenRows),inputWidth=columnAxis.rangeSize(inputStartColumn,inputEndColumn),inputHeight=rowAxis.rangeSize(inputStartRow,inputEndRow)
  const dropdown=!activeCell?.spill_source&&(activeValidation?.rule_type==='list'||activeValidation?.rule_type==='list_range')&&activeValidation.show_dropdown?activeValidation:undefined
  // A chart cell has no text, so what is announced is a description of it.
  const activeSparkline=parseSparkline(activeCell?.value)
  // The chip is how a link is opened: a bare click still selects the cell, so
  // moving around a sheet full of links never navigates away by accident.
  const activeLink=editing?undefined:cellLink(activeCell)
  const textEditing=editing&&!dropdown
  const hint=textEditing?formulaHint(functionCatalog,draft,caret):undefined
  // Repeating an entry already in the column is the most common thing anybody
  // types, so those values are offered whenever a formula hint is not showing.
  const valueSuggestions=textEditing&&!hint?suggestColumnValues(cells,activeColumn,activeRow,draft):[]
  /**
   * 값과 캐럿을 지금 맞춰 두고, React 가 다시 그린 뒤에 한 번 더 맞춘다.
   *
   * 다음 프레임에 캐럿을 되돌릴 때는 **그 사이에 사람이 더 치지 않았을 때만**
   * 해야 한다. 눌린 순간의 자리를 나중에 그대로 다시 적용하면, 그 사이에 친
   * 글자가 앞으로 끌려가 "둘째 줄" 이 "줄둘째 " 가 된다. 부하가 걸려 프레임이
   * 늦을수록 잘 일어난다.
   */
  const settleCaret=(expected:string,start:number,end:number)=>{
    requestAnimationFrame(()=>{
      const field=editorInput.current
      if(field&&field.value===expected)field.setSelectionRange(start,end)
    })
  }
  // Accepting a suggestion rewrites the draft and puts the caret inside the
  // brackets, so typing can continue with the arguments.
  const chooseValue=(value:string)=>{
    setDraft(value)
    setCaret(value.length)
    setSuggestion(0)
    settleCaret(value,value.length,value.length)
  }
  const chooseSuggestion=(name:string)=>{
    if(!hint)return
    const next=applySuggestion(draft,hint.context,name)
    setDraft(next.text)
    setCaret(next.caret)
    setSuggestion(0)
    settleCaret(next.text,next.caret,next.caret)
  }
  const selectionAddress=selection.startRow===selection.endRow&&selection.startColumn===selection.endColumn?address(activeRow,activeColumn):`${address(selection.startRow,selection.startColumn)}:${address(selection.endRow,selection.endColumn)}`
  return <div className="grid-viewport" id="kanpic-grid" ref={viewport} tabIndex={0} onFocus={event=>{if(event.target===event.currentTarget)focusGrid()}} onScroll={(event)=>setScroll({left:event.currentTarget.scrollLeft,top:event.currentTarget.scrollTop})} onKeyDown={keyDown} onCopy={copy} onCut={cut} onPaste={paste} aria-label="스프레드시트 그리드">
    <div className="grid-spacer" style={{width:headerWidth+columnAxis.extent,height:headerHeight+rowAxis.extent}}><canvas ref={canvas} className="grid-canvas" data-conditional-cells={conditionalCells.size} onPointerDown={pointerDown} onPointerMove={pointerMove} onPointerUp={pointerUp} onPointerCancel={pointerCancel} onDoubleClick={doubleClick} onContextMenu={openContextMenu}/></div>
    {menu&&<ContextMenu x={menu.x} y={menu.y} items={menu.items} label={menu.label} onClose={()=>{setMenu(undefined);focusGrid()}}/>}
    {dropdown&&!editing&&<button className="cell-dropdown-trigger" aria-label={`${selectionAddress} 드롭다운 열기`} title={dropdown.help_text||'드롭다운 선택'} style={{left:inputLeft+inputWidth-23,top:inputTop,width:22,height:inputHeight}} onClick={()=>setEditing(true)}>▾</button>}
    {editing&&dropdown&&<div className="cell-dropdown" role="listbox" aria-label={`${selectionAddress} 드롭다운`} style={{left:inputLeft,top:inputTop+inputHeight,minWidth:Math.max(inputWidth,180)}}>{ruleOptions(dropdown)?.map((option,index)=><button role="option" aria-selected={optionForValue(dropdown,activeCell?.value)===option} aria-label={`드롭다운 값 ${optionLabel(option)}`} key={index} onClick={()=>{setEditing(false);focusGrid();void saveCell(option.value,'',activeRow,activeColumn)}}><i style={{background:option.color||'#e5e7eb'}}/><span>{optionLabel(option)}</span></button>)}<button className="cell-dropdown-cancel" onClick={()=>{setEditing(false);focusGrid()}}>취소</button></div>}
    <textarea ref={editorInput} className={`cell-editor${textEditing?'':' idle'}`} aria-label={`${selectionAddress} 셀 입력`} rows={1} spellCheck={false}
      style={textEditing?{left:inputLeft,top:inputTop,width:inputWidth,height:inputHeight}:{left:inputLeft,top:inputTop}}
      value={textEditing?draft:''}
      onCompositionStart={()=>{composing.current=true}}
      onCompositionEnd={()=>{composing.current=false}}
      onChange={(event)=>{setCaret(event.target.selectionStart??event.target.value.length);setSuggestion(0);if(textEditing)setDraft(event.target.value);else beginTyping(event.target.value)}}
      onSelect={(event)=>setCaret(event.currentTarget.selectionStart??0)}
      onBlur={()=>{if(textEditing){setEditing(false);void commit(draft)}}}
      onKeyDown={(event)=>{
        if(!textEditing)return
        const primary=event.ctrlKey||event.metaKey
        if(primary&&event.shiftKey&&event.key.toLowerCase()==='v'){pasteAsValues.current=true;return}
        // A key that ends an IME composition must not also commit the cell.
        if(composing.current||event.nativeEvent.isComposing)return
        const suggestions=hint?.matches??[]
        if(suggestions.length>0){
          if(event.key==='ArrowDown'){event.preventDefault();setSuggestion((suggestion+1)%suggestions.length);return}
          if(event.key==='ArrowUp'){event.preventDefault();setSuggestion((suggestion-1+suggestions.length)%suggestions.length);return}
          if(event.key==='Tab'||(event.key==='Enter'&&!primary)){event.preventDefault();chooseSuggestion(suggestions[suggestion].name);return}
          if(event.key==='Escape'){event.preventDefault();setSuggestion(-1);return}
        }
        // A column suggestion is taken with Tab or the arrow keys only. Enter
        // stays a commit so typing a value that happens to be a prefix of an
        // existing one is never rewritten.
        if(valueSuggestions.length>0&&suggestion>=0){
          if(event.key==='ArrowDown'){event.preventDefault();setSuggestion((suggestion+1)%valueSuggestions.length);return}
          if(event.key==='ArrowUp'){event.preventDefault();setSuggestion((suggestion-1+valueSuggestions.length)%valueSuggestions.length);return}
          if(event.key==='Tab'){event.preventDefault();chooseValue(valueSuggestions[suggestion]);return}
          if(event.key==='Escape'){event.preventDefault();setSuggestion(-1);return}
        }
        // F4 로 참조를 상대 → 절대 → 행 고정 → 열 고정 순으로 돌린다.
        if(event.key==='F4'){
          const field=editorInput.current
          const cycled=cycleReference(draft,field?.selectionStart??draft.length,field?.selectionEnd??draft.length)
          if(cycled){
            event.preventDefault()
            setDraft(cycled.text);setCaret(cycled.end)
            if(field){field.value=cycled.text;field.setSelectionRange(cycled.start,cycled.end)}
            settleCaret(cycled.text,cycled.start,cycled.end)
          }
          return
        }
        if(event.key==='Enter'&&event.altKey){
          event.preventDefault()
          const field=editorInput.current
          const at=field?.selectionStart??draft.length,to=field?.selectionEnd??at
          const next=draft.slice(0,at)+'\n'+draft.slice(to)
          setDraft(next);setCaret(at+1)
          // 캐럿을 다음 프레임에만 옮기면 그 사이에 친 글자가 이전 자리로
          // 들어간다. 값과 캐럿을 지금 맞춰 두고, React 가 다시 그린 뒤에도
          // 한 번 더 맞춘다.
          if(field){field.value=next;field.setSelectionRange(at+1,at+1)}
          settleCaret(next,at+1,at+1)
          return
        }
        if(primary&&event.key==='Enter'){event.preventDefault();void fillDraft(draft)}
        else if(event.key==='Enter'){event.preventDefault();commitAndMove(event.shiftKey?-1:1,0)}
        else if(event.key==='Tab'){event.preventDefault();commitAndMove(0,event.shiftKey?-1:1)}
        else if(event.key==='Escape'){event.preventDefault();setEditing(false);setDraft(activeText)}
      }}/>
    {textEditing&&hint&&suggestion>=0&&<FormulaAutocomplete hint={hint} active={suggestion} left={inputLeft} top={inputTop+inputHeight+1} onChoose={chooseSuggestion}/>}
    {valueSuggestions.length>0&&suggestion>=0&&<div className="value-suggest" role="listbox" aria-label="열 값 제안" style={{left:inputLeft,top:inputTop+inputHeight+1,minWidth:Math.max(inputWidth,150)}}>
      {valueSuggestions.map((value,index)=><button key={value} role="option" aria-selected={index===suggestion} className={index===suggestion?'active':undefined}
        onMouseDown={event=>{event.preventDefault();chooseValue(value)}}>{value}</button>)}
      <small>Tab으로 채우기</small>
    </div>}
    {activeLink&&<div className="cell-link" style={{left:inputLeft,top:inputTop+inputHeight+2}}>
      <a href={activeLink.href} target={activeLink.internal?undefined:'_blank'} rel="noreferrer noopener" title={activeLink.href}
        onClick={event=>{
          if(!activeLink.internal)return
          // A link into this workbook moves the selection instead of reloading:
          // a reload would drop the editor's unsaved state to go two cells over.
          const target=workbookRangeTarget(activeLink.href)
          if(target&&onOpenRange&&onOpenRange(target.sheetId,target.range)){event.preventDefault();return}
          event.preventDefault();window.location.assign(activeLink.href)
        }}><Link2/> {activeLink.linkLabel}</a>
    </div>}
    {noteHover&&<div className={noteHover.failure?'cell-note cell-error':'cell-note'} role="tooltip" style={{left:noteHover.x,top:noteHover.y+2}}>
      {noteHover.failure&&<><strong>{noteHover.failure.code} {noteHover.failure.summary}</strong><span>{noteHover.failure.next}</span></>}
      {noteHover.text&&<span>{noteHover.text}</span>}
    </div>}
    <div className="sr-only" aria-live="polite">선택 범위 {selectionAddress}, 활성 셀 값 {activeSparkline?describeSparkline(activeSparkline):activeText||'비어 있음'}{activeCell?.note?`, 메모 ${activeCell.note}`:''}{activeCell?.spill_source?`, ${activeCell.spill_source} 배열 수식 결과`:''}{fillPreview?`, 자동 채우기 미리보기 ${address(fillPreview.startRow,fillPreview.startColumn)}:${address(fillPreview.endRow,fillPreview.endColumn)}`:''}</div>
  </div>
}
