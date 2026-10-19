import { Navigate, useParams, useSearchParams } from 'react-router'
import { SectionTabs } from '../../components/SectionTabs'
import { ErrorAlert, PageHeader, Spinner } from '../../components/ui'
import { movementIcons } from '../../lib/navigation'
import { useOptions } from '../../lib/useOptions'
import { AdjustForm } from './AdjustForm'
import { ReceiveForm } from './ReceiveForm'
import { ShipForm } from './ShipForm'
import { StockTakeForm } from './StockTakeForm'
import { TransferForm } from './TransferForm'

const kinds = {
  receive: { title: 'Receive stock', description: 'Goods arriving from a supplier. Each receipt updates the weighted-average cost.', Form: ReceiveForm },
  ship: { title: 'Ship stock', description: 'Goods leaving the warehouse, valued at the average cost when you post.', Form: ShipForm },
  transfer: { title: 'Transfer stock', description: 'Move stock between two bins. One movement, two ledger entries.', Form: TransferForm },
  adjust: { title: 'Adjust stock', description: 'Correct stock after a count, damage, expiry or theft.', Form: AdjustForm },
  count: { title: 'Stock take', description: 'Count a bin and correct every difference in one go. Only differences are posted.', Form: StockTakeForm },
} as const

type Kind = keyof typeof kinds

export function RecordMovement() {
  const { kind = 'receive' } = useParams()
  const [params] = useSearchParams()
  const { products, locations, error } = useOptions()

  if (!(kind in kinds)) return <Navigate to="/record/receive" replace />
  const { title, description, Form } = kinds[kind as Kind]
  const product = params.get('product') ?? ''
  const query = product ? `?product=${product}` : ''

  return (
    <>
      <PageHeader eyebrow="Record movement" title={title} description={description} />
      <SectionTabs
        label="Movement type"
        tabs={[
          { to: `/record/receive${query}`, label: 'Receive', Icon: movementIcons.receive },
          { to: `/record/ship${query}`, label: 'Ship', Icon: movementIcons.ship },
          { to: `/record/transfer${query}`, label: 'Transfer', Icon: movementIcons.transfer },
          { to: `/record/adjust${query}`, label: 'Adjust', Icon: movementIcons.adjust },
          { to: '/record/count', label: 'Stock take', Icon: movementIcons.count },
        ]}
      />
      <ErrorAlert error={error} />
      {!products || !locations ? !error && <Spinner /> : <Form key={kind} products={products} locations={locations} initialProduct={product} />}
    </>
  )
}
