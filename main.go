package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/bingoohuang/sshman/config"
	_ "github.com/bingoohuang/sshman/config"
	"github.com/bingoohuang/sshman/controller"
	"github.com/bingoohuang/sshman/controller/middleware"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

//go:embed static/*
var staticFs embed.FS

//go:embed view/*
var viewFs embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	conf := flag.String("conf", "sshman.toml", "config toml file path")
	initial := flag.Bool("init", false, "create example config file sshman.toml")
	flag.Parse()

	setupSampleConfFile(*initial)
	config.LoadConfig(*conf)

	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	r := gin.New()
	r.Use(gin.Recovery())
	r.StaticFS("/static", WithPrefix("/static", http.FS(staticFs)))
	r.GET("/", redirectLogin)
	r.GET("/login", serveViewFile("/view/login.html"))
	r.GET("/console", serveViewFile("/view/console.html"))
	r.GET("/servers", serveViewFile("/view/servers.html"))
	r.GET("/add", serveViewFile("/view/add.html"))
	r.GET("/setpass", serveViewFile("/view/setpass.html"))
	r.GET("/openterm", serveViewFile("/view/openterm.html"))
	r.GET("/term", serveViewFile("/view/term.html"))

	v1 := r.Group("/v1")
	{
		v1.POST("/login", controller.Login)
		v1.POST("/send", controller.Send)
		v1.GET("/term/:sid", controller.WsSsh)
		v1.GET("/sftp/:sid", controller.Sftp_ssh)
		v1Auth := v1.Use(middleware.Auth())
		v1Auth.GET("/userinfo", controller.Info)
		v1Auth.POST("/nickname", controller.UpdataNick)
		v1Auth.POST("/addser", controller.Addser)
		v1Auth.POST("/repass", controller.ResetPass)
		v1Auth.POST("/delete", controller.Del)
		v1Auth.POST("/getterm", controller.GetTerm)
	}
	if err := r.Run(config.Conf.Web.Port); err != nil {
		log.Panicf("Web Serve Start Err : %v", err)
	}
}

func setupSampleConfFile(initial bool) {
	if !initial {
		return
	}

	configData, err := viewFs.ReadFile("view/sshman.toml")
	if err != nil {
		fmt.Printf("read template /view/sshman.toml error %v\n", err)
	}

	const confFile = "sshman.toml"
	if err = os.WriteFile(confFile, configData, 0644); err == nil {
		fmt.Printf("%s created successfully\n", confFile)
	} else {
		fmt.Printf("%s created error %v\n", confFile, err)
	}
	os.Exit(0)
}

func redirectLogin(c *gin.Context) {
	c.Redirect(302, "/login")
}

func serveViewFile(filepath string) func(c *gin.Context) {
	return func(c *gin.Context) { c.FileFromFS(filepath, http.FS(viewFs)) }
}

type prefixFS struct {
	p string
	f http.FileSystem
}

func (a prefixFS) Open(name string) (http.File, error) { return a.f.Open(a.p + name) }

func WithPrefix(prefix string, f http.FileSystem) http.FileSystem { return prefixFS{p: prefix, f: f} }
