package pantheon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const currentConfigSchemaVersion = 1

var legacyConfigFields = map[string]bool{
	"model":          true,
	"ollama_url":     true,
	"local_model":    true,
	"frontier_model": true,
	"vault_path":     true,
	"roadmap_path":   true,
}

type canonicalConfig struct {
	SchemaVersion int                      `json:"schema_version"`
	Meta          map[string]any           `json:"meta,omitempty"`
	Runtime       *canonicalRuntime        `json:"runtime,omitempty"`
	Paths         *canonicalPaths          `json:"paths,omitempty"`
	Models        *canonicalModels         `json:"models,omitempty"`
	Apollo        *canonicalApolloSettings `json:"apollo,omitempty"`
	Time          *canonicalTime           `json:"time,omitempty"`
	Retention     *canonicalRetention      `json:"retention,omitempty"`
	Ingestion     *canonicalIngestion      `json:"ingestion,omitempty"`
	Rumor         *canonicalRumor          `json:"rumor,omitempty"`
	Present       map[string]any           `json:"present,omitempty"`
	Past          map[string]any           `json:"past,omitempty"`
	Weave         map[string]any           `json:"weave,omitempty"`
	Future        map[string]any           `json:"future,omitempty"`
	Mercury       *canonicalMercury        `json:"mercury,omitempty"`
	Bifrost       *canonicalBifrost        `json:"bifrost,omitempty"`
	Mail          *canonicalMail           `json:"mail,omitempty"`
	Zodiac        *canonicalZodiac         `json:"zodiac,omitempty"`
	Prometheus    map[string]any           `json:"prometheus,omitempty"`
	Notifications map[string]any           `json:"notifications,omitempty"`
	Zeus          map[string]any           `json:"zeus,omitempty"`
	Archimedes    map[string]any           `json:"archimedes,omitempty"`
	Panopticon    map[string]any           `json:"panopticon,omitempty"`
	Setup         map[string]any           `json:"setup,omitempty"`
	Organs        map[string]any           `json:"organs,omitempty"`
}

type canonicalRuntime struct {
	Listen *string `json:"listen,omitempty"`
}

type canonicalPaths struct {
	Vault             *string `json:"vault,omitempty"`
	Roadmap           *string `json:"roadmap,omitempty"`
	Intake            *string `json:"intake,omitempty"`
	FatherTime        *string `json:"father_time_chronicles,omitempty"`
	ProcessedMarkdown *string `json:"processed_markdown,omitempty"`
	RawRetention      *string `json:"raw_retention,omitempty"`
	Raw               *string `json:"raw,omitempty"`
	ApolloReports     *string `json:"apollo_reports,omitempty"`
}

type canonicalIngestion struct {
	Whisper *canonicalWhisper `json:"whisper,omitempty"`
	Media   *canonicalMedia   `json:"media,omitempty"`
}

type canonicalWhisper struct {
	Enabled  *bool   `json:"enabled,omitempty"`
	Binary   *string `json:"binary,omitempty"`
	Model    *string `json:"model,omitempty"`
	Language *string `json:"language,omitempty"`
}

type canonicalMedia struct {
	Converter *string `json:"converter,omitempty"`
}

type canonicalRumor struct {
	Enabled               *bool    `json:"enabled,omitempty"`
	WatchPaths            []string `json:"watch_paths,omitempty"`
	WhisperModel          *string  `json:"whisper_model,omitempty"`
	Language              *string  `json:"language,omitempty"`
	RetryLimit            *int     `json:"retry_limit,omitempty"`
	TranscriptDestination *string  `json:"transcript_destination,omitempty"`
	OnComplete            *string  `json:"on_complete,omitempty"`
	StabilitySeconds      *int     `json:"stability_seconds,omitempty"`
	PollSeconds           *int     `json:"poll_seconds,omitempty"`
}

type canonicalTime struct {
	Timezone *string `json:"timezone,omitempty"`
}

type canonicalRetention struct {
	RawInputDays *int `json:"raw_input_days,omitempty"`
}

type canonicalMercury struct {
	Mode                  *string `json:"mode,omitempty"`
	Agent                 *string `json:"agent,omitempty"`
	Binary                *string `json:"binary,omitempty"`
	DeterministicFallback *bool   `json:"deterministic_fallback,omitempty"`
	RoundOnCronWake       *bool   `json:"round_on_cron_wake,omitempty"`
}

type canonicalBifrost struct {
	Enabled *bool                 `json:"enabled,omitempty"`
	Mail    *canonicalBifrostMail `json:"mail,omitempty"`
}

type canonicalBifrostMail struct {
	Account     *string `json:"account,omitempty"`
	Mailbox     *string `json:"mailbox,omitempty"`
	Capability  *string `json:"capability,omitempty"`
	Schedule    *string `json:"schedule,omitempty"`
	MaxMessages *int    `json:"max_messages,omitempty"`
}

type canonicalMail struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	Adapter          *string `json:"adapter,omitempty"`
	Account          *string `json:"account,omitempty"`
	Inbox            *string `json:"inbox,omitempty"`
	BacklogEnabled   *bool   `json:"backlog_enabled,omitempty"`
	BatchSize        *int    `json:"batch_size,omitempty"`
	Direction        *string `json:"direction,omitempty"`
	MarkReadPolicy   *string `json:"mark_read_policy,omitempty"`
	AttachmentPolicy *string `json:"attachment_policy,omitempty"`
	ActionCapability *string `json:"action_capability,omitempty"`
}

type canonicalZodiac struct {
	Enabled            *bool   `json:"enabled,omitempty"`
	DefaultTimezone    *string `json:"default_timezone,omitempty"`
	PersonalProjection *string `json:"personal_projection,omitempty"`
	WorkProjection     *string `json:"work_projection,omitempty"`
	WorkHoursStart     *string `json:"work_hours_start,omitempty"`
	WorkHoursEnd       *string `json:"work_hours_end,omitempty"`
	BriefTime          *string `json:"brief_time,omitempty"`
	BriefCatchUpPolicy *string `json:"brief_catch_up_policy,omitempty"`
	ConflictPolicy     *string `json:"conflict_policy,omitempty"`
	WakeSeconds        *int    `json:"wake_seconds,omitempty"`
}

type canonicalModels struct {
	Apollo     *canonicalApollo     `json:"apollo,omitempty"`
	Archimedes *canonicalArchimedes `json:"archimedes,omitempty"`
}

type canonicalApollo struct {
	Model    *string `json:"model,omitempty"`
	Endpoint *string `json:"endpoint,omitempty"`
}

type canonicalApolloSettings struct {
	MaxRetrievalRounds       *int                       `json:"max_retrieval_rounds,omitempty"`
	MaxSources               *int                       `json:"max_sources,omitempty"`
	MaxContextBytes          *int                       `json:"max_context_bytes,omitempty"`
	MaxReasoningDurationSecs *int                       `json:"max_reasoning_duration_seconds,omitempty"`
	RequireCitations         *bool                      `json:"require_citations,omitempty"`
	ExternalResearch         *canonicalExternalResearch `json:"external_research,omitempty"`
}

type canonicalExternalResearch struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type canonicalArchimedes struct {
	Model *string `json:"model,omitempty"`
}

type legacyConfig struct {
	Model         string `json:"model,omitempty"`
	OllamaURL     string `json:"ollama_url,omitempty"`
	LocalModel    string `json:"local_model,omitempty"`
	FrontierModel string `json:"frontier_model,omitempty"`
	VaultPath     string `json:"vault_path,omitempty"`
	RoadmapPath   string `json:"roadmap_path,omitempty"`
}

