-- init-db.sql; initialize the database, including migrations
-- APPEND ONLY ONCE IN PRODUCTION

create table if not exists oauths(
    id integer primary key,
    -- enum

    label text check(length(label) < 16) not null unique
    -- the string label of the oauth identity provider
) strict;

insert or ignore into oauths (id, label) values
    (0, 'None'),
    (1, 'Testing'),
    (2, 'Google'),
    (3, 'Apple'),
    (4, 'OnlyLogins');

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

    sub text check(length(sub) >= 8 and length(sub) <= 64),
    -- JWT sub field as assigned by oauth provider

    oauths_id integer not null,
    -- foreign key into the associated oauth provider (unique w/ sub)

    username text check(length(username) > 1 and length(username) < 24) not null unique,
    -- the chosen or default username (unique)

    email text check(length(email) < 128) not null,
    -- the email address (not a key, not unique)

    upsert integer default (cast(unixepoch('now') as int)) not null,
    -- upsert (modification) timestamp

    standings_id integer not null default 1,
    -- foreign key into standings table

    creation integer default (cast(unixepoch('now') as int)) not null,
    -- creation timestamp

    unique(sub, oauths_id) -- alternate key
    foreign key (oauths_id) references oauths(id)
    foreign key (standings_id) references standings(id)
) strict;

create table if not exists usernamechanges (
    id integer primary key autoincrement,
    -- auto

    users_id integer not null,
    -- foreign key into users; the one making a change

    chosen text not null,
    -- the newly chosen username

    previous text not null,
    -- the previously held username

    modification integer default (cast(unixepoch('now') as int)) not null,
    -- modification timestamp

    foreign key(users_id) references users(id) on delete cascade
) strict;

-- trigger for tracking changes to usernames
create trigger if not exists track_usernamechanges after update of username on users
for each row begin
    insert into usernamechanges(users_id, chosen, previous)
    values (NEW.id, NEW.username, OLD.username);
end;

create table if not exists userprofiles (
    users_id integer primary key not null,
    -- foreign key into users; this is their profile information

    company text default '' check(length(company) < 64) not null,
    -- a company the user associates with

    loc text default '' check(length(loc) < 64) not null,
    -- a location the user associates with

    link text default '' check(length(link) < 180) not null,
    -- a url the user associates with

    showemail integer default 0 not null,
    -- a boolean indicating whether to share email address

    about text default '' check(length(about) < 512) not null,
    -- a small bit of text about the user

    foreign key(users_id) references users(id) on delete cascade
) strict;

-- trigger for creating user profile upon insertion into users
create trigger if not exists track_users_create_profile after insert on users
begin
    insert into userprofiles (users_id) values (NEW.id);
end;
