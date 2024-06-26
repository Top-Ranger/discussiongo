CREATE DATABASE discussiongo;
CREATE TABLE discussiongo.user (name VARCHAR(600) NOT NULL, salt VARCHAR(600), encodedpasswort VARCHAR(600), admin BOOLEAN, comment LONGTEXT DEFAULT '', invitedby VARCHAR(600) DEFAULT '', invitationdirect BOOL DEFAULT 0, lastseen BIGINT UNSIGNED DEFAULT 0, PRIMARY KEY(name));
CREATE TABLE discussiongo.topic (id BIGINT UNSIGNED AUTO_INCREMENT, name TEXT, creator VARCHAR(600), created BIGINT UNSIGNED, lastmodified BIGINT UNSIGNED, closed BOOL DEFAULT 0, pinned BOOL DEFAULT 0, PRIMARY KEY(id));
CREATE INDEX idx_topic_lastmodified_desc ON discussiongo.topic (lastmodified DESC);
CREATE TABLE discussiongo.post (id BIGINT UNSIGNED AUTO_INCREMENT, content LONGTEXT, poster VARCHAR(600), time BIGINT UNSIGNED, topic BIGINT UNSIGNED, FOREIGN KEY(topic) REFERENCES topic(id) ON UPDATE CASCADE ON DELETE CASCADE, PRIMARY KEY(id));
CREATE INDEX idx_post_topic_time_asc ON discussiongo.post (topic, time ASC);
CREATE TABLE discussiongo.invitations (id VARCHAR(600) NOT NULL, creator VARCHAR(600), FOREIGN KEY(creator) REFERENCES user(name) ON UPDATE CASCADE ON DELETE CASCADE, PRIMARY KEY(id));
CREATE TABLE discussiongo.times (name VARCHAR(600) NOT NULL, topic BIGINT UNSIGNED, time BIGINT UNSIGNED, PRIMARY KEY(name, topic), FOREIGN KEY(name) REFERENCES user(name) ON UPDATE CASCADE ON DELETE CASCADE, FOREIGN KEY(topic) REFERENCES topic(id) ON UPDATE CASCADE ON DELETE CASCADE);
CREATE TABLE discussiongo.events (id BIGINT UNSIGNED AUTO_INCREMENT, type BIGINT UNSIGNED NOT NULL, user VARCHAR(600) NOT NULL, topic VARCHAR(600), date BIGINT UNSIGNED NOT NULL, data BLOB, affecteduser VARCHAR(600), PRIMARY KEY(id));
CREATE INDEX idx_events_topic ON discussiongo.events (topic);
CREATE TABLE discussiongo.files (id BIGINT UNSIGNED AUTO_INCREMENT, name VARCHAR(600) NOT NULL, user VARCHAR(600) NOT NULL, topic VARCHAR(600) NOT NULL, date BIGINT UNSIGNED, data LONGBLOB, FOREIGN KEY(user) REFERENCES user(name) ON UPDATE CASCADE ON DELETE CASCADE, PRIMARY KEY(id));
CREATE INDEX idx_files_user ON discussiongo.files (name);
CREATE INDEX idx_files_topic ON discussiongo.files (topic);
CREATE TABLE discussiongo.authtoken (id VARCHAR(600) NOT NULL PRIMARY KEY, user TEXT NOT NULL, validUntil INTEGER NOT NULL);
CREATE INDEX discussiongo.idx_authtoken_id ON authtoken (id);
CREATE TABLE discussiongo.meta (mkey VARCHAR(600) NOT NULL, value VARCHAR(600), PRIMARY KEY(mkey));
INSERT INTO discussiongo.meta VALUES ('version', 'MySQL-2');