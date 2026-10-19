import { describe, expect, it } from 'vitest'
import type { Location, Warehouse } from '../api/types'
import { binLabel, distanceFromCentre, fillPercent, fillText } from './layout'

const warehouse: Warehouse = { id: 1, code: 'MAIN', name: 'Main', x: 0, y: 0, radius_m: 50, capacity: 1000, used: 0 }

describe('layout helpers', () => {
  it('measures straight-line distance from the warehouse centre', () => {
    expect(distanceFromCentre(warehouse, 3, 4)).toBe(5)
  })
  it('describes fill levels', () => {
    expect(fillText(45, 60)).toBe('45 / 60')
    expect(fillText(45, null)).toBe('45 / no limit')
    expect(fillPercent(45, 60)).toBe(75)
    expect(fillPercent(70, 60)).toBe(100)
    expect(fillPercent(10, null)).toBe(0)
  })
  it('labels bins for dropdowns', () => {
    const bin: Location = { id: 1, code: 'DOCK', description: 'Receiving dock', warehouse_id: 1, x: 0, y: 0, capacity: null, auto_place: false, used: 3, created_at: '' }
    expect(binLabel(bin)).toBe('DOCK · Receiving dock · 3 / no limit · manual only')
  })
})
