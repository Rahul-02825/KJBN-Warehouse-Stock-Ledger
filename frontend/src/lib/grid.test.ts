import { describe, expect, it } from 'vitest'
import type { StockCell } from '../api/types'
import { isZero, toGrid } from './grid'

const cell = (product_id: number, sku: string, location_id: number, location_code: string, quantity: string): StockCell => ({
  product_id,
  sku,
  product_name: sku,
  unit: 'pcs',
  location_id,
  location_code,
  quantity,
  average_cost: '0',
})

describe('toGrid', () => {
  it('makes one row per product and one column per bin', () => {
    const { locations, rows } = toGrid([cell(1, 'A', 2, 'B-01', '5'), cell(1, 'A', 1, 'A-01', '0'), cell(2, 'B', 2, 'B-01', '3.5'), cell(2, 'B', 1, 'A-01', '1')])
    expect(locations.map((l) => l.code)).toEqual(['A-01', 'B-01'])
    expect(rows.map((r) => r.sku)).toEqual(['A', 'B'])
    expect(rows[1].quantities).toEqual({ 1: '1', 2: '3.5' })
  })
  it('handles an empty warehouse', () => {
    expect(toGrid([])).toEqual({ locations: [], rows: [] })
  })
})

describe('isZero', () => {
  it('recognises zero in any form', () => {
    expect(isZero('0')).toBe(true)
    expect(isZero('0.0000')).toBe(true)
    expect(isZero(undefined)).toBe(true)
    expect(isZero('0.5')).toBe(false)
  })
})
