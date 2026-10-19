import type { Location, Warehouse } from '../api/types'
import { formatDecimal } from './format'

export function distanceFromCentre(warehouse: Warehouse, x: number, y: number): number {
  return Math.hypot(x - warehouse.x, y - warehouse.y)
}

export function fillText(used: string, capacity: number | null): string {
  const stored = formatDecimal(used)
  return capacity === null ? `${stored} / no limit` : `${stored} / ${formatDecimal(String(capacity))}`
}

export function fillPercent(used: string, capacity: number | null): number {
  return capacity === null ? 0 : Math.min(100, Math.round((Number(used) / capacity) * 100))
}

export function binLabel(location: Location): string {
  const parts = [location.code]
  if (location.description) parts.push(location.description)
  if (location.used !== undefined) parts.push(fillText(location.used, location.capacity))
  if (!location.auto_place) parts.push('manual only')
  return parts.join(' · ')
}
