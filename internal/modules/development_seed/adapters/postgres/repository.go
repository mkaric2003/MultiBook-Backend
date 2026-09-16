package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mkaric2003/multibook-backend/internal/platform/database"
)

type Repository struct{ pool *pgxpool.Pool }

// The catalog is mechanically exported from the Firebase demo definitions in
// the mobile app, so development seed data stays aligned during the migration.
//
//go:embed firebase_demo_catalog.json
var firebaseDemoCatalogJSON []byte

type demoCatalog struct {
	Stays    []demoStay    `json:"stays"`
	Services []demoService `json:"services"`
}
type demoStay struct {
	Name, CategoryID, City, Address, Description string
	Latitude, Longitude, Rating                  float64
	Price, Reviews                               int
}
type demoService struct {
	Name, CategoryID, ServiceName, SecondaryServiceName, City, Address, Description string
	Price, Duration, SecondaryPrice, SecondaryDuration, Reviews                     int
	Rating                                                                          float64
}

func firebaseDemoCatalog() (demoCatalog, error) {
	var catalog demoCatalog
	if err := json.Unmarshal(firebaseDemoCatalogJSON, &catalog); err != nil {
		return demoCatalog{}, fmt.Errorf("decode Firebase demo catalog: %w", err)
	}
	return catalog, nil
}

func NewRepository(p *pgxpool.Pool) *Repository { return &Repository{p} }

func (r *Repository) SeedStays(ctx context.Context, owner string) (int, error) {
	created := 0
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		count, err := r.seedStaysInTransaction(ctx, tx, owner)
		if err != nil {
			return err
		}
		created = count
		return nil
	})
	return created, err
}

func (r *Repository) seedStaysInTransaction(ctx context.Context, tx pgx.Tx, owner string) (int, error) {
	catalog, err := firebaseDemoCatalog()
	if err != nil {
		return 0, err
	}
	var selectedBusinessID uuid.UUID
	for i, stay := range catalog.Stays {
		imageURL := demoStayImageURL(stay.CategoryID, i)
		// Keep previously seeded stays aligned too. Flutter renders these values
		// directly with Image.network, so demo media must be a stable public URL.
		if _, err := tx.Exec(ctx, `UPDATE business_media AS media SET storage_path=$3
			FROM businesses business
			WHERE media.business_id=business.id
			  AND business.owner_id=$1
			  AND business.type='stay'
			  AND business.name=$2
			  AND media.media_type IN ('logo','cover')`, owner, stay.Name, imageURL); err != nil {
			return 0, err
		}
		var id uuid.UUID
		err := tx.QueryRow(ctx, `INSERT INTO businesses(owner_id,type,name,name_normalized,category_id,currency,short_description,average_rating,review_count)VALUES($1,'stay',$2,lower($2),$3,'BAM',$4,$5,$6) RETURNING id`, owner, stay.Name, stay.CategoryID, stay.Description, stay.Rating, stay.Reviews).Scan(&id)
		if err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO business_locations(business_id,city,city_normalized,address,coordinates)VALUES($1,$2,lower($2),$3,ST_SetSRID(ST_MakePoint($4,$5),4326)::geography)`, id, stay.City, stay.Address, stay.Longitude, stay.Latitude); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stay_details(business_id,inventory_type,base_price_minor)VALUES($1,'single_unit',$2)`, id, int64(stay.Price)*100); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO business_media(business_id,media_type,storage_path)VALUES($1,'cover',$2)`, id, imageURL); err != nil {
			return 0, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stay_amenities(business_id,amenity_code)VALUES($1,'wifi'),($1,'parking')`, id); err != nil {
			return 0, err
		}
		for _, collectionID := range featuredStayCollections(stay.CategoryID) {
			if _, err = tx.Exec(ctx, `INSERT INTO business_featured_collections(business_id,collection_id)VALUES($1,$2)`, id, collectionID); err != nil {
				return 0, err
			}
		}
		if i == 0 {
			selectedBusinessID = id
		}
	}
	if err := selectSeedBusiness(ctx, tx, owner, selectedBusinessID); err != nil {
		return 0, err
	}
	return len(catalog.Stays), nil
}

func (r *Repository) SeedServices(ctx context.Context, owner string) (int, error) {
	created := 0
	err := database.WithinTransaction(ctx, r.pool, func(tx pgx.Tx) error {
		count, err := r.seedServicesInTransaction(ctx, tx, owner)
		if err != nil {
			return err
		}
		created = count
		return nil
	})
	return created, err
}

