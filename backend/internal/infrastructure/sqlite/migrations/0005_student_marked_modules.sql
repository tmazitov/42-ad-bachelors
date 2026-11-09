-- Modules a student marked completed themselves: courses whose results are
-- recorded outside 42 (GEN courses, capstone, internship). Only the
-- curriculum's self-reported modules count toward progress.
CREATE TABLE student_marked_modules (
    student_id  INTEGER NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    module_code TEXT    NOT NULL,
    marked_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (student_id, module_code)
);
