import type { DimensionGroup } from '../types'

/** How far the outline gutter grows for each level of nesting. */
export const OUTLINE_STEP=11

/**
 * The rows or columns a collapsed group folds away. Collapsing keeps its own
 * state on the group rather than writing into the hidden ranges, so expanding
 * never reveals something the user had hidden by hand.
 */
export function collapsedIndexes(groups:DimensionGroup[]|undefined){
  const hidden=new Set<number>()
  for(const group of groups??[]){
    if(!group.collapsed)continue
    for(let index=group.start;index<=group.end;index+=1)hidden.add(index)
  }
  return hidden
}

/** How wide the gutter has to be to draw every level of this sheet's outline. */
export function outlineSize(groups:DimensionGroup[]|undefined){
  if(!groups||groups.length===0)return 0
  return (Math.max(...groups.map(group=>group.depth))+1)*OUTLINE_STEP+4
}

export type OutlineControl={group:DimensionGroup;offset:number}

/**
 * The control for a group sits one step past the end of its range, which is
 * where a spreadsheet puts the +/- box, and stays visible when the group is
 * collapsed because its own rows are gone.
 */
export function controlIndexFor(group:DimensionGroup){return group.end+1}

/** The groups whose bracket passes through this row or column. */
export function groupsAt(groups:DimensionGroup[]|undefined,index:number){
  return (groups??[]).filter(group=>index>=group.start&&index<=controlIndexFor(group))
}

/** The group whose control box is drawn at this row or column, if any. */
export function controlAt(groups:DimensionGroup[]|undefined,index:number,depth:number){
  return (groups??[]).find(group=>group.depth===depth&&controlIndexFor(group)===index)
}

/**
 * The innermost group covering a selection, which is what collapse, expand and
 * ungroup act on.
 */
export function innermostGroup(groups:DimensionGroup[]|undefined,start:number,end:number){
  let best:DimensionGroup|undefined
  for(const group of groups??[]){
    if(group.start>start||group.end<end)continue
    if(!best||group.end-group.start<best.end-best.start)best=group
  }
  return best
}
