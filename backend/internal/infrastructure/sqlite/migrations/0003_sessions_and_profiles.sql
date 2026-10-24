-- Profile flags shown on the profile page.
ALTER TABLE students ADD COLUMN is_staff INTEGER NOT NULL DEFAULT 0;
ALTER TABLE students ADD COLUMN is_alumni INTEGER NOT NULL DEFAULT 0;

-- When the student first signed in; NULL for students who were only viewed.
-- Everyone stored so far signed in.
ALTER TABLE students ADD COLUMN registered_at TEXT;
UPDATE students SET registered_at = created_at;

-- Intra slug of each project, for links.
ALTER TABLE student_projects ADD COLUMN slug TEXT NOT NULL DEFAULT '';

-- Sign-in sessions; only the SHA-256 of the cookie token is stored.
CREATE TABLE sessions (
    token_hash TEXT    PRIMARY KEY,
    student_id INTEGER NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at INTEGER NOT NULL -- unix seconds
);

CREATE INDEX sessions_expires_at ON sessions (expires_at);
