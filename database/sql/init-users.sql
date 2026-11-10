-- Copyright (c) 2026 Cattle Cloud LLC
-- SPDX-License-Identifier: AGPL-3.0
--
-- init-users.sql; initialize default/built-in users
-- pre-fill with sentinel user(s)
insert or ignore into users (username, email, standings_id) values
    ("_nobody", "nobody@onlylogins.invalid", 5);
