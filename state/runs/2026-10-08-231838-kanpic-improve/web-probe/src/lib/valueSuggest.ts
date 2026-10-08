import type { Cell } from '../types'

/** How many suggestions are worth showing under a cell. */
const MAX_SUGGESTIONS=6

/** How far up and down the column to look for existing entries. */
const SCAN_LIMIT=500

/**
 * Finds the text already used in this column that starts with what is being
 * typed. Repeating a value is the most common thing anybody does in a
 * spreadsheet column, so the existing entries are the best suggestions there
 * are — and they cost nothing, because the rows are already loaded.
 */
export function suggestColumnValues(cells:Map<string,Cell>,column:number,row:number,draft:string){
  const typed=draft.trim()
  if(typed===''||typed.startsWith('='))return []
  const needle=typed.toLowerCase()
  const seen=new Map<string,number>()
  for(const cell of cells.values()){
    if(cell.column!==column||cell.row===row)continue
    if(Math.abs(cell.row-row)>SCAN_LIMIT)continue
    // A formula's result changes with its inputs, so only typed text is worth
    // repeating; numbers are quicker to type than to pick from a list.
    if(cell.formula||typeof cell.value!=='string')continue
    const value=cell.value.trim()
    if(value===''||value.toLowerCase()===needle||!value.toLowerCase().startsWith(needle))continue
    seen.set(value,(seen.get(value)??0)+1)
  }
  return [...seen.entries()]
    // The value used most often comes first; ties keep alphabetical order so
    // the list does not jump around as the sheet fills up.
    .sort((first,second)=>second[1]-first[1]||first[0].localeCompare(second[0],'ko-KR'))
    .slice(0,MAX_SUGGESTIONS)
    .map(([value])=>value)
}
