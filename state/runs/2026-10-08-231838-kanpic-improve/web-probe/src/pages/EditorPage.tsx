import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AlertTriangle, AlignCenter, Eye, Grid2X2, Lock, MessageCircle, Settings, AlignLeft, AlignRight, ArrowUpDown, BadgeCheck, BarChart3, Bold, Bot, ChevronLeft, Download, Filter, Grid3X3, History, Italic, Link2, MessageSquare, MoreHorizontal, Paintbrush, Palette, Presentation, Redo2, Search, Share2, Square, Table, Table2, TableCellsMerge, TableCellsSplit, Underline, Undo2, Workflow, ZoomIn, ZoomOut } from 'lucide-react'
import { AppHeader } from '../components/AppHeader'
import { AIPanel } from '../components/AIPanel'
import { AutomationPanel } from '../components/AutomationPanel'
import { FormulaAutocomplete, formulaHint, namedFunctionDocs, sheetTableDocs, useFunctionCatalog } from '../components/FormulaAutocomplete'
import { applySuggestion } from '../lib/formulaSuggest'
import { explainFormulaError, formulaErrorCode } from '../lib/formulaError'
import { survivesChange, transformSelection } from '../lib/structureTransform'
import { FormulaIssueNotice } from '../components/FormulaIssueNotice'
import { ImageOverlay } from '../components/ImageOverlay'
import { StalledSaveNotice } from '../components/StalledSaveNotice'
import { cycleReference } from '../lib/referenceCycle'
import { CanvasGrid,type GridMenuCommand,type GridShortcut } from '../components/CanvasGrid'
import { WorkbookMenuBar,type WorkbookMenu } from '../components/WorkbookMenuBar'
import { ShareDialog,accessSummary } from '../components/ShareDialog'
import { useUserDirectory, userLabel } from '../state/directory'
import { QuickSwitcher,type QuickItem } from '../components/QuickSwitcher'
import { ApiError } from '../lib/api'
import { fetchHandoffOrigin, handoffFormatLabel, handoffOriginDetail, handoffOriginLabel, handoffResource, sendHandoff, type HandoffTarget } from '../lib/handoff'
import { ContextMenu,type MenuItem } from '../components/ContextMenu'
import { ChartDialog } from '../components/ChartDialog'
import { ChartOverlay } from '../components/ChartOverlay'
import { SlicerOverlay } from '../components/SlicerOverlay'
import { ConnectionPanel } from '../components/ConnectionPanel'
import { CellHistoryDialog } from '../components/CellHistoryDialog'
import { LinkDialog } from '../components/LinkDialog'
import { PromptDialog, type PromptRequest } from '../components/PromptDialog'
import { columnName, parseFilterRange } from '../lib/filter'
import { ChartPanel } from '../components/ChartPanel'
import '../components/ChartLauncher.css'
import { CommentPanel } from '../components/CommentPanel'
import { ConflictPanel } from '../components/ConflictPanel'
import { ConditionalFormatDialog } from '../components/ConditionalFormatDialog'
import { PresentationPanel,type PresentationRecord } from '../components/PresentationPanel'
import { GoalSeekDialog,type GoalSeekOutcome } from '../components/GoalSeekDialog'
import { DataTableDialog,type DataTableOutcome } from '../components/DataTableDialog'
import { ScenarioDialog } from '../components/ScenarioDialog'
import { FlashFillDialog } from '../components/FlashFillDialog'
import { planFlashFill,type FillPlan } from '../lib/flashFillPlan'
import { PresentationDialog,type PresentationAnalysis,type PresentationDeck,type PresentationResult,type PresentationTemplate } from '../components/PresentationDialog'
import { DataValidationDialog } from '../components/DataValidationDialog'
import { FilterDialog } from '../components/FilterDialog'
import { FormatDialog,type BorderFormatCommand } from '../components/FormatDialog'
import { NoteDialog } from '../components/NoteDialog'
import { ColumnFilterMenu } from '../components/ColumnFilterMenu'
import { ProtectedRangeDialog } from '../components/ProtectedRangeDialog'
import { brushPatch } from '../lib/formatBrush'
import { ColumnStatsPanel } from '../components/ColumnStatsPanel'
import { LayoutDialog,type LayoutCommand } from '../components/LayoutDialog'
import { NamedFunctionDialog } from '../components/NamedFunctionDialog'
import { WatchRuleDialog } from '../components/WatchRuleDialog'
import { NamedRangeDialog } from '../components/NamedRangeDialog'
import { SheetTableDialog } from '../components/SheetTableDialog'
import { PivotDialog } from '../components/PivotDialog'
import { PivotPanel } from '../components/PivotPanel'
import { PivotResultDialog } from '../components/PivotResultDialog'
import { ResizableRightPanel, type RightPanelKey } from '../components/ResizableRightPanel'
import { SheetTabs } from '../components/SheetTabs'
import { CopySheetDialog,SheetManagerDialog } from '../components/SheetManagerDialog'
import { SortDialog } from '../components/SortDialog'
import { StructureDialog,type StructureCommand } from '../components/StructureDialog'
import { VersionPanel } from '../components/VersionPanel'
import { WorkbookSearchDialog } from '../components/WorkbookSearchDialog'
import { WorkbookShortcutsDialog } from '../components/WorkbookShortcutsDialog'
import { FunctionListDialog } from '../components/FunctionListDialog'
import { api, address, newIdempotencyKey } from '../lib/api'
import { collaborationClientId } from '../lib/client'
import { MAX_GRID_COLUMNS, MAX_GRID_ROWS, MAX_PASTE_CELLS, type PastedCell } from '../lib/clipboard'
import { cellMerge,mergeInRegion,mergeStyle as applyMergeStyle,selectedMergedBounds } from '../lib/merge'
import { discardOutbox, enqueue, flushOutbox, listOutbox, retryOutbox, type OutboxOperation } from '../lib/outbox'
import { blocked } from '../lib/outboxQueue'
import { forSheet, pendingInWorkbook } from '../lib/outboxScope'
import { materializeSort,type SortOptions } from '../lib/sort'
import { dataRegion, looksLikeHeaderRow, type GridRegion } from '../lib/dataRegion'
import { cleanupText, convertTextNumbers, removeDuplicateRows, splitTextToColumns, trimWhitespace, unmergeAndFill, type SplitDelimiter } from '../lib/dataCleanup'
import { SplitDialog } from '../components/SplitDialog'
import { CleanupDialog, type CleanupTarget } from '../components/CleanupDialog'
import { SortScopeDialog } from '../components/SortScopeDialog'
import { SubtotalDialog } from '../components/SubtotalDialog'
import { planRemoveSubtotals, type SubtotalPlan } from '../lib/subtotal'
import { clearTableStyleCells, DEFAULT_TABLE_OPTIONS, TABLE_THEMES, tableStyleCells, type TableStyleOptions } from '../lib/tableStyle'
import { printAreaRegion, printableDocument, usedRegion } from '../lib/printSheet'
import { PrintOptionsDialog, loadPrintChoice, type PrintChoice } from '../components/PrintOptionsDialog'
import { CompareDialog, type CompareSource } from '../components/CompareDialog'
import { ConsolidateDialog } from '../components/ConsolidateDialog'
import { InspectDialog } from '../components/InspectDialog'
import type { TableFinding } from '../lib/inspectTable'
import { CONSOLIDATE_LABELS, type ConsolidateFunction, type ConsolidateResult, type ConsolidateSource } from '../lib/consolidate'
import { splitSheetRange, type CompareResult } from '../lib/compareLists'
import { collapsedIndexes } from '../lib/outline'
import { useCollaborationStore } from '../state/collaboration'
import type { ServerEvent } from '../state/collaboration'
import { cellKey, selectedBounds, useEditorStore } from '../state/editor'
import type { SheetImage,ShareRole,AIExecutionResult, AutomationExecutionResult, BuildInfo, Cell, CellConflict, CellConflictResolutionResult, Chart, ConditionalFormat, ConditionalFormatCell, ConditionalFormatEvaluation, DataValidation, FilterResult, FilterView, MutationResult, NamedFunction, NamedRange, Pivot, WatchRule as WatchRuleType, ProtectedRange, SheetStats, PivotData, ReplaceResult, Session, Sheet, SheetTable, Scenario, ScenarioComparison, SheetLayoutResult, Slicer, WorkbookConnections, ValidationEvaluation, Workbook, WorkbookSearchMatch } from '../types'

function patchStyle(style:Record<string,unknown>|undefined,patch:Record<string,unknown>){const merged={...(style??{})};for(const [key,value] of Object.entries(patch)){if(value===null)delete merged[key];else merged[key]=value}return merged}
function parseCellAddress(value:string){const match=/^([A-Z]+)([1-9]\d*)$/.exec(value.toUpperCase());if(!match)return;let column=0;for(const character of match[1])column=column*26+character.charCodeAt(0)-64;return{row:Number(match[2]),column}}
function parseNavigationRange(value:string){const parts=value.trim().replaceAll('$','').split(':');if(parts.length<1||parts.length>2)return;const first=parseCellAddress(parts[0]),last=parseCellAddress(parts[1]??parts[0]);if(!first||!last)return;return{startRow:Math.min(first.row,last.row),startColumn:Math.min(first.column,last.column),endRow:Math.max(first.row,last.row),endColumn:Math.max(first.column,last.column)}}
// 삭제한 자리를 사람이 읽을 수 있게 적는다. 무엇을 되돌리는지 모르면
// 되돌리기 버튼은 또 하나의 도박이 된다.
function structureSummary(command:StructureCommand){
  const axis=command.axis==='row'?'행':'열'
  const label=command.axis==='row'?String(command.index):columnName(command.index)
  return command.count>1?`${axis} ${label}부터 ${command.count}개`:`${axis} ${label}`
}
function spillInRange(cells:Map<string,Cell>,range:{startRow:number;startColumn:number;endRow:number;endColumn:number}){for(let row=range.startRow;row<=range.endRow;row+=1)for(let column=range.startColumn;column<=range.endColumn;column+=1){const cell=cells.get(cellKey(row,column));if(cell?.spill_source)return{cell,coordinate:address(row,column)}}}
// The grid keeps one input focused so IME composition works from the first
// keystroke. That input only owns the keyboard while a cell is actually open,
// so an idle grid still runs the workbook shortcuts.
function editableTarget(target:EventTarget|null){
  if(!(target instanceof HTMLElement))return false
  const editable=target.closest('input, textarea, select, [contenteditable="true"]')
  if(!editable)return false
  if(editable.classList.contains('cell-editor'))return !editable.classList.contains('idle')
  return true
}
function gridShortcut(shortcut:GridShortcut){window.dispatchEvent(new CustomEvent<GridShortcut>('kanpic:grid-shortcut',{detail:shortcut}))}
const TEXT_COLORS:Array<{label:string;value:string|null}>=[
  {label:'기본',value:null},{label:'검정',value:'#1c2b33'},{label:'회색',value:'#6b7a84'},{label:'빨강',value:'#dc2626'},
  {label:'주황',value:'#ea580c'},{label:'초록',value:'#15803d'},{label:'파랑',value:'#2563eb'},{label:'보라',value:'#7c3aed'},
]
const FILL_COLORS:Array<{label:string;value:string|null}>=[
  {label:'없음',value:null},{label:'연회색',value:'#f1f5f7'},{label:'연빨강',value:'#fee2e2'},{label:'연주황',value:'#ffedd5'},
  {label:'연노랑',value:'#fef3c7'},{label:'연초록',value:'#dcfce7'},{label:'연파랑',value:'#dbeafe'},{label:'연보라',value:'#ede9fe'},
]
const MAX_SPLIT_COLUMNS=20
const MAX_PRINT_ROWS=20_000
// 시트마다 칸을 통째로 읽어 한 번에 견주므로 한없이 늘릴 수 없다. 열둘이면
// 한 해 열두 달이 들어간다 — 통합에서 가장 흔한 모양이다.
const MAX_CONSOLIDATE_SHEETS=12
// 서버가 한 번에 재는 조건부 서식 칸 수의 한도와 같다.
const MAX_CONDITIONAL_CELLS=10_000
const CLEARABLE_STYLE_KEYS=['bold','italic','underline','strike','color','background','font_size','font_family','horizontal_align','vertical_align','number_format','text_mode','wrap','text_rotation','borders']

