-- Copyright (c) 2026 Cattle Cloud LLC
-- SPDX-License-Identifier: AGPL-3.0
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

create table if not exists plans (
    id integer primary key,
    -- enum

    label text check(length(label) < 16) not null unique,
    -- the name of the plan

    price integer not null,
    -- the price of the plan in cents

    docs_limit integer not null
    -- the maximum number of documents in this plan
) strict;

insert or ignore into plans (id, label, price, docs_limit) values
    (1, 'free',      0,  256),
    (2, 'patreon', 199, 8192);

create table if not exists users (
    id integer primary key autoincrement,
    -- auto

    username text check(length(username) > 1 and length(username) < 48) not null unique,
    -- the chosen username (unique)

    email text check(length(email) >= 3 and length(email) < 128) not null,
    -- the email address (not unique)

    plans_id integer not null default 1,
    -- the plan the user is subscribed to; default free

    standings_id integer not null default 1,
    -- foreign key into standings table; default normal

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

create index if not exists idx_logins_users_id on logins(users_id);

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
    -- the display name of the folder; unique per user

    slug text check(length(slug) > 0 and length(slug) < 32) not null,
    -- the url-safe name of the folder; unique per user

    icon integer not null check(icon >= 0 and icon <= 17) not null,
    -- enum; the associated icon

    protected integer not null check(protected in (0, 1)) default 0,
    -- enum; 1: permenant, 0: can be modified / deleted

    unique(users_id, label),
    unique(users_id, slug),
    foreign key (users_id) references users(id) on delete cascade
) strict;

-- upon inserting into the users table, automatically create a default
-- "miscellaneous" folder for the new user (icon = 7, protected = 1)
create trigger if not exists default_folder_after_user_insert
after insert on users
for each row
begin
    insert into folders (users_id, label, slug, icon, protected)
    values (new.id, "miscellaneous", "miscellaneous", 7, 1);
end;

create table if not exists documents (
    id integer primary key autoincrement,
    -- auto

    folders_id integer not null,
    -- foreign key into folders; the folder in which this document belongs

    label text check(length(label) > 0 and length(label) < 32) not null,
    -- the display name of the document; unique per folder

    slug text check(length(slug) > 0 and length(slug) < 32) not null,
    -- the url-safe name of the document; unique per folder

    creation integer default (unixepoch('now')) not null,
    -- creation timestamp

    modification integer default (unixepoch('now')) not null,
    -- modification timestamp

    unique(folders_id, label),
    unique(folders_id, slug),
    foreign key (folders_id) references folders(id) on delete cascade
) strict;

create index if not exists idx_documents_folder_listing
    on documents(folders_id, modification desc, label, slug);

create table if not exists weblinks (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 1024) not null,
    -- encrypted weblink value

    foreign key (did) references documents(id) on delete cascade
) strict;

create index if not exists idx_weblinks_did on weblinks(did);

create table if not exists usernames (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 256) not null,
    -- encrypted username value

    foreign key (did) references documents(id) on delete cascade
) strict;

create index if not exists idx_usernames_did on usernames(did);

create table if not exists passwords (
    id integer primary key autoincrement,
    -- auto

    did integer not null,
    -- foreign key into documents

    v blob check(length(v) > 0 and length(v) < 1024) not null,
    -- encrypted password value

    foreign key (did) references documents(id) on delete cascade
) strict;

create index if not exists idx_passwords_did on passwords(did);

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

create index if not exists idx_keyvals_did on keyvals(did);

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

create table if not exists shares (
    uid text check(length(uid) == 36) primary key,
    -- unique uuidv4

    users_id integer not null,
    -- key into users; the creator

    content blob check(length(content) < 8196) not null,
    -- the content of the share; encrypted

    ttl integer check(ttl > 60) not null,
    -- time to live, in seconds

    creation integer default (unixepoch('now')) not null,
    -- timestamp of creation

    foreign key(users_id) references users(id) on delete cascade
) strict;

create index if not exists idx_shares_creation on shares(creation);
create index if not exists idx_shares_users_id on shares(users_id);

create table if not exists stripe_customers (
    users_id integer primary key,
    -- foreign key into users; exactly one stripe customer per user

    customer_id text check(length(customer_id) > 8) not null,
    -- the stripe customer identifier

    creation integer default (unixepoch('now')) not null,
    -- creation timestamp

    foreign key(users_id) references users(id) on delete cascade
) strict;

create index if not exists idx_stripe_customers_customer_id on stripe_customers(customer_id);

create table if not exists stripe_subscriptions (
    id integer primary key autoincrement,
    -- auto

    users_id integer not null,
    -- foreign key into users; the subscriber

    subscription_id text check(length(subscription_id) > 8) not null,
    -- the stripe subscription identifier

    price_id text check(length(price_id) > 8) not null,
    -- the stripe price identifier

    status text check(length(status) < 32) not null,
    -- subscription status (active, past_due, canceled, etc.)

    current_period_end integer not null,
    -- timestamp of current period end

    cancel_at integer not null default 0,
    -- timestamp at which the subscription will be canceled; 0 when not scheduled

    creation integer default (unixepoch('now')) not null,
    -- creation timestamp

    unique (users_id),
    unique (subscription_id),
    foreign key(users_id) references users(id) on delete cascade
) strict;

create index if not exists idx_stripe_subscriptions_expiry   on stripe_subscriptions(status, current_period_end);

create table if not exists stripe_events (
    event_id text primary key,
    -- unique stripe event identifier

    etype text not null,
    -- stripe event type

    livemode integer check(livemode in (0, 1)) not null,
    -- whether the event originated in live mode

    created integer not null,
    -- timestamp the event was created, per stripe

    received integer default (unixepoch('now')) not null,
    -- timestamp we received the event

    payload blob not null,
    -- raw verified json payload, kept for replay

    status text check(status in ('pending', 'done', 'failed', 'dead')) not null default 'pending',
    -- processing state; failed events remain for later retry

    attempts integer not null default 0,
    -- number of processing attempts

    last_error text not null default '',
    -- most recent processing failure

    updated integer default (unixepoch('now')) not null
    -- timestamp of the last state transition
) strict;

create index if not exists idx_stripe_events_status on stripe_events(status, received);

-- 2026-11-18: list documents by slug (lowercased, unique per folder)
-- instead of modification time; the unique(folders_id, slug) autoindex
-- serves this sort, so the old listing index is dropped.
drop index if exists idx_documents_folder_listing;