func (r *Repository) seedServicesInTransaction(ctx context.Context, tx pgx.Tx, owner string) (int, error) {
	catalog, err := firebaseDemoCatalog()
	if err != nil {
		return 0, err
	}
	batch := &pgx.Batch{}
	var selectedBusinessID uuid.UUID
	for i, service := range catalog.Services {
		imageURL := demoServiceImageURL(service.CategoryID)
		// The cards render coverPhotoUrl directly with Image.network. Keep every
		// catalog service (including records made by an earlier seed) on a direct
		// HTTPS URL instead of a Firebase Storage path or transient image URL.
		batch.Queue(`UPDATE business_media AS media SET storage_path=$3
			FROM businesses business
			WHERE media.business_id=business.id
			  AND business.owner_id=$1
			  AND business.type='service'
			  AND business.name=$2
			  AND media.media_type IN ('logo','cover')`, owner, service.Name, imageURL)

		businessID, primaryStaffID, secondaryStaffID := uuid.New(), uuid.New(), uuid.New()
		if i == 0 {
			selectedBusinessID = businessID
		}
		offeringA, offeringB := uuid.New(), uuid.New()
		batch.Queue(`INSERT INTO businesses(id,owner_id,type,name,name_normalized,category_id,currency,short_description,average_rating,review_count)VALUES($1,$2,'service',$3,lower($3),$4,'BAM',$5,$6,$7)`, businessID, owner, service.Name, service.CategoryID, service.Description, service.Rating, service.Reviews)
		batch.Queue(`INSERT INTO business_locations(business_id,city,city_normalized,address,coordinates)VALUES($1,$2,lower($2),$3,ST_SetSRID(ST_MakePoint($4,$5),4326)::geography)`, businessID, service.City, service.Address, 18.3+float64(i)/100, 43.7+float64(i)/100)
		batch.Queue(`INSERT INTO service_details(business_id,time_zone)VALUES($1,'Europe/Sarajevo')`, businessID)
		batch.Queue(`INSERT INTO business_media(business_id,media_type,storage_path)VALUES($1,'logo',$2),($1,'cover',$2)`, businessID, imageURL)
		for _, collectionID := range featuredServiceCollections(service.CategoryID) {
			batch.Queue(`INSERT INTO business_featured_collections(business_id,collection_id)VALUES($1,$2)`, businessID, collectionID)
		}
		batch.Queue(`INSERT INTO service_staff(id,business_id,name,title,commission_rate)VALUES($1,$2,$3,'Demo specialist',$4),($5,$2,$6,'Demo specialist',$7)`, primaryStaffID, businessID, fmt.Sprintf("Demo Provider %d", i+1), 60+float64(i%4)*5, secondaryStaffID, fmt.Sprintf("Demo Provider %dB", i+1), 45+float64(i%4)*5)
		batch.Queue(`INSERT INTO service_offerings(id,business_id,name,duration_minutes,price_minor)VALUES($1,$2,$3,$4,$5),($6,$2,$7,$8,$9)`, offeringA, businessID, service.ServiceName, service.Duration, int64(service.Price)*100, offeringB, service.SecondaryServiceName, service.SecondaryDuration, int64(service.SecondaryPrice)*100)
		batch.Queue(`INSERT INTO service_staff_offerings(staff_id,offering_id)VALUES($1,$2),($1,$3),($4,$2),($4,$3)`, primaryStaffID, offeringA, offeringB, secondaryStaffID)
		for weekday := 0; weekday < 5; weekday++ {
			batch.Queue(`INSERT INTO service_staff_weekly_availability(staff_id,weekday,start_minutes,end_minutes)VALUES($1,$2,540,1020),($3,$2,600,1080)`, primaryStaffID, weekday, secondaryStaffID)
		}
	}
	results := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return 0, err
		}
	}
	if err := results.Close(); err != nil {
		return 0, err
	}
	if err := selectSeedBusiness(ctx, tx, owner, selectedBusinessID); err != nil {
		return 0, err
	}
	return len(catalog.Services), nil
}

func featuredStayCollections(categoryID string) []string {
	switch categoryID {
	case "hotel":
		return []string{"romantic_getaways", "weekend_escapes", "city_breaks"}
	case "resort":
		return []string{"romantic_getaways", "family_friendly", "pool_stays"}
	case "villa", "pool_villa":
		return []string{"romantic_getaways", "family_friendly", "pool_stays"}
	case "beach_villa":
		return []string{"romantic_getaways", "beachfront_stays"}
	case "apartment", "aparthotel", "hostel", "guesthouse":
		return []string{"weekend_escapes", "city_breaks"}
	case "cottage":
		return []string{"family_friendly", "pet_friendly"}
	case "cabin", "mountain_cabin", "glamping":
		return []string{"pet_friendly", "mountain_escapes"}
	case "vacation_home":
		return []string{"family_friendly"}
	default:
		return nil
	}
}

