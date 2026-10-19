import type { ReactNode } from 'react'
import { SectionTabs } from '../../components/SectionTabs'
import { reportTabs } from '../../lib/navigation'
import { PageHeader } from '../../components/ui'

export function ReportHeader({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return (
    <>
      <PageHeader eyebrow="Reports" title={title} description={description} actions={actions} />
      <SectionTabs tabs={reportTabs} label="Reports" />
    </>
  )
}
