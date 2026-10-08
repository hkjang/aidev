import { describe,expect,it } from 'vitest'
import { applyRule,inferRule,type FillExample } from './flashFill'

const rows=(pairs:Array<[string[],string]>):FillExample[]=>pairs.map(([sources,output])=>({sources,output}))

const fill=(examples:FillExample[],rest:string[][])=>{
  const rule=inferRule(examples)
  return rule?rest.map(sources=>applyRule(rule,sources)):undefined
}

describe('flashFill', () => {
  // 주소에서 아이디만 떼는 일은 손으로 가장 많이 반복되는 것 중 하나다.
  it('learns to take the part before a delimiter', () => {
    const filled=fill(rows([[['hong@example.com'],'hong']]),[['kim@sample.co.kr'],['lee.min@x.io']])
    expect(filled).toEqual(['kim','lee.min'])
  })

  it('learns to take the part after a delimiter', () => {
    const filled=fill(rows([[['hong@example.com'],'example.com']]),[['kim@sample.co.kr']])
    expect(filled).toEqual(['sample.co.kr'])
  })

  // 두 열을 붙이면서 사이에 글자를 넣는 것도 규칙이다.
  it('learns to join columns with text between them', () => {
    const filled=fill(rows([[['홍','길동'],'홍 길동']]),[['김','철수'],['이','영희']])
    expect(filled).toEqual(['김 철수','이 영희'])
  })

  it('learns to reorder columns', () => {
    const filled=fill(rows([[['길동','홍'],'홍길동']]),[['철수','김']])
    expect(filled).toEqual(['김철수'])
  })

  it('learns a case change', () => {
    const filled=fill(rows([[['seoul'],'SEOUL']]),[['busan'],['jeju']])
    expect(filled).toEqual(['BUSAN','JEJU'])
  })

  // 예시 하나로는 맞았지만 둘째 예시가 아니라고 말하면, 그 규칙은 틀린 것이다.
  it('takes every example into account, not just the first', () => {
    // 첫 줄만 보면 "앞 두 글자" 로도 "점 앞" 으로도 설명된다. 둘째 줄이 가른다.
    const filled=fill(rows([
      [['ab.cd'],'ab'],
      [['xyz.pq'],'xyz'],
    ]),[['hello.world']])
    expect(filled).toEqual(['hello'])
  })

  // 설명되지 않는 예시가 있으면 아무것도 하지 않는다. 반쯤 맞는 규칙으로
  // 수백 줄을 채우면 어디가 틀렸는지 아무도 모른다.
  it('refuses when no rule explains every example', () => {
    expect(inferRule(rows([
      [['ab.cd'],'ab'],
      [['xyz.pq'],'완전히 다른 값'],
    ]))).toBeUndefined()
  })

  it('refuses when the example owes nothing to its row', () => {
    expect(inferRule(rows([[['ab.cd'],'직접 쓴 값']]))).toBeUndefined()
  })

  it('has nothing to say without an example', () => {
    expect(inferRule([])).toBeUndefined()
    expect(inferRule(rows([[['a'],'   ']]))).toBeUndefined()
  })

  // 규칙이 어떤 줄에서는 성립하지 않을 수 있다. 그럴 때 빈 칸을 쓴다.
  it('writes nothing where the rule cannot reach', () => {
    const filled=fill(rows([[['hong@example.com'],'hong']]),[['구분자가 없는 값']])
    expect(filled).toEqual([''])
  })

  it('keeps a rule that reads the middle of a value', () => {
    const filled=fill(rows([[['2026-08-23'],'08']]),[['2027-01-15']])
    expect(filled).toEqual(['01'])
  })
})

