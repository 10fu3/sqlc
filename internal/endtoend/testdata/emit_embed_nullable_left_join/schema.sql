CREATE TABLE authors (
  id   BIGSERIAL PRIMARY KEY,
  name text      NOT NULL,
  bio  text
);

CREATE TABLE posts (
  id          BIGSERIAL PRIMARY KEY,
  title       text      NOT NULL,
  body        text      NOT NULL,
  author_id   bigint    REFERENCES authors(id),
  reviewer_id bigint    REFERENCES authors(id)
);