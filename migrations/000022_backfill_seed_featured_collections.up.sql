INSERT INTO business_featured_collections (business_id, collection_id)
SELECT b.id, mapping.collection_id
FROM businesses b
JOIN (
  VALUES
    ('hotel', 'romantic_getaways'), ('resort', 'romantic_getaways'), ('villa', 'romantic_getaways'), ('beach_villa', 'romantic_getaways'),
    ('resort', 'family_friendly'), ('cottage', 'family_friendly'), ('vacation_home', 'family_friendly'), ('pool_villa', 'family_friendly'),
    ('apartment', 'weekend_escapes'), ('aparthotel', 'weekend_escapes'), ('hostel', 'weekend_escapes'), ('hotel', 'weekend_escapes'), ('guesthouse', 'weekend_escapes'),
    ('beach_villa', 'beachfront_stays'),
    ('cottage', 'pet_friendly'), ('cabin', 'pet_friendly'), ('mountain_cabin', 'pet_friendly'), ('glamping', 'pet_friendly'),
    ('pool_villa', 'pool_stays'), ('villa', 'pool_stays'), ('resort', 'pool_stays'),
    ('mountain_cabin', 'mountain_escapes'), ('cabin', 'mountain_escapes'), ('glamping', 'mountain_escapes'),
    ('apartment', 'city_breaks'), ('aparthotel', 'city_breaks'), ('hotel', 'city_breaks'), ('hostel', 'city_breaks'),
    ('massage_spa', 'wellness_spa'), ('massage_therapy', 'wellness_spa'), ('spa_wellness', 'wellness_spa'), ('physiotherapy', 'wellness_spa'), ('personal_training', 'wellness_spa'),
    ('hair_salon', 'beauty_grooming'), ('barbershop', 'beauty_grooming'), ('beauty_salon', 'beauty_grooming'), ('nail_salon', 'beauty_grooming'), ('tattoo_piercing', 'beauty_grooming'),
    ('electrician', 'home_repairs'), ('plumber', 'home_repairs'), ('locksmith', 'home_repairs'), ('hvac_service', 'home_repairs'), ('painter_decorator', 'home_repairs'), ('cleaning_service', 'home_repairs'),
    ('automotive_service', 'auto_services'), ('car_wash_detailing', 'auto_services'),
    ('dental_clinic', 'health_care'), ('medical_clinic', 'health_care'), ('physiotherapy', 'health_care'), ('personal_training', 'health_care'),
    ('tutoring', 'learn_grow'), ('veterinary_pet_care', 'pet_care'),
    ('photography_videography', 'professional_services'), ('legal_consultation', 'professional_services'), ('accounting_consultation', 'professional_services'), ('professional_service', 'professional_services')
) AS mapping(category_id, collection_id) ON mapping.category_id = b.category_id
ON CONFLICT (business_id, collection_id) DO NOTHING;
