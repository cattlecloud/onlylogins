-- init-db.sql; initialize the database, including migrations
-- APPEND ONLY ONCE IN PRODUCTION

create table if not exists standings (
    id integer primary key,
    -- enum

    label text check(length(label) < 16) not null unique
    -- the string label of the user standing
) strict;

insert or ignore into standings (id, label) values
    (1, 'Normal'),
    (2, "Admin"),
    (3, "Moderator"),
    (4, "Banned"),
    (5, "Sentinel");

create table if not exists users (
    id integer primary key autoincrement,
    -- auto

    username text check(length(username) > 1 and length(username) < 48) not null unique,
    -- the chosen username (unique)

    standings_id integer not null default 1,
    -- foreign key into standings table

    creation integer default (unixepoch('now')) not null,
    -- creation timestamp

    modification integer default (unixepoch('now')) not null,
    -- modification timestamp

    foreign key (standings_id) references standings(id)
) strict;

create table if not exists authorize (
    users_id integer primary key not null,
    -- foreign key into users table

    shibboleth text check(length(shibboleth) == 36) not null,
    -- what the passphrase encrypts

    expectation blob check(length(expectation) > 0) not null,
    -- what is expected when the user encrypts the shibboleth

    foreign key (users_id) references users(id)
) strict;