func featuredServiceCollections(categoryID string) []string {
	switch categoryID {
	case "massage_spa", "massage_therapy", "spa_wellness":
		return []string{"wellness_spa"}
	case "physiotherapy", "personal_training":
		return []string{"wellness_spa", "health_care"}
	case "hair_salon", "barbershop", "beauty_salon", "nail_salon", "tattoo_piercing":
		return []string{"beauty_grooming"}
	case "electrician", "plumber", "locksmith", "hvac_service", "painter_decorator", "cleaning_service":
		return []string{"home_repairs"}
	case "automotive_service", "car_wash_detailing":
		return []string{"auto_services"}
	case "dental_clinic", "medical_clinic":
		return []string{"health_care"}
	case "tutoring":
		return []string{"learn_grow"}
	case "veterinary_pet_care":
		return []string{"pet_care"}
	case "photography_videography", "legal_consultation", "accounting_consultation", "professional_service":
		return []string{"professional_services"}
	default:
		return nil
	}
}

// selectSeedBusiness keeps development seeding usable by provider flows that
// require an active selected business immediately after the seed completes.
func selectSeedBusiness(ctx context.Context, tx pgx.Tx, owner string, businessID uuid.UUID) error {
	if businessID == uuid.Nil {
		return nil
	}
	command, err := tx.Exec(ctx, `UPDATE users SET selected_business_id=$2 WHERE id=$1`, owner, businessID)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("select seeded business: owner %q was not found", owner)
	}
	return nil
}

// demoServiceImageURL mirrors the Firebase demo seed. These are already
// public HTTPS URLs, so Flutter must not resolve them through Firebase Storage.
func demoServiceImageURL(categoryID string) string {
	switch categoryID {
	case "hair_salon", "barbershop":
		return "https://images.unsplash.com/photo-1560066984-138dadb4c035?auto=format&fit=crop&w=1200&q=85"
	case "beauty_salon", "nail_salon", "tattoo_piercing":
		return "https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?auto=format&fit=crop&w=1200&q=85"
	case "dental_clinic", "medical_clinic":
		return "https://images.unsplash.com/photo-1629909613654-28e377c37b09?auto=format&fit=crop&w=1200&q=85"
	case "physiotherapy", "personal_training":
		return "https://images.unsplash.com/photo-1571019613454-1cb2f99b2d8b?auto=format&fit=crop&w=1200&q=85"
	case "massage_spa", "massage_therapy", "spa_wellness":
		return "https://images.unsplash.com/photo-1540555700478-4be289fbecef?auto=format&fit=crop&w=1200&q=85"
	case "tutoring":
		return "https://images.unsplash.com/photo-1523240795612-9a054b0db644?auto=format&fit=crop&w=1200&q=85"
	case "electrician", "plumber", "locksmith", "hvac_service", "painter_decorator":
		return "https://images.unsplash.com/photo-1621905252507-b35492cc74b4?auto=format&fit=crop&w=1200&q=85"
	case "cleaning_service":
		return "https://images.unsplash.com/photo-1581578731548-c64695cc6952?auto=format&fit=crop&w=1200&q=85"
	case "automotive_service", "car_wash_detailing":
		return "https://images.unsplash.com/photo-1492144534655-ae79c964c9d7?auto=format&fit=crop&w=1200&q=85"
	case "veterinary_pet_care":
		return "https://images.unsplash.com/photo-1628009368231-7bb7cfcb0def?auto=format&fit=crop&w=1200&q=85"
	case "photography_videography":
		return "https://images.unsplash.com/photo-1452780212940-6f5c0d14d848?auto=format&fit=crop&w=1200&q=85"
	case "legal_consultation", "accounting_consultation", "professional_service":
		return "https://images.unsplash.com/photo-1450101499163-c8848c66ca85?auto=format&fit=crop&w=1200&q=85"
	default:
		return "https://images.unsplash.com/photo-1566073771259-6a8506099945?auto=format&fit=crop&w=1200&q=85"
	}
}

// demoStayImageURL deliberately follows the same direct-HTTPS rule as service
// seed media. It avoids redirect-only placeholder hosts which do not render
// consistently in the Flutter image pipeline.
func demoStayImageURL(categoryID string, index int) string {
	switch categoryID {
	case "hotel":
		return "https://images.unsplash.com/photo-1566073771259-6a8506099945?auto=format&fit=crop&w=1200&q=85"
	case "resort":
		return "https://images.unsplash.com/photo-1582719478250-c89cae4dc85b?auto=format&fit=crop&w=1200&q=85"
	case "apartment", "condo":
		return "https://images.unsplash.com/photo-1522708323590-d24dbb6b0267?auto=format&fit=crop&w=1200&q=85"
	case "villa":
		return "https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&w=1200&q=85"
	case "hostel":
		return "https://images.unsplash.com/photo-1555854877-bab0e564b8d5?auto=format&fit=crop&w=1200&q=85"
	case "cabin", "cottage":
		return "https://images.unsplash.com/photo-1449157291145-7efd050a4d0e?auto=format&fit=crop&w=1200&q=85"
	default:
		images := []string{
			"https://images.unsplash.com/photo-1542314831-068cd1dbfeeb?auto=format&fit=crop&w=1200&q=85",
			"https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&w=1200&q=85",
		}
		return images[index%len(images)]
	}
}
