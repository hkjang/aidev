import { describe, expect, it } from 'vitest'
import { cellKey } from '../state/editor'
import { columnPages, printableDocument, printableWidth, usedRegion , cellCSSForTest , printColumnPages , printAreaRegion } from './printSheet'
import type { Cell } from '../types'

function grid(entries:Array<[number,number,unknown,Record<string,unknown>?]>){
  const cells=new Map<string,Cell>()
  for(const [row,column,value,style] of entries)cells.set(cellKey(row,column),{sheet_id:'sheet',row,column,value,style,updated_at:''})
  return cells
}

describe('usedRegion',()=>{
  it('bounds the non-empty cells and ignores blanks',()=>{
    const cells=grid([[2,3,'항목'],[5,4,12],[9,9,'']])
    expect(usedRegion(cells)).toEqual({startRow:2,startColumn:3,endRow:5,endColumn:4})
  })

  it('returns nothing for an empty sheet',()=>{
    expect(usedRegion(new Map())).toBeUndefined()
  })
})

describe('printableDocument',()=>{
  it('renders the used range with headers and cell styling',()=>{
    const cells=grid([[1,1,'매출',{bold:true,background:'#fef3c7'}],[1,2,1200,{horizontal_align:'right'}]])
    const html=printableDocument(cells,{title:'분기 보고',sheetName:'요약',gridlines:true,headers:true})
    expect(html).toContain('<title>분기 보고</title>')
    expect(html).toContain('요약 · A1:B1')
    expect(html).toContain('font-weight:700')
    expect(html).toContain('background:#fef3c7')
    expect(html).toContain('text-align:right')
    // Column and row headers appear when they are requested.
    expect(html).toContain('<th>A</th>')
    expect(html).toContain('<th class="row-head">1</th>')
  })

  it('escapes cell text so content cannot inject markup',()=>{
    const html=printableDocument(grid([[1,1,'<script>alert(1)</script>']]),{title:'제목',sheetName:'시트',gridlines:false,headers:false})
    expect(html).not.toContain('<script>alert')
    expect(html).toContain('&lt;script&gt;')
  })

  it('says so when there is nothing to print',()=>{
    expect(printableDocument(new Map(),{title:'빈 워크북',sheetName:'시트1',gridlines:true,headers:true})).toContain('인쇄할 데이터가 없습니다')
  })
})

describe('printing hidden rows',()=>{
  const sheet=()=>{
    const cells=new Map<string,Cell>()
    ;[['서울',100],['부산',50],['대구',30]].forEach(([name,amount],index)=>{
      cells.set(cellKey(index+1,1),{sheet_id:'s',row:index+1,column:1,value:name} as Cell)
      cells.set(cellKey(index+1,2),{sheet_id:'s',row:index+1,column:2,value:amount} as Cell)
    })
    return cells
  }

  it('leaves out the rows the reader cannot see',()=>{
    const html=printableDocument(sheet(),{title:'보고',sheetName:'요약',gridlines:true,headers:true,hiddenRows:new Set([2,3])})
    expect(html).toContain('서울')
    expect(html).not.toContain('부산')
    expect(html).not.toContain('대구')
  })

  it('keeps the original row numbers so the page matches the sheet',()=>{
    const html=printableDocument(sheet(),{title:'보고',sheetName:'요약',gridlines:true,headers:true,hiddenRows:new Set([2])})
    expect(html).toContain('<th class="row-head">3</th>')
    expect(html).not.toContain('<th class="row-head">2</th>')
  })

  it('prints everything when nothing is hidden',()=>{
    const html=printableDocument(sheet(),{title:'보고',sheetName:'요약',gridlines:true,headers:true})
    for(const name of ['서울','부산','대구'])expect(html).toContain(name)
  })
})

