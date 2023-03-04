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
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('01');
