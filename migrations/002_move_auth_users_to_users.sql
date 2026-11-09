BEGIN;

DO $migration$
DECLARE
    sequence_name TEXT;
BEGIN
    IF to_regclass('public.auth_users') IS NULL THEN
        RETURN;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.auth_users old_user
        JOIN public.users existing_user
          ON existing_user.id = old_user.id
          OR LOWER(existing_user.username) = LOWER(old_user.username)
          OR LOWER(existing_user.email) = LOWER(old_user.email)
    ) THEN
        RAISE EXCEPTION 'Cannot migrate auth_users: an ID, username, or email already exists in users';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM public.auth_users
        WHERE id < 1 OR id > 2147483647
           OR length(username) > 50
           OR length(email) > 100
           OR length(password_hash) > 255
    ) THEN
        RAISE EXCEPTION 'Cannot migrate auth_users: account data does not fit the users table';
    END IF;

    INSERT INTO public.users
        (id, username, email, password_hash, is_active, created_at, updateda_at)
    SELECT
        id::INTEGER,
        username,
        email,
        password_hash,
        CASE WHEN is_active THEN B'1' ELSE B'0' END,
        created_at::TIMESTAMP,
        updated_at::TIMESTAMP
    FROM public.auth_users;

    sequence_name := pg_get_serial_sequence('public.users', 'id');
    IF sequence_name IS NULL THEN
        RAISE EXCEPTION 'Cannot migrate auth_users: users.id has no sequence';
    END IF;

    PERFORM setval(
        sequence_name::regclass,
        COALESCE((SELECT MAX(id)::BIGINT FROM public.users), 1),
        EXISTS (SELECT 1 FROM public.users)
    );

    DROP TABLE public.auth_users;
END
$migration$;

COMMIT;
