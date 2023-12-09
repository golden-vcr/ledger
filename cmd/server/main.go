package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/codingconcepts/env"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	"golang.org/x/sync/errgroup"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger/gen/queries"
	"github.com/golden-vcr/ledger/internal/admin"
	"github.com/golden-vcr/ledger/internal/cheer"
	"github.com/golden-vcr/ledger/internal/notifications"
	"github.com/golden-vcr/ledger/internal/outflow"
	"github.com/golden-vcr/ledger/internal/records"
	"github.com/golden-vcr/server-common/db"
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

	LedgerShowtimeSecretKey string `env:"LEDGER_SHOWTIME_SECRET_KEY"`
}

func main() {
	// Parse config from environment variables
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("error loading .env file: %v", err)
	}
	config := Config{}
	if err := env.Set(&config); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// Shut down cleanly on signal
	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer close()

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
		log.Fatalf("error opening database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("error connecting to database: %v", err)
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
			log.Fatalf("pq.Listener failed: %v", err)
		}
	})
	if err := pqListener.Listen("ledger_flow_change"); err != nil {
		log.Fatalf("failed to issue LISTEN command for pq listener: %v", err)
	}
	pqEvents := make(chan *notifications.FlowChangeNotification)
	go func() {
		for {
			select {
			case <-ctx.Done():
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
	authClient, err := auth.NewClient(ctx, config.AuthURL)
	if err != nil {
		log.Fatalf("error initializing auth client: %v", err)
	}

	// Start setting up our HTTP handlers, using gorilla/mux for routing
	r := mux.NewRouter()

	// The webapp makes requests to GET /balance or GET /history, authenticated with the
	// logged-in user's auth token, in order to get records for that user
	{
		recordsServer := records.NewServer(q)
		recordsServer.RegisterRoutes(authClient, r)

		notificationsServer := notifications.NewServer(ctx, q, pqEvents)
		go notificationsServer.ReadPostgresNotifications(ctx)
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
	// Twitch. This route is authorized via a secret key that's known only to ledger and
	// showtime, restricting access to internal use only and allowing us to identify the
	// target user by ID, without supplying their user access token.
	{
		cheerServer := cheer.NewServer(q, config.LedgerShowtimeSecretKey)
		cheerServer.RegisterRoutes(r, authClient)
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
	addr := fmt.Sprintf("%s:%d", config.BindAddr, config.ListenPort)
	server := &http.Server{Addr: addr, Handler: r}
	fmt.Printf("Listening on %s...\n", addr)
	var wg errgroup.Group
	wg.Go(server.ListenAndServe)

	select {
	case <-ctx.Done():
		fmt.Printf("Received signal; closing server...\n")
		server.Shutdown(context.Background())
	}

	err = wg.Wait()
	if err == http.ErrServerClosed {
		fmt.Printf("Server closed.\n")
	} else {
		log.Fatalf("error running server: %v", err)
	}
}
