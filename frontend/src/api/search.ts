import { buildQuery, request } from './client'
import type { SearchParams, SearchResults } from '@/types'

export async function search(p: SearchParams): Promise<SearchResults> {
  return request<SearchResults>(`/search${buildQuery({ ...p })}`)
}
