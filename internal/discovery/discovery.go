package discovery

import (
	"os"

	"github.com/BryanAguilar-Dev/CaratCore/internal/model"
)

func Scan() model.Computer {

	computer := model.Computer{}

	hostname, err := os.Hostname()
	if err == nil {
		computer.Hostname = hostname
	}

	return computer
}
