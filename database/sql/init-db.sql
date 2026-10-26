-- SPDX-License-Identifier: AGPL-3.0
-- Copyright (c) 2026 Cattle Cloud LLC
--
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

    icon integer not null check(icon >= 0 and icon <= 17) not null,
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

create table if not exists documents (
    id integer primary key autoincrement,
    -- auto

    folders_id integer not null,
    -- foreign key into folders; the folder in which this document belongs

    label text check(length(label) > 0 and length(label) < 32) not null,
    -- the name of the document; unique per folder

    creation integer default (unixepoch('now')) not null,
    -- creation timestamp

    modification integer default (unixepoch('now')) not null,
    -- modification timestamp

    unique(folders_id, label),
    foreign key (folders_id) references folders(id) on delete cascade
) strict;

create table if not exists weblinks (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 1024) not null,
    -- encrypted weblink value

    foreign key (did) references documents(id) on delete cascade
) strict;

create table if not exists usernames (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 256) not null,
    -- encrypted username value

    foreign key (did) references documents(id) on delete cascade
) strict;

create table if not exists passwords (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 1024) not null,
    -- encrypted password value

    foreign key (did) references documents(id) on delete cascade
) strict;

create table if not exists keyvals (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    k text check(length(k) > 0 and length(k) < 1024) not null,
    -- the name of the key:value pair

    v blob check(length(v) > 0 and length(v) < 1024) not null,
    -- encrypted value of the key:value pair

    unique(did, k),
    -- at most one row per (document, key) pair

    foreign key (did) references documents(id) on delete cascade
) strict;

create table if not exists notes (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 8192) not null,
    -- encrypted content of the note

    unique(did),
    -- at most one note per document

    foreign key (did) references documents(id) on delete cascade
) strict;

create table if not exists cron_states (
    id integer primary key,
    -- enum

    label text check(length(label) < 10) not null unique
    -- the string label of the cron state
) strict;

insert or ignore into cron_states (id, label) values
    (1, 'disable'),
    (2, 'running'),
    (3, 'stopped'),
    (4, 'pending');

create table if not exists crons (
    id integer primary key autoincrement,
    -- auto

    label text check(length(label) > 1) not null unique,
    -- the distinct name of the cronjob

    states_id integer not null default 4,
    -- enum of the current state of the cron

    modification integer default (unixepoch('now')) not null,
    -- latest update timestamp

    foreign key(states_id) references cron_states(id) on delete cascade
) strict;

create table if not exists cron_runs (
    id integer primary key autoincrement,
    -- auto

    crons_id integer not null,
    -- foreign key into crons; the cron that ran and stopped

    code integer check(code >=0 and code <= 255) not null default 0,
    -- the exit code of the latest attempt

    started integer not null default 0,
    -- the time at which the cron last started running

    stopped integer not null default 0,
    -- the time at which the latest run stopped (success or failure)

    foreign key(crons_id) references crons(id) on delete cascade
) strict;

create index if not exists idx_cron_runs_started on cron_runs(started);
create index if not exists idx_cron_runs_crons_id on cron_runs(crons_id);

-- trigger for initializing an empty cron_runs row when a cron is created
create trigger if not exists init_cron_runs_zero after insert on crons
for each row begin
    insert into cron_runs (crons_id) values (NEW.id);
end;
