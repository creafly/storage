package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Messages struct {
	Errors  ErrorMessages   `json:"errors"`
	Success SuccessMessages `json:"success"`
}

type ErrorMessages struct {
	Unauthorized   string `json:"unauthorized"`
	Forbidden      string `json:"forbidden"`
	TenantRequired string `json:"tenantRequired"`
	FileRequired   string `json:"fileRequired"`
	ReadFailed     string `json:"readFailed"`
	UploadFailed   string `json:"uploadFailed"`
	FetchFailed    string `json:"fetchFailed"`
	DeleteFailed   string `json:"deleteFailed"`
	InvalidID      string `json:"invalidId"`
	NotFound       string `json:"notFound"`
	FileTooLarge   string `json:"fileTooLarge"`
	InvalidType    string `json:"invalidType"`
}

type SuccessMessages struct {
	FileUploaded string `json:"fileUploaded"`
	FileDeleted  string `json:"fileDeleted"`
}

var (
	messagesCache = make(map[string]*Messages)
	cacheMutex    sync.RWMutex
	defaultLocale = "en-US"
)

func PreloadLocales() {
	locales := []string{"en-US", "ru-RU"}
	for _, locale := range locales {
		_, _ = loadMessages(locale)
	}
}

func GetMessages(locale string) *Messages {
	cacheMutex.RLock()
	if msgs, ok := messagesCache[locale]; ok {
		cacheMutex.RUnlock()
		return msgs
	}
	cacheMutex.RUnlock()

	msgs, err := loadMessages(locale)
	if err != nil {
		msgs, _ = loadMessages(defaultLocale)
	}
	return msgs
}

func loadMessages(locale string) (*Messages, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if msgs, ok := messagesCache[locale]; ok {
		return msgs, nil
	}

	paths := []string{
		filepath.Join("resources", "locale", locale+".json"),
		filepath.Join("resources/locale", locale+".json"),
	}

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}

	if err != nil {
		return getDefaultMessages(), err
	}

	var msgs Messages
	if err := json.Unmarshal(data, &msgs); err != nil {
		return getDefaultMessages(), err
	}

	messagesCache[locale] = &msgs
	return &msgs, nil
}

func getDefaultMessages() *Messages {
	return &Messages{
		Errors: ErrorMessages{
			Unauthorized:   "Unauthorized",
			Forbidden:      "Access denied",
			TenantRequired: "Tenant ID is required",
			FileRequired:   "File is required",
			ReadFailed:     "Failed to read file",
			UploadFailed:   "Failed to upload file",
			FetchFailed:    "Failed to fetch data",
			DeleteFailed:   "Failed to delete file",
			InvalidID:      "Invalid ID format",
			NotFound:       "Not found",
			FileTooLarge:   "File is too large",
			InvalidType:    "Invalid file type",
		},
		Success: SuccessMessages{
			FileUploaded: "File uploaded successfully",
			FileDeleted:  "File deleted successfully",
		},
	}
}

func ParseLocale(acceptLanguage string) string {
	if acceptLanguage == "" {
		return defaultLocale
	}

	parts := strings.Split(acceptLanguage, ",")
	if len(parts) == 0 {
		return defaultLocale
	}

	lang := strings.TrimSpace(strings.Split(parts[0], ";")[0])

	if strings.HasPrefix(lang, "ru") {
		return "ru-RU"
	}
	return "en-US"
}
