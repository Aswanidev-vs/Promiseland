import { buildQuery, request } from './client'
import type { Anime, Genre } from '@/types'

export async function searchAnime(q: string): Promise<Anime[]> {
  return request<Anime[]>(`/anime${buildQuery({ q })}`)
}

export async function getAnime(id: number): Promise<Anime> {
  return request<Anime>(`/anime/${id}`)
}

export async function listGenres(): Promise<Genre[]> {
  return request<Genre[]>('/genres')
}
