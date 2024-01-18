package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/codingconcepts/env"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/lib/pq"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/ledger/internal/admin"
	"github.com/golden-vcr/ledger/internal/cheer"
	"github.com/golden-vcr/ledger/internal/notifications"
	"github.com/golden-vcr/ledger/internal/outflow"
	"github.com/golden-vcr/ledger/internal/records"
	"github.com/golden-vcr/ledger/internal/subscription"
	"github.com/golden-vcr/server-common/db"
	"github.com/golden-vcr/server-common/entry"
)

type Config struct {
	BindAddr   string `env:"BIND_ADDR"`
	ListenPort uint16 `env:"LISTEN_PORT" default:"5003"`

	AuthURL string `env:"AUTH_URL" default:"http://localhost:5002"`

	TwitchClientId     string `env:"TWITCH_CLIENT_ID" required:"true"`
	TwitchClientSecret string `env:"TWITCH_CLIENT_SECRET" required:"true"`

	DatabaseHost     string `env:"PGHOST" required:"true"`
	DatabasePort     int    `env:"PGPORT" required:"true"`
	DatabaseName     string `env:"PGDATABASE" required:"true"`
	DatabaseUser     string `env:"PGUSER" required:"true"`
	DatabasePassword string `env:"PGPASSWORD" required:"true"`
	DatabaseSslMode  string `env:"PGSSLMODE"`
}

func main() {
	app := entry.NewApplication("ledger")
	defer app.Stop()

	// Parse config from environment variables
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		app.Fail("Failed to load .env file", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		app.Fail("Failed to load config", err)
	}

	// Configure our database connection and initialize a Queries struct, so we can read
	// and write to the 'showtime' schema in response to HTTP requests, EventSub
	// notifications, etc.
	connectionString := db.FormatConnectionString(
		config.DatabaseHost,
		config.DatabasePort,
		config.DatabaseName,
		config.DatabaseUser,
		config.DatabasePassword,
		config.DatabaseSslMode,
	)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		app.Fail("Failed to open sql.DB", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		app.Fail("Failed to connect to database", err)
	}
	q := queries.New(db)

	// Initialize a database listener that will notify us whenever transactions are
	// created or updated
	pqListener := pq.NewListener(connectionString, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		switch ev {
		case pq.ListenerEventConnected:
			fmt.Printf("pq listener connected (err: %v)\n", err)
		case pq.ListenerEventDisconnected:
			fmt.Printf("pq listener disconnected (err: %v)\n", err)
		case pq.ListenerEventReconnected:
			fmt.Printf("pq listener reconnected (err: %v)\n", err)
		case pq.ListenerEventConnectionAttemptFailed:
			fmt.Printf("pq listener connection attempt failed (err: %v)\n", err)
		}
		if err != nil {
			app.Fail("pq.Listener failed", err)
		}
	})
	if err := pqListener.Listen("ledger_flow_change"); err != nil {
		app.Fail("Failed to issue LISTEN command for pq listener", err)
	}
	pqEvents := make(chan *notifications.FlowChangeNotification)
	go func() {
		for {
			select {
			case <-app.Context().Done():
				return
			case notification := <-pqListener.NotificationChannel():
				var event notifications.FlowChangeNotification
				if err := json.Unmarshal([]byte(notification.Extra), &event); err != nil {
					fmt.Printf("Failed to unmarshal extra data from notification: %v\n", err)
					continue
				}
				pqEvents <- &event
			}
		}
	}()

	// Prepare an auth client that we can use to validate (and identify users from)
	// Twitch user access tokens
	authClient, err := auth.NewClient(app.Context(), config.AuthURL)
	if err != nil {
		app.Fail("Failed to initialize auth client", err)
	}

	// Start setting up our HTTP handlers, using gorilla/mux for routing
	r := mux.NewRouter()

	// The webapp makes requests to GET /balance or GET /history, authenticated with the
	// logged-in user's auth token, in order to get records for that user
	{
		recordsServer := records.NewServer(q)
		recordsServer.RegisterRoutes(authClient, r)

		notificationsServer := notifications.NewServer(app.Context(), q, pqEvents)
		go notificationsServer.ReadPostgresNotifications(app.Context())
		notificationsServer.RegisterRoutes(authClient, r)
	}

	// Admin-only sections of the webapp can make requests to POST /inflow/manual-credit
	// in order to award discretionary points to any user
	{
		adminServer := admin.NewServer(q, config.TwitchClientId, config.TwitchClientSecret)
		adminServer.RegisterRoutes(authClient, r)
	}

	// The showtime service can use POST /inflow/cheer to award bits in response to the
	// Twitch channel.cheer webhook, which is called to signify the receipt of bits via
	// Twitch. This route is authorized only when the request carries an authoritative
	// JWT that was issued by the auth server to another internal service.
	{
		cheerServer := cheer.NewServer(q)
		cheerServer.RegisterRoutes(r, authClient)
	}

	// POST /inflow/subscription and POST /inflow/gift-sub work similarly, responding to
	// Twitch events by granting points as thanks for subscriptions
	{
		subscriptionServer := subscription.NewServer(q)
		subscriptionServer.RegisterRoutes(r, authClient)
	}

	// Internal APIs can use POST /outflow to create pending transactions that deduct
	// points in order to take advantage of app features, and PATCH|DELETE /outflow/:id
	// to finalize those transactions
	{
		outflowServer := outflow.NewServer(q)
		outflowServer.RegisterRoutes(authClient, r)
	}

	// Handle incoming HTTP connections until our top-level context is canceled, at
	// which point shut down cleanly
	entry.RunServer(app, r, config.BindAddr, int(config.ListenPort))
}
