CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       telegram_id BIGINT UNIQUE NOT NULL
);

CREATE TABLE links (
                       id SERIAL PRIMARY KEY,
                       url TEXT UNIQUE NOT NULL,
                       changed_at TIMESTAMP DEFAULT now()
);

CREATE TABLE link_users (
                            user_id INT REFERENCES users(id) ON DELETE CASCADE,
                            link_id INT REFERENCES links(id) ON DELETE CASCADE,
                            PRIMARY KEY (user_id, link_id)
);

CREATE TABLE filters (
                         id SERIAL PRIMARY KEY,
                         name TEXT UNIQUE NOT NULL
);

CREATE TABLE link_filters (
                              link_id INT REFERENCES links(id) ON DELETE CASCADE,
                              filter_id INT REFERENCES filters(id) ON DELETE CASCADE,
                              PRIMARY KEY (link_id, filter_id)
);

CREATE TABLE tags (
                      id SERIAL PRIMARY KEY,
                      name TEXT UNIQUE NOT NULL
);

CREATE TABLE link_tags (
                           link_id INT REFERENCES links(id) ON DELETE CASCADE,
                           tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
                           PRIMARY KEY (link_id, tag_id)
);
