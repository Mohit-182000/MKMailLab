-- Core mail storage. Hot list columns live in `emails`; bodies, raw source and
-- headers live in side tables so inbox queries never touch large values.

CREATE TABLE emails (
    id                INTEGER PRIMARY KEY,
    message_id        TEXT    NOT NULL DEFAULT '',
    subject           TEXT    NOT NULL DEFAULT '',
    from_name         TEXT    NOT NULL DEFAULT '',
    from_addr         TEXT    NOT NULL DEFAULT '',
    to_json           TEXT    NOT NULL DEFAULT '[]', -- denormalised for list display
    envelope_from     TEXT    NOT NULL DEFAULT '',
    date_header       INTEGER,                       -- unix ms, NULL if absent/invalid
    received_at       INTEGER NOT NULL,              -- unix ms (UTC)
    size_bytes        INTEGER NOT NULL DEFAULT 0,
    has_html          INTEGER NOT NULL DEFAULT 0,
    has_text          INTEGER NOT NULL DEFAULT 0,
    attachment_count  INTEGER NOT NULL DEFAULT 0,
    snippet           TEXT    NOT NULL DEFAULT '',
    is_read           INTEGER NOT NULL DEFAULT 0,
    is_starred        INTEGER NOT NULL DEFAULT 0,
    remote_addr       TEXT    NOT NULL DEFAULT '',
    helo              TEXT    NOT NULL DEFAULT '',
    parse_status      TEXT    NOT NULL DEFAULT 'ok',
    parse_errors_json TEXT    NOT NULL DEFAULT '[]'
);

CREATE INDEX idx_emails_received   ON emails (received_at DESC, id DESC);
CREATE INDEX idx_emails_from       ON emails (from_addr);
CREATE INDEX idx_emails_subject    ON emails (subject);
CREATE INDEX idx_emails_message_id ON emails (message_id);
CREATE INDEX idx_emails_unread     ON emails (received_at DESC) WHERE is_read = 0;

CREATE TABLE email_bodies (
    email_id  INTEGER PRIMARY KEY REFERENCES emails (id) ON DELETE CASCADE,
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT ''
);

CREATE TABLE email_raw (
    email_id INTEGER PRIMARY KEY REFERENCES emails (id) ON DELETE CASCADE,
    raw      BLOB    NOT NULL
);

CREATE TABLE email_recipients (
    id       INTEGER PRIMARY KEY,
    email_id INTEGER NOT NULL REFERENCES emails (id) ON DELETE CASCADE,
    kind     TEXT    NOT NULL CHECK (kind IN ('to', 'cc', 'bcc', 'reply_to', 'envelope')),
    name     TEXT    NOT NULL DEFAULT '',
    address  TEXT    NOT NULL
);

CREATE INDEX idx_recipients_address ON email_recipients (address, email_id);
CREATE INDEX idx_recipients_email   ON email_recipients (email_id);

CREATE TABLE email_headers (
    email_id INTEGER NOT NULL REFERENCES emails (id) ON DELETE CASCADE,
    ordinal  INTEGER NOT NULL,
    name     TEXT    NOT NULL,
    value    TEXT    NOT NULL,
    PRIMARY KEY (email_id, ordinal)
) WITHOUT ROWID;

CREATE TABLE email_attachments (
    id           INTEGER PRIMARY KEY,
    email_id     INTEGER NOT NULL REFERENCES emails (id) ON DELETE CASCADE,
    part_index   INTEGER NOT NULL,
    filename     TEXT    NOT NULL DEFAULT '',
    content_type TEXT    NOT NULL DEFAULT '',
    size_bytes   INTEGER NOT NULL DEFAULT 0,
    content_id   TEXT    NOT NULL DEFAULT '',
    inline       INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_attachments_email ON email_attachments (email_id);

CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT    NOT NULL,
    updated_at INTEGER NOT NULL
) WITHOUT ROWID;
