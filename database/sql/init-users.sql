-- init-users.sql; initialize default/built-in users

-- pre-fill with sentinel user(s)
insert or ignore into users (sub, oauths_id, username, email, standings_id) values
    ("sentinel-1", 0, "_nobody", "nobody@onlylogins.net", 5);

-- pre-fill with testcase user(s)
insert or ignore into users (sub, oauths_id, username, email) values
    ("testuser002", 1, "xav",  "xav@test.local"),
    ("testuser003", 1, "yumi", "yumi@test.local");
