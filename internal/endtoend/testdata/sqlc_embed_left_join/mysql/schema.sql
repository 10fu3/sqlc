CREATE TABLE users (
    id integer NOT NULL PRIMARY KEY,
    name varchar(255) NOT NULL,
    email varchar(255) NOT NULL
);

CREATE TABLE profiles (
    id integer NOT NULL PRIMARY KEY,
    user_id integer NULL,
    bio text NULL,
    avatar_url varchar(255) NULL
);

CREATE TABLE posts (
    id integer NOT NULL PRIMARY KEY,
    user_id integer NOT NULL,
    title varchar(255) NOT NULL,
    content text NOT NULL
);

CREATE TABLE comments (
    id integer NOT NULL PRIMARY KEY,
    post_id integer NULL,
    content text NOT NULL,
    author_name varchar(255) NOT NULL
);