// 열 머리글이 표 본문의 첫 줄이면 첫 장에만 찍힌다. thead에 있어야 브라우저가
// 장마다 다시 그려 주고, 둘째 장부터도 어느 칸이 무슨 열인지 알 수 있다.
it('puts the column headings in a thead so every page repeats them',()=>{
  const cells=new Map<string,Cell>()
  for(let row=1;row<=3;row+=1)cells.set(cellKey(row,1),{sheet_id:'s',row,column:1,value:row,updated_at:''})
  const html=printableDocument(cells,{title:'긴 표',sheetName:'Sheet1',gridlines:true,headers:true})
  expect(html).toContain('<thead><tr><th class="corner"></th><th>A</th></tr></thead>')
  expect(html).toContain('<tbody>')
  // 본문 줄은 thead 밖에 있어야 한다.
  const head=html.slice(html.indexOf('<thead>'),html.indexOf('</thead>'))
  const body=html.slice(html.indexOf('<tbody>'))
  expect(body).toContain('<th class="row-head">1</th>')
  expect(head).not.toContain('row-head')
})

it('omits the thead when headings are turned off',()=>{
  const cells=new Map<string,Cell>()
  cells.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'값',updated_at:''})
  const html=printableDocument(cells,{title:'표',sheetName:'Sheet1',gridlines:false,headers:false})
  expect(html).not.toContain('<thead>')
  expect(html).toContain('<tbody>')
})


// 예전에는 표를 종이 폭에 맞춰 통째로 눌러 담았다. 열이 서른 개면 한 열이
// 몇 밀리미터로 찌그러져 읽을 수가 없었다. 엑셀은 들어가는 만큼 찍고 나머지를
// 다음 장으로 넘긴다.
describe('columnPages',()=>{
  it('keeps columns that fit on one page together',()=>{
    expect(columnPages(1,5,()=>100)).toHaveLength(1)
    expect(columnPages(1,5,()=>100)[0]).toMatchObject({start:1,end:5})
  })

  it('moves the columns that do not fit onto the next page',()=>{
    const pages=columnPages(1,20,()=>100)
    expect(pages.length).toBeGreaterThan(1)
    expect(pages[0].start).toBe(1)
    // 장과 장 사이에 빠지거나 겹치는 열이 없어야 한다.
    for(let at=1;at<pages.length;at+=1)expect(pages[at].start).toBe(pages[at-1].end+1)
    expect(pages[pages.length-1].end).toBe(20)
  })

  it('gives a column wider than the page a page of its own',()=>{
    const pages=columnPages(1,3,column=>column===2?2000:100)
    expect(pages.map(page=>[page.start,page.end])).toEqual([[1,1],[2,2],[3,3]])
  })

  it('follows the real width of each column',()=>{
    // 좁은 열은 한 장에 많이, 넓은 열은 적게 들어간다.
    expect(columnPages(1,12,()=>50).length).toBeLessThan(columnPages(1,12,()=>300).length)
  })
})

describe('printableDocument across pages',()=>{
  const wide=new Map<string,Cell>()
  for(let column=1;column<=12;column+=1)wide.set(cellKey(1,column),{sheet_id:'s',row:1,column,value:`열${column}`,updated_at:''})

  it('breaks a wide sheet into pages instead of squeezing it',()=>{
    const html=printableDocument(wide,{title:'넓은 표',sheetName:'시트1',gridlines:true,headers:true,columnWidth:()=>200})
    // 눌러 담지 않는다.
    expect(html).not.toContain('width:100%')
    expect(html.match(/<table>/g)?.length).toBeGreaterThan(1)
    // 어느 장에서든 어느 열의 몇 행인지 알 수 있어야 한다.
    expect(html.match(/<thead>/g)?.length).toBe(html.match(/<table>/g)?.length)
    expect(html.match(/class="row-head">1</g)?.length).toBe(html.match(/<table>/g)?.length)
    // 열은 하나도 빠지지 않는다.
    for(let column=1;column<=12;column+=1)expect(html).toContain(`열${column}`)
  })

  it('stays a single table when everything fits',()=>{
    const html=printableDocument(wide,{title:'좁은 표',sheetName:'시트1',gridlines:true,headers:true,columnWidth:()=>40})
    expect(html.match(/<table>/g)?.length).toBe(1)
  })
})

