package kitsu

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/log"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func GetURL(url string) []byte {
	logger.Info("→ r",
		slog.String("url", url),
		slog.String("method", http.MethodGet),
	)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ⚠️ unsafe
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Errorf("bad status %s: %s", resp.Status, string(b)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func GetSearchByName(name string) []byte {
	baseURL, err := url.Parse("https://kitsu.io/api/edge/manga")
	if err != nil {
		panic(err)
	}

	params := url.Values{}
	params.Add("filter[text]", name)
	baseURL.RawQuery = params.Encode()

	return GetURL(baseURL.String())
}

func GetMangaAllChaptersInfo(mangaID string) []byte {
	baseURL, err := url.Parse("https://kitsu.io/api/edge/chapters")
	if err != nil {
		panic(err)
	}

	baseURL.Path = path.Join(baseURL.Path, mangaID)
	return GetURL(baseURL.String())
}

func GetMangaChapterInfo(mangaID string, chapter string) []byte {
	baseURL, err := url.Parse("https://kitsu.io/api/edge/manga")
	if err != nil {
		panic(err)
	}

	baseURL.Path = path.Join(baseURL.Path, mangaID, "chapters")
	params := url.Values{}
	params.Add("filter[number]", chapter)
	baseURL.RawQuery = params.Encode()

	return GetURL(baseURL.String())
}
