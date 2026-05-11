-- Copyright (C) 2026 Henry Stern
-- SPDX-License-Identifier: MIT
--
-- Bootstrap the SOGo SQL user-source table with the integration test
-- user. Run as user `postgres` against database `sogo` (the database
-- itself + the `sogo` role are created earlier in the Dockerfile).

CREATE TABLE IF NOT EXISTS sogo_users (
    c_uid       VARCHAR(255) PRIMARY KEY,
    c_name      VARCHAR(255),
    c_password  VARCHAR(255),
    c_cn        VARCHAR(255),
    mail        VARCHAR(255)
);

INSERT INTO sogo_users (c_uid, c_name, c_password, c_cn, mail)
VALUES (
    'integration', 'integration', 'integration',
    'Integration Test', 'integration@asmcp.test'
)
ON CONFLICT (c_uid) DO NOTHING;

GRANT ALL ON TABLE sogo_users TO sogo;
