-- migrate:up
CREATE VIRTUAL TABLE IF NOT EXISTS videos_fts USING fts5(id, title, channel);
INSERT INTO videos_fts SELECT v.id, v.title, c.name FROM videos v, channels c ON (v.channel_id=c.id);

-- migrate:down
DROP TABLE videos_fts;

