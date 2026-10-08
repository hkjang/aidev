import { Grid3X3 } from 'lucide-react'

export function Brand({ compact=false }: {compact?:boolean}) {
  return <a className="brand" href="/" aria-label="kanpic 홈">
    <span className="brand-mark"><Grid3X3 size={compact?18:21}/></span>
    {!compact && <span>kanpic</span>}
  </a>
}
