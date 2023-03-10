CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE channels (
	id VARCHAR(64) NOT NULL,
	url VARCHAR(256) NOT NULL,
	name VARCHAR(64) NOT NULL,
	rss VARCHAR(256) NOT NULL,
	autodownload INTEGER NOT NULL,
	PRIMARY KEY (id)
);
CREATE TABLE videos (
	id VARCHAR(64) NOT NULL,
	url VARCHAR(256) NOT NULL,
	title VARCHAR(256) NOT NULL,
	published DATETIME NOT NULL,
	channel_id INTEGER NOT NULL,
	downloaded INTEGER NOT NULL,
	PRIMARY KEY (id),
	CONSTRAINT fk_channel
		FOREIGN KEY(channel_id)
		REFERENCES channels (id)
		ON DELETE CASCADE
);
CREATE VIRTUAL TABLE videos_fts USING fts5(id, title, channel)
/* videos_fts(id,title,channel) */;
CREATE TABLE IF NOT EXISTS 'videos_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE IF NOT EXISTS 'videos_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'videos_fts_content'(id INTEGER PRIMARY KEY, c0, c1, c2);
CREATE TABLE IF NOT EXISTS 'videos_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'videos_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('01'),
  ('02');
