-- init-users.sql; initialize default/built-in users

-- pre-fill with sentinel user(s)
insert or ignore into users (username, standings_id) values
    ("_nobody", 5);