// 200행짜리 표를 인쇄하면 둘째 장부터는 어느 칸이 무슨 뜻인지 알 수 없었다.
// 화면에서 고정해 둔 행은 "여기까지가 머리글" 이라고 이미 말해 둔 것이다.
describe('repeating the frozen rows', ()=>{
  const long=new Map<string,Cell>()
  long.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'제품',updated_at:''})
  long.set(cellKey(1,2),{sheet_id:'s',row:1,column:2,value:'단가',updated_at:''})
  for(let row=2;row<=8;row+=1){
    long.set(cellKey(row,1),{sheet_id:'s',row,column:1,value:`품목${row}`,updated_at:''})
    long.set(cellKey(row,2),{sheet_id:'s',row,column:2,value:row*100,updated_at:''})
  }

  it('puts the frozen row in the head so every page carries it',()=>{
    const html=printableDocument(long,{title:'긴 표',sheetName:'시트1',gridlines:true,headers:true,frozenRows:1})
    const head=html.slice(html.indexOf('<thead>'),html.indexOf('</thead>'))
    expect(head).toContain('제품')
    expect(head).toContain('단가')
    // 머리글로 올라간 행이 본문에 한 번 더 나오면 안 된다.
    expect(html.match(/제품/g)).toHaveLength(1)
    expect(html).toContain('품목2')
  })

  it('leaves the body alone when nothing is frozen',()=>{
    const html=printableDocument(long,{title:'긴 표',sheetName:'시트1',gridlines:true,headers:true})
    const head=html.slice(html.indexOf('<thead>'),html.indexOf('</thead>'))
    expect(head).not.toContain('제품')
    expect(html).toContain('제품')
  })

  it('repeats the header on each column page too',()=>{
    const wide=new Map(long)
    for(let column=3;column<=12;column+=1)wide.set(cellKey(1,column),{sheet_id:'s',row:1,column,value:`열${column}`,updated_at:''})
    const html=printableDocument(wide,{title:'넓고 긴 표',sheetName:'시트1',gridlines:true,headers:true,frozenRows:1,columnWidth:()=>200})
    const tables=html.match(/<table>/g)?.length??0
    expect(tables).toBeGreaterThan(1)
    expect(html.match(/tr class="frozen"/g)?.length).toBe(tables)
  })

  it('does not repeat so many rows that no data fits',()=>{
    const html=printableDocument(long,{title:'긴 표',sheetName:'시트1',gridlines:true,headers:true,frozenRows:99})
    // 다섯 줄까지만 머리글로 올리고 나머지는 본문에 남는다.
    expect(html.match(/tr class="frozen"/g)).toHaveLength(5)
    expect(html).toContain('품목7')
  })

  it('leaves out a frozen row the reader cannot see',()=>{
    const html=printableDocument(long,{title:'긴 표',sheetName:'시트1',gridlines:true,headers:true,frozenRows:1,hiddenRows:new Set([1])})
    expect(html).not.toContain('제품')
    expect(html).toContain('품목2')
  })
})

// 인쇄물의 style 속성은 사람이 적은 서식 값으로 만들어진다. 값 하나가 속성을
// 빠져나가면 그 뒤로 문서 전체의 모양이 바뀐다.
describe('print styles', () => {
  it('keeps a style value from escaping its attribute', () => {
    const escaped=cellCSSForTest({color:'red;} body{display:none} .x{',background:'"><script>',font_family:'Inter"; x:y'})
    expect(escaped).not.toContain('}')
    expect(escaped).not.toContain('"')
    expect(escaped).not.toContain('<')
  })

  it('still carries the formats people actually use', () => {
    const css=cellCSSForTest({background:'#ff0000',color:'rgb(0, 0, 255)',bold:true,font_size:14,font_family:'Pretendard',horizontal_align:'center'})
    expect(css).toContain('background:#ff0000')
    expect(css).toContain('color:rgb(0, 0, 255)')
    expect(css).toContain('font-weight:700')
    expect(css).toContain('font-size:14px')
    expect(css).toContain('font-family:Pretendard')
    expect(css).toContain('text-align:center')
  })

  // 글자 크기는 종이에 그려진다. 터무니없는 값이 오면 한 칸이 한 장을 잡아먹는다.
  it('bounds a font size to something printable', () => {
    expect(cellCSSForTest({font_size:100000})).toContain('font-size:400px')
    expect(cellCSSForTest({font_size:0})).toContain('font-size:4px')
  })
})

