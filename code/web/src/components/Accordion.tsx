import { useState, ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'

interface AccordionProps {
  title: string
  defaultOpen?: boolean
  children: ReactNode
}

export default function Accordion({ title, defaultOpen = false, children }: AccordionProps) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="border border-stone-200 rounded-lg overflow-hidden bg-white">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium text-gray-900 hover:bg-stone-50 transition-colors duration-150 cursor-pointer"
      >
        {title}
        <ChevronDown className={`h-4 w-4 text-gray-400 transition-transform duration-200 ${open ? 'rotate-180' : ''}`} />
      </button>
      <div
        className={`transition-all duration-200 ease-out overflow-hidden ${
          open ? 'max-h-[2000px] opacity-100' : 'max-h-0 opacity-0'
        }`}
      >
        <div className="px-4 pb-4 pt-1">{children}</div>
      </div>
    </div>
  )
}
