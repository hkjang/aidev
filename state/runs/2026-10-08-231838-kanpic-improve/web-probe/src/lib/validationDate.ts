/**
 * 검증에서 글자를 날짜로 읽는 규칙. 서버의 internal/workbook/validation.go
 * 가 정하는 것과 같아야 한다.
 *
 * 예전에는 Date.parse 를 그대로 썼다. 브라우저는 훨씬 너그러워서
 * "2023/03/15" 나 "March 15, 2023" 도 받아 준다. 서버는 받지 않는다.
 * 화면은 "괜찮다" 하고 값을 적어 넣고, 서버가 그 묶음을 물리쳤다.
 *
 * 더 나쁜 것은 둘 다 받아 주면서 다른 날로 읽던 것이다.
 *
 *   "2023-03-15 00:00:00"   서버 2023-03-15T00:00Z
 *                           화면 2023-03-14T15:00Z   (KST 로 읽어 하루 전)
 *
 * 그러면 "2023-03-15 이후" 라는 검증이 같은 값에 대해 화면과 서버에서 다른
 * 답을 낸다. 사람은 어느 쪽이 맞는지 알 수 없다.
 *
 * 서버가 읽는 세 가지 꼴만 읽고, 모두 UTC 로 본다.
 */
export function parseValidationDate(text:string):number|undefined{
  const value=text.trim()
  // 2006-01-02
  const plain=/^(\d{4})-(\d{2})-(\d{2})$/.exec(value)
  if(plain)return utc(plain[1],plain[2],plain[3],'0','0','0')
  // 2006-01-02 15:04:05 — 자리를 적지 않았으므로 UTC 로 본다.
  const spaced=/^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/.exec(value)
  if(spaced)return utc(spaced[1],spaced[2],spaced[3],spaced[4],spaced[5],spaced[6])
  // RFC3339 — 자리를 적었으므로 그 자리를 그대로 쓴다.
  if(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(value)){
    const parsed=Date.parse(value)
    return Number.isFinite(parsed)?parsed:undefined
  }
  return undefined
}

function utc(year:string,month:string,day:string,hour:string,minute:string,second:string){
  const y=Number(year),mo=Number(month),d=Number(day),h=Number(hour),mi=Number(minute),se=Number(second)
  // 13월이나 32일은 날짜가 아니다. Date.UTC 는 다음 달로 넘겨 버리므로
  // 직접 본다 — 서버의 time.Parse 는 그런 것을 물리친다.
  if(mo<1||mo>12||d<1||d>31||h>23||mi>59||se>59)return undefined
  const stamp=Date.UTC(y,mo-1,d,h,mi,se)
  const back=new Date(stamp)
  if(back.getUTCMonth()!==mo-1||back.getUTCDate()!==d)return undefined
  return stamp
}
