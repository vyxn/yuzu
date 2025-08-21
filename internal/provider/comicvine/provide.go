package comicvine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/vyxn/yuzu/internal/pkg/req"
	"github.com/vyxn/yuzu/internal/standard"
)

const baseURL = "http://comicvine.gamespot.com/api/volumes"

var apiKey = os.Getenv("COMICVINE_API_KEY")

type ComicVineComicInfoProvider struct {
	// cache map[string]MangaInfo
}

func NewComicVineProvider() *ComicVineComicInfoProvider {
	return &ComicVineComicInfoProvider{}
}

func (p *ComicVineComicInfoProvider) ProvideChapter(
	series, chapter string,
) *standard.ComicInfoChapter {
	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	q := u.Query()
	q.Add("api_key", apiKey)
	q.Add("format", "json")
	q.Add("filter", "name:"+series)
	u.RawQuery = q.Encode()

	data, _ := req.Get(context.Background(), u.String(), nil)
	// slog.Info("resp", slog.String("data", string(data)))

	type List struct {
		Error                string `json:"error"`
		Limit                int    `json:"limit"`
		Offset               int    `json:"offset"`
		NumberOfPageResults  int    `json:"number_of_page_results"`
		NumberOfTotalResults int    `json:"number_of_total_results"`
		StatusCode           int    `json:"status_code"`
		Results              []struct {
			Aliases         string `json:"aliases"`
			APIDetailURL    string `json:"api_detail_url"`
			CountOfIssues   int    `json:"count_of_issues"`
			DateAdded       string `json:"date_added"`
			DateLastUpdated string `json:"date_last_updated"`
			Deck            any    `json:"deck"`
			Description     string `json:"description"`
			FirstIssue      struct {
				APIDetailURL string `json:"api_detail_url"`
				ID           int    `json:"id"`
				Name         string `json:"name"`
				IssueNumber  string `json:"issue_number"`
			} `json:"first_issue"`
			ID    int `json:"id"`
			Image struct {
				IconURL        string `json:"icon_url"`
				MediumURL      string `json:"medium_url"`
				ScreenURL      string `json:"screen_url"`
				ScreenLargeURL string `json:"screen_large_url"`
				SmallURL       string `json:"small_url"`
				SuperURL       string `json:"super_url"`
				ThumbURL       string `json:"thumb_url"`
				TinyURL        string `json:"tiny_url"`
				OriginalURL    string `json:"original_url"`
				ImageTags      string `json:"image_tags"`
			} `json:"image"`
			LastIssue struct {
				APIDetailURL string `json:"api_detail_url"`
				ID           int    `json:"id"`
				Name         string `json:"name"`
				IssueNumber  string `json:"issue_number"`
			} `json:"last_issue"`
			Name      string `json:"name"`
			Publisher struct {
				APIDetailURL string `json:"api_detail_url"`
				ID           int    `json:"id"`
				Name         string `json:"name"`
			} `json:"publisher"`
			SiteDetailURL string `json:"site_detail_url"`
			StartYear     string `json:"start_year"`
		} `json:"results"`
		Version string `json:"version"`
	}
	var list List
	if err := json.Unmarshal(data, &list); err != nil {
		err = fmt.Errorf("error Unmarshaling json response: %w", err)
		slog.Error(
			"error",
			slog.Any("error", err),
			slog.String("data", string(data)),
		)
		return nil
	}

	if list.StatusCode != 1 {
		err = fmt.Errorf("error on comicvine response: %s", list.Error)
		slog.Error("error", slog.Any("error", err))
		return nil
	}

	for n, e := range list.Results {
		if n == 0 {
			continue
		}
		u, err := url.Parse(e.APIDetailURL)
		if err != nil {
			err = fmt.Errorf("error building url <%s>: %w", e.APIDetailURL, err)
			slog.Error("error", slog.Any("error", err))
			return nil
		}

		q := u.Query()
		q.Add("api_key", apiKey)
		q.Add("format", "json")
		u.RawQuery = q.Encode()

		data, _ := req.Get(context.Background(), u.String(), nil)

		type MangaInfo struct {
			Error                string `json:"error"`
			Limit                int    `json:"limit"`
			Offset               int    `json:"offset"`
			NumberOfPageResults  int    `json:"number_of_page_results"`
			NumberOfTotalResults int    `json:"number_of_total_results"`
			StatusCode           int    `json:"status_code"`
			Results              struct {
				Aliases      any    `json:"aliases"`
				APIDetailURL string `json:"api_detail_url"`
				Characters   []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					Count         string `json:"count"`
				} `json:"characters"`
				Concepts []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					Count         string `json:"count"`
				} `json:"concepts"`
				CountOfIssues   int    `json:"count_of_issues"`
				DateAdded       string `json:"date_added"`
				DateLastUpdated string `json:"date_last_updated"`
				Deck            any    `json:"deck"`
				Description     string `json:"description"`
				FirstIssue      struct {
					APIDetailURL string `json:"api_detail_url"`
					ID           int    `json:"id"`
					Name         string `json:"name"`
					IssueNumber  string `json:"issue_number"`
				} `json:"first_issue"`
				ID    int `json:"id"`
				Image struct {
					IconURL        string `json:"icon_url"`
					MediumURL      string `json:"medium_url"`
					ScreenURL      string `json:"screen_url"`
					ScreenLargeURL string `json:"screen_large_url"`
					SmallURL       string `json:"small_url"`
					SuperURL       string `json:"super_url"`
					ThumbURL       string `json:"thumb_url"`
					TinyURL        string `json:"tiny_url"`
					OriginalURL    string `json:"original_url"`
					ImageTags      string `json:"image_tags"`
				} `json:"image"`
				Issues []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					IssueNumber   string `json:"issue_number"`
				} `json:"issues"`
				LastIssue struct {
					APIDetailURL string `json:"api_detail_url"`
					ID           int    `json:"id"`
					Name         string `json:"name"`
					IssueNumber  string `json:"issue_number"`
				} `json:"last_issue"`
				Locations []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					Count         string `json:"count"`
				} `json:"locations"`
				Name    string `json:"name"`
				Objects []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					Count         string `json:"count"`
				} `json:"objects"`
				People []struct {
					APIDetailURL  string `json:"api_detail_url"`
					ID            int    `json:"id"`
					Name          string `json:"name"`
					SiteDetailURL string `json:"site_detail_url"`
					Count         string `json:"count"`
				} `json:"people"`
				Publisher struct {
					APIDetailURL string `json:"api_detail_url"`
					ID           int    `json:"id"`
					Name         string `json:"name"`
				} `json:"publisher"`
				SiteDetailURL string `json:"site_detail_url"`
				StartYear     string `json:"start_year"`
			} `json:"results"`
			Version string `json:"version"`
		}
		var manga MangaInfo
		if err := json.Unmarshal(data, &manga); err != nil {
			err = fmt.Errorf("error Unmarshaling json response: %w", err)
			slog.Error(
				"error",
				slog.Any("error", err),
				slog.String("data", string(data)),
			)
			return nil
		}

		ci := &standard.ComicInfoChapter{
			Title:  "",
			Series: manga.Results.Name,
		}

		for _, issue := range manga.Results.Issues {
			if issue.IssueNumber != chapter {
				continue
			}

			u, err := url.Parse(issue.APIDetailURL)
			if err != nil {
				// TODO: handle this
				panic(err)
			}

			u.RawQuery = q.Encode()
			data, err := req.Get(context.Background(), u.String(), nil)
			if err != nil {
				// TODO: handle this
				panic(err)
			}

			type Issue struct {
				Error                string `json:"error"`
				Limit                int    `json:"limit"`
				Offset               int    `json:"offset"`
				NumberOfPageResults  int    `json:"number_of_page_results"`
				NumberOfTotalResults int    `json:"number_of_total_results"`
				StatusCode           int    `json:"status_code"`
				Results              struct {
					Aliases          any    `json:"aliases"`
					APIDetailURL     string `json:"api_detail_url"`
					AssociatedImages []any  `json:"associated_images"`
					CharacterCredits []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"character_credits"`
					CharacterDiedIn []any `json:"character_died_in"`
					ConceptCredits  []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"concept_credits"`
					CoverDate                 string `json:"cover_date"`
					DateAdded                 string `json:"date_added"`
					DateLastUpdated           string `json:"date_last_updated"`
					Deck                      any    `json:"deck"`
					Description               string `json:"description"`
					FirstAppearanceCharacters any    `json:"first_appearance_characters"`
					FirstAppearanceConcepts   any    `json:"first_appearance_concepts"`
					FirstAppearanceLocations  any    `json:"first_appearance_locations"`
					FirstAppearanceObjects    any    `json:"first_appearance_objects"`
					FirstAppearanceStoryarcs  any    `json:"first_appearance_storyarcs"`
					FirstAppearanceTeams      any    `json:"first_appearance_teams"`
					HasStaffReview            bool   `json:"has_staff_review"`
					ID                        int    `json:"id"`
					Image                     struct {
						IconURL        string `json:"icon_url"`
						MediumURL      string `json:"medium_url"`
						ScreenURL      string `json:"screen_url"`
						ScreenLargeURL string `json:"screen_large_url"`
						SmallURL       string `json:"small_url"`
						SuperURL       string `json:"super_url"`
						ThumbURL       string `json:"thumb_url"`
						TinyURL        string `json:"tiny_url"`
						OriginalURL    string `json:"original_url"`
						ImageTags      string `json:"image_tags"`
					} `json:"image"`
					IssueNumber     string `json:"issue_number"`
					LocationCredits []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"location_credits"`
					Name          string `json:"name"`
					ObjectCredits []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"object_credits"`
					PersonCredits []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
						Role          string `json:"role"`
					} `json:"person_credits"`
					SiteDetailURL   string `json:"site_detail_url"`
					StoreDate       string `json:"store_date"`
					StoryArcCredits []any  `json:"story_arc_credits"`
					TeamCredits     []struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"team_credits"`
					TeamDisbandedIn []any `json:"team_disbanded_in"`
					Volume          struct {
						APIDetailURL  string `json:"api_detail_url"`
						ID            int    `json:"id"`
						Name          string `json:"name"`
						SiteDetailURL string `json:"site_detail_url"`
					} `json:"volume"`
				} `json:"results"`
				Version string `json:"version"`
			}
			var issue Issue
			if err := json.Unmarshal(data, &issue); err != nil {
				err = fmt.Errorf("error Unmarshaling json response: %w", err)
				slog.Error(
					"error",
					slog.Any("error", err),
					slog.String("data", string(data)),
				)
				return nil
			}

			ci.Title = issue.Results.Name
			ci.Number = issue.Results.IssueNumber
			ci.Summary = issue.Results.Description

			break
		}

		return ci
	}

	return nil
}
