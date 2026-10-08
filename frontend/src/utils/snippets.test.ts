import { describe, expect, it } from 'vitest'
import { AuthMode, type SmtpConfig } from '@/types'
import { buildSnippets, connectionInfo } from './snippets'

const base: SmtpConfig = {
  host: '127.0.0.1',
  port: 1025,
  authMode: AuthMode.AuthAny,
  username: 'ignored',
  password: 'ignored',
  maxMessageMb: 25,
  maxConnections: 100,
  autoStart: true,
}

describe('connectionInfo', () => {
  it('hides credentials unless auth is required', () => {
    const c = connectionInfo(base)
    expect(c).toMatchObject({ host: '127.0.0.1', port: 1025, username: '', password: '', authRequired: false })
  })

  it('uses loopback when bound to all interfaces and prefers the live port', () => {
    const c = connectionInfo({ ...base, host: '0.0.0.0' }, 2525)
    expect(c.host).toBe('127.0.0.1')
    expect(c.port).toBe(2525)
  })

  it('includes credentials when required', () => {
    const c = connectionInfo({ ...base, authMode: AuthMode.AuthRequired, username: 'dev', password: 'p@ss' })
    expect(c).toMatchObject({ username: 'dev', password: 'p@ss', authRequired: true })
  })
})

describe('buildSnippets', () => {
  it('renders Laravel env without credentials', () => {
    const laravel = buildSnippets(connectionInfo(base)).find((s) => s.id === 'laravel')!
    expect(laravel.code).toContain('MAIL_HOST=127.0.0.1')
    expect(laravel.code).toContain('MAIL_PORT=1025')
    expect(laravel.code).toContain('MAIL_USERNAME=null')
  })

  it('escapes credentials per language', () => {
    const snippets = buildSnippets(
      connectionInfo({ ...base, authMode: AuthMode.AuthRequired, username: 'dev', password: 'p@ss"x' }),
    )
    const php = snippets.find((s) => s.id === 'php')!
    expect(php.code).toBe('MAILER_DSN=smtp://dev:p%40ss%22x@127.0.0.1:1025')
    const node = snippets.find((s) => s.id === 'node')!
    expect(node.code).toContain('auth: { user: "dev", pass: "p@ss\\"x" }')
  })
})
