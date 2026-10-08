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

/** Display order of groups in the picker. */
export const SNIPPET_GROUPS = [
  'PHP',
  'JavaScript / TypeScript',
  'Python',
  'Ruby',
  'Java / Kotlin',
  '.NET (C#)',
  'Go',
  'Rust',
  'Elixir',
  'Other languages',
  'Command line',
  'Containers',
  'Generic',
] as const

export type SnippetGroup = (typeof SNIPPET_GROUPS)[number]

export interface Snippet {
  id: string
  label: string
  group: SnippetGroup
  /** File or language shown above the code, e.g. ".env" or "C#". */
  language: string
  code: string
  /** Package install command, if a library is needed. */
  install?: string
  /** Short guidance shown above the code. */
  note?: string
  /** Extra search terms. */
  keywords?: string[]
  /** True when the integration cannot send SMTP credentials. */
  noAuth?: boolean
}

/** Double-quoted string literal (JS/TS, C#, Java, Go, Rust, Python, Elixir, JSON). */
const dq = (s: string) => JSON.stringify(s)
/** Single-quoted literal for PHP / Python / Ruby / Perl / Dart. */
const sq = (s: string) => `'${s.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`
const lines = (...parts: (string | false | null | undefined)[]) => parts.filter((p) => p !== false && p != null).join('\n')

