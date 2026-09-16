CREATE TYPE business_media_type AS ENUM ('logo', 'cover', 'gallery');

CREATE TABLE business_locations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  city TEXT NOT NULL,
  city_normalized TEXT NOT NULL,
  address TEXT NOT NULL,
  country_code TEXT,
  coordinates GEOGRAPHY(POINT, 4326) NOT NULL,
  is_primary BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT business_locations_city_not_blank CHECK (length(btrim(city)) > 0),
  CONSTRAINT business_locations_address_not_blank CHECK (length(btrim(address)) > 0)
);

CREATE UNIQUE INDEX business_locations_one_primary_idx
  ON business_locations (business_id)
  WHERE is_primary;
CREATE INDEX business_locations_city_idx ON business_locations (city_normalized);
CREATE INDEX business_locations_coordinates_idx ON business_locations USING GIST (coordinates);

CREATE TRIGGER business_locations_set_updated_at
BEFORE UPDATE ON business_locations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE business_media (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
  media_type business_media_type NOT NULL,
  storage_path TEXT NOT NULL,
  position SMALLINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT business_media_storage_path_not_blank CHECK (length(btrim(storage_path)) > 0),
  CONSTRAINT business_media_gallery_position_check CHECK (
    (media_type = 'gallery' AND position BETWEEN 0 AND 6)
    OR (media_type <> 'gallery' AND position IS NULL)
  )
);

CREATE UNIQUE INDEX business_media_one_logo_idx
  ON business_media (business_id) WHERE media_type = 'logo';
CREATE UNIQUE INDEX business_media_one_cover_idx
  ON business_media (business_id) WHERE media_type = 'cover';
CREATE UNIQUE INDEX business_media_gallery_position_idx
  ON business_media (business_id, position) WHERE media_type = 'gallery';
