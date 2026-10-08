import { Trash2 } from 'lucide-react'
import { useEffect,useState } from 'react'
import { createPortal } from 'react-dom'
import type { ChartPosition,SheetImage } from '../types'
import './ImageOverlay.css'

/**
 * 시트 위에 떠 있는 그림. 차트와 같은 층, 같은 픽셀 좌표, 같은 손짓이다 —
 * 머리띠를 끌면 옮기고 오른쪽 아래 모서리를 끌면 크기를 바꾼다. 그림의 본체는
 * 자기 주소에서 오므로 목록은 가볍다.
 */
function ImageCard({image,readOnly,onUpdate,onDelete}:{image:SheetImage;readOnly:boolean;onUpdate:(image:SheetImage,input:Record<string,unknown>)=>Promise<SheetImage>;onDelete:(image:SheetImage)=>Promise<void>}){
  const [position,setPosition]=useState<ChartPosition>(image.position)
  const [error,setError]=useState('')
  useEffect(()=>setPosition(image.position),[image.position.x,image.position.y,image.position.width,image.position.height])
  const ratio=image.natural_height/Math.max(1,image.natural_width)
  const startGesture=(kind:'move'|'resize',event:React.PointerEvent)=>{
    if(readOnly)return
    event.preventDefault();event.stopPropagation()
    const startX=event.clientX,startY=event.clientY,start=position;let latest=start
    const move=(next:PointerEvent)=>{
      const dx=next.clientX-startX,dy=next.clientY-startY
      if(kind==='move')latest={...start,x:Math.round(Math.max(0,start.x+dx)),y:Math.round(Math.max(0,start.y+dy))}
      else{
        // 크기는 가로만 따라가고 세로는 그림의 비율을 지킨다. 찌그러진 사진은 아무도 원하지 않는다.
        const width=Math.round(Math.max(16,Math.min(4000,start.width+dx)))
        latest={...start,width,height:Math.round(Math.max(16,Math.min(4000,width*ratio)))}
      }
      setPosition(latest)
    }
    const stop=()=>{
      window.removeEventListener('pointermove',move);window.removeEventListener('pointerup',stop)
      if(JSON.stringify(latest)!==JSON.stringify(image.position))void onUpdate(image,{position:latest,expected_revision:image.revision}).catch(reason=>{setError(reason instanceof Error?reason.message:'그림을 옮기지 못했습니다.');setPosition(image.position)})
    }
    window.addEventListener('pointermove',move);window.addEventListener('pointerup',stop)
  }
  return <figure className="sheet-image" style={{left:position.x,top:position.y,width:position.width,height:position.height}} data-image-id={image.id} title={error||undefined}>
    <div className="sheet-image-bar" onPointerDown={event=>startGesture('move',event)}>
      <span>{image.natural_width}×{image.natural_height}</span>
      {!readOnly&&<button aria-label="그림 삭제" title="그림 삭제" onPointerDown={event=>event.stopPropagation()} onClick={()=>{if(confirm('이 그림을 지울까요?'))void onDelete(image)}}><Trash2/></button>}
    </div>
    <img src={`/api/v1/images/${image.id}/content?r=${image.revision}`} alt="" draggable={false}/>
    {!readOnly&&<span className="sheet-image-resize" aria-label="그림 크기 조절" onPointerDown={event=>startGesture('resize',event)}/>}
  </figure>
}

export function ImageOverlay({images,readOnly,onUpdate,onDelete}:{images:SheetImage[];readOnly:boolean;onUpdate:(image:SheetImage,input:Record<string,unknown>)=>Promise<SheetImage>;onDelete:(image:SheetImage)=>Promise<void>}){
  const [target,setTarget]=useState<Element|null>(null)
  useEffect(()=>setTarget(document.querySelector('.sheet-area')),[images.length])
  if(!target||images.length===0)return null
  return createPortal(<div className="image-overlay-layer" aria-label="시트 그림 레이어">{images.map(image=><ImageCard key={image.id} image={image} readOnly={readOnly} onUpdate={onUpdate} onDelete={onDelete}/>)}</div>,target)
}
