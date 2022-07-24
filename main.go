package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-funcards/grpc-server"
	"github.com/go-funcards/grpc-server/grpc_middleware/recovery"
	"github.com/go-funcards/mongodb"
	"github.com/go-funcards/user-service/internal/config"
	"github.com/go-funcards/user-service/internal/user"
	"github.com/go-funcards/user-service/internal/user/db"
	"github.com/go-funcards/user-service/proto/v1"
	"github.com/go-funcards/validate"
	"github.com/jwreagor/grpc-zerolog"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io"
	"net"
	"os"
	"time"
)

//go:generate sh genproto.sh

const (
	envConfigFile = "CONFIG_FILE"
	envLogLevel   = "LOG_LEVEL"
	envLogPretty  = "LOG_PRETTY"
)

var (
	version     string
	buildDate   string
	buildTime   string
	configFile  string
	logLevelStr string
	logLevel    zerolog.Level
	logOutput   io.Writer
)

func init() {
	flag.StringVar(&configFile, "c", "config.yaml", "application config path")
	flag.StringVar(&logLevelStr, "log-level", "info", "application log level")
	flag.Parse()

	if os.Getenv(envConfigFile) != "" {
		configFile = os.Getenv(envConfigFile)
	}

	if os.Getenv(envLogLevel) != "" {
		logLevelStr = os.Getenv(envLogLevel)
	}
	logLevel, _ = zerolog.ParseLevel(logLevelStr)
	if zerolog.NoLevel == logLevel {
		logLevel = zerolog.InfoLevel
	}

	if os.Getenv(envLogPretty) == "" {
		logOutput = os.Stdout
	} else {
		logOutput = zerolog.ConsoleWriter{Out: os.Stdout}
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "severity"
}

func main() {
	ctx := context.Background()

	log := zerolog.
		New(logOutput).
		Level(logLevel).
		With().
		Caller().
		Timestamp().
		Str("system", "grpc").
		Str("span.kind", "server").
		Str("server.name", os.Args[0]).
		Str("server.version", fmt.Sprintf("%s.%s.%s", version, buildDate, buildTime)).
		Logger()

	grpclog.SetLoggerV2(grpczerolog.New(log))

	cfg := config.GetConfig(configFile, log)

	validate.Default.RegisterStructRules(cfg.Validation.Rules, []any{
		v1.CreateUserRequest{},
		v1.UpdateUserRequest{},
		v1.DeleteUserRequest{},
		v1.UsersRequest{},
		v1.UserByEmailAndPasswordRequest{},
	}...)

	mongoDB := mongodb.GetDB(ctx, cfg.MongoDB.URI, log)
	storage := db.NewStorage(ctx, mongoDB, log)

	register := func(server *grpc.Server) {
		v1.RegisterUserServer(server, user.NewUserServer(storage, log))
	}

	lis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create tcp listener")
	}

	log.Info().Msgf("bind application to addr: %s", lis.Addr().(*net.TCPAddr).String())

	grpcserver.Start(ctx, lis, register, log, grpc.ChainUnaryInterceptor(
		mongodb.ErrorUnaryServerInterceptor(),
		validate.DefaultValidatorUnaryServerInterceptor(),
		grpc_recovery.UnaryServerInterceptor(),
	))
}
