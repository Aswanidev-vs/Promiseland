import { requestForm } from './client'
import type { Media } from '@/types'

/** No upload endpoint is listed in the v1 surface; media is its own resource. */
export const MEDIA_UPLOAD_PATH = '/media'

export async function uploadMedia(file: File): Promise<Media> {
  const form = new FormData()
  form.append('file', file)
  return requestForm<Media>(MEDIA_UPLOAD_PATH, form)
}
