import { buildQuery, request } from './client'
import type { Page, PaginationParams, Report, ReportInput, ResolveReportInput } from '@/types'

export async function report(input: ReportInput): Promise<void> {
  await request<void>('/reports', {
    method: 'POST',
    body: JSON.stringify({
      target_type: input.targetType,
      target_id: input.targetId,
      reason: input.reason,
      details: input.details,
    }),
  })
}

export async function listReports(p: PaginationParams = {}): Promise<Page<Report>> {
  return request<Page<Report>>(`/moderation/reports${buildQuery({ ...p })}`)
}

export async function resolveReport(id: number, input: ResolveReportInput): Promise<void> {
  await request<void>(`/moderation/reports/${id}/resolve`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}