// 조건부 서식은 셀에 저장된 서식이 아니라 값에 따라 그때그때 정해진다. 인쇄가
// 저장된 서식만 읽으면, 사람이 읽으라고 칠해 놓은 표가 종이에서는 아무 표시
// 없는 숫자 뭉치가 된다.
describe('printed conditional formatting', () => {
  const cells=new Map([
    [cellKey(1,1),{sheet_id:'s',row:1,column:1,value:10,updated_at:''}],
    [cellKey(2,1),{sheet_id:'s',row:2,column:1,value:500,style:{bold:true},updated_at:''}],
  ])

  it('paints the cell the rule painted, over the cell own format', () => {
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:false,
      conditional:new Map([[cellKey(2,1),{style:{background:'#00ff00'}}]])})
    expect(html).toContain('background:#00ff00')
    // 셀 자신의 서식도 남는다 — 얹히는 것이지 갈아치우는 것이 아니다.
    expect(html).toContain('font-weight:700')
  })

  it('draws a data bar as a band the width of its ratio', () => {
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:false,
      conditional:new Map([[cellKey(2,1),{bar:{color:'#38a3a5',ratio:0.75}}]])})
    expect(html).toContain('linear-gradient(to right,#38a3a5 75%,transparent 75%)')
  })

  // 아이콘은 흑백으로 인쇄해도 모양이 남는 글자로 찍는다.
  it('prints an icon beside the value rather than instead of it', () => {
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:false,
      conditional:new Map([[cellKey(2,1),{icon:{style:'3Arrows',index:2,count:3}}]])})
    expect(html).toContain('▲')
    expect(html).toContain('500')
  })

  it('ignores an icon set it has no glyphs for', () => {
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:false,
      conditional:new Map([[cellKey(2,1),{icon:{style:'7Hearts',index:0,count:7}}]])})
    expect(html).toContain('500')
    expect(html).not.toContain('undefined')
  })
})

// 인쇄 영역을 정해 두면 그 범위만 종이에 나가야 한다. 정해 두지 않으면
// 내용이 있는 곳 전체가 나간다.
//
// 엑셀 파일에서 가져온 인쇄 영역도 같은 자리에 담기므로, 원래 문서가 표
// 한 덩어리만 내도록 짜여 있었다면 그 뜻이 그대로 지켜진다.
describe('the print area limits what goes on paper',()=>{
  const cells=new Map<string,Cell>()
  for(const [row,column,value] of [[1,1,'품목'],[1,2,'수량'],[2,1,'연필'],[2,2,10],[9,5,'영역 밖']] as Array<[number,number,unknown]>)
    cells.set(`${row}:${column}`,{sheet_id:'s',row,column,value,updated_at:'now'} as Cell)

  it('prints only the area when one is set',()=>{
    const limited=printableDocument(cells,{title:'t',sheetName:'s',gridlines:false,headers:true,printArea:'A1:B2'})
    expect(limited).toContain('연필')
    expect(limited).not.toContain('영역 밖')
  })

  it('prints everything that has content when none is set',()=>{
    const whole=printableDocument(cells,{title:'t',sheetName:'s',gridlines:false,headers:true})
    expect(whole).toContain('연필')
    expect(whole).toContain('영역 밖')
  })

  it('falls back to the used region when the area cannot be read',()=>{
    const broken=printableDocument(cells,{title:'t',sheetName:'s',gridlines:false,headers:true,printArea:'말이 안 되는 값'})
    expect(broken).toContain('영역 밖')
  })
})

