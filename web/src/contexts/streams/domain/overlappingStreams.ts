import { coversSubject } from '@/shared/domain/subjectMatch'

interface SubjectOwner {
  name: string
  subjects?: string[]
}

export interface StreamOverlap {
  name: string
  subjects: string[]
}

export function overlappingStreams(subjects: string[], streams: SubjectOwner[], except?: string): StreamOverlap[] {
  const wanted = subjects.map((subject) => subject.trim()).filter(Boolean)
  const overlaps: StreamOverlap[] = []
  for (const stream of streams) {
    if (stream.name === except) continue
    const clashing = (stream.subjects ?? []).filter((taken) => wanted.some((want) => coversSubject(want, taken) || coversSubject(taken, want)))
    if (clashing.length > 0) overlaps.push({ name: stream.name, subjects: clashing })
  }
  return overlaps
}
