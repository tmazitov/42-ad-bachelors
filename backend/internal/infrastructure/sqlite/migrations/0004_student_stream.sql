-- The stream a student chose (curriculum.StreamChoice key); NULL until they
-- pick one, in which case their best stream is counted.
ALTER TABLE students ADD COLUMN stream TEXT;
