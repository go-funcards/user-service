package cmd

import (
	"fmt"
	"github.com/go-funcards/grpc-server"
	"github.com/go-funcards/user-service/internal/config"
	"github.com/go-funcards/user-service/internal/user"
	"github.com/go-funcards/user-service/internal/user/db"
	"github.com/go-funcards/user-service/proto/v1"
	"github.com/go-funcards/validate"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"log"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve User Service gRPC",
	Long:  "Serve User Service gRPC",
	Run:   executeServeCommand,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func executeServeCommand(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	cfg, err := config.GetConfig(globalFlags.ConfigFile)
	if err != nil {
		log.Fatal(err)
	}

	logger, err := cfg.Log.BuildLogger(cfg.Debug)
	if err != nil {
		panic(err)
	}

	logger.Info(fmt.Sprintf("starting: %s", use))
	logger.Info(fmt.Sprintf("version: %s", version))

	validate.Default.RegisterStructRules(cfg.Rules, []any{
		v1.CreateUserRequest{},
		v1.UpdateUserRequest{},
		v1.DeleteUserRequest{},
		v1.UsersRequest{},
		v1.UserByEmailAndPasswordRequest{},
	}...)

	mongoDB, err := cfg.MongoDB.GetDatabase(ctx)
	if err != nil {
		panic(err)
	}

	storage, err := db.NewStorage(ctx, mongoDB, logger)
	if err != nil {
		panic(err)
	}

	register := func(srv *grpc.Server) {
		v1.RegisterUserServer(srv, user.NewUserService(storage, logger))
	}

	grpcserver.Start(
		ctx,
		cfg.Server.Listen.Listener(logger),
		register,
		logger,
		grpc.ChainUnaryInterceptor(grpcserver.ValidatorUnaryServerInterceptor(validate.Default)),
	)
}
