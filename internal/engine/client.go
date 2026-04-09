package engine

type EngineHttpClient struct {
	baseUrl string
}

func NewClient(baseUrl string) *EngineHttpClient {
	return &EngineHttpClient{
		baseUrl: baseUrl,
	}
}
