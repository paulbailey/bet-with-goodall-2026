interface FlagProps {
  fi: string
  className?: string
}

export function Flag({ fi, className }: FlagProps) {
  if (!fi) return null
  return <span className={`fi fi-${fi}${className ? ` ${className}` : ''}`} />
}
