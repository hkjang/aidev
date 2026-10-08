/**
 * 수식 오류를 사람이 읽을 수 있게 옮긴다. 셀에 `#VALUE!` 만 찍혀 있으면
 * 무엇이 잘못됐는지 알 수 없고, 고칠 방법은 더더욱 알 수 없다.
 */
export type FormulaErrorExplanation={code:string;summary:string;next:string}

const explanations:Record<string,{summary:string;next:string}>={
  '#DIV/0!':{
    summary:'0으로 나눴습니다.',
    next:'나누는 값이 비어 있거나 0인지 확인하세요. `=IFERROR(A1/B1, "")` 로 감싸면 빈 칸으로 둘 수 있습니다.',
  },
  '#N/A':{
    summary:'찾는 값이 범위에 없습니다.',
    next:'조회 범위와 찾을 값의 형태(숫자인지 텍스트인지, 앞뒤 공백)를 확인하세요. `XLOOKUP` 은 네 번째 인수로 없을 때 쓸 값을 지정할 수 있습니다.',
  },
  '#NAME?':{
    summary:'모르는 이름이나 함수를 만났습니다.',
    next:'함수 이름의 철자와 이름 범위가 실제로 있는지 확인하세요. 텍스트를 따옴표로 감싸지 않으면 이름으로 읽힙니다. 지원하는 함수는 도움말 › 함수 목록에 있습니다.',
  },
  '#NUM!':{
    summary:'계산할 수 없는 숫자입니다.',
    next:'음수의 제곱근이나 너무 큰 값, 해가 없는 재무 계산인지 확인하세요.',
  },
  '#REF!':{
    summary:'가리키던 셀이 사라졌습니다.',
    next:'참조하던 행이나 열, 시트가 지워졌을 때 생깁니다. 실행 취소로 되돌리거나 수식의 참조를 다시 지정하세요.',
  },
  '#VALUE!':{
    summary:'인수의 종류나 개수가 맞지 않습니다.',
    next:'숫자 자리에 텍스트가 들어갔는지, 인수를 빠뜨리지 않았는지 확인하세요. 수식 입력창에서 함수 이름 뒤 괄호 안에 커서를 두면 인수 순서가 표시됩니다.',
  },
  '#SPILL!':{
    summary:'결과를 펼칠 자리가 막혀 있습니다.',
    next:'여러 셀로 펼쳐지는 수식입니다. 결과가 들어갈 아래·오른쪽 칸을 비우세요.',
  },
  '#CIRC!':{
    summary:'수식이 돌고 돌아 자기 자신을 참조합니다.',
    next:'참조 사슬 어딘가가 이 셀로 돌아옵니다. 수식이 자기 셀이나 자신을 참조하는 셀을 가리키는지 확인하세요.',
  },
  '#NULL!':{
    summary:'겹치는 곳이 없는 범위를 가리켰습니다.',
    next:'범위 사이의 구분자를 확인하세요. 쉼표 대신 공백이 들어가면 두 범위가 겹치는 부분을 뜻합니다.',
  },
  '#ERROR!':{
    summary:'수식을 읽을 수 없습니다.',
    next:'괄호와 따옴표의 짝, 인수 사이의 쉼표를 확인하세요.',
  },
}

export const formulaErrorCodes=Object.keys(explanations)

/** 셀 값이 수식 오류 코드인지 확인한다. */
export function formulaErrorCode(value:unknown):string|undefined{
  if(typeof value!=='string')return undefined
  const code=value.trim().toUpperCase()
  return code in explanations?code:undefined
}

/**
 * 오류 코드의 뜻과 다음에 할 일을 돌려준다. 엔진이 구체적인 이유를 함께
 * 준 경우에는 그것을 앞세운다. 총론보다 각론이 언제나 더 쓸모 있다.
 */
export function explainFormulaError(code:string,message?:string):FormulaErrorExplanation|undefined{
  const known=explanations[code.trim().toUpperCase()]
  if(!known)return undefined
  const detail=message?.trim()
  return {code:code.trim().toUpperCase(),summary:detail||known.summary,next:known.next}
}
