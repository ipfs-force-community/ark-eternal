package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/asamuj/ark-eternal/database"
	"github.com/asamuj/ark-eternal/service"
)

func main() {
	app := &cli.Command{
		Name:  "pdp-hackthon",
		Usage: "pdp hackthon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "db_path",
				Value: "./pdp.db",
				Usage: "Path to the database file",
			},
			&cli.StringFlag{
				Name:  "private_key_path",
				Value: "./pdp.pri",
				Usage: "Path to the private key file",
			},
			&cli.IntFlag{
				Name:  "proof_set_id",
				Value: 40,
				Usage: "ID of the proof set",
			},
			&cli.StringFlag{
				Name:  "service_url",
				Value: "https://yablu.net",
				Usage: "URL of the service",
			},
			&cli.Int32Flag{
				Name:  "port",
				Value: 12345,
				Usage: "Port to run the service on",
			},
		},
		Action: action,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func action(ctx context.Context, cmd *cli.Command) error {
	db, err := database.InitDB(cmd.String("db_path"))
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	privateKey, err := service.LoadPrivateKey(cmd.String("private_key_path"))
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	service.NewService(db, privateKey, cmd.Int("proof_set_id"), cmd.String("service_url"), "pdp-hackthon").
		Run(cmd.Int32("port"))

	return nil
}
