import { request } from './client'
import type { LoginInput, RegisterInput, User } from '@/types'

export async function register(input: RegisterInput): Promise<User> {
  return request<User>('/auth/register', { method: 'POST', body: JSON.stringify(input) })
}

export async function login(input: LoginInput): Promise<User> {
  return request<User>('/auth/login', { method: 'POST', body: JSON.stringify(input) })
}

export async function logout(): Promise<void> {
  await request<void>('/auth/logout', { method: 'POST' })
}

export async function getMe(): Promise<User> {
  return request<User>('/auth/me')
}

export async function verifyEmail(token: string): Promise<void> {
  await request<void>('/auth/verify', {
    method: 'POST',
    body: JSON.stringify({ token }),
  })
}

export async function requestPasswordReset(email: string): Promise<void> {
  await request<void>('/auth/password-reset/request', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export async function confirmPasswordReset(token: string, password: string): Promise<void> {
  await request<void>('/auth/password-reset/confirm', {
    method: 'POST',
    body: JSON.stringify({ token, password }),
  })
}
