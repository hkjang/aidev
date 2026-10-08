/**
 * 고른 파일을 그 파일이 스스로 밝힌 인코딩대로 읽는다.
 *
 * `File.text()` 는 무엇이 적혀 있든 언제나 UTF-8 로 푼다. 그런데 관리자가
 * 사용자 명단을 뽑는 가장 흔한 길인 윈도우 파워셸의 `Export-Csv` 와 엑셀의
 * '유니코드 텍스트' 저장은 UTF-16LE 로 쓰고 파일 앞에 그것을 알리는
 * 표시(BOM)를 둔다. 그런 파일을 UTF-8 로 풀면 글자가 아닌 것이 나오고,
 * 머리글이 `user_id` 로 보이지 않아 서버는 "머리글 줄에 user_id 열이 있어야
 * 합니다" 로 되돌려보낸다 — 짐작할 것이 하나도 없는 파일인데도 사람에게는
 * 다른 도구로 저장을 다시 하는 것 말고 길이 없다.
 *
 * 서버의 internal/delimited.ToUTF8 과 같은 규칙이다. 같은 파일은 어느 문으로
 * 들어오든 같은 표가 되어야 한다.
 */

// 파일이 스스로 무엇으로 적혔는지 밝힐 때 앞에 두는 표시. 어느 것도 그 자체로
// 올바른 UTF-8 이 아니므로, 이것으로 시작하는 파일은 우연히 그 바이트를 담은
// 것이 아니라 인코딩을 말하고 있는 것이다.
const utf16LEBOM=[0xFF,0xFE]
const utf16BEBOM=[0xFE,0xFF]
const utf32LEBOM=[0xFF,0xFE,0x00,0x00]

/**
 * 밝힌 대로 옮겨 읽는다. 아무것도 밝히지 않은 파일은 예전대로 UTF-8 로 푼다 —
 * 표시가 없으면 UTF-16 인지, 그 바이트를 담은 멀쩡한 파일인지 가릴 길이 없고,
 * 짐작은 멀쩡한 파일을 망가뜨리는 쪽으로만 틀린다.
 */
export function decodeAnnounced(bytes:Uint8Array):string{
  // UTF-32LE 는 UTF-16LE 와 같은 표시로 열고 뒤에 0 두 개를 둔다. UTF-16 으로
  // 읽으면 읽지도 못할 파일에서 글자 아닌 것을 만들어 내므로, 예전처럼 UTF-8
  // 로 풀어 두어 서버가 거절하게 둔다.
  if(startsWith(bytes,utf32LEBOM))return utf8(bytes)
  if(startsWith(bytes,utf16LEBOM))return new TextDecoder('utf-16le').decode(bytes.subarray(utf16LEBOM.length))
  if(startsWith(bytes,utf16BEBOM))return new TextDecoder('utf-16be').decode(bytes.subarray(utf16BEBOM.length))
  // UTF-8 표시는 푸는 쪽이 스스로 뗀다.
  return utf8(bytes)
}

/** readFileText 는 고른 파일을 밝힌 대로 읽어 글자로 준다. */
export async function readFileText(file:File):Promise<string>{
  return decodeAnnounced(new Uint8Array(await file.arrayBuffer()))
}

const utf8=(bytes:Uint8Array)=>new TextDecoder().decode(bytes)
const startsWith=(bytes:Uint8Array,mark:number[])=>bytes.length>=mark.length&&mark.every((byte,index)=>bytes[index]===byte)
