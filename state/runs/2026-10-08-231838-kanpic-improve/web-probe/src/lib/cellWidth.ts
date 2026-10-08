/**
 * 자리에 안 들어가는 숫자를 잘라 보여 주면 안 된다. `1,234,567` 의 앞부분만
 * 보이면 `1,234` 로 읽히고, 가로로 눌러 담으면 읽을 수가 없다. 스프레드시트가
 * `####` 를 쓰는 이유다 — 값이 아니라 **열이 좁다** 는 뜻이다.
 */
export function hashesWhenTooNarrow(text:string,available:number,measure:(value:string)=>number){
  if(text===''||available<=0)return text
  if(measure(text)<=available)return text
  const unit=measure('#')
  if(!(unit>0))return text
  // 적어도 하나는 보여야 좁다는 것을 알 수 있다.
  const count=Math.max(1,Math.floor(available/unit))
  return '#'.repeat(count)
}
