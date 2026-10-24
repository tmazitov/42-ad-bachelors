-- Validated 42 projects per student, replaced on every sync. Progress is
-- computed from these against the curriculum at read time.
CREATE TABLE student_projects (
    student_id INTEGER NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    project_id INTEGER NOT NULL,
    final_mark INTEGER NOT NULL,
    marked_at  TEXT,
    PRIMARY KEY (student_id, project_id)
);

-- NULL until the student's projects are synced for the first time.
ALTER TABLE students ADD COLUMN projects_synced_at TEXT;