// 틀린 규칙으로 수백 줄을 채우는 것이 이 기능이 할 수 있는 가장 나쁜 일이다.
// 확신이 서지 않으면 아무것도 하지 않는 편이 낫다.
describe('flashFill refuses rather than guesses', () => {
  it('does not read a coincidence in the middle of a value as a rule', () => {
    // 'a' 는 원본 가운데에 우연히 있을 뿐이다.
    expect(inferRule(rows([[['xaz'],'a']]))).toBeUndefined()
  })

  it('drops a rule that two examples disagree about', () => {
    expect(inferRule(rows([
      [['홍','길동'],'홍길동'],
      [['김','철수'],'철수김'],
    ]))).toBeUndefined()
  })

  // 예시가 늘어날수록 규칙은 더 확실해져야지 흔들려서는 안 된다.
  it('holds the same rule as more examples agree', () => {
    const one=inferRule(rows([[['hong@example.com'],'hong']]))
    const three=inferRule(rows([
      [['hong@example.com'],'hong'],
      [['kim@sample.co.kr'],'kim'],
      [['a.b@c.d'],'a.b'],
    ]))
    expect(one).toBeDefined()
    expect(three).toBeDefined()
    expect(applyRule(three!,['lee@x.io'])).toBe('lee')
  })

  it('survives an empty source cell without inventing text', () => {
    const rule=inferRule(rows([[['홍','길동'],'홍 길동']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['','철수'])).toBe(' 철수')
  })

  // 아주 긴 값에서도 끝나야 한다.
  it('finishes on a long value', () => {
    const long='가'.repeat(400)+'@'+'나'.repeat(400)
    const rule=inferRule(rows([[[long],'가'.repeat(400)]]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['다다@라라'])).toBe('다다')
  })
})

// 본보기가 하나뿐일 때가 가장 위험하다. 어긋나는 본보기가 없으니 틀린 규칙도
// 통과하기 때문이다.
describe('flashFill with a single example', () => {
  // 2026-08-23 에서 "2026년 8월 23일" 을 만들려면 앞의 0을 떼야 하는데, 그 규칙이
  // 없으니 "8월" 을 글자로 굳혀 넣게 된다. 그러면 다음 줄도 8월이 된다.
  it('refuses a rule that freezes this row data into text', () => {
    expect(inferRule(rows([[['2026-08-23'],'2026년 8월 23일']]))).toBeUndefined()
  })

  // 같은 모양이라도 자료를 굳혀 넣지 않으면 괜찮다.
  it('keeps a rule whose text is punctuation the person typed', () => {
    const rule=inferRule(rows([[['2026-08-23'],'2026년 08월 23일']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['2027-01-15'])).toBe('2027년 01월 15일')
  })

  it('keeps a label that owes nothing to the row', () => {
    const rule=inferRule(rows([[['1234'],'ID-1234']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['5678'])).toBe('ID-5678')
  })

  // 본보기를 하나 더 주면 굳혀 넣을 수 없게 되므로 규칙이 잡힌다.
  it('accepts what a second example makes safe', () => {
    const rule=inferRule(rows([
      [['2026-08-23'],'23/08/2026'],
      [['2027-01-15'],'15/01/2027'],
    ]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['2028-12-01'])).toBe('01/12/2028')
  })

  // 사번과 상품코드는 늘 자릿수를 맞춰 쓴다.
  it('learns to pad a number to a fixed width', () => {
    const rule=inferRule(rows([[['7'],'007'],[['12'],'012']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['5'])).toBe('005')
    expect(applyRule(rule!,['123'])).toBe('123')
  })

  // 채움 글자는 자리를 메우는 글자여야 한다. 앞에 붙은 글자를 모두 채움으로
  // 읽으면 성이 채움 글자가 되어 다음 줄의 김철수가 홍철수가 된다.
  it('does not read a word as a padding character', () => {
    const rule=inferRule(rows([[['길동','홍'],'홍길동']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['철수','김'])).toBe('김철수')
  })

  // "007" 과 "0012" 는 자릿수를 맞춘 것이 아니라 앞에 00 을 붙인 것이다.
  // 둘 다 설명되므로 그렇게 읽는 것이 맞다.
  it('reads a fixed prefix as a prefix, not as padding', () => {
    const rule=inferRule(rows([[['7'],'007'],[['12'],'0012']]))
    expect(rule).toBeDefined()
    expect(applyRule(rule!,['5'])).toBe('005')
    expect(applyRule(rule!,['123'])).toBe('00123')
  })

  // 어느 쪽으로도 설명되지 않으면 채우지 않는다.
  it('refuses when no padding or prefix explains both', () => {
    expect(inferRule(rows([[['7'],'007'],[['12'],'12']]))).toBeUndefined()
  })
})
