import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it,vi } from 'vitest'
import type { Chart,ChartData,Sheet } from '../types'
import { ChartDialog } from './ChartDialog'
import { ChartPlot } from './ChartOverlay'

const sheet:Sheet={id:'sheet-1',workbook_id:'book-1',name:'Sheet1',position:0,hidden:false,layout:{revision:1,frozen_rows:0,frozen_columns:0},created_at:'2026-08-01T00:00:00Z'}
const chart:Chart={id:'chart-1',workbook_id:'book-1',workbook_version:3,sheet_id:sheet.id,source_sheet_id:sheet.id,type:'bar',title:'월별 매출',source_range:'A1:B3',first_row_headers:true,first_column_labels:true,legend_position:'right',position:{x:24,y:24,width:560,height:320},revision:1,created_by:'alice',updated_by:'alice',created_at:'2026-08-01T00:00:00Z',updated_at:'2026-08-01T00:00:00Z'}

afterEach(cleanup)

describe('ChartDialog',()=>{
  it('creates a chart from the selected range with explicit header settings',async()=>{
    const create=vi.fn().mockResolvedValue(chart)
    render(<ChartDialog activeSheetId={sheet.id} selectionRange="A1:C8" sheets={[sheet]} onClose={vi.fn()} onCreate={create} onUpdate={vi.fn()} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('차트 제목'),{target:{value:'분기 실적'}})
    fireEvent.change(screen.getByLabelText('차트 유형'),{target:{value:'line'}})
    fireEvent.click(screen.getByText('차트 저장'))
    await waitFor(()=>expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0][0]).toMatchObject({sheet_id:sheet.id,source_sheet_id:sheet.id,source_range:'A1:C8',type:'line',title:'분기 실적',first_row_headers:true,first_column_labels:true})
  })

  it('sends the current revision when editing',async()=>{
    const update=vi.fn().mockResolvedValue({...chart,type:'pie',revision:2})
    render(<ChartDialog chart={chart} activeSheetId={sheet.id} selectionRange="D1:E4" sheets={[sheet]} onClose={vi.fn()} onCreate={vi.fn()} onUpdate={update} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('차트 유형'),{target:{value:'pie'}})
    fireEvent.click(screen.getByText('차트 저장'))
    await waitFor(()=>expect(update).toHaveBeenCalledTimes(1))
    expect(update.mock.calls[0][0]).toEqual(chart)
    expect(update.mock.calls[0][1]).toMatchObject({type:'pie',expected_revision:1})
  })
})

describe('ChartPlot',()=>{
  it('renders server-provided series as accessible SVG without interpreting labels as HTML',()=>{
    const data:ChartData={chart,workbook_version:3,series:[{name:'매출 <img src=x>',points:[{category:'1월 <script>',value:10},{category:'2월',value:20}]}]}
    const {container}=render(<ChartPlot data={data}/>)
    expect(screen.getByRole('img',{name:'월별 매출'})).toBeInTheDocument()
    expect(container.querySelectorAll('rect').length).toBeGreaterThan(2)
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
    expect(container.textContent).toContain('매출 <img src=x>')
  })

  it('shows a stable empty state for a broken source reference',()=>{
    render(<ChartPlot data={{chart:{...chart,source_sheet_id:undefined,source_range:'#REF!'},workbook_version:4,series:[],warning:'#REF!'}}/>)
    expect(screen.getByText('#REF!')).toBeInTheDocument()
  })

  it('hides the category legend for a pie chart when requested',()=>{
    const {container}=render(<ChartPlot data={{chart:{...chart,type:'pie',legend_position:'none'},workbook_version:3,series:[{name:'매출',points:[{category:'1월',value:10},{category:'2월',value:20}]}]}}/>)
    expect(container.querySelectorAll('path')).toHaveLength(2)
    expect(container.querySelectorAll('text')).toHaveLength(0)
  })
})