type configResolution struct {
	Config        config
	SchemaVersion int
	Legacy        bool
	Sources       map[string]string
}

type config struct {
	Model                             string   `json:"model,omitempty"`
	OllamaURL                         string   `json:"ollama_url,omitempty"`
	LocalModel                        string   `json:"local_model,omitempty"`
	FrontierModel                     string   `json:"frontier_model,omitempty"`
	VaultPath                         string   `json:"vault_path,omitempty"`
	RoadmapPath                       string   `json:"roadmap_path,omitempty"`
	IntakePath                        string   `json:"intake_path,omitempty"`
	FatherTimeChroniclesPath          string   `json:"father_time_chronicles_path,omitempty"`
	RawPath                           string   `json:"raw_path,omitempty"`
	ApolloReportsPath                 string   `json:"apollo_reports_path,omitempty"`
	RawInputDays                      int      `json:"raw_input_days,omitempty"`
	Timezone                          string   `json:"timezone,omitempty"`
	WhisperEnabled                    bool     `json:"whisper_enabled,omitempty"`
	WhisperBinary                     string   `json:"whisper_binary,omitempty"`
	WhisperModel                      string   `json:"whisper_model,omitempty"`
	WhisperLanguage                   string   `json:"whisper_language,omitempty"`
	MediaConverter                    string   `json:"media_converter,omitempty"`
	RumorEnabled                      bool     `json:"rumor_enabled,omitempty"`
	RumorWatchPaths                   []string `json:"rumor_watch_paths,omitempty"`
	RumorWhisperModel                 string   `json:"rumor_whisper_model,omitempty"`
	RumorLanguage                     string   `json:"rumor_language,omitempty"`
	RumorRetryLimit                   int      `json:"rumor_retry_limit,omitempty"`
	RumorTranscriptDestination        string   `json:"rumor_transcript_destination,omitempty"`
	RumorOnComplete                   string   `json:"rumor_on_complete,omitempty"`
	RumorStabilitySeconds             int      `json:"rumor_stability_seconds,omitempty"`
	RumorPollSeconds                  int      `json:"rumor_poll_seconds,omitempty"`
	MercuryMode                       string   `json:"mercury_mode,omitempty"`
	MercuryAgent                      string   `json:"mercury_agent,omitempty"`
	MercuryBinary                     string   `json:"mercury_binary,omitempty"`
	MercuryDeterministicFallback      bool     `json:"mercury_deterministic_fallback,omitempty"`
	MercuryRoundOnCronWake            bool     `json:"mercury_round_on_cron_wake,omitempty"`
	BifrostEnabled                    bool     `json:"bifrost_enabled,omitempty"`
	BifrostMailAccount                string   `json:"bifrost_mail_account,omitempty"`
	BifrostMailMailbox                string   `json:"bifrost_mailbox,omitempty"`
	BifrostMailCapabilityID           string   `json:"bifrost_mail_capability_id,omitempty"`
	BifrostMailSchedule               string   `json:"bifrost_mail_schedule,omitempty"`
	BifrostMailMaxMessages            int      `json:"bifrost_mail_max_messages,omitempty"`
	MailEnabled                       bool     `json:"mail_enabled,omitempty"`
	MailAdapter                       string   `json:"mail_adapter,omitempty"`
	MailAccount                       string   `json:"mail_account,omitempty"`
	MailInbox                         string   `json:"mail_inbox,omitempty"`
	MailBacklogEnabled                bool     `json:"mail_backlog_enabled,omitempty"`
	MailBatchSize                     int      `json:"mail_batch_size,omitempty"`
	MailDirection                     string   `json:"mail_direction,omitempty"`
	MailMarkReadPolicy                string   `json:"mail_mark_read_policy,omitempty"`
	MailAttachmentPolicy              string   `json:"mail_attachment_policy,omitempty"`
	MailActionCapabilityID            string   `json:"mail_action_capability_id,omitempty"`
	ZodiacEnabled                     bool     `json:"zodiac_enabled,omitempty"`
	ZodiacDefaultTimezone             string   `json:"zodiac_default_timezone,omitempty"`
	ZodiacPersonalProjection          string   `json:"zodiac_personal_projection,omitempty"`
	ZodiacWorkProjection              string   `json:"zodiac_work_projection,omitempty"`
	ZodiacWorkHoursStart              string   `json:"zodiac_work_hours_start,omitempty"`
	ZodiacWorkHoursEnd                string   `json:"zodiac_work_hours_end,omitempty"`
	ZodiacBriefTime                   string   `json:"zodiac_brief_time,omitempty"`
	ZodiacBriefCatchUpPolicy          string   `json:"zodiac_brief_catch_up_policy,omitempty"`
	ZodiacConflictPolicy              string   `json:"zodiac_conflict_policy,omitempty"`
	ZodiacWakeSeconds                 int      `json:"zodiac_wake_seconds,omitempty"`
	ApolloMaxRetrievalRounds          int      `json:"apollo_max_retrieval_rounds,omitempty"`
	ApolloMaxSources                  int      `json:"apollo_max_sources,omitempty"`
	ApolloMaxContextBytes             int      `json:"apollo_max_context_bytes,omitempty"`
	ApolloMaxReasoningDurationSeconds int      `json:"apollo_max_reasoning_duration_seconds,omitempty"`
	ApolloRequireCitations            bool     `json:"apollo_require_citations,omitempty"`
	ApolloExternalResearchEnabled     bool     `json:"apollo_external_research_enabled,omitempty"`
	IntakeConfigured                  bool     `json:"-"`
	FatherTimeConfigured              bool     `json:"-"`
	SchemaVersion                     int      `json:"schema_version,omitempty"`
	ListenAddress                     string   `json:"listen_address,omitempty"`
}

type migrationResult struct {
	Path       string
	Legacy     bool
	Noop       bool
	Applied    bool
	BackupPath string
	Proposed   []byte
}

func loadConfigPath(path string) (config, error) {
	resolution, err := readConfigResolution(path)
	if err != nil {
		return config{}, err
	}
	return resolution.Config, nil
}

func readConfigResolution(path string) (configResolution, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configResolution{}, err
	}
	return parseConfigBytes(path, data)
}

func parseConfigBytes(path string, data []byte) (configResolution, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return configResolution{}, fmt.Errorf("configuration %s is malformed JSON: %w", path, err)
	}
	if raw == nil {
		return configResolution{}, fmt.Errorf("configuration %s must contain a JSON object", path)
	}

	if versionRaw, ok := raw["schema_version"]; ok {
		var version int
		if err := json.Unmarshal(versionRaw, &version); err != nil {
			return configResolution{}, fmt.Errorf("configuration %s has invalid schema_version: %w", path, err)
		}
		if version != currentConfigSchemaVersion {
			return configResolution{}, fmt.Errorf("configuration %s uses unsupported schema_version %d (supported: %d)", path, version, currentConfigSchemaVersion)
		}
		var canonical canonicalConfig
		if err := json.Unmarshal(data, &canonical); err != nil {
			return configResolution{}, fmt.Errorf("configuration %s has invalid field types: %w", path, err)
		}
		return resolveCanonicalConfig(path, canonical)
	}

	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return configResolution{}, fmt.Errorf("configuration %s has invalid legacy field types: %w", path, err)
	}
	return resolveLegacyConfig(path, raw, legacy)
}

