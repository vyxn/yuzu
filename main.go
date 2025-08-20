package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/kitsu"
	"github.com/vyxn/yuzu/internal/lib"
	"github.com/vyxn/yuzu/internal/pkg/log"
	"github.com/vyxn/yuzu/internal/provider/myanimelist"
)

var logger *slog.Logger

func init() {
	logger = log.NewLogger()
}

func main() {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(mLogger)

	// Routes
	e.GET("/", hello)
	e.GET("/mangaInfo", hMangaInfo)
	e.GET("/mangaChapters", hMangaChapters)
	e.GET("/comicinfo", hComicInfo)
	e.GET("/lib", hLib)

	// Start server
	port := ":8080"
	logger.Info("http server started", slog.String("port", port))
	if err := e.Start(port); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		logger.Error("failed to start server", "error", err)
	}
}

// Middleware
func mLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		logger.Info("← h",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.String("query", req.URL.RawQuery),
		)
		return next(c)
	}
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

	// p := kitsu.NewKitsuProvider()
	p := myanimelist.NewMyAnimeListProvider()
	ci := p.ProvideChapter(series, chapter)


	return c.XML(http.StatusOK, ci)
}

func hLib(c echo.Context) error {
	err := lib.Process("testlib")
	if err != nil {
		panic(err)
	}
	return c.String(http.StatusOK, "all good")
}
