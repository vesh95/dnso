ALTER TABLE zones ADD COLUMN serial INTEGER;
UPDATE zones SET serial = strftime('%f', 'now') * 1000