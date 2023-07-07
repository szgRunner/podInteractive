package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"

	"podInteractive/pkg/docker"
	"podInteractive/pkg/page"
	"podInteractive/pkg/pod"
	"podInteractive/pkg/utils"
)

/**
 * 将日志输出到标准输出
 */
func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	var path string
	flag.StringVar(&path, "kubeConfig", "", "kubernetes config path")
	flag.Parse()
	utils.SetKubeConfigPath(path)

	fmt.Println("SERVER 9999")
	container := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Route(ws.GET("/").To(page.Index))
	ws.Route(ws.GET("/log").To(page.Log))
	ws.Route(ws.GET("/exec").To(page.Exec))

	//pod
	ws.Route(ws.GET("/ns/{ns}/podName/{podName}/log").To(pod.PodLog))
	ws.Route(ws.GET("/ns/{ns}/podName/{podName}/exec").To(pod.PodExec))
	ws.Route(ws.POST("/pod/resize").To(pod.Resize)).Produces(restful.MIME_JSON)

	//docker
	ws.Route(ws.GET("/docker/{containerId}/exec").To(docker.Exec))
	ws.Route(ws.GET("/docker/{containerId}/log").To(docker.Log))
	ws.Route(ws.POST("/docker/resize").To(docker.Resize)).Produces(restful.MIME_JSON)

	container.Add(ws)

	// Add container filter to enable CORS
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST"},
		CookiesAllowed: false,
		Container:      container}
	container.Filter(cors.Filter)
	container.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	log.Fatal(http.ListenAndServe(":9999", container))
}
