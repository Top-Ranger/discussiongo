CREATE TABLE discussiongo.authtoken (id VARCHAR(600) NOT NULL PRIMARY KEY, user TEXT NOT NULL, validUntil INTEGER NOT NULL);
CREATE INDEX discussiongo.idx_authtoken_id ON authtoken (id);
UPDATE discussiongo.meta SET value='MySQL-2' WHERE mkey='version';