func resolveCanonicalConfig(path string, canonical canonicalConfig) (configResolution, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return configResolution{}, fmt.Errorf("resolve home directory: %w", err)
	}
	resolved := config{
		SchemaVersion:   currentConfigSchemaVersion,
		ListenAddress:   defaultListenAddress,
		IntakePath:      filepath.Join(home, "Pantheon Intake"),
		RawInputDays:    30,
		WhisperEnabled:  true,
		WhisperLanguage: "auto",
		RumorRetryLimit: 3, RumorLanguage: "auto", RumorOnComplete: "intake", RumorStabilitySeconds: 2, RumorPollSeconds: 5,
		MercuryMode: "hermes", MercuryAgent: "Terra", MercuryDeterministicFallback: true, MercuryRoundOnCronWake: true,
		BifrostEnabled: true, BifrostMailAccount: "gmail", BifrostMailMailbox: "INBOX", BifrostMailCapabilityID: "gmail-mail", BifrostMailSchedule: "hourly", BifrostMailMaxMessages: 50,
		MailEnabled: true, MailAdapter: "himalaya", MailAccount: "gmail", MailInbox: "INBOX", MailBacklogEnabled: true, MailBatchSize: 100, MailDirection: "newest_to_oldest", MailMarkReadPolicy: "standing_authority_only", MailAttachmentPolicy: "approval_required", MailActionCapabilityID: "gmail-mail-actions",
		ZodiacEnabled: true, ZodiacDefaultTimezone: "America/Los_Angeles", ZodiacPersonalProjection: "apple", ZodiacWorkProjection: "google", ZodiacWorkHoursStart: "09:00", ZodiacWorkHoursEnd: "17:00", ZodiacBriefTime: "07:00", ZodiacBriefCatchUpPolicy: "run_once", ZodiacConflictPolicy: "needs_you", ZodiacWakeSeconds: 30,
		ApolloMaxRetrievalRounds: 12, ApolloMaxSources: 64, ApolloMaxContextBytes: 200000, ApolloMaxReasoningDurationSeconds: 900, ApolloRequireCitations: true,
	}
	sources := map[string]string{
		"schema_version":          "user",
		"runtime.listen":          "default",
		"models.archimedes.model": "default",
	}
	if canonical.Runtime != nil && canonical.Runtime.Listen != nil {
		resolved.ListenAddress = strings.TrimSpace(*canonical.Runtime.Listen)
		sources["runtime.listen"] = "user"
	}
	if canonical.Paths != nil {
		if canonical.Paths.Vault != nil {
			resolved.VaultPath = *canonical.Paths.Vault
			sources["paths.vault"] = "user"
		}
		if canonical.Paths.Roadmap != nil {
			resolved.RoadmapPath = *canonical.Paths.Roadmap
			sources["paths.roadmap"] = "user"
		}
		if canonical.Paths.Intake != nil {
			if strings.TrimSpace(*canonical.Paths.Intake) != "" {
				resolved.IntakePath = *canonical.Paths.Intake
				resolved.IntakeConfigured = true
				sources["paths.intake"] = "user"
			} else {
				sources["paths.intake"] = "default"
			}
		} else {
			sources["paths.intake"] = "default"
		}
		if canonical.Paths.FatherTime != nil {
			if strings.TrimSpace(*canonical.Paths.FatherTime) != "" {
				resolved.FatherTimeChroniclesPath = *canonical.Paths.FatherTime
				resolved.FatherTimeConfigured = true
				sources["paths.father_time_chronicles"] = "user"
			} else {
				sources["paths.father_time_chronicles"] = "default beneath paths.vault"
			}
		} else {
			sources["paths.father_time_chronicles"] = "default beneath paths.vault"
		}
		if canonical.Paths.Raw != nil {
			if strings.TrimSpace(*canonical.Paths.Raw) != "" {
				resolved.RawPath = *canonical.Paths.Raw
				sources["paths.raw"] = "user"
			} else {
				sources["paths.raw"] = "default beneath paths.vault"
			}
		} else {
			sources["paths.raw"] = "default beneath paths.vault"
		}
		if canonical.Paths.ApolloReports != nil {
			if strings.TrimSpace(*canonical.Paths.ApolloReports) != "" {
				resolved.ApolloReportsPath = *canonical.Paths.ApolloReports
				sources["paths.apollo_reports"] = "user"
			} else {
				sources["paths.apollo_reports"] = "default beneath paths.vault"
			}
		} else {
			sources["paths.apollo_reports"] = "default beneath paths.vault"
		}
	} else {
		sources["paths.intake"] = "default"
		sources["paths.father_time_chronicles"] = "default beneath paths.vault"
		sources["paths.raw"] = "default beneath paths.vault"
		sources["paths.apollo_reports"] = "default beneath paths.vault"
	}
	if canonical.Time != nil && canonical.Time.Timezone != nil {
		resolved.Timezone = strings.TrimSpace(*canonical.Time.Timezone)
		sources["time.timezone"] = "user"
	} else {
		sources["time.timezone"] = "machine local"
	}
	if canonical.Retention != nil && canonical.Retention.RawInputDays != nil {
		resolved.RawInputDays = *canonical.Retention.RawInputDays
		sources["retention.raw_input_days"] = "user"
	} else {
		sources["retention.raw_input_days"] = "default"
	}
	if canonical.Models != nil {
		if canonical.Models.Apollo != nil {
			if canonical.Models.Apollo.Model != nil {
				resolved.LocalModel = strings.TrimSpace(*canonical.Models.Apollo.Model)
				resolved.Model = resolved.LocalModel
				sources["models.apollo.model"] = "user"
			}
			if canonical.Models.Apollo.Endpoint != nil {
				resolved.OllamaURL = strings.TrimSpace(*canonical.Models.Apollo.Endpoint)
				sources["models.apollo.endpoint"] = "user"
			}
		}
		if canonical.Models.Archimedes != nil && canonical.Models.Archimedes.Model != nil {
			resolved.FrontierModel = strings.TrimSpace(*canonical.Models.Archimedes.Model)
			sources["models.archimedes.model"] = "user"
		}
	}
	if canonical.Apollo != nil {
		if canonical.Apollo.MaxRetrievalRounds != nil {
			resolved.ApolloMaxRetrievalRounds = *canonical.Apollo.MaxRetrievalRounds
		}
		if canonical.Apollo.MaxSources != nil {
			resolved.ApolloMaxSources = *canonical.Apollo.MaxSources
		}
		if canonical.Apollo.MaxContextBytes != nil {
			resolved.ApolloMaxContextBytes = *canonical.Apollo.MaxContextBytes
		}
		if canonical.Apollo.MaxReasoningDurationSecs != nil {
			resolved.ApolloMaxReasoningDurationSeconds = *canonical.Apollo.MaxReasoningDurationSecs
		}
		if canonical.Apollo.RequireCitations != nil {
			resolved.ApolloRequireCitations = *canonical.Apollo.RequireCitations
		}
		if canonical.Apollo.ExternalResearch != nil && canonical.Apollo.ExternalResearch.Enabled != nil {
			resolved.ApolloExternalResearchEnabled = *canonical.Apollo.ExternalResearch.Enabled
		}
	}
	if canonical.Ingestion != nil {
		if canonical.Ingestion.Whisper != nil {
			if canonical.Ingestion.Whisper.Enabled != nil {
				resolved.WhisperEnabled = *canonical.Ingestion.Whisper.Enabled
			}
			if canonical.Ingestion.Whisper.Binary != nil {
				resolved.WhisperBinary = strings.TrimSpace(*canonical.Ingestion.Whisper.Binary)
			}
			if canonical.Ingestion.Whisper.Model != nil {
				resolved.WhisperModel = strings.TrimSpace(*canonical.Ingestion.Whisper.Model)
			}
			if canonical.Ingestion.Whisper.Language != nil {
				resolved.WhisperLanguage = strings.TrimSpace(*canonical.Ingestion.Whisper.Language)
			}
		}
		if canonical.Ingestion.Media != nil && canonical.Ingestion.Media.Converter != nil {
			resolved.MediaConverter = strings.TrimSpace(*canonical.Ingestion.Media.Converter)
		}
	}
	if canonical.Rumor != nil {
		if canonical.Rumor.Enabled != nil {
			resolved.RumorEnabled = *canonical.Rumor.Enabled
		}
		if canonical.Rumor.WatchPaths != nil {
			resolved.RumorWatchPaths = append([]string(nil), canonical.Rumor.WatchPaths...)
		}
		if canonical.Rumor.WhisperModel != nil {
			resolved.RumorWhisperModel = strings.TrimSpace(*canonical.Rumor.WhisperModel)
		}
		if canonical.Rumor.Language != nil {
			resolved.RumorLanguage = strings.TrimSpace(*canonical.Rumor.Language)
		}
		if canonical.Rumor.RetryLimit != nil {
			resolved.RumorRetryLimit = *canonical.Rumor.RetryLimit
		}
		if canonical.Rumor.TranscriptDestination != nil {
			resolved.RumorTranscriptDestination = strings.TrimSpace(*canonical.Rumor.TranscriptDestination)
		}
		if canonical.Rumor.OnComplete != nil {
			resolved.RumorOnComplete = strings.TrimSpace(*canonical.Rumor.OnComplete)
		}
		if canonical.Rumor.StabilitySeconds != nil {
			resolved.RumorStabilitySeconds = *canonical.Rumor.StabilitySeconds
		}
		if canonical.Rumor.PollSeconds != nil {
			resolved.RumorPollSeconds = *canonical.Rumor.PollSeconds
		}
	}
	if canonical.Mercury != nil {
		if canonical.Mercury.Mode != nil {
			resolved.MercuryMode = strings.TrimSpace(*canonical.Mercury.Mode)
		}
		if canonical.Mercury.Agent != nil {
			resolved.MercuryAgent = strings.TrimSpace(*canonical.Mercury.Agent)
		}
		if canonical.Mercury.Binary != nil {
			resolved.MercuryBinary = strings.TrimSpace(*canonical.Mercury.Binary)
		}
		if canonical.Mercury.DeterministicFallback != nil {
			resolved.MercuryDeterministicFallback = *canonical.Mercury.DeterministicFallback
		}
		if canonical.Mercury.RoundOnCronWake != nil {
			resolved.MercuryRoundOnCronWake = *canonical.Mercury.RoundOnCronWake
		}
	}
	if canonical.Bifrost != nil {
		if canonical.Bifrost.Enabled != nil {
			resolved.BifrostEnabled = *canonical.Bifrost.Enabled
		}
		if canonical.Bifrost.Mail != nil {
			if canonical.Bifrost.Mail.Account != nil {
				resolved.BifrostMailAccount = strings.TrimSpace(*canonical.Bifrost.Mail.Account)
			}
			if canonical.Bifrost.Mail.Mailbox != nil {
				resolved.BifrostMailMailbox = strings.TrimSpace(*canonical.Bifrost.Mail.Mailbox)
			}
			if canonical.Bifrost.Mail.Capability != nil {
				resolved.BifrostMailCapabilityID = strings.TrimSpace(*canonical.Bifrost.Mail.Capability)
			}
			if canonical.Bifrost.Mail.Schedule != nil {
				resolved.BifrostMailSchedule = strings.TrimSpace(*canonical.Bifrost.Mail.Schedule)
			}
			if canonical.Bifrost.Mail.MaxMessages != nil {
				resolved.BifrostMailMaxMessages = *canonical.Bifrost.Mail.MaxMessages
			}
		}
	}
	if canonical.Mail != nil {
		if canonical.Mail.Enabled != nil {
			resolved.MailEnabled = *canonical.Mail.Enabled
		}
		if canonical.Mail.Adapter != nil {
			resolved.MailAdapter = strings.TrimSpace(*canonical.Mail.Adapter)
		}
		if canonical.Mail.Account != nil {
			resolved.MailAccount = strings.TrimSpace(*canonical.Mail.Account)
		}
		if canonical.Mail.Inbox != nil {
			resolved.MailInbox = strings.TrimSpace(*canonical.Mail.Inbox)
		}
		if canonical.Mail.BacklogEnabled != nil {
			resolved.MailBacklogEnabled = *canonical.Mail.BacklogEnabled
		}
		if canonical.Mail.BatchSize != nil {
			resolved.MailBatchSize = *canonical.Mail.BatchSize
		}
		if canonical.Mail.Direction != nil {
			resolved.MailDirection = strings.TrimSpace(*canonical.Mail.Direction)
		}
		if canonical.Mail.MarkReadPolicy != nil {
			resolved.MailMarkReadPolicy = strings.TrimSpace(*canonical.Mail.MarkReadPolicy)
		}
		if canonical.Mail.AttachmentPolicy != nil {
			resolved.MailAttachmentPolicy = strings.TrimSpace(*canonical.Mail.AttachmentPolicy)
		}
		if canonical.Mail.ActionCapability != nil {
			resolved.MailActionCapabilityID = strings.TrimSpace(*canonical.Mail.ActionCapability)
		}
	}
	if canonical.Zodiac != nil {
		if canonical.Zodiac.Enabled != nil {
			resolved.ZodiacEnabled = *canonical.Zodiac.Enabled
		}
		if canonical.Zodiac.DefaultTimezone != nil {
			resolved.ZodiacDefaultTimezone = strings.TrimSpace(*canonical.Zodiac.DefaultTimezone)
		}
		if canonical.Zodiac.PersonalProjection != nil {
			resolved.ZodiacPersonalProjection = strings.TrimSpace(*canonical.Zodiac.PersonalProjection)
		}
		if canonical.Zodiac.WorkProjection != nil {
			resolved.ZodiacWorkProjection = strings.TrimSpace(*canonical.Zodiac.WorkProjection)
		}
		if canonical.Zodiac.WorkHoursStart != nil {
			resolved.ZodiacWorkHoursStart = strings.TrimSpace(*canonical.Zodiac.WorkHoursStart)
		}
		if canonical.Zodiac.WorkHoursEnd != nil {
			resolved.ZodiacWorkHoursEnd = strings.TrimSpace(*canonical.Zodiac.WorkHoursEnd)
		}
		if canonical.Zodiac.BriefTime != nil {
			resolved.ZodiacBriefTime = strings.TrimSpace(*canonical.Zodiac.BriefTime)
		}
		if canonical.Zodiac.BriefCatchUpPolicy != nil {
			resolved.ZodiacBriefCatchUpPolicy = strings.TrimSpace(*canonical.Zodiac.BriefCatchUpPolicy)
		}
		if canonical.Zodiac.ConflictPolicy != nil {
			resolved.ZodiacConflictPolicy = strings.TrimSpace(*canonical.Zodiac.ConflictPolicy)
		}
		if canonical.Zodiac.WakeSeconds != nil {
			resolved.ZodiacWakeSeconds = *canonical.Zodiac.WakeSeconds
		}
	}
	if resolved.MercuryMode == "" {
		resolved.MercuryMode = "hermes"
	}
	if resolved.MercuryAgent == "" {
		resolved.MercuryAgent = "Terra"
	}
	if resolved.RawPath == "" && resolved.VaultPath != "" {
		resolved.RawPath = filepath.Join(resolved.VaultPath, "80 Pantheon Raw")
	}
	if resolved.ApolloReportsPath == "" && resolved.VaultPath != "" {
		resolved.ApolloReportsPath = filepath.Join(resolved.VaultPath, "70 Pantheon Reports", "Apollo")
	}
	if resolved.ApolloMaxRetrievalRounds == 0 {
		resolved.ApolloMaxRetrievalRounds = 12
	}
	if resolved.ApolloMaxSources == 0 {
		resolved.ApolloMaxSources = 64
	}
	if resolved.ApolloMaxContextBytes == 0 {
		resolved.ApolloMaxContextBytes = 200000
	}
	if resolved.ApolloMaxReasoningDurationSeconds == 0 {
		resolved.ApolloMaxReasoningDurationSeconds = 900
	}
	if resolved.WhisperLanguage == "" {
		resolved.WhisperLanguage = "auto"
	}
	if resolved.RumorLanguage == "" {
		resolved.RumorLanguage = resolved.WhisperLanguage
	}
	if resolved.RumorOnComplete == "" {
		resolved.RumorOnComplete = "intake"
	}
	if resolved.RumorTranscriptDestination == "" && resolved.VaultPath != "" {
		resolved.RumorTranscriptDestination = filepath.Join(resolved.VaultPath, "80 Pantheon Raw", "Audio Transcripts")
	}
	if resolved.BifrostMailAccount == "" {
		resolved.BifrostMailAccount = "gmail"
	}
	if resolved.BifrostMailMailbox == "" {
		resolved.BifrostMailMailbox = "INBOX"
	}
	if resolved.BifrostMailCapabilityID == "" {
		resolved.BifrostMailCapabilityID = "gmail-mail"
	}
	if resolved.BifrostMailSchedule == "" {
		resolved.BifrostMailSchedule = "hourly"
	}
	if resolved.MailActionCapabilityID == "" {
		resolved.MailActionCapabilityID = "gmail-mail-actions"
	}
	if resolved.BifrostMailMaxMessages == 0 {
		resolved.BifrostMailMaxMessages = 50
	}
	if resolved.MailAdapter == "" {
		resolved.MailAdapter = "himalaya"
	}
	if resolved.MailAccount == "" {
		resolved.MailAccount = resolved.BifrostMailAccount
	}
	if resolved.MailInbox == "" {
		resolved.MailInbox = resolved.BifrostMailMailbox
	}
	if resolved.MailBatchSize == 0 {
		resolved.MailBatchSize = 100
	}
	if resolved.MailDirection == "" {
		resolved.MailDirection = "newest_to_oldest"
	}
	if resolved.MailMarkReadPolicy == "" {
		resolved.MailMarkReadPolicy = "standing_authority_only"
	}
	if resolved.MailAttachmentPolicy == "" {
		resolved.MailAttachmentPolicy = "approval_required"
	}
	if resolved.ZodiacDefaultTimezone == "" {
		resolved.ZodiacDefaultTimezone = "America/Los_Angeles"
	}
	if resolved.ZodiacPersonalProjection == "" {
		resolved.ZodiacPersonalProjection = "apple"
	}
	if resolved.ZodiacWorkProjection == "" {
		resolved.ZodiacWorkProjection = "google"
	}
	if resolved.ZodiacWorkHoursStart == "" {
		resolved.ZodiacWorkHoursStart = "09:00"
	}
	if resolved.ZodiacWorkHoursEnd == "" {
		resolved.ZodiacWorkHoursEnd = "17:00"
	}
	if resolved.ZodiacBriefTime == "" {
		resolved.ZodiacBriefTime = "07:00"
	}
	if resolved.ZodiacBriefCatchUpPolicy == "" {
		resolved.ZodiacBriefCatchUpPolicy = "run_once"
	}
	if resolved.ZodiacConflictPolicy == "" {
		resolved.ZodiacConflictPolicy = "needs_you"
	}
	if resolved.ZodiacWakeSeconds == 0 {
		resolved.ZodiacWakeSeconds = 30
	}
	if err := validateResolvedConfig(resolved); err != nil {
		return configResolution{}, fmt.Errorf("configuration %s is invalid: %w", path, err)
	}
	return configResolution{Config: resolved, SchemaVersion: currentConfigSchemaVersion, Sources: sources}, nil
}

