package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	appointmentshttp "github.com/mkaric2003/multibook-backend/internal/modules/appointments/adapters/http"
	appointmentspostgres "github.com/mkaric2003/multibook-backend/internal/modules/appointments/adapters/postgres"
	appointmentsapp "github.com/mkaric2003/multibook-backend/internal/modules/appointments/application"
	auditpostgres "github.com/mkaric2003/multibook-backend/internal/modules/audit/adapters/postgres"
	bookingshttp "github.com/mkaric2003/multibook-backend/internal/modules/bookings/adapters/http"
	bookingspostgres "github.com/mkaric2003/multibook-backend/internal/modules/bookings/adapters/postgres"
	bookingsapp "github.com/mkaric2003/multibook-backend/internal/modules/bookings/application"
	locationshttp "github.com/mkaric2003/multibook-backend/internal/modules/business_locations/adapters/http"
	locationspostgres "github.com/mkaric2003/multibook-backend/internal/modules/business_locations/adapters/postgres"
	locationsapp "github.com/mkaric2003/multibook-backend/internal/modules/business_locations/application"
	mediahttp "github.com/mkaric2003/multibook-backend/internal/modules/business_media/adapters/http"
	mediapostgres "github.com/mkaric2003/multibook-backend/internal/modules/business_media/adapters/postgres"
	mediaapp "github.com/mkaric2003/multibook-backend/internal/modules/business_media/application"
	businesseshttp "github.com/mkaric2003/multibook-backend/internal/modules/businesses/adapters/http"
	businessespostgres "github.com/mkaric2003/multibook-backend/internal/modules/businesses/adapters/postgres"
	businessesapp "github.com/mkaric2003/multibook-backend/internal/modules/businesses/application"
	chatbackground "github.com/mkaric2003/multibook-backend/internal/modules/chat/adapters/background"
	chathttp "github.com/mkaric2003/multibook-backend/internal/modules/chat/adapters/http"
	chatpostgres "github.com/mkaric2003/multibook-backend/internal/modules/chat/adapters/postgres"
	chatapp "github.com/mkaric2003/multibook-backend/internal/modules/chat/application"
	discoveryhttp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/http"
	discoverypostgres "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/adapters/postgres"
	discoveryapp "github.com/mkaric2003/multibook-backend/internal/modules/customer_discovery/application"
	seedhttp "github.com/mkaric2003/multibook-backend/internal/modules/development_seed/adapters/http"
	seedpostgres "github.com/mkaric2003/multibook-backend/internal/modules/development_seed/adapters/postgres"
	seedapp "github.com/mkaric2003/multibook-backend/internal/modules/development_seed/application"
	draftshttp "github.com/mkaric2003/multibook-backend/internal/modules/drafts/adapters/http"
	draftspostgres "github.com/mkaric2003/multibook-backend/internal/modules/drafts/adapters/postgres"
	draftsapp "github.com/mkaric2003/multibook-backend/internal/modules/drafts/application"
	metricshttp "github.com/mkaric2003/multibook-backend/internal/modules/metrics/adapters/http"
	metricspostgres "github.com/mkaric2003/multibook-backend/internal/modules/metrics/adapters/postgres"
	metricsapp "github.com/mkaric2003/multibook-backend/internal/modules/metrics/application"
	notificationsfirebase "github.com/mkaric2003/multibook-backend/internal/modules/notifications/adapters/firebase"
	notificationshttp "github.com/mkaric2003/multibook-backend/internal/modules/notifications/adapters/http"
	notificationspostgres "github.com/mkaric2003/multibook-backend/internal/modules/notifications/adapters/postgres"
	notificationsapp "github.com/mkaric2003/multibook-backend/internal/modules/notifications/application"
	paymentmethodshttp "github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/adapters/http"
	paymentmethodspostgres "github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/adapters/postgres"
	paymentmethodsapp "github.com/mkaric2003/multibook-backend/internal/modules/payment_methods/application"
	promotionshttp "github.com/mkaric2003/multibook-backend/internal/modules/promotions/adapters/http"
	promotionspostgres "github.com/mkaric2003/multibook-backend/internal/modules/promotions/adapters/postgres"
	promotionsapp "github.com/mkaric2003/multibook-backend/internal/modules/promotions/application"
	recenthttp "github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/adapters/http"
	recentpostgres "github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/adapters/postgres"
	recentapp "github.com/mkaric2003/multibook-backend/internal/modules/recently_viewed/application"
	reviewshttp "github.com/mkaric2003/multibook-backend/internal/modules/reviews/adapters/http"
	reviewspostgres "github.com/mkaric2003/multibook-backend/internal/modules/reviews/adapters/postgres"
	reviewsapp "github.com/mkaric2003/multibook-backend/internal/modules/reviews/application"
	savedhttp "github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/adapters/http"
	savedpostgres "github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/adapters/postgres"
	savedapp "github.com/mkaric2003/multibook-backend/internal/modules/saved_businesses/application"
	availabilityhttp "github.com/mkaric2003/multibook-backend/internal/modules/service_availability/adapters/http"
	availabilitypostgres "github.com/mkaric2003/multibook-backend/internal/modules/service_availability/adapters/postgres"
	availabilityapp "github.com/mkaric2003/multibook-backend/internal/modules/service_availability/application"
	offeringshttp "github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/adapters/http"
	offeringspostgres "github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/adapters/postgres"
	offeringsapp "github.com/mkaric2003/multibook-backend/internal/modules/service_offerings/application"
	searchhttp "github.com/mkaric2003/multibook-backend/internal/modules/service_search/adapters/http"
	searchpostgres "github.com/mkaric2003/multibook-backend/internal/modules/service_search/adapters/postgres"
	searchapp "github.com/mkaric2003/multibook-backend/internal/modules/service_search/application"
	staffhttp "github.com/mkaric2003/multibook-backend/internal/modules/service_staff/adapters/http"
	staffpostgres "github.com/mkaric2003/multibook-backend/internal/modules/service_staff/adapters/postgres"
	staffapp "github.com/mkaric2003/multibook-backend/internal/modules/service_staff/application"
	serviceshttp "github.com/mkaric2003/multibook-backend/internal/modules/services/adapters/http"
	servicespostgres "github.com/mkaric2003/multibook-backend/internal/modules/services/adapters/postgres"
	servicesapp "github.com/mkaric2003/multibook-backend/internal/modules/services/application"
	staysearchhttp "github.com/mkaric2003/multibook-backend/internal/modules/stay_search/adapters/http"
	staysearchpostgres "github.com/mkaric2003/multibook-backend/internal/modules/stay_search/adapters/postgres"
	staysearchapp "github.com/mkaric2003/multibook-backend/internal/modules/stay_search/application"
	unitshttp "github.com/mkaric2003/multibook-backend/internal/modules/stay_units/adapters/http"
	unitspostgres "github.com/mkaric2003/multibook-backend/internal/modules/stay_units/adapters/postgres"
	unitsapp "github.com/mkaric2003/multibook-backend/internal/modules/stay_units/application"
	stayshttp "github.com/mkaric2003/multibook-backend/internal/modules/stays/adapters/http"
	stayspostgres "github.com/mkaric2003/multibook-backend/internal/modules/stays/adapters/postgres"
	staysapp "github.com/mkaric2003/multibook-backend/internal/modules/stays/application"
	supporthttp "github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/adapters/http"
	supportpostgres "github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/adapters/postgres"
	supportapp "github.com/mkaric2003/multibook-backend/internal/modules/support_tickets/application"
	usershttp "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/http"
	userspostgres "github.com/mkaric2003/multibook-backend/internal/modules/users/adapters/postgres"
	usersapp "github.com/mkaric2003/multibook-backend/internal/modules/users/application"
	"github.com/mkaric2003/multibook-backend/internal/platform/auth"
	"github.com/mkaric2003/multibook-backend/internal/platform/config"
	"github.com/mkaric2003/multibook-backend/internal/shared/httpx"
	"golang.org/x/time/rate"
)

