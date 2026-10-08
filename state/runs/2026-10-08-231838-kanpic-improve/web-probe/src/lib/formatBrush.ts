/** The style keys the brush carries between cells. */
export const BRUSH_STYLE_KEYS=['bold','italic','underline','strike','color','background','font_size','font_family',
  'horizontal_align','vertical_align','number_format','text_mode','wrap','text_rotation','borders'] as const

/**
 * The patch that makes a target cell look like the copied one. Keys the source
 * does not set are cleared rather than left alone, because copying a format
 * means the target ends up looking like the source — not like a mixture.
 * Merge metadata is deliberately left out: painting a format must never merge
 * or unmerge cells.
 */
export function brushPatch(style:Record<string,unknown>|undefined){
  const source=style??{}
  const patch:Record<string,unknown>={}
  for(const key of BRUSH_STYLE_KEYS){
    patch[key]=Object.prototype.hasOwnProperty.call(source,key)?source[key]:null
  }
  return patch
}

/** Whether the copied format would change anything at all. */
export function brushIsEmpty(style:Record<string,unknown>|undefined){
  const patch=brushPatch(style)
  return Object.values(patch).every(value=>value===null)
}
