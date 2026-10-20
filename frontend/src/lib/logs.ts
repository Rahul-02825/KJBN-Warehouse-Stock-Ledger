import type { LogLine } from '../api/types'

const shownSeparately = new Set(['time', 'level', 'msg', 'request_id'])

export function logDetails(line: LogLine): [string, string][] {
  return Object.entries(line)
    .filter(([key]) => !shownSeparately.has(key))
    .map(([key, value]) => [key, typeof value === 'string' ? value : JSON.stringify(value)])
}
