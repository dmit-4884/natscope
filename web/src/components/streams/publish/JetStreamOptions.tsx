import { useState, type ReactNode } from 'react'
import { Badge, Dropdown, Input, Toggle } from '@/components/ui'
import { SectionPanel } from '@/components/common/forms/SectionPanel'
import type { PublishOptions, ScheduleKind } from './publishOptions'

interface Props {
  value: PublishOptions
  onChange: (next: PublishOptions) => void
  ttlUnavailable?: string
  incrementUnavailable?: string
  scheduleUnavailable?: string
  cronUnavailable?: string
}

const SCHEDULE_KINDS: ReadonlyArray<{ value: ScheduleKind; label: string }> = [
  { value: 'at', label: 'Once at a time' },
  { value: 'every', label: 'Every interval' },
  { value: 'cron', label: 'Cron' },
]

function Field({ id, label, hint, children }: { id: string; label: string; hint?: string; children: ReactNode }) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-gray-700 mb-1">
        {label}
      </label>
      {children}
      {hint && <p className="text-xs text-content-tertiary mt-1">{hint}</p>}
    </div>
  )
}

function Unavailable({ reason }: { reason?: string }) {
  return reason ? <p className="text-xs text-status-warning-text mt-1">{reason}</p> : null
}

export function JetStreamOptions({
  value,
  onChange,
  ttlUnavailable,
  incrementUnavailable,
  scheduleUnavailable,
  cronUnavailable,
}: Props) {
  const [isOpen, setIsOpen] = useState(false)
  const schedule = value.schedule
  const setSchedule = (next: Partial<PublishOptions['schedule']>) =>
    onChange({ ...value, schedule: { ...schedule, ...next } })
  const inUse = value.ttl.trim() !== '' || value.increment.trim() !== '' || schedule.enabled

  return (
    <SectionPanel
      label="JetStream options"
      isOpen={isOpen}
      onToggle={() => setIsOpen((open) => !open)}
      badge={inUse ? <Badge variant="primary" size="sm">Active</Badge> : undefined}
    >
      <div className="space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <Field id="publish-ttl" label="Message TTL" hint="Expires this message on its own, e.g. 30s, 5m or 1h.">
              <Input
                id="publish-ttl"
                value={value.ttl}
                placeholder="30s"
                disabled={!!ttlUnavailable}
                onChange={(e) => onChange({ ...value, ttl: e.target.value })}
              />
            </Field>
            <Unavailable reason={ttlUnavailable} />
          </div>
          <div>
            <Field id="publish-increment" label="Counter increment" hint="Adds to the subject's counter; sent without a body.">
              <Input
                id="publish-increment"
                value={value.increment}
                placeholder="+1"
                inputMode="numeric"
                disabled={!!incrementUnavailable}
                onChange={(e) => onChange({ ...value, increment: e.target.value })}
              />
            </Field>
            <Unavailable reason={incrementUnavailable} />
          </div>
        </div>

        <div>
          <div className="flex items-center gap-2">
            <Toggle
              checked={schedule.enabled}
              onChange={(enabled) => setSchedule({ enabled })}
              disabled={!!scheduleUnavailable}
              label="Schedule this message"
            />
            <span className="text-sm text-gray-700">Schedule this message</span>
          </div>
          <Unavailable reason={scheduleUnavailable} />
        </div>

        {schedule.enabled && !scheduleUnavailable && (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field id="publish-schedule-kind" label="Schedule type">
              <Dropdown
                id="publish-schedule-kind"
                value={schedule.kind}
                onChange={(kind) => setSchedule({ kind: kind as ScheduleKind })}
                options={SCHEDULE_KINDS.map((o) => (o.value !== 'at' && cronUnavailable ? { ...o, disabled: true } : o))}
              />
              {cronUnavailable && (
                <p className="text-xs text-content-tertiary mt-1">Every interval and cron: {cronUnavailable}</p>
              )}
            </Field>

            {schedule.kind === 'at' && (
              <Field id="publish-schedule-at" label="Publish at">
                <Input
                  id="publish-schedule-at"
                  type="datetime-local"
                  value={schedule.at}
                  onChange={(e) => setSchedule({ at: e.target.value })}
                />
              </Field>
            )}
            {schedule.kind === 'every' && (
              <Field id="publish-schedule-every" label="Interval" hint="For example 30s, 5m or 1h.">
                <Input
                  id="publish-schedule-every"
                  value={schedule.every}
                  placeholder="5m"
                  onChange={(e) => setSchedule({ every: e.target.value })}
                />
              </Field>
            )}
            {schedule.kind === 'cron' && (
              <>
                <Field id="publish-schedule-cron" label="Cron expression" hint="Six fields with seconds, or @hourly, @daily, …">
                  <Input
                    id="publish-schedule-cron"
                    value={schedule.cron}
                    placeholder="0 0 9 * * *"
                    onChange={(e) => setSchedule({ cron: e.target.value })}
                  />
                </Field>
                <Field id="publish-schedule-tz" label="Time zone" hint="IANA name; empty means UTC.">
                  <Input
                    id="publish-schedule-tz"
                    value={schedule.timeZone}
                    placeholder="Europe/Amsterdam"
                    onChange={(e) => setSchedule({ timeZone: e.target.value })}
                  />
                </Field>
              </>
            )}

            <Field
              id="publish-schedule-target"
              label="Target subject"
              hint="Where the scheduled message goes; must be in this stream. Keep one schedule per publish subject."
            >
              <Input
                id="publish-schedule-target"
                value={schedule.target}
                placeholder="orders.run"
                onChange={(e) => setSchedule({ target: e.target.value })}
              />
            </Field>
            <Field id="publish-schedule-ttl" label="Scheduled message TTL" hint="Optional; needs per-message TTL on the stream.">
              <Input
                id="publish-schedule-ttl"
                value={schedule.ttl}
                placeholder="1h"
                onChange={(e) => setSchedule({ ttl: e.target.value })}
              />
            </Field>
          </div>
        )}
      </div>
    </SectionPanel>
  )
}
