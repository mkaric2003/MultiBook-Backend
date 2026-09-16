CREATE TABLE featured_collections (
  id TEXT PRIMARY KEY,
  business_type business_type NOT NULL,
  title_key TEXT NOT NULL,
  subtitle_key TEXT NOT NULL,
  image_url TEXT NOT NULL,
  position SMALLINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT featured_collections_id_not_blank CHECK (length(btrim(id)) > 0),
  CONSTRAINT featured_collections_position_non_negative CHECK (position >= 0),
  UNIQUE (business_type, position)
);

CREATE TRIGGER featured_collections_set_updated_at
BEFORE UPDATE ON featured_collections
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO featured_collections (id, business_type, title_key, subtitle_key, image_url, position) VALUES
('romantic_getaways', 'stay', 'romanticGetaways', 'perfectForCouples', 'https://images.unsplash.com/photo-1544550285-f813152fb2fd?auto=format&fit=crop&w=1000&q=85', 0),
('family_friendly', 'stay', 'familyFriendly', 'kidApprovedStays', 'https://images.unsplash.com/photo-1540541338287-41700207dee6?auto=format&fit=crop&w=1000&q=85', 1),
('weekend_escapes', 'stay', 'weekendEscapes', 'quickCityBreaks', 'https://images.unsplash.com/photo-1485871981521-5b1fd3805eee?auto=format&fit=crop&w=1000&q=85', 2),
('beachfront_stays', 'stay', 'beachfrontStays', 'oceanViewsIncluded', 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e?auto=format&fit=crop&w=1000&q=85', 3),
('pet_friendly', 'stay', 'petFriendlyStays', 'bringYourBestFriend', 'https://images.unsplash.com/photo-1601758228041-f3b2795255f1?auto=format&fit=crop&w=1000&q=85', 4),
('pool_stays', 'stay', 'poolStays', 'makeASplash', 'https://images.unsplash.com/photo-1540541338287-41700207dee6?auto=format&fit=crop&w=1000&q=85', 5),
('mountain_escapes', 'stay', 'mountainEscapes', 'freshAirAndViews', 'https://images.unsplash.com/photo-1510798831971-661eb04b3739?auto=format&fit=crop&w=1000&q=85', 6),
('city_breaks', 'stay', 'cityBreaks', 'stayCloseToAction', 'https://images.unsplash.com/photo-1519501025264-65ba15a82390?auto=format&fit=crop&w=1000&q=85', 7),
('wellness_spa', 'service', 'wellnessAndSpa', 'relaxAndRejuvenate', 'https://images.unsplash.com/photo-1540555700478-4be289fbecef?auto=format&fit=crop&w=1000&q=85', 0),
('beauty_grooming', 'service', 'beautyAndGrooming', 'lookYourBest', 'https://images.unsplash.com/photo-1562322140-8baeececf3df?auto=format&fit=crop&w=1000&q=85', 1),
('home_repairs', 'service', 'homeRepairs', 'fixItRight', 'https://images.unsplash.com/photo-1621905252507-b35492cc74b4?auto=format&fit=crop&w=1000&q=85', 2),
('auto_services', 'service', 'autoServices', 'keepMoving', 'https://images.unsplash.com/photo-1492144534655-ae79c964c9d7?auto=format&fit=crop&w=1000&q=85', 3),
('health_care', 'service', 'healthAndCare', 'feelYourBest', 'https://images.unsplash.com/photo-1519494026892-80bbd2d6fd0d?auto=format&fit=crop&w=1000&q=85', 4),
('learn_grow', 'service', 'learnAndGrow', 'buildNewSkills', 'https://images.unsplash.com/photo-1523240795612-9a054b0db644?auto=format&fit=crop&w=1000&q=85', 5),
('pet_care', 'service', 'petCare', 'careForEveryCompanion', 'https://images.unsplash.com/photo-1551884831-bbf3cdc6469e?auto=format&fit=crop&w=1000&q=85', 6),
('professional_services', 'service', 'professionalServices', 'expertSupportWhenNeeded', 'https://images.unsplash.com/photo-1450101499163-c8848c66ca85?auto=format&fit=crop&w=1000&q=85', 7);