func NewRouter(appContext context.Context, pool *pgxpool.Pool, authenticator *auth.FirebaseAuthenticator, cfg config.Config, logger *slog.Logger) http.Handler {
	var push notificationsapp.PushSender
	if authenticator != nil {
		if client, err := authenticator.MessagingClient(context.Background()); err != nil {
			logger.Warn("firebase messaging startup failed", "error", err)
		} else {
			push = notificationsfirebase.NewPushSender(client)
		}
	}
	notificationsService := notificationsapp.NewService(
		notificationspostgres.NewCommandRepository(pool),
		notificationspostgres.NewQueries(pool),
		push,
	)
	notificationsHandler := notificationshttp.NewHandler(notificationsService)
	paymentMethodsHandler := paymentmethodshttp.NewHandler(paymentmethodsapp.NewService(
		paymentmethodspostgres.NewCommandRepository(pool),
		paymentmethodspostgres.NewQueries(pool),
	))
	chatUpdates := chatpostgres.NewChatUpdateListener(cfg.DatabaseURL, logger)
	chatUpdates.Run(appContext)
	chatHandler := chathttp.NewHandler(chatapp.NewService(
		chatpostgres.NewCommandRepository(pool),
		chatpostgres.NewQueries(pool),
		chatUpdates,
	))
	if push != nil {
		chatbackground.NewPushWorker(
			chatapp.NewPushDispatcher(chatpostgres.NewPushOutbox(pool), notificationsService),
			logger,
		).Run(appContext)
	}
	auditWriter := auditpostgres.NewWriter()
	usersService := usersapp.NewService(userspostgres.NewCommandRepository(pool), userspostgres.NewQueries(pool))
	usersHandler := usershttp.NewHandler(usersService)
	businessesService := businessesapp.NewService(businessespostgres.NewCommandRepository(pool, auditWriter), businessespostgres.NewQueries(pool))
	businessesHandler := businesseshttp.NewHandler(businessesService)
	discoveryQueries := discoverypostgres.NewQueries(pool)
	discoveryHandler := discoveryhttp.NewHandler(discoveryapp.NewService(discoveryQueries))
	savedHandler := savedhttp.NewHandler(savedapp.NewService(savedpostgres.NewCommandRepository(pool), savedpostgres.NewQueries(pool), discoveryQueries))
	recentHandler := recenthttp.NewHandler(recentapp.NewService(recentpostgres.NewCommandRepository(pool), recentpostgres.NewQueries(pool), discoveryQueries))
	reviewsHandler := reviewshttp.NewHandler(reviewsapp.NewService(reviewspostgres.NewCommandRepository(pool), reviewspostgres.NewQueries(pool)))
	metricsUpdates := metricspostgres.NewMetricsUpdateListener(cfg.DatabaseURL, logger)
	metricsUpdates.Run(appContext)
	metricsHandler := metricshttp.NewHandler(metricsapp.NewService(metricspostgres.NewQueries(pool), metricsUpdates))
	locationsHandler := locationshttp.NewHandler(locationsapp.NewService(locationspostgres.NewCommandRepository(pool, auditWriter)))
	mediaHandler := mediahttp.NewHandler(mediaapp.NewService(mediapostgres.NewCommandRepository(pool, auditWriter)))
	staysHandler := stayshttp.NewHandler(staysapp.NewService(stayspostgres.NewCommandRepository(pool, auditWriter), stayspostgres.NewQueries(pool)))
	servicesHandler := serviceshttp.NewHandler(servicesapp.NewService(servicespostgres.NewCommandRepository(pool, auditWriter), servicespostgres.NewQueries(pool)))
	offeringsHandler := offeringshttp.NewHandler(offeringsapp.NewService(offeringspostgres.NewCommandRepository(pool, auditWriter), offeringspostgres.NewQueries(pool)))
	staffHandler := staffhttp.NewHandler(staffapp.NewService(staffpostgres.NewCommandRepository(pool, auditWriter), staffpostgres.NewQueries(pool)))
	availabilityHandler := availabilityhttp.NewHandler(availabilityapp.NewService(availabilitypostgres.NewCommandRepository(pool, auditWriter), availabilitypostgres.NewQueries(pool)))
	discountApplier := promotionspostgres.NewDiscountApplier()
	promotionsHandler := promotionshttp.NewHandler(promotionsapp.NewService(
		promotionspostgres.NewCommandRepository(pool),
		promotionspostgres.NewQueries(pool),
	))
	appointmentsHandler := appointmentshttp.NewHandler(appointmentsapp.NewService(appointmentspostgres.NewCommandRepository(pool, auditWriter, discountApplier), appointmentspostgres.NewQueries(pool), notificationsService))
	bookingsHandler := bookingshttp.NewHandler(bookingsapp.NewService(bookingspostgres.NewRepository(pool, discountApplier), notificationsService))
	searchHandler := searchhttp.NewHandler(searchapp.NewService(searchpostgres.NewQueries(pool)))
	staySearchHandler := staysearchhttp.NewHandler(staysearchapp.NewService(staysearchpostgres.NewQueries(pool)))
	seedHandler := seedhttp.NewHandler(seedapp.NewService(seedpostgres.NewRepository(pool), cfg.AppEnv))
	draftsHandler := draftshttp.NewHandler(draftsapp.NewService(draftspostgres.NewCommandRepository(pool), draftspostgres.NewQueries(pool)))
	unitsHandler := unitshttp.NewHandler(unitsapp.NewService(unitspostgres.NewCommandRepository(pool, auditWriter), unitspostgres.NewQueries(pool)))
	supportHandler := supporthttp.NewHandler(supportapp.NewService(
		supportpostgres.NewCommandRepository(pool),
		supportpostgres.NewQueries(pool),
	))
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(recoverer(logger))
	router.Use(requestLogger(logger))
	router.Use(cors(cfg.CORSAllowedOrigins))
	router.Use(rateLimit(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst))
	router.Use(requestTimeout(15 * time.Second))

	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/readyz", readiness(pool))

	router.Route("/v1", func(r chi.Router) {
		r.Use(authenticator.Require)
		r.Use(usershttp.ProvisionUser(usersService))
		r.Get("/users/me", usersHandler.CurrentUser)
		r.Patch("/users/me", usersHandler.UpdateProfile)
		r.Put("/users/me/role", usersHandler.UpdateRole)
		r.Put("/users/me/selected-business", businessesHandler.Select)
		r.Get("/bookings", bookingsHandler.List)
		r.Patch("/bookings/{bookingID}/cancel", bookingsHandler.Cancel)
		r.Patch("/bookings/{bookingID}/status", bookingsHandler.Status)
		r.Get("/appointments", appointmentsHandler.List)
		r.Get("/notifications", notificationsHandler.List)
		r.Get("/notifications/unread-count", notificationsHandler.UnreadCount)
		r.Patch("/notifications/{notificationID}/read", notificationsHandler.MarkRead)
		r.Put("/notification-devices/{deviceID}", notificationsHandler.PutDevice)
		r.Delete("/notification-devices/{deviceID}", notificationsHandler.DeleteDevice)
		r.Get("/payment-methods", paymentMethodsHandler.List)
		r.Post("/payment-methods", paymentMethodsHandler.Create)
		r.Patch("/payment-methods/{paymentMethodID}/default", paymentMethodsHandler.SetDefault)
		r.Delete("/payment-methods/{paymentMethodID}", paymentMethodsHandler.Delete)
		r.Post("/conversations", chatHandler.GetOrCreate)
		r.Get("/conversations", chatHandler.List)
		r.Get("/conversations/unread-count", chatHandler.UnreadCount)
		r.Get("/conversations/{conversationID}", chatHandler.Get)
		r.Get("/conversations/{conversationID}/messages", chatHandler.ListMessages)
		r.Post("/conversations/{conversationID}/messages", chatHandler.SendMessage)
		r.Patch("/conversations/{conversationID}/read", chatHandler.MarkRead)
		r.Put("/conversations/{conversationID}/typing", chatHandler.SetTyping)
		r.Put("/conversations/{conversationID}/presence", chatHandler.SetPresence)
		r.Get("/chat/stream", chatHandler.Stream)
		r.Get("/support-tickets", supportHandler.List)
		r.Post("/support-tickets", supportHandler.Create)
		r.Patch("/appointments/{appointmentID}/status", appointmentsHandler.Status)
		r.Patch("/appointments/{appointmentID}/reschedule", appointmentsHandler.Reschedule)
		r.Get("/services/search", searchHandler.Search)
		r.Get("/stays/search", staySearchHandler.Search)
		r.Get("/discovery/businesses", discoveryHandler.ListPopular)
		r.Get("/discovery/cities", discoveryHandler.ListCities)
		r.Get("/discovery/featured-collections", discoveryHandler.ListFeaturedCollections)
		r.Get("/discovery/businesses/{businessID}", discoveryHandler.GetDetail)
		r.Get("/discovery/search", discoveryHandler.Search)
		r.Get("/discovery/recommended-stays", discoveryHandler.ListRecommendedStays)
		r.Get("/saved-businesses", savedHandler.List)
		r.Get("/saved-businesses/{businessID}", savedHandler.IsSaved)
		r.Put("/saved-businesses/{businessID}", savedHandler.Save)
		r.Delete("/saved-businesses/{businessID}", savedHandler.Remove)
		r.Get("/recently-viewed", recentHandler.List)
		r.Put("/recently-viewed/{businessID}", recentHandler.Record)
		r.Get("/drafts/booking", draftsHandler.GetBooking)
		r.Put("/drafts/booking", draftsHandler.PutBooking)
		r.Delete("/drafts/booking", draftsHandler.DeleteBooking)
		r.Get("/drafts/appointment", draftsHandler.GetAppointment)
		r.Put("/drafts/appointment", draftsHandler.PutAppointment)
		r.Delete("/drafts/appointment", draftsHandler.DeleteAppointment)
		r.Post("/development/seed/stays", seedHandler.Stays)
		r.Post("/development/seed/services", seedHandler.Services)
		r.Route("/businesses", func(r chi.Router) {
			r.Post("/", businessesHandler.Create)
			r.Get("/", businessesHandler.ListOwned)
			r.Get("/{businessID}/reviews", reviewsHandler.List)
			r.Get("/{businessID}/review-status", reviewsHandler.HasReview)
			r.Post("/{businessID}/reviews", reviewsHandler.Create)
			r.Get("/{businessID}/dashboard-metrics", metricsHandler.GetDashboardMetrics)
			r.Get("/{businessID}/dashboard-metrics/stream", metricsHandler.StreamDashboardMetrics)
			r.Get("/{businessID}/earnings", metricsHandler.GetEarningsMetrics)
			r.Get("/{businessID}/earnings/stream", metricsHandler.StreamEarningsMetrics)
			r.Get("/{businessID}/promotions", promotionsHandler.List)
			r.Get("/{businessID}/promotions/active", promotionsHandler.Active)
			r.Post("/{businessID}/promotions", promotionsHandler.Create)
			r.Patch("/{businessID}/promotions/{promotionID}/active", promotionsHandler.SetActive)
			r.Delete("/{businessID}/promotions/{promotionID}", promotionsHandler.Delete)
			r.Get("/{businessID}", businessesHandler.GetOwned)
			r.Put("/{businessID}", businessesHandler.Replace)
			r.Patch("/{businessID}", businessesHandler.Update)
			r.Delete("/{businessID}", businessesHandler.Archive)
			r.Put("/{businessID}/location", locationsHandler.Upsert)
			r.Put("/{businessID}/media", mediaHandler.Replace)
			r.Get("/{businessID}/stay", staysHandler.GetOwned)
			r.Put("/{businessID}/stay", staysHandler.Upsert)
			r.Get("/{businessID}/stay/unit-types", unitsHandler.List)
			r.Post("/{businessID}/stay/unit-types", unitsHandler.Create)
			r.Patch("/{businessID}/stay/unit-types/{unitTypeID}", unitsHandler.Update)
			r.Delete("/{businessID}/stay/unit-types/{unitTypeID}", unitsHandler.Archive)
			r.Get("/{businessID}/service", servicesHandler.Get)
			r.Put("/{businessID}/service", servicesHandler.Upsert)
			r.Get("/{businessID}/service/offerings", offeringsHandler.List)
			r.Post("/{businessID}/service/offerings", offeringsHandler.Create)
			r.Patch("/{businessID}/service/offerings/{offeringID}", offeringsHandler.Update)
			r.Delete("/{businessID}/service/offerings/{offeringID}", offeringsHandler.Archive)
			r.Get("/{businessID}/service/staff", staffHandler.List)
			r.Post("/{businessID}/service/staff", staffHandler.Create)
			r.Patch("/{businessID}/service/staff/{staffID}", staffHandler.Update)
			r.Delete("/{businessID}/service/staff/{staffID}", staffHandler.Archive)
			r.Get("/{businessID}/service/staff/{staffID}/weekly-availability", availabilityHandler.ListWeekly)
			r.Post("/{businessID}/service/staff/{staffID}/weekly-availability", availabilityHandler.CreateWeekly)
			r.Patch("/{businessID}/service/staff/{staffID}/weekly-availability/{weeklyAvailabilityID}", availabilityHandler.UpdateWeekly)
			r.Delete("/{businessID}/service/staff/{staffID}/weekly-availability/{weeklyAvailabilityID}", availabilityHandler.DeleteWeekly)
			r.Get("/{businessID}/service/staff/{staffID}/availability-blocks", availabilityHandler.ListBlocks)
			r.Post("/{businessID}/service/staff/{staffID}/availability-blocks", availabilityHandler.CreateBlock)
			r.Patch("/{businessID}/service/staff/{staffID}/availability-blocks/{availabilityBlockID}", availabilityHandler.UpdateBlock)
			r.Delete("/{businessID}/service/staff/{staffID}/availability-blocks/{availabilityBlockID}", availabilityHandler.DeleteBlock)
			r.Get("/{businessID}/service/staff/{staffID}/available-slots", appointmentsHandler.AvailableSlots)
			r.Post("/{businessID}/service/appointments", appointmentsHandler.Create)
			r.Post("/{businessID}/stay/bookings", bookingsHandler.Create)
			r.Get("/{businessID}/stay/availability", bookingsHandler.Availability)
		})
	})

	return router
}

func readiness(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.WriteRequestError(w, r, http.StatusServiceUnavailable, "database_unavailable", "database is unavailable")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