func resolveLegacyConfig(path string, raw map[string]json.RawMessage, legacy legacyConfig) (configResolution, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return configResolution{}, fmt.Errorf("resolve home directory: %w", err)
	}
	var unknown []string
	for key := range raw {
		if !legacyConfigFields[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return configResolution{}, fmt.Errorf("configuration %s contains unrecognized legacy fields: %s; migration will not reinterpret them", path, strings.Join(unknown, ", "))
	}
	localModel := strings.TrimSpace(legacy.LocalModel)
	if localModel == "" {
		localModel = strings.TrimSpace(legacy.Model)
	}
	resolved := config{
		Model:           localModel,
		LocalModel:      localModel,
		OllamaURL:       strings.TrimSpace(legacy.OllamaURL),
		FrontierModel:   strings.TrimSpace(legacy.FrontierModel),
		VaultPath:       legacy.VaultPath,
		RoadmapPath:     legacy.RoadmapPath,
		IntakePath:      filepath.Join(home, "Pantheon Intake"),
		RawPath:         "",
		RawInputDays:    30,
		WhisperEnabled:  true,
		WhisperLanguage: "auto",
		RumorRetryLimit: 3, RumorLanguage: "auto", RumorOnComplete: "intake", RumorStabilitySeconds: 2, RumorPollSeconds: 5,
		MercuryMode: "hermes", MercuryAgent: "Terra", MercuryDeterministicFallback: true, MercuryRoundOnCronWake: true,
		BifrostEnabled: true, BifrostMailAccount: "gmail", BifrostMailMailbox: "INBOX", BifrostMailCapabilityID: "gmail-mail", BifrostMailSchedule: "hourly", BifrostMailMaxMessages: 50,
		MailEnabled: true, MailAdapter: "himalaya", MailAccount: "gmail", MailInbox: "INBOX", MailBacklogEnabled: true, MailBatchSize: 100, MailDirection: "newest_to_oldest", MailMarkReadPolicy: "standing_authority_only", MailAttachmentPolicy: "approval_required", MailActionCapabilityID: "gmail-mail-actions",
		ZodiacEnabled: true, ZodiacDefaultTimezone: "America/Los_Angeles", ZodiacPersonalProjection: "apple", ZodiacWorkProjection: "google", ZodiacWorkHoursStart: "09:00", ZodiacWorkHoursEnd: "17:00", ZodiacBriefTime: "07:00", ZodiacBriefCatchUpPolicy: "run_once", ZodiacConflictPolicy: "needs_you", ZodiacWakeSeconds: 30,
		ApolloMaxRetrievalRounds: 12, ApolloMaxSources: 64, ApolloMaxContextBytes: 200000, ApolloMaxReasoningDurationSeconds: 900, ApolloRequireCitations: true,
		SchemaVersion: 0,
		ListenAddress: defaultListenAddress,
	}
	if resolved.VaultPath != "" {
		resolved.RawPath = filepath.Join(resolved.VaultPath, "80 Pantheon Raw")
	}
	if err := validateResolvedConfig(resolved); err != nil {
		return configResolution{}, fmt.Errorf("legacy configuration %s is invalid: %w", path, err)
	}
	sources := map[string]string{
		"schema_version": "legacy",
		"runtime.listen": "default",
	}
	if _, ok := raw["local_model"]; ok {
		sources["models.apollo.model"] = "legacy"
	} else if _, ok := raw["model"]; ok {
		sources["models.apollo.model"] = "legacy"
	}
	if _, ok := raw["ollama_url"]; ok {
		sources["models.apollo.endpoint"] = "legacy"
	}
	if _, ok := raw["frontier_model"]; ok {
		sources["models.archimedes.model"] = "legacy"
	} else {
		sources["models.archimedes.model"] = "default"
	}
	if _, ok := raw["vault_path"]; ok {
		sources["paths.vault"] = "legacy"
	}
	if _, ok := raw["roadmap_path"]; ok {
		sources["paths.roadmap"] = "legacy"
	}
	sources["paths.intake"] = "default"
	sources["paths.father_time_chronicles"] = "default beneath paths.vault"
	sources["paths.raw"] = "default beneath paths.vault"
	sources["retention.raw_input_days"] = "default"
	sources["time.timezone"] = "machine local"
	return configResolution{Config: resolved, SchemaVersion: 0, Legacy: true, Sources: sources}, nil
}

