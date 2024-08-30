package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/open2b/scriggo/native"
	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/builtinserver"
	"github.com/restream/reindexer/v3/bindings/builtinserver/config"
	"log"
	"reflect"
	"regoxer/engine"
	"strconv"
	"time"

	// for sitemap
	"github.com/ikeikeikeike/go-sitemap-generator/v2/stm"

	// for configs
	"github.com/laurent22/toml-go"
)

func main() {
	var parser toml.Parser
	tomlConfig := parser.ParseFile("config.toml")
	port := tomlConfig.GetInt("server.port", 3000)

	server := NewServer(tomlConfig)
	server.app.Get("/static*", static.New("./static"))
	err := server.Start("localhost:" + strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Failed to run, got error %v", err)
	}
}

func (s *Server) Start(bind string) error {
	return s.app.Listen(bind)
}

func NewServer(tomlConfig toml.Document) *Server {
	vEngine := engine.New("templates", ".html")
	s := &Server{
		app: fiber.New(fiber.Config{
			Views:     vEngine,
			BodyLimit: 1024 * 1024 * 1024, // this is the default limit of 1024MB
		}),
		db:       loadReindexer(),
		config:   tomlConfig,
		SiteName: tomlConfig.GetString("site.name"),
	}

	s.app.Get("/", s.indexPage)
	s.app.Get("/admin", s.admin)
	s.app.Get("/sitemap.xml", s.sitemap)

	return s
}

func (s *Server) admin(ctx fiber.Ctx) error {
	return ctx.Redirect().To("/static/admin/index.html")
}

type Server struct {
	app      *fiber.App
	db       *reindexer.Reindexer
	config   toml.Document
	SiteName string
}

func (s *Server) NotImplemented(ctx *fiber.Ctx) error {
	return nil
}

const NsName = "feed"

func loadReindexer() *reindexer.Reindexer {
	// Create server config with custom storage path
	cfg := config.DefaultServerConfig()
	cfg.Storage.Path = "db"
	// Initialize reindexer binding in builtinserver mode
	db := reindexer.NewReindex("builtinserver://rdx_test_db", reindexer.WithServerConfig(time.Second*100, cfg))
	// Check if DB was initialized correctly
	if db.Status().Err != nil {
		panic(db.Status().Err)
	}
	//defer db.Close()

	// Create or open namespace with indexes and schema from struct TestItem
	err := db.OpenNamespace(NsName, reindexer.DefaultNamespaceOptions(), FeedLine{})
	if err != nil {
		panic(err)
	}
	print("Loaded!")

	return db
}

type FeedLine struct {
	ID       int    `reindex:"id,,pk"`
	Title    string `reindex:"title,text"`
	Story    string `reindex:"story,text"`
	Category string `reindex:"category,tree"`
	Tags     string `reindex:"tags,tree"`
	Author   string `reindex:"author,tree"`

	Date    int64  `reindex:"date,tree"`
	PartsId string `reindex:"parts,tree"`
	Url     string `reindex:"url,tree"`
}

// todo add pagination and sitemap for firemap
func (s *Server) sitemap(ctx fiber.Ctx) error {
	iterator := s.db.Query(NsName).
		Select("title", "id").
		Limit(50_000).
		Exec()

	sm := stm.NewSitemap(10)

	sm.Create()
	sm.SetDefaultHost("https://" + s.config.GetString("server.host"))

	for iterator.Next() {
		line := iterator.Object().(*FeedLine)
		sm.Add(stm.URL{
			{"loc", "/story/" + line.Url},
		})
	}

	_, err := ctx.Write(sm.XMLContent())
	return err
}

func (s *Server) indexPage(ctx fiber.Ctx) error {
	//it, _ := s.db.Query(NsName).
	//	Where("id", reindexer.EQ, 1).
	//	Limit(1).
	//	Get()
	//
	//feed := it.(*FeedLine)

	// Fake data for demo

	var feed = []FeedLine{}
	feed = append(feed, FeedLine{
		ID:     0,
		Title:  "First title",
		Author: "koplenov :D",
	})
	feed = append(feed, FeedLine{
		ID:     1,
		Title:  "Second title",
		Author: "miro laku :>",
	})
	feed = append(feed, FeedLine{
		ID:     2,
		Title:  "Third title",
		Author: "dmitru moth :/",
	})

	return ctx.Render("index", fiber.Map{
		"feed": &feed,
		"HTML": reflect.TypeOf((*native.HTML)(nil)).Elem(),
		//"ref":  &feed,
	})
}
