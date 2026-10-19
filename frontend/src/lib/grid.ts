import type { StockCell } from '../api/types'

export interface GridRow {
  productId: number
  sku: string
  name: string
  unit: string
  quantities: Record<number, string> // location id -> quantity
}

// Turns the API's flat list (one row per product x bin) into one row per product.
// The API already sorts by product name, then bin code, so order is kept as received.
export function toGrid(cells: StockCell[]): { locations: { id: number; code: string }[]; rows: GridRow[] } {
  const locations: { id: number; code: string }[] = []
  const rows: GridRow[] = []
  const rowByProduct = new Map<number, GridRow>()

  for (const c of cells) {
    if (!locations.some((l) => l.id === c.location_id)) locations.push({ id: c.location_id, code: c.location_code })
    let row = rowByProduct.get(c.product_id)
    if (!row) {
      row = { productId: c.product_id, sku: c.sku, name: c.product_name, unit: c.unit, quantities: {} }
      rowByProduct.set(c.product_id, row)
      rows.push(row)
    }
    row.quantities[c.location_id] = c.quantity
  }
  locations.sort((a, b) => a.code.localeCompare(b.code))
  return { locations, rows }
}

export function isZero(quantity: string | undefined): boolean {
  return quantity === undefined || /^0+(\.0+)?$/.test(quantity)
}
