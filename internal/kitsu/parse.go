package kitsu

import (
	"encoding/json"
	"time"
)

type Links struct {
	Links struct {
		Self    string `json:"self"`
		Related string `json:"related"`
	} `json:"links"`
}

type MangaInfo struct {
	Data struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Links struct {
			Self string `json:"self"`
		} `json:"links"`
		Attributes struct {
			CreatedAt           time.Time         `json:"createdAt"`
			UpdatedAt           time.Time         `json:"updatedAt"`
			Slug                string            `json:"slug"`
			Synopsis            string            `json:"synopsis"`
			Description         string            `json:"description"`
			CoverImageTopOffset int               `json:"coverImageTopOffset"`
			Titles              map[string]string `json:"titles"`
			CanonicalTitle      string            `json:"canonicalTitle"`
			AbbreviatedTitles   []string          `json:"abbreviatedTitles"`
			AverageRating       string            `json:"averageRating"`
			UserCount           int               `json:"userCount"`
			FavoritesCount      int               `json:"favoritesCount"`
			StartDate           string            `json:"startDate"`
			EndDate             string            `json:"endDate"`
			NextRelease         any               `json:"nextRelease"`
			PopularityRank      int               `json:"popularityRank"`
			RatingRank          int               `json:"ratingRank"`
			AgeRating           any               `json:"ageRating"`
			AgeRatingGuide      any               `json:"ageRatingGuide"`
			Subtype             string            `json:"subtype"`
			Status              string            `json:"status"`
			Tba                 any               `json:"tba"`
			PosterImage         struct {
				Original string `json:"original"`
			} `json:"posterImage"`
			CoverImage struct {
				Original string `json:"original"`
			} `json:"coverImage"`
			ChapterCount  int    `json:"chapterCount"`
			VolumeCount   int    `json:"volumeCount"`
			Serialization any    `json:"serialization"`
			MangaType     string `json:"mangaType"`
		} `json:"attributes"`
		Relationships struct {
			Genres             Links `json:"genres"`
			Categories         Links `json:"categories"`
			Castings           Links `json:"castings"`
			Installments       Links `json:"installments"`
			Mappings           Links `json:"mappings"`
			Reviews            Links `json:"reviews"`
			MediaRelationships Links `json:"mediaRelationships"`
			Characters         Links `json:"characters"`
			Staff              Links `json:"staff"`
			Productions        Links `json:"productions"`
			Quotes             Links `json:"quotes"`
			Chapters           Links `json:"chapters"`
			MangaCharacters    Links `json:"mangaCharacters"`
			MangaStaff         Links `json:"mangaStaff"`
		} `json:"relationships"`
	} `json:"data"`
}

func ParseMangaInfo(data []byte) MangaInfo {
	var mangaInfo MangaInfo
	if err := json.Unmarshal(data, &mangaInfo); err != nil {
		panic(err)
	}

	return mangaInfo
}

func ParseMangaListSelfLink(data []byte) string {
	type Links struct {
		Self string `json:"self"`
	}
	type Item struct {
		Links Links `json:"links"`
	}
	type Response struct {
		Data []Item `json:"data"`
	}

	var res Response
	if err := json.Unmarshal(data, &res); err != nil {
		panic(err)
	}

	for _, d := range res.Data {
		return d.Links.Self
	}

	return ""
}
