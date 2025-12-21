package g

import (
	"main/helpers/meilisearch"
)

type MeiliConfig struct {
	BaseUrl    string `yaml:"base-url"`
	IndexName  string `yaml:"index-name"`
	PrimaryKey string `yaml:"primary-key"`
	MasterKey  string `yaml:"master-key,omitempty"`
}

var MeiliClient *meilisearch.Client

func initMeili() {
	MeiliClient = meilisearch.NewMeiliClient(
		config.MeiliConfig.BaseUrl,
		config.MeiliConfig.IndexName,
		config.MeiliConfig.MasterKey,
	)
	MeiliClient.PrimaryKey = config.MeiliConfig.PrimaryKey
}
