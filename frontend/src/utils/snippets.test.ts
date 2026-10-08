import { describe, expect, it } from 'vitest'
import { AuthMode, type SmtpConfig } from '@/types'
import { SNIPPET_GROUPS, buildSnippets, connectionInfo } from './snippets'

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

const required: SmtpConfig = { ...base, authMode: AuthMode.AuthRequired, username: 'devuser', password: 's3cret' }

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
    expect(connectionInfo(required)).toMatchObject({ username: 'devuser', password: 's3cret', authRequired: true })
  })
})

describe('buildSnippets catalogue', () => {
  const noAuthSnippets = buildSnippets(connectionInfo({ ...base, port: 2526 }))
  const authSnippets = buildSnippets(connectionInfo({ ...required, port: 2526 }))

  it('covers every group with unique ids', () => {
    const ids = noAuthSnippets.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
    for (const g of SNIPPET_GROUPS) {
      expect(noAuthSnippets.some((s) => s.group === g), `group ${g} empty`).toBe(true)
    }
    expect(noAuthSnippets.length).toBeGreaterThanOrEqual(30)
  })

  it('every snippet uses the configured port', () => {
    for (const s of noAuthSnippets) {
      expect(s.code, s.id).toContain('2526')
    }
  })

  it('every snippet that supports auth includes the credentials when required', () => {
    for (const s of authSnippets.filter((x) => !x.noAuth)) {
      expect(s.code, s.id).toContain('s3cret')
      expect(s.code, s.id).toContain('devuser')
    }
  })

  it('no snippet leaks placeholder credentials when auth is not required', () => {
    for (const s of noAuthSnippets) {
      expect(s.code, s.id).not.toContain('ignored')
      expect(s.code, s.id).not.toContain('undefined')
      expect(s.code, s.id).not.toMatch(/\n\n\n/)
    }
  })

  it('renders Laravel env without credentials', () => {
    const laravel = noAuthSnippets.find((s) => s.id === 'laravel')!
    expect(laravel.code).toContain('MAIL_HOST=127.0.0.1')
    expect(laravel.code).toContain('MAIL_USERNAME=null')
  })

  it('escapes credentials per language', () => {
    const tricky = buildSnippets(
      connectionInfo({ ...base, authMode: AuthMode.AuthRequired, username: 'dev', password: `p@ss"x'y` }),
    )
    const byId = (id: string) => tricky.find((s) => s.id === id)!.code
    expect(byId('symfony')).toBe("MAILER_DSN=smtp://dev:p%40ss%22x'y@127.0.0.1:1025")
    expect(byId('node')).toContain(`auth: { user: "dev", pass: "p@ss\\"x'y" }`)
    expect(byId('phpmailer')).toContain(`$mail->Password = 'p@ss"x\\'y';`)
    expect(byId('python')).toContain(`smtp.login('dev', 'p@ss"x\\'y')`)
  })
})
