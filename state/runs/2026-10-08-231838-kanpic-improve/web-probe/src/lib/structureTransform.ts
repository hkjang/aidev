export type StructuralChange={
  axis:'row'|'column'
  action:'insert'|'delete'|'move'
  index:number
  count:number
  destination?:number
}

/**
 * 다른 사람이 행이나 열을 넣거나 지우면 그 아래의 모든 주소가 밀립니다.
 * 내 선택도 같이 밀려야 보고 있던 데이터를 계속 보게 됩니다. 서버가 늦게
 * 도착한 편집의 주소를 옮기는 것과 같은 규칙입니다.
 *
 * 지워진 자리를 가리키고 있었다면 갈 곳이 없으므로 그 자리를 물려받은
 * 위치를 돌려줍니다. 화면이 맨 위로 튀는 것보다 낫습니다.
 */
export function transformPosition(position:number,change:StructuralChange):number{
  if(change.action==='move')return movedPosition(position,change)
  if(change.action==='insert')return position>=change.index?position+change.count:position
  const end=change.index+change.count-1
  if(position>=change.index&&position<=end)return change.index
  return position>end?position-change.count:position
}

/** 옮기기는 자리바꿈이라 사라지는 위치가 없습니다. */
function movedPosition(position:number,change:StructuralChange){
  const destination=change.destination??change.index
  const end=change.index+change.count-1
  const landing=destination>change.index?destination-change.count:destination
  if(position>=change.index&&position<=end)return position-change.index+landing
  if(position<change.index&&position>=landing)return position+change.count
  if(position>end&&position<landing+change.count)return position-change.count
  return position
}

/**
 * 그 자리가 살아남았는지 알려 줍니다. 선택은 지워진 자리에서도 근처에
 * 머무르면 되지만, 입력 중이던 값은 다릅니다. 겨냥한 셀이 사라졌는데 그
 * 자리를 물려받은 셀에 값을 쓰면 남의 데이터를 덮어씁니다.
 */
export function survivesChange(row:number,column:number,change:StructuralChange){
  if(change.action!=='delete')return true
  const position=change.axis==='column'?column:row
  return position<change.index||position>change.index+change.count-1
}

/** 선택 영역 전체를 옮깁니다. 축에 맞는 좌표만 움직입니다. */
export function transformSelection(
  selection:{activeRow:number;activeColumn:number;anchorRow:number;anchorColumn:number},
  change:StructuralChange,
){
  if(change.axis==='column')return {
    ...selection,
    activeColumn:transformPosition(selection.activeColumn,change),
    anchorColumn:transformPosition(selection.anchorColumn,change),
  }
  return {
    ...selection,
    activeRow:transformPosition(selection.activeRow,change),
    anchorRow:transformPosition(selection.anchorRow,change),
  }
}
