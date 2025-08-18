package kitsu

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

func GetUrl(url string) []byte {
	client := &http.Client{Timeout: 10 * time.Second}

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
	baseUrl, err := url.Parse("https://kitsu.io/api/edge/manga")
	if err != nil {
		panic(err)
	}

	params := url.Values{}
	params.Add("filter[text]", name)
	baseUrl.RawQuery = params.Encode()

	return GetUrl(baseUrl.String())
}

func GetMangaAllChaptersInfo(mangaId string) []byte {
	baseUrl, err := url.Parse("https://kitsu.io/api/edge/chapters")
	if err != nil {
		panic(err)
	}

	baseUrl.Path = path.Join(baseUrl.Path, mangaId)
	return GetUrl(baseUrl.String())
}

func GetMangaChapterInfo(mangaId string, chapter string) []byte {
	baseUrl, err := url.Parse("https://kitsu.io/api/edge/manga")
	if err != nil {
		panic(err)
	}

	baseUrl.Path = path.Join(baseUrl.Path, mangaId, "chapters")
	params := url.Values{}
	params.Add("filter[number]", chapter)
	baseUrl.RawQuery = params.Encode()

	return GetUrl(baseUrl.String())
}
