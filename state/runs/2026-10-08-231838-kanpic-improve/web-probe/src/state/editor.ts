import { create } from 'zustand'
import type { Cell, MutationResult } from '../types'

type SaveState = 'saved'|'saving'|'offline'|'conflict'|'error'
type EditorState = {
  activeRow:number
  activeColumn:number
  anchorRow:number
  anchorColumn:number
  editing:boolean
  /** Text being edited, shared by the grid editor and the formula bar. */
  draft:string
  zoom:number
  cells:Map<string,Cell>
  saveState:SaveState
  conflicts:number
  /**
   * 방금 한 편집이 만든 수식 오류. 행을 지워 다른 곳의 수식이 깨져도
   * 지금까지 아무 말이 없었다. 서버는 이미 알려 주고 있었다.
   */
  formulaIssues:MutationResult['formula_errors']
  /** 방금 저장한 편집이 실행시킨 자동화 중 실패한 것. 셀 변경 트리거는
   *  아무도 지켜보지 않으므로 편집자에게 그 자리에서 알려야 한다. */
  automationFailures:NonNullable<MutationResult['automation_failures']>
  /** 방금 한 편집이 풀어 버린 병합. 붙여넣기 아래로 조용히 사라지지 않게 말한다. */
  unmergedRanges:string[]
  /**
   * 되돌릴 수 없는 편집 직전에 서버가 남긴 자동 백업. 행·열 삭제는 실행
   * 취소로 되살릴 수 없어서, 이 버전으로 복원하는 것이 유일한 회수 경로다.
   */
  editBackup?:{versionId:string;summary:string}
  undoStack:string[]
  redoStack:string[]
  select:(row:number,column:number,extend?:boolean)=>void
  setEditing:(editing:boolean)=>void
  setDraft:(draft:string)=>void
  setZoom:(zoom:number)=>void
  putCells:(cells:Cell[])=>void
  /**
   * 화면에 들고 있던 셀만 버립니다. 다른 사람이 행을 지우면 주소가 밀려서
   * 다시 읽어야 하지만, 그렇다고 내 선택 위치와 입력 중이던 값까지 버릴
   * 이유는 없습니다. 전체 초기화는 편집을 끝내 버려 입력이 사라집니다.
   */
  clearCells:()=>void
  /**
   * 선택이 옮겨간 뒤에도 살려 둘 입력. 그리드는 선택이 바뀌면 입력창을 그
   * 셀의 값으로 다시 채우는데, 다른 사람이 행을 지워 내 자리가 밀린 경우에는
   * 치고 있던 값이 그대로 따라가야 한다.
   */
  carriedDraft?:{row:number;column:number;text:string}
  carryDraft:(carried:{row:number;column:number;text:string}|undefined)=>void
  replaceRange:(cells:Cell[],startRow:number,startColumn:number,endRow:number,endColumn:number)=>void
  putCell:(cell:Cell)=>void
  setSaveState:(saveState:SaveState,conflicts?:number)=>void
  /** 방금 한 편집이 다른 사람 때문에 자리를 잃은 경우 그 좌표. */
  droppedCells:NonNullable<MutationResult['dropped_cells']>
  reportEdit:(result:Pick<MutationResult,'formula_errors'|'dropped_cells'|'automation_failures'|'unmerged_ranges'>)=>void
  reportRecoverableEdit:(backup:{versionId:string;summary:string}|undefined,errors?:MutationResult['formula_errors'])=>void
  clearFormulaIssues:()=>void
  recordOperation:(operationId:string)=>void
  takeUndo:()=>string|undefined
  restoreUndo:(operationId:string)=>void
  completeUndo:(undoOperationId:string)=>void
  takeRedo:()=>string|undefined
  restoreRedo:(operationId:string)=>void
  completeRedo:(redoOperationId:string)=>void
  reset:()=>void
}

const key=(row:number,column:number)=>`${row}:${column}`

