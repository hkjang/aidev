import type { Cell } from '../types'
import type { PastedCell } from './clipboard'
import type { GridRegion } from './dataRegion'
import { cellKey } from '../state/editor'

export type TableTheme={
  id:string
  name:string
  header:string
  headerText:string
  band:string
  outline:string
  inner:string
}

/** Table looks, kept close to the palette the rest of the product uses. */
export const TABLE_THEMES:TableTheme[]=[
  {id:'teal',name:'청록',header:'#0f766e',headerText:'#ffffff',band:'#f2f8f7',outline:'#0f766e',inner:'#cfe3e0'},
  {id:'blue',name:'파랑',header:'#2563eb',headerText:'#ffffff',band:'#f1f5fe',outline:'#2563eb',inner:'#d3dffb'},
  {id:'slate',name:'회색',header:'#475569',headerText:'#ffffff',band:'#f4f6f8',outline:'#475569',inner:'#d9dee4'},
  {id:'amber',name:'주황',header:'#b45309',headerText:'#ffffff',band:'#fdf5e9',outline:'#b45309',inner:'#f0dcc0'},
  {id:'plain',name:'테두리만',header:'',headerText:'',band:'',outline:'#94a3b8',inner:'#dde3e8'},
]

export type TableStyleOptions={
  headerRow:boolean
  bandedRows:boolean
  borders:boolean
  totalRow:boolean
}

export const DEFAULT_TABLE_OPTIONS:TableStyleOptions={headerRow:true,bandedRows:true,borders:true,totalRow:false}

function mergeStyle(style:Record<string,unknown>|undefined,patch:Record<string,unknown>){
  const merged={...(style??{})}
  for(const [key,value] of Object.entries(patch)){
    if(value===null)delete merged[key]
    else merged[key]=value
  }
  return merged
}

const side=(color:string)=>({style:'thin',color})

/**
 * Builds the cells that turn a range into a formatted table: a header band,
 * alternating body rows, an outlined edge with lighter inner rules and an
 * optional emphasised total row. Values and formulas are carried through
 * untouched so applying a look never disturbs the data.
 */
export function tableStyleCells(cells:Map<string,Cell>,region:GridRegion,theme:TableTheme,options:TableStyleOptions):PastedCell[]{
  const writes:PastedCell[]=[]
  for(let row=region.startRow;row<=region.endRow;row+=1){
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      const current=cells.get(cellKey(row,column))
      const header=options.headerRow&&row===region.startRow&&theme.header!==''
      const total=options.totalRow&&row===region.endRow&&!header
      const bodyIndex=row-region.startRow-(options.headerRow?1:0)
      const banded=options.bandedRows&&!header&&!total&&theme.band!==''&&bodyIndex%2===1
      const patch:Record<string,unknown>={
        background:header?theme.header:total?theme.band||null:banded?theme.band:null,
        color:header?theme.headerText:null,
        bold:header||total?true:null,
        horizontal_align:header?'center':null,
      }
      if(options.borders){
        patch.borders={
          top:row===region.startRow?side(theme.outline):side(header||total?theme.outline:theme.inner),
          bottom:row===region.endRow?side(theme.outline):null,
          left:column===region.startColumn?side(theme.outline):side(theme.inner),
          right:column===region.endColumn?side(theme.outline):null,
        }
      }
      writes.push({
        row,column,
        value:current?.formula?undefined:current?.value,
        formula:current?.formula,
        style:mergeStyle(current?.style,patch),
      })
    }
  }
  return writes
}

/** Removes the look a table style applied and leaves the data alone. */
export function clearTableStyleCells(cells:Map<string,Cell>,region:GridRegion):PastedCell[]{
  const writes:PastedCell[]=[]
  for(let row=region.startRow;row<=region.endRow;row+=1){
    for(let column=region.startColumn;column<=region.endColumn;column+=1){
      const current=cells.get(cellKey(row,column))
      writes.push({
        row,column,
        value:current?.formula?undefined:current?.value,
        formula:current?.formula,
        style:mergeStyle(current?.style,{background:null,color:null,bold:null,borders:null,horizontal_align:null}),
      })
    }
  }
  return writes
}
