package internal

import (
	_ "embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"strings"

	// "log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/vyxn/yuzu/internal/kitsu"
	"github.com/vyxn/yuzu/internal/lib"
	"github.com/vyxn/yuzu/internal/pkg/assert"
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

	// More UI focused
	e.GET("/libraries/files", hLibraryFileTree)
	e.GET("/libraries/files/:path", hLibraryFileTree)
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

	ps := []provider.ComicInfoProvider{}
	switch prov {
	case "comicvine":
		ps = append(ps, providerComicVine)
	case "kitsu":
		ps = append(ps, kitsu.NewKitsuProvider())
	case "myanimelist":
		ps = append(ps, providerMyAnimeList)
	default:
		ps = append(ps, providerComicVine, providerMyAnimeList)
		ps = append(ps, kitsu.NewKitsuProvider())

	}
	ci, err := provider.MergedComicInfoChapter(
		c.Request().Context(),
		series,
		chapter,
		ps...)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

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

type FSEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Date string `json:"date"`
	Size *int64 `json:"size,omitempty"`
}

// TODO: handle error with more sense than this
func hLibraryFileTree(c echo.Context) error {
	var err error
	p := c.Param("path")
	p, err = url.QueryUnescape(p)
	if err != nil {
		return yerr.WithStackf("unescaping path <%s>: %w", p, err)
	}

	root := "/testlib"
	if p != "" && p != "/" {
		root = p
	}
	slog.Info("root", slog.String("p", p), slog.String("root", root))

	q := c.QueryParam("q")
	var re *regexp.Regexp
	if strings.Contains(q, "/") {
		pattern := strings.Trim(q, "/")
		slog.Info(
			"regex pattern",
			slog.String("q", q),
			slog.String("pattern", pattern),
		)
		re, err = regexp.Compile(pattern)
		if err != nil {
			return yerr.WithStackf("compiling search regexp <%s>: %w", q, err)
		}
	}

	rootNoPrefix := strings.TrimPrefix(root, "/")

	fsEntries := []FSEntry{}
	err = filepath.WalkDir(
		strings.TrimPrefix(root, "/"),
		func(path string, d fs.DirEntry, err error) error {
			if p != "" && p != "/" && path == rootNoPrefix {
				return nil
			}

			if q != "" && re != nil {
				if !re.MatchString(path) {
					return nil
				}
			} else if q != "" {
				if !strings.Contains(path, q) {
					return nil
				}
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			// slog.Info(
			// 	"fs entry",
			// 	slog.String("path", path),
			// 	slog.String("name", d.Name()),
			// 	slog.String("type", d.Type().String()),
			// )

			t := "file"
			sz := info.Size()
			s := &sz
			if d.IsDir() {
				t = "folder"
				s = nil
			}

			fsEntries = append(fsEntries, FSEntry{
				ID:   "/" + path,
				Type: t,
				Date: info.ModTime().UTC().Format("2006-01-02 15:04:05"),
				Size: s,
			})
			return nil
		},
	)
	if err != nil {
		return echo.ErrNotFound.SetInternal(err)
	}

	return c.JSON(http.StatusOK, fsEntries)
}
