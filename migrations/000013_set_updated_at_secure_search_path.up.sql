-- Trigger functions run in the caller's session. Pin the lookup path so an
-- untrusted schema cannot shadow names used by this function.
ALTER FUNCTION public.set_updated_at() SET search_path = pg_catalog;
