-- Students who logged in through the app. id is the 42 intra user id.
CREATE TABLE students (
    id           INTEGER PRIMARY KEY,
    login        TEXT    NOT NULL UNIQUE,
    display_name TEXT    NOT NULL,
    image_url    TEXT,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
