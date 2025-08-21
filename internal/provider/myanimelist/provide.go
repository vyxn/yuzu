// Package myanimelist implements the provider interface for myanimelist metadata
package myanimelist

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/vyxn/yuzu/internal/pkg/log"
	"github.com/vyxn/yuzu/internal/standard"
)

const baseURL = "https://api.myanimelist.net/v2/manga"

var clientID = os.Getenv("MYANIMELIST_CLIENT_ID")

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

type MyAnimeListComicInfoProvider struct {
	// cache map[string]MangaInfo
}

func NewMyAnimeListProvider() *MyAnimeListComicInfoProvider {
	return &MyAnimeListComicInfoProvider{}
}

func (p *MyAnimeListComicInfoProvider) ProvideChapter(
	series, chapter string,
) *standard.ComicInfoChapter {
	// mangaInfo, ok := p.cache[series]
	// if !ok {
	// 	mangaInfo = ParseMangaInfo(mangaInfoRes)
	// 	p.cache[series] = mangaInfo
	// }
	res, err := getComicInfo(series)
	if err != nil {
		logger.Error("error", slog.String("error", err.Error()))
		return nil
	}

	return res
}

func getURL(url string) ([]byte, error) {
	logger.Info("→ r",
		slog.String("url", url),
		slog.String("method", http.MethodGet),
	)

	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-MAL-CLIENT-ID", clientID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, err := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status %s: %s: %w", resp.Status, string(b), err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

func getBestMatchID(series string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("q", series)
	u.RawQuery = params.Encode()

	data, err := getURL(u.String())
	if err != nil {
		return "", err
	}

	type ListResult struct {
		Data []struct {
			Node struct {
				ID          int    `json:"id"`
				Title       string `json:"title"`
				MainPicture struct {
					Medium string `json:"medium"`
					Large  string `json:"large"`
				} `json:"main_picture"`
			} `json:"node"`
		} `json:"data"`
		Paging struct {
			Next string `json:"next"`
		} `json:"paging"`
	}
	var res ListResult
	if err := json.Unmarshal(data, &res); err != nil {
		return "", err
	}

	for _, m := range res.Data {
		return strconv.Itoa(m.Node.ID), nil
	}

	return "", nil
}

func getComicInfo(series string) (*standard.ComicInfoChapter, error) {
	id, err := getBestMatchID(series)
	if err != nil {
		return nil, err
	}

	if id == "" {
		return nil, fmt.Errorf("error manga id not found for %s", series)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, id)

	params := u.Query()
	params.Set(
		"fields",
		"id,title,main_picture,alternative_titles,start_date,end_date,synopsis,mean,rank,popularity,num_list_users,num_scoring_users,nsfw,created_at,updated_at,media_type,status,genres,my_list_status,num_volumes,num_chapters,authors{first_name,last_name},background,serialization{name}",
	)
	u.RawQuery = params.Encode()

	data, err := getURL(u.String())
	if err != nil {
		return nil, err
	}

	type MangaInfo struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		MainPicture struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
		} `json:"main_picture"`
		AlternativeTitles struct {
			Synonyms []string `json:"synonyms"`
			En       string   `json:"en"`
			Ja       string   `json:"ja"`
		} `json:"alternative_titles"`
		StartDate       string    `json:"start_date"`
		Synopsis        string    `json:"synopsis"`
		Mean            float64   `json:"mean"`
		Rank            int       `json:"rank"`
		Popularity      int       `json:"popularity"`
		NumListUsers    int       `json:"num_list_users"`
		NumScoringUsers int       `json:"num_scoring_users"`
		Nsfw            string    `json:"nsfw"`
		CreatedAt       time.Time `json:"created_at"`
		UpdatedAt       time.Time `json:"updated_at"`
		MediaType       string    `json:"media_type"`
		Status          string    `json:"status"`
		Genres          []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
		NumVolumes  int `json:"num_volumes"`
		NumChapters int `json:"num_chapters"`
		Authors     []struct {
			Node struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"node"`
			Role string `json:"role"`
		} `json:"authors"`
		// Pictures []struct {
		// 	Medium string `json:"medium"`
		// 	Large  string `json:"large"`
		// } `json:"pictures"`
		Background    string `json:"background"`
		Serialization []struct {
			Node struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"node"`
		} `json:"serialization"`
	}
	var res MangaInfo
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	return &standard.ComicInfoChapter{
		Series:  res.Title,
		Summary: res.Synopsis,
		Notes:   "Autogenerated with yuzu 🍋",
		Manga:   "YesAndRightToLeft",
	}, nil
}
