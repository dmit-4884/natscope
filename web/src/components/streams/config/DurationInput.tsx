import { useEffect, useRef, useState } from 'react'
import { Input } from '@/components/ui'
import { parseDurationToNs } from '@/utils/duration'
import { nanosToGoDuration } from '../natsCli'

interface Props {
  value: number
  onChange: (ns: number) => void
  onInvalid?: (invalid: boolean) => void
  readOnly: boolean
  labelId: string
}

function display(ns: number): string {
  return ns > 0 ? nanosToGoDuration(ns) : '0'
}

export function DurationInput({ value, onChange, onInvalid, readOnly, labelId }: Props) {
  const [text, setText] = useState(() => display(value))
  const [error, setError] = useState<string>()
  const emitted = useRef(value)
  const report = useRef(onInvalid)
  report.current = onInvalid

  useEffect(() => {
    if (value === emitted.current) return
    emitted.current = value
    setText(display(value))
    setError(undefined)
    report.current?.(false)
  }, [value])

  useEffect(() => () => report.current?.(false), [])

  const handleChange = (next: string) => {
    setText(next)
    const parsed = parseDurationToNs(next)
    if (parsed.error !== undefined || parsed.ns === undefined) {
      setError(parsed.error)
      onInvalid?.(true)
      return
    }
    setError(undefined)
    onInvalid?.(false)
    emitted.current = parsed.ns
    onChange(parsed.ns)
  }

  return (
    <Input
      value={text}
      readOnly={readOnly}
      mono
      placeholder="e.g. 30s, 12h or 7d"
      error={!!error}
      errorMessage={error}
      onChange={readOnly ? undefined : (e) => handleChange(e.target.value)}
      aria-labelledby={labelId}
    />
  )
}
