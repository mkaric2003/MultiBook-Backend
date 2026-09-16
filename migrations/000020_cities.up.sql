CREATE TABLE cities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  normalized_name TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT cities_name_not_blank CHECK (length(btrim(name)) > 0)
);

CREATE TRIGGER cities_set_updated_at
BEFORE UPDATE ON cities
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO cities (name, normalized_name)
SELECT MIN(BTRIM(city)), city_normalized
FROM business_locations
WHERE BTRIM(city) <> ''
GROUP BY city_normalized
ON CONFLICT (normalized_name) DO NOTHING;

CREATE FUNCTION register_business_location_city() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO cities (name, normalized_name)
  VALUES (BTRIM(NEW.city), NEW.city_normalized)
  ON CONFLICT (normalized_name) DO NOTHING;
  RETURN NEW;
END;
$$;

CREATE TRIGGER business_locations_register_city
AFTER INSERT OR UPDATE OF city, city_normalized ON business_locations
FOR EACH ROW EXECUTE FUNCTION register_business_location_city();