/** Ready-to-paste configuration for languages, frameworks and tools. */
export function buildSnippets(c: ConnectionInfo): Snippet[] {
  const { host, port, username: user, password: pass, authRequired: auth } = c
  const addr = `${host}:${port}`

  return [
    // ── PHP ──────────────────────────────────────────────────────────────
    {
      id: 'laravel',
      label: 'Laravel',
      group: 'PHP',
      language: '.env',
      note: 'Run `php artisan config:clear` after changing .env if config is cached.',
      code: lines(
        'MAIL_MAILER=smtp',
        `MAIL_HOST=${host}`,
        `MAIL_PORT=${port}`,
        `MAIL_USERNAME=${user || 'null'}`,
        `MAIL_PASSWORD=${pass || 'null'}`,
        'MAIL_ENCRYPTION=null',
        'MAIL_FROM_ADDRESS="hello@example.com"',
        'MAIL_FROM_NAME="${APP_NAME}"',
      ),
    },
    {
      id: 'laravel-sail',
      label: 'Laravel Sail / Docker',
      group: 'PHP',
      language: '.env',
      keywords: ['docker', 'sail'],
      note: 'Containers reach Windows through host.docker.internal. If the connection is refused, set Listen address to 0.0.0.0 in Server settings.',
      code: lines(
        'MAIL_MAILER=smtp',
        'MAIL_HOST=host.docker.internal',
        `MAIL_PORT=${port}`,
        `MAIL_USERNAME=${user || 'null'}`,
        `MAIL_PASSWORD=${pass || 'null'}`,
        'MAIL_ENCRYPTION=null',
      ),
    },
    {
      id: 'symfony',
      label: 'Symfony Mailer',
      group: 'PHP',
      language: '.env',
      keywords: ['dsn'],
      code: auth
        ? `MAILER_DSN=smtp://${encodeURIComponent(user)}:${encodeURIComponent(pass)}@${addr}`
        : `MAILER_DSN=smtp://${addr}`,
    },
    {
      id: 'phpmailer',
      label: 'PHPMailer',
      group: 'PHP',
      language: 'PHP',
      install: 'composer require phpmailer/phpmailer',
      code: lines(
        '<?php',
        'use PHPMailer\\PHPMailer\\PHPMailer;',
        '',
        "require 'vendor/autoload.php';",
        '',
        '$mail = new PHPMailer(true);',
        '$mail->isSMTP();',
        `$mail->Host = ${sq(host)};`,
        `$mail->Port = ${port};`,
        `$mail->SMTPAuth = ${auth ? 'true' : 'false'};`,
        auth && `$mail->Username = ${sq(user)};`,
        auth && `$mail->Password = ${sq(pass)};`,
        "$mail->SMTPSecure = '';",
        '$mail->SMTPAutoTLS = false;',
        '',
        "$mail->setFrom('app@example.com', 'My App');",
        "$mail->addAddress('user@example.com');",
        "$mail->Subject = 'Hello from PHPMailer';",
        "$mail->Body = 'It works!';",
        '$mail->send();',
      ),
    },
    {
      id: 'wordpress',
      label: 'WordPress',
      group: 'PHP',
      language: "functions.php (or a must-use plugin)",
      keywords: ['wp', 'wp_mail'],
      note: 'Routes every wp_mail() call through MKMailLab. Remove it before deploying.',
      code: lines(
        "add_action('phpmailer_init', function ($phpmailer) {",
        '    $phpmailer->isSMTP();',
        `    $phpmailer->Host = ${sq(host)};`,
        `    $phpmailer->Port = ${port};`,
        `    $phpmailer->SMTPAuth = ${auth ? 'true' : 'false'};`,
        auth && `    $phpmailer->Username = ${sq(user)};`,
        auth && `    $phpmailer->Password = ${sq(pass)};`,
        "    $phpmailer->SMTPSecure = '';",
        '    $phpmailer->SMTPAutoTLS = false;',
        '});',
      ),
    },
    {
      id: 'codeigniter',
      label: 'CodeIgniter 4',
      group: 'PHP',
      language: '.env',
      code: lines(
        'email.protocol = smtp',
        `email.SMTPHost = ${host}`,
        `email.SMTPPort = ${port}`,
        `email.SMTPUser = ${user}`,
        `email.SMTPPass = ${pass}`,
        'email.SMTPCrypto =',
      ),
    },
    {
      id: 'php-ini',
      label: 'PHP mail() (php.ini)',
      group: 'PHP',
      language: 'php.ini',
      keywords: ['native', 'mail function', 'xampp', 'laragon', 'wamp'],
      noAuth: true,
      note: 'Windows only. The built-in mail() function cannot authenticate, so set Authentication to "Accept any" or "Disabled". Restart Apache/PHP afterwards.',
      code: lines('[mail function]', `SMTP = ${host}`, `smtp_port = ${port}`, 'sendmail_from = app@example.com'),
    },

    // ── JavaScript / TypeScript ──────────────────────────────────────────
    {
      id: 'node',
      label: 'Node.js (Nodemailer)',
      group: 'JavaScript / TypeScript',
      language: 'JavaScript',
      install: 'npm install nodemailer',
      keywords: ['express', 'bun', 'deno', 'next.js', 'nextjs', 'nuxt', 'typescript'],
      note: 'Also works in Bun, and in Deno via "npm:nodemailer". Next.js / Nuxt server code uses the same transport.',
      code: lines(
        "import nodemailer from 'nodemailer'",
        '',
        'const transporter = nodemailer.createTransport({',
        `  host: ${dq(host)},`,
        `  port: ${port},`,
        '  secure: false,',
        '  ignoreTLS: true,',
        auth && `  auth: { user: ${dq(user)}, pass: ${dq(pass)} },`,
        '})',
        '',
        'await transporter.sendMail({',
        "  from: '\"My App\" <app@example.com>',",
        "  to: 'user@example.com',",
        "  subject: 'Hello from Node.js',",
        "  html: '<p>It works!</p>',",
        '})',
      ),
    },
    {
      id: 'nestjs',
      label: 'NestJS',
      group: 'JavaScript / TypeScript',
      language: 'app.module.ts',
      install: 'npm install @nestjs-modules/mailer nodemailer',
      code: lines(
        "import { MailerModule } from '@nestjs-modules/mailer'",
        '',
        '@Module({',
        '  imports: [',
        '    MailerModule.forRoot({',
        '      transport: {',
        `        host: ${dq(host)},`,
        `        port: ${port},`,
        '        secure: false,',
        '        ignoreTLS: true,',
        auth && `        auth: { user: ${dq(user)}, pass: ${dq(pass)} },`,
        '      },',
        "      defaults: { from: '\"My App\" <app@example.com>' },",
        '    }),',
        '  ],',
        '})',
        'export class AppModule {}',
      ),
    },
    {
      id: 'adonisjs',
      label: 'AdonisJS',
      group: 'JavaScript / TypeScript',
      language: '.env',
      code: lines('SMTP_HOST=' + host, 'SMTP_PORT=' + port, `SMTP_USERNAME=${user}`, `SMTP_PASSWORD=${pass}`),
    },
    {
      id: 'strapi',
      label: 'Strapi',
      group: 'JavaScript / TypeScript',
      language: 'config/plugins.js',
      install: 'npm install @strapi/provider-email-nodemailer',
      code: lines(
        'module.exports = () => ({',
        '  email: {',
        '    config: {',
        "      provider: 'nodemailer',",
        '      providerOptions: {',
        `        host: ${dq(host)},`,
        `        port: ${port},`,
        '        secure: false,',
        '        ignoreTLS: true,',
        auth && `        auth: { user: ${dq(user)}, pass: ${dq(pass)} },`,
        '      },',
        "      settings: { defaultFrom: 'app@example.com', defaultReplyTo: 'app@example.com' },",
        '    },',
        '  },',
        '})',
      ),
    },

    // ── Python ───────────────────────────────────────────────────────────
    {
      id: 'python',
      label: 'Python (smtplib)',
      group: 'Python',
      language: 'Python',
      code: lines(
        'import smtplib',
        'from email.message import EmailMessage',
        '',
        'msg = EmailMessage()',
        "msg['From'] = 'app@example.com'",
        "msg['To'] = 'user@example.com'",
        "msg['Subject'] = 'Hello from Python'",
        "msg.set_content('It works!')",
        '',
        `with smtplib.SMTP(${sq(host)}, ${port}) as smtp:`,
        auth && `    smtp.login(${sq(user)}, ${sq(pass)})`,
        '    smtp.send_message(msg)',
      ),
    },
    {
      id: 'django',
      label: 'Django',
      group: 'Python',
      language: 'settings.py',
      code: lines(
        "EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'",
        `EMAIL_HOST = ${sq(host)}`,
        `EMAIL_PORT = ${port}`,
        `EMAIL_HOST_USER = ${sq(user)}`,
        `EMAIL_HOST_PASSWORD = ${sq(pass)}`,
        'EMAIL_USE_TLS = False',
        'EMAIL_USE_SSL = False',
      ),
    },
    {
      id: 'flask',
      label: 'Flask-Mail',
      group: 'Python',
      language: 'Python',
      install: 'pip install Flask-Mail',
      code: lines(
        'from flask_mail import Mail',
        '',
        'app.config.update(',
        `    MAIL_SERVER=${sq(host)},`,
        `    MAIL_PORT=${port},`,
        '    MAIL_USE_TLS=False,',
        '    MAIL_USE_SSL=False,',
        `    MAIL_USERNAME=${auth ? sq(user) : 'None'},`,
        `    MAIL_PASSWORD=${auth ? sq(pass) : 'None'},`,
        "    MAIL_DEFAULT_SENDER='app@example.com',",
        ')',
        'mail = Mail(app)',
      ),
    },
    {
      id: 'fastapi',
      label: 'FastAPI (fastapi-mail)',
      group: 'Python',
      language: 'Python',
      install: 'pip install fastapi-mail',
      code: lines(
        'from fastapi_mail import ConnectionConfig',
        '',
        'conf = ConnectionConfig(',
        `    MAIL_USERNAME=${sq(user)},`,
        `    MAIL_PASSWORD=${sq(pass)},`,
        "    MAIL_FROM='app@example.com',",
        `    MAIL_SERVER=${sq(host)},`,
        `    MAIL_PORT=${port},`,
        '    MAIL_STARTTLS=False,',
        '    MAIL_SSL_TLS=False,',
        `    USE_CREDENTIALS=${auth ? 'True' : 'False'},`,
        '    VALIDATE_CERTS=False,',
        ')',
      ),
    },

    // ── Ruby ─────────────────────────────────────────────────────────────
    {
      id: 'rails',
      label: 'Ruby on Rails',
      group: 'Ruby',
      language: 'config/environments/development.rb',
      keywords: ['actionmailer', 'action mailer'],
      code: lines(
        'config.action_mailer.delivery_method = :smtp',
        'config.action_mailer.smtp_settings = {',
        `  address: ${dq(host)},`,
        `  port: ${port},`,
        auth && `  user_name: ${dq(user)},`,
        auth && `  password: ${dq(pass)},`,
        auth && '  authentication: :plain,',
        '  enable_starttls_auto: false',
        '}',
      ),
    },
    {
      id: 'ruby',
      label: 'Ruby (Net::SMTP)',
      group: 'Ruby',
      language: 'Ruby',
      install: 'gem install net-smtp',
      code: lines(
        'require "net/smtp"',
        '',
        'message = <<~MAIL',
        '  From: My App <app@example.com>',
        '  To: user@example.com',
        '  Subject: Hello from Ruby',
        '',
        '  It works!',
        'MAIL',
        '',
        auth
          ? `Net::SMTP.start(${dq(host)}, ${port}, user: ${dq(user)}, secret: ${dq(pass)}, authtype: :plain) do |smtp|`
          : `Net::SMTP.start(${dq(host)}, ${port}) do |smtp|`,
        '  smtp.send_message message, "app@example.com", "user@example.com"',
        'end',
      ),
    },

    // ── Java / Kotlin ────────────────────────────────────────────────────
    {
      id: 'spring',
      label: 'Spring Boot',
      group: 'Java / Kotlin',
      language: 'application.properties',
      keywords: ['kotlin', 'javamailsender'],
      install: 'Add dependency: org.springframework.boot:spring-boot-starter-mail',
      code: lines(
        `spring.mail.host=${host}`,
        `spring.mail.port=${port}`,
        `spring.mail.username=${user}`,
        `spring.mail.password=${pass}`,
        `spring.mail.properties.mail.smtp.auth=${auth}`,
        'spring.mail.properties.mail.smtp.starttls.enable=false',
      ),
    },
    {
      id: 'jakarta',
      label: 'Jakarta Mail (JavaMail)',
      group: 'Java / Kotlin',
      language: 'Java',
      install: 'Add dependency: org.eclipse.angus:angus-mail',
      code: lines(
        'Properties props = new Properties();',
        `props.put("mail.smtp.host", ${dq(host)});`,
        `props.put("mail.smtp.port", "${port}");`,
        `props.put("mail.smtp.auth", "${auth}");`,
        '',
        'Session session = Session.getInstance(props);',
        'MimeMessage msg = new MimeMessage(session);',
        'msg.setFrom(new InternetAddress("app@example.com"));',
        'msg.setRecipients(Message.RecipientType.TO, "user@example.com");',
        'msg.setSubject("Hello from Java");',
        'msg.setText("It works!");',
        auth ? `Transport.send(msg, ${dq(user)}, ${dq(pass)});` : 'Transport.send(msg);',
      ),
    },
    {
      id: 'quarkus',
      label: 'Quarkus',
      group: 'Java / Kotlin',
      language: 'application.properties',
      note: 'quarkus.mailer.mock=false is required, otherwise dev mode only logs emails.',
      code: lines(
        `quarkus.mailer.host=${host}`,
        `quarkus.mailer.port=${port}`,
        'quarkus.mailer.start-tls=DISABLED',
        'quarkus.mailer.mock=false',
        'quarkus.mailer.from=app@example.com',
        auth && `quarkus.mailer.username=${user}`,
        auth && `quarkus.mailer.password=${pass}`,
      ),
    },

    // ── .NET ─────────────────────────────────────────────────────────────
    {
      id: 'dotnet',
      label: 'System.Net.Mail',
      group: '.NET (C#)',
      language: 'C#',
      keywords: ['smtpclient', 'csharp', 'vb.net'],
      code: lines(
        'using System.Net;',
        'using System.Net.Mail;',
        '',
        `using var client = new SmtpClient(${dq(host)}, ${port})`,
        '{',
        '    EnableSsl = false,',
        auth ? `    Credentials = new NetworkCredential(${dq(user)}, ${dq(pass)}),` : '    UseDefaultCredentials = false,',
        '};',
        '',
        'client.Send("app@example.com", "user@example.com", "Hello from .NET", "It works!");',
      ),
    },
    {
      id: 'mailkit',
      label: 'MailKit',
      group: '.NET (C#)',
      language: 'C#',
      install: 'dotnet add package MailKit',
      keywords: ['mimekit', 'csharp'],
      code: lines(
        'using MailKit.Net.Smtp;',
        'using MailKit.Security;',
        'using MimeKit;',
        '',
        'var message = new MimeMessage();',
        'message.From.Add(new MailboxAddress("My App", "app@example.com"));',
        'message.To.Add(new MailboxAddress("User", "user@example.com"));',
        'message.Subject = "Hello from MailKit";',
        'message.Body = new TextPart("plain") { Text = "It works!" };',
        '',
        'using var client = new SmtpClient();',
        `await client.ConnectAsync(${dq(host)}, ${port}, SecureSocketOptions.None);`,
        auth && `await client.AuthenticateAsync(${dq(user)}, ${dq(pass)});`,
        'await client.SendAsync(message);',
        'await client.DisconnectAsync(true);',
      ),
    },
    {
      id: 'fluentemail',
      label: 'ASP.NET Core (FluentEmail)',
      group: '.NET (C#)',
      language: 'Program.cs',
      install: 'dotnet add package FluentEmail.Smtp',
      keywords: ['aspnet', 'asp.net'],
      code: lines(
        'builder.Services',
        '    .AddFluentEmail("app@example.com")',
        auth
          ? `    .AddSmtpSender(${dq(host)}, ${port}, ${dq(user)}, ${dq(pass)});`
          : `    .AddSmtpSender(${dq(host)}, ${port});`,
      ),
    },

    // ── Go ───────────────────────────────────────────────────────────────
    {
      id: 'go',
      label: 'Go (net/smtp)',
      group: 'Go',
      language: 'Go',
      keywords: ['golang'],
      code: lines(
        'package main',
        '',
        'import "net/smtp"',
        '',
        'func main() {',
        '\tmsg := []byte("From: app@example.com\\r\\n" +',
        '\t\t"To: user@example.com\\r\\n" +',
        '\t\t"Subject: Hello from Go\\r\\n\\r\\n" +',
        '\t\t"It works!\\r\\n")',
        '',
        auth ? `\tauth := smtp.PlainAuth("", ${dq(user)}, ${dq(pass)}, ${dq(host)})` : '\tvar auth smtp.Auth // no authentication',
        `\terr := smtp.SendMail(${dq(addr)}, auth, "app@example.com", []string{"user@example.com"}, msg)`,
        '\tif err != nil {',
        '\t\tpanic(err)',
        '\t}',
        '}',
      ),
    },

    // ── Rust ─────────────────────────────────────────────────────────────
    {
      id: 'rust',
      label: 'Rust (lettre)',
      group: 'Rust',
      language: 'Rust',
      install: 'cargo add lettre',
      code: lines(
        auth
          ? 'use lettre::{transport::smtp::authentication::Credentials, Message, SmtpTransport, Transport};'
          : 'use lettre::{Message, SmtpTransport, Transport};',
        '',
        'fn main() {',
        '    let email = Message::builder()',
        '        .from("My App <app@example.com>".parse().unwrap())',
        '        .to("user@example.com".parse().unwrap())',
        '        .subject("Hello from Rust")',
        '        .body(String::from("It works!"))',
        '        .unwrap();',
        '',
        '    // builder_dangerous = plain SMTP without TLS (fine for local testing)',
        `    let mailer = SmtpTransport::builder_dangerous(${dq(host)})`,
        `        .port(${port})`,
        auth && `        .credentials(Credentials::new(${dq(user)}.into(), ${dq(pass)}.into()))`,
        '        .build();',
        '',
        '    mailer.send(&email).unwrap();',
        '}',
      ),
    },

    // ── Elixir ───────────────────────────────────────────────────────────
    {
      id: 'phoenix',
      label: 'Phoenix (Swoosh)',
      group: 'Elixir',
      language: 'config/dev.exs',
      install: 'Add {:gen_smtp, "~> 1.2"} to mix.exs deps',
      keywords: ['elixir', 'swoosh'],
      code: lines(
        'config :my_app, MyApp.Mailer,',
        '  adapter: Swoosh.Adapters.SMTP,',
        `  relay: ${dq(host)},`,
        `  port: ${port},`,
        '  ssl: false,',
        '  tls: :never,',
        auth ? `  username: ${dq(user)},` : null,
        auth ? `  password: ${dq(pass)},` : null,
        auth ? '  auth: :always,' : '  auth: :never,',
        '  retries: 1',
      ),
    },

    // ── Other languages ──────────────────────────────────────────────────
    {
      id: 'perl',
      label: 'Perl (Net::SMTP)',
      group: 'Other languages',
      language: 'Perl',
      code: lines(
        'use Net::SMTP;',
        '',
        `my $smtp = Net::SMTP->new(${sq(host)}, Port => ${port}) or die "Cannot connect: $@";`,
        auth && `$smtp->auth(${sq(user)}, ${sq(pass)});`,
        "$smtp->mail('app@example.com');",
        "$smtp->to('user@example.com');",
        '$smtp->data();',
        '$smtp->datasend("From: app\\@example.com\\nTo: user\\@example.com\\nSubject: Hello from Perl\\n\\nIt works!\\n");',
        '$smtp->dataend();',
        '$smtp->quit;',
      ),
    },
    {
      id: 'dart',
      label: 'Dart / Flutter (mailer)',
      group: 'Other languages',
      language: 'Dart',
      install: 'dart pub add mailer',
      keywords: ['flutter'],
      code: lines(
        "import 'package:mailer/mailer.dart';",
        "import 'package:mailer/smtp_server.dart';",
        '',
        'Future<void> main() async {',
        `  final server = SmtpServer(${sq(host)},`,
        `      port: ${port},`,
        '      ssl: false,',
        '      allowInsecure: true,',
        auth && `      username: ${sq(user)},`,
        auth && `      password: ${sq(pass)},`,
        '  );',
        '',
        '  final message = Message()',
        "    ..from = const Address('app@example.com', 'My App')",
        "    ..recipients.add('user@example.com')",
        "    ..subject = 'Hello from Dart'",
        "    ..text = 'It works!';",
        '',
        '  await send(message, server);',
        '}',
      ),
    },

    // ── Command line ─────────────────────────────────────────────────────
    {
      id: 'curl',
      label: 'cURL',
      group: 'Command line',
      language: 'Shell',
      keywords: ['bash', 'cmd', 'terminal', 'libcurl', 'c', 'c++'],
      note: 'Save the message as email.txt first. curl ships with Windows 10 and later. The same options work with libcurl from C/C++.',
      code: lines(
        '# email.txt',
        '# From: My App <app@example.com>',
        '# To: user@example.com',
        '# Subject: Hello from curl',
        '#',
        '# It works!',
        '',
        `curl smtp://${addr} --mail-from app@example.com --mail-rcpt user@example.com --upload-file email.txt` +
          (auth ? ` --user "${user}:${pass}"` : ''),
      ),
    },
    {
      id: 'powershell',
      label: 'PowerShell',
      group: 'Command line',
      language: 'PowerShell',
      keywords: ['windows', 'send-mailmessage'],
      note: 'Send-MailMessage is marked obsolete for internet mail but works fine for local testing.',
      code: lines(
        auth && `$cred = New-Object PSCredential(${sq(user)}, (ConvertTo-SecureString ${sq(pass)} -AsPlainText -Force))`,
        `Send-MailMessage -SmtpServer ${host} -Port ${port} \``,
        '  -From "app@example.com" -To "user@example.com" `',
        auth ? '  -Subject "Hello from PowerShell" -Body "It works!" `' : '  -Subject "Hello from PowerShell" -Body "It works!"',
        auth && '  -Credential $cred',
      ),
    },
    {
      id: 'swaks',
      label: 'swaks',
      group: 'Command line',
      language: 'Shell',
      keywords: ['swiss army knife'],
      code:
        `swaks --server ${host} --port ${port} --from app@example.com --to user@example.com ` +
        '--header "Subject: Hello from swaks" --body "It works!"' +
        (auth ? ` --auth PLAIN --auth-user ${user} --auth-password ${pass}` : ''),
    },

    // ── Containers ───────────────────────────────────────────────────────
    {
      id: 'docker',
      label: 'Docker Compose',
      group: 'Containers',
      language: 'docker-compose.yml',
      keywords: ['container', 'wsl', 'host.docker.internal'],
      note: 'Inside a container, 127.0.0.1 is the container itself. Use host.docker.internal, and if the connection is refused set Listen address to 0.0.0.0 in Server settings.',
      code: lines(
        'services:',
        '  app:',
        '    environment:',
        '      MAIL_HOST: host.docker.internal',
        `      MAIL_PORT: "${port}"`,
        auth && `      MAIL_USERNAME: ${dq(user)}`,
        auth && `      MAIL_PASSWORD: ${dq(pass)}`,
        '    extra_hosts:',
        '      - "host.docker.internal:host-gateway"',
      ),
    },

    // ── Generic ──────────────────────────────────────────────────────────
    {
      id: 'generic',
      label: 'Any SMTP client',
      group: 'Generic',
      language: 'Settings',
      keywords: ['other', 'manual', 'thunderbird', 'outlook'],
      note: 'Use these values in any application or library that can send email over SMTP.',
      code: lines(
        `Host:        ${host}`,
        `Port:        ${port}`,
        `Security:    None (no SSL/TLS, no STARTTLS)`,
        auth ? `Username:    ${user}` : 'Username:    (empty, or any value)',
        auth ? `Password:    ${pass}` : 'Password:    (empty, or any value)',
        `Auth:        ${auth ? 'PLAIN or LOGIN' : 'Not required'}`,
      ),
    },
  ]
}