func validateResolvedConfig(cfg config) error {
	if strings.TrimSpace(cfg.LocalModel) == "" {
		return errors.New("models.apollo.model is required (legacy fields: local_model or model)")
	}
	if strings.TrimSpace(cfg.OllamaURL) == "" {
		return errors.New("models.apollo.endpoint is required (legacy field: ollama_url)")
	}
	listen := strings.TrimSpace(cfg.ListenAddress)
	if listen == "" {
		listen = defaultListenAddress
	}
	if listen != defaultListenAddress {
		return fmt.Errorf("runtime.listen must remain the approved localhost address %q", defaultListenAddress)
	}
	endpoint, err := url.Parse(cfg.OllamaURL)
	if err != nil || endpoint.Scheme != "http" || !localApolloHosts[strings.ToLower(endpoint.Hostname())] {
		return errors.New("models.apollo.endpoint must use approved local HTTP transport")
	}
	if endpoint.User != nil {
		return errors.New("models.apollo.endpoint must not contain credentials; configure secrets through a separate mechanism")
	}
	if cfg.RawInputDays <= 0 || cfg.RawInputDays > 36500 {
		return errors.New("retention.raw_input_days must be between 1 and 36500")
	}
	if strings.TrimSpace(cfg.Timezone) != "" {
		if _, err := time.LoadLocation(strings.TrimSpace(cfg.Timezone)); err != nil {
			return fmt.Errorf("time.timezone is invalid: %w", err)
		}
	}
	if strings.TrimSpace(cfg.ZodiacDefaultTimezone) == "" {
		return errors.New("zodiac.default_timezone is required")
	}
	if _, err := time.LoadLocation(strings.TrimSpace(cfg.ZodiacDefaultTimezone)); err != nil {
		return fmt.Errorf("zodiac.default_timezone is invalid: %w", err)
	}
	for field, value := range map[string]string{"zodiac.personal_projection": cfg.ZodiacPersonalProjection, "zodiac.work_projection": cfg.ZodiacWorkProjection} {
		if value != "apple" && value != "google" && value != "both" && value != "none" {
			return fmt.Errorf("%s must be apple, google, both, or none", field)
		}
	}
	for field, value := range map[string]string{"zodiac.work_hours_start": cfg.ZodiacWorkHoursStart, "zodiac.work_hours_end": cfg.ZodiacWorkHoursEnd, "zodiac.brief_time": cfg.ZodiacBriefTime} {
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Errorf("%s must use HH:MM: %w", field, err)
		}
	}
	if cfg.ZodiacBriefCatchUpPolicy != "skip" && cfg.ZodiacBriefCatchUpPolicy != "run_once" && cfg.ZodiacBriefCatchUpPolicy != "reconcile" && cfg.ZodiacBriefCatchUpPolicy != "needs_you" {
		return errors.New("zodiac.brief_catch_up_policy must be skip, run_once, reconcile, or needs_you")
	}
	if cfg.ZodiacConflictPolicy != "needs_you" {
		return errors.New("zodiac.conflict_policy must remain needs_you until an explicit deterministic policy exists")
	}
	if cfg.ZodiacWakeSeconds < 5 || cfg.ZodiacWakeSeconds > 86400 {
		return errors.New("zodiac.wake_seconds must be between 5 and 86400")
	}
	for field, value := range map[string]string{"paths.vault": cfg.VaultPath, "paths.roadmap": cfg.RoadmapPath, "paths.intake": cfg.IntakePath, "paths.father_time_chronicles": cfg.FatherTimeChroniclesPath, "paths.raw": cfg.RawPath, "paths.apollo_reports": cfg.ApolloReportsPath, "ingestion.whisper.binary": cfg.WhisperBinary, "ingestion.media.converter": cfg.MediaConverter} {
		if value == "" {
			continue
		}
		if strings.IndexByte(value, 0) >= 0 {
			return fmt.Errorf("%s contains a NUL byte", field)
		}
		if !filepath.IsAbs(value) {
			return fmt.Errorf("%s must be an absolute path when configured", field)
		}
	}
	if cfg.FatherTimeChroniclesPath != "" && cfg.VaultPath != "" {
		rel, err := filepath.Rel(filepath.Clean(cfg.VaultPath), filepath.Clean(cfg.FatherTimeChroniclesPath))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("paths.father_time_chronicles must remain beneath paths.vault")
		}
	}
	if cfg.RawPath != "" && cfg.VaultPath != "" {
		rel, err := filepath.Rel(filepath.Clean(cfg.VaultPath), filepath.Clean(cfg.RawPath))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("paths.raw must remain beneath paths.vault")
		}
	}
	if cfg.ApolloReportsPath != "" && cfg.VaultPath != "" {
		rel, err := filepath.Rel(filepath.Clean(cfg.VaultPath), filepath.Clean(cfg.ApolloReportsPath))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("paths.apollo_reports must remain beneath paths.vault")
		}
	}
	for _, watchPath := range cfg.RumorWatchPaths {
		if strings.TrimSpace(watchPath) == "" {
			return errors.New("rumor.watch_paths must not contain empty paths")
		}
		if !filepath.IsAbs(watchPath) {
			return errors.New("rumor.watch_paths must contain absolute paths")
		}
	}
	if cfg.RumorTranscriptDestination != "" && !filepath.IsAbs(cfg.RumorTranscriptDestination) {
		return errors.New("rumor.transcript_destination must be an absolute path when configured")
	}
	if cfg.RumorRetryLimit <= 0 || cfg.RumorRetryLimit > 100 {
		return errors.New("rumor.retry_limit must be between 1 and 100")
	}
	if strings.TrimSpace(cfg.BifrostMailAccount) == "" || strings.TrimSpace(cfg.BifrostMailMailbox) == "" || strings.TrimSpace(cfg.BifrostMailCapabilityID) == "" {
		return errors.New("bifrost.mail account, mailbox, and capability must be configured")
	}
	if cfg.BifrostMailMaxMessages <= 0 || cfg.BifrostMailMaxMessages > 250 {
		return errors.New("bifrost.mail.max_messages must be between 1 and 250")
	}
	if cfg.MailAdapter != "himalaya" && cfg.MailAdapter != "gmail" && cfg.MailAdapter != "mcp" {
		return errors.New("mail.adapter must be himalaya, gmail, or mcp")
	}
	if strings.TrimSpace(cfg.MailAccount) == "" || strings.TrimSpace(cfg.MailInbox) == "" {
		return errors.New("mail.account and mail.inbox must be configured")
	}
	if cfg.MailBatchSize <= 0 || cfg.MailBatchSize > 100 {
		return errors.New("mail.backlog.batch_size must be between 1 and 100")
	}
	if cfg.MailDirection != "newest_to_oldest" {
		return errors.New("mail.backlog.direction must be newest_to_oldest")
	}
	if cfg.MailMarkReadPolicy != "standing_authority_only" && cfg.MailMarkReadPolicy != "never" {
		return errors.New("mail.mark_read_policy must be standing_authority_only or never")
	}
	if cfg.MailAttachmentPolicy != "approval_required" {
		return errors.New("mail.attachment_policy must be approval_required")
	}
	if strings.TrimSpace(cfg.MailActionCapabilityID) == "" {
		return errors.New("mail.action_capability must be configured")
	}
	if cfg.RumorStabilitySeconds < 0 || cfg.RumorStabilitySeconds > 86400 {
		return errors.New("rumor.stability_seconds must be between 0 and 86400")
	}
	if cfg.RumorPollSeconds <= 0 || cfg.RumorPollSeconds > 86400 {
		return errors.New("rumor.poll_seconds must be between 1 and 86400")
	}
	if cfg.RumorOnComplete != "intake" && cfg.RumorOnComplete != "none" && cfg.RumorOnComplete != "" {
		return errors.New("rumor.on_complete must be intake or none")
	}
	if cfg.ApolloMaxRetrievalRounds <= 0 || cfg.ApolloMaxSources <= 0 || cfg.ApolloMaxContextBytes <= 0 || cfg.ApolloMaxReasoningDurationSeconds <= 0 {
		return errors.New("apollo retrieval and reasoning budgets must be positive")
	}
	return nil
}

