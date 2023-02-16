package env

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	JwtKey             []byte
	JweKey             []byte
	GithubClientID     string
	GithubClientSecret string
)

func init() {
	// .env (default)
	godotenv.Load()

	// .env.local # local user specific (git ignored)
	godotenv.Overload(".env.local")

	// set vars
	JwtKey = []byte(os.Getenv("JWT_KEY"))
	JweKey = []byte(os.Getenv("JWE_KEY"))
	GithubClientID = os.Getenv("GITHUB_CLIENT_ID")
	GithubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
}

func Port(def int) int {
	return getint("PORT", def)
}

// private functions

func get(key string, def string) string {
	s := os.Getenv(key)
	if len(s) == 0 {
		return def // return default
	}
	return s
}

func getint(key string, def int) int {
	i, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return i
}
