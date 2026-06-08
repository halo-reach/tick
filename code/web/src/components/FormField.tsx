import { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  required?: boolean
  error?: string
  className?: string
  children: ReactNode
}

export default function FormField({ label, required, error, className, children }: FormFieldProps) {
  return (
    <div className={className}>
      <label className="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-600">
        <span>
          {required && <span className="text-red-500 mr-0.5">*</span>}
          {label}
        </span>
        {error && <span className="text-xs text-red-500 bg-red-50 px-1.5 py-0.5 rounded">{error}</span>}
      </label>
      {children}
    </div>
  )
}
