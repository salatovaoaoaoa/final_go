package dotenv

import (
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	return godotenv.Load()
}

func GetEnvVar(name string) string {
	return os.Getenv(name)
}