func canonicalConfigFor(cfg config) canonicalConfig {
	listen := cfg.ListenAddress
	if listen == "" {
		listen = defaultListenAddress
	}
	vault, roadmap, intake, fatherTime, raw, apolloReports := cfg.VaultPath, cfg.RoadmapPath, cfg.IntakePath, cfg.FatherTimeChroniclesPath, cfg.RawPath, cfg.ApolloReportsPath
	model, endpoint, archimedes := cfg.LocalModel, cfg.OllamaURL, cfg.FrontierModel
	retentionDays := cfg.RawInputDays
	if retentionDays == 0 {
		retentionDays = 30
	}
	return canonicalConfig{
		SchemaVersion: currentConfigSchemaVersion,
		Runtime:       &canonicalRuntime{Listen: &listen},
		Paths:         &canonicalPaths{Vault: &vault, Roadmap: &roadmap, Intake: &intake, FatherTime: &fatherTime, Raw: &raw, ApolloReports: &apolloReports},
		Time:          &canonicalTime{Timezone: &cfg.Timezone},
		Retention:     &canonicalRetention{RawInputDays: &retentionDays},
		Models: &canonicalModels{
			Apollo:     &canonicalApollo{Model: &model, Endpoint: &endpoint},
			Archimedes: &canonicalArchimedes{Model: &archimedes},
		},
		Ingestion: &canonicalIngestion{
			Whisper: &canonicalWhisper{Enabled: &cfg.WhisperEnabled, Binary: &cfg.WhisperBinary, Model: &cfg.WhisperModel, Language: &cfg.WhisperLanguage},
			Media:   &canonicalMedia{Converter: &cfg.MediaConverter},
		},
		Rumor:   &canonicalRumor{Enabled: &cfg.RumorEnabled, WatchPaths: append([]string(nil), cfg.RumorWatchPaths...), WhisperModel: &cfg.RumorWhisperModel, Language: &cfg.RumorLanguage, RetryLimit: &cfg.RumorRetryLimit, TranscriptDestination: &cfg.RumorTranscriptDestination, OnComplete: &cfg.RumorOnComplete, StabilitySeconds: &cfg.RumorStabilitySeconds, PollSeconds: &cfg.RumorPollSeconds},
		Mercury: &canonicalMercury{Mode: &cfg.MercuryMode, Agent: &cfg.MercuryAgent, Binary: &cfg.MercuryBinary, DeterministicFallback: &cfg.MercuryDeterministicFallback, RoundOnCronWake: &cfg.MercuryRoundOnCronWake},
		Bifrost: &canonicalBifrost{Enabled: &cfg.BifrostEnabled, Mail: &canonicalBifrostMail{Account: &cfg.BifrostMailAccount, Mailbox: &cfg.BifrostMailMailbox, Capability: &cfg.BifrostMailCapabilityID, Schedule: &cfg.BifrostMailSchedule, MaxMessages: &cfg.BifrostMailMaxMessages}},
		Mail:    &canonicalMail{Enabled: &cfg.MailEnabled, Adapter: &cfg.MailAdapter, Account: &cfg.MailAccount, Inbox: &cfg.MailInbox, BacklogEnabled: &cfg.MailBacklogEnabled, BatchSize: &cfg.MailBatchSize, Direction: &cfg.MailDirection, MarkReadPolicy: &cfg.MailMarkReadPolicy, AttachmentPolicy: &cfg.MailAttachmentPolicy, ActionCapability: &cfg.MailActionCapabilityID},
		Apollo:  &canonicalApolloSettings{MaxRetrievalRounds: &cfg.ApolloMaxRetrievalRounds, MaxSources: &cfg.ApolloMaxSources, MaxContextBytes: &cfg.ApolloMaxContextBytes, MaxReasoningDurationSecs: &cfg.ApolloMaxReasoningDurationSeconds, RequireCitations: &cfg.ApolloRequireCitations, ExternalResearch: &canonicalExternalResearch{Enabled: &cfg.ApolloExternalResearchEnabled}},
	}
}

