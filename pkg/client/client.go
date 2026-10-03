package client

import (
	"fmt"
	"net/http"
	"time"
)

// Client is the Z.AI API client. It is safe for concurrent use. Each API
// family is exposed as a service (Chat, Models, Images, …); all of them share
// one transport (see transport.go), so retry, error parsing, and
// observability hooks behave identically across endpoints.
type Client struct {
	config     Config
	httpClient *http.Client
	hooks      []Hook

	chat        *ChatService
	models      *ModelsService
	detection   *DetectionService
	quota       *QuotaService
	account     *AccountService
	tools       *ToolsService
	images      *ImagesService
	videos      *VideosService
	audio       *AudioService
	layout      *LayoutService
	files       *FilesService
	batch       *BatchService
	agents      *AgentsService
	embeddings  *EmbeddingsService
	moderations *ModerationsService
	rerank      *RerankService
	voice       *VoiceService
	fileParser  *FileParserService
	anthropic   *AnthropicService
}

// NewClient creates a client from config, resolving unset fields to their
// defaults (see Config).
func NewClient(config Config) (*Client, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	config = config.withDefaults()

	c := &Client{
		config:     config,
		httpClient: config.HTTPClient,
		hooks:      config.Hooks,
	}
	c.chat = &ChatService{client: c}
	c.models = &ModelsService{client: c}
	c.detection = &DetectionService{client: c}
	c.quota = &QuotaService{client: c}
	c.account = &AccountService{client: c}
	c.tools = &ToolsService{client: c}
	c.images = &ImagesService{client: c}
	c.videos = &VideosService{client: c}
	c.audio = &AudioService{client: c}
	c.layout = &LayoutService{client: c}
	c.files = &FilesService{client: c}
	c.batch = &BatchService{client: c}
	c.agents = &AgentsService{client: c}
	c.embeddings = &EmbeddingsService{client: c}
	c.moderations = &ModerationsService{client: c}
	c.rerank = &RerankService{client: c}
	c.voice = &VoiceService{client: c}
	c.fileParser = &FileParserService{client: c}
	c.anthropic = &AnthropicService{client: c}
	return c, nil
}

// chinaAPIKey returns the credential for BigModelBaseURL calls: ChinaAPIKey
// when set, otherwise APIKey (see Config.ChinaAPIKey).
func (c *Client) chinaAPIKey() string {
	if c.config.ChinaAPIKey != "" {
		return c.config.ChinaAPIKey
	}
	return c.config.APIKey
}

// MonitorTimezone returns the timezone the monitor (quota/usage) API operates
// in (Config.MonitorTimezone, defaulting to MonitorServerTZ). Render layers
// use it to convert the server-local x_time bucket labels into the viewer's
// local time and to annotate reset times.
func (c *Client) MonitorTimezone() *time.Location { return c.config.MonitorTimezone }

// Region returns the regional gateway the client targets.
func (c *Client) Region() Region { return c.config.Region }

// Chat returns the chat-completions service.
func (c *Client) Chat() *ChatService { return c.chat }

// Models returns the models service.
func (c *Client) Models() *ModelsService { return c.models }

// Detection returns the account-type detection service.
func (c *Client) Detection() *DetectionService { return c.detection }

// Quota returns the coding-plan quota/usage (monitor) service.
func (c *Client) Quota() *QuotaService { return c.quota }

// Account returns the account (biz) service.
func (c *Client) Account() *AccountService { return c.account }

// Tools returns the tools service (web search, web reader, tokenizer).
func (c *Client) Tools() *ToolsService { return c.tools }

// Images returns the image generation service.
func (c *Client) Images() *ImagesService { return c.images }

// Videos returns the video generation service.
func (c *Client) Videos() *VideosService { return c.videos }

// Audio returns the speech (TTS) and transcription (ASR) service.
func (c *Client) Audio() *AudioService { return c.audio }

// Layout returns the layout parsing (OCR) service.
func (c *Client) Layout() *LayoutService { return c.layout }

// Files returns the file upload/management service.
func (c *Client) Files() *FilesService { return c.files }

// Batch returns the batch job service.
func (c *Client) Batch() *BatchService { return c.batch }

// Agents returns the agents (specialized-agent invocation) service.
func (c *Client) Agents() *AgentsService { return c.agents }

// Embeddings returns the text-embeddings service. It calls BigModelBaseURL
// (open.bigmodel.cn), not Config.BaseURL — see Config.ChinaAPIKey.
func (c *Client) Embeddings() *EmbeddingsService { return c.embeddings }

// Moderations returns the content-moderation service. It calls
// BigModelBaseURL (open.bigmodel.cn), not Config.BaseURL — see
// Config.ChinaAPIKey.
func (c *Client) Moderations() *ModerationsService { return c.moderations }

// Rerank returns the document-reranking service.
func (c *Client) Rerank() *RerankService { return c.rerank }

// Voice returns the voice-cloning service.
func (c *Client) Voice() *VoiceService { return c.voice }

// FileParser returns the document-parsing service.
func (c *Client) FileParser() *FileParserService { return c.fileParser }

// Anthropic returns the Anthropic-compatible Messages service.
func (c *Client) Anthropic() *AnthropicService { return c.anthropic }
