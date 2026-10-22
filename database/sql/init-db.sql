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

    email text check(length(email) >= 3 and length(email) < 128) not null,
    -- the email address (not unique)

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

    salt blob check(length(salt) == 16) not null,
    -- the per-user salt
    -- argon2id(passphrase, salt) = key

    verifier blob check(length(verifier) > 0) not null,
    -- the key check value for verifying the passphrase
    -- hmac-sha256(key, "onlylogins-verifier") = verifier

    foreign key (users_id) references users(id) on delete cascade
) strict;

create table if not exists logins (
    id integer primary key autoincrement,
    -- auto

    users_id integer not null,
    -- foreign key into users; the user logging in [attempt]

    ip text check(length(ip) > 0 and length(ip) < 64) not null,
    -- the ip address of the client

    ua text check(length(ua) < 512) not null,
    -- the user agent of the client

    success integer check(success in (0, 1)) not null,
    -- enum; 1: success, 0: failure

    creation integer default (unixepoch('now')) not null,
    -- timestamp of the login attempt

    foreign key (users_id) references users(id) on delete cascade
) strict;

create table if not exists bogus (
    id integer primary key autoincrement,
    -- auto

    ip text check(length(ip) > 0 and length(ip) < 64) not null,
    -- the ip address of the client

    ua text check(length(ua) < 512) not null,
    -- the user agent of the client

    creation integer default (unixepoch('now')) not null
    -- timestamp of the login attempt
) strict;

create table if not exists folders (
    id integer primary key autoincrement,
    -- auto

    users_id integer not null,
    -- foreign key into users; the user owning the folder

    label text check(length(label) > 0 and length(label) < 32) not null,
    -- the name of the folder; unique per user

    icon integer not null check(icon >= 0 and icon <= 11) not null,
    -- enum; the associated icon

    perma integer not null check(perma in (0, 1)) default 0,
    -- enum; 1: permenant, 0: can be modified / deleted

    unique(users_id, label),
    foreign key (users_id) references users(id) on delete cascade
) strict;

-- upon inserting into the users table, automatically create a default
-- "miscellaneous" folder for the new user (icon = 7, perma = 1)

create trigger if not exists default_folder_after_user_insert
after insert on users
for each row
begin
    insert into folders (users_id, label, icon, perma)
    values (new.id, "miscellaneous", 7, 1);
end;