describe('종이 방향과 여백', () => {
  // 넓은 표를 세로로만 찍으면 열이 자꾸 다음 장으로 넘어간다. 가로로 놓으면
  // 한 장에 훨씬 많이 들어가야 한다.
  it('가로로 놓으면 한 장에 더 많은 열이 들어간다', () => {
    const portrait=printableWidth('portrait','normal')
    const landscape=printableWidth('landscape','normal')
    expect(landscape).toBeGreaterThan(portrait)
    expect(columnPages(1,20,()=>100,landscape).length).toBeLessThan(columnPages(1,20,()=>100,portrait).length)
  })

  it('여백을 좁히면 한 장에 더 들어간다', () => {
    expect(printableWidth('portrait','narrow')).toBeGreaterThan(printableWidth('portrait','normal'))
    expect(printableWidth('portrait','wide')).toBeLessThan(printableWidth('portrait','normal'))
  })

  it('종이 크기와 여백을 실제로 적어 낸다', () => {
    const cells=new Map<string,Cell>([['1:1',{sheet_id:'s',row:1,column:1,value:'값',updated_at:''}]])
    const wide=printableDocument(cells,{title:'가로',sheetName:'Sheet1',gridlines:true,headers:true,orientation:'landscape',margin:'narrow'})
    expect(wide).toContain('size:A4 landscape')
    expect(wide).toContain('margin:8mm')
    const tall=printableDocument(cells,{title:'세로',sheetName:'Sheet1',gridlines:true,headers:true})
    expect(tall).toContain('size:A4 portrait')
    expect(tall).toContain('margin:14mm')
  })

  // 한 장 너비에 맞추라고 하면 열을 나누는 대신 표를 줄인다. 열이 서른 개인
  // 표를 장마다 나누어 보는 것보다 한눈에 보는 편이 나을 때가 있다.
  it('너비에 맞추면 표를 하나로 두고 줄인다', () => {
    const cells=new Map<string,Cell>()
    for(let column=1;column<=20;column+=1)cells.set(`1:${column}`,{sheet_id:'s',row:1,column,value:column,updated_at:''})
    cells.set('2:1',{sheet_id:'s',row:2,column:1,value:'두 번째 줄',updated_at:''})
    const split=printableDocument(cells,{title:'나누기',sheetName:'Sheet1',gridlines:true,headers:true,columnWidth:()=>120})
    const fitted=printableDocument(cells,{title:'맞추기',sheetName:'Sheet1',gridlines:true,headers:true,columnWidth:()=>120,fit:'width'})
    // 나누면 표가 여럿, 맞추면 하나다.
    expect(split.match(/<table>/g)?.length ?? 0).toBeGreaterThan(1)
    expect(fitted.match(/<table>/g)?.length ?? 0).toBe(1)
    // 줄이는 것은 zoom 으로 한다. transform 은 장 나눔을 망가뜨린다.
    expect(fitted).toMatch(/zoom:0\.\d+/)
    expect(split).not.toContain('zoom:')
  })

  it('표가 종이보다 좁으면 줄이지 않는다', () => {
    const cells=new Map<string,Cell>([['1:1',{sheet_id:'s',row:1,column:1,value:'값',updated_at:''}],['2:1',{sheet_id:'s',row:2,column:1,value:'값',updated_at:''}]])
    const fitted=printableDocument(cells,{title:'작은 표',sheetName:'Sheet1',gridlines:true,headers:true,columnWidth:()=>100,fit:'width'})
    expect(fitted).not.toContain('zoom:')
  })
})

