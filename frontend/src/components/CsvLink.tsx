import { Download } from 'lucide-react'
import { apiURL } from '../api/client'
import { icon } from './icon'

export function CsvLink({ path }: { path: string }) {
  return (
    <a
      href={apiURL(path)}
      className="inline-flex items-center gap-2 rounded-btn border border-line px-4 py-2 text-[14px] font-medium text-ink transition-colors duration-200 hover:border-petrol"
    >
      <Download {...icon} />
      Download CSV
    </a>
  )
}
