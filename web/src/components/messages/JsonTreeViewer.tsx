import { memo, useState, useCallback, useMemo, useRef, useEffect } from 'react'
import { CopyIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { formatBytes, formatCount } from '@/utils/formatters'
import { copyText } from '@/utils/clipboard'
import { plural } from '@/utils/plural'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'

interface JsonTreeViewerProps {
  data: unknown
  title?: string
  defaultExpanded?: boolean
  maxInitialDepth?: number
  searchable?: boolean
  initialHeight?: number
  jsonIndentSize?: number
  // Truncated-preview mode: payload is a fixed-size text snippet; body switches
  // to <pre>, tree controls go inert, amber Load-full CTA appears under header.
  truncated?: boolean
  truncatedFullSize?: number
  onLoadFull?: () => void
  loadingFull?: boolean
  loadFullError?: string | null
}

interface TreeNodeProps {
  keyName: string | number | null
  value: unknown
  path: string
  depth: number
  isLast: boolean
  expandedPaths: Set<string>
  toggleExpand: (path: string) => void
  searchTerm: string
  maxInitialDepth: number
  showTypeLabels: boolean
  jsonIndentSize: number
}

const childPath = (path: string, key: string | number): string => {
  const segment = String(key).replace(/~/g, '~0').replace(/\./g, '~1')
  return path ? `${path}.${segment}` : segment
}

// Try to parse a string as JSON object/array.
const tryParseJsonString = (value: unknown): { parsed: unknown; isJsonString: boolean } => {
  if (typeof value !== 'string') return { parsed: null, isJsonString: false }
  const trimmed = value.trim()
  if ((trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
    try {
      const parsed = JSON.parse(trimmed)
      if (typeof parsed === 'object' && parsed !== null) {
        return { parsed, isJsonString: true }
      }
    } catch { /* not valid JSON */ }
  }
  return { parsed: null, isJsonString: false }
}

// Get value type for styling
const getValueType = (value: unknown): string => {
  if (value === null) return 'null'
  if (value === undefined) return 'undefined'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

// Get display value
const getDisplayValue = (value: unknown, type: string): string => {
  if (type === 'string') return `"${value}"`
  if (type === 'null') return 'null'
  if (type === 'undefined') return 'undefined'
  if (type === 'boolean') return value ? 'true' : 'false'
  if (type === 'number' || type === 'bigint') return String(value)
  return String(value)
}


// Value color classes (simple dark theme)
const valueColorClasses: Record<string, string> = {
  string: 'text-green-400',
  number: 'text-blue-400',
  boolean: 'text-yellow-400',
  null: 'text-content-tertiary',
  undefined: 'text-content-tertiary',
}

// Check if string matches search
const matchesSearch = (str: string, searchTerm: string): boolean => {
  if (!searchTerm) return false
  return str.toLowerCase().includes(searchTerm.toLowerCase())
}

let cachedTerm = ''
let cachedRegex: RegExp | null = null
const searchRegexFor = (term: string): RegExp => {
  if (term !== cachedTerm || !cachedRegex) {
    cachedTerm = term
    cachedRegex = new RegExp(`(${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
  }
  return cachedRegex
}

// Highlight matched text
const HighlightText: React.FC<{ text: string; searchTerm: string }> = ({ text, searchTerm }) => {
  if (!searchTerm) return <>{text}</>

  const parts = text.split(searchRegexFor(searchTerm))

  return (
    <>
      {parts.map((part, i) =>
        part.toLowerCase() === searchTerm.toLowerCase()
          ? <mark key={i} className="bg-yellow-400/30 rounded px-0.5">{part}</mark>
          : part
      )}
    </>
  )
}

const TreeNode: React.FC<TreeNodeProps> = memo(function TreeNode({
  keyName,
  value,
  path,
  depth,
  isLast,
  expandedPaths,
  toggleExpand,
  searchTerm,
  maxInitialDepth,
  showTypeLabels,
  jsonIndentSize,
}) {
  void isLast // Part of interface, unused in rendering
  const type = getValueType(value)
  const isExpandable = type === 'object' || type === 'array'
  const isExpanded = expandedPaths.has(path)

  // Whether this node or descendants match search.
  const hasSearchMatch = useMemo(() => {
    if (!searchTerm) return false
    const keyMatches = keyName !== null && matchesSearch(String(keyName), searchTerm)
    if (keyMatches) return true
    if (!isExpandable) {
      return matchesSearch(getDisplayValue(value, type), searchTerm)
    }
    return false
  }, [keyName, value, type, isExpandable, searchTerm])

  const handleKeyClick = useCallback(() => {
    if (path) {
      copyText(path, `Copied path: ${path}`)
    }
  }, [path])

  const handleValueClick = useCallback(() => {
    const valueToCopy = isExpandable
      ? JSON.stringify(value, null, jsonIndentSize)
      : String(value)
    copyText(valueToCopy, 'Copied value')
  }, [value, isExpandable, jsonIndentSize])

  // Detect embedded JSON in string value; before early return to respect hooks
  // rules.
  const jsonStringParsed = useMemo(() => tryParseJsonString(value), [value])

  // For objects and arrays.
  if (isExpandable) {
    const entries = type === 'array'
      ? (value as unknown[]).map((v, i) => [i, v] as const)
      : Object.entries(value as Record<string, unknown>)
    const count = entries.length
    const label = type === 'array' ? `Array (${count})` : `Object (${plural(count, 'key')})`

    return (
      <div className={`${depth > 0 ? 'ml-4' : ''}`}>
        <div className="flex items-center group">
          <button
            onClick={() => toggleExpand(path)}
            aria-label={isExpanded ? 'Collapse' : 'Expand'}
            className="w-4 h-4 flex items-center justify-center text-content-tertiary hover:text-gray-300 mr-1"
          >
            <svg
              className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path d="M6 6L14 10L6 14V6Z" />
            </svg>
          </button>

          {keyName !== null && (
            <>
              <button
                type="button"
                onClick={handleKeyClick}
                className={`cursor-pointer hover:bg-gray-700 px-1 rounded text-gray-200 ${
                  hasSearchMatch ? 'bg-yellow-500/20' : ''
                }`}
                title="Click to copy path"
              >
                <HighlightText text={String(keyName)} searchTerm={searchTerm} />
              </button>
              <span className="text-content-secondary mx-1">:</span>
            </>
          )}

          {showTypeLabels && (
            <button
              type="button"
              onClick={handleValueClick}
              className="text-content-tertiary text-xs cursor-pointer hover:bg-gray-700 px-1 rounded"
              title="Click to copy value"
            >
              {label}
            </button>
          )}

          {!isExpanded && (
            <span className="text-content-secondary ml-2 text-xs">
              {type === 'array' ? '[...]' : '{...}'}
            </span>
          )}
        </div>

        {isExpanded && (
          <div className="border-l border-gray-800 ml-2">
            {entries.map(([k, v], index) => (
              <TreeNode
                key={childPath(path, k)}
                keyName={k}
                value={v}
                path={childPath(path, k)}
                depth={depth + 1}
                isLast={index === entries.length - 1}
                expandedPaths={expandedPaths}
                toggleExpand={toggleExpand}
                searchTerm={searchTerm}
                maxInitialDepth={maxInitialDepth}
                showTypeLabels={showTypeLabels}
                jsonIndentSize={jsonIndentSize}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // Render embedded JSON string as expandable tree
  if (jsonStringParsed.isJsonString) {
    const parsed = jsonStringParsed.parsed
    const entries = Array.isArray(parsed)
      ? (parsed as unknown[]).map((v, i) => [i, v] as const)
      : Object.entries(parsed as Record<string, unknown>)
    const count = entries.length
    const label = Array.isArray(parsed) ? `JSON string — Array (${count})` : `JSON string — Object (${plural(count, 'key')})`

    return (
      <div className={`${depth > 0 ? 'ml-4' : ''}`}>
        <div className="flex items-center group">
          <button
            onClick={() => toggleExpand(path)}
            aria-label={isExpanded ? 'Collapse' : 'Expand'}
            className="w-4 h-4 flex items-center justify-center text-content-tertiary hover:text-gray-300 mr-1"
          >
            <svg
              className={`w-3 h-3 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path d="M6 6L14 10L6 14V6Z" />
            </svg>
          </button>

          {keyName !== null && (
            <>
              <button
                type="button"
                onClick={handleKeyClick}
                className={`cursor-pointer hover:bg-gray-700 px-1 rounded text-gray-200 ${
                  hasSearchMatch ? 'bg-yellow-500/20' : ''
                }`}
                title="Click to copy path"
              >
                <HighlightText text={String(keyName)} searchTerm={searchTerm} />
              </button>
              <span className="text-content-secondary mx-1">:</span>
            </>
          )}

          {showTypeLabels && (
            <button
              type="button"
              onClick={handleValueClick}
              className="text-orange-400/80 text-xs cursor-pointer hover:bg-gray-700 px-1 rounded"
              title="Click to copy value"
            >
              {label}
            </button>
          )}

          {!isExpanded && (
            <span className="text-content-secondary ml-2 text-xs">
              {Array.isArray(parsed) ? '[...]' : '{...}'}
            </span>
          )}
        </div>

        {isExpanded && (
          <div className="border-l border-orange-900/40 ml-2">
            {entries.map(([k, v], index) => (
              <TreeNode
                key={childPath(path, k)}
                keyName={k}
                value={v}
                path={childPath(path, k)}
                depth={depth + 1}
                isLast={index === entries.length - 1}
                expandedPaths={expandedPaths}
                toggleExpand={toggleExpand}
                searchTerm={searchTerm}
                maxInitialDepth={maxInitialDepth}
                showTypeLabels={showTypeLabels}
                jsonIndentSize={jsonIndentSize}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // items-start + flex-wrap (not items-center): baseline math made multi-MB
  // string values paint invisible until hover.
  const displayValue = getDisplayValue(value, type)
  const colorClass = valueColorClasses[type] || 'text-content-muted'
  // Heavy leaf (>50 KB string): render in scrollable <pre> with
  // content-visibility:auto, else a single inline span thrashes layout/paint.
  const heavyLeaf = !isExpandable && type === 'string' && displayValue.length > 50_000

  return (
    <div className={`flex items-start flex-wrap ${depth > 0 ? 'ml-4' : ''} py-0.5`}>
      <span className="w-4 mr-1 shrink-0" /> {/* Spacing to align with expandable nodes */}

      {keyName !== null && (
        <>
          <button
            type="button"
            onClick={handleKeyClick}
            className={`cursor-pointer hover:bg-gray-700 px-1 rounded text-gray-200 shrink-0 ${
              hasSearchMatch ? 'bg-yellow-500/20' : ''
            }`}
            title="Click to copy path"
          >
            <HighlightText text={String(keyName)} searchTerm={searchTerm} />
          </button>
          <span className="text-content-secondary mx-1 shrink-0">:</span>
        </>
      )}

      {heavyLeaf ? (
        <button
          type="button"
          onClick={handleValueClick}
          className={`cursor-pointer hover:bg-gray-700 px-1 rounded break-all whitespace-pre-wrap text-left min-w-0 max-h-64 overflow-auto scrollbar-dark m-0 font-mono text-sm w-full ${colorClass}`}
          // content-visibility skips painting off-screen glyphs — required so
          // multi-MB values don't freeze the tree.
          style={{ contentVisibility: 'auto', containIntrinsicSize: '0 256px' }}
          title={`${formatCount(displayValue.length)} chars — click to copy full value`}
        >
          {displayValue}
        </button>
      ) : (
        <button
          type="button"
          onClick={handleValueClick}
          className={`cursor-pointer hover:bg-gray-700 px-1 rounded break-all text-left min-w-0 ${colorClass} ${
            hasSearchMatch && !isExpandable ? 'bg-yellow-500/20' : ''
          }`}
          title="Click to copy value"
        >
          <HighlightText text={displayValue} searchTerm={searchTerm} />
        </button>
      )}
    </div>
  )
})

const getAllPaths = (obj: unknown, prefix = '', maxDepth = Infinity, depth = 0): string[] => {
  const paths: string[] = []
  if (depth > maxDepth) return paths

  if (obj === null || typeof obj !== 'object') {
    const { parsed, isJsonString } = tryParseJsonString(obj)
    if (isJsonString) {
      paths.push(prefix)
      paths.push(...getAllPaths(parsed, prefix, maxDepth, depth + 1))
    }
    return paths
  }

  if (Array.isArray(obj)) {
    paths.push(prefix)
    obj.forEach((item, index) => {
      paths.push(...getAllPaths(item, childPath(prefix, index), maxDepth, depth + 1))
    })
  } else {
    paths.push(prefix)
    Object.entries(obj as Record<string, unknown>).forEach(([key, value]) => {
      paths.push(...getAllPaths(value, childPath(prefix, key), maxDepth, depth + 1))
    })
  }

  return paths
}

const JsonTreeViewer = memo(function JsonTreeViewer({
  data,
  title,
  defaultExpanded = true,
  maxInitialDepth = 10,
  searchable = true,
  initialHeight = 384,
  jsonIndentSize = 2,
  truncated = false,
  truncatedFullSize,
  onLoadFull,
  loadingFull = false,
  loadFullError,
}: JsonTreeViewerProps) {
  // String data → truncated text preview; tree controls render disabled (not
  // hidden) since they need a parsed object.
  const isStringPreview = typeof data === 'string'
  const [searchTerm, setSearchTerm] = useState('')
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set(['']))
  const [contentHeight, setContentHeight] = useState(initialHeight)
  const [showTypeLabels, setShowTypeLabels] = useState(true)
  const isDraggingResize = useRef(false)
  const dragStartY = useRef(0)
  const dragStartH = useRef(0)

  const searchInputRef = useRef<HTMLInputElement>(null)

  // Vertical resize drag
  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDraggingResize.current) return
      const delta = e.clientY - dragStartY.current
      setContentHeight(Math.max(128, dragStartH.current + delta))
    }
    const handleMouseUp = () => {
      if (isDraggingResize.current) {
        isDraggingResize.current = false
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
      }
    }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [])

  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDraggingResize.current = true
    dragStartY.current = e.clientY
    dragStartH.current = contentHeight
    document.body.style.cursor = 'row-resize'
    document.body.style.userSelect = 'none'
  }, [contentHeight])

  useEffect(() => {
    if (defaultExpanded) {
      const paths = getAllPaths(data, '', maxInitialDepth)
      setExpandedPaths(new Set(paths))
    } else {
      setExpandedPaths(new Set(['']))
    }
  }, [data, defaultExpanded, maxInitialDepth])

  // Toggle single path
  const toggleExpand = useCallback((path: string) => {
    setExpandedPaths(prev => {
      const next = new Set(prev)
      if (next.has(path)) {
        next.delete(path)
      } else {
        next.add(path)
      }
      return next
    })
  }, [])

  // Expand all
  const expandAll = useCallback(() => {
    const allPaths = getAllPaths(data)
    setExpandedPaths(new Set(allPaths))
  }, [data])

  // Collapse all
  const collapseAll = useCallback(() => {
    setExpandedPaths(new Set(['']))
  }, [])

  // Copy entire JSON
  const copyAll = useCallback(() => {
    copyText(JSON.stringify(data, null, jsonIndentSize), 'Copied entire JSON')
  }, [data, jsonIndentSize])

  const debouncedSearch = useDebouncedValue(searchTerm, 250)

  useEffect(() => {
    if (debouncedSearch) {
      expandAll()
    }
  }, [debouncedSearch, expandAll])

  const jsonString = useMemo(
    () => (isStringPreview ? '' : JSON.stringify(data)),
    [data, isStringPreview],
  )

  // Count matches
  const matchCount = useMemo(() => {
    if (!debouncedSearch || !jsonString) return 0
    const regex = new RegExp(debouncedSearch.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi')
    const matches = jsonString.match(regex)
    return matches?.length || 0
  }, [jsonString, debouncedSearch])

  return (
    <div className="bg-surface-inverse rounded-lg border border-gray-800">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 bg-gray-800/50 border-b border-gray-800">
        <div className="flex items-center gap-2">
          {title && <span className="font-medium text-sm text-content-muted">{title}</span>}
        </div>

        <div className="flex items-center gap-2">
          {searchable && (
            <div className="relative">
              <Tooltip content={isStringPreview ? 'Search disabled in preview — load full payload' : ''}>
                <input
                  ref={searchInputRef}
                  type="text"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder="Search..."
                  disabled={isStringPreview}
                  className="w-32 px-2 py-1 text-xs bg-gray-800 border border-gray-700 rounded text-gray-300 placeholder-content-secondary focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                />
              </Tooltip>
              {searchTerm && !isStringPreview && (
                <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-content-tertiary">
                  {matchCount}
                </span>
              )}
            </div>
          )}

          <Tooltip content={isStringPreview ? 'Available after loading full payload' : (showTypeLabels ? 'Hide type annotations' : 'Show type annotations')}>
            <button
              onClick={() => setShowTypeLabels(prev => !prev)}
              disabled={isStringPreview}
              className={`p-1 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${showTypeLabels ? 'text-content-tertiary hover:text-gray-300' : 'text-blue-400 bg-blue-400/10'}`}
              aria-label={isStringPreview ? 'Available after loading full payload' : (showTypeLabels ? 'Hide type annotations' : 'Show type annotations')}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
              </svg>
            </button>
          </Tooltip>

          <Tooltip content={isStringPreview ? 'Available after loading full payload' : 'Expand all'}>
            <button
              onClick={expandAll}
              disabled={isStringPreview}
              className="p-1 text-content-tertiary hover:text-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label={isStringPreview ? 'Available after loading full payload' : 'Expand all'}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
              </svg>
            </button>
          </Tooltip>

          <Tooltip content={isStringPreview ? 'Available after loading full payload' : 'Collapse all'}>
            <button
              onClick={collapseAll}
              disabled={isStringPreview}
              className="p-1 text-content-tertiary hover:text-gray-300 rounded disabled:opacity-50 disabled:cursor-not-allowed"
              aria-label={isStringPreview ? 'Available after loading full payload' : 'Collapse all'}
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
              </svg>
            </button>
          </Tooltip>

          <Tooltip content={isStringPreview ? 'Copy preview text' : 'Copy JSON'}>
            <button
              onClick={() => isStringPreview
                ? copyText(data as string, 'Copied preview')
                : copyAll()}
              className="p-1 text-content-tertiary hover:text-gray-300 rounded"
              aria-label={isStringPreview ? 'Copy preview text' : 'Copy JSON'}
            >
              <CopyIcon className="w-4 h-4" />
            </button>
          </Tooltip>
        </div>
      </div>

      {/* Truncated CTA — between header and content so the warning sits next to the preview. */}
      {truncated && (
        <div className="flex items-start justify-between gap-3 border-b border-gray-800 bg-amber-500/10 px-3 py-2">
          <div className="min-w-0 text-xs">
            <p className="font-medium text-amber-200">
              Truncated preview — only the start of the payload is shown
            </p>
            <p className="mt-0.5 text-amber-200/70">
              Full payload:{' '}
              <span className="font-mono font-medium">
                {truncatedFullSize ? formatBytes(truncatedFullSize) : '—'}
              </span>
              . Click to fetch the complete message and its decoded view.
            </p>
            {loadFullError && (
              <p className="mt-1 text-red-400" title={loadFullError}>
                Load failed: {loadFullError}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onLoadFull}
            disabled={loadingFull || !onLoadFull}
            className="shrink-0 inline-flex items-center gap-1.5 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-content-inverse shadow-sm transition-colors hover:bg-accent-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            {loadingFull ? (
              <>
                <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" className="opacity-25" />
                  <path fill="currentColor" className="opacity-75" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                </svg>
                Loading…
              </>
            ) : (
              <>
                <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5m0 0l5-5m-5 5V3" />
                </svg>
                Load full payload
              </>
            )}
          </button>
        </div>
      )}

      {/* Body — tree for structured data, <pre> for the truncate preview snippet. */}
      <div
        className="p-3 font-mono text-sm overflow-auto scrollbar-dark"
        style={{ height: contentHeight }}
      >
        {isStringPreview ? (
          <pre className="whitespace-pre text-gray-200 leading-relaxed">{data as string}</pre>
        ) : (
          <TreeNode
            keyName={null}
            value={data}
            path=""
            depth={0}
            isLast={true}
            expandedPaths={expandedPaths}
            toggleExpand={toggleExpand}
            searchTerm={debouncedSearch}
            maxInitialDepth={maxInitialDepth}
            showTypeLabels={showTypeLabels}
            jsonIndentSize={jsonIndentSize}
          />
        )}
      </div>

      {/* Resize handle */}
      <div
        className="h-1.5 cursor-row-resize flex items-center justify-center group hover:bg-gray-700 transition-colors rounded-b-lg"
        onMouseDown={handleResizeStart}
      >
        <div className="w-8 h-0.5 rounded-full bg-gray-700 group-hover:bg-gray-500 transition-colors" />
      </div>
    </div>
  )
})

export default JsonTreeViewer
