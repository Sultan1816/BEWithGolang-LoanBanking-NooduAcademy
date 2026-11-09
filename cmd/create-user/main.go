package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"

	"noodu-login-service/internal/config"
	"noodu-login-service/internal/users"
)

func main() {
	username := flag.String("username", "", "username akun baru")
	email := flag.String("email", "", "email akun baru")
	flag.Parse()

	if strings.TrimSpace(*username) == "" || strings.TrimSpace(*email) == "" {
		fmt.Fprintln(os.Stderr, "username dan email wajib diisi")
		os.Exit(2)
	}
	if len(*username) > 50 || len(*email) > 100 {
		fmt.Fprintln(os.Stderr, "username atau email terlalu panjang")
		os.Exit(2)
	}

	password, err := readPassword("Password baru (minimal 12 karakter): ")
	if err != nil {
		fail(err)
	}
	confirmation, err := readPassword("Ulangi password: ")
	if err != nil {
		fail(err)
	}
	if string(password) != string(confirmation) {
		fmt.Fprintln(os.Stderr, "password tidak sama")
		os.Exit(2)
	}
	if len(password) < 12 || len(password) > 72 {
		fmt.Fprintln(os.Stderr, "password harus memiliki panjang 12 sampai 72 byte")
		os.Exit(2)
	}

	passwordHash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	clear(password)
	clear(confirmation)
	if err != nil {
		fail(err)
	}

	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		fail(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		fail(fmt.Errorf("gagal terhubung ke PostgreSQL: %w", err))
	}
	user, err := users.NewStore(db).Create(ctx, *username, *email, string(passwordHash))
	if err != nil {
		fail(fmt.Errorf("gagal membuat akun; pastikan email dan username belum digunakan: %w", err))
	}
	fmt.Printf("akun berhasil dibuat: %s (%s), id %d\n", user.Username, user.Email, user.ID)
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}
	return password, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