func marshalCanonicalConfig(cfg config) ([]byte, error) {
	data, err := json.MarshalIndent(canonicalConfigFor(cfg), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func migrateConfig(path string, apply bool) (migrationResult, error) {
	resolution, err := readConfigResolution(path)
	if err != nil {
		return migrationResult{}, err
	}
	result := migrationResult{Path: path, Legacy: resolution.Legacy}
	if !resolution.Legacy {
		result.Noop = true
		return result, nil
	}
	proposed, err := marshalCanonicalConfig(resolution.Config)
	if err != nil {
		return migrationResult{}, fmt.Errorf("render migrated configuration: %w", err)
	}
	result.Proposed = proposed
	if !apply {
		return result, nil
	}
	backupPath, err := backupConfig(path)
	if err != nil {
		return migrationResult{}, fmt.Errorf("backup legacy configuration: %w", err)
	}
	result.BackupPath = backupPath
	if err := writeValidatedConfig(path, proposed); err != nil {
		return migrationResult{}, fmt.Errorf("apply migrated configuration (original preserved at %s): %w", backupPath, err)
	}
	result.Applied = true
	return result, nil
}

func backupConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return backupPath, nil
}

func writeValidatedConfig(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".pantheon-config-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if _, err := readConfigResolution(tmpPath); err != nil {
		return fmt.Errorf("validate temporary configuration: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func renderResolvedConfig(path string, resolution configResolution) string {
	cfg := resolution.Config
	var b strings.Builder
	b.WriteString("Configuration\n\n")
	fmt.Fprintf(&b, "path: %s\n", path)
	if resolution.Legacy {
		b.WriteString("format: legacy flat (readable; migration available)\n")
	} else {
		b.WriteString("format: current schema\n")
	}
	if resolution.Legacy {
		b.WriteString("schema_version: (not declared; legacy)\n")
	} else {
		fmt.Fprintf(&b, "schema_version: %d (%s)\n", resolution.SchemaVersion, resolution.Sources["schema_version"])
	}
	fmt.Fprintf(&b, "runtime.listen: %s (%s)\n", cfg.ListenAddress, resolution.Sources["runtime.listen"])
	fmt.Fprintf(&b, "paths.vault: %s (%s)\n", displayConfigValue(cfg.VaultPath), sourceOrOptional(resolution.Sources, "paths.vault"))
	fmt.Fprintf(&b, "paths.roadmap: %s (%s)\n", displayConfigValue(cfg.RoadmapPath), sourceOrOptional(resolution.Sources, "paths.roadmap"))
	fmt.Fprintf(&b, "paths.intake: %s (%s)\n", displayConfigValue(cfg.IntakePath), sourceOrOptional(resolution.Sources, "paths.intake"))
	fmt.Fprintf(&b, "paths.father_time_chronicles: %s (%s)\n", displayConfigValue(effectiveFatherTimePath(cfg)), sourceOrOptional(resolution.Sources, "paths.father_time_chronicles"))
	fmt.Fprintf(&b, "paths.raw: %s (%s)\n", displayConfigValue(effectiveRawPath(cfg)), sourceOrOptional(resolution.Sources, "paths.raw"))
	fmt.Fprintf(&b, "paths.apollo_reports: %s (%s)\n", displayConfigValue(cfg.ApolloReportsPath), sourceOrOptional(resolution.Sources, "paths.apollo_reports"))
	fmt.Fprintf(&b, "retention.raw_input_days: %d (%s)\n", cfg.RawInputDays, sourceOrOptional(resolution.Sources, "retention.raw_input_days"))
	timezone := cfg.Timezone
	if timezone == "" {
		timezone = time.Local.String()
	}
	fmt.Fprintf(&b, "time.timezone: %s (%s)\n", timezone, sourceOrOptional(resolution.Sources, "time.timezone"))
	fmt.Fprintf(&b, "models.apollo.model: %s (%s)\n", cfg.LocalModel, sourceOrOptional(resolution.Sources, "models.apollo.model"))
	fmt.Fprintf(&b, "models.apollo.endpoint: %s (%s)\n", cfg.OllamaURL, sourceOrOptional(resolution.Sources, "models.apollo.endpoint"))
	fmt.Fprintf(&b, "models.archimedes.model: %s (%s; reserved/inactive)\n", displayConfigValue(cfg.FrontierModel), sourceOrOptional(resolution.Sources, "models.archimedes.model"))
	fmt.Fprintf(&b, "ingestion.whisper: %s (binary=%s, model=%s, language=%s)\n", readinessConfigBool(cfg.WhisperEnabled), displayConfigValue(cfg.WhisperBinary), displayConfigValue(cfg.WhisperModel), cfg.WhisperLanguage)
	fmt.Fprintf(&b, "ingestion.media.converter: %s\n", displayConfigValue(cfg.MediaConverter))
	fmt.Fprintf(&b, "bifrost.enabled: %t\n", cfg.BifrostEnabled)
	fmt.Fprintf(&b, "bifrost.mail: account=%s mailbox=%s capability=%s schedule=%s max_messages=%d\n", cfg.BifrostMailAccount, cfg.BifrostMailMailbox, cfg.BifrostMailCapabilityID, cfg.BifrostMailSchedule, cfg.BifrostMailMaxMessages)
	fmt.Fprintf(&b, "mail: enabled=%t adapter=%s account=%s inbox=%s backlog=%t batch_size=%d direction=%s mark_read=%s attachments=%s\n", cfg.MailEnabled, cfg.MailAdapter, cfg.MailAccount, cfg.MailInbox, cfg.MailBacklogEnabled, cfg.MailBatchSize, cfg.MailDirection, cfg.MailMarkReadPolicy, cfg.MailAttachmentPolicy)
	rumorModel := cfg.RumorWhisperModel
	if rumorModel == "" {
		rumorModel = cfg.WhisperModel
	}
	fmt.Fprintf(&b, "rumor: %s (watch_paths=%s, model=%s, language=%s, retry_limit=%d, stability_seconds=%d, poll_seconds=%d, on_complete=%s)\n", readinessConfigBool(cfg.RumorEnabled), displayConfigValue(strings.Join(cfg.RumorWatchPaths, ",")), displayConfigValue(rumorModel), cfg.RumorLanguage, cfg.RumorRetryLimit, cfg.RumorStabilitySeconds, cfg.RumorPollSeconds, cfg.RumorOnComplete)
	fmt.Fprintf(&b, "mercury.mode: %s\n", displayConfigValue(cfg.MercuryMode))
	fmt.Fprintf(&b, "mercury.agent: %s\n", displayConfigValue(cfg.MercuryAgent))
	fmt.Fprintf(&b, "mercury.binary: %s\n", displayConfigValue(cfg.MercuryBinary))
	fmt.Fprintf(&b, "mercury.deterministic_fallback: %t\n", cfg.MercuryDeterministicFallback)
	fmt.Fprintf(&b, "mercury.round_on_cron_wake: %t\n", cfg.MercuryRoundOnCronWake)
	fmt.Fprintf(&b, "apollo.budgets: rounds=%d sources=%d context_bytes=%d duration_seconds=%d citations=%t external_research=%t\n", cfg.ApolloMaxRetrievalRounds, cfg.ApolloMaxSources, cfg.ApolloMaxContextBytes, cfg.ApolloMaxReasoningDurationSeconds, cfg.ApolloRequireCitations, cfg.ApolloExternalResearchEnabled)
	b.WriteString("active Kernel 18 behavior: local Present normalization and Rumor audio intake; no semantic reasoning or deletion\n")
	b.WriteString("secrets: omitted from resolved output\n")
	return b.String()
}

func readinessConfigBool(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func displayConfigValue(value string) string {
	if value == "" {
		return "(not configured)"
	}
	return value
}

func sourceOrOptional(sources map[string]string, field string) string {
	if source := sources[field]; source != "" {
		return source
	}
	return "optional"
}

func formatMigrationResult(result migrationResult, apply bool) string {
	if result.Noop {
		return fmt.Sprintf("Configuration already uses schema_version %d; no migration required.\n", currentConfigSchemaVersion)
	}
	var b strings.Builder
	if apply {
		fmt.Fprintf(&b, "Migrated legacy configuration: %s\n", result.Path)
		fmt.Fprintf(&b, "backup: %s\n", result.BackupPath)
	} else {
		fmt.Fprintf(&b, "Migration dry run: %s\n", result.Path)
		b.WriteString("no files written\n")
	}
	b.WriteString("proposed configuration:\n")
	b.Write(result.Proposed)
	return b.String()
}

func validateConfigFile(path string) (configResolution, error) {
	resolution, err := readConfigResolution(path)
	if err != nil {
		return configResolution{}, err
	}
	return resolution, nil
}

func configPathFromDefaults() (string, error) {
	opts, err := defaultDaemonOptions()
	if err != nil {
		return "", err
	}
	return opts.ConfigPath, nil
}
