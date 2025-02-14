package url

import (
	"gohw/shared/utils/dotenv"
)

const (
    SERVER_HOST_KEY = "APP_SERVER_HOST"
    SERVER_PORT_KEY = "APP_SERVER_PORT"
)

func GetServerUrl() string {
    host := dotenv.GetEnvVar(SERVER_HOST_KEY)
    port := dotenv.GetEnvVar(SERVER_PORT_KEY)
    return "http://" + host + ":" + port
}
