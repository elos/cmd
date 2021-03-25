package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"google.golang.org/grpc"

	apb "github.com/elos/x/auth/proto"
)

var authDir string

func init() {
	u, err := user.Current()
	if err != nil {
		log.Fatalf("user.Current error: %v", err)
	}
	authDir = filepath.Join(u.HomeDir, "elos", "auth")
	os.MkdirAll(authDir, 0700)
}

const help = `The e command interacts with elos.

Subcommands:
  - auth`

func main() {
	ctx := context.Background()

	if len(os.Args) == 1 {
		fmt.Println(help)
		return
	}

	switch os.Args[1] {
	case "auth":
		runAuth(ctx, os.Args[2:])
	}
}

func runAuth(ctx context.Context, args []string) {
	conn, err := grpc.Dial(":3333", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	authc := apb.NewAuthClient(conn)

	var public, private string
	fmt.Print("Public:  ")
	fmt.Scanln(&public)
	fmt.Print("Private: ")
	fmt.Scanln(&private)

	ar, err := authc.Authenticate(ctx, &apb.AuthenticateRequest{
		Public:  public,
		Private: private,
	})

	if err != nil {
		log.Fatal(err)
	}

	switch ar.Type {
	case apb.AuthenticateResponse_CREDENTIALED:
		if err := saveAuthConfig(&AuthConfig{
			Public:  public,
			Private: private,
			Created: time.Now(),
		}); err != nil {
			log.Fatal("unable to save auth config: %s", err)
		}
	default:
		fmt.Printf("bad auth type: %s\n", ar.Type)
	}
}

func saveAuthConfig(ac *AuthConfig) error {
	bs, err := json.Marshal(ac)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(authDir, "config"), bs, 0600)
}

type AuthConfig struct {
	Public, Private string

	Created time.Time
}