export function EditorPage({workbookId,build,session}:{workbookId:string;build?:BuildInfo;session?:Session}) {
  const client=useQueryClient();const workbook=useQuery({queryKey:['workbook',workbookId],queryFn:()=>api<Workbook>(`/api/v1/workbooks/${workbookId}`),retry:(count,error)=>!(error instanceof ApiError&&error.status===403)&&count<2})
  const [activeSheet,setActiveSheet]=useState<Sheet|undefined>();const [serverVersion,setServerVersion]=useState(1);const [rightPanel,setRightPanel]=useState<RightPanelKey|null>(()=>new URLSearchParams(window.location.search).has('comment_id')?'comments':'ai'),[searchOpen,setSearchOpen]=useState(false),[shortcutsOpen,setShortcutsOpen]=useState(false),[sortOpen,setSortOpen]=useState(false),[structureOpen,setStructureOpen]=useState(false),[layoutOpen,setLayoutOpen]=useState(false),[noteOpen,setNoteOpen]=useState(false),[historyCell,setHistoryCell]=useState<string>(),[linkOpen,setLinkOpen]=useState(false),[splitTarget,setSplitTarget]=useState<{region:GridRegion;cells:Map<string,Cell>}>(),[cleanup,setCleanup]=useState<{mode:'duplicates'|'trim'|'subtotals'|'numbers'|'unmerge';target:CleanupTarget}>(),[sortScope,setSortScope]=useState<{column:number;direction:'asc'|'desc';block:{region:GridRegion;cells:Map<string,Cell>};selection:GridRegion}>(),[subtotal,setSubtotal]=useState<{region:GridRegion;cells:Map<string,Cell>;headerRows:number;occupiedBelow:number}>(),[prompt,setPrompt]=useState<PromptRequest>(),[protectedOpen,setProtectedOpen]=useState(false),[columnFilter,setColumnFilter]=useState<{column:number;x:number;y:number}>(),[formatBrush,setFormatBrush]=useState<{style:Record<string,unknown>;sticky:boolean}>(),[formatOpen,setFormatOpen]=useState(false),[filterOpen,setFilterOpen]=useState(false),[validationOpen,setValidationOpen]=useState(false),[conditionalFormatOpen,setConditionalFormatOpen]=useState(false),[presentationOpen,setPresentationOpen]=useState(false),[goalSeekOpen,setGoalSeekOpen]=useState(false),[flashFill,setFlashFill]=useState<{plan:FillPlan;column:number}>(),[namedRangeOpen,setNamedRangeOpen]=useState(false),[namedFunctionOpen,setNamedFunctionOpen]=useState(false),[sheetTableOpen,setSheetTableOpen]=useState(false),[dataTableOpen,setDataTableOpen]=useState(false),[scenarioOpen,setScenarioOpen]=useState(false),[printOpen,setPrintOpen]=useState<{region?:GridRegion;rows?:number;truncated?:boolean}|false>(false),[compare,setCompare]=useState<{left:CompareSource;right:CompareSource}>(),[consolidateSources,setConsolidateSources]=useState<ConsolidateSource[]>(),[inspect,setInspect]=useState<CleanupTarget>(),[consolidateSkipped,setConsolidateSkipped]=useState<string[]>([]),[watchOpen,setWatchOpen]=useState(false),[chartDialog,setChartDialog]=useState<Chart|null>(),[pivotDialog,setPivotDialog]=useState<Pivot|null>(),[pivotResult,setPivotResult]=useState<Pivot>()
  // 수식 입력창에도 그리드와 같은 함수 제안을 붙인다. 긴 수식일수록 이쪽에
  // 쓰는데, 정작 인수 안내는 셀 안에서만 나오고 있었다.
  const namedFunctions=useQuery({queryKey:['named-functions',workbookId],queryFn:()=>api<{items:NamedFunction[]}>(`/api/v1/workbooks/${workbookId}/named-functions`)})
  const scenarios=useQuery({queryKey:['scenarios',workbookId],queryFn:()=>api<{items:Scenario[]}>(`/api/v1/workbooks/${workbookId}/scenarios`)})
  const refreshScenarios=async()=>{await client.invalidateQueries({queryKey:['scenarios',workbookId]})}
  const createScenario=async(input:Record<string,unknown>)=>{const item=await api<Scenario>(`/api/v1/workbooks/${workbookId}/scenarios`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});updateVersion(item.workbook_version);await refreshScenarios();return item}
  const updateScenario=async(id:string,input:Record<string,unknown>)=>{const item=await api<Scenario>(`/api/v1/scenarios/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshScenarios();return item}
  const deleteScenario=async(item:Scenario)=>{await api(`/api/v1/scenarios/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshScenarios()}
  const sheetTables=useQuery({queryKey:['tables',workbookId],queryFn:()=>api<{items:SheetTable[]}>(`/api/v1/workbooks/${workbookId}/tables`)})
  const functionCatalog=useFunctionCatalog([...namedFunctionDocs(namedFunctions.data?.items??[]),...sheetTableDocs(sheetTables.data?.items??[])])
  const formulaInput=useRef<HTMLTextAreaElement|null>(null)
  // 협업 이벤트 처리기는 소켓 연결과 함께 고정되므로 활성 시트를 참조로 읽는다.
  const activeSheetRef=useRef<string|undefined>(undefined)
  const [formulaCaret,setFormulaCaret]=useState(0),[formulaSuggestion,setFormulaSuggestion]=useState(0)
  const [nameBoxValue,setNameBoxValue]=useState('A1'),[pendingNavigation,setPendingNavigation]=useState<{sheetId:string;range:{startRow:number;startColumn:number;endRow:number;endColumn:number}}>()
  const [showGridlines,setShowGridlines]=useState(true),[functionsOpen,setFunctionsOpen]=useState(false)
  const [tableMenu,setTableMenu]=useState<{x:number;y:number}>(),[borderMenu,setBorderMenu]=useState<{x:number;y:number}>()
  const [tableTheme,setTableTheme]=useState('teal'),[tableOptions,setTableOptions]=useState<TableStyleOptions>(DEFAULT_TABLE_OPTIONS),[borderColor,setBorderColor]=useState('#94a3b8')
  const [showFormulas,setShowFormulas]=useState(false),[replaceMode,setReplaceMode]=useState(false),[shareOpen,setShareOpen]=useState(false),[quickOpen,setQuickOpen]=useState(false),[sheetManagerOpen,setSheetManagerOpen]=useState(false),[copySheet,setCopySheet]=useState<Sheet>(),[requestingAccess,setRequestingAccess]=useState(false),[accessRequested,setAccessRequested]=useState(false)
  const layoutQueue=useRef<Promise<unknown>>(Promise.resolve()),nameBoxRef=useRef<HTMLInputElement>(null)
  const [overflowMenu,setOverflowMenu]=useState<{x:number;y:number}>()
  const [agentDraft,setAgentDraft]=useState<{mode:Extract<GridMenuCommand,{command:'agent'}>['mode'];request:string;key:number}>({mode:'agent',request:'',key:0})
  const routeNavigation=useRef((()=>{const parameters=new URLSearchParams(window.location.search);return{sheetId:parameters.get('sheet_id')??'',range:parameters.get('range')??'',commentId:parameters.get('comment_id')??''}})()).current,routeNavigationApplied=useRef(false)
  const editor=useEditorStore();const editorSelection=selectedMergedBounds(editor.cells,selectedBounds(editor));const activeCell=editor.cells.get(cellKey(editor.activeRow,editor.activeColumn));const formula=activeCell?.formula||(activeCell?.value==null?'':String(activeCell.value))
  const conflicts=useQuery({queryKey:['cell-conflicts',workbookId,false],queryFn:()=>api<{items:CellConflict[]}>(`/api/v1/workbooks/${workbookId}/conflicts`)})
  const connect=useCollaborationStore(state=>state.connect),disconnect=useCollaborationStore(state=>state.disconnect),sendCursor=useCollaborationStore(state=>state.sendCursor),sendSelection=useCollaborationStore(state=>state.sendSelection)
  const collaborationStatus=useCollaborationStore(state=>state.status),collaborators=useCollaborationStore(state=>state.users)
  // Presence cursors and the collaborator count show names, not raw identifiers.
  const collaboratorDirectory=useUserDirectory(Object.values(collaborators).map(user=>user.actor_id))
  const collaboratorLabels=Object.fromEntries(Object.values(collaborators).map(user=>[(user.actor_id??'').toLowerCase(),userLabel(user.actor_id??'',collaboratorDirectory)]))
  const updateVersion=useCallback((value:number)=>setServerVersion(current=>Math.max(current,value)),[])
  const handleCollaborationVersion=useCallback((value:number,event:ServerEvent)=>{updateVersion(value);const data=event.data as {structural?:boolean;sheet_id?:string;axis?:'row'|'column';action?:'insert'|'delete'|'move';index?:number;count?:number;destination?:number}|undefined;
    if(data?.structural&&event.client_id!==collaborationClientId()){
      // 셀을 다시 읽어야 하므로 초기화는 필요하다. 하지만 초기화는 선택
      // 위치와 입력 중이던 내용까지 지운다. 다른 사람이 행을 하나 지웠다고
      // 내 화면이 맨 위로 튀고 치던 값이 사라질 이유는 없다.
      const store=useEditorStore.getState()
      const keep={activeRow:store.activeRow,activeColumn:store.activeColumn,anchorRow:store.anchorRow,anchorColumn:store.anchorColumn}
      const sameSheet=!data.sheet_id||data.sheet_id===activeSheetRef.current
      const change=sameSheet&&data.axis&&data.action&&data.index&&data.count
        ?{axis:data.axis,action:data.action,index:data.index,count:data.count,destination:data.destination}
        :undefined
      // 셀은 주소가 밀렸으니 다시 읽어야 하지만, 전체 초기화는 편집을 끝내
      // 버려 치고 있던 값이 사라진다. 들고 있던 셀만 버린다.
      store.clearCells()
      const moved=change?transformSelection(keep,change):keep
      if(moved.activeRow!==keep.activeRow||moved.activeColumn!==keep.activeColumn||moved.anchorRow!==keep.anchorRow||moved.anchorColumn!==keep.anchorColumn){
        const draft=store.draft,editing=store.editing
        store.select(moved.anchorRow,moved.anchorColumn)
        store.select(moved.activeRow,moved.activeColumn,true)
        // 입력 중이던 값은 겨냥한 셀이 살아남았을 때만 되돌린다. 사라진
        // 셀의 자리를 물려받은 셀에 그 값을 쓰면 남의 데이터를 덮어쓴다.
        if(editing&&(!change||survivesChange(keep.activeRow,keep.activeColumn,change))){
          store.carryDraft({row:moved.activeRow,column:moved.activeColumn,text:draft})
          store.setDraft(draft);store.setEditing(true)
        }
      }
    }client.invalidateQueries({queryKey:['workbook',workbookId]});client.invalidateQueries({queryKey:['cell-conflicts',workbookId]});client.invalidateQueries({queryKey:['data-validations']});client.invalidateQueries({queryKey:['conditional-formats']});client.invalidateQueries({queryKey:['named-ranges',workbookId]});client.invalidateQueries({queryKey:['tables',workbookId]});client.invalidateQueries({queryKey:['watch-rules',workbookId]});client.invalidateQueries({queryKey:['charts',workbookId]});client.invalidateQueries({queryKey:['pivots',workbookId]});client.invalidateQueries({queryKey:['pivot-data']});client.invalidateQueries({queryKey:['filter-views']});client.invalidateQueries({queryKey:['filter-result']})},[client,updateVersion,workbookId])
  const handleCollaborationEvent=useCallback((event:ServerEvent)=>{if(event.type==='comment.changed'){client.invalidateQueries({queryKey:['comments',workbookId]});client.invalidateQueries({queryKey:['mention-notifications']})}if(event.type==='operation.conflict'){client.invalidateQueries({queryKey:['cell-conflicts',workbookId]});setRightPanel('conflicts')}},[client,workbookId])
  useEffect(()=>{if(workbook.data){
    setServerVersion(workbook.data.version)
    // A hidden sheet stays out of the editor, so the selection falls back to the
    // first visible sheet.
    setActiveSheet(current=>{
      const same=workbook.data.sheets.find(sheet=>sheet.id===current?.id)
      if(same&&!same.hidden)return same
      return workbook.data.sheets.find(sheet=>!sheet.hidden)??workbook.data.sheets[0]
    })
  }},[workbook.data])
  useEffect(()=>{if(routeNavigationApplied.current||!workbook.data)return;routeNavigationApplied.current=true;const sheet=workbook.data.sheets.find(candidate=>candidate.id===routeNavigation.sheetId);if(!sheet)return;const target=parseNavigationRange(routeNavigation.range);if(target)setPendingNavigation({sheetId:sheet.id,range:target});setActiveSheet(sheet)},[routeNavigation,workbook.data])
  useEffect(()=>{editor.reset();if(pendingNavigation&&pendingNavigation.sheetId===activeSheet?.id){const target=pendingNavigation.range;editor.select(target.startRow,target.startColumn);editor.select(target.endRow,target.endColumn,true);setPendingNavigation(undefined)}},[activeSheet?.id])
  const selectionAddress=editorSelection.startRow===editorSelection.endRow&&editorSelection.startColumn===editorSelection.endColumn?address(editor.activeRow,editor.activeColumn):`${address(editorSelection.startRow,editorSelection.startColumn)}:${address(editorSelection.endRow,editorSelection.endColumn)}`
  // Keep whatever the user is typing: only mirror the selection into the name
  // box when it does not have focus.
  useEffect(()=>{if(document.activeElement!==nameBoxRef.current)setNameBoxValue(selectionAddress)},[selectionAddress])
  useEffect(()=>{activeSheetRef.current=activeSheet?.id},[activeSheet?.id])
  useEffect(()=>{connect(workbookId,handleCollaborationVersion,handleCollaborationEvent);return()=>disconnect()},[connect,disconnect,handleCollaborationEvent,handleCollaborationVersion,workbookId])
  useEffect(()=>{if(activeSheet&&collaborationStatus==='connected'){sendCursor({sheet_id:activeSheet.id,row:editor.activeRow,column:editor.activeColumn});sendSelection({sheet_id:activeSheet.id,start:{row:editorSelection.startRow,column:editorSelection.startColumn},end:{row:editorSelection.endRow,column:editorSelection.endColumn}})}},[activeSheet,collaborationStatus,sendCursor,sendSelection])
  useEffect(()=>{if(editor.conflicts>0)client.invalidateQueries({queryKey:['cell-conflicts',workbookId]})},[client,editor.conflicts,workbookId])
  useEffect(()=>{if(!conflicts.data)return;const current=useEditorStore.getState(),count=conflicts.data.items.length;if((current.saveState==='saved'||current.saveState==='conflict')&&(current.conflicts!==count||current.saveState!==(count>0?'conflict':'saved')))current.setSaveState(count>0?'conflict':'saved',count)},[conflicts.data])
  const nextSheetName=(prefix='Sheet')=>{const used=new Set((workbook.data?.sheets??[]).map(sheet=>sheet.name.toLowerCase()));for(let index=1;;index++){const name=`${prefix}${index}`;if(!used.has(name.toLowerCase()))return name}}
  const filterViews=useQuery({queryKey:['filter-views',activeSheet?.id],queryFn:()=>api<{items:FilterView[]}>(`/api/v1/sheets/${activeSheet!.id}/filter-views`),enabled:Boolean(activeSheet)})
  const activeFilter=filterViews.data?.items.find(view=>view.active)
  const filterResult=useQuery({queryKey:['filter-result',activeFilter?.id,activeFilter?.updated_at,serverVersion],queryFn:()=>api<FilterResult>(`/api/v1/filter-views/${activeFilter!.id}:evaluate`,{method:'POST',body:'{}'}),enabled:Boolean(activeFilter)})
  const refreshFilters=async()=>{await client.invalidateQueries({queryKey:['filter-views',activeSheet?.id]});await client.invalidateQueries({queryKey:['filter-result']})}
  const createFilter=async(input:{name:string;range:string;header_rows:number;criteria:unknown[];active:boolean})=>{const item=await api<FilterView>(`/api/v1/sheets/${activeSheet!.id}/filter-views`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});await refreshFilters();return item}
  const updateFilter=async(id:string,input:Record<string,unknown>)=>{const item=await api<FilterView>(`/api/v1/filter-views/${id}`,{method:'PATCH',body:JSON.stringify(input)});await refreshFilters();return item}
  const deleteFilter=async(id:string)=>{await api(`/api/v1/filter-views/${id}`,{method:'DELETE'});await refreshFilters()}
  const protections=useQuery({queryKey:['protected-ranges',activeSheet?.id,serverVersion],queryFn:()=>api<{items:ProtectedRange[]}>(`/api/v1/sheets/${activeSheet!.id}/protected-ranges`),enabled:Boolean(activeSheet)})
  const refreshProtections=async()=>{await client.invalidateQueries({queryKey:['protected-ranges',activeSheet?.id]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createProtection=async(input:Record<string,unknown>)=>{await api(`/api/v1/sheets/${activeSheet!.id}/protected-ranges`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});await refreshProtections()}
  const deleteProtection=async(rule:ProtectedRange)=>{await api(`/api/v1/protected-ranges/${rule.id}`,{method:'DELETE'});await refreshProtections()}
  const validations=useQuery({queryKey:['data-validations',activeSheet?.id],queryFn:()=>api<{items:DataValidation[]}>(`/api/v1/sheets/${activeSheet!.id}/data-validations`),enabled:Boolean(activeSheet)})
  const refreshValidations=async()=>{await client.invalidateQueries({queryKey:['data-validations',activeSheet?.id]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createValidation=async(input:Record<string,unknown>)=>{const item=await api<DataValidation>(`/api/v1/sheets/${activeSheet!.id}/data-validations`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});updateVersion(item.workbook_version);await refreshValidations();return item}
  const updateValidation=async(id:string,input:Record<string,unknown>)=>{const item=await api<DataValidation>(`/api/v1/data-validations/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshValidations();return item}
  const deleteValidation=async(rule:DataValidation)=>{await api(`/api/v1/data-validations/${rule.id}?expected_revision=${rule.revision}`,{method:'DELETE'});await refreshValidations();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
  const evaluateValidation=(id:string)=>api<ValidationEvaluation>(`/api/v1/data-validations/${id}:evaluate`,{method:'POST',body:'{}'})
  const conditionalFormats=useQuery({queryKey:['conditional-formats',activeSheet?.id],queryFn:()=>api<{items:ConditionalFormat[]}>(`/api/v1/sheets/${activeSheet!.id}/conditional-formats`),enabled:Boolean(activeSheet)})
  const refreshConditionalFormats=async()=>{await client.invalidateQueries({queryKey:['conditional-formats',activeSheet?.id]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createConditionalFormat=async(input:Record<string,unknown>)=>{const idempotencyKey=newIdempotencyKey();const item=await api<ConditionalFormat>(`/api/v1/sheets/${activeSheet!.id}/conditional-formats`,{method:'POST',headers:{'Idempotency-Key':idempotencyKey},body:JSON.stringify({...input,idempotency_key:idempotencyKey})});updateVersion(item.workbook_version);await refreshConditionalFormats();return item}
  const updateConditionalFormat=async(id:string,input:Record<string,unknown>)=>{const item=await api<ConditionalFormat>(`/api/v1/conditional-formats/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshConditionalFormats();return item}
  const deleteConditionalFormat=async(rule:ConditionalFormat)=>{await api(`/api/v1/conditional-formats/${rule.id}?expected_revision=${rule.revision}`,{method:'DELETE'});await refreshConditionalFormats();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
  // 프레젠테이션 기능은 관리자가 켜야 나타난다. 꺼져 있으면 메뉴에 넣지
  // 않는다 — 눌러 봐야 안 된다고 답하는 항목은 없느니만 못하다.
  const presentationConfig=useQuery({queryKey:['presentation-config'],queryFn:()=>api<{enabled:boolean}>('/api/v1/presentation/config'),staleTime:300_000})
  const presentationEnabled=presentationConfig.data?.enabled===true
  // 다른 서비스로 보내기. 허용 목록(handoff.peers)이 비어 있으면 목록이 비고 단추가 없다.
  const handoffTargets=useQuery({queryKey:['handoff-targets'],queryFn:()=>api<{source:string;targets:HandoffTarget[]}>('/api/v1/handoff/targets'),staleTime:300_000})
  // 넘겨받은 워크북이면 어디서 왔는지 제목 아래 한 줄로 말한다. 출처는 바뀌지 않으므로 한 번만 묻는다.
  const handoffOrigin=useQuery({queryKey:['handoff-origin',workbookId],queryFn:()=>fetchHandoffOrigin(workbookId),staleTime:Infinity,retry:false})
  const handoffMenu=(handoffTargets.data?.targets??[]).flatMap(target=>target.formats.map(format=>({kind:'item' as const,label:`${target.name} 으로 ${handoffFormatLabel[format]} 보내기`,onSelect:()=>void sendHandoff(target,handoffResource(workbookId,format,activeSheet?.id),format).catch(error=>alert(error instanceof Error?error.message:'다른 서비스로 보내지 못했습니다.'))})))
  const previewPresentation=useCallback((input:Record<string,unknown>)=>api<{deck:PresentationDeck;analysis:PresentationAnalysis}>(`/api/v1/sheets/${activeSheet!.id}/presentations`,{method:'POST',body:JSON.stringify(input)}),[activeSheet?.id])
  const createPresentation=useCallback(async(input:Record<string,unknown>)=>{
    const made=await api<{presentation:PresentationResult}>(`/api/v1/sheets/${activeSheet!.id}/presentations`,{method:'POST',body:JSON.stringify(input)})
    await client.invalidateQueries({queryKey:['presentations',workbookId]})
    return made
  },[activeSheet?.id,client,workbookId])
  const loadPresentationTemplates=useCallback(async()=>(await api<{items:PresentationTemplate[]}>('/api/v1/presentation/templates')).items,[])
  const downloadPresentation=async(id:string)=>{
    const response=await fetch(`/api/v1/presentations/${id}/export`,{credentials:'same-origin'})
    if(!response.ok)return alert('프레젠테이션을 내려받지 못했습니다.')
    const blob=await response.blob()
    const disposition=response.headers.get('Content-Disposition')||''
    const encoded=disposition.match(/filename\*=UTF-8''([^;]+)/)?.[1]
    const name=encoded?decodeURIComponent(encoded):'presentation.pptx'
    const link=document.createElement('a');link.href=URL.createObjectURL(blob);link.download=name;link.click();URL.revokeObjectURL(link.href)
  }
  const presentationList=useQuery({queryKey:['presentations',workbookId],queryFn:()=>api<{items:PresentationRecord[]}>(`/api/v1/workbooks/${workbookId}/presentations`),enabled:presentationEnabled})
  const refreshPresentationList=()=>client.invalidateQueries({queryKey:['presentations',workbookId]})
  const refreshPresentation=async(record:PresentationRecord)=>{await api(`/api/v1/presentations/${record.id}/refresh`,{method:'POST',body:'{}'});await refreshPresentationList()}
  /**
   * 빠른 이동에 띄울 워크북. 예전에는 limit 없이 불러 전부 받아 놓고
   * 스무 개만 썼다. 워크북이 208개인 곳에서 편집기를 열 때마다 134KB 를
   * 받으면서, 정작 스물한 번째부터는 이름을 적어도 찾히지 않았다.
   *
   * 이제 앞의 스무 개만 받고, 그 밖의 것은 적은 이름으로 서버에 묻는다.
   */
  const workbookList=useQuery({queryKey:['workbooks','quick'],queryFn:()=>api<{items:Workbook[]}>('/api/v1/workbooks?limit=20'),staleTime:60_000})
  const [quickQuery,setQuickQuery]=useState('')
  const workbookSearch=useQuery({
    queryKey:['workbook-quick-search',quickQuery],
    queryFn:()=>api<{items:Workbook[]}>(`/api/v1/workbooks?limit=20&q=${encodeURIComponent(quickQuery)}`),
    enabled:quickOpen&&quickQuery.trim().length>=2,
  })
  const namedRanges=useQuery({queryKey:['named-ranges',workbookId],queryFn:()=>api<{items:NamedRange[]}>(`/api/v1/workbooks/${workbookId}/named-ranges`)})
	const images=useQuery({queryKey:['images',workbookId,activeSheet?.id],queryFn:()=>api<{items:SheetImage[]}>(`/api/v1/workbooks/${workbookId}/images?sheet_id=${activeSheet!.id}`),enabled:Boolean(activeSheet)})
	const imageInput=useRef<HTMLInputElement>(null)
	/** 그림을 올린다. 파일을 고르거나 클립보드에서 붙여넣거나 — 길은 둘, 하는 일은 하나다. */
	const uploadImage=async(file:File)=>{
		if(!activeSheet||!writable())return
		const body=new FormData();body.append('file',file,file.name||'image')
		try{
			await api<SheetImage>(`/api/v1/sheets/${activeSheet.id}/images`,{method:'POST',headers:{'Idempotency-Key':newIdempotencyKey()},body})
			await client.invalidateQueries({queryKey:['images',workbookId]})
		}catch(reason){alert(reason instanceof Error?reason.message:'그림을 넣지 못했습니다.')}
	}
	const updateImage=async(image:SheetImage,input:Record<string,unknown>)=>{const updated=await api<SheetImage>(`/api/v1/images/${image.id}`,{method:'PATCH',body:JSON.stringify(input)});await client.invalidateQueries({queryKey:['images',workbookId]});return updated}
	const deleteImage=async(image:SheetImage)=>{await api<void>(`/api/v1/images/${image.id}?expected_revision=${image.revision}`,{method:'DELETE'});await client.invalidateQueries({queryKey:['images',workbookId]})}
	const charts=useQuery({queryKey:['charts',workbookId,activeSheet?.id],queryFn:()=>api<{items:Chart[]}>(`/api/v1/workbooks/${workbookId}/charts?sheet_id=${activeSheet!.id}`),enabled:Boolean(activeSheet)})
	const pivots=useQuery({queryKey:['pivots',workbookId,activeSheet?.id],queryFn:()=>api<{items:Pivot[]}>(`/api/v1/workbooks/${workbookId}/pivots?sheet_id=${activeSheet!.id}`),enabled:Boolean(activeSheet)})
  // 표가 바뀌면 그 이름을 쓰는 모든 칸의 값이 바뀐다. 워크북까지 함께 새로
  // 읽어야 화면이 서버와 같아진다.
  const refreshSheetTables=async()=>{await client.invalidateQueries({queryKey:['tables',workbookId]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createSheetTable=async(input:Record<string,unknown>)=>{const item=await api<SheetTable>(`/api/v1/workbooks/${workbookId}/tables`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});updateVersion(item.workbook_version);await refreshSheetTables();return item}
  const updateSheetTable=async(id:string,input:Record<string,unknown>)=>{const item=await api<SheetTable>(`/api/v1/tables/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshSheetTables();return item}
  const deleteSheetTable=async(item:SheetTable)=>{await api(`/api/v1/tables/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshSheetTables();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
  const refreshNamedRanges=async()=>{await client.invalidateQueries({queryKey:['named-ranges',workbookId]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createNamedRange=async(input:Record<string,unknown>)=>{const item=await api<NamedRange>(`/api/v1/workbooks/${workbookId}/named-ranges`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});updateVersion(item.workbook_version);await refreshNamedRanges();return item}
  const updateNamedRange=async(id:string,input:Record<string,unknown>)=>{const item=await api<NamedRange>(`/api/v1/named-ranges/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshNamedRanges();return item}
  const deleteNamedRange=async(item:NamedRange)=>{await api(`/api/v1/named-ranges/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshNamedRanges();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
  // 이름 있는 수식이 바뀌면 그것을 쓰는 모든 칸의 값이 바뀐다. 워크북까지
  // 함께 새로 읽어야 화면이 서버와 같아진다.
  const refreshNamedFunctions=async()=>{await client.invalidateQueries({queryKey:['named-functions',workbookId]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const createNamedFunction=async(input:Record<string,unknown>)=>{const item=await api<NamedFunction>(`/api/v1/workbooks/${workbookId}/named-functions`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});updateVersion(item.workbook_version);await refreshNamedFunctions();return item}
  const updateNamedFunction=async(id:string,input:Record<string,unknown>)=>{const item=await api<NamedFunction>(`/api/v1/named-functions/${id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(item.workbook_version);await refreshNamedFunctions();return item}
  const deleteNamedFunction=async(item:NamedFunction)=>{await api(`/api/v1/named-functions/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshNamedFunctions();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
  const watchRules=useQuery({queryKey:['watch-rules',workbookId],queryFn:()=>api<{items:WatchRuleType[]}>(`/api/v1/workbooks/${workbookId}/watch-rules`)})
  const refreshWatchRules=async()=>{await client.invalidateQueries({queryKey:['watch-rules',workbookId]})}
  const createWatchRule=async(input:Record<string,unknown>)=>{const item=await api<WatchRuleType>(`/api/v1/workbooks/${workbookId}/watch-rules`,{method:'POST',body:JSON.stringify({...input,idempotency_key:newIdempotencyKey()})});await refreshWatchRules();return item}
  const updateWatchRule=async(id:string,input:Record<string,unknown>)=>{const item=await api<WatchRuleType>(`/api/v1/watch-rules/${id}`,{method:'PATCH',body:JSON.stringify(input)});await refreshWatchRules();return item}
  const deleteWatchRule=async(item:WatchRuleType)=>{await api(`/api/v1/watch-rules/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshWatchRules()}
	const refreshCharts=async()=>{await client.invalidateQueries({queryKey:['charts',workbookId]});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
	const createChart=async(input:Record<string,unknown>)=>{const idempotencyKey=newIdempotencyKey();const item=await api<Chart>(`/api/v1/workbooks/${workbookId}/charts`,{method:'POST',headers:{'Idempotency-Key':idempotencyKey},body:JSON.stringify({...input,idempotency_key:idempotencyKey})});updateVersion(item.workbook_version);await refreshCharts();return item}
	const updateChart=async(item:Chart,input:Record<string,unknown>)=>{const updated=await api<Chart>(`/api/v1/charts/${item.id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(updated.workbook_version);await refreshCharts();return updated}
	const deleteChart=async(item:Chart)=>{await api(`/api/v1/charts/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshCharts();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
	const refreshPivots=async()=>{await client.invalidateQueries({queryKey:['pivots',workbookId]});await client.invalidateQueries({queryKey:['pivot-data']});await client.invalidateQueries({queryKey:['workbook',workbookId]})}
	const createPivot=async(input:Record<string,unknown>)=>{const idempotencyKey=newIdempotencyKey();const item=await api<Pivot>(`/api/v1/workbooks/${workbookId}/pivots`,{method:'POST',headers:{'Idempotency-Key':idempotencyKey},body:JSON.stringify({...input,idempotency_key:idempotencyKey})});updateVersion(item.workbook_version);await refreshPivots();return item}
	const updatePivot=async(item:Pivot,input:Record<string,unknown>)=>{const updated=await api<Pivot>(`/api/v1/pivots/${item.id}`,{method:'PATCH',body:JSON.stringify(input)});updateVersion(updated.workbook_version);await refreshPivots();return updated}
	const deletePivot=async(item:Pivot)=>{await api(`/api/v1/pivots/${item.id}?expected_revision=${item.revision}`,{method:'DELETE'});await refreshPivots();const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`);updateVersion(latest.version)}
	const refreshPivot=async(item:Pivot)=>{await api<PivotData>(`/api/v1/pivots/${item.id}/refresh`,{method:'POST',body:'{}'});await refreshPivots()}
  const navigateToRange=(sheetId:string,value:string)=>{
    const target=parseNavigationRange(value),sheet=workbook.data?.sheets.find(candidate=>candidate.id===sheetId)
    if(!target||!sheet)return false
    if(activeSheet?.id===sheetId){editor.select(target.startRow,target.startColumn);editor.select(target.endRow,target.endColumn,true)}
    else{setPendingNavigation({sheetId,range:target});setActiveSheet(sheet)}
    // Jumping should leave the keyboard on the grid, unless a dialog is still
    // open and owns the focus.
    if(!document.querySelector('[role="dialog"][aria-modal="true"]'))window.setTimeout(()=>gridShortcut({command:'focus-grid'}),0)
    return true
  }
  // After a successful jump the name box shows the resolved range and hands
  // focus back to the grid, so the typed name is never left stale.
  const submitNameBox=()=>{
    const value=nameBoxValue.trim(),named=(namedRanges.data?.items??[]).find(item=>item.name.toLowerCase()===value.toLowerCase())
    // 표 이름도 받는다. 이름 범위처럼 이름으로 가리키는 것이므로, 한쪽만
    // 되면 사람은 왜 되고 안 되는지 알 수 없다.
    const table=named?undefined:(sheetTables.data?.items??[]).find(item=>item.name.toLowerCase()===value.toLowerCase())
    const jump=named??table
    const target=jump?{sheetId:jump.sheet_id,reference:jump.range}:activeSheet?{sheetId:activeSheet.id,reference:value}:undefined
    const parsed=target?parseNavigationRange(target.reference):undefined
    if(!target||!parsed||!navigateToRange(target.sheetId,target.reference)){setNameBoxValue(selectionAddress);return}
    setNameBoxValue(parsed.startRow===parsed.endRow&&parsed.startColumn===parsed.endColumn?address(parsed.startRow,parsed.startColumn):`${address(parsed.startRow,parsed.startColumn)}:${address(parsed.endRow,parsed.endColumn)}`)
    nameBoxRef.current?.blur()
    gridShortcut({command:'focus-grid'})
  }
  const refreshWorkbook=async()=>client.invalidateQueries({queryKey:['workbook',workbookId]})
  const createSheet=async()=>{const sheet=await api<Sheet>(`/api/v1/workbooks/${workbookId}/sheets`,{method:'POST',body:JSON.stringify({name:nextSheetName()})});setActiveSheet(sheet);await refreshWorkbook()}
  const updateSheet=async(sheet:Sheet,input:Record<string,unknown>)=>{const updated=await api<Sheet>(`/api/v1/sheets/${sheet.id}`,{method:'PATCH',body:JSON.stringify(input)});if(activeSheet?.id===sheet.id)setActiveSheet(updated);await refreshWorkbook()}
  const setSheetHidden=async(sheet:Sheet,hidden:boolean)=>{
    await updateSheet(sheet,{hidden})
    if(hidden&&activeSheet?.id===sheet.id){
      const next=(workbook.data?.sheets??[]).find(candidate=>candidate.id!==sheet.id&&!candidate.hidden)
      if(next)setActiveSheet(next)
    }
  }
  const duplicateSheet=async(sheet:Sheet)=>{const duplicated=await api<Sheet>(`/api/v1/sheets/${sheet.id}/duplicate`,{method:'POST',body:'{}'});setActiveSheet(duplicated);await refreshWorkbook()}
  const deleteSheet=async(sheet:Sheet)=>{
    const ordered=workbook.data!.sheets;const index=ordered.findIndex(item=>item.id===sheet.id);const fallback=ordered[index===0?1:index-1]
    // 시트 삭제는 그 안의 모든 셀을 버리고 셀 단위로 되돌릴 수 없다. 서버가
    // 삭제 직전에 남긴 스냅숏이 유일한 회수 경로라서 그 번호를 붙들어 둔다.
    const deletion=await api<{backup_version_id?:string;sheet_name?:string}>(`/api/v1/sheets/${sheet.id}`,{method:'DELETE'})
    if(activeSheet?.id===sheet.id&&fallback)setActiveSheet(fallback)
    await refreshWorkbook()
    if(deletion?.backup_version_id)editor.reportRecoverableEdit({versionId:deletion.backup_version_id,summary:`시트 ${deletion.sheet_name||sheet.name}`})
  }
  const revertOperation=async(mode:'undo'|'redo')=>{if(!writable())return;if(!navigator.onLine){alert('Undo와 Redo는 서버에 다시 연결한 후 사용할 수 있습니다.');return}const target=mode==='undo'?editor.takeUndo():editor.takeRedo();if(!target)return;editor.setSaveState('saving');try{const result=await api<MutationResult>(`/api/v1/operations/${target}:undo`,{method:'POST',body:JSON.stringify({idempotency_key:`undo:${target}`,client_id:collaborationClientId()})});updateVersion(result.server_version);if(result.applied_cells>0){if(mode==='undo')editor.completeUndo(result.operation_id);else editor.completeRedo(result.operation_id)}else{if(mode==='undo')editor.restoreUndo(target);else editor.restoreRedo(target)}editor.setSaveState(result.conflicts.length?'conflict':'saved',result.conflicts.length)}catch{if(mode==='undo')editor.restoreUndo(target);else editor.restoreRedo(target);editor.setSaveState('error')}}
  const denyWrite=()=>{editor.setSaveState('error');alert('보기 전용 권한입니다. 소유자에게 편집 권한을 요청하세요.')}
  const writable=()=>{const role=workbook.data?.access_role??'owner';if(role==='editor'||role==='owner')return true;denyWrite();return false}
  // 큐가 포기한 변경은 화면에 나와야 한다. 멈춘 것을 말하지 않으면 사람은
  // 저장된 줄 알고 창을 닫는다.
  const [stalled,setStalled]=useState<OutboxOperation[]>([])
  const [pending,setPending]=useState(0)
  const refreshStalled=useCallback(async()=>{
    const own=pendingInWorkbook(await listOutbox(),(workbook.data?.sheets??[]).map(sheet=>sheet.id))
    setPending(own.length);setStalled(blocked(own))
  },[workbook.data])
  useEffect(()=>{void refreshStalled();const notice=()=>void refreshStalled();window.addEventListener('kanpic:outbox-blocked',notice)
    const timer=window.setInterval(()=>void refreshStalled(),3000)
    return()=>{window.removeEventListener('kanpic:outbox-blocked',notice);window.clearInterval(timer)}},[refreshStalled])
  // 밀린 편집을 두고 창을 닫으면, 그 워크북을 다시 열기 전까지는 서버로 나가지 못한다.
  // 곧 나갈 것까지 붙잡지는 않는다. 연결이 끊겼거나 이미 멈춘 것이 있을 때만 묻는다.
  useEffect(()=>{
    if(pending===0||(navigator.onLine&&stalled.length===0))return
    const ask=(event:BeforeUnloadEvent)=>{event.preventDefault();event.returnValue=''}
    window.addEventListener('beforeunload',ask);return()=>window.removeEventListener('beforeunload',ask)
  },[pending,stalled.length])
  // 저장 큐는 다른 워크북에서 밀린 작업까지 함께 보낸다. 그 응답을 이 시트에 얹으면
  // 남의 서버 버전과 작업 번호가 섞이므로, 화면에 반영하는 것은 이 시트의 결과뿐이다.
  const outboxApplied=(sheetId:string|undefined)=>forSheet<MutationResult>(sheetId,applied=>{updateVersion(applied.server_version);editor.reportEdit(applied);if(!applied.duplicate&&applied.applied_cells>0)editor.recordOperation(applied.operation_id);editor.setSaveState(applied.conflicts?.length?'conflict':'saved',applied.conflicts?.length||0)})
  const outboxFlushed=(sheetId:string|undefined)=>forSheet<MutationResult>(sheetId,applied=>{updateVersion(applied.server_version);editor.reportEdit(applied)})
  // 막힌 작업이 남의 워크북 것이라면 여기의 행·열 변경까지 막을 이유가 없다.
  const workbookPending=async()=>pendingInWorkbook(await listOutbox(),(workbook.data?.sheets??[]).map(sheet=>sheet.id))
  const applyFormat=async(patch:Record<string,unknown>,border?:BorderFormatCommand)=>{if(!activeSheet||!writable())return;const rows=editorSelection.endRow-editorSelection.startRow+1,columns=editorSelection.endColumn-editorSelection.startColumn+1;if(rows*columns>MAX_PASTE_CELLS){alert(`서식 적용은 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`);return}const updatedAt=new Date().toISOString(),optimistic:Cell[]=[];for(let row=editorSelection.startRow;row<=editorSelection.endRow;row+=1)for(let column=editorSelection.startColumn;column<=editorSelection.endColumn;column+=1){const current=editor.cells.get(cellKey(row,column));optimistic.push({sheet_id:activeSheet.id,row,column,value:current?.value,formula:current?.formula,spill_source:current?.spill_source,style:patchStyle(current?.style,patch),updated_at:updatedAt})}editor.putCells(optimistic);editor.setSaveState(navigator.onLine?'saving':'offline');const id=newIdempotencyKey();await enqueue({id,sheetId:activeSheet.id,workbookId,endpoint:'format',attempts:0,createdAt:Date.now(),body:{base_version:serverVersion,idempotency_key:id,client_id:collaborationClientId(),range:`${address(editorSelection.startRow,editorSelection.startColumn)}:${address(editorSelection.endRow,editorSelection.endColumn)}`,...(Object.keys(patch).length>0?{style:patch}:{}),...(border?{border}:{})}});await flushOutbox(outboxApplied(activeSheet?.id))}
  const changeMerge=async(merge:boolean)=>{if(!activeSheet||!writable())return;const rows=editorSelection.endRow-editorSelection.startRow+1,columns=editorSelection.endColumn-editorSelection.startColumn+1;if(merge&&rows*columns<2){alert('두 개 이상의 셀을 선택해 병합하세요.');return}if(rows*columns>MAX_PASTE_CELLS){alert(`셀 병합은 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 가능합니다.`);return}const spill=spillInRange(editor.cells,editorSelection);if(spill){alert(`${spill.coordinate}은(는) ${spill.cell.spill_source} 배열 수식의 결과이므로 병합할 수 없습니다.`);return}if(merge){for(let row=editorSelection.startRow;row<=editorSelection.endRow;row+=1)for(let column=editorSelection.startColumn;column<=editorSelection.endColumn;column+=1)if(cellMerge(editor.cells.get(cellKey(row,column)))){alert('선택 범위가 기존 병합 셀과 겹칩니다. 먼저 병합을 해제하세요.');return}}const updatedAt=new Date().toISOString(),optimistic:Cell[]=[];for(let row=editorSelection.startRow;row<=editorSelection.endRow;row+=1)for(let column=editorSelection.startColumn;column<=editorSelection.endColumn;column+=1){const current=editor.cells.get(cellKey(row,column));optimistic.push({sheet_id:activeSheet.id,row,column,value:current?.value,formula:current?.formula,spill_source:current?.spill_source,style:applyMergeStyle(current?.style,editorSelection,merge),updated_at:updatedAt})}editor.putCells(optimistic);editor.setSaveState(navigator.onLine?'saving':'offline');const id=newIdempotencyKey(),endpoint=merge?'merge':'unmerge';await enqueue({id,sheetId:activeSheet.id,workbookId,endpoint,attempts:0,createdAt:Date.now(),body:{base_version:serverVersion,idempotency_key:id,client_id:collaborationClientId(),range:`${address(editorSelection.startRow,editorSelection.startColumn)}:${address(editorSelection.endRow,editorSelection.endColumn)}`}});await flushOutbox(outboxApplied(activeSheet?.id))}
  const sortSelection=async(options:SortOptions,target?:{startRow:number;startColumn:number;endRow:number;endColumn:number})=>{if(!activeSheet||!writable())return;const editorSelection=target??selectedBounds(useEditorStore.getState());const spill=spillInRange(editor.cells,editorSelection);if(spill){const error=new Error(`${spill.coordinate}은(는) ${spill.cell.spill_source} 배열 수식의 결과이므로 정렬할 수 없습니다.`);alert(error.message);throw error}let optimistic:Cell[];try{optimistic=materializeSort(editor.cells,editorSelection,options,activeSheet.id)}catch(error){alert(error instanceof Error?error.message:'범위를 정렬하지 못했습니다.');throw error}editor.putCells(optimistic);editor.setSaveState(navigator.onLine?'saving':'offline');const id=newIdempotencyKey();await enqueue({id,sheetId:activeSheet.id,workbookId,endpoint:'sort',attempts:0,createdAt:Date.now(),body:{base_version:serverVersion,idempotency_key:id,client_id:collaborationClientId(),range:`${address(editorSelection.startRow,editorSelection.startColumn)}:${address(editorSelection.endRow,editorSelection.endColumn)}`,keys:options.keys,header_rows:options.headerRows,case_sensitive:options.caseSensitive,literal_order:options.literalOrder===true}});await flushOutbox(outboxApplied(activeSheet?.id))}
  const applyStructure=async(command:StructureCommand)=>{if(!activeSheet||!writable())return;if(!navigator.onLine){alert('행과 열 구조 변경은 서버에 연결된 상태에서만 사용할 수 있습니다.');throw new Error('offline')}editor.setSaveState('saving');try{await flushOutbox(outboxFlushed(activeSheet?.id));if((await workbookPending()).length>0)throw new Error('저장 대기 중인 변경을 먼저 서버에 반영해야 합니다.');const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`),idempotencyKey=newIdempotencyKey();const result=await api<MutationResult>(`/api/v1/sheets/${activeSheet.id}/structure:apply`,{method:'PATCH',headers:{'Idempotency-Key':idempotencyKey},body:JSON.stringify({...command,base_version:latest.version,idempotency_key:idempotencyKey,client_id:collaborationClientId()})});const landed=command.action==='move'&&command.destination!==undefined?command.destination>command.index?command.destination-command.count:command.destination:command.index;const targetRow=Math.max(1,command.axis==='row'?landed:editorSelection.startRow),targetColumn=Math.max(1,command.axis==='column'?landed:editorSelection.startColumn);editor.reset();editor.select(targetRow,targetColumn);updateVersion(result.server_version);editor.reportRecoverableEdit(command.action==='delete'&&result.backup_version_id?{versionId:result.backup_version_id,summary:structureSummary(command)}:undefined,result.formula_errors);editor.setSaveState('saved');await Promise.all([client.invalidateQueries({queryKey:['workbook',workbookId]}),client.invalidateQueries({queryKey:['data-validations']}),client.invalidateQueries({queryKey:['conditional-formats']}),client.invalidateQueries({queryKey:['named-ranges',workbookId]}),client.invalidateQueries({queryKey:['tables',workbookId]}),client.invalidateQueries({queryKey:['watch-rules',workbookId]}),client.invalidateQueries({queryKey:['charts',workbookId]}),client.invalidateQueries({queryKey:['pivots',workbookId]}),client.invalidateQueries({queryKey:['pivot-data']}),client.invalidateQueries({queryKey:['filter-views']}),client.invalidateQueries({queryKey:['filter-result']})])}catch(error){editor.setSaveState('error');const message=error instanceof Error?error.message:'행 또는 열을 변경하지 못했습니다.';if(message!=='offline')alert(message);throw error}}
  const applySheetLayout=async(command:LayoutCommand)=>{if(!activeSheet||!writable())return;if(!navigator.onLine){alert('시트 레이아웃은 서버에 연결된 상태에서 변경할 수 있습니다.');throw new Error('offline')}editor.setSaveState('saving');try{await flushOutbox(outboxFlushed(activeSheet?.id));if((await workbookPending()).length>0)throw new Error('저장 대기 중인 변경을 먼저 서버에 반영해야 합니다.');const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`),sheet=latest.sheets.find(item=>item.id===activeSheet.id);if(!sheet)throw new Error('시트를 찾을 수 없습니다.');const idempotencyKey=newIdempotencyKey();const result=await api<SheetLayoutResult>(`/api/v1/sheets/${activeSheet.id}/layout:apply`,{method:'PATCH',headers:{'Idempotency-Key':idempotencyKey},body:JSON.stringify({...command,expected_revision:sheet.layout?.revision??1,idempotency_key:idempotencyKey,client_id:collaborationClientId()})});setActiveSheet(current=>current?.id===result.sheet_id?{...current,layout:result.layout}:current);updateVersion(result.server_version);editor.setSaveState('saved');await client.invalidateQueries({queryKey:['workbook',workbookId]})}catch(error){editor.setSaveState('error');const message=error instanceof Error?error.message:'시트 레이아웃을 변경하지 못했습니다.';if(message!=='offline')alert(message);throw error}}
  // Header drags, auto-fit and menu commands can fire back to back, so layout
  // mutations run one at a time and never race on the sheet layout revision.
  /**
   * 인쇄 영역은 "이 범위만 종이에 낸다" 는 시트의 성질이다. 정해 두지
   * 않으면 내용이 있는 곳 전체가 나간다.
   *
   * 엑셀 파일에서 가져올 때도 함께 따라온다. 원래 문서가 표 한 덩어리만
   * 내도록 짜여 있었다면 그 뜻이 지켜져야 한다.
   */
  const setPrintArea=async()=>{
    if(!activeSheet)return
    const bounds=selectedBounds(useEditorStore.getState())
    const range=`${address(bounds.startRow,bounds.startColumn)}:${address(bounds.endRow,bounds.endColumn)}`
    await applyLayout({action:'print_area_set',range})
  }
  const clearPrintArea=async()=>{
    if(!activeSheet?.layout?.print_area)return
    await applyLayout({action:'print_area_clear'})
  }
  const applyLayout=(command:LayoutCommand)=>{
    const next=layoutQueue.current.catch(()=>{}).then(()=>applySheetLayout(command))
    layoutQueue.current=next
    return next
  }
  // A slicer is a filter control stored with the sheet layout. Adding one
  // needs a filter view to drive, so it offers to create one over the selected
  // range rather than failing on a sheet that has no filter yet.
  const addSlicer=async()=>{
    if(!activeSheet||!writable())return
    let view=activeFilter
    if(!view){
      const working=await resolveWorkingBlock()
      if(!working)return alert('슬라이서를 만들 데이터 범위를 찾지 못했습니다.')
      const bounds=working.region
      if(!window.confirm('이 시트에는 필터 보기가 없습니다. 선택한 데이터 범위로 필터 보기를 만들고 슬라이서를 추가할까요?'))return
      const idempotencyKey=newIdempotencyKey()
      view=await api<FilterView>(`/api/v1/sheets/${activeSheet.id}/filter-views`,{method:'POST',body:JSON.stringify({idempotency_key:idempotencyKey,name:'필터 보기',range:`${address(bounds.startRow,bounds.startColumn)}:${address(bounds.endRow,bounds.endColumn)}`,header_rows:1,active:true})})
      await refreshFilters()
    }
    const column=editor.activeColumn
    const bounds=parseFilterRange(view.range)
    if(!bounds||column<bounds.startColumn||column>bounds.endColumn)return alert('필터 범위 안의 열을 선택한 뒤 슬라이서를 추가해 주세요.')
    const header=bounds.startRow<=bounds.endRow?editor.cells.get(cellKey(bounds.startRow,column))?.value:undefined
    const count=activeSheet.layout?.slicers?.length??0
    await applyLayout({action:'slicer_add',slicer:{filter_view_id:view.id,column,title:typeof header==='string'&&header.trim()?header.trim().slice(0,64):undefined,position:{x:24+count*24,y:24+count*24,width:220,height:260}}})
  }
  const updateSlicer=async(slicer:Slicer)=>{await applyLayout({action:'slicer_update',slicer})}
  const removeSlicer=async(slicer:Slicer)=>{await applyLayout({action:'slicer_remove',slicer:{id:slicer.id}})}
  const exportWorkbook=async(format:'xlsx'|'csv')=>{const response=await fetch('/api/v1/exports',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({workbook_id:workbookId,sheet_id:activeSheet?.id,format})});if(!response.ok)return alert('파일을 내보내지 못했습니다.');const skipped=Number(response.headers.get('X-Kanpic-Skipped-Charts')||0),skippedImages=Number(response.headers.get('X-Kanpic-Skipped-Images')||0);const blob=await response.blob();const disposition=response.headers.get('Content-Disposition')||'';const encoded=disposition.match(/filename\*=UTF-8''([^;]+)/)?.[1];const basic=disposition.match(/filename="?([^";]+)"?/)?.[1];const name=encoded?decodeURIComponent(encoded):basic||`kanpic.${format}`;const link=document.createElement('a');link.href=URL.createObjectURL(blob);link.download=name;link.click();URL.revokeObjectURL(link.href);const left=[skipped>0?`일정표 등 엑셀에 같은 그림이 없는 차트 ${skipped}개`:'',skippedImages>0?`그림 ${skippedImages}장`:''].filter(Boolean);if(left.length>0)alert(`${left.join('와 ')}은(는) 파일에 담기지 않았습니다. 시트의 자료는 그대로 들어 있습니다.`)}
  // A refresh recalculates every formula on the server, so the editor takes
  // the same route it takes after a version restore: drop what is loaded and
  // read the new values back.
  const handleConnectionsRefreshed=async(result:WorkbookConnections)=>{
    if(!result.version)return
    editor.reset()
    updateVersion(result.version)
    await client.invalidateQueries({queryKey:['workbook',workbookId]})
  }
  const handleRestored=async(result:MutationResult)=>{editor.reset();updateVersion(result.server_version);await Promise.all([client.invalidateQueries({queryKey:['workbook',workbookId]}),client.invalidateQueries({queryKey:['conditional-formats']}),client.invalidateQueries({queryKey:['data-validations']}),client.invalidateQueries({queryKey:['named-ranges',workbookId]}),client.invalidateQueries({queryKey:['charts',workbookId]}),client.invalidateQueries({queryKey:['pivots',workbookId]})])}
  const handleConflictResolved=(result:CellConflictResolutionResult)=>{updateVersion(result.operation.server_version);if(!result.operation.duplicate&&result.operation.applied_cells>0)editor.recordOperation(result.operation.operation_id);editor.setSaveState('saved');client.invalidateQueries({queryKey:['workbook',workbookId]})}
  const handleAIExecuted=(result:AIExecutionResult)=>{updateVersion(result.operation.server_version);editor.reset();editor.setSaveState('saved');client.invalidateQueries({queryKey:['workbook',workbookId]});client.invalidateQueries({queryKey:['ai-actions',workbookId]});client.invalidateQueries({queryKey:['agent-runs',workbookId]})}
  const prepareAutomationExecution=async()=>{
    if(!navigator.onLine)throw new Error('자동화 검증과 실행은 서버에 연결된 상태에서만 사용할 수 있습니다.')
    if(useEditorStore.getState().editing)gridShortcut({command:'commit-draft'})
    if(useEditorStore.getState().editing)throw new Error('편집 중인 셀 입력을 먼저 확정하세요.')
    const deadline=Date.now()+15_000
    while(useEditorStore.getState().saveState==='saving'){
      if(Date.now()>=deadline)throw new Error('셀 변경 저장이 완료되지 않았습니다. 저장 상태를 확인한 뒤 다시 시도하세요.')
      await new Promise(resolve=>window.setTimeout(resolve,50))
    }
    if(useEditorStore.getState().saveState==='error')throw new Error('저장 오류가 있는 셀 변경을 먼저 해결하세요.')
    editor.setSaveState('saving')
    try{
      await flushOutbox(outboxApplied(activeSheet?.id))
      if((await workbookPending()).length>0)throw new Error('저장 대기 중인 변경을 서버에 반영하지 못했습니다. 연결 또는 편집 충돌을 확인하세요.')
      const current=useEditorStore.getState()
		if(current.conflicts>0)throw new Error('셀 편집 충돌을 먼저 해결한 뒤 자동화를 검증하세요.')
      current.setSaveState(current.conflicts?'conflict':'saved',current.conflicts)
		const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`)
		updateVersion(latest.version)
		return latest.version
	}catch(reason){const current=useEditorStore.getState();current.setSaveState(current.conflicts?'conflict':'error',current.conflicts);throw reason}
  }
  const handleAutomationExecuted=(result:AutomationExecutionResult)=>{updateVersion(result.operation.server_version);editor.setSaveState('saved');client.invalidateQueries({queryKey:['workbook',workbookId]});client.invalidateQueries({queryKey:['automations',workbookId]})}
  /**
   * Writes the hover note on the selection. The note lives on the cell, so it
   * is sent on its own path that leaves values and formatting untouched.
   */
  const applyNote=async(note:string)=>{
    if(!activeSheet||!writable())return
    const updatedAt=new Date().toISOString(),optimistic:Cell[]=[]
    for(let row=editorSelection.startRow;row<=editorSelection.endRow;row+=1)for(let column=editorSelection.startColumn;column<=editorSelection.endColumn;column+=1){
      const current=editor.cells.get(cellKey(row,column))
      optimistic.push({sheet_id:activeSheet.id,row,column,value:current?.value,formula:current?.formula,spill_source:current?.spill_source,style:current?.style,note:note||undefined,updated_at:updatedAt})
    }
    editor.putCells(optimistic)
    editor.setSaveState(navigator.onLine?'saving':'offline')
    const id=newIdempotencyKey()
    await enqueue({id,sheetId:activeSheet.id,workbookId,endpoint:'note',attempts:0,createdAt:Date.now(),body:{base_version:serverVersion,idempotency_key:id,client_id:collaborationClientId(),range:`${address(editorSelection.startRow,editorSelection.startColumn)}:${address(editorSelection.endRow,editorSelection.endColumn)}`,note}})
    await flushOutbox(outboxApplied(activeSheet?.id))
  }
  /**
   * Picks up the active cell's format. A double click keeps the brush loaded
   * so a format can be painted onto several ranges in a row.
   */
  // The shortcut handler is registered once, so it reads the brush by ref.
  const formatBrushRef=useRef<{style:Record<string,unknown>;sticky:boolean}|undefined>(undefined)
  formatBrushRef.current=formatBrush
  const pickUpFormat=(sticky:boolean)=>{
    if(!canWrite)return
    setFormatBrush({style:activeCell?.style??{},sticky})
  }
  const paintFormat=async(range:{startRow:number;startColumn:number;endRow:number;endColumn:number})=>{
    if(!formatBrush||!activeSheet)return
    const patch=brushPatch(formatBrush.style)
    if(!formatBrush.sticky)setFormatBrush(undefined)
    editor.select(range.startRow,range.startColumn)
    editor.select(range.endRow,range.endColumn,true)
    await applyFormat(patch)
  }
  const clearFormat=()=>applyFormat(Object.fromEntries(CLEARABLE_STYLE_KEYS.map(key=>[key,null])))
  const hideSelection=(axis:'row'|'column')=>applyLayout(axis==='row'
    ?{action:'hide',axis,start:editorSelection.startRow,count:editorSelection.endRow-editorSelection.startRow+1}
    :{action:'hide',axis,start:editorSelection.startColumn,count:editorSelection.endColumn-editorSelection.startColumn+1})
  const freezeToSelection=()=>applyLayout({action:'freeze',frozen_rows:Math.max(0,Math.min(100,editorSelection.startRow-1)),frozen_columns:Math.max(0,Math.min(50,editorSelection.startColumn-1))})
  // Whole-row and whole-column selections insert or delete directly; anything
  // else opens the dialog so the axis is chosen explicitly.
  const changeStructure=async(action:'insert'|'delete')=>{
    const wholeRows=editorSelection.startColumn<=1&&editorSelection.endColumn>=MAX_GRID_COLUMNS
    const wholeColumns=editorSelection.startRow<=1&&editorSelection.endRow>=MAX_GRID_ROWS
    if(!wholeRows&&!wholeColumns){setStructureOpen(true);return}
    const axis=wholeRows?'row':'column'
    const index=axis==='row'?editorSelection.startRow:editorSelection.startColumn
    const count=axis==='row'?editorSelection.endRow-editorSelection.startRow+1:editorSelection.endColumn-editorSelection.startColumn+1
    if(action==='delete'&&!window.confirm(`선택한 ${count}개 ${axis==='row'?'행':'열'}을 삭제할까요? 삭제 전 복구 버전이 자동 생성됩니다.`))return
    await applyStructure({axis,action,index,count})
  }
  const moveSheet=(delta:number)=>{const sheets=workbook.data?.sheets??[];if(sheets.length<2||!activeSheet)return;const index=sheets.findIndex(sheet=>sheet.id===activeSheet.id);setActiveSheet(sheets[(index+delta+sheets.length)%sheets.length])}
  const openSearch=(replace:boolean)=>{setReplaceMode(replace);setSearchOpen(true)}
  const copySelectionLink=async()=>{
    const parameters=new URLSearchParams({sheet_id:activeSheet?.id??'',range:selectionAddress})
    const link=`${window.location.origin}/workbooks/${workbookId}?${parameters.toString()}`
    try{await navigator.clipboard?.writeText(link)}catch{window.prompt('셀 링크를 복사하세요.',link)}
  }
  const handleReplaced=async(result:ReplaceResult)=>{updateVersion(result.server_version);editor.reset();editor.setSaveState('saved');await client.invalidateQueries({queryKey:['workbook',workbookId]})}
  // Sorting a whole column keeps the surrounding data block together, so the
  // computed region is confirmed before the existing range sort runs.
  // The block a sort or a cleanup acts on is decided from what the server says
  // the sheet holds, never from the rows the grid happens to have loaded: a
  // window-sized region sorts the visible rows and leaves the rest behind.
  const resolveSheetBlock=async(seedRow:number,seedColumn:number)=>{
    if(!activeSheet)return undefined
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
    if(!used||used.max_row<1)return undefined
    const label=`A1:${address(Math.min(used.max_row,MAX_PRINT_ROWS),Math.min(Math.max(used.max_column,1),MAX_GRID_COLUMNS))}`
    const read=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
    if(!read)return undefined
    const cells=new Map(read.items.map(cell=>[cellKey(cell.row,cell.column),cell]))
    const region=dataRegion(cells,seedRow,seedColumn,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS})
    return {region,cells}
  }
  // The first row of the run of numbers that ends just above a cell, read from
  // the server so a long column sums all of itself rather than the part the
  // grid happens to hold.
  const resolveNumericRun=async(row:number,column:number)=>{
    if(!activeSheet||row<2)return undefined
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
    if(!used||used.max_row<1)return undefined
    const label=`${address(1,column)}:${address(Math.min(Math.max(used.max_row,row),MAX_PRINT_ROWS),column)}`
    const read=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
    if(!read)return undefined
    const numbers=new Set(read.items.filter(cell=>typeof cell.value==='number').map(cell=>cell.row))
    let first=row
    while(first-1>=1&&numbers.has(first-1))first-=1
    return first===row?undefined:first
  }
  // Subtotals rewrite the block and push whatever sits under it down, so the
  // rows below are read for the warning the same way the split dialog reads
  // the columns to its right.
  const openSubtotal=async()=>{
    if(!writable())return
    const seed=await resolveWorkingBlock()
    const resolved=await resolveSheetBlock(seed.region.startRow,seed.region.startColumn)
    const region=resolved?.region??seed.region
    const cells=resolved?.cells??seed.cells
    let occupiedBelow=0
    for(let row=region.endRow+1;row<=region.endRow+40;row+=1){
      let filled=false
      for(let column=region.startColumn;column<=region.endColumn&&!filled;column+=1)
        if(cleanupText(cells.get(cellKey(row,column)))!=='')filled=true
      if(filled)occupiedBelow+=1
    }
    setSubtotal({region,cells,headerRows:looksLikeHeaderRow(cells,region)?1:0,occupiedBelow})
  }
  const applySubtotals=async(plan:SubtotalPlan)=>{
    await writeCells(plan.writes)
    // Each group folds away behind its own subtotal row, which is what makes a
    // long report readable once the totals are in place.
    for(const group of plan.groups){
      if(group.end<=group.start)continue
      await applyLayout({action:'group',axis:'row',start:group.start,count:group.end-group.start+1})
    }
  }
  const sortColumn=async(column:number,direction:'asc'|'desc')=>{
    if(!writable())return
    const seedRow=editorSelection.startRow
    const block=await resolveSheetBlock(seedRow,column)
    if(!block){alert('정렬할 데이터가 없습니다.');return}
    setSortScope({column,direction,block,selection:{...editorSelection}})
  }
  const runSort=async(region:GridRegion,cells:Map<string,Cell>,column:number,direction:'asc'|'desc')=>{
    editor.select(region.startRow,region.startColumn);editor.select(region.endRow,region.endColumn,true)
    await sortSelection({keys:[{column,direction}],headerRows:looksLikeHeaderRow(cells,region)?1:0,caseSensitive:false},region)
  }
  // Cleanup and split rewrite whole blocks of cells at once, so they share the
  // queue the paste path already uses instead of writing cell by cell.
  const writeCells=async(inputs:PastedCell[])=>{
    if(!activeSheet||!writable()||inputs.length===0)return
    if(inputs.length>MAX_PASTE_CELLS){alert(`한 번에 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 변경할 수 있습니다.`);return}
    const updatedAt=new Date().toISOString()
    editor.putCells(inputs.map(cell=>({sheet_id:activeSheet.id,...cell,updated_at:updatedAt})))
    editor.setSaveState(navigator.onLine?'saving':'offline')
    const id=newIdempotencyKey()
    await enqueue({id,sheetId:activeSheet.id,workbookId,endpoint:'paste',attempts:0,createdAt:Date.now(),body:{base_version:serverVersion,idempotency_key:id,client_id:collaborationClientId(),cells:inputs}})
    await flushOutbox(outboxApplied(activeSheet?.id))
  }
  // A one cell selection means the whole surrounding block, the way sheet-wide
  // cleanup and sorting behave elsewhere.
  /**
   * The block a data command should act on. The grid only holds the rows on
   * screen, so working out the block from memory would sort or de-duplicate
   * the visible part of a table and leave the rest behind. The used range is
   * read from the server first.
   */
  /** Reads the sheet's used range from the server, bounded for printing. */
  const readUsedCells=async()=>{
    const empty={cells:editor.cells,rows:0,truncated:false}
    if(!activeSheet)return empty
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
    if(!used||used.max_row<1)return empty
    const lastRow=Math.min(used.max_row,MAX_PRINT_ROWS),lastColumn=Math.min(Math.max(used.max_column,1),MAX_GRID_COLUMNS)
    const label=`${address(1,1)}:${address(lastRow,lastColumn)}`
    const result=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
    if(!result)return empty
    return {cells:new Map(result.items.map(cell=>[cellKey(cell.row,cell.column),cell])),rows:used.max_row,truncated:used.max_row>MAX_PRINT_ROWS}
  }
  const resolveWorkingBlock=async():Promise<{region:typeof editorSelection;cells:Map<string,Cell>}>=>{
    const single=editorSelection.startRow===editorSelection.endRow&&editorSelection.startColumn===editorSelection.endColumn
    if(!activeSheet)return {region:editorSelection,cells:editor.cells}
    const read=async(range:{startRow:number;startColumn:number;endRow:number;endColumn:number})=>{
      const label=`${address(range.startRow,range.startColumn)}:${address(range.endRow,range.endColumn)}`
      const result=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
      return result?new Map(result.items.map(cell=>[cellKey(cell.row,cell.column),cell])):undefined
    }
    if(!single){
      const cells=await read(editorSelection)
      return {region:editorSelection,cells:cells??editor.cells}
    }
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
    if(!used||used.max_row<1)return {region:editorSelection,cells:editor.cells}
    const cells=await read({startRow:1,startColumn:1,endRow:Math.min(used.max_row,MAX_GRID_ROWS),endColumn:Math.min(Math.max(used.max_column,1),MAX_GRID_COLUMNS)})
    if(!cells)return {region:editorSelection,cells:editor.cells}
    return {region:dataRegion(cells,editorSelection.startRow,editorSelection.startColumn,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS}),cells}
  }
  /** 한 칸을 씨앗 삼아 그 칸이 속한 표를 읽는다. */
  const resolveBlockAround=async(row:number,column:number)=>{
    if(!activeSheet)return undefined
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
    if(!used||used.max_row<1)return undefined
    const label=`${address(1,1)}:${address(Math.min(used.max_row,MAX_GRID_ROWS),Math.min(Math.max(used.max_column,1),MAX_GRID_COLUMNS))}`
    const result=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
    if(!result)return undefined
    const cells=new Map(result.items.map(cell=>[cellKey(cell.row,cell.column),cell]))
    return {region:dataRegion(cells,row,column,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS}),cells}
  }
  /** The block for menu labels, which cannot wait for a read. */
  const workingRegion=()=>{
    const single=editorSelection.startRow===editorSelection.endRow&&editorSelection.startColumn===editorSelection.endColumn
    return single?dataRegion(editor.cells,editorSelection.startRow,editorSelection.startColumn,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS}):editorSelection
  }
  // Both cleanups preview before they write. Deduplication also needs to know
  // the table around the selection: shifting rows up in some columns and not
  // the others is how a tidy-up turns into scrambled data.
  /**
   * 자료를 옮기는 정리는 병합된 칸을 만나면 멈춘다.
   *
   * 그냥 두면 병합 서식이 다른 행으로 딸려 가고, 서버는 그 묶음을 통째로
   * 물리친다 — "stored merge metadata is invalid" 는 사람이 무엇을 해야
   * 하는지 알려 주지 않는다. 정렬이 오래 전부터 그렇게 하듯, 어느 칸이
   * 걸렸는지 짚어서 먼저 말한다.
   */
  const mergeBlocks=(cells:Map<string,Cell>,region:GridRegion,what:string)=>{
    const found=mergeInRegion(cells,region)
    if(!found)return false
    alert(`${address(found.row,found.column)}은(는) 병합된 셀이므로 ${what} 수 없습니다. 병합을 해제한 뒤 다시 시도하세요.`)
    return true
  }
  const openCleanup=async(mode:'duplicates'|'trim'|'subtotals'|'numbers'|'unmerge')=>{
    if(!writable())return
    const {region,cells}=await resolveWorkingBlock()
    const target:CleanupTarget={region,cells,headerRows:looksLikeHeaderRow(cells,region)?1:0}
    // Removing subtotals has to see the rows the run added, which sit below
    // the selection as often as inside it, so the whole table is resolved.
    if(mode==='subtotals'){
      const resolved=await resolveSheetBlock(region.startRow,region.startColumn)
      if(resolved){target.region=resolved.region;target.cells=resolved.cells;target.headerRows=looksLikeHeaderRow(resolved.cells,resolved.region)?1:0}
      if(mergeBlocks(target.cells,target.region,'부분합을 지울'))return
      setCleanup({mode,target})
      return
    }
    if(activeSheet){
      const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
      const used=stats?.items.find(item=>item.sheet_id===activeSheet.id)
      if(used&&used.max_row>0){
        const label=`A1:${address(Math.min(used.max_row,MAX_PRINT_ROWS),Math.min(Math.max(used.max_column,1),MAX_GRID_COLUMNS))}`
        const sheetCells=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
        if(sheetCells){
          const all=new Map(sheetCells.items.map(cell=>[cellKey(cell.row,cell.column),cell]))
          const block=dataRegion(all,region.startRow,region.startColumn,{rows:MAX_GRID_ROWS,columns:MAX_GRID_COLUMNS})
          target.block=block
          target.blockCells=all
          target.headerRows=looksLikeHeaderRow(all,block)?1:0
        }
      }
    }
    if(mode==='duplicates'&&mergeBlocks(target.blockCells??target.cells,target.block??target.region,'중복 행을 지울'))return
    setCleanup({mode,target})
  }
  const applyCleanup=async(mode:'duplicates'|'trim'|'subtotals'|'numbers'|'unmerge',region:GridRegion,cells:Map<string,Cell>,headerRows:number)=>{
    // 미리보기를 띄운 자리와 실제로 고쳐 쓰는 자리가 다를 수 있으므로,
    // 쓰기 직전에 쓰는 자리를 놓고 한 번 더 본다.
    if((mode==='duplicates'||mode==='subtotals')&&mergeBlocks(cells,region,mode==='duplicates'?'중복 행을 지울':'부분합을 지울'))return
    if(mode==='subtotals'){
      await writeCells(planRemoveSubtotals(cells,region).writes)
      // The folds the run created have nothing left to fold, so they go with
      // the rows: an outline control over plain data is a puzzle.
      const groups=activeSheet?.layout?.row_groups??[]
      for(const group of groups)
        if(group.start>=region.startRow&&group.end<=region.endRow)
          await applyLayout({action:'ungroup',axis:'row',start:group.start,count:group.end-group.start+1})
      return
    }
    const preview=mode==='duplicates'?removeDuplicateRows(cells,region,headerRows):mode==='numbers'?convertTextNumbers(cells,region):mode==='unmerge'?unmergeAndFill(cells,region):trimWhitespace(cells,region)
    await writeCells(preview.writes)
  }
  // The dialog reads the block once and previews from it, so the numbers it
  // shows and the split it performs come from the same snapshot.
  const openSplitDialog=async()=>{
    if(!writable())return
    const block=await resolveWorkingBlock()
    // The split lands to the right of the selection, so the columns it would
    // overwrite are read too. Judging that from the selection alone would
    // report "nothing to overwrite" while quietly replacing a neighbour.
    if(activeSheet){
      const first=block.region.startColumn,last=Math.min(MAX_GRID_COLUMNS,first+MAX_SPLIT_COLUMNS)
      const label=`${address(block.region.startRow,first)}:${address(block.region.endRow,last)}`
      const neighbours=await api<{items:Cell[]}>(`/api/v1/sheets/${activeSheet.id}/ranges/${label}`).catch(()=>undefined)
      if(neighbours){
        const cells=new Map(block.cells)
        for(const cell of neighbours.items)cells.set(cellKey(cell.row,cell.column),cell)
        setSplitTarget({region:block.region,cells})
        return
      }
    }
    setSplitTarget(block)
  }
  const splitColumn=async(delimiter:SplitDelimiter,block?:{region:GridRegion;cells:Map<string,Cell>})=>{
    const {region,cells}=block??await resolveWorkingBlock()
    const preview=splitTextToColumns(cells,region,delimiter)
    if(preview.columns<2){alert('선택한 열에서 구분할 수 있는 값을 찾지 못했습니다.');return}
    // 나눈 값은 오른쪽 열로 퍼진다. 그 자리에 병합이 있으면 덮인 칸 일부만
    // 서식을 잃어 반쪽짜리 병합이 남는다 — 아무 곳에도 적히지 않는 채로.
    if(mergeBlocks(cells,{...region,endColumn:region.startColumn+preview.columns-1},'열로 나눌'))return
    await writeCells(preview.writes)
  }
  // A single cell selection means the whole surrounding block, so formatting a
  // table never requires selecting it by hand first.
  const applyTableStyle=async(themeID:string)=>{
    if(!writable())return
    const {region,cells}=await resolveWorkingBlock(),theme=TABLE_THEMES.find(item=>item.id===themeID)
    if(!theme)return
    const rows=region.endRow-region.startRow+1,columns=region.endColumn-region.startColumn+1
    if(rows*columns>MAX_PASTE_CELLS){alert(`테이블 서식은 최대 ${MAX_PASTE_CELLS.toLocaleString()}셀까지 적용할 수 있습니다.`);return}
    setTableTheme(themeID)
    editor.select(region.startRow,region.startColumn);editor.select(region.endRow,region.endColumn,true)
    await writeCells(tableStyleCells(cells,region,theme,tableOptions))
  }
  const clearTableStyle=async()=>{
    if(!writable())return
    const {region,cells}=await resolveWorkingBlock()
    await writeCells(clearTableStyleCells(cells,region))
  }
  const applyBorderPreset=(preset:BorderFormatCommand['preset'])=>applyFormat({},{preset,style:preset==='none'?'none':'thin',color:borderColor})
  const quickSort=async(direction:'asc'|'desc')=>sortColumn(editorSelection.startColumn,direction)
  // The canvas cannot be printed directly, so printing renders the used range
  // into a hidden document the browser can paginate.
  /**
   * 인쇄할 범위의 조건부 서식을 받아 온다. 서버는 한 번에 재는 칸 수에 한도가
   * 있으므로 범위를 잘라 여러 번 묻는다. 도중에 실패하면 인쇄 자체를 막지는
   * 않는다 — 색이 빠진 종이가 안 나오는 종이보다는 낫다.
   */
  const printConditional=async(sheetId:string,cells:Map<string,Cell>)=>{
    const painted=new Map<string,ConditionalFormatCell>()
    const region=usedRegion(cells)
    if(!region)return painted
    const perRequest=Math.max(1,Math.floor(MAX_CONDITIONAL_CELLS/Math.max(1,region.endColumn-region.startColumn+1)))
    try{
      for(let start=region.startRow;start<=region.endRow;start+=perRequest){
        const end=Math.min(region.endRow,start+perRequest-1)
        const range=`${address(start,region.startColumn)}:${address(end,region.endColumn)}`
        const evaluated=await api<ConditionalFormatEvaluation>(`/api/v1/sheets/${sheetId}/conditional-formats:evaluate?range=${encodeURIComponent(range)}`)
        for(const item of evaluated.items)painted.set(cellKey(item.row,item.column),item)
      }
    }catch{
      return painted
    }
    return painted
  }
  /**
   * 손으로 채워 둔 칸을 본보기 삼아 같은 열의 빈 칸을 채운다. 규칙을 알아내지
   * 못하면 왜 못 했는지 말한다 — 아무 일도 일어나지 않는 것이 가장 나쁘다.
   */
  const startFlashFill=async()=>{
    if(!activeSheet||!writable())return
    const block=await resolveWorkingBlock()
    const column=editorSelection.startColumn
    // 대상 열만 골라 놓는 것이 가장 자연스러운 몸짓인데, 그러면 고른 것이
    // 그 열뿐이라 규칙을 만들 재료가 없다. 그럴 때는 옆 열들을 표에서 찾아
    // 붙이되, 사람이 고른 줄 범위는 그대로 지킨다.
    let region=block.region
    let cells=block.cells
    if(editorSelection.startColumn===editorSelection.endColumn){
      const around=await resolveBlockAround(editorSelection.startRow,editorSelection.startColumn)
      if(around&&around.region.startColumn<around.region.endColumn){
        cells=around.cells
        region=editorSelection.startRow===editorSelection.endRow
          ?around.region
          :{...around.region,startRow:editorSelection.startRow,endRow:editorSelection.endRow}
      }
    }
    if(column<region.startColumn||column>region.endColumn){
      alert('채울 열을 표 안에서 선택하세요.')
      return
    }
    const plan=planFlashFill(cells,region,column)
    if(plan==='no-examples'){alert('먼저 한두 칸을 손으로 채워 본보기를 보여 주세요.');return}
    if(plan==='nothing-to-fill'){alert('이 열에는 빈 칸이 없습니다.');return}
    if(plan==='no-rule'){alert('채워 둔 값에서 규칙을 찾지 못했습니다. 본보기를 하나 더 채워 보거나, 옆 열의 값으로 만들 수 있는 값인지 확인하세요.');return}
    setFlashFill({plan,column})
  }
  // 화면에서 정한 열 너비. 미리보기와 실제 인쇄가 같은 너비로 세야 "한 장에
  // 들어갑니다" 하고 두 장을 뱉는 일이 없다. 확대 배율은 화면 사정이므로 뺀다.
  const printColumnWidths=useMemo(()=>new Map((activeSheet?.layout?.column_widths??[]).map(item=>[item.index,item.size])),[activeSheet?.layout?.column_widths])
  /**
   * 페이지 설정을 열기 전에 인쇄가 읽는 것과 똑같은 칸을 읽는다.
   *
   * 처음에는 시트 크기만 물어 값싸게 넘겼는데, 그 수는 인쇄가 쓰는 범위와
   * 달랐다. 시트 크기는 가장 큰 행과 열만 알려 주므로 자료가 D열에서
   * 시작해도 A열부터 세었고, 값 없이 서식만 든 칸까지 세었다. 미리보기가
   * 네 장이라고 하고 세 장이 나오면 미리보기를 볼 이유가 없다.
   *
   * 인쇄와 같은 칸에서 같은 함수로 범위를 구한다. 열 때 인쇄만큼 걸리지만,
   * 값싸고 틀린 수보다 낫다.
   */
  const openPrintOptions=async()=>{
    if(!activeSheet){setPrintOpen({});return}
    const printed=await readUsedCells()
    const region=printAreaRegion(activeSheet.layout?.print_area)??usedRegion(printed.cells)
    setPrintOpen({region,rows:printed.rows,truncated:printed.truncated})
  }
  /**
   * 여러 시트를 합치기 전에 각 시트의 자료가 있는 자리를 읽는다.
   *
   * 시트마다 크기가 다르므로 한 범위를 모두에 씌우면 어느 시트는 빈 줄까지
   * 세고 어느 시트는 뒷부분이 잘린다. 시트마다 자기 크기를 물어 본다.
   */
  const openConsolidate=async()=>{
    if(!writable())return
    const sheets=(workbook.data?.sheets??[]).filter(sheet=>!sheet.hidden)
    if(sheets.length<2){alert('합칠 시트가 두 개 이상 있어야 합니다.');return}
    const stats=await api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`).catch(()=>undefined)
    // 시트마다 칸을 통째로 읽으므로 한없이 늘리면 여는 것만으로 브라우저가
    // 멈춘다. 넘치면 조용히 버리지 않고 무엇을 못 읽었는지 말한다.
    if(sheets.length>MAX_CONSOLIDATE_SHEETS){
      alert(`시트가 ${sheets.length}개입니다. 한 번에 ${MAX_CONSOLIDATE_SHEETS}개까지 합칠 수 있어 앞의 ${MAX_CONSOLIDATE_SHEETS}개만 읽습니다.`)
    }
    const sources:ConsolidateSource[]=[],withoutTable:string[]=[]
    for(const sheet of sheets.slice(0,MAX_CONSOLIDATE_SHEETS)){
      const used=stats?.items.find(item=>item.sheet_id===sheet.id)
      if(!used||used.max_row<2||used.max_column<2){withoutTable.push(sheet.name);continue}
      const endRow=Math.min(used.max_row,MAX_PRINT_ROWS),endColumn=Math.min(used.max_column,MAX_GRID_COLUMNS)
      const label=`${address(1,1)}:${address(endRow,endColumn)}`
      const result=await api<{items:Cell[]}>(`/api/v1/sheets/${sheet.id}/ranges/${label}`).catch(()=>undefined)
      if(!result){withoutTable.push(sheet.name);continue}
      const cells=new Map(result.items.map(cell=>[cellKey(cell.row,cell.column),cell]))
      // 표가 A1 에서 시작한다고 여기면 안 된다. C3 에서 시작하는 시트는 첫
      // 행이 비어 있어 항목 이름을 하나도 못 찾고, 고른 목록에는 남아 있는
      // 채로 아무것도 보태지 않는다. 인쇄에서 똑같은 잘못을 한 적이 있다.
      const region=usedRegion(cells)
      if(!region||region.endRow<=region.startRow||region.endColumn<=region.startColumn){withoutTable.push(sheet.name);continue}
      sources.push({sheetName:sheet.name,cells,region})
    }
    if(sources.length<2){alert('이름표가 있는 표를 가진 시트가 두 개 이상 있어야 합니다.');return}
    // 표가 없어 못 읽은 시트는 목록에 나오지 않는다. 왜 없는지 말하지 않으면
    // 사람은 자기가 고른 것이 다 들어갔다고 믿는다.
    setConsolidateSources(sources)
    setConsolidateSkipped(withoutTable)
  }
  /** 합친 결과를 새 시트에 적는다. 원래 시트는 건드리지 않는다. */
  const writeConsolidated=async(result:ConsolidateResult,operation:ConsolidateFunction)=>{
    const cells:PastedCell[]=[{row:1,column:1,value:CONSOLIDATE_LABELS[operation],formula:undefined,style:{bold:true}}]
    result.columnLabels.forEach((label,index)=>cells.push({row:1,column:index+2,value:label,formula:undefined,style:{bold:true}}))
    result.rowLabels.forEach((label,rowIndex)=>{
      cells.push({row:rowIndex+2,column:1,value:label,formula:undefined,style:{bold:true}})
      result.columnLabels.forEach((_,columnIndex)=>{
        const value=result.values.get(`${rowIndex}:${columnIndex}`)
        // 없는 것은 0이 아니다. 0을 적으면 실적이 0원이었다는 뜻이 된다.
        if(value===undefined)return
        cells.push({row:rowIndex+2,column:columnIndex+2,value,formula:undefined,style:undefined})
      })
    })
    if(cells.length>MAX_PASTE_CELLS){alert(`결과가 ${MAX_PASTE_CELLS.toLocaleString()}셀을 넘어 시트로 옮길 수 없습니다.`);return}
    try{
      const sheet=await api<Sheet>(`/api/v1/workbooks/${workbookId}/sheets`,{method:'POST',body:JSON.stringify({name:nextSheetName('통합')})})
      const idempotencyKey=newIdempotencyKey()
      const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`)
      await api<MutationResult>(`/api/v1/sheets/${sheet.id}/cells:batch`,{method:'PATCH',
        body:JSON.stringify({base_version:latest.version,idempotency_key:idempotencyKey,client_id:collaborationClientId(),cells})})
      setConsolidateSources(undefined)
      setActiveSheet(sheet)
      await refreshWorkbook()
    }catch(error){alert(error instanceof Error?error.message:'통합 시트를 만들지 못했습니다.')}
  }
  /**
   * 표를 훑어 고칠 만한 것을 세어 보여 준다.
   *
   * 정리 도구가 여섯 개인데 사람이 "데이터 정리" 를 열어 볼 생각을 해야
   * 알 수 있었다. 무엇이 잘못됐는지 모르는 채로 열어 보는 사람은 없다.
   */
  const openInspect=async()=>{
    const {region,cells}=await resolveWorkingBlock()
    setInspect({region,cells,headerRows:looksLikeHeaderRow(cells,region)?1:0})
  }
  /** 검사에서 고른 갈래를 그 갈래의 미리보기로 넘긴다. */
  const fixFromInspect=(kind:TableFinding['kind'])=>{
    setInspect(undefined)
    if(kind==='unmerge'||kind==='numbers'||kind==='trim'||kind==='duplicates'||kind==='subtotals')void openCleanup(kind)
  }
  /** 사람이 적은 "시트!범위" 를 워크북의 시트에 붙인다. */
  const parseSheetRange=(value:string,sheets:Sheet[]):{sheet:Sheet;region:GridRegion}|undefined=>{
    const parsed=splitSheetRange(value)
    if(!parsed)return undefined
    const name=parsed.sheetName??activeSheet?.name??''
    const sheet=sheets.find(candidate=>candidate.name.toLowerCase()===name.toLowerCase())
    return sheet?{sheet,region:parsed.region}:undefined
  }
  /**
   * 두 목록을 견주는 흐름. 왼쪽은 지금 있는 자리의 자료 덩어리, 오른쪽은
   * 사람이 적어 주는 "시트!범위" 다.
   *
   * 대사는 대개 시트 두 개를 견주는 일이다 — 은행에서 받은 내역과 장부처럼.
   * 한 시트 안으로 제한하면 정작 하고 싶은 일을 못 한다.
   */
  const openCompare=async()=>{
    // 메뉴에서는 보기 전용일 때 잠겨 있지만 명령 팔레트는 그대로 부른다.
    // 막지 않으면 보기 전용 사용자가 범위를 고르고 견주기까지 한 뒤에야
    // 결과를 적을 수 없다는 말을 듣는다.
    if(!activeSheet||!writable())return
    const block=await resolveWorkingBlock()
    const sheets=workbook.data?.sheets??[]
    const other=sheets.find(sheet=>sheet.id!==activeSheet.id&&!sheet.hidden)
    setPrompt({
      title:'두 목록 비교',label:'비교할 범위',confirmLabel:'비교',
      value:other?`${other.name}!A1:${address(100,Math.max(1,block.region.endColumn))}`:'',
      placeholder:'예: 장부!A1:C100',
      hint:'다른 시트라면 시트 이름 뒤에 느낌표를 붙여 적으세요.',
      validate:value=>parseSheetRange(value,sheets)?undefined:'시트!A1:C100 꼴로 적어 주세요. 시트 이름이 맞는지도 확인하세요.',
      onSubmit:value=>{
        const target=parseSheetRange(value,sheets)
        if(!target)return
        void (async()=>{
          const label=`${address(target.region.startRow,target.region.startColumn)}:${address(target.region.endRow,target.region.endColumn)}`
          const result=await api<{items:Cell[]}>(`/api/v1/sheets/${target.sheet.id}/ranges/${label}`).catch(()=>undefined)
          if(!result){alert('비교할 범위를 읽지 못했습니다.');return}
          setCompare({
            left:{sheet:activeSheet,region:block.region,cells:block.cells},
            right:{sheet:target.sheet,region:target.region,cells:new Map(result.items.map(cell=>[cellKey(cell.row,cell.column),cell]))},
          })
        })()
      },
    })
  }
  /** 견준 결과를 새 시트에 적는다. 원래 자료는 건드리지 않는다. */
  const writeCompareReport=async(result:CompareResult,keys:{left:number;right:number},headerRows:number)=>{
    if(!compare)return
    const rows:Array<Array<string|number>>=[]
    rows.push(['구분','시트','행','키'])
    for(const row of result.onlyLeft)rows.push(['왼쪽에만',compare.left.sheet.name,row.row,row.label])
    for(const row of result.onlyRight)rows.push(['오른쪽에만',compare.right.sheet.name,row.row,row.label])
    for(const item of result.duplicated)
      rows.push(['거듭 나옴',item.side==='left'?compare.left.sheet.name:compare.right.sheet.name,'',`${item.label} × ${item.count}`])
    if(rows.length===1)rows.push(['맞음','','','두 목록이 키 기준으로 완전히 맞습니다.'])
    const cells:PastedCell[]=[]
    rows.forEach((row,rowIndex)=>row.forEach((value,columnIndex)=>{
      if(value==='')return
      cells.push({row:rowIndex+1,column:columnIndex+1,value,formula:undefined,style:rowIndex===0?{bold:true}:undefined})
    }))
    if(cells.length>MAX_PASTE_CELLS){alert(`결과가 ${MAX_PASTE_CELLS.toLocaleString()}셀을 넘어 시트로 옮길 수 없습니다.`);return}
    try{
      const sheet=await api<Sheet>(`/api/v1/workbooks/${workbookId}/sheets`,{method:'POST',body:JSON.stringify({name:nextSheetName('대사')})})
      const idempotencyKey=newIdempotencyKey()
      const latest=await api<Workbook>(`/api/v1/workbooks/${workbookId}`)
      await api<MutationResult>(`/api/v1/sheets/${sheet.id}/cells:batch`,{method:'PATCH',
        body:JSON.stringify({base_version:latest.version,idempotency_key:idempotencyKey,client_id:collaborationClientId(),cells})})
      setCompare(undefined)
      setActiveSheet(sheet)
      await refreshWorkbook()
    }catch(error){alert(error instanceof Error?error.message:'결과 시트를 만들지 못했습니다.')}
    void keys;void headerRows
  }
  const printSheet=async(choice:PrintChoice=loadPrintChoice())=>{
    if(!activeSheet)return
    // The grid holds only the rows on screen, so printing from memory would
    // put a page of whatever was scrolled to in front of the reader.
    const printed=await readUsedCells()
    if(printed.truncated&&!window.confirm(`시트가 ${printed.rows.toLocaleString()}행입니다. 앞의 ${MAX_PRINT_ROWS.toLocaleString()}행만 인쇄합니다. 계속할까요?`))return
    // Filters, hidden rows and folded groups all mean the same thing to a
    // reader: that row is not on the screen they are printing.
    const hiddenRows=new Set<number>(filterResult.data?.hidden_rows??[])
    for(const range of activeSheet.layout?.hidden_rows??[])
      for(let row=range.start;row<=range.end;row+=1)hiddenRows.add(row)
    for(const row of collapsedIndexes(activeSheet.layout?.row_groups))hiddenRows.add(row)
    // 화면에서 정한 열 너비 그대로 찍어야 종이 폭에 몇 열이 들어가는지 셀 수
    // 있다. 확대 배율은 화면 사정이므로 뺀다.
    const printWidths=printColumnWidths
    // 조건부 서식은 값에 따라 그때그때 정해지므로 셀에 저장돼 있지 않다.
    // 따로 물어보지 않으면 사람이 읽으라고 칠해 놓은 표가 종이에서는 아무
    // 표시 없는 숫자 뭉치가 된다.
    const conditional=await printConditional(activeSheet.id,printed.cells)
    const html=printableDocument(printed.cells,{title:workbook.data?.title??'kanpic',sheetName:activeSheet.name,gridlines:showGridlines,headers:true,hiddenRows,conditional,
      columnWidth:column=>printColumnWidths.get(column)??108,frozenRows:activeSheet.layout?.frozen_rows??0,
      printArea:activeSheet.layout?.print_area,
      orientation:choice.orientation,margin:choice.margin,fit:choice.fit})
    const frame=document.createElement('iframe')
    frame.setAttribute('aria-hidden','true');frame.style.cssText='position:fixed;right:0;bottom:0;width:0;height:0;border:0'
    frame.src='/print-frame'
    const ready=new Promise<void>((resolve,reject)=>{
      frame.addEventListener('load',()=>resolve(),{once:true})
      frame.addEventListener('error',()=>reject(new Error('print frame')),{once:true})
      window.setTimeout(()=>reject(new Error('print frame timed out')),8000)
    })
    document.body.appendChild(frame)
    try{await ready}catch{frame.remove();alert('인쇄 화면을 열지 못했습니다.');return}
    const target=frame.contentWindow?.document
    if(!target){frame.remove();alert('인쇄 화면을 열지 못했습니다.');return}
    target.open();target.write(html);target.close()
    frame.contentWindow?.focus();frame.contentWindow?.print()
    window.setTimeout(()=>frame.remove(),1000)
  }
  const toggleFullscreen=()=>{
    if(document.fullscreenElement)void document.exitFullscreen().catch(()=>{})
    else void document.documentElement.requestFullscreen?.().catch(()=>alert('이 브라우저에서는 전체 화면을 사용할 수 없습니다.'))
  }
  const createWorkbook=async()=>{const created=await api<Workbook>('/api/v1/workbooks',{method:'POST',body:JSON.stringify({title:'제목 없는 워크북',workspace_id:'default'})});window.location.href=`/workbooks/${created.id}`}
  const duplicateWorkbook=async()=>{const copy=await api<Workbook>(`/api/v1/workbooks/${workbookId}/duplicate`,{method:'POST',body:JSON.stringify({title:`${workbook.data?.title??'워크북'} 복사본`})});window.location.href=`/workbooks/${copy.id}`}
  const renameWorkbook=()=>setPrompt({
    title:'워크북 이름 변경',label:'워크북 이름',value:workbook.data?.title??'',confirmLabel:'이름 바꾸기',
    validate:value=>value.trim()===''?'이름을 입력하세요.':undefined,
    onSubmit:value=>{
      if(value.trim()===workbook.data?.title)return
      void api<Workbook>(`/api/v1/workbooks/${workbookId}`,{method:'PATCH',body:JSON.stringify({title:value.trim()})})
        .then(refreshWorkbook)
        .catch(error=>alert(error instanceof Error?error.message:'이름을 바꾸지 못했습니다.'))
    },
  })
  const trashWorkbook=async()=>{
    if(!window.confirm(`'${workbook.data?.title??''}' 워크북을 휴지통으로 옮길까요? 홈 화면의 휴지통에서 복원할 수 있습니다.`))return
    await api(`/api/v1/workbooks/${workbookId}`,{method:'DELETE'})
    window.location.href='/'
  }
  const insertLink=()=>setLinkOpen(true)
  const handleGridMenu=(command:GridMenuCommand)=>{
    switch(command.command){
      case 'sort-dialog':setSortOpen(true);return
      case 'sort-column':void sortColumn(command.column,command.direction);return
      case 'filter':setFilterOpen(true);return
      case 'comment':setRightPanel('comments');return
      case 'named-range':setNamedRangeOpen(true);return
      case 'conditional-format':setConditionalFormatOpen(true);return
      case 'data-validation':setValidationOpen(true);return
      case 'chart':setChartDialog(null);return
      case 'pivot':setPivotDialog(null);return
      case 'format-dialog':setFormatOpen(true);return
      case 'layout-dialog':setLayoutOpen(true);return
      case 'note':setNoteOpen(true);return
      case 'cell-history':setHistoryCell(address(command.row,command.column));return
      case 'subtotal':void openSubtotal();return
      case 'cleanup-duplicates':void openCleanup('duplicates');return
      case 'split-columns':void openSplitDialog();return
      case 'column-stats':setRightPanel('stats');return
      case 'column-filter':setColumnFilter({column:command.column,x:command.x,y:command.y});return
      case 'structure-dialog':setStructureOpen(true);return
      case 'clear-format':void clearFormat();return
      case 'find-replace':openSearch(true);return
      case 'merge':void changeMerge(command.merge);return
      case 'agent':setAgentDraft(current=>({mode:command.mode,request:command.request,key:current.key+1}));setRightPanel('ai');return
    }
  }
  const saveWorkbook=async()=>{if(!navigator.onLine){editor.setSaveState('offline');return}editor.setSaveState('saving');try{await flushOutbox(outboxApplied(activeSheet?.id));const current=useEditorStore.getState();if(current.saveState==='saving')current.setSaveState(current.conflicts?'conflict':'saved',current.conflicts)}catch{editor.setSaveState('error')}}
  useEffect(()=>{const shortcut=(event:KeyboardEvent)=>{if(event.defaultPrevented||editableTarget(event.target)||document.querySelector('[role="dialog"][aria-modal="true"]'))return;const primary=event.ctrlKey||event.metaKey,key=event.key.toLowerCase(),numberFormats:Record<string,string>={Digit1:'#,##0.00',Digit2:'hh:mm:ss',Digit3:'yyyy-mm-dd',Digit4:'₩#,##0',Digit5:'0.00%',Digit6:'0.00E+00'}
    if(primary&&event.altKey&&key==='m'){event.preventDefault();setRightPanel('comments');return}
    if(primary&&event.code==='Slash'){event.preventDefault();setShortcutsOpen(true);return}
    if(primary&&key==='h'){event.preventDefault();openSearch(true);return}
    if(primary&&key==='k'){event.preventDefault();setQuickOpen(true);return}
    if(primary&&key==='p'){event.preventDefault();void printSheet();return}
    // 엑셀을 쓰던 사람은 빠른 채우기를 Ctrl+E 로 부른다.
    if(primary&&key==='e'){event.preventDefault();void startFlashFill();return}
    if(event.key==='F11'&&!primary&&!event.shiftKey){event.preventDefault();toggleFullscreen();return}
    if(primary&&key==='f'){event.preventDefault();openSearch(false);return}
    if(primary&&event.code==='Backquote'){event.preventDefault();setShowFormulas(current=>!current);return}
    if(primary&&event.code==='Backslash'){event.preventDefault();void clearFormat();return}
    if(primary&&event.altKey&&(event.code==='Equal'||event.code==='NumpadAdd')){event.preventDefault();void changeStructure('insert');return}
    if(primary&&event.altKey&&(event.code==='Minus'||event.code==='NumpadSubtract')){event.preventDefault();void changeStructure('delete');return}
    if(event.altKey&&!primary&&(event.code==='Equal'||event.code==='NumpadAdd')){event.preventDefault();gridShortcut({command:'auto-sum'});return}
    if(primary&&!event.shiftKey&&event.code==='Semicolon'){event.preventDefault();gridShortcut({command:'insert-today'});return}
    if(primary&&event.shiftKey&&event.code==='Semicolon'){event.preventDefault();gridShortcut({command:'insert-now'});return}
    if(primary&&event.altKey&&event.code==='Digit0'){event.preventDefault();void hideSelection('column');return}
    if(primary&&event.altKey&&event.code==='Digit9'){event.preventDefault();void hideSelection('row');return}
    if(primary&&event.shiftKey&&event.code==='Digit0'){event.preventDefault();void applyLayout({action:'show_all',axis:'column'});return}
    if(primary&&event.shiftKey&&event.code==='Digit9'){event.preventDefault();void applyLayout({action:'show_all',axis:'row'});return}
    if(primary&&event.key==='PageDown'){event.preventDefault();moveSheet(1);return}
    if(primary&&event.key==='PageUp'){event.preventDefault();moveSheet(-1);return}
    if(event.key==='PageDown'||event.key==='PageUp'){event.preventDefault();gridShortcut({command:'move-page',direction:event.altKey?event.key==='PageDown'?'right':'left':event.key==='PageDown'?'down':'up',extend:event.shiftKey});return}
    if(primary&&event.shiftKey&&key==='a'){event.preventDefault();gridShortcut({command:'select-data-region'});return}
    if(primary&&key==='s'){event.preventDefault();void saveWorkbook();return}
    if(primary&&key==='z'){event.preventDefault();void revertOperation(event.shiftKey?'redo':'undo');return}
    if(primary&&key==='y'){event.preventDefault();void revertOperation('redo');return}
    if(event.shiftKey&&event.key==='F11'){event.preventDefault();void createSheet();return}
    if(primary&&event.shiftKey&&numberFormats[event.code]){event.preventDefault();void applyFormat({number_format:numberFormats[event.code]});return}
    if(event.altKey&&event.shiftKey&&event.code==='Digit5'){event.preventDefault();void applyFormat({strike:activeCell?.style?.strike!==true});return}
    if(primary&&event.shiftKey&&key==='b'){event.preventDefault();void applyFormat({bold:activeCell?.style?.bold!==true});return}
    if(primary&&event.shiftKey&&key==='i'){event.preventDefault();void applyFormat({italic:activeCell?.style?.italic!==true});return}
    if(primary&&event.shiftKey&&key==='u'){event.preventDefault();void applyFormat({underline:activeCell?.style?.underline!==true});return}
    if(primary&&event.shiftKey&&key==='l'){event.preventDefault();void applyFormat({horizontal_align:'left'});return}
    if(primary&&event.shiftKey&&key==='e'){event.preventDefault();void applyFormat({horizontal_align:'center'});return}
    if(primary&&event.shiftKey&&key==='r'){event.preventDefault();void applyFormat({horizontal_align:'right'});return}
    if(primary&&key==='b'){event.preventDefault();void applyFormat({bold:activeCell?.style?.bold!==true});return}
    if(primary&&key==='i'){event.preventDefault();void applyFormat({italic:activeCell?.style?.italic!==true});return}
    if(primary&&key==='u'){event.preventDefault();void applyFormat({underline:activeCell?.style?.underline!==true});return}
    if(primary&&key==='a'){event.preventDefault();gridShortcut({command:'select-all'});return}
    if(primary&&event.code==='Space'){event.preventDefault();gridShortcut({command:'select-column'});return}
    if(event.shiftKey&&event.code==='Space'){event.preventDefault();gridShortcut({command:'select-row'});return}
    if(primary&&['ArrowUp','ArrowDown','ArrowLeft','ArrowRight'].includes(event.key)){event.preventDefault();gridShortcut({command:'move-data-edge',direction:event.key.replace('Arrow','').toLowerCase() as 'up'|'down'|'left'|'right',extend:event.shiftKey});return}
    if(primary&&!event.shiftKey&&key==='d'){event.preventDefault();gridShortcut({command:'fill-down'});return}
    if(primary&&!event.shiftKey&&key==='r'){event.preventDefault();gridShortcut({command:'fill-right'});return}
    // Escape puts a loaded format brush down, which is the way out of it.
    if(event.key==='Escape'&&formatBrushRef.current){event.preventDefault();setFormatBrush(undefined);return}
    if(primary&&event.key==='Home'){event.preventDefault();gridShortcut({command:'move-first'});return}
    if(primary&&event.key==='End'){event.preventDefault();gridShortcut({command:'move-last'})}}
    window.addEventListener('keydown',shortcut);return()=>window.removeEventListener('keydown',shortcut)},[activeCell?.style?.bold,activeCell?.style?.italic,activeCell?.style?.strike,activeCell?.style?.underline,applyFormat,createSheet,revertOperation,saveWorkbook])
  if(workbook.isLoading)return <div className="editor-loading">kanpic 편집기를 준비하는 중…</div>
  // A workbook that exists but is not shared with this user offers the Sheets
  // style request-access flow instead of a dead end.
  if(workbook.error instanceof ApiError&&workbook.error.status===403)return <div className="page-shell"><AppHeader build={build} session={session}/><main className="access-denied">
    <section>
      <Lock/>
      <h1>액세스 권한이 필요합니다</h1>
      <p>이 워크북은 소유자가 지정한 사용자, 부서 또는 역할만 열 수 있습니다. 필요한 권한을 요청하면 소유자에게 전달됩니다.</p>
      {accessRequested
        ?<div className="access-requested" role="status">액세스 요청을 보냈습니다. 소유자가 승인하면 이 링크로 다시 열 수 있습니다.</div>
        :<div className="access-actions">
          <button className="primary" disabled={requestingAccess} onClick={()=>{
            setRequestingAccess(true)
            void api<unknown>(`/api/v1/workbooks/${workbookId}/access-requests`,{method:'POST',body:JSON.stringify({requested_role:'editor'})})
              .then(()=>setAccessRequested(true))
              .catch(reason=>alert(reason instanceof Error?reason.message:'액세스 요청을 보내지 못했습니다.'))
              .finally(()=>setRequestingAccess(false))
          }}>편집 권한 요청</button>
          <button className="secondary" disabled={requestingAccess} onClick={()=>{
            setRequestingAccess(true)
            void api<unknown>(`/api/v1/workbooks/${workbookId}/access-requests`,{method:'POST',body:JSON.stringify({requested_role:'viewer'})})
              .then(()=>setAccessRequested(true))
              .catch(reason=>alert(reason instanceof Error?reason.message:'액세스 요청을 보내지 못했습니다.'))
              .finally(()=>setRequestingAccess(false))
          }}>보기 권한 요청</button>
        </div>}
      <a href="/">워크북 목록으로 돌아가기</a>
    </section>
  </main></div>
  if(!workbook.data||!activeSheet)return <div className="editor-loading error">워크북을 찾을 수 없습니다.</div>
  const accessRole:ShareRole=workbook.data.access_role??'owner'
  const canWrite=accessRole==='editor'||accessRole==='owner'
  const canComment=canWrite||accessRole==='commenter'
  const readOnly=!canWrite
  const conflictCount=Math.max(conflicts.data?.items.length??0,editor.conflicts)
  const displaySaveState=conflictCount>0&&editor.saveState==='saved'?'conflict':editor.saveState
  const queueLabel=pending>0&&(displaySaveState==='saving'||displaySaveState==='offline')?` · ${pending.toLocaleString()}건 대기`:''
  const saveLabel={saved:'모든 변경사항 저장됨',saving:'저장 중…',offline:'오프라인 · 로컬 저장',conflict:`충돌 ${conflictCount}건`,error:'저장 오류'}[displaySaveState]+queueLabel
  const mergedSelection=Boolean(cellMerge(editor.cells.get(cellKey(editorSelection.startRow,editorSelection.startColumn))))
  const selectedRows=editorSelection.endRow-editorSelection.startRow+1,selectedColumns=editorSelection.endColumn-editorSelection.startColumn+1
  const numberFormatItems:MenuItem[]=[
    {kind:'item',label:'자동',onSelect:()=>void applyFormat({number_format:null})},
    {kind:'item',label:'숫자 1,234.56',shortcut:'Ctrl+Shift+1',onSelect:()=>void applyFormat({number_format:'#,##0.00'})},
    {kind:'item',label:'통화 ₩1,234',shortcut:'Ctrl+Shift+4',onSelect:()=>void applyFormat({number_format:'₩#,##0'})},
    {kind:'item',label:'백분율 12.34%',shortcut:'Ctrl+Shift+5',onSelect:()=>void applyFormat({number_format:'0.00%'})},
    {kind:'item',label:'날짜 2026-08-03',shortcut:'Ctrl+Shift+3',onSelect:()=>void applyFormat({number_format:'yyyy-mm-dd'})},
    {kind:'item',label:'시간 13:45:30',shortcut:'Ctrl+Shift+2',onSelect:()=>void applyFormat({number_format:'hh:mm:ss'})},
    {kind:'item',label:'지수 1.23E+04',shortcut:'Ctrl+Shift+6',onSelect:()=>void applyFormat({number_format:'0.00E+00'})},
  ]
  // Quick navigation: sheets, named ranges, a direct A1 address, other
  // workbooks and every editor command in one keyboard-driven list.
  const quickItems:QuickItem[]=[
    ...workbook.data.sheets.filter(sheet=>!sheet.hidden).map(sheet=>({
      id:`sheet:${sheet.id}`,group:'시트',label:sheet.name,
      hint:sheet.id===activeSheet.id?'현재 시트':`${workbook.data.title} 시트`,
      icon:<Table2/>,keywords:'sheet tab 시트 탭',run:()=>setActiveSheet(sheet),
    })),
    ...(namedRanges.data?.items??[]).map(item=>({
      id:`named:${item.id}`,group:'이름 범위',label:item.name,hint:item.range,icon:<Link2/>,
      keywords:'named range 이름 범위',run:()=>navigateToRange(item.sheet_id,item.range),
    })),
    ...(sheetTables.data?.items??[]).map(item=>({
      id:`table:${item.id}`,group:'표',label:item.name,hint:item.range,icon:<Table2/>,
      keywords:'table 표',run:()=>navigateToRange(item.sheet_id,item.range),
    })),
    {id:'cmd:share',group:'명령',label:'공유 설정 열기',icon:<Share2/>,keywords:'share 공유 권한',run:()=>setShareOpen(true)},
    {id:'cmd:sheets',group:'명령',label:'모든 시트 관리',icon:<Table2/>,keywords:'sheet manager 시트 관리 숨김',run:()=>setSheetManagerOpen(true)},
    {id:'cmd:find',group:'명령',label:'워크북 검색',shortcut:'Ctrl+F',icon:<Search/>,keywords:'find search 찾기 검색',run:()=>openSearch(false)},
    {id:'cmd:replace',group:'명령',label:'찾기 및 바꾸기',shortcut:'Ctrl+H',icon:<Search/>,keywords:'replace 바꾸기 치환',run:()=>openSearch(true)},
    {id:'cmd:comments',group:'명령',label:'댓글 패널',shortcut:'Ctrl+Alt+M',icon:<MessageSquare/>,keywords:'comment 댓글',run:()=>setRightPanel('comments')},
    {id:'cmd:history',group:'명령',label:'버전 이력',icon:<History/>,keywords:'version history 버전 이력',run:()=>setRightPanel('history')},
    {id:'cmd:ai',group:'명령',label:'Workbook Agent',icon:<Bot/>,keywords:'ai agent assistant 도우미 에이전트',run:()=>setRightPanel('ai')},
    {id:'cmd:charts',group:'명령',label:'차트 패널',icon:<BarChart3/>,keywords:'chart 차트',run:()=>setRightPanel('charts')},
    {id:'cmd:pivots',group:'명령',label:'피벗 패널',icon:<Table2/>,keywords:'pivot 피벗',run:()=>setRightPanel('pivots')},
    ...(presentationEnabled?[{id:'cmd:presentations',group:'명령',label:'프레젠테이션 목록',icon:<Presentation/>,keywords:'presentation ppt pptx 프레젠테이션 발표 슬라이드',run:()=>setRightPanel('presentations')}]:[]),
    {id:'cmd:conflicts',group:'명령',label:`편집 충돌${conflictCount>0?` ${conflictCount}건`:''}`,icon:<AlertTriangle/>,keywords:'conflict 충돌',run:()=>setRightPanel('conflicts')},
    {id:'cmd:formulas',group:'명령',label:showFormulas?'수식 표시 끄기':'수식 표시 켜기',shortcut:'Ctrl+`',icon:<Search/>,keywords:'formula view 수식 보기',run:()=>setShowFormulas(current=>!current)},
    {id:'cmd:sort',group:'명령',label:'범위 정렬',icon:<ArrowUpDown/>,keywords:'sort 정렬',run:()=>setSortOpen(true)},
    {id:'cmd:filter',group:'명령',label:'필터 보기',icon:<Filter/>,keywords:'filter 필터',run:()=>setFilterOpen(true)},
    {id:'cmd:validation',group:'명령',label:'데이터 검증',icon:<BadgeCheck/>,keywords:'validation 검증',run:()=>setValidationOpen(true)},
    {id:'cmd:conditional',group:'명령',label:'조건부 서식',icon:<Palette/>,keywords:'conditional format 조건부 서식',run:()=>setConditionalFormatOpen(true)},
    {id:'cmd:named',group:'명령',label:'이름 범위 관리',icon:<Link2/>,keywords:'named range 이름 범위',run:()=>setNamedRangeOpen(true)},
    {id:'cmd:named-function',group:'명령',label:'이름 있는 수식 관리',icon:<Link2/>,keywords:'named function 이름 있는 수식 사용자 정의 함수',run:()=>setNamedFunctionOpen(true)},
    {id:'cmd:layout',group:'명령',label:'시트 레이아웃',icon:<Table2/>,keywords:'layout freeze 고정 레이아웃',run:()=>setLayoutOpen(true)},
    {id:'cmd:structure',group:'명령',label:'행과 열 관리',icon:<Table2/>,keywords:'row column 행 열',run:()=>setStructureOpen(true)},
    {id:'cmd:export',group:'명령',label:'XLSX로 내보내기',icon:<Download/>,keywords:'export xlsx 내보내기',run:()=>void exportWorkbook('xlsx')},
    {id:'cmd:print',group:'명령',label:'인쇄',shortcut:'Ctrl+P',icon:<Download/>,keywords:'print 인쇄 출력',run:()=>void printSheet()},
    {id:'cmd:watch',group:'명령',label:'변경 알림 설정',icon:<Link2/>,keywords:'watch notify 알림 변경 지켜보기 메일',run:()=>setWatchOpen(true)},
    {id:'cmd:page-setup',group:'명령',label:'페이지 설정',icon:<Download/>,keywords:'page setup 페이지 설정 인쇄 방향 여백 가로 세로',run:()=>void openPrintOptions()},
    {id:'cmd:functions',group:'명령',label:'함수 목록',icon:<Search/>,keywords:'function 함수 수식',run:()=>setFunctionsOpen(true)},
    {id:'cmd:gridlines',group:'명령',label:showGridlines?'눈금선 숨기기':'눈금선 표시',icon:<Grid2X2/>,keywords:'gridline 눈금선 격자',run:()=>setShowGridlines(current=>!current)},
    {id:'cmd:fullscreen',group:'명령',label:'전체 화면',shortcut:'F11',icon:<Grid2X2/>,keywords:'fullscreen 전체 화면',run:()=>toggleFullscreen()},
    {id:'cmd:subtotal',group:'명령',label:'부분합',icon:<Table2/>,keywords:'subtotal 부분합 소계 그룹 합계',run:()=>void openSubtotal()},
    {id:'cmd:subtotal-remove',group:'명령',label:'부분합 제거',icon:<Table2/>,keywords:'subtotal 부분합 제거 소계 삭제',run:()=>void openCleanup('subtotals')},
    {id:'cmd:slicer',group:'명령',label:'슬라이서 추가',icon:<Table2/>,keywords:'slicer 슬라이서 필터 버튼',run:()=>void addSlicer().catch(error=>alert(error instanceof Error?error.message:'슬라이서를 추가하지 못했습니다.'))},
    {id:'cmd:connections',group:'명령',label:'데이터 연결',icon:<Table2/>,keywords:'connection importrange 연결 가져오기 새로 고침',run:()=>setRightPanel('connections')},
    {id:'cmd:column-stats',group:'명령',label:'열 통계',icon:<Table2/>,keywords:'stats 통계 요약 분포',run:()=>setRightPanel('stats')},
    {id:'cmd:protect',group:'명령',label:'시트·범위 보호',icon:<Table2/>,keywords:'protect 보호 잠금 권한',run:()=>setProtectedOpen(true)},
    {id:'cmd:cell-history',group:'명령',label:'편집 기록 표시',icon:<Table2/>,keywords:'history 이력 기록 변경',run:()=>setHistoryCell(address(editor.activeRow,editor.activeColumn))},
    {id:'cmd:dedupe',group:'명령',label:'중복 항목 삭제',icon:<Table2/>,keywords:'duplicate 중복 정리',run:()=>void openCleanup('duplicates')},
    {id:'cmd:trim',group:'명령',label:'공백 제거',icon:<Table2/>,keywords:'trim 공백 정리',run:()=>void openCleanup('trim')},
    {id:'cmd:inspect-table',group:'명령',label:'표 검사',icon:<Table2/>,keywords:'inspect 검사 정리 제안 점검 진단',run:()=>void openInspect()},
    {id:'cmd:consolidate',group:'명령',label:'여러 시트 합치기',icon:<Table2/>,keywords:'consolidate 통합 합치기 시트 합계',run:()=>void openConsolidate()},
    {id:'cmd:compare-lists',group:'명령',label:'두 목록 비교',icon:<Table2/>,keywords:'compare 비교 대사 목록 차이 reconcile',run:()=>void openCompare()},
    {id:'cmd:unmerge-fill',group:'명령',label:'병합 해제하고 채우기',icon:<Table2/>,keywords:'unmerge 병합 해제 채우기 merge fill',run:()=>void openCleanup('unmerge')},
    {id:'cmd:split',group:'명령',label:'텍스트를 열로 분할',icon:<Table2/>,keywords:'split 분할 열',run:()=>void openSplitDialog()},
    {id:'cmd:shortcuts',group:'명령',label:'단축키 목록',shortcut:'Ctrl+/',icon:<Search/>,keywords:'shortcut 단축키',run:()=>setShortcutsOpen(true)},
    ...(workbookList.data?.items??[]).filter(item=>item.id!==workbookId).map(item=>({
      id:`workbook:${item.id}`,group:'워크북',label:item.title,
      hint:item.access_role==='owner'?'내 소유':`${userLabel(item.owner_id,collaboratorDirectory)} 소유`,
      icon:<Grid2X2/>,keywords:'workbook 워크북 파일',run:()=>{window.location.href=`/workbooks/${item.id}`},
    })),
    {id:'nav:home',group:'이동',label:'워크북 목록',icon:<Grid2X2/>,keywords:'home 홈 목록',run:()=>{window.location.href='/'}},
    {id:'nav:preferences',group:'이동',label:'개인 환경설정',icon:<Settings/>,keywords:'preferences 설정',run:()=>{window.location.href='/preferences'}},
    ...(session?.admin?[{id:'nav:admin',group:'이동',label:'관리자 콘솔',icon:<Settings/>,keywords:'admin 관리자',run:()=>{window.location.href='/admin'}}]:[]),
  ]
  const activeError=explainFormulaError(formulaErrorCode(activeCell?.value)??'')
  /**
   * 값과 캐럿을 지금 맞춰 두고 다음 프레임에 한 번 더 맞추되, **그 사이에 값이
   * 바뀌지 않았을 때만** 되돌린다. 눌린 순간의 자리를 나중에 그대로 적용하면
   * 그 사이에 친 글자가 앞으로 끌려간다. 격자의 셀 편집기와 같은 이유다.
   */
  const settleFormulaCaret=(expected:string,start:number,end:number)=>{
    requestAnimationFrame(()=>{
      const field=formulaInput.current
      if(field&&field.value===expected)field.setSelectionRange(start,end)
    })
  }
  const formulaBarHint=editor.editing&&!readOnly?formulaHint(functionCatalog,editor.draft,formulaCaret):undefined
  const chooseFormulaSuggestion=(name:string)=>{
    if(!formulaBarHint)return
    const next=applySuggestion(editor.draft,formulaBarHint.context,name)
    editor.setDraft(next.text)
    setFormulaCaret(next.caret)
    setFormulaSuggestion(0)
    settleFormulaCaret(next.text,next.caret,next.caret)
  }
  const menus:WorkbookMenu[]=[
    {label:'파일',items:[
      {kind:'item',label:'저장',shortcut:'Ctrl+S',onSelect:()=>void saveWorkbook()},
      {kind:'submenu',label:'새로 만들기',items:[
        {kind:'item',label:'워크북',onSelect:()=>void createWorkbook()},
        {kind:'item',label:'시트',shortcut:'Shift+F11',disabled:!canWrite,onSelect:()=>void createSheet()},
      ]},
      {kind:'item',label:'사본 만들기',onSelect:()=>void duplicateWorkbook()},
      {kind:'item',label:'워크북 이름 변경…',disabled:!canWrite,onSelect:()=>void renameWorkbook()},
      {kind:'separator'},
      {kind:'item',label:'새 시트 추가',shortcut:'Shift+F11',onSelect:()=>void createSheet()},
      {kind:'item',label:'시트 복제',disabled:!canWrite,onSelect:()=>void duplicateSheet(activeSheet)},
      {kind:'item',label:'모든 시트 관리…',onSelect:()=>setSheetManagerOpen(true)},
      {kind:'item',label:'다른 워크북으로 시트 복사…',onSelect:()=>setCopySheet(activeSheet)},
      {kind:'separator'},
      {kind:'item',label:'XLSX로 내보내기',onSelect:()=>void exportWorkbook('xlsx')},
      {kind:'item',label:'현재 시트 CSV로 내보내기',onSelect:()=>void exportWorkbook('csv')},
      ...(handoffMenu.length>0?[{kind:'submenu' as const,label:'다른 서비스로 보내기',items:handoffMenu}]:[]),
      {kind:'item',label:'인쇄',shortcut:'Ctrl+P',onSelect:()=>void printSheet()},
      {kind:'item',label:'페이지 설정…',onSelect:()=>void openPrintOptions()},
      {kind:'item',label:'인쇄 영역으로 설정',onSelect:()=>void setPrintArea()},
      {kind:'item',label:'인쇄 영역 해제',disabled:!activeSheet?.layout?.print_area,onSelect:()=>void clearPrintArea()},
      {kind:'separator'},
      {kind:'item',label:'공유 설정…',onSelect:()=>setShareOpen(true)},
      {kind:'item',label:'버전 이력',onSelect:()=>setRightPanel('history')},
      {kind:'item',label:'워크북 목록으로',onSelect:()=>{window.location.href='/'}},
      {kind:'separator'},
      {kind:'item',label:'휴지통으로 이동',danger:true,disabled:workbook.data.access_role!=='owner',onSelect:()=>void trashWorkbook()},
    ]},
    {label:'수정',items:[
      {kind:'item',label:'실행 취소',shortcut:'Ctrl+Z',disabled:editor.undoStack.length===0,onSelect:()=>void revertOperation('undo')},
      {kind:'item',label:'다시 실행',shortcut:'Ctrl+Y',disabled:editor.redoStack.length===0,onSelect:()=>void revertOperation('redo')},
      {kind:'separator'},
      {kind:'item',label:'잘라내기',shortcut:'Ctrl+X',onSelect:()=>gridShortcut({command:'cut'})},
      {kind:'item',label:'복사',shortcut:'Ctrl+C',onSelect:()=>gridShortcut({command:'copy'})},
      {kind:'item',label:'붙여넣기',shortcut:'Ctrl+V',onSelect:()=>gridShortcut({command:'paste'})},
      {kind:'submenu',label:'특수 붙여넣기',items:[
        {kind:'item',label:'값만 붙여넣기',shortcut:'Ctrl+Shift+V',onSelect:()=>gridShortcut({command:'paste-values'})},
        {kind:'item',label:'서식만 붙여넣기',onSelect:()=>gridShortcut({command:'paste-special',mode:'format'})},
        {kind:'item',label:'행과 열 바꿔 붙여넣기',onSelect:()=>gridShortcut({command:'paste-special',mode:'transpose'})},
      ]},
      {kind:'separator'},
      {kind:'item',label:'내용 지우기',shortcut:'Delete',onSelect:()=>gridShortcut({command:'clear-contents'})},
      {kind:'item',label:'서식 지우기',shortcut:'Ctrl+\\',onSelect:()=>void clearFormat()},
      {kind:'separator'},
      {kind:'item',label:'찾기',shortcut:'Ctrl+F',onSelect:()=>openSearch(false)},
      {kind:'item',label:'찾기 및 바꾸기',shortcut:'Ctrl+H',onSelect:()=>openSearch(true)},
      {kind:'item',label:'전체 선택',shortcut:'Ctrl+A',onSelect:()=>gridShortcut({command:'select-all'})},
      {kind:'item',label:'데이터 영역 선택',shortcut:'Ctrl+Shift+A',onSelect:()=>gridShortcut({command:'select-data-region'})},
    ]},
    {label:'보기',items:[
      {kind:'item',label:'수식 표시',shortcut:'Ctrl+`',checked:showFormulas,onSelect:()=>setShowFormulas(current=>!current)},
      {kind:'item',label:'눈금선 표시',checked:showGridlines,onSelect:()=>setShowGridlines(current=>!current)},
      {kind:'item',label:'전체 화면',shortcut:'F11',onSelect:()=>toggleFullscreen()},
      {kind:'separator'},
      {kind:'item',label:'확대',onSelect:()=>editor.setZoom(editor.zoom+.1)},
      {kind:'item',label:'축소',onSelect:()=>editor.setZoom(editor.zoom-.1)},
      {kind:'item',label:'100%로 보기',onSelect:()=>editor.setZoom(1)},
      {kind:'separator'},
      {kind:'item',label:'활성 셀 앞까지 고정',onSelect:()=>void freezeToSelection()},
      {kind:'item',label:'고정 해제',disabled:(activeSheet.layout?.frozen_rows??0)===0&&(activeSheet.layout?.frozen_columns??0)===0,onSelect:()=>void applyLayout({action:'freeze',frozen_rows:0,frozen_columns:0})},
      {kind:'item',label:'모든 행 표시',onSelect:()=>void applyLayout({action:'show_all',axis:'row'})},
      {kind:'item',label:'모든 열 표시',onSelect:()=>void applyLayout({action:'show_all',axis:'column'})},
      {kind:'item',label:'현재 시트 숨기기',disabled:!canWrite||(workbook.data.sheets.filter(sheet=>!sheet.hidden).length<2),onSelect:()=>void setSheetHidden(activeSheet,true)},
      {kind:'separator'},
      {kind:'item',label:'시트 레이아웃…',onSelect:()=>setLayoutOpen(true)},
      {kind:'item',label:'단축키 목록',shortcut:'Ctrl+/',onSelect:()=>setShortcutsOpen(true)},
    ]},
    {label:'삽입',items:[
      {kind:'item',label:`위에 행 ${selectedRows}개 삽입`,onSelect:()=>void applyStructure({axis:'row',action:'insert',index:editorSelection.startRow,count:selectedRows})},
      {kind:'item',label:`아래에 행 ${selectedRows}개 삽입`,onSelect:()=>void applyStructure({axis:'row',action:'insert',index:editorSelection.endRow+1,count:selectedRows})},
      {kind:'item',label:`왼쪽에 열 ${selectedColumns}개 삽입`,onSelect:()=>void applyStructure({axis:'column',action:'insert',index:editorSelection.startColumn,count:selectedColumns})},
      {kind:'item',label:`오른쪽에 열 ${selectedColumns}개 삽입`,onSelect:()=>void applyStructure({axis:'column',action:'insert',index:editorSelection.endColumn+1,count:selectedColumns})},
      {kind:'item',label:'행과 열 관리…',onSelect:()=>setStructureOpen(true)},
      {kind:'separator'},
      {kind:'item',label:'차트…',disabled:!canWrite,onSelect:()=>setChartDialog(null)},
      {kind:'item',label:'이미지…',disabled:!canWrite,onSelect:()=>imageInput.current?.click()},
      {kind:'item',label:'피벗 테이블…',disabled:!canWrite,onSelect:()=>setPivotDialog(null)},
      {kind:'item',label:'댓글',shortcut:'Ctrl+Alt+M',disabled:!canComment,onSelect:()=>setRightPanel('comments')},
      {kind:'item',label:'이름 범위…',onSelect:()=>setNamedRangeOpen(true)},
      {kind:'item',label:'이름 있는 수식…',onSelect:()=>setNamedFunctionOpen(true)},
      {kind:'item',label:'표…',onSelect:()=>{void refreshSheetTables();setSheetTableOpen(true)}},
      {kind:'item',label:'변경 알림…',onSelect:()=>setWatchOpen(true)},
      {kind:'item',label:'자동 합계',shortcut:'Alt+=',onSelect:()=>gridShortcut({command:'auto-sum'})},
      {kind:'submenu',label:'함수',items:[
        ...['SUM','AVERAGE','COUNT','COUNTA','MAX','MIN','MEDIAN','PRODUCT'].map(name=>({kind:'item',label:name,onSelect:()=>gridShortcut({command:'insert-function',name})} as MenuItem)),
        {kind:'separator'},
        {kind:'item',label:'전체 함수 목록…',onSelect:()=>setFunctionsOpen(true)},
      ]},
      {kind:'item',label:'링크…',disabled:!canWrite,onSelect:()=>insertLink()},
      {kind:'item',label:'드롭다운…',disabled:!canWrite,onSelect:()=>setValidationOpen(true)},
      {kind:'separator'},
      {kind:'item',label:'오늘 날짜',shortcut:'Ctrl+;',onSelect:()=>gridShortcut({command:'insert-today'})},
      {kind:'item',label:'현재 시간',shortcut:'Ctrl+Shift+;',onSelect:()=>gridShortcut({command:'insert-now'})},
    ]},
    {label:'서식',items:[
      {kind:'item',label:'굵게',shortcut:'Ctrl+B',checked:activeCell?.style?.bold===true,onSelect:()=>void applyFormat({bold:activeCell?.style?.bold!==true})},
      {kind:'item',label:'기울임',shortcut:'Ctrl+I',checked:activeCell?.style?.italic===true,onSelect:()=>void applyFormat({italic:activeCell?.style?.italic!==true})},
      {kind:'item',label:'밑줄',shortcut:'Ctrl+U',checked:activeCell?.style?.underline===true,onSelect:()=>void applyFormat({underline:activeCell?.style?.underline!==true})},
      {kind:'item',label:'취소선',shortcut:'Alt+Shift+5',checked:activeCell?.style?.strike===true,onSelect:()=>void applyFormat({strike:activeCell?.style?.strike!==true})},
      {kind:'submenu',label:'표시 형식',items:numberFormatItems},
      {kind:'submenu',label:'맞춤',items:[
        {kind:'item',label:'왼쪽 정렬',shortcut:'Ctrl+Shift+L',onSelect:()=>void applyFormat({horizontal_align:'left'})},
        {kind:'item',label:'가운데 정렬',shortcut:'Ctrl+Shift+E',onSelect:()=>void applyFormat({horizontal_align:'center'})},
        {kind:'item',label:'오른쪽 정렬',shortcut:'Ctrl+Shift+R',onSelect:()=>void applyFormat({horizontal_align:'right'})},
        {kind:'item',label:'위쪽 맞춤',onSelect:()=>void applyFormat({vertical_align:'top'})},
        {kind:'item',label:'가운데 맞춤',onSelect:()=>void applyFormat({vertical_align:'middle'})},
        {kind:'item',label:'아래쪽 맞춤',onSelect:()=>void applyFormat({vertical_align:'bottom'})},
        {kind:'item',label:'자동 줄바꿈',onSelect:()=>void applyFormat({text_mode:'wrap'})},
        {kind:'item',label:'넘치게 표시',onSelect:()=>void applyFormat({text_mode:'overflow'})},
        {kind:'item',label:'잘라서 표시',onSelect:()=>void applyFormat({text_mode:'clip'})},
      ]},
      {kind:'separator'},
      {kind:'submenu',label:'글꼴 크기',items:[8,9,10,11,12,14,18,24,36].map(size=>({kind:'item',label:`${size}`,checked:activeCell?.style?.font_size===size,onSelect:()=>void applyFormat({font_size:size})} as MenuItem))},
      {kind:'submenu',label:'글꼴',items:['Inter','Pretendard','Malgun Gothic','Times New Roman','Courier New'].map(family=>({kind:'item',label:family,checked:activeCell?.style?.font_family===family,onSelect:()=>void applyFormat({font_family:family})} as MenuItem))},
      {kind:'submenu',label:'텍스트 색',items:TEXT_COLORS.map(color=>({kind:'item',label:color.label,checked:activeCell?.style?.color===color.value,onSelect:()=>void applyFormat({color:color.value})} as MenuItem))},
      {kind:'submenu',label:'채우기 색',items:FILL_COLORS.map(color=>({kind:'item',label:color.label,checked:activeCell?.style?.background===color.value,onSelect:()=>void applyFormat({background:color.value})} as MenuItem))},
      {kind:'submenu',label:'텍스트 회전',items:[{label:'없음',value:0},{label:'위로 45°',value:-45},{label:'아래로 45°',value:45},{label:'위로 90°',value:-90},{label:'아래로 90°',value:90}].map(option=>({kind:'item',label:option.label,checked:(activeCell?.style?.text_rotation??0)===option.value,onSelect:()=>void applyFormat({text_rotation:option.value})} as MenuItem))},
      {kind:'separator'},
      {kind:'item',label:mergedSelection?'셀 병합 해제':'셀 병합',onSelect:()=>void changeMerge(!mergedSelection)},
      {kind:'submenu',label:'테이블 서식',items:[
        ...TABLE_THEMES.map(theme=>({kind:'item',label:theme.name,checked:tableTheme===theme.id,onSelect:()=>void applyTableStyle(theme.id)} as MenuItem)),
        {kind:'separator'},
        {kind:'item',label:'테이블 서식 지우기',onSelect:()=>void clearTableStyle()},
      ]},
      {kind:'submenu',label:'테두리',items:[
        {kind:'item',label:'모든 테두리',onSelect:()=>void applyBorderPreset('all')},
        {kind:'item',label:'바깥쪽 테두리',onSelect:()=>void applyBorderPreset('outer')},
        {kind:'item',label:'안쪽 테두리',onSelect:()=>void applyBorderPreset('inner')},
        {kind:'item',label:'테두리 지우기',onSelect:()=>void applyBorderPreset('none')},
        {kind:'item',label:'테두리 설정…',onSelect:()=>setFormatOpen(true)},
      ]},
      {kind:'item',label:'조건부 서식…',onSelect:()=>setConditionalFormatOpen(true)},
      {kind:'item',label:'서식 세부 설정…',onSelect:()=>setFormatOpen(true)},
      {kind:'item',label:'서식 지우기',shortcut:'Ctrl+\\',onSelect:()=>void clearFormat()},
    ]},
    {label:'데이터',items:[
      {kind:'submenu',label:'정렬',items:[
        {kind:'item',label:'선택 열 기준 정렬 A → Z',disabled:!canWrite,onSelect:()=>void quickSort('asc')},
        {kind:'item',label:'선택 열 기준 정렬 Z → A',disabled:!canWrite,onSelect:()=>void quickSort('desc')},
        {kind:'item',label:'범위 정렬…',onSelect:()=>setSortOpen(true)},
      ]},
      {kind:'item',label:'필터 보기…',onSelect:()=>setFilterOpen(true)},
      {kind:'submenu',label:'가정 분석',items:[
        {kind:'item' as const,label:'목표값 찾기…',onSelect:()=>setGoalSeekOpen(true)},
        {kind:'item' as const,label:'데이터 표…',onSelect:()=>setDataTableOpen(true)},
        {kind:'item' as const,label:'시나리오…',onSelect:()=>{void refreshScenarios();setScenarioOpen(true)}},
      ]},
      ...(presentationEnabled?[{kind:'submenu' as const,label:'프레젠테이션',items:[
        {kind:'item' as const,label:'프레젠테이션 만들기…',onSelect:()=>setPresentationOpen(true)},
        {kind:'item' as const,label:'만든 프레젠테이션 목록',onSelect:()=>setRightPanel('presentations')},
      ]}]:[]),
      {kind:'item',label:'슬라이서 추가',disabled:!canWrite,onSelect:()=>void addSlicer().catch(error=>alert(error instanceof Error?error.message:'슬라이서를 추가하지 못했습니다.'))},
      {kind:'item',label:'데이터 검증…',onSelect:()=>setValidationOpen(true)},
      {kind:'item',label:'피벗 테이블…',onSelect:()=>setPivotDialog(null)},
      {kind:'item',label:'열 통계',onSelect:()=>setRightPanel('stats')},
      {kind:'item',label:'데이터 연결…',onSelect:()=>setRightPanel('connections')},
      {kind:'item',label:'범위 보호…',disabled:!canWrite,onSelect:()=>setProtectedOpen(true)},
      {kind:'item',label:'이름 범위…',onSelect:()=>setNamedRangeOpen(true)},
      {kind:'separator'},
      {kind:'submenu',label:'데이터 정리',disabled:!canWrite,items:[
        {kind:'item',label:'빠른 채우기…',onSelect:()=>void startFlashFill()},
        {kind:'item',label:'표 검사…',onSelect:()=>void openInspect()},
        {kind:'separator'},
        {kind:'item',label:'중복 항목 삭제…',onSelect:()=>void openCleanup('duplicates')},
        {kind:'item',label:'공백 제거…',onSelect:()=>void openCleanup('trim')},
        {kind:'item',label:'텍스트로 저장된 숫자…',onSelect:()=>void openCleanup('numbers')},
        {kind:'item',label:'병합 해제하고 채우기…',onSelect:()=>void openCleanup('unmerge')},
      ]},
      {kind:'item',label:'텍스트를 열로 분할…',disabled:!canWrite,onSelect:()=>void openSplitDialog()},
      {kind:'item',label:'부분합…',disabled:!canWrite,onSelect:()=>void openSubtotal()},
      {kind:'item',label:'부분합 제거…',disabled:!canWrite,onSelect:()=>void openCleanup('subtotals')},
      {kind:'separator'},
      {kind:'item',label:`선택 행 숨기기 (${selectedRows}개)`,shortcut:'Ctrl+Alt+9',onSelect:()=>void hideSelection('row')},
      {kind:'item',label:`선택 열 숨기기 (${selectedColumns}개)`,shortcut:'Ctrl+Alt+0',onSelect:()=>void hideSelection('column')},
      {kind:'separator'},
      {kind:'item',label:'두 목록 비교…',disabled:!canWrite,onSelect:()=>void openCompare()},
      {kind:'item',label:'여러 시트 합치기…',disabled:!canWrite,onSelect:()=>void openConsolidate()},
      {kind:'item',label:'워크북 검색',shortcut:'Ctrl+F',onSelect:()=>openSearch(false)},
    ]},
    {label:'도구',items:[
      {kind:'item',label:'Workbook Agent',checked:rightPanel==='ai',onSelect:()=>setRightPanel(current=>current==='ai'?null:'ai')},
      {kind:'item',label:'자동화',checked:rightPanel==='automation',disabled:!canWrite,onSelect:()=>setRightPanel(current=>current==='automation'?null:'automation')},
      ...(presentationEnabled?[{kind:'item' as const,label:'프레젠테이션 목록',checked:rightPanel==='presentations',onSelect:()=>setRightPanel(current=>current==='presentations'?null:'presentations')}]:[]),
      {kind:'item',label:'차트 패널',checked:rightPanel==='charts',onSelect:()=>setRightPanel(current=>current==='charts'?null:'charts')},
      {kind:'item',label:'피벗 패널',checked:rightPanel==='pivots',onSelect:()=>setRightPanel(current=>current==='pivots'?null:'pivots')},
      {kind:'item',label:'댓글',checked:rightPanel==='comments',onSelect:()=>setRightPanel(current=>current==='comments'?null:'comments')},
      {kind:'item',label:`편집 충돌${conflictCount>0?` (${conflictCount})`:''}`,checked:rightPanel==='conflicts',onSelect:()=>setRightPanel(current=>current==='conflicts'?null:'conflicts')},
      {kind:'separator'},
      {kind:'item',label:'개인 환경설정',onSelect:()=>{window.location.href='/preferences'}},
    ]},
    {label:'도움말',items:[
      {kind:'item',label:'함수 목록',onSelect:()=>setFunctionsOpen(true)},
      {kind:'item',label:'단축키 목록',shortcut:'Ctrl+/',onSelect:()=>setShortcutsOpen(true)},
      {kind:'label',label:`kanpic ${build?.version??''}`.trim()},
    ]},
  ]
  return <div className="editor-shell">
    {/* 도구 모음을 다 지나야 시트에 닿는다. 키보드만 쓰는 사람에게 마흔 번
        넘는 탭을 요구할 수는 없다. 이 링크는 초점을 받을 때만 보인다. */}
    <a className="skip-link" href="#kanpic-grid" onClick={event=>{event.preventDefault();document.getElementById('kanpic-grid')?.focus()}}>시트로 건너뛰기</a>
    <AppHeader build={build} session={session}><div className="editor-title"><a href="/" aria-label="뒤로"><ChevronLeft/></a><div><strong>{workbook.data.title}</strong><small className={conflictCount>0?'interactive':''} role={conflictCount>0?'button':undefined} tabIndex={conflictCount>0?0:undefined} onClick={conflictCount>0?()=>setRightPanel('conflicts'):undefined} onKeyDown={conflictCount>0?event=>{if(event.key==='Enter'||event.key===' '){event.preventDefault();setRightPanel('conflicts')}}:undefined}><span className={`save-dot ${displaySaveState}`}/>{saveLabel} · v{serverVersion}{handoffOrigin.data&&<span className="handoff-origin" title={handoffOriginDetail(handoffOrigin.data)}>· {handoffOriginLabel(handoffOrigin.data)}</span>}</small></div>{accessRole!=='owner'&&<span className={`access-badge ${accessRole}`} title={accessSummary({role:accessRole,source:workbook.data.access_source,source_label:workbook.data.access_source})}>{accessRole==='viewer'?<><Eye/> 보기 전용</>:accessRole==='commenter'?<><MessageCircle/> 댓글 가능</>:<>편집자</>}</span>}</div></AppHeader>
    <div className="editor-actions"><WorkbookMenuBar menus={menus}/><div className="share-actions"><span className={`collaboration-count ${collaborationStatus}`} title={Object.values(collaborators).map(user=>userLabel(user.actor_id??'',collaboratorDirectory)).join(', ')}><i/>{collaborationStatus==='connected'?`${Object.keys(collaborators).length}명 접속`:collaborationStatus==='offline'?'오프라인':'재연결 중'}</span><button className="ghost" onClick={()=>exportWorkbook('xlsx')} title="XLSX로 내보내기"><Download/> XLSX 내보내기</button><button className="ghost" onClick={()=>exportWorkbook('csv')} title="현재 시트를 CSV로 내보내기">CSV</button><button className="primary" onClick={()=>setShareOpen(true)}><Share2/> 공유{(workbook.data.shared_count??0)>0?` ${workbook.data.shared_count}`:''}</button></div></div>
    <div className="toolbar"><button aria-label="실행 취소" title="실행 취소" disabled={readOnly||editor.undoStack.length===0} onClick={()=>revertOperation('undo')}><Undo2/></button><button aria-label="다시 실행" title="다시 실행" disabled={readOnly||editor.redoStack.length===0} onClick={()=>revertOperation('redo')}><Redo2/></button><button aria-label="워크북 검색" title="워크북 검색 (Ctrl/⌘+F)" onClick={()=>setSearchOpen(true)}><Search/></button><span className="divider"/><select aria-label="글꼴" className="toolbar-select font-family" disabled={readOnly} value={typeof activeCell?.style?.font_family==='string'?activeCell.style.font_family:'Inter'} onChange={event=>void applyFormat({font_family:event.target.value})}><option>Inter</option><option>Pretendard</option><option>Arial</option><option>Georgia</option><option>monospace</option></select><select aria-label="글꼴 크기" className="toolbar-select font-size" disabled={readOnly} value={typeof activeCell?.style?.font_size==='number'?activeCell.style.font_size:12} onChange={event=>void applyFormat({font_size:Number(event.target.value)})}>{[8,9,10,11,12,14,16,18,20,24,28,32].map(size=><option key={size}>{size}</option>)}</select><button aria-label="굵게" title="굵게" disabled={readOnly} className={activeCell?.style?.bold===true?'active':''} onClick={()=>void applyFormat({bold:activeCell?.style?.bold!==true})}><Bold/></button><button aria-label="기울임" title="기울임" disabled={readOnly} className={activeCell?.style?.italic===true?'active':''} onClick={()=>void applyFormat({italic:activeCell?.style?.italic!==true})}><Italic/></button><button aria-label="밑줄" title="밑줄" disabled={readOnly} className={activeCell?.style?.underline===true?'active':''} onClick={()=>void applyFormat({underline:activeCell?.style?.underline!==true})}><Underline/></button><label className="toolbar-color" title="글자색"><span>A</span><input aria-label="글자색" type="color" disabled={readOnly} value={typeof activeCell?.style?.color==='string'?activeCell.style.color:'#1c2733'} onChange={event=>void applyFormat({color:event.target.value})}/></label><label className="toolbar-color background" title="셀 배경색"><span>▰</span><input aria-label="셀 배경색" type="color" disabled={readOnly} value={typeof activeCell?.style?.background==='string'?activeCell.style.background:'#ffffff'} onChange={event=>void applyFormat({background:event.target.value})}/></label><span className="divider"/><button aria-label="왼쪽 정렬" title="왼쪽 정렬" disabled={readOnly} className={activeCell?.style?.horizontal_align==='left'?'active':''} onClick={()=>void applyFormat({horizontal_align:'left'})}><AlignLeft/></button><button aria-label="가운데 정렬" title="가운데 정렬" disabled={readOnly} className={activeCell?.style?.horizontal_align==='center'?'active':''} onClick={()=>void applyFormat({horizontal_align:'center'})}><AlignCenter/></button><button aria-label="오른쪽 정렬" title="오른쪽 정렬" disabled={readOnly} className={activeCell?.style?.horizontal_align==='right'?'active':''} onClick={()=>void applyFormat({horizontal_align:'right'})}><AlignRight/></button><button aria-label={mergedSelection?'병합 해제':'셀 병합'} title={mergedSelection?'병합 해제':'셀 병합'} disabled={readOnly} className={mergedSelection?'active':''} onClick={()=>void changeMerge(!mergedSelection)}>{mergedSelection?<TableCellsSplit/>:<TableCellsMerge/>}</button><button aria-label="서식 복사" title={formatBrush?'서식 복사 중 · Esc로 취소':'서식 복사 (두 번 클릭하면 계속 사용)'} disabled={readOnly} className={formatBrush?'active':''} onClick={()=>{if(formatBrush)setFormatBrush(undefined);else pickUpFormat(false)}} onDoubleClick={()=>pickUpFormat(true)}><Paintbrush/></button><button aria-label="테이블 서식" title="테이블 서식" disabled={readOnly} aria-haspopup="menu" aria-expanded={Boolean(tableMenu)} onClick={event=>{const rect=event.currentTarget.getBoundingClientRect();setTableMenu(current=>current?undefined:{x:rect.left,y:rect.bottom+4})}}><Table/></button><button aria-label="테두리" title="테두리" disabled={readOnly} aria-haspopup="menu" aria-expanded={Boolean(borderMenu)} onClick={event=>{const rect=event.currentTarget.getBoundingClientRect();setBorderMenu(current=>current?undefined:{x:rect.left,y:rect.bottom+4})}}><Grid3X3/></button><button aria-label="범위 정렬" title="범위 정렬" disabled={readOnly} onClick={()=>setSortOpen(true)}><ArrowUpDown/></button><button aria-label="필터 보기" title="필터 보기" className={activeFilter?'active':''} onClick={()=>setFilterOpen(true)}><Filter/></button><button aria-label="데이터 검증" title="데이터 검증" disabled={readOnly} className={(validations.data?.items.length??0)>0?'active':''} onClick={()=>setValidationOpen(true)}><BadgeCheck/></button><button aria-label="조건부 서식" title="조건부 서식" disabled={readOnly} className={(conditionalFormats.data?.items.length??0)>0?'active':''} onClick={()=>setConditionalFormatOpen(true)}><Palette/></button><button aria-label="선택 범위 링크 복사" title="선택 범위 링크 복사" onClick={()=>void copySelectionLink()}><Link2/></button><span className="toolbar-spacer"/><button aria-label="축소" title="축소" onClick={()=>editor.setZoom(editor.zoom-.1)}><ZoomOut/></button><span className="zoom-value" aria-live="polite">{Math.round(editor.zoom*100)}%</span><button aria-label="확대" title="확대" onClick={()=>editor.setZoom(editor.zoom+.1)}><ZoomIn/></button><button aria-label="차트 패널" title="차트 패널" onClick={()=>setRightPanel(current=>current==='charts'?null:'charts')} className={rightPanel==='charts'?'active':''}><BarChart3/></button><button aria-label="피벗 패널" title="피벗 패널" onClick={()=>setRightPanel(current=>current==='pivots'?null:'pivots')} className={rightPanel==='pivots'?'active':''}><Table2/></button>{canWrite&&<button aria-label="자동화 패널" title="자동화 패널" onClick={()=>setRightPanel(current=>current==='automation'?null:'automation')} className={rightPanel==='automation'?'active':''}><Workflow/></button>}{presentationEnabled&&<button aria-label="프레젠테이션 패널" title="만든 프레젠테이션 목록" onClick={()=>setRightPanel(current=>current==='presentations'?null:'presentations')} className={rightPanel==='presentations'?'active':''}><Presentation/></button>}<span className="divider"/><button aria-label="AI 도우미" title="Workbook Agent" onClick={()=>setRightPanel(current=>current==='ai'?null:'ai')} className={rightPanel==='ai'?'active':''}><Bot/></button><button aria-label="편집 충돌" title={`편집 충돌 ${conflictCount}건`} onClick={()=>setRightPanel(current=>current==='conflicts'?null:'conflicts')} className={rightPanel==='conflicts'||conflictCount>0?'active':''}><AlertTriangle/></button><button aria-label="버전 이력" title="버전 이력" onClick={()=>setRightPanel(current=>current==='history'?null:'history')} className={rightPanel==='history'?'active':''}><History/></button><button aria-label="댓글" title="댓글" onClick={()=>setRightPanel(current=>current==='comments'?null:'comments')} className={rightPanel==='comments'?'active':''}><MessageSquare/></button><button aria-label="추가 도구" title="추가 도구" aria-haspopup="menu" aria-expanded={Boolean(overflowMenu)} onClick={event=>{const rect=event.currentTarget.getBoundingClientRect();setOverflowMenu(current=>current?undefined:{x:Math.max(8,rect.right-252),y:rect.bottom+4})}}><MoreHorizontal/></button></div>
    {overflowMenu&&<ContextMenu x={overflowMenu.x} y={overflowMenu.y} label="추가 도구 메뉴" onClose={()=>setOverflowMenu(undefined)} items={[
      {kind:'item',label:'수식 표시',shortcut:'Ctrl+`',checked:showFormulas,onSelect:()=>setShowFormulas(current=>!current)},
      {kind:'item',label:'서식 지우기',shortcut:'Ctrl+\\',onSelect:()=>void clearFormat()},
      {kind:'separator'},
      {kind:'item',label:'차트 패널',checked:rightPanel==='charts',onSelect:()=>setRightPanel(current=>current==='charts'?null:'charts')},
      {kind:'item',label:'피벗 패널',checked:rightPanel==='pivots',onSelect:()=>setRightPanel(current=>current==='pivots'?null:'pivots')},
      {kind:'item',label:'자동화',checked:rightPanel==='automation',disabled:!canWrite,onSelect:()=>setRightPanel(current=>current==='automation'?null:'automation')},
      ...(presentationEnabled?[{kind:'item' as const,label:'프레젠테이션 목록',checked:rightPanel==='presentations',onSelect:()=>setRightPanel(current=>current==='presentations'?null:'presentations')}]:[]),
      {kind:'separator'},
      {kind:'item',label:'찾기 및 바꾸기',shortcut:'Ctrl+H',onSelect:()=>openSearch(true)},
      {kind:'item',label:'단축키 목록',shortcut:'Ctrl+/',onSelect:()=>setShortcutsOpen(true)},
    ]}/>}
    <div className="formula-bar"><form onSubmit={event=>{event.preventDefault();submitNameBox()}}><input className="name-box" ref={nameBoxRef} aria-label="이름 상자" list="named-range-options" value={nameBoxValue} onChange={event=>setNameBoxValue(event.target.value)} onBlur={()=>{if(!nameBoxValue.trim())setNameBoxValue(selectionAddress)}}/><datalist id="named-range-options">{(namedRanges.data?.items??[]).map(item=><option key={item.id} value={item.name}>{item.range}</option>)}</datalist></form><button className="named-range-trigger" aria-label="이름 범위 관리" title="이름 범위 관리" onClick={()=>setNamedRangeOpen(true)}><Link2/></button><span>fx</span><textarea className="formula-input" rows={1} spellCheck={false} aria-label="수식 입력창" ref={formulaInput} value={editor.editing?editor.draft:formula} readOnly={readOnly}
      onSelect={event=>setFormulaCaret(event.currentTarget.selectionStart??0)}
      onFocus={()=>{
        if(readOnly||editor.editing)return
        const source=activeCell?.spill_source&&parseCellAddress(activeCell.spill_source)
        if(source)editor.select(source.row,source.column)
        else{editor.setDraft(formula);editor.setEditing(true)}
      }}
      onChange={event=>{if(!readOnly){setFormulaCaret(event.target.selectionStart??event.target.value.length);setFormulaSuggestion(0);editor.setDraft(event.target.value);editor.setEditing(true)}}}
      onKeyDown={event=>{
        if(event.nativeEvent.isComposing)return
        // 제안 목록이 떠 있으면 방향키와 Tab·Enter는 목록의 것이다. 셀 안에서와
        // 같은 조작이어야 어디에 쓰든 손이 헷갈리지 않는다.
        const suggestions=formulaBarHint?.matches??[]
        if(suggestions.length>0&&formulaSuggestion>=0){
          if(event.key==='ArrowDown'){event.preventDefault();setFormulaSuggestion((formulaSuggestion+1)%suggestions.length);return}
          if(event.key==='ArrowUp'){event.preventDefault();setFormulaSuggestion((formulaSuggestion-1+suggestions.length)%suggestions.length);return}
          if(event.key==='Tab'||(event.key==='Enter'&&!(event.ctrlKey||event.metaKey))){event.preventDefault();chooseFormulaSuggestion(suggestions[formulaSuggestion].name);return}
          if(event.key==='Escape'){event.preventDefault();setFormulaSuggestion(-1);return}
        }
        // F4 로 참조 고정을 돌리는 것도 셀 안에서와 같아야 한다.
        if(event.key==='F4'){
          const field=event.currentTarget
          const cycled=cycleReference(field.value,field.selectionStart??field.value.length,field.selectionEnd??field.value.length)
          if(cycled){
            event.preventDefault()
            editor.setDraft(cycled.text);editor.setEditing(true);setFormulaCaret(cycled.end)
            field.value=cycled.text;field.setSelectionRange(cycled.start,cycled.end)
            settleFormulaCaret(cycled.text,cycled.start,cycled.end)
          }
          return
        }
        // Alt+Enter writes a line break here too, so a multi-line cell can be
        // edited from the formula bar without losing its breaks.
        if(event.key==='Enter'&&event.altKey){
          event.preventDefault()
          const field=event.currentTarget
          const at=field.selectionStart??editor.draft.length,to=field.selectionEnd??at
          const next=editor.draft.slice(0,at)+'\n'+editor.draft.slice(to)
          editor.setDraft(next);editor.setEditing(true)
          // 다음 프레임에만 캐럿을 옮기면 그 사이에 친 글자가 이전 자리로 간다.
          field.value=next;field.setSelectionRange(at+1,at+1)
          settleFormulaCaret(next,at+1,at+1)
          return
        }
        if(event.key==='Enter'){event.preventDefault();gridShortcut({command:'commit-draft'})}
        else if(event.key==='Escape'){event.preventDefault();editor.setEditing(false);gridShortcut({command:'focus-grid'})}
      }}/>
      {formulaBarHint&&formulaSuggestion>=0&&<FormulaAutocomplete hint={formulaBarHint} active={formulaSuggestion} left={88} top={31} onChoose={chooseFormulaSuggestion}/>}
      {/* 오류는 마우스를 올려야만 설명을 볼 수 있으면 안 된다. 키보드로 셀을
          옮겨 다니는 사람에게는 이 자리가 유일한 안내다. */}
      {!editor.editing&&activeError&&<span className="formula-error" title={activeError.next}><strong>{activeError.code}</strong> {activeError.summary}</span>}</div>
    <div className="editor-body"><div className="sheet-area"><CanvasGrid sheetId={activeSheet.id} workbookId={workbookId} onPasteImage={file=>void uploadImage(file)} layout={activeSheet.layout} version={serverVersion} onVersion={updateVersion} hiddenRows={filterResult.data?.hidden_rows??[]} validations={validations.data?.items??[]} conditionalFormats={conditionalFormats.data?.items??[]} tables={sheetTables.data?.items??[]} filterView={activeFilter} formatBrush={Boolean(formatBrush)} onPaintFormat={range=>void paintFormat(range)} showFormulas={showFormulas} showGridlines={showGridlines} readOnly={readOnly} userLabels={collaboratorLabels} onLayout={applyLayout} onStructure={applyStructure} onMenuCommand={handleGridMenu} onOpenRange={navigateToRange} onResolveNumericRun={resolveNumericRun}/><SheetTabs sheets={workbook.data.sheets} activeSheetId={activeSheet.id} version={serverVersion} saveState={displaySaveState} saveLabel={activeFilter&&filterResult.data?`${saveLabel} · 필터 ${filterResult.data.visible_count.toLocaleString()}행` :saveLabel} onStatusClick={conflictCount>0?()=>setRightPanel('conflicts'):undefined} onSelect={setActiveSheet} onCreate={createSheet} onRename={(sheet,name)=>updateSheet(sheet,{name})} onDuplicate={duplicateSheet} onMove={(sheet,position)=>updateSheet(sheet,{position})} onColor={(sheet,color)=>updateSheet(sheet,{color})} onHidden={setSheetHidden} onDelete={deleteSheet} readOnly={readOnly} onManage={()=>setSheetManagerOpen(true)} onCopyTo={sheet=>setCopySheet(sheet)}/></div>
      {rightPanel&&<ResizableRightPanel key={rightPanel} panelKey={rightPanel}>
        {rightPanel==='ai'&&<AIPanel key={agentDraft.key} workbookId={workbookId} workbookName={workbook.data.title} sheetId={activeSheet.id} sheetName={activeSheet.name} selectionRange={selectionAddress} baseVersion={serverVersion} initialMode={agentDraft.mode} initialRequest={agentDraft.request} onClose={()=>setRightPanel(null)} onExecuted={handleAIExecuted}/>}
        {rightPanel==='automation'&&<AutomationPanel workbookId={workbookId} workbookVersion={serverVersion} sheets={workbook.data.sheets} activeSheetId={activeSheet.id} selectionRange={selectionAddress} prepareExecution={prepareAutomationExecution} onClose={()=>setRightPanel(null)} onExecuted={handleAutomationExecuted}/>}
        {rightPanel==='connections'&&<ConnectionPanel workbookId={workbookId} version={serverVersion} readOnly={readOnly} onClose={()=>setRightPanel(null)} onRefreshed={handleConnectionsRefreshed}/>}
        {rightPanel==='stats'&&<ColumnStatsPanel workbookId={workbookId} workbookVersion={serverVersion} sheetId={activeSheet.id} column={editor.activeColumn} onClose={()=>setRightPanel(null)}/>}
        {rightPanel==='history'&&<VersionPanel workbookId={workbookId} currentVersion={serverVersion} onClose={()=>setRightPanel(null)} onRestored={handleRestored}/>}
        {rightPanel==='comments'&&<CommentPanel workbookId={workbookId} sheetId={activeSheet.id} selectionRange={selectionAddress} currentActor={session?.user?.id??'local-user'} focusThreadId={routeNavigation.commentId||undefined} onNavigate={navigateToRange} onClose={()=>setRightPanel(null)}/>}
        {rightPanel==='conflicts'&&<ConflictPanel workbookId={workbookId} sheets={workbook.data.sheets} currentActor={session?.user?.id??'local-user'} onClose={()=>setRightPanel(null)} onNavigate={navigateToRange} onResolved={handleConflictResolved}/>}
        {rightPanel==='presentations'&&<PresentationPanel items={presentationList.data?.items??[]} sheetNames={new Map(workbook.data.sheets.map(sheet=>[sheet.id,sheet.name]))} onClose={()=>setRightPanel(null)} onCreate={()=>setPresentationOpen(true)} onRefresh={refreshPresentation} onDownload={record=>downloadPresentation(record.id)}/>}
        {rightPanel==='charts'&&<ChartPanel charts={charts.data?.items??[]} sheets={workbook.data.sheets} onClose={()=>setRightPanel(null)} onCreate={()=>setChartDialog(null)} onEdit={item=>setChartDialog(item)} onNavigate={item=>{if(item.source_sheet_id&&item.source_range!=='#REF!')navigateToRange(item.source_sheet_id,item.source_range)}}/>}
        {rightPanel==='pivots'&&<PivotPanel pivots={pivots.data?.items??[]} sheets={workbook.data.sheets} onClose={()=>setRightPanel(null)} onCreate={()=>setPivotDialog(null)} onEdit={item=>setPivotDialog(item)} onOpen={setPivotResult} onRefresh={refreshPivot} onNavigate={item=>{if(item.source_sheet_id&&item.source_range!=='#REF!')navigateToRange(item.source_sheet_id,item.source_range)}}/>}
      </ResizableRightPanel>}
    </div>
    <StalledSaveNotice count={stalled.length}
      onRetry={()=>{void retryOutbox(stalled).then(()=>{setStalled([]);editor.setSaveState('saving');return flushOutbox(outboxApplied(activeSheet?.id))}).then(refreshStalled)}}
      onDiscard={()=>{if(!confirm(`저장하지 못한 변경 ${stalled.length.toLocaleString()}건을 버립니다. 이 변경은 사라지고 서버에 있는 값이 남습니다.`))return;void discardOutbox(stalled).then(()=>{setStalled([]);editor.setSaveState('saved');window.dispatchEvent(new CustomEvent('kanpic:outbox-discarded'));return client.invalidateQueries({queryKey:['workbook',workbookId]})})}}/>
    <FormulaIssueNotice issues={editor.formulaIssues} dropped={editor.droppedCells} automations={editor.automationFailures} unmerged={editor.unmergedRanges} backup={editor.editBackup} onClose={()=>editor.clearFormulaIssues()}
      onRevert={async backup=>{
        // 행·열 삭제는 서버가 셀 단위로 되돌리지 못한다. 삭제 직전에 남겨 둔
        // 자동 백업으로 복원하는 것이 유일한 회수 경로다.
        if(!confirm(`${backup.summary} 삭제를 되돌릴까요? 삭제 직전 상태로 복원되며 현재 상태는 자동 백업됩니다.`))return
        editor.setSaveState('saving')
        try{
          const result=await api<MutationResult>(`/api/v1/versions/${backup.versionId}:restore`,{method:'POST',body:'{}'})
          await handleRestored(result)
          editor.clearFormulaIssues()
          editor.setSaveState('saved')
        }catch(error){editor.setSaveState('error');alert(error instanceof Error?error.message:'되돌리지 못했습니다.')}
      }}
      onOpen={issue=>{
        // 오류가 다른 시트에 생겼을 수도 있다. 그 자리로 데려가지 않으면
        // 몇 곳인지 아는 것만으로는 고칠 수가 없다.
        const target=issue.sheet_id&&issue.sheet_id!==activeSheet.id?issue.sheet_id:activeSheet.id
        navigateToRange(target,address(issue.row,issue.column))
        editor.clearFormulaIssues()
      }}/>
    <SlicerOverlay slicers={activeSheet.layout?.slicers??[]} views={filterViews.data?.items??[]} sheetId={activeSheet.id} version={serverVersion} readOnly={readOnly}
      onApply={async(view,criteria)=>{await updateFilter(view.id,{criteria})}} onUpdate={updateSlicer} onRemove={removeSlicer}/>
    <input ref={imageInput} type="file" hidden accept="image/png,image/jpeg,image/gif,image/webp" onChange={event=>{const file=event.target.files?.[0];event.target.value='';if(file)void uploadImage(file)}}/>
    <ImageOverlay images={images.data?.items??[]} readOnly={!canWrite} onUpdate={updateImage} onDelete={deleteImage}/>
    <ChartOverlay charts={charts.data?.items??[]} version={serverVersion} onEdit={item=>setChartDialog(item)} onUpdate={updateChart} onDelete={deleteChart} onNavigate={item=>{if(item.source_sheet_id&&item.source_range!=='#REF!')navigateToRange(item.source_sheet_id,item.source_range)}}/>
    {chartDialog!==undefined&&<ChartDialog chart={chartDialog??undefined} activeSheetId={activeSheet.id} selectionRange={selectionAddress} sheets={workbook.data.sheets} onClose={()=>setChartDialog(undefined)} onCreate={createChart} onUpdate={updateChart} onDelete={deleteChart}/>}
    {pivotDialog!==undefined&&<PivotDialog pivot={pivotDialog??undefined} activeSheetId={activeSheet.id} selectionRange={selectionAddress} sheets={workbook.data.sheets} onClose={()=>setPivotDialog(undefined)} onCreate={createPivot} onUpdate={updatePivot} onDelete={deletePivot}/>}
    {pivotResult&&<PivotResultDialog pivot={pivotResult} version={serverVersion} onClose={()=>setPivotResult(undefined)} onRefresh={refreshPivot}/>}
    {sortOpen&&<SortDialog range={editorSelection} onClose={()=>setSortOpen(false)} onSort={sortSelection}/>}
    {structureOpen&&<StructureDialog range={editorSelection} onClose={()=>setStructureOpen(false)} onApply={applyStructure}/>}
    {layoutOpen&&<LayoutDialog range={editorSelection} layout={activeSheet.layout} onClose={()=>setLayoutOpen(false)} onApply={applyLayout}/>}
    {columnFilter&&activeFilter&&<ColumnFilterMenu view={activeFilter} sheetId={activeSheet.id} version={serverVersion} column={columnFilter.column}
      label={address(1,columnFilter.column).replace(/\d+$/,'')} x={columnFilter.x} y={columnFilter.y}
      onClose={()=>setColumnFilter(undefined)}
      onSort={direction=>void sortColumn(columnFilter.column,direction)}
      onApply={async criteria=>{await updateFilter(activeFilter.id,{criteria})}}/>}
    {protectedOpen&&activeSheet&&<ProtectedRangeDialog range={editorSelection} rules={protections.data?.items??[]} onClose={()=>setProtectedOpen(false)} onCreate={createProtection} onDelete={deleteProtection}/>}
    {subtotal&&<SubtotalDialog region={subtotal.region} cells={subtotal.cells} headerRows={subtotal.headerRows} occupiedBelow={subtotal.occupiedBelow}
      onClose={()=>setSubtotal(undefined)} onApply={applySubtotals}/>}
    {sortScope&&<SortScopeDialog column={sortScope.column} direction={sortScope.direction} block={sortScope.block} selection={sortScope.selection}
      onClose={()=>setSortScope(undefined)} onSort={(region,cells)=>runSort(region,cells,sortScope.column,sortScope.direction)}/>}
    {cleanup&&<CleanupDialog mode={cleanup.mode} target={cleanup.target} onClose={()=>setCleanup(undefined)}
      onApply={(region,cells,headerRows)=>applyCleanup(cleanup.mode,region,cells,headerRows)}/>}
    {splitTarget&&<SplitDialog cells={splitTarget.cells} region={splitTarget.region} onClose={()=>setSplitTarget(undefined)}
      onApply={delimiter=>splitColumn(delimiter,splitTarget)}/>}
    {inspect&&<InspectDialog cells={inspect.cells} region={inspect.region} headerRows={inspect.headerRows} onClose={()=>setInspect(undefined)} onFix={fixFromInspect}/>}
    {consolidateSources&&<ConsolidateDialog sources={consolidateSources} skipped={consolidateSkipped} onClose={()=>setConsolidateSources(undefined)} onApply={writeConsolidated}/>}
    {compare&&<CompareDialog left={compare.left} right={compare.right} onClose={()=>setCompare(undefined)} onReport={writeCompareReport}/>}
    {prompt&&<PromptDialog request={prompt} onClose={()=>setPrompt(undefined)}/>}
    {linkOpen&&activeSheet&&workbook.data&&<LinkDialog workbookId={workbookId} sheets={workbook.data.sheets} activeSheetId={activeSheet.id} selectionRange={selectionAddress}
      onClose={()=>setLinkOpen(false)} onApply={formula=>{
        // The dialog already asked everything it needs, so the link lands in the
        // cell rather than leaving a half-typed formula in the editor.
        gridShortcut({command:'commit-text',text:formula})
      }}/>}
    {historyCell&&activeSheet&&<CellHistoryDialog sheetId={activeSheet.id} address={historyCell} version={serverVersion} onClose={()=>setHistoryCell(undefined)}/>}
    {noteOpen&&<NoteDialog address={selectionAddress} note={editor.cells.get(cellKey(editor.activeRow,editor.activeColumn))?.note??''} onClose={()=>setNoteOpen(false)} onApply={applyNote}/>}
    {formatOpen&&<FormatDialog style={activeCell?.style} onClose={()=>setFormatOpen(false)} onApply={applyFormat}/>}
    {filterOpen&&<FilterDialog range={editorSelection} views={filterViews.data?.items??[]} result={filterResult.data} onClose={()=>setFilterOpen(false)} onCreate={createFilter} onUpdate={updateFilter} onDelete={deleteFilter}/>} 
    {validationOpen&&<DataValidationDialog range={editorSelection} rules={validations.data?.items??[]} onClose={()=>setValidationOpen(false)} onCreate={createValidation} onUpdate={updateValidation} onDelete={deleteValidation} onEvaluate={evaluateValidation}/>} 
    {flashFill&&<FlashFillDialog plan={flashFill.plan} column={flashFill.column} onClose={()=>setFlashFill(undefined)}
      onApply={async plan=>{await writeCells(plan.writes.map(write=>({row:write.row,column:plan.column,value:write.value})))}}/>}
    {scenarioOpen&&activeSheet&&<ScenarioDialog
      scenarios={scenarios.data?.items??[]}
      activeSheetId={activeSheet.id}
      defaultTarget={address(editorSelection.startRow,editorSelection.startColumn)}
      onClose={()=>setScenarioOpen(false)}
      onCreate={createScenario} onUpdate={updateScenario} onDelete={deleteScenario}
      onCompare={async targets=>(await api<{result:ScenarioComparison}>(`/api/v1/sheets/${activeSheet.id}/scenarios:compare`,{method:'POST',body:JSON.stringify({targets})})).result}/>}
    {dataTableOpen&&activeSheet&&<DataTableDialog
      defaultTarget={address(editorSelection.startRow,editorSelection.startColumn)}
      onClose={()=>setDataTableOpen(false)}
      onCompute={async input=>(await api<{result:DataTableOutcome}>(`/api/v1/sheets/${activeSheet.id}/data-table`,{method:'POST',body:JSON.stringify({
        target:input.target,column_input:input.columnInput,column_values:input.columnValues,
        row_input:input.rowInput,row_values:input.rowValues,
      })})).result}/>}
    {goalSeekOpen&&activeSheet&&<GoalSeekDialog
      defaultTarget={address(editorSelection.startRow,editorSelection.startColumn)}
      canWrite={canWrite}
      onClose={()=>setGoalSeekOpen(false)}
      onSeek={async input=>(await api<{result:GoalSeekOutcome}>(`/api/v1/sheets/${activeSheet.id}/goal-seek`,{method:'POST',body:JSON.stringify(input)})).result}
      onApply={async(cell,value)=>{
        const parsed=parseFilterRange(`${cell}:${cell}`)
        if(!parsed)throw new Error('바꿀 셀 주소가 올바르지 않습니다.')
        await writeCells([{row:parsed.startRow,column:parsed.startColumn,value}])
      }}/>}
    {presentationOpen&&activeSheet&&<PresentationDialog range={workingRegion()} onClose={()=>setPresentationOpen(false)} onPreview={previewPresentation} onCreate={createPresentation} onLoadTemplates={loadPresentationTemplates} onDownload={downloadPresentation}/>}
    {conditionalFormatOpen&&<ConditionalFormatDialog range={editorSelection} rules={conditionalFormats.data?.items??[]} onClose={()=>setConditionalFormatOpen(false)} onCreate={createConditionalFormat} onUpdate={updateConditionalFormat} onDelete={deleteConditionalFormat}/>}
    {watchOpen&&activeSheet&&<WatchRuleDialog selection={editorSelection} sheets={workbook.data.sheets} activeSheetId={activeSheet.id} rules={watchRules.data?.items??[]} onClose={()=>setWatchOpen(false)} onCreate={createWatchRule} onUpdate={updateWatchRule} onDelete={deleteWatchRule}/>}
    {printOpen&&<PrintOptionsDialog onClose={()=>setPrintOpen(false)} region={printOpen.region} truncated={printOpen.truncated}
      columnWidth={column=>printColumnWidths.get(column)??108}
      onPrint={choice=>{setPrintOpen(false);void printSheet(choice)}}/>}
    {namedFunctionOpen&&<NamedFunctionDialog functions={namedFunctions.data?.items??[]} onClose={()=>setNamedFunctionOpen(false)} onCreate={createNamedFunction} onUpdate={updateNamedFunction} onDelete={deleteNamedFunction}/>}
    {sheetTableOpen&&<SheetTableDialog selection={editorSelection} activeSheetId={activeSheet.id} sheets={workbook.data.sheets} tables={sheetTables.data?.items??[]} onClose={()=>setSheetTableOpen(false)} onCreate={createSheetTable} onUpdate={updateSheetTable} onDelete={deleteSheetTable} onRefresh={refreshSheetTables} onNavigate={item=>{navigateToRange(item.sheet_id,item.range);setSheetTableOpen(false)}}/>}
    {namedRangeOpen&&<NamedRangeDialog selection={editorSelection} activeSheetId={activeSheet.id} sheets={workbook.data.sheets} ranges={namedRanges.data?.items??[]} onClose={()=>setNamedRangeOpen(false)} onCreate={createNamedRange} onUpdate={updateNamedRange} onDelete={deleteNamedRange} onNavigate={item=>{navigateToRange(item.sheet_id,item.range);setNamedRangeOpen(false)}}/>}
    {sheetManagerOpen&&<SheetManagerDialog workbook={workbook.data} sheets={workbook.data.sheets} activeSheetId={activeSheet.id} readOnly={readOnly} onClose={()=>setSheetManagerOpen(false)} onSelect={setActiveSheet} onRename={(sheet,name)=>updateSheet(sheet,{name})} onMove={(sheet,position)=>updateSheet(sheet,{position})} onHidden={setSheetHidden} onDelete={deleteSheet} onCopyTo={sheet=>{setSheetManagerOpen(false);setCopySheet(sheet)}}/>}
    {copySheet&&<CopySheetDialog sheet={copySheet} workbookId={workbookId} onClose={()=>setCopySheet(undefined)} onCopied={target=>{
      setCopySheet(undefined)
      void client.invalidateQueries({queryKey:['workbooks']})
      if(target.id===workbookId)void refreshWorkbook()
      else if(window.confirm(`'${target.title}' 워크북으로 복사했습니다. 지금 이동할까요?`))window.location.href=`/workbooks/${target.id}`
    }}/>}
    {quickOpen&&<QuickSwitcher items={quickItems} onClose={()=>setQuickOpen(false)} onQuery={setQuickQuery} dynamicItems={query=>{
      const found=(workbookSearch.data?.items??[])
        .filter(item=>item.id!==workbookId&&!(workbookList.data?.items??[]).some(loaded=>loaded.id===item.id))
        .map(item=>({id:`found:${item.id}`,group:'워크북 검색',label:item.title,hint:'이름으로 찾은 워크북',
          icon:<Grid2X2/>,run:()=>{window.location.href=`/workbooks/${item.id}`}}))
      const target=parseNavigationRange(query)
      if(!target)return found
      return [{id:'address',group:'셀 이동',label:query.trim().toUpperCase(),hint:`${activeSheet.name} 시트의 이 범위로 이동`,icon:<Search/>,run:()=>navigateToRange(activeSheet.id,query)},...found]
    }}/>}
    {shareOpen&&<ShareDialog workbook={workbook.data} onClose={()=>setShareOpen(false)} onChanged={()=>{void client.invalidateQueries({queryKey:['workbook',workbookId]})}}/>}
    {shortcutsOpen&&<WorkbookShortcutsDialog onClose={()=>setShortcutsOpen(false)}/>}
    {tableMenu&&<ContextMenu x={tableMenu.x} y={tableMenu.y} label="테이블 서식 메뉴" onClose={()=>setTableMenu(undefined)} items={[
      {kind:'label',label:`테이블 서식 · ${address(workingRegion().startRow,workingRegion().startColumn)}:${address(workingRegion().endRow,workingRegion().endColumn)}`},
      ...TABLE_THEMES.map(theme=>({kind:'item',label:theme.name,checked:tableTheme===theme.id,icon:<i className="table-swatch" style={{background:theme.header||theme.outline}}/>,onSelect:()=>void applyTableStyle(theme.id)} as MenuItem)),
      {kind:'separator'},
      {kind:'submenu',label:'옵션',icon:<Settings/>,items:[
        {kind:'item',label:'머리글 행',checked:tableOptions.headerRow,onSelect:()=>setTableOptions(current=>({...current,headerRow:!current.headerRow}))},
        {kind:'item',label:'줄무늬 행',checked:tableOptions.bandedRows,onSelect:()=>setTableOptions(current=>({...current,bandedRows:!current.bandedRows}))},
        {kind:'item',label:'테두리',checked:tableOptions.borders,onSelect:()=>setTableOptions(current=>({...current,borders:!current.borders}))},
        {kind:'item',label:'합계 행 강조',checked:tableOptions.totalRow,onSelect:()=>setTableOptions(current=>({...current,totalRow:!current.totalRow}))},
      ]},
      {kind:'item',label:'테이블 서식 지우기',icon:<Square/>,onSelect:()=>void clearTableStyle()},
    ]}/>}
    {borderMenu&&<ContextMenu x={borderMenu.x} y={borderMenu.y} label="테두리 메뉴" onClose={()=>setBorderMenu(undefined)} items={[
      {kind:'item',label:'모든 테두리',icon:<Grid3X3/>,onSelect:()=>void applyBorderPreset('all')},
      {kind:'item',label:'바깥쪽 테두리',icon:<Square/>,onSelect:()=>void applyBorderPreset('outer')},
      {kind:'item',label:'안쪽 테두리',onSelect:()=>void applyBorderPreset('inner')},
      {kind:'separator'},
      {kind:'item',label:'위쪽',onSelect:()=>void applyBorderPreset('top')},
      {kind:'item',label:'아래쪽',onSelect:()=>void applyBorderPreset('bottom')},
      {kind:'item',label:'왼쪽',onSelect:()=>void applyBorderPreset('left')},
      {kind:'item',label:'오른쪽',onSelect:()=>void applyBorderPreset('right')},
      {kind:'item',label:'안쪽 가로',onSelect:()=>void applyBorderPreset('horizontal')},
      {kind:'item',label:'안쪽 세로',onSelect:()=>void applyBorderPreset('vertical')},
      {kind:'separator'},
      {kind:'submenu',label:'선 색',icon:<Palette/>,items:[
        {label:'회색',value:'#94a3b8'},{label:'검정',value:'#1c2733'},{label:'청록',value:'#0f766e'},{label:'파랑',value:'#2563eb'},{label:'빨강',value:'#dc2626'},
      ].map(color=>({kind:'item',label:color.label,checked:borderColor===color.value,icon:<i className="table-swatch" style={{background:color.value}}/>,onSelect:()=>setBorderColor(color.value)} as MenuItem))},
      {kind:'item',label:'테두리 지우기',danger:true,onSelect:()=>void applyBorderPreset('none')},
      {kind:'item',label:'테두리 설정…',onSelect:()=>setFormatOpen(true)},
    ]}/>}
    {functionsOpen&&<FunctionListDialog onClose={()=>setFunctionsOpen(false)} onInsert={name=>gridShortcut({command:'insert-function',name})}/>}
    <WorkbookSearchDialog open={searchOpen} workbookId={workbookId} version={serverVersion} sheetId={activeSheet.id} sheetName={activeSheet.name} replaceMode={replaceMode} onClose={()=>{setSearchOpen(false);setReplaceMode(false)}} onNavigate={(item:WorkbookSearchMatch)=>navigateToRange(item.sheet_id,item.address)} onReplaced={result=>void handleReplaced(result)}/>
  </div>
}
