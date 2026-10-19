// Response shapes from the backend. Decimals arrive as strings and stay strings:
// the UI never does money or quantity arithmetic with JavaScript numbers.

export type Decimal = string

export interface ListResponse<T> {
  data: T[]
}

export interface PagedResponse<T> extends ListResponse<T> {
  page: number
  page_size: number
  total: number
}

export interface LowStockRow {
  product_id: number
  sku: string
  name: string
  unit: string
  reorder_level: Decimal
  total_quantity: Decimal
  shortfall: Decimal
}

export interface ValuationResponse {
  data: {
    product_id: number
    sku: string
    name: string
    unit: string
    total_quantity: Decimal
    average_cost: Decimal
    total_value: Decimal
  }[]
  total_value: Decimal
}



export const UNITS = ['pcs', 'kg', 'litre'] as const
export type Unit = (typeof UNITS)[number]

export interface Product {
  id: number
  sku: string
  name: string
  unit: Unit
  reorder_level: Decimal
  average_cost: Decimal
  created_at: string
  updated_at: string
}

export interface Location {
  id: number
  code: string
  description: string
  warehouse_id: number
  x: number
  y: number
  capacity: number | null
  auto_place: boolean
  used?: Decimal
  created_at: string
}

export interface Warehouse {
  id: number
  code: string
  name: string
  x: number
  y: number
  radius_m: number
  capacity: number | null
  used: Decimal
}

export interface PlacementPreview {
  warehouse_id: number
  unit: Unit
  start_location_id: number
  start_location_code: string
  reason: string
  allocations: { location_id: number; location_code: string; quantity: Decimal; distance_m: number }[]
}

// One cell of the stock grid: a product in a bin.
export interface StockCell {
  product_id: number
  sku: string
  product_name: string
  unit: Unit
  location_id: number
  location_code: string
  quantity: Decimal
  average_cost: Decimal
}

export type MovementType = 'IN' | 'OUT' | 'TRANSFER' | 'ADJUSTMENT' | 'REVERSAL'

export interface ProductBalances {
  product_id: number
  sku: string
  unit: Unit
  total_quantity: Decimal
  data: { location_id: number; location_code: string; quantity: Decimal; updated_at: string }[]
}

// One ledger entry with running balances, oldest first.
export interface LedgerEntry {
  movement_id: number
  created_at: string
  type: MovementType
  reference: string
  reason: string | null
  created_by: string
  location_id: number
  location_code: string
  quantity_delta: Decimal
  unit_cost: Decimal | null
  running_total: Decimal
  running_location_balance: Decimal
}

export interface ProductLedger {
  product_id: number
  sku: string
  unit: Unit
  data: LedgerEntry[]
}

// A posted movement as returned by the posting endpoints.
export interface Movement {
  id: number
  type: MovementType
  product_id: number
  from_location_id: number | null
  to_location_id: number | null
  quantity: Decimal
  unit_cost: Decimal | null
  total_value: Decimal | null
  reference: string
  reason: string | null
  note: string
  created_by: string
  reverses_movement_id: number | null
  created_at: string
}

export const ADJUSTMENT_REASONS = [
  { value: 'damaged', label: 'Damaged' },
  { value: 'expired', label: 'Expired' },
  { value: 'count_correction', label: 'Count correction' },
  { value: 'theft', label: 'Theft' },
  { value: 'other', label: 'Other (note required)' },
] as const

// A movement with names and reversal status, as listed in history and shown in detail.
export interface MovementDetail extends Movement {
  sku: string
  product_name: string
  unit: Unit
  from_location_code: string | null
  to_location_code: string | null
  is_reversed: boolean
  reversed_by_id: number | null
  entries?: { location_id: number; quantity_delta: Decimal; unit_cost: Decimal | null }[]
}

export const MOVEMENT_TYPES: MovementType[] = ['IN', 'OUT', 'TRANSFER', 'ADJUSTMENT', 'REVERSAL']

export interface ReconciliationRow {
  product_id: number
  sku: string
  location_id: number
  location_code: string
  cached_quantity: Decimal
  ledger_quantity: Decimal
  difference: Decimal
}

export interface CostCheckRow {
  product_id: number
  sku: string
  name: string
  stored_average: Decimal
  recomputed_average: Decimal
  difference: Decimal
  matches: boolean
}

export interface ForecastRow {
  product_id: number
  sku: string
  name: string
  unit: Unit
  last_year_demand: Decimal
  current_stock: Decimal
  gap: Decimal
  average_cost: Decimal
  gap_value: Decimal
}

export interface ActivityDay {
  day: string
  in_count: number
  out_count: number
  transfer_count: number
  adjustment_count: number
  reversal_count: number
  in_value: Decimal
  out_value: Decimal
}

export interface TopMover {
  product_id: number
  sku: string
  name: string
  unit: Unit
  shipments: number
  shipped_quantity: Decimal
  shipped_value: Decimal
}

export interface ReconciliationResponse {
  data: ReconciliationRow[]
  drift_count: number
}
