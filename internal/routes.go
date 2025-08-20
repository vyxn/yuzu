package internal

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/kitsu"
	"github.com/vyxn/yuzu/internal/lib"
	"github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/provider/comicvine"
	"github.com/vyxn/yuzu/internal/provider/myanimelist"
)

func SetupRoutes(e *echo.Echo) {
	e.GET("/", hello)
	e.GET("/mangaInfo", hMangaInfo)
	e.GET("/mangaChapters", hMangaChapters)
	e.GET("/comicinfo", hComicInfo)
	e.GET("/lib", hLib)
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func hMangaInfo(c echo.Context) error {
	name := c.QueryParam("name")
	mangaSearchRes := kitsu.GetSearchByName(name)
	mangaURL := kitsu.ParseMangaListSelfLink(mangaSearchRes)

	mangaInfoRes := kitsu.GetURL(mangaURL)
	mangaInfo := kitsu.ParseMangaInfo(mangaInfoRes)

	return c.String(http.StatusOK, fmt.Sprintf("%+v", mangaInfo))
}

func hMangaChapters(c echo.Context) error {
	name := c.QueryParam("name")
	chapter := c.QueryParam("chapter")
	//volume := c.QueryParam("volume")

	mangaSearchRes := kitsu.GetSearchByName(name)
	mangaURL := kitsu.ParseMangaListSelfLink(mangaSearchRes)

	mangaInfoRes := kitsu.GetURL(mangaURL)
	mangaInfo := kitsu.ParseMangaInfo(mangaInfoRes)

	var info []byte
	if chapter != "" {
		info = kitsu.GetMangaChapterInfo(mangaInfo.Data.ID, chapter)
	} else {
		info = kitsu.GetURL(mangaInfo.Data.Relationships.Chapters.Links.Self)
	}

	return c.String(http.StatusOK, fmt.Sprintf("%+v", string(info)))
}

func hComicInfo(c echo.Context) error {
	series := c.QueryParam("s")
	chapter := c.QueryParam("c")
	prov := c.QueryParam("p")

	ps := []provider.ComicInfoProvider{}
	switch prov {
	case "comicvine":
		ps = append(ps, comicvine.NewComicVineProvider())
	case "kitsu":
		ps = append(ps, kitsu.NewKitsuProvider())
	case "myanimelist":
		ps = append(ps, myanimelist.NewMyAnimeListProvider())
	default:
		ps = append(ps, myanimelist.NewMyAnimeListProvider())
		ps = append(ps, kitsu.NewKitsuProvider())

	}
	ci := provider.MergedComicInfoChapter(series, chapter, ps...)

	assert.Assert(ci != nil, "we should have a comicinfochapter here")

	return c.XML(http.StatusOK, ci)
}

func hLib(c echo.Context) error {
	err := lib.Process("testlib")
	if err != nil {
		panic(err)
	}
	return c.String(http.StatusOK, "all good")
}
