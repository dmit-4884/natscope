import type { MicroService } from '@/api/discovery'
import { plural } from '@/utils/plural'
import { HealthBadge } from './HealthBadge'
import { healthOf, sumWindows, type EndpointWindow } from './serviceRates'
import { formatErrorShare, formatRequestRate } from './servicesUtils'

interface Props {
  services: MicroService[]
  selected: string | null
  windows: EndpointWindow[] | null
  statsShown: boolean
  onSelect: (name: string) => void
}

export function ServiceList({ services, selected, windows, statsShown, onSelect }: Props) {
  return (
    <nav aria-label="Services" className="py-1">
      <ul>
        {services.map((service) => {
          const window = windows ? sumWindows(windows, (w) => w.service === service.name) : undefined
          const active = service.name === selected
          return (
            <li key={service.name}>
              <button
                type="button"
                onClick={() => onSelect(service.name)}
                aria-current={active ? 'page' : undefined}
                className={`w-full text-left px-4 py-2.5 border-l-2 transition-colors ${
                  active ? 'bg-accent-light border-l-accent' : 'border-l-transparent hover:bg-surface-secondary'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className={`truncate text-sm font-medium ${active ? 'text-accent-text' : 'text-content-primary'}`}>
                    {service.name}
                  </span>
                  {statsShown && <HealthBadge health={healthOf(window)} compact />}
                </div>
                <div className="mt-0.5 text-xs text-content-tertiary truncate">
                  {plural(service.instances.length, 'instance')}
                  {service.versions.length > 0 && ` · v${service.versions.join(', v')}`}
                </div>
                {statsShown && (
                  <div className="mt-0.5 text-xs text-content-secondary tabular-nums">
                    {window ? `${formatRequestRate(window)} · ${formatErrorShare(window)} errors` : 'Measuring…'}
                  </div>
                )}
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
