BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_idx
    ON public.users (LOWER(username));

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_idx
    ON public.users (LOWER(email));

COMMIT;
