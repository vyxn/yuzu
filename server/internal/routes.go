package internal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/AsaiYusuke/jsonpath/v2"
	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/kitsu"
	"github.com/vyxn/yuzu/internal/lib"

	// "github.com/vyxn/yuzu/internal/pkg/assert"
	"github.com/vyxn/yuzu/internal/pkg/yerr"
	"github.com/vyxn/yuzu/internal/provider"
	"github.com/vyxn/yuzu/internal/provider/comicvine"
	"github.com/vyxn/yuzu/internal/provider/myanimelist"
)

//go:embed static/favicon.ico
var favicon []byte

var providerComicVine provider.ComicInfoProvider
var providerMyAnimeList provider.ComicInfoProvider

func SetupRoutes(e *echo.Echo) {
	providerComicVine = comicvine.NewComicVineProvider(
		os.Getenv("COMICVINE_API_KEY"),
	)
	providerMyAnimeList = myanimelist.NewMyAnimeListProvider(
		os.Getenv("MYANIMELIST_CLIENT_ID"),
	)

	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "image/x-icon", favicon)
	})

	e.GET("/", hello)
	e.GET("/mangaInfo", hMangaInfo)
	e.GET("/mangaChapters", hMangaChapters)
	e.GET("/comicinfo", hComicInfo)
	e.GET("/lib", hLib)
	e.POST("/jsonPath", hJsonPath)
}

// Handler
func hello(c echo.Context) error {
	e := c.QueryParam("e")
	if e != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "error on hello").
			SetInternal(fmt.Errorf("wrapped with something: %w", yerr.WithStackf("test yerr")))
	}
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

	if p, ok := provider.Providers[prov]; ok {
		data, err := p.Run(map[string]string{
			"series":  series,
			"chapter": chapter,
		})
		if err != nil {
			return echo.ErrBadRequest.SetInternal(err)
		}
		return c.Blob(http.StatusOK, p.MimeType(), data)
	}
	// ps := []provider.ComicInfoProvider{}
	// switch prov {
	// case "comicvine":
	// 	ps = append(ps, providerComicVine)
	// case "kitsu":
	// 	ps = append(ps, kitsu.NewKitsuProvider())
	// case "myanimelist":
	// 	ps = append(ps, providerMyAnimeList)
	// default:
	// 	ps = append(ps, providerComicVine, providerMyAnimeList)
	// 	ps = append(ps, kitsu.NewKitsuProvider())
	//
	// }
	// ci, err := provider.MergedComicInfoChapter(
	// 	c.Request().Context(),
	// 	series,
	// 	chapter,
	// 	ps...)
	// if err != nil {
	// 	return echo.ErrNotFound.SetInternal(err)
	// }

	// assert.Assert(ci != nil, "we should have a comicinfochapter here")
	// return c.XML(http.StatusOK, ci)
	return nil
}

func hLib(c echo.Context) error {
	err := lib.Process("testlib")
	if err != nil {
		panic(err)
	}
	return c.String(http.StatusOK, "all good")
}

func hJsonPath(c echo.Context) error {
	p := c.QueryParam("path")
	j, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	var jsonValue any
	err = json.Unmarshal(j, &jsonValue)
	if err != nil {
		return yerr.WithStackf(
			"unmarshalling: %w",
			err,
		)
	}

	slog.Warn("/jsonPath", slog.String("path", p), slog.Any("json", "m"))
	out, err := jsonpath.Retrieve(p, jsonValue)
	if err != nil {
		println(err.Error())
		return yerr.WithStackf("retrieving jsonpath: %s %w", out, err)
	}

	return c.JSON(http.StatusOK, out)
}