export const useEditorStore=create<EditorState>((set,get)=>({
  activeRow:1,activeColumn:1,anchorRow:1,anchorColumn:1,editing:false,draft:'',zoom:1,cells:new Map(),saveState:'saved',conflicts:0,formulaIssues:[],droppedCells:[],automationFailures:[],unmergedRanges:[],editBackup:undefined,undoStack:[],redoStack:[],
  select:(activeRow,activeColumn,extend=false)=>set((state)=>({activeRow,activeColumn,anchorRow:extend?state.anchorRow:activeRow,anchorColumn:extend?state.anchorColumn:activeColumn,editing:false})),
  setEditing:(editing)=>set({editing}),
  setDraft:(draft)=>set({draft}),
  setZoom:(zoom)=>set({zoom:Math.max(.5,Math.min(2,zoom))}),
  putCells:(cells)=>set((state)=>{const next=new Map(state.cells);cells.forEach((cell)=>next.set(key(cell.row,cell.column),cell));return{cells:next}}),
  replaceRange:(incoming,startRow,startColumn,endRow,endColumn)=>set((state)=>{const cells=new Map(state.cells);for(const [address,cell] of cells){if(cell.row>=startRow&&cell.row<=endRow&&cell.column>=startColumn&&cell.column<=endColumn)cells.delete(address)}incoming.forEach(cell=>cells.set(key(cell.row,cell.column),cell));return{cells}}),
  putCell:(cell)=>set((state)=>{const cells=new Map(state.cells);cells.set(key(cell.row,cell.column),cell);return{cells}}),
  clearCells:()=>set({cells:new Map()}),
  carryDraft:(carriedDraft)=>set({carriedDraft}),
  setSaveState:(saveState,conflicts=0)=>set({saveState,conflicts}),
  // 오류가 없는 편집은 이전 안내를 지운다. 고친 뒤에도 경고가 남아 있으면
  // 무엇이 지금 문제인지 알 수 없다.
  reportEdit:(result)=>set({formulaIssues:result.formula_errors?.length?result.formula_errors:[],droppedCells:result.dropped_cells?.length?result.dropped_cells:[],automationFailures:result.automation_failures?.length?result.automation_failures:[],unmergedRanges:result.unmerged_ranges?.length?result.unmerged_ranges:[],editBackup:undefined}),
  reportRecoverableEdit:(editBackup,errors)=>set({formulaIssues:errors&&errors.length>0?errors:[],droppedCells:[],automationFailures:[],unmergedRanges:[],editBackup}),
  clearFormulaIssues:()=>set({formulaIssues:[],droppedCells:[],automationFailures:[],unmergedRanges:[],editBackup:undefined}),
  recordOperation:(operationId)=>{if(!operationId||get().undoStack.at(-1)===operationId)return;set((state)=>({undoStack:[...state.undoStack,operationId].slice(-100),redoStack:[]}))},
  takeUndo:()=>{const stack=get().undoStack;if(stack.length===0)return;const operationId=stack[stack.length-1];set({undoStack:stack.slice(0,-1)});return operationId},
  restoreUndo:(operationId)=>set((state)=>({undoStack:[...state.undoStack,operationId].slice(-100)})),
  completeUndo:(undoOperationId)=>set((state)=>({redoStack:[...state.redoStack,undoOperationId].slice(-100)})),
  takeRedo:()=>{const stack=get().redoStack;if(stack.length===0)return;const operationId=stack[stack.length-1];set({redoStack:stack.slice(0,-1)});return operationId},
  restoreRedo:(operationId)=>set((state)=>({redoStack:[...state.redoStack,operationId].slice(-100)})),
  completeRedo:(redoOperationId)=>set((state)=>({undoStack:[...state.undoStack,redoOperationId].slice(-100)})),
  // reset은 시트를 옮기거나 워크북을 다시 읽을 때도 불린다. 방금 한 편집이
  // 남긴 안내는 그때 사라지면 안 된다. 시트를 옮겼다고 지운 시트를 되돌릴
  // 길이 없어지는 것은 아니기 때문이다.
  reset:()=>set({activeRow:1,activeColumn:1,anchorRow:1,anchorColumn:1,editing:false,draft:'',cells:new Map(),saveState:'saved',conflicts:0,undoStack:[],redoStack:[]}),
}))

export const cellKey=key
export function selectedBounds(state:Pick<EditorState,'activeRow'|'activeColumn'|'anchorRow'|'anchorColumn'>){return{startRow:Math.min(state.anchorRow,state.activeRow),startColumn:Math.min(state.anchorColumn,state.activeColumn),endRow:Math.max(state.anchorRow,state.activeRow),endColumn:Math.max(state.anchorColumn,state.activeColumn)}}
