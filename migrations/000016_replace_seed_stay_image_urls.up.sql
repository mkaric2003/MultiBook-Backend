-- Seeded stay cards are rendered through Image.network. Replace the former
-- redirect-based picsum URLs with the same direct public HTTPS URLs used by
-- seeded service cards.
UPDATE business_media AS media
SET storage_path = CASE business.category_id
  WHEN 'hotel' THEN 'https://images.unsplash.com/photo-1566073771259-6a8506099945?auto=format&fit=crop&w=1200&q=85'
  WHEN 'resort' THEN 'https://images.unsplash.com/photo-1582719478250-c89cae4dc85b?auto=format&fit=crop&w=1200&q=85'
  WHEN 'apartment' THEN 'https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=85'
  WHEN 'condo' THEN 'https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=85'
  WHEN 'villa' THEN 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=85'
  WHEN 'hostel' THEN 'https://images.unsplash.com/photo-1555854877-bab0e564b8d5?auto=format&fit=crop&w=1200&q=85'
  WHEN 'cabin' THEN 'https://images.unsplash.com/photo-1449157291145-7efd050a4d0e?auto=format&fit=crop&w=1200&q=85'
  WHEN 'cottage' THEN 'https://images.unsplash.com/photo-1449157291145-7efd050a4d0e?auto=format&fit=crop&w=1200&q=85'
  ELSE 'https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?auto=format&fit=crop&w=1200&q=85'
END
FROM businesses AS business
WHERE media.business_id = business.id
  AND business.type = 'stay'
  AND media.media_type IN ('logo', 'cover')
  AND media.storage_path LIKE 'https://picsum.photos/%';
