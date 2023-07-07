package pod

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"net/http"
	"podInteractive/pkg/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 	ws.Route(ws.GET("/ns/{ns}/podName/{podName}/log")
func PodLog(request *restful.Request, response *restful.Response) {
	params := request.PathParameters()
	ns := params["ns"]
	podName := params["podName"]
	containerName := request.QueryParameter("containerName")
	if containerName == "" {
		// 没有指定，获取第一个
		containerName, _ = GetFirstContainerName(ns, podName)
	}

	// 完成 ws 协议的握手操作
	c, err := upgrader.Upgrade(response, request.Request, nil)
	if err != nil {
		fmt.Println("websocket-> upgrade-> err:", err)
		return
	}

	//defer c.Close()

	cancelCtx, cancel := context.WithCancel(request.Request.Context())
	readerGroup, ctx := errgroup.WithContext(cancelCtx)

	go func() {
		for {
			if _, _, err := c.NextReader(); err != nil {
				cancel()
				c.Close()
				break
			}
		}
	}()
	logEvent := make(chan []byte)

	ReadLog(ctx, readerGroup, logEvent, ns, podName, containerName)

	go func() {
		_ = readerGroup.Wait()
		close(logEvent)
	}()
	done := false
	for !done {
		select {
		case item, ok := <-logEvent:
			if !ok {
				done = true
				break
			}
			if err := writeData(c, item); err != nil {
				cancel()
			}

		}
	}

}

func writeData(c *websocket.Conn, buf []byte) error {
	messageWriter, err := c.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	if _, err := messageWriter.Write(buf); err != nil {
		return err
	}
	return messageWriter.Close()
}
func GetFirstContainerName(ns string, podName string) (string, error) {
	pod, err := utils.Cli().CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if len(pod.Spec.Containers) == 0 {
		return "", errors.New("没有容器")
	}
	return pod.Spec.Containers[0].Name, nil
}

func ReadLog(ctx context.Context, eg *errgroup.Group, logEvent chan []byte, ns, podName, containerName string) {
	eg.Go(func() error {
		//now := time.Now()
		req := utils.Cli().CoreV1().RESTClient().Get().
			Resource("pods").
			Name(podName).
			Namespace(ns).
			SubResource("log").
			Context(ctx).
			Param("container", containerName).
			Param("follow", "true").
			//Param("sinceTime", now.Add(-time.Hour*3).Format(time.RFC3339)).
			Param("tailLines", "500").
			Param("timestamps", "false").
			VersionedParams(&v1.PodLogOptions{}, scheme.ParameterCodec)

		podLogStream, err := req.Stream()
		if err != nil {
			fmt.Println("podLogStream-> err: ", err)
			return err
		}

		podLogReader := bufio.NewReader(podLogStream)
		for {
			line, err := podLogReader.ReadBytes('\n')
			if err != nil {
				fmt.Println("podLogReader -> ReadBytes-> err: ", err)
				return err
			}
			logEvent <- line
			if err == io.EOF {
				podLogStream.Close()
				break
			}
		}
		return nil
	})
}
