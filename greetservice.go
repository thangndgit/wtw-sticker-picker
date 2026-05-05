package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	_ "golang.org/x/image/webp"
)

type GreetService struct{}

type AppSettings struct {
	LaunchOnStartup bool   `json:"launchOnStartup"`
	Theme           string `json:"theme"`
}

var popupToastState struct {
	sync.Mutex
	message string
}

type StickerPack struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ThumbData string `json:"thumbDataUrl"`
	Count     int    `json:"count"`
}

func (g *GreetService) GetSettings() (AppSettings, error) {
	settings, err := loadSettings()
	if err != nil {
		return AppSettings{}, err
	}
	startupEnabled, err := isLaunchOnStartupEnabled()
	if err != nil {
		return AppSettings{}, err
	}
	settings.LaunchOnStartup = startupEnabled
	return settings, nil
}

func (g *GreetService) UpdateSettings(next AppSettings) (AppSettings, error) {
	return updateSettings(next)
}

func (g *GreetService) ApplySettingsAndShowPopup(next AppSettings) (AppSettings, error) {
	settings, err := updateSettings(next)
	if err != nil {
		return AppSettings{}, err
	}
	popupToastState.Lock()
	popupToastState.message = "Applied successfully"
	popupToastState.Unlock()

	app := application.Get()
	if app != nil {
		showPopupNearCursor(app, popupWindowName)
	}
	g.HideSettings()
	return settings, nil
}

func (g *GreetService) ConsumePopupToast() string {
	popupToastState.Lock()
	defer popupToastState.Unlock()
	message := popupToastState.message
	popupToastState.message = ""
	return message
}

func updateSettings(next AppSettings) (AppSettings, error) {
	settings, err := loadSettings()
	if err != nil {
		return AppSettings{}, err
	}
	settings.Theme = normalizeTheme(next.Theme)

	if err := setLaunchOnStartup(next.LaunchOnStartup); err != nil {
		return AppSettings{}, err
	}
	startupEnabled, err := isLaunchOnStartupEnabled()
	if err != nil {
		return AppSettings{}, err
	}
	settings.LaunchOnStartup = startupEnabled

	if err := saveSettings(settings); err != nil {
		return AppSettings{}, err
	}
	return settings, nil
}

type StickerItem struct {
	ID      string `json:"id"`
	PackID  string `json:"packId"`
	Name    string `json:"name"`
	DataURL string `json:"dataUrl"`
}

func (g *GreetService) Greet(name string) string {
	return "Hello " + name + "!"
}

func (g *GreetService) HidePopup() {
	window, ok := application.Get().Window.GetByName(popupWindowName)
	if !ok {
		return
	}
	window.Hide()
}

func (g *GreetService) HideSettings() {
	window, ok := application.Get().Window.GetByName(settingsWindowName)
	if !ok {
		return
	}
	window.Hide()
}

func (g *GreetService) PasteSticker(dataURL string) error {
	mimeType, rawBytes, err := decodeStickerDataURL(dataURL)
	if err != nil {
		return err
	}
	if mimeType == "image/webp" && isAnimatedWebP(rawBytes) {
		if err := pasteRawStickerIntoCapturedTarget(".webp", rawBytes); err != nil {
			return err
		}
		g.HidePopup()
		return nil
	}
	if mimeType == "image/gif" {
		if err := pasteRawStickerIntoCapturedTarget(".gif", rawBytes); err != nil {
			return err
		}
		g.HidePopup()
		return nil
	}

	pngBytes, err := stickerRawToPNG(rawBytes)
	if err != nil {
		return err
	}
	g.HidePopup()
	if err := writeStickerImageToClipboard(pngBytes); err != nil {
		return err
	}

	if err := pasteIntoCapturedTarget(); err != nil {
		return err
	}
	return nil
}

