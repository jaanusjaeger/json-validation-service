package conf

import (
	"encoding/json"
	"io/ioutil"

	"github.com/jaanusjaeger/json-validation-service/internal/server"
)

type Conf struct {
	Server server.Conf
}

// LoadJSON loads the global configuration from a JSON file
func LoadJSON(path string) (Conf, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return Conf{}, err
	}
	var conf Conf
	err = json.Unmarshal(data, &conf)
	return conf, err
}
