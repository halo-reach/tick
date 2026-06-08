import { useState, ReactNode } from 'react'

interface TabItem {
  key: string
  label: string
  content: ReactNode
}

interface TabsProps {
  items: TabItem[]
  activeKey?: string
  defaultKey?: string
  onChange?: (key: string) => void
}

export default function Tabs({ items, activeKey, defaultKey, onChange }: TabsProps) {
  const [internalKey, setInternalKey] = useState(defaultKey || items[0]?.key || '')
  const current = activeKey ?? internalKey

  const handleChange = (key: string) => {
    setInternalKey(key)
    onChange?.(key)
  }

  return (
    <div>
      <div className="flex gap-1 border-b border-stone-200 mb-4">
        {items.map((item) => (
          <button
            key={item.key}
            type="button"
            onClick={() => handleChange(item.key)}
            className={`px-3 py-2 text-sm font-medium transition-colors duration-150 cursor-pointer border-b-2 -mb-px ${
              current === item.key
                ? 'border-gray-900 text-gray-900'
                : 'border-transparent text-gray-400 hover:text-gray-600'
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>
      <div>{items.find((i) => i.key === current)?.content}</div>
    </div>
  )
}
