import { useRef, useState, useEffect, useCallback } from 'react'
import { copyText } from '@/utils/clipboard'
import { helpers } from '@/utils/helpers'
import Tooltip from './Tooltip'

export default function HelpersDropdown() {
  const helpersRef = useRef<HTMLDivElement>(null)
  const [showHelpers, setShowHelpers] = useState(false)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (helpersRef.current && !helpersRef.current.contains(e.target as Node)) {
        setShowHelpers(false)
      }
    }
    if (showHelpers) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showHelpers])

  const copyHelper = useCallback(async (template: string) => {
    await copyText(template, `Copied ${template}`)
  }, [])

  return (
    <div className="relative" ref={helpersRef}>
      <Tooltip content="Dynamic value helpers">
        <button
          onClick={() => setShowHelpers(!showHelpers)}
          className={`p-1 rounded transition-colors ${showHelpers ? 'text-orange-400 bg-orange-400/10' : 'text-content-tertiary hover:text-gray-300 hover:bg-gray-700'}`}
          aria-label="Dynamic value helpers"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>
      </Tooltip>

      {showHelpers && (
        <div className="absolute right-0 top-full mt-1 z-20 bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-2">
          <div className="px-3 pb-2 mb-1 border-b border-gray-700">
            <span className="text-xs font-medium text-gray-300">Dynamic Helpers</span>
            <p className="text-xs text-content-tertiary mt-0.5">Click to copy. Replaced on publish.</p>
          </div>
          <ul className="max-h-64 overflow-y-auto">
            {helpers.map((helper) => (
              <li key={helper.template}>
                <button
                  type="button"
                  onClick={() => copyHelper(helper.template)}
                  className="w-full flex items-center justify-between gap-3 px-3 py-1.5 whitespace-nowrap hover:bg-gray-700 cursor-pointer"
                >
                  <code className="text-xs font-mono text-orange-400">{helper.template}</code>
                  <span className="text-xs text-content-tertiary">{helper.description}</span>
                </button>
              </li>
            ))}
          </ul>
          <div className="px-3 pt-2 mt-1 border-t border-gray-700">
            <p className="text-xs text-content-tertiary">
              <code className="text-orange-400/70">$</code> prefix = same value everywhere
            </p>
            <p className="text-xs text-content-secondary mt-0.5">
              e.g. <code className="text-orange-400/70">{'{{'}<span className="text-yellow-400">$</span>uuid{'}}'}</code> in subject & body
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