func (g *GreetService) ListStickerPacks() ([]StickerPack, error) {
	packEntries, err := fs.ReadDir(stickerAssets, stickerRoot)
	if err != nil {
		return nil, err
	}

	result := make([]StickerPack, 0)
	for _, entry := range packEntries {
		if !entry.IsDir() {
			continue
		}
		packID := entry.Name()
		stickers, err := listStickerFiles(packID)
		if err != nil || len(stickers) == 0 {
			continue
		}
		thumbData, err := readStickerDataURL(packID, stickers[0])
		if err != nil {
			continue
		}
		result = append(result, StickerPack{
			ID:        packID,
			Name:      packID,
			ThumbData: thumbData,
			Count:     len(stickers),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

func (g *GreetService) GetPackStickers(packID string) ([]StickerItem, error) {
	packID = sanitizePackID(packID)
	if packID == "" {
		return nil, fmt.Errorf("invalid pack id")
	}

	stickers, err := listStickerFiles(packID)
	if err != nil {
		return nil, err
	}

	items := make([]StickerItem, 0, len(stickers))
	for _, stickerFile := range stickers {
		dataURL, err := readStickerDataURL(packID, stickerFile)
		if err != nil {
			continue
		}
		items = append(items, StickerItem{
			ID:      path.Join(packID, stickerFile),
			PackID:  packID,
			Name:    stickerFile,
			DataURL: dataURL,
		})
	}

	return items, nil
}

func sanitizePackID(v string) string {
	clean := filepath.Clean(strings.TrimSpace(v))
	if clean == "." || clean == "/" || strings.Contains(clean, "..") || strings.Contains(clean, string(filepath.Separator)) {
		return ""
	}
	return clean
}

func listStickerFiles(packID string) ([]string, error) {
	packPath := path.Join(stickerRoot, packID)
	entries, err := fs.ReadDir(stickerAssets, packPath)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(path.Ext(entry.Name()))
		if ext == ".webp" || ext == ".gif" || ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
			files = append(files, entry.Name())
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return compareStickerFilename(files[i], files[j])
	})
	return files, nil
}

func compareStickerFilename(a, b string) bool {
	aName := strings.TrimSuffix(a, path.Ext(a))
	bName := strings.TrimSuffix(b, path.Ext(b))
	aNum, aErr := strconv.Atoi(aName)
	bNum, bErr := strconv.Atoi(bName)
	if aErr == nil && bErr == nil {
		return aNum < bNum
	}
	return strings.ToLower(a) < strings.ToLower(b)
}

func readStickerDataURL(packID, fileName string) (string, error) {
	fullPath := path.Join(stickerRoot, packID, fileName)
	data, err := stickerAssets.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	mimeType := detectStickerMIME(fileName)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)), nil
}

func detectStickerMIME(fileName string) string {
	switch strings.ToLower(path.Ext(fileName)) {
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "image/webp"
	}
}

func decodeStickerDataURL(dataURL string) (string, []byte, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return "", nil, fmt.Errorf("empty sticker data")
	}
	comma := strings.Index(dataURL, ",")
	if comma <= 0 {
		return "", nil, fmt.Errorf("invalid data url")
	}
	header := strings.ToLower(dataURL[:comma])
	if !strings.Contains(header, ";base64") {
		return "", nil, fmt.Errorf("unsupported data url encoding")
	}
	rawBase64 := dataURL[comma+1:]
	raw, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		return "", nil, fmt.Errorf("decode sticker base64: %w", err)
	}
	mimeType := "application/octet-stream"
	if strings.HasPrefix(header, "data:") {
		if semi := strings.Index(header[5:], ";"); semi >= 0 {
			mimeType = header[5 : 5+semi]
		}
	}
	return mimeType, raw, nil
}

func stickerRawToPNG(raw []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode sticker image: %w", err)
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return out.Bytes(), nil
}

func isAnimatedWebP(raw []byte) bool {
	if len(raw) < 16 {
		return false
	}
	if string(raw[:4]) != "RIFF" || string(raw[8:12]) != "WEBP" {
		return false
	}
	// Animated WebP always contains an ANIM or ANMF chunk.
	return bytes.Contains(raw, []byte("ANIM")) || bytes.Contains(raw, []byte("ANMF"))
}

const (
	settingsDirName  = "wtw-sticker-picker"
	settingsFileName = "settings.json"
	themeDark        = "dark"
	themeLight       = "light"
)

func defaultSettings() AppSettings {
	return AppSettings{
		LaunchOnStartup: false,
		Theme:           themeDark,
	}
}

func normalizeTheme(theme string) string {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case themeLight:
		return themeLight
	default:
		return themeDark
	}
}

func settingsFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, settingsDirName, settingsFileName), nil
}

func loadSettings() (AppSettings, error) {
	defaults := defaultSettings()
	settingsPath, err := settingsFilePath()
	if err != nil {
		return defaults, err
	}
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read settings file: %w", err)
	}
	var parsed AppSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return defaults, fmt.Errorf("parse settings file: %w", err)
	}
	parsed.Theme = normalizeTheme(parsed.Theme)
	return parsed, nil
}

func saveSettings(settings AppSettings) error {
	settings.Theme = normalizeTheme(settings.Theme)
	settingsPath, err := settingsFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, payload, 0o644); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	return nil
}
