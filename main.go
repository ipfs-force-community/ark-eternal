package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/ipfs-force-community/ark-eternal/database"
	"github.com/ipfs-force-community/ark-eternal/service"
)

func main() {
	app := &cli.Command{
		Name:  "ark external",
		Usage: "ark external service",
		Commands: []*cli.Command{
			{
				Name:   "create-proof-set-id",
				Usage:  "Create a proof set ID",
				Action: createProofSetID,
			},
			{
				Name:  "add-roots",
				Usage: "Add roots to a proof set on the PDP service",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "root",
						Usage:    "Root CID and its subroots. Format: rootCID:subrootCID1+subrootCID2,...",
						Required: true,
					},
				},
				Action: addRoots,
			},
			{
				Name:  "export-public-key",
				Usage: "Export the public key of the PDP service",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					publicKey, err := service.ExportPublicKey(cmd.String("private_key_path"))
					if err != nil {
						return fmt.Errorf("failed to export public key: %w", err)
					}

					fmt.Println("Public Key:", publicKey)
					return nil
				},
			},
		},
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
				Value: 390,
				Usage: "ID of the proof set",
			},
			&cli.StringFlag{
				Name:  "service_name",
				Value: "pdp-service",
			},
			&cli.StringFlag{
				Name:  "service_url",
				Value: "https://caliberation-pdp.infrafolio.com",
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ser := service.NewService(ctx, db, privateKey, cmd.Int("proof_set_id"), cmd.String("service_url"), cmd.String("service_name"))

	wg := &sync.WaitGroup{}
	exit := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		ser.Schedule()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ser.Start(cmd.Int32("port")); err != nil {
			slog.Error("failed to start service", "error", err)
			cancel()
			close(exit)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		cancel()
		close(exit)
	}()

	<-exit

	if err := ser.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close service: %v\n", err)
	}

	wg.Wait()

	return nil
}

func createProofSetID(ctx context.Context, cmd *cli.Command) error {
	jwtToken, err := service.GetJWTToken(cmd.String("service_name"), cmd.String("private_key_path"))
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %w", err)
	}

	txHash, err := service.CreateProofSet("0x6170dE2b09b404776197485F3dc6c968Ef948505", "", cmd.String("service_url"), jwtToken)
	if err != nil {
		return fmt.Errorf("failed to create proof set: %w", err)
	}

	fmt.Printf("Proof set created successfully with transaction hash: %s\n", txHash)
	return nil
}

func addRoots(ctx context.Context, cmd *cli.Command) error {
	jwtToken, err := service.GetJWTToken(cmd.String("service_name"), cmd.String("private_key_path"))
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %w", err)
	}

	if err := service.AddRoots("", cmd.String("service_url"), jwtToken, cmd.Int("proof_set_id"), cmd.StringSlice("root")); err != nil {
		return fmt.Errorf("failed to add roots to proof set: %w", err)
	}

	fmt.Println("Roots added successfully to the proof set.")
	return nil
}
