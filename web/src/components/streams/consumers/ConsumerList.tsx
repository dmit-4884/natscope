import { Button, Badge, Spinner, EmptyState, SearchInput, PlusIcon, RefreshIcon, UsersIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { plural } from '@/utils/plural'
import type { ConsumerInfo } from '@/types/nats'

const CONSUMER_ICON = <UsersIcon className="w-full h-full" />

interface Props {
  consumers: ConsumerInfo[]
  selectedName: string | null
  searchQuery: string
  onSearchChange: (value: string) => void
  isLoading: boolean
  isFetching: boolean
  onRefetch: () => void
  onSelect: (consumer: ConsumerInfo) => void
  onCreate: () => void
}

export function ConsumerList({
  consumers,
  selectedName,
  searchQuery,
  onSearchChange,
  isLoading,
  isFetching,
  onRefetch,
  onSelect,
  onCreate,
}: Props) {
  const filtered = consumers.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()))

  return (
    <div className="w-72 border-r bg-surface-primary flex flex-col min-h-0">
      <div className="p-3 border-b">
        <SearchInput
          placeholder="Search consumers..."
          value={searchQuery}
          onChange={onSearchChange}
          debounce={200}
          size="sm"
          className="mb-2"
        />
        <div className="flex items-center justify-between">
          <span className="text-xs text-content-tertiary">
            {plural(filtered.length, 'consumer')}
          </span>
          <div className="flex items-center gap-1">
            <Tooltip content="Refresh">
              <Button variant="ghost" size="sm" onClick={onRefetch} disabled={isFetching} aria-label="Refresh">
                <RefreshIcon className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
              </Button>
            </Tooltip>
            <Button variant="ghost" size="sm" onClick={onCreate}>
              <PlusIcon className="w-4 h-4 mr-1" />
              New
            </Button>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {isLoading ? (
          <div className="flex items-center justify-center p-4">
            <Spinner size="sm" />
          </div>
        ) : (
          filtered.map((consumer) => (
            <button
              type="button"
              key={consumer.name}
              className={`w-full text-left appearance-none p-3 border-b cursor-pointer hover:bg-surface-secondary ${
                selectedName === consumer.name ? 'bg-accent-light' : ''
              }`}
              onClick={() => onSelect(consumer)}
            >
              <div className="flex items-center justify-between mb-1">
                <span className="font-medium text-sm truncate">{consumer.name}</span>
                <div className="flex gap-1 shrink-0">
                  {consumer.config?.durable_name ? (
                    <Badge variant="default" size="sm">
                      Durable
                    </Badge>
                  ) : (
                    <Badge variant="default" size="sm">
                      Ephemeral
                    </Badge>
                  )}
                  {consumer.push_bound && (
                    <Badge variant="primary" size="sm">
                      Push
                    </Badge>
                  )}
                  {consumer.paused && (
                    <Badge variant="warning" size="sm">
                      Paused
                    </Badge>
                  )}
                </div>
              </div>
              <div className="text-xs text-content-tertiary flex gap-3">
                <span>{consumer.num_pending} pending</span>
                <span>{consumer.num_ack_pending} unacked</span>
              </div>
              {consumer.config?.filter_subject && (
                <div className="text-xs text-content-muted mt-1 truncate font-mono">{consumer.config.filter_subject}</div>
              )}
            </button>
          ))
        )}

        {!isLoading && filtered.length === 0 && (
          <EmptyState
            size="sm"
            icon={CONSUMER_ICON}
            title={searchQuery ? 'No consumers match your search' : 'No consumers'}
            description={searchQuery ? undefined : 'Add a consumer to start delivering messages.'}
          />
        )}
      </div>
    </div>
  )
}
