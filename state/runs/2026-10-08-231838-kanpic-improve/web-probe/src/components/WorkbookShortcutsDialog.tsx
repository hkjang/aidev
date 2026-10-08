import { useEffect } from 'react'
import { useDialog } from '../lib/useDialog'

type Shortcut = { action:string; keys:string[] }

const sections:Array<{title:string; shortcuts:Shortcut[]}> = [
  {title:'기본 작업',shortcuts:[
    {action:'저장',keys:['Ctrl / ⌘ + S']},
    {action:'실행 취소',keys:['Ctrl / ⌘ + Z']},
    {action:'다시 실행',keys:['Ctrl / ⌘ + Y','Ctrl / ⌘ + Shift + Z']},
    {action:'빠른 이동 (시트·워크북·명령)',keys:['Ctrl / ⌘ + K']},
    {action:'워크북 검색',keys:['Ctrl / ⌘ + F']},
    {action:'찾기 및 바꾸기',keys:['Ctrl / ⌘ + H']},
    {action:'빠른 채우기',keys:['Ctrl / ⌘ + E']},
    {action:'단축키 목록',keys:['Ctrl / ⌘ + /']},
    {action:'새 시트 추가',keys:['Shift + F11']},
    {action:'이전 / 다음 시트로 이동',keys:['Ctrl / ⌘ + PageUp','Ctrl / ⌘ + PageDown']},
    {action:'댓글 패널 열기',keys:['Ctrl / ⌘ + Alt + M']},
  ]},
  {title:'셀 편집과 채우기',shortcuts:[
    {action:'셀 편집 시작',keys:['Enter','F2']},
    {action:'입력 완료 후 이동',keys:['Enter (아래)','Shift + Enter (위)','Tab (오른쪽)','Shift + Tab (왼쪽)']},
    {action:'편집 취소',keys:['Esc']},
    {action:'내용 삭제',keys:['Backspace','Delete']},
    {action:'복사 / 잘라내기 / 붙여넣기',keys:['Ctrl / ⌘ + C','Ctrl / ⌘ + X','Ctrl / ⌘ + V']},
    {action:'값만 붙여넣기',keys:['Ctrl / ⌘ + Shift + V']},
    {action:'선택 범위에 입력값 채우기',keys:['Ctrl / ⌘ + Enter']},
    {action:'아래 / 오른쪽 채우기',keys:['Ctrl / ⌘ + D','Ctrl / ⌘ + R']},
    {action:'자동 합계',keys:['Alt + =']},
    {action:'오늘 날짜 / 현재 시간 입력',keys:['Ctrl / ⌘ + ;','Ctrl / ⌘ + Shift + ;']},
  ]},
  {title:'행과 열',shortcuts:[
    {action:'행·열 삽입 / 삭제 열기',keys:['Ctrl / ⌘ + Alt + =','Ctrl / ⌘ + Alt + -']},
    {action:'선택 행 / 열 숨기기',keys:['Ctrl / ⌘ + Alt + 9','Ctrl / ⌘ + Alt + 0']},
    {action:'숨긴 행 / 열 모두 표시',keys:['Ctrl / ⌘ + Shift + 9','Ctrl / ⌘ + Shift + 0']},
    {action:'머리글 경계 드래그',keys:['행 높이·열 너비 조절']},
    {action:'머리글 경계 더블클릭',keys:['자동 맞춤']},
    {action:'컨텍스트 메뉴 열기',keys:['마우스 오른쪽 클릭','Shift + F10']},
    {action:'대화상자 닫기',keys:['Esc']},
  ]},
  {title:'선택과 탐색',shortcuts:[
    {action:'셀 이동 / 범위 확장',keys:['방향키','Shift + 방향키']},
    {action:'데이터 영역 끝으로 이동 / 확장',keys:['Ctrl / ⌘ + 방향키','Ctrl / ⌘ + Shift + 방향키']},
    {action:'전체 시트 선택',keys:['Ctrl / ⌘ + A']},
    {action:'현재 열 / 행 선택',keys:['Ctrl + Space','Shift + Space']},
    {action:'시트 처음 / 끝으로 이동',keys:['Ctrl / ⌘ + Home','Ctrl / ⌘ + End']},
  ]},
  {title:'서식',shortcuts:[
    {action:'굵게 / 기울임 / 밑줄',keys:['Ctrl / ⌘ + B','Ctrl / ⌘ + I','Ctrl / ⌘ + U']},
    {action:'취소선',keys:['Alt + Shift + 5']},
    {action:'왼쪽 / 가운데 / 오른쪽 정렬',keys:['Ctrl / ⌘ + Shift + L','Ctrl / ⌘ + Shift + E','Ctrl / ⌘ + Shift + R']},
    {action:'숫자 / 시간 / 날짜 표시 형식',keys:['Ctrl / ⌘ + Shift + 1','Ctrl / ⌘ + Shift + 2','Ctrl / ⌘ + Shift + 3']},
    {action:'통화 / 백분율 / 지수 표시 형식',keys:['Ctrl / ⌘ + Shift + 4','Ctrl / ⌘ + Shift + 5','Ctrl / ⌘ + Shift + 6']},
  ]},
]

export function WorkbookShortcutsDialog({onClose}:{onClose:()=>void}){
  useEffect(()=>{const close=(event:KeyboardEvent)=>{if(event.key==='Escape')onClose()};window.addEventListener('keydown',close);return()=>window.removeEventListener('keydown',close)},[onClose])
  const dialog=useDialog<HTMLElement>(onClose)
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal shortcuts-modal" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="워크북 단축키">
      <h2>Google Sheets 스타일 단축키</h2>
      <p>Windows·ChromeOS에서는 Ctrl, macOS에서는 ⌘를 사용합니다.</p>
      <div className="shortcut-sections">
        {sections.map(section=><section key={section.title}><h3>{section.title}</h3>{section.shortcuts.map(shortcut=><div className="shortcut-row" key={shortcut.action}><span>{shortcut.action}</span><div>{shortcut.keys.map(key=><kbd key={key}>{key}</kbd>)}</div></div>)}</section>)}
      </div>
      <div className="modal-actions"><button className="primary" onClick={onClose}>닫기</button></div>
    </section>
  </div>
}
