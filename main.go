package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/kitsu"
)

func main() {
	e := echo.New()
	e.Use(logger)

	// Routes
	e.GET("/", hello)
	e.GET("/mangaInfo", hMangaInfo)
	e.GET("/mangaChapters", hMangaChapters)
	e.GET("/comicinfo", hComicInfo)

	// Start server
	if err := e.Start(":8080"); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
}

// Middleware
func logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		fmt.Printf("Called: %s, With: [%s]\n", req.URL.Path, req.URL.RawQuery)
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
	mangaUrl := kitsu.ParseMangaListSelfLink(mangaSearchRes)

	mangaInfoRes := kitsu.GetUrl(mangaUrl)
	mangaInfo := kitsu.ParseMangaInfo(mangaInfoRes)

	return c.String(http.StatusOK, fmt.Sprintf("%+v", mangaInfo))
}

func hMangaChapters(c echo.Context) error {
	name := c.QueryParam("name")
	chapter := c.QueryParam("chapter")
	//volume := c.QueryParam("volume")

	mangaSearchRes := kitsu.GetSearchByName(name)
	mangaUrl := kitsu.ParseMangaListSelfLink(mangaSearchRes)

	mangaInfoRes := kitsu.GetUrl(mangaUrl)
	mangaInfo := kitsu.ParseMangaInfo(mangaInfoRes)

	var info []byte
	if chapter != "" {
		info = kitsu.GetMangaChapterInfo(mangaInfo.Data.ID, chapter)
	} else {
		info = kitsu.GetUrl(mangaInfo.Data.Relationships.Chapters.Links.Self)
	}

	return c.String(http.StatusOK, fmt.Sprintf("%+v", string(info)))
}

func hComicInfo(c echo.Context) error {
	name := c.QueryParam("name")
	chapter := c.QueryParam("chapter")

	mangaSearchRes := kitsu.GetSearchByName(name)
	mangaUrl := kitsu.ParseMangaListSelfLink(mangaSearchRes)

	mangaInfoRes := kitsu.GetUrl(mangaUrl)
	mangaInfo := kitsu.ParseMangaInfo(mangaInfoRes)

	info := kitsu.GetMangaChapterInfo(mangaInfo.Data.ID, chapter)
	chapterInfo := kitsu.ParseMangaChapter(info)

	ci := kitsu.ParseToComicInfoChapter(mangaInfo, chapterInfo)

	return c.XML(http.StatusOK, ci)
}
