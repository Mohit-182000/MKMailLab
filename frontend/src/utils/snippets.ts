import { AuthMode, type SmtpConfig } from '@/types'

export interface ConnectionInfo {
  host: string
  port: number
  username: string
  password: string
  /** Whether clients must send credentials. */
  authRequired: boolean
}

/**
 * The address apps should connect to. A server bound to all interfaces is
 * still reached locally via 127.0.0.1.
 */
export function connectionInfo(cfg: SmtpConfig, actualPort?: number): ConnectionInfo {
  const host = cfg.host === '0.0.0.0' || cfg.host === '::' ? '127.0.0.1' : cfg.host
  const required = cfg.authMode === AuthMode.AuthRequired
  return {
    host,
    port: actualPort || cfg.port,
    username: required ? cfg.username : '',
    password: required ? cfg.password : '',
    authRequired: required,
  }
}

export interface Snippet {
  id: string
  label: string
  language: string
  code: string
}

/** Ready-to-paste configuration for common frameworks. */
export function buildSnippets(c: ConnectionInfo): Snippet[] {
  const user = c.username
  const pass = c.password
  const q = (s: string) => JSON.stringify(s)

  return [
    {
      id: 'laravel',
      label: 'Laravel',
      language: '.env',
      code: [
        'MAIL_MAILER=smtp',
        `MAIL_HOST=${c.host}`,
        `MAIL_PORT=${c.port}`,
        `MAIL_USERNAME=${user || 'null'}`,
        `MAIL_PASSWORD=${pass || 'null'}`,
        'MAIL_ENCRYPTION=null',
        'MAIL_FROM_ADDRESS="hello@example.com"',
        'MAIL_FROM_NAME="${APP_NAME}"',
      ].join('\n'),
    },
    {
      id: 'node',
      label: 'Node.js',
      language: 'JavaScript (Nodemailer)',
      code: [
        "import nodemailer from 'nodemailer'",
        '',
        'const transporter = nodemailer.createTransport({',
        `  host: ${q(c.host)},`,
        `  port: ${c.port},`,
        '  secure: false,',
        ...(c.authRequired ? [`  auth: { user: ${q(user)}, pass: ${q(pass)} },`] : []),
        '})',
        '',
        'await transporter.sendMail({',
        "  from: 'app@example.com',",
        "  to: 'user@example.com',",
        "  subject: 'Hello from Node',",
        "  html: '<p>It works!</p>',",
        '})',
      ].join('\n'),
    },
    {
      id: 'python',
      label: 'Python',
      language: 'Python (smtplib)',
      code: [
        'import smtplib',
        'from email.message import EmailMessage',
        '',
        'msg = EmailMessage()',
        "msg['From'] = 'app@example.com'",
        "msg['To'] = 'user@example.com'",
        "msg['Subject'] = 'Hello from Python'",
        "msg.set_content('It works!')",
        '',
        `with smtplib.SMTP(${q(c.host)}, ${c.port}) as smtp:`,
        ...(c.authRequired ? [`    smtp.login(${q(user)}, ${q(pass)})`] : []),
        '    smtp.send_message(msg)',
      ].join('\n'),
    },
    {
      id: 'dotnet',
      label: '.NET',
      language: 'C# (System.Net.Mail)',
      code: [
        'using System.Net;',
        'using System.Net.Mail;',
        '',
        `using var client = new SmtpClient(${q(c.host)}, ${c.port})`,
        '{',
        '    EnableSsl = false,',
        c.authRequired
          ? `    Credentials = new NetworkCredential(${q(user)}, ${q(pass)}),`
          : '    UseDefaultCredentials = false,',
        '};',
        '',
        'client.Send("app@example.com", "user@example.com", "Hello from .NET", "It works!");',
      ].join('\n'),
    },
    {
      id: 'php',
      label: 'PHP',
      language: 'PHP (Symfony Mailer DSN)',
      code: c.authRequired
        ? `MAILER_DSN=smtp://${encodeURIComponent(user)}:${encodeURIComponent(pass)}@${c.host}:${c.port}`
        : `MAILER_DSN=smtp://${c.host}:${c.port}`,
    },
    {
      id: 'django',
      label: 'Django',
      language: 'settings.py',
      code: [
        "EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'",
        `EMAIL_HOST = ${q(c.host)}`,
        `EMAIL_PORT = ${c.port}`,
        `EMAIL_HOST_USER = ${q(user)}`,
        `EMAIL_HOST_PASSWORD = ${q(pass)}`,
        'EMAIL_USE_TLS = False',
      ].join('\n'),
    },
  ]
}
