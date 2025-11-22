package g

import "fmt"

type MeiliConfig struct {
	BaseUrl        string `yaml:"base-url"`
	IndexName      string `yaml:"index-name"`
	PrimaryKey     string `yaml:"primary-key"`
	MasterKey      string `yaml:"master-key,omitempty"`
	saveUrlCache   string
	searchUrlCache string
}

func (m *MeiliConfig) GetSaveUrl() string {
	if m.saveUrlCache != "" {
		return m.saveUrlCache
	}
	m.saveUrlCache = fmt.Sprintf("%s/indexes/%s/documents?primaryKey=%s", m.BaseUrl, m.IndexName, m.PrimaryKey)
	return m.saveUrlCache
}
func (m *MeiliConfig) GetSearchUrl() string {
	if m.searchUrlCache != "" {
		return m.searchUrlCache
	}
	m.searchUrlCache = fmt.Sprintf("%s/indexes/%s/search", m.BaseUrl, m.IndexName)
	return m.searchUrlCache
}
