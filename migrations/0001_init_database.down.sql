CREATE INDEX IF NOT EXISTS idx_link_users_user ON link_users(user_id);
CREATE INDEX IF NOT EXISTS idx_link_users_link ON link_users(link_id);


DROP TABLE IF EXISTS link_tags;
DROP TABLE IF EXISTS link_filters;
DROP TABLE IF EXISTS link_users;

DROP INDEX IF EXISTS idx_links_changed_at;
DROP INDEX IF EXISTS idx_links_id;
DROP INDEX IF EXISTS idx_links_url;

DROP INDEX IF EXISTS idx_users;

DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS filters;
DROP TABLE IF EXISTS links;
DROP TABLE IF EXISTS users;