describe('printColumnPages',()=>{
  const region={startRow:1,startColumn:1,endRow:10,endColumn:20}
  const wide=()=>200

  it('is the same split the printed document uses',()=>{
    // 미리보기가 실제 인쇄와 다른 셈을 쓰면 "한 장에 들어갑니다" 라고 보여
    // 주고 두 장을 뱉는다. 한 쪽만 고쳐도 어긋나므로 문서에서 세어 맞춘다.
    const pages=printColumnPages(region,wide,'portrait','normal','none')
    const cells=new Map<string,Cell>()
    for(let column=region.startColumn;column<=region.endColumn;column+=1)
      cells.set(cellKey(1,column),{sheet_id:'s',row:1,column,value:'x',updated_at:''})
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:true,
      columnWidth:wide,orientation:'portrait',margin:'normal',fit:'none'})
    expect(html.match(/<table/g)?.length).toBe(pages.length)
  })

  it('splits fewer times on landscape than portrait',()=>{
    expect(printColumnPages(region,wide,'landscape','normal','none').length)
      .toBeLessThan(printColumnPages(region,wide,'portrait','normal','none').length)
  })

  it('splits fewer times with narrow margins',()=>{
    expect(printColumnPages(region,wide,'portrait','narrow','none').length)
      .toBeLessThanOrEqual(printColumnPages(region,wide,'portrait','wide','none').length)
  })

  it('never splits when the table is squeezed onto one page width',()=>{
    const pages=printColumnPages(region,wide,'portrait','normal','width')
    expect(pages).toHaveLength(1)
    expect(pages[0]).toMatchObject({start:1,end:20})
  })
})

describe('the preview region matches the printed region',()=>{
  // 미리보기가 인쇄와 다른 범위를 세면 장수가 어긋난다. 시트 크기만 물어
  // 보던 시절에는 자료가 D열에서 시작해도 A열부터 세어, 실제로는 세 장인
  // 표를 네 장이라고 보여 주었다.
  it('starts where the data starts, not at A1',()=>{
    const cells=new Map<string,Cell>()
    for(let column=4;column<=23;column+=1)cells.set(cellKey(1,column),{sheet_id:'s',row:1,column,value:'x',updated_at:''})
    const region=usedRegion(cells)!
    expect(region.startColumn).toBe(4)
    const wide=()=>200
    const pages=printColumnPages(region,wide,'portrait','normal','none')
    const html=printableDocument(cells,{title:'t',sheetName:'s',gridlines:true,headers:true,
      columnWidth:wide,orientation:'portrait',margin:'normal',fit:'none'})
    expect(html.match(/<table/g)?.length).toBe(pages.length)
  })

  it('ignores cells that hold only formatting, the way printing does',()=>{
    // 서식만 든 칸은 종이에 아무것도 내지 않는다. 그것까지 세면 미리보기가
    // 있지도 않은 열을 장에 넣는다.
    const cells=new Map<string,Cell>()
    cells.set(cellKey(1,1),{sheet_id:'s',row:1,column:1,value:'x',updated_at:''})
    cells.set(cellKey(1,40),{sheet_id:'s',row:1,column:40,style:{bold:true},updated_at:''})
    expect(usedRegion(cells)?.endColumn).toBe(1)
  })
})

describe('printAreaRegion', () => {
  it('reads an ordinary print area', () => {
    expect(printAreaRegion('A1:D20')).toEqual({ startRow: 1, startColumn: 1, endRow: 20, endColumn: 4 })
  })

  it('reads the whole sheet', () => {
    expect(printAreaRegion('A1:XFD1048576')).toEqual({ startRow: 1, startColumn: 1, endRow: 1048576, endColumn: 16384 })
  })

  // 인쇄 화면은 영역의 줄마다 한 줄씩 짠다. 시트가 가진 적 없는 넓이를
  // 인쇄 영역으로 적어 두면 그 파일을 여는 사람의 화면이 멈췄다.
  it('ignores an area that reaches past the sheet', () => {
    expect(printAreaRegion('A1:AAAAAAAA10')).toBeUndefined()
    expect(printAreaRegion('A1:XFE5')).toBeUndefined()
    expect(printAreaRegion('A1:A1048577')).toBeUndefined()
  })
})
