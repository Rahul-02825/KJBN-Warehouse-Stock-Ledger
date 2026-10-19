import { apiURL } from '../api/client'

export function CsvLink({ path }: { path: string }) {
  return (
    <a href={apiURL(path)} className="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
      Download CSV
    </a>
  )
}
