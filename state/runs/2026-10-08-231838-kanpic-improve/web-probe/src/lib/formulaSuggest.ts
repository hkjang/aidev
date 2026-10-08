export type FunctionDoc={name:string;category:string;syntax:string;summary:string}

export type FormulaContext={
  /** The partial function name being typed, empty when there is nothing to complete. */
  token:string
  /** Where the token starts, so accepting a suggestion can replace exactly it. */
  start:number
  /** The call the caret sits inside, which is what the signature hint describes. */
  call?:{name:string;argument:number}
}

const NAME_CHARACTER=/[A-Za-z0-9_.]/

/**
 * Reads the formula around the caret: what is being typed, and which function
 * call it belongs to. Text inside quotes is left alone so a name mentioned in
 * a string never triggers a suggestion.
 */
export function formulaContext(text:string,caret:number):FormulaContext|undefined{
  if(!text.startsWith('='))return undefined
  const upto=text.slice(0,Math.max(0,Math.min(caret,text.length)))
  let inString=false
  for(let index=0;index<upto.length;index+=1){
    if(upto[index]==='"'){
      if(inString&&upto[index+1]==='"'){index+=1;continue}
      inString=!inString
    }
  }
  if(inString)return undefined
  let start=upto.length
  while(start>0&&NAME_CHARACTER.test(upto[start-1]))start-=1
  const token=upto.slice(start)
  // A name directly followed by "(" is a call, not something to complete.
  const typing=/^[A-Za-z][A-Za-z0-9_.]*$/.test(token)?token:''
  return {token:typing,start:typing?start:upto.length,call:enclosingCall(upto)}
}

/** Walks back through balanced parentheses to the call the caret is inside. */
function enclosingCall(upto:string):{name:string;argument:number}|undefined{
  let depth=0,argument=0,inString=false
  for(let index=upto.length-1;index>=0;index-=1){
    const character=upto[index]
    if(character==='"'){inString=!inString;continue}
    if(inString)continue
    if(character===')'){depth+=1;continue}
    if(character===','&&depth===0){argument+=1;continue}
    if(character!=='(')continue
    if(depth>0){depth-=1;continue}
    let start=index
    while(start>0&&NAME_CHARACTER.test(upto[start-1]))start-=1
    const name=upto.slice(start,index).toUpperCase()
    if(!/^[A-Za-z][A-Za-z0-9_.]*$/.test(name))return undefined
    return {name,argument}
  }
  return undefined
}

/**
 * Ranks the functions worth offering for a partial name: names that start with
 * what was typed come first, then names that merely contain it.
 */
export function suggestFunctions(functions:FunctionDoc[],token:string,limit=8):FunctionDoc[]{
  if(!token)return []
  const needle=token.toUpperCase()
  // A fully typed name goes first. Otherwise typing TEXT and pressing Tab
  // would accept TEXTJOIN, because it is listed earlier.
  const exact=functions.filter(item=>item.name===needle)
  const starts=functions.filter(item=>item.name!==needle&&item.name.startsWith(needle))
  const contains=functions.filter(item=>!item.name.startsWith(needle)&&item.name.includes(needle))
  return [...exact,...starts,...contains].slice(0,limit)
}

/** Replaces the partial name with the chosen function and an open bracket. */
export function applySuggestion(text:string,context:FormulaContext,name:string){
  const after=text.slice(context.start+context.token.length)
  const bracket=after.startsWith('(')?'':'('
  return {text:text.slice(0,context.start)+name+bracket+after,caret:context.start+name.length+bracket.length}
